package reports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/luannt2002/enterprise-ai-control-layer/core/decimal"
)

// errKeyRace do kho trả khi INSERT bản ghi đụng khoá duy nhất (tenant, unit_key) vì một giao dịch khác vừa tạo nó: bên gọi huỷ giao
// dịch, đọc lại và đi theo đường TR1 (§7.5: thua đua thì so lại nội dung với bản thắng).
var errKeyRace = errors.New("va chạm khoá duy nhất (tenant, unit_key)")

// versionRow là một phiên bản nội dung của bản ghi (chỉ thêm). ContentHash phục vụ TR1, Fingerprint phục vụ TR2; Raw giữ từng byte.
type versionRow struct {
	Key         string
	Version     int
	ContentHash string
	Fingerprint string
	Raw         string
	Snapshot    []byte
}

// txn là phần của một kho mà engine cần, trong MỘT giao dịch đã đặt tenant. Engine không biết kho là bộ nhớ hay PG: cùng một luật chạy
// trên cả hai, nên hai kho không thể lệch nhau về logic. Engine gom mọi thay đổi vào opctx rồi ghi MỘT lượt ở cuối (flush).
type txn interface {
	getRecord(key string) (*Record, error) // nil khi chưa có
	versionsOf(key string) ([]versionRow, error)
	findByFingerprint(fp string) (string, error) // unit_key nhỏ nhất (so byte) có phiên bản mang vân tay này; rỗng khi không có
	slotRecords(entity, day string) ([]*Record, error)
	insertRecord(r *Record) error // errKeyRace khi trùng khoá
	updateRecord(r *Record) error
	insertVersion(v versionRow) error
	insertHistory(h HistoryEntry) error
	insertLedger(e DeliveryEntry) error
}

// slotKey là ô (thực thể, ngày xảy ra): đơn vị của khoá tuần tự hoá (§5.6 mục 2). Mọi nhóm nghi trùng nằm gọn trong một ô.
type slotKey struct{ Entity, Day string }

func slotOf(r *Record) (slotKey, bool) {
	if r.Entity == "" || r.OccurredOn == "" {
		return slotKey{}, false
	}
	return slotKey{Entity: r.Entity, Day: r.OccurredOn}, true
}

// groupKey là khoá nhóm nghi trùng TR3 (§5.1): (thực thể, loại, ngày xảy ra, giá trị, đơn vị) trong một tenant, bất kể người gửi và chữ.
type groupKey struct{ Entity, Kind, Day, Number, Unit string }

func groupOf(r *Record) groupKey {
	n := r.Value.Number
	if c, ok := decimal.Canonical(n); ok {
		n = c
	}
	return groupKey{Entity: r.Entity, Kind: string(r.Kind), Day: r.OccurredOn, Number: n, Unit: r.Value.Unit}
}

// inGroup: bản có sent_at, đang ACCEPTED hoặc HELD possible_duplicate (§5.1). Bản HELD vì lý do khác chưa thuộc nhóm.
func inGroup(r *Record) bool {
	if r.SentAt.IsZero() || r.Entity == "" || r.OccurredOn == "" || r.Value.Number == "" {
		return false
	}
	return r.Disposition == DispAccepted || (r.Disposition == DispHeld && r.Reason == ReasonPossibleDuplicate)
}

// before là thứ tự chuẩn (sent_at, unit_key) của nhóm (§5.6 mục 1).
func before(a, b *Record) bool {
	if !a.SentAt.Equal(b.SentAt) {
		return a.SentAt.Before(b.SentAt)
	}
	return a.Key < b.Key
}

// opctx là trạng thái của MỘT thao tác trên kho: các bản ghi đã đọc hoặc đã đổi, các dòng phiên bản, lịch sử, sổ sẽ ghi. Không có gì
// chạm kho cho tới flush, nên thao tác lỗi giữa chừng không để lại nửa vời.
type opctx struct {
	tenant string
	opt    Options
	tx     txn
	now    time.Time

	cache   map[string]*Record
	missing map[string]bool
	isNew   map[string]bool
	dirty   map[string]bool
	slots   map[slotKey]bool // các ô đã chạm (cả ô cũ lẫn ô mới của bản ghi bị đổi): settle chạy trên đúng các ô này
	loaded  map[slotKey]bool

	versions []versionRow
	history  []HistoryEntry
	ledger   []DeliveryEntry

	afterSlotRead func() // chỉ test đặt: điểm dừng giữa lúc đọc nhóm và lúc ghi (DB26)
}

func newOpctx(tenant string, opt Options, tx txn) *opctx {
	return &opctx{
		tenant: tenant, opt: opt, tx: tx, now: opt.now(),
		cache: map[string]*Record{}, missing: map[string]bool{}, isNew: map[string]bool{}, dirty: map[string]bool{},
		slots: map[slotKey]bool{}, loaded: map[slotKey]bool{},
	}
}

// get trả bản ghi theo khoá (con trỏ dùng chung trong thao tác); nil khi chưa có.
func (c *opctx) get(key string) (*Record, error) {
	if r, ok := c.cache[key]; ok {
		return r, nil
	}
	if c.missing[key] {
		return nil, nil
	}
	r, err := c.tx.getRecord(key)
	if err != nil {
		return nil, err
	}
	if r == nil {
		c.missing[key] = true
		return nil, nil
	}
	c.cache[key] = r
	return r, nil
}

// slotRecs trả mọi bản ghi của một ô, theo unit_key. Bản đã nằm trong cache (đã đổi hoặc mới tạo) thắng bản đọc từ kho; bản đổi ô thì
// theo ô MỚI của nó.
func (c *opctx) slotRecs(s slotKey) ([]*Record, error) {
	if !c.loaded[s] {
		rs, err := c.tx.slotRecords(s.Entity, s.Day)
		if err != nil {
			return nil, err
		}
		for _, r := range rs {
			if _, ok := c.cache[r.Key]; !ok {
				c.cache[r.Key] = r
			}
		}
		c.loaded[s] = true
		if c.afterSlotRead != nil {
			c.afterSlotRead()
		}
	}
	var out []*Record
	for _, r := range c.cache {
		if k, ok := slotOf(r); ok && k == s {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

// touch ghi nhận ô CŨ của r; gọi TRƯỚC khi đổi r. commit ghi nhận ô MỚI và đánh dấu r bẩn; gọi SAU khi đổi. Nhờ vậy mọi đường đổi bản
// ghi, kể cả đường chưa ai nghĩ tới, đều làm settle nhìn lại đúng các ô bị ảnh hưởng (R6-01).
func (c *opctx) touch(r *Record) {
	if k, ok := slotOf(r); ok {
		c.slots[k] = true
	}
}

func (c *opctx) commit(r *Record) {
	if k, ok := slotOf(r); ok {
		c.slots[k] = true
	}
	c.dirty[r.Key] = true
}

func (c *opctx) sortedSlots() []slotKey {
	out := make([]slotKey, 0, len(c.slots))
	for k := range c.slots {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Entity != out[j].Entity {
			return out[i].Entity < out[j].Entity
		}
		return out[i].Day < out[j].Day
	})
	return out
}

// hist thêm một dòng lịch sử cho r (trạng thái SAU khi đổi); from là trạng thái trước.
func (c *opctx) hist(r *Record, event string, from Disposition, actor, detail string, dec Decision) {
	c.history = append(c.history, HistoryEntry{
		Tenant: c.tenant, Key: r.Key, At: c.now, Actor: actor, Event: event, From: from, To: r.Disposition, Reason: r.Reason,
		Detail: detail, Version: r.Version, DuplicateOf: r.DuplicateOf, Decision: dec, PackVersion: r.PackVersion,
	})
}

// recipientFor tính người nhận theo thực thể (§6.3): bên gọi cấp hàm; không xác định được thì director.
func (c *opctx) recipientFor(entity string) string {
	if c.opt.Recipient != nil && entity != "" {
		if s := c.opt.Recipient(entity); s != "" {
			return s
		}
	}
	return DefaultRecipient
}

func snapshotJSON(r *Record) []byte {
	b, err := json.Marshal(r)
	if err != nil {
		return []byte("{}")
	}
	return b
}

func versionOf(r *Record) versionRow {
	return versionRow{Key: r.Key, Version: r.Version, ContentHash: r.ContentHash, Fingerprint: r.Fingerprint, Raw: r.Raw, Snapshot: snapshotJSON(r)}
}

// versionsOf trả mọi phiên bản của một khoá: đã lưu cộng phiên bản vừa thêm trong thao tác này.
func (c *opctx) versionsOf(key string) ([]versionRow, error) {
	vs, err := c.tx.versionsOf(key)
	if err != nil {
		return nil, err
	}
	for _, v := range c.versions {
		if v.Key == key {
			vs = append(vs, v)
		}
	}
	return vs, nil
}

// finishVerdict điền mặc định cho bản ghi mới hoặc phiên bản mới: HELD và QUARANTINED luôn có chi tiết (CL2) và người nhận (BB2);
// ACCEPTED không mang mã lý do.
func (c *opctx) finishVerdict(r *Record) {
	switch r.Disposition {
	case DispHeld, DispQuarantined:
		if r.Detail == "" {
			r.Detail = "mã lý do " + string(r.Reason) + " (luật đọc không ghi chi tiết)"
		}
		r.Recipient = c.recipientFor(r.Entity)
	case DispAccepted:
		r.Reason = ""
	}
}

func (c *opctx) newRecord(it Item, hash, fp string) *Record {
	r := &Record{
		Tenant: c.tenant, Key: it.Key, Channel: it.Channel, SourceType: it.SourceType, SenderID: it.SenderID, SentAt: it.SentAt,
		OccurredOn: it.OccurredOn, Entity: it.Entity, Metric: it.Metric, Value: it.Value, Approx: it.Approx, Kind: it.Kind,
		Corrects: it.Corrects, Raw: it.Raw, Attachments: append([]Attachment(nil), it.Attachments...), Note: it.Note,
		Disposition: Disposition(it.Verdict.Outcome), Reason: it.Verdict.Reason, Detail: it.Verdict.Detail,
		Version: 1, PackVersion: it.PackVersion, TraceID: it.TraceID, Fingerprint: fp, ContentHash: hash,
	}
	c.finishVerdict(r)
	return r
}

// deliver xử lý MỘT lần giao theo thứ tự áp của §3.8: (3) TR1, (4) TR2, (5) Verdict của luật đọc, (6) TR3. Thu hồi và sửa tin (bước 1, 2
// của thứ tự áp) là việc của lớp phục hồi (bước 7): chúng đi qua Retract và Amend, không qua đây.
func (c *opctx) deliver(batchID string, ordinal int, it Item) (DeliveryEntry, error) {
	if batchID == "" || ordinal < 1 {
		return DeliveryEntry{}, inputErr(ErrInvalidItem, "lần giao cần mã lô khác rỗng và thứ tự từ 1 (lô %q, thứ tự %d)", batchID, ordinal)
	}
	it = it.sanitized()
	if err := it.validate(); err != nil {
		return DeliveryEntry{}, err
	}
	orig := it.Key
	hash, fp := contentHashOf(it.Raw), fingerprintOf(c.tenant, it)
	hit, err := c.tr12(&it, hash, fp)
	if err != nil {
		return DeliveryEntry{}, err
	}
	if hit != nil {
		return c.logDelivery(c.duplicateEntry(batchID, ordinal, orig, it, hit)), nil
	}
	r := c.newRecord(it, hash, fp)
	c.cache[r.Key] = r
	c.isNew[r.Key] = true
	c.commit(r)
	c.versions = append(c.versions, versionOf(r))
	c.hist(r, EventDelivered, "", "ingest", "lô "+batchID+", thứ tự "+strconv.Itoa(ordinal), "")
	if err := c.settle(); err != nil {
		return DeliveryEntry{}, err
	}
	return c.logDelivery(DeliveryEntry{
		Tenant: c.tenant, BatchID: batchID, Ordinal: ordinal, Key: orig, Outcome: Outcome(r.Disposition), Reason: r.Reason,
		Detail: r.Detail, RecordRef: r.Key, PackVersion: it.PackVersion, TraceID: it.TraceID, RecordedAt: c.now,
	}), nil
}

func (c *opctx) duplicateEntry(batchID string, ordinal int, orig string, it Item, h *dupHit) DeliveryEntry {
	detail := "lần giao trùng bản " + h.ref
	switch h.reason {
	case ReasonDuplicateKey:
		detail += " theo khoá nguồn (TR1)"
	case ReasonDuplicateFingerprint:
		detail += " theo vân tay (TR2)"
	}
	return DeliveryEntry{
		Tenant: c.tenant, BatchID: batchID, Ordinal: ordinal, Key: orig, Outcome: OutcomeDuplicate, Reason: h.reason, Detail: detail,
		RecordRef: h.ref, PackVersion: it.PackVersion, TraceID: it.TraceID, RecordedAt: c.now,
	}
}

// logDelivery thêm dòng sổ của lần giao: MỌI lần giao đúng một dòng, kể cả ACCEPTED và DUPLICATE (BB1).
func (c *opctx) logDelivery(e DeliveryEntry) DeliveryEntry {
	c.ledger = append(c.ledger, e)
	return e
}

// flush ghi mọi thay đổi MỘT lượt: bản ghi mới (theo khoá), bản ghi đổi, phiên bản, lịch sử, sổ. Thứ tự này thoả khoá ngoại.
func (c *opctx) flush() error {
	keys := make([]string, 0, len(c.dirty))
	for k := range c.dirty {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if c.isNew[k] {
			if err := c.tx.insertRecord(c.cache[k]); err != nil {
				return err
			}
		}
	}
	for _, k := range keys {
		if !c.isNew[k] {
			if err := c.tx.updateRecord(c.cache[k]); err != nil {
				return err
			}
		}
	}
	for _, v := range c.versions {
		if err := c.tx.insertVersion(v); err != nil {
			return err
		}
	}
	for _, h := range c.history {
		if err := c.tx.insertHistory(h); err != nil {
			return err
		}
	}
	for _, e := range c.ledger {
		if err := c.tx.insertLedger(e); err != nil {
			return err
		}
	}
	return nil
}

// close kết thúc thao tác với kết quả err của nó. commit = true khi kho phải được giữ lại. Lỗi mang keepAudit (lệnh bị từ chối kèm
// audit) vẫn ghi phần audit rồi trả lỗi; mọi lỗi khác huỷ hết.
func (c *opctx) close(err error) (commit bool, ret error) {
	if err == nil {
		if ferr := c.flush(); ferr != nil {
			return false, ferr
		}
		return true, nil
	}
	var a interface{ keepAudit() bool }
	if errors.As(err, &a) && a.keepAudit() {
		if ferr := c.flush(); ferr != nil {
			return false, ferr
		}
		return true, err
	}
	return false, err
}

// totalsOf tính tổng đọc (§3.6) từ các bản ghi: chỉ ACCEPTED loại OCCURRED.
func totalsOf(recs []Record) []Total {
	type k struct{ e, d, u string }
	acc := map[k]*Total{}
	for i := range recs {
		r := &recs[i]
		if r.Disposition != DispAccepted || r.Kind != KindOccurred || r.Entity == "" || r.OccurredOn == "" || r.Value.Number == "" {
			continue
		}
		key := k{r.Entity, r.OccurredOn, r.Value.Unit}
		t := acc[key]
		if t == nil {
			t = &Total{Entity: r.Entity, OccurredOn: r.OccurredOn, Unit: r.Value.Unit, Number: "0"}
			acc[key] = t
		}
		if n, err := decimal.Add(t.Number, r.Value.Number); err == nil {
			t.Number = n
		}
		t.Count++
		t.Approx = t.Approx || r.Approx
	}
	out := make([]Total, 0, len(acc))
	for _, t := range acc {
		out = append(out, *t)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Entity != b.Entity {
			return a.Entity < b.Entity
		}
		if a.OccurredOn != b.OccurredOn {
			return a.OccurredOn < b.OccurredOn
		}
		return a.Unit < b.Unit
	})
	return out
}

// deliverBatch là DeliverBatch chung cho mọi kho: mã lô cấp MỚI mỗi lần gọi, không suy từ nội dung (§3.8): nếu suy từ nội dung thì lần
// giao lại đúng nội dung sẽ trùng khoá với dòng của lần trước và che việc rơi im lặng.
func deliverBatch(ctx context.Context, st Store, items []Item) (Receipt, error) {
	batch := uuid.NewString()
	entries := make([]DeliveryEntry, 0, len(items))
	for i, it := range items {
		if err := ctx.Err(); err != nil {
			return BuildReceipt(batch, entries), err
		}
		e, err := st.Deliver(ctx, batch, i+1, it)
		if err != nil {
			return BuildReceipt(batch, entries), fmt.Errorf("đơn vị thứ %d (%s): %w", i+1, it.Key, err)
		}
		entries = append(entries, e)
	}
	return BuildReceipt(batch, entries), nil
}
