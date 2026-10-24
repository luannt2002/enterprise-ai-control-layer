package reports

import (
	"errors"
	"fmt"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/decimal"
)

// Quyết định của người (§6.2) và hai nguyên thủy ÁP DỤNG của lớp phục hồi: Retract, Amend. Mọi thao tác ở đây kết thúc bằng settle (tr3.go): quyết định cho ra
// ACCEPTED chạy TR3 (§5.6 mục 3), bản rời nhóm làm bản bị bác theo nó quay lại HELD (§5.6 mục 4 (c)).
//
// Quy ước lỗi: lệnh bị TỪ CHỐI (sai quyết định cho trạng thái, màn hình cũ, thiếu trường) là refusal: bản ghi KHÔNG đổi, một dòng audit vẫn ghi, lỗi mang nhãn input.
// Mọi kiểm tra chạy TRƯỚC khi đổi bất cứ thứ gì (lỗi mang audit làm kho ghi lại mọi thứ đã đổi trong thao tác, nên không được từ chối giữa chừng).

// refusal là lỗi từ chối của một quyết định; decide gắn nhãn input và ghi audit.
type refusal struct{ err error }

func (e refusal) Error() string { return e.err.Error() }
func (e refusal) Unwrap() error { return e.err }

func refuseWith(sentinel error, format string, a ...any) error {
	return refusal{fmt.Errorf("%w: %s", sentinel, fmt.Sprintf(format, a...))}
}

// auditedErr đánh dấu lỗi mà kho phải vẫn ghi phần audit (engine.close).
type auditedErr struct{ error }

func (e auditedErr) Unwrap() error { return e.error }
func (auditedErr) keepAudit() bool { return true }

// auditClip là độ dài tối đa (ký tự) của giá trị do người gửi lệnh cấp mà ta ghi vào audit (U14: chưa nêu giới hạn ở hợp đồng; chọn khi mở mã).
const auditClip = 200

func clip(s string) string {
	r := []rune(s)
	if len(r) > auditClip {
		return string(r[:auditClip]) + "…"
	}
	return s
}

// refuse ghi một dòng audit "lệnh bị từ chối" lên bản ghi r (bản ghi không đổi) rồi trả lỗi mang nhãn input và dấu audit.
func (c *opctx) refuse(r *Record, d Decision, actor, why string, err error) error {
	who := clip(actor)
	if who == "" {
		who = "(không nêu người quyết)" // dòng audit luôn có "ai"; lệnh thiếu người quyết vẫn được ghi lại
	}
	c.history = append(c.history, HistoryEntry{
		Tenant: c.tenant, Key: r.Key, At: c.now, Actor: who, Event: EventRefused, From: r.Disposition, To: r.Disposition, Reason: r.Reason,
		Detail:  fmt.Sprintf("%s bị từ chối: %s; người quyết %q, vì sao %q", d, clip(err.Error()), clip(actor), clip(why)),
		Version: r.Version, DuplicateOf: r.DuplicateOf, Decision: d, PackVersion: r.PackVersion,
	})
	return contracts.Label(contracts.FaultInput, auditedErr{err})
}

// decisionAllowed là bảng §6.2: quyết định nào cho mã lý do nào. DISMISS và REPROCESS của bản QUARANTINED chưa có (doc.go); APPLY cho id_conflict chỉ khi mục chờ là lần
// giao sửa tin hoặc thu hồi (bước 7), nên chưa có ở đây.
func decisionAllowed(r *Record, d Decision) bool {
	switch r.Disposition {
	case DispQuarantined:
		return d == DecCorrect
	case DispHeld:
		switch r.Reason {
		case ReasonAmbiguousEntity:
			return d == DecChoose || d == DecReject
		case ReasonNumberReadingUncertain, ReasonDateConflict, ReasonAmbiguousKind, ReasonIDConflict:
			return d == DecCorrect || d == DecReject
		case ReasonCorrectionUnconfirmed:
			return d == DecApply || d == DecReject
		case ReasonPossibleDuplicate:
			return d == DecKeepBoth || d == DecMarkDuplicate
		case ReasonSenderNotAuthorized:
			return d == DecAuthorize || d == DecReject
		}
	}
	return false
}

func (c *opctx) stamp(r *Record, d Decision, actor string) {
	r.DecidedBy, r.DecidedAt, r.Decision = actor, c.now, d
}

// decide áp MỘT quyết định của người. Trả bản ghi sau quyết định và sau TR3 (có thể đã là HELD possible_duplicate).
func (c *opctx) decide(cmd Command) (Record, error) {
	key := cleanText(cmd.Key)
	r, err := c.getLocked(key)
	if err != nil {
		return Record{}, err
	}
	if r == nil {
		return Record{}, inputErr(ErrNotFound, "unit_key %q", key)
	}
	actor, why := cleanText(cmd.Actor), cleanText(cmd.Why)
	c.cause = fmt.Sprintf("quyết định %s trên %s (trước đó %s %s)", cmd.Decision, r.Key, r.Disposition, r.Reason)
	switch {
	case actor == "":
		err = refuseWith(ErrDecisionNotAllowed, "thiếu người quyết (audit cần ai)")
	case !decisionAllowed(r, cmd.Decision):
		err = refuseWith(ErrDecisionNotAllowed, "%s không được phép cho bản %s %s (§6.2)", cmd.Decision, r.Disposition, r.Reason)
	default:
		switch cmd.Decision {
		case DecKeepBoth, DecMarkDuplicate:
			err = c.decidePair(r, cmd, actor, why)
		case DecReject:
			err = c.decideReject(r, actor, why)
		case DecChoose, DecCorrect, DecAuthorize:
			err = c.decideAccept(r, cmd, actor, why)
		case DecApply:
			err = c.decideApply(r, cmd, actor, why)
		default:
			err = refuseWith(ErrDecisionNotAllowed, "quyết định %q không có", cmd.Decision)
		}
	}
	var rf refusal
	if errors.As(err, &rf) {
		return Record{}, c.refuse(r, cmd.Decision, actor, why, err)
	}
	if err != nil {
		return Record{}, err
	}
	if err := c.settle(); err != nil {
		return Record{}, err
	}
	return r.clone(), nil
}

// decidePair: KEEP_BOTH và MARK_DUPLICATE cho bản HELD possible_duplicate. Lệnh mang duplicate_of mà người đã thấy (§6.2): thiếu thì từ chối (đóng mặc định); lệch
// duplicate_of hiện tại thì từ chối (màn hình cũ); "có mặt nhưng rỗng" (màn hình ghi bản đứng trước đã rời nhóm) khác "thiếu" (R4-03).
func (c *opctx) decidePair(r *Record, cmd Command, actor, why string) error {
	if cmd.SeenDuplicateOf == nil {
		return refuseWith(ErrMissingSeen, "lệnh %s phải mang duplicate_of mà người quyết đã thấy (rỗng nếu màn hình ghi bản đứng trước đã rời nhóm)", cmd.Decision)
	}
	seen := cleanText(*cmd.SeenDuplicateOf)
	if seen != r.DuplicateOf {
		return refusal{&StaleDecisionError{Key: r.Key, Seen: seen, Current: r.DuplicateOf}}
	}
	if cmd.Decision == DecMarkDuplicate && r.DuplicateOf == "" {
		return refuseWith(ErrNoCounterpart, "bản %s không có bản kia trong nhóm: nhận MARK_DUPLICATE thì cả hai bản rơi khỏi tổng và hàng chờ", r.Key)
	}
	from := r.Disposition
	c.touch(r)
	if cmd.Decision == DecKeepBoth {
		r.Disposition, r.Reason = DispAccepted, ""
		if seen != "" && !r.reviewedWith(seen) {
			r.ReviewedPairs = append(r.ReviewedPairs, seen) // cộng dồn, không ghi đè các cặp trước (§5.6 mục 3)
		}
		r.Detail = fmt.Sprintf("KEEP_BOTH bởi %s (cặp đã xét với %q): %s", actor, seen, why)
	} else {
		r.Disposition = DispRejected // mã lý do và duplicate_of (bản được giữ) giữ nguyên
		r.Detail = fmt.Sprintf("MARK_DUPLICATE bởi %s: trùng bản %s: %s", actor, r.DuplicateOf, why)
	}
	c.stamp(r, cmd.Decision, actor)
	c.commit(r)
	c.hist(r, EventDecision, from, actor, r.Detail, cmd.Decision)
	return nil
}

func (c *opctx) decideReject(r *Record, actor, why string) error {
	from := r.Disposition
	c.touch(r)
	r.Disposition = DispRejected
	r.Detail = fmt.Sprintf("REJECT bởi %s: %s", actor, why)
	c.stamp(r, DecReject, actor)
	c.commit(r)
	c.hist(r, EventDecision, from, actor, r.Detail, DecReject)
	return nil
}

// applyCorrect áp các trường người nhập (CORRECT) lên bản n; trường để trống thì giữ.
func applyCorrect(n *Record, cmd Command) error {
	if e := cleanText(cmd.Entity); e != "" {
		n.Entity = e
	}
	if cmd.Value != nil {
		v := Quantity{Number: cmd.Value.Number, Unit: cleanText(cmd.Value.Unit)}
		canon, ok := decimal.Canonical(v.Number)
		if !ok {
			return refuseWith(ErrInvalidItem, "lượng %q không phải số thập phân", clip(v.Number))
		}
		v.Number = canon
		n.Value = v
	}
	if d := cleanText(cmd.OccurredOn); d != "" {
		if _, err := time.Parse(dateLayout, d); err != nil {
			return refuseWith(ErrInvalidItem, "ngày xảy ra %q không phải YYYY-MM-DD", clip(d))
		}
		n.OccurredOn = d
	}
	switch cmd.NewKind {
	case "":
	case KindOccurred, KindPlan, KindCorrection:
		n.Kind = cmd.NewKind
	default:
		return refuseWith(ErrInvalidItem, "loại tin %q ngoài {OCCURRED, PLAN, CORRECTION}", clip(string(cmd.NewKind)))
	}
	return nil
}

// decideAccept: CHOOSE, CORRECT, AUTHORIZE cho ra ACCEPTED (rồi settle chạy TR3: có thể thành HELD possible_duplicate).
func (c *opctx) decideAccept(r *Record, cmd Command, actor, why string) error {
	n := r.clone()
	bump := false
	switch cmd.Decision {
	case DecChoose:
		ent := cleanText(cmd.Entity)
		if ent == "" {
			return refuseWith(ErrInvalidItem, "CHOOSE thiếu thực thể được chọn")
		}
		n.Entity, bump = ent, true
	case DecCorrect:
		if err := applyCorrect(&n, cmd); err != nil {
			return err
		}
		bump = true
	}
	return c.acceptRecord(r, n, cmd.Decision, actor, why, bump)
}

// checkAcceptable: bản được nhận phải đủ trường (BB6: không nhận bản đoán).
func checkAcceptable(n *Record, d Decision) error {
	if n.Entity == "" || n.OccurredOn == "" || n.Value.Number == "" || n.Value.Unit == "" {
		return refuseWith(ErrInvalidItem, "bản được nhận sau %s phải có thực thể, ngày xảy ra, lượng và đơn vị (BB6: không nhận bản đoán)", d)
	}
	return nil
}

// acceptRecord ghi n (bản r sau chỉnh) thành ACCEPTED, kèm dấu quyết định. bump: nội dung đổi nên thêm một phiên bản (bản máy đọc còn ở các phiên bản trước). Khi nhóm của bản
// đổi, các cặp đã xét liên quan bị xoá (cặp đã xét là về nội dung cũ).
func (c *opctx) acceptRecord(r *Record, n Record, d Decision, actor, why string, bump bool) error {
	if err := checkAcceptable(&n, d); err != nil {
		return err
	}
	from := r.Disposition
	oldSlot, hadSlot := slotOf(r)
	moved := groupOf(r) != groupOf(&n) || !r.SentAt.Equal(n.SentAt)
	c.touch(r)
	n.Disposition, n.Reason, n.DuplicateOf = DispAccepted, "", ""
	n.Detail = fmt.Sprintf("%s bởi %s: %s", d, actor, why)
	c.stamp(&n, d, actor)
	if bump {
		n.Version = r.Version + 1
	}
	if moved {
		n.ReviewedPairs = nil
	}
	*r = n
	if moved && hadSlot {
		if err := c.purgePairsAbout(r.Key, oldSlot); err != nil {
			return err
		}
	}
	c.commit(r)
	if bump {
		c.versions = append(c.versions, versionOf(r))
	}
	c.hist(r, EventDecision, from, actor, r.Detail, d)
	return nil
}

// decideApply: APPLY đính chính (§6.2): bản gốc SUPERSEDED (đường rời nhóm mà R6-01 nêu), bản đính chính cùng unit_key thành OCCURRED ACCEPTED.
func (c *opctx) decideApply(r *Record, cmd Command, actor, why string) error {
	target := cleanText(cmd.Target)
	if target == "" {
		target = r.Corrects
	}
	if target == "" {
		return refuseWith(ErrInvalidItem, "APPLY thiếu bản gốc (Target) và bản đính chính không ghi corrects")
	}
	if target == r.Key {
		return refuseWith(ErrInvalidItem, "bản đính chính không thể thay chính nó")
	}
	t, err := c.getLocked(target)
	if err != nil {
		return err
	}
	if t == nil {
		return refuseWith(ErrNotFound, "bản gốc %q không có", clip(target))
	}
	switch t.Disposition {
	case DispAccepted, DispHeld, DispRejected:
	default:
		return refuseWith(ErrDecisionNotAllowed, "bản gốc %s đang %s: không thay được", t.Key, t.Disposition)
	}
	n := r.clone()
	n.Kind, n.Corrects = KindOccurred, t.Key
	if err := checkAcceptable(&n, DecApply); err != nil { // kiểm TRƯỚC khi đổi bản gốc
		return err
	}
	from := t.Disposition
	c.touch(t)
	t.Disposition, t.SupersededBy = DispSuperseded, r.Key
	t.Detail = fmt.Sprintf("bị thay bởi đính chính %s (APPLY bởi %s): %s", r.Key, actor, why)
	c.commit(t)
	c.hist(t, EventSuperseded, from, actor, t.Detail, DecApply)
	return c.acceptRecord(r, n, DecApply, actor, why, true)
}

// purgePairsAbout xoá khỏi các bản của ô s mọi cặp đã xét với bản key: nội dung của key vừa đổi nên cặp đã xét (về nội dung cũ) không còn miễn cho nội dung mới. Chỉ siết, có audit.
func (c *opctx) purgePairsAbout(key string, s slotKey) error {
	recs, err := c.slotRecs(s)
	if err != nil {
		return err
	}
	for _, q := range recs {
		if q.Key == key || !q.reviewedWith(key) {
			continue
		}
		c.touch(q)
		var kept []string
		for _, k := range q.ReviewedPairs {
			if k != key {
				kept = append(kept, k)
			}
		}
		q.ReviewedPairs = kept
		c.commit(q)
		c.hist(q, EventPairReset, q.Disposition, ActorTR3, fmt.Sprintf("cặp đã xét với %s bị xoá vì nội dung của %s đổi (cặp đã xét là về nội dung cũ)", key, key), "")
	}
	return nil
}

// retract đưa bản ghi sang RETRACTED: rời tổng và nhóm TR3, còn trong lịch sử (§7.3). Nguyên thủy ÁP DỤNG: không kiểm người gửi, không tạo dấu thu hồi, không tạo phiên bản chờ.
func (c *opctx) retract(key, actor, why string) error {
	key, actor, why = cleanText(key), cleanText(actor), cleanText(why)
	r, err := c.getLocked(key)
	if err != nil {
		return err
	}
	if r == nil {
		return inputErr(ErrNotFound, "unit_key %q", key)
	}
	if actor == "" {
		return inputErr(ErrDecisionNotAllowed, "thu hồi thiếu người thực hiện (audit cần ai)")
	}
	if r.Disposition == DispRetracted || r.Disposition == DispSuperseded {
		return inputErr(ErrDecisionNotAllowed, "bản %s đã %s: không thu hồi lại", r.Key, r.Disposition)
	}
	c.cause = "thu hồi bản " + r.Key
	from := r.Disposition
	c.touch(r)
	r.Disposition = DispRetracted
	r.Detail = fmt.Sprintf("thu hồi bởi %s: %s", actor, why)
	c.commit(r)
	c.hist(r, EventRetracted, from, actor, r.Detail, "")
	return c.settle()
}

// amend thêm một phiên bản nội dung mới cho bản ghi (version + 1, §7.2): kết cục mới theo it.Verdict, vào hoặc rời nhóm TR3 theo nội dung mới. Nguyên thủy ÁP DỤNG: it.Key bị bỏ qua; người
// gửi, đường vào và quyết định cũ của người không bị kiểm hay đổi ở đây (lớp phục hồi, bước 7, kiểm trước khi gọi). Nội dung không đổi hoặc bản đã RETRACTED, SUPERSEDED bị từ chối: nguyên thủy
// không im lặng bỏ qua.
func (c *opctx) amend(key string, it Item, actor, why string) error {
	key, actor, why = cleanText(key), cleanText(actor), cleanText(why)
	r, err := c.getLocked(key)
	if err != nil {
		return err
	}
	if r == nil {
		return inputErr(ErrNotFound, "unit_key %q", key)
	}
	if actor == "" {
		return inputErr(ErrDecisionNotAllowed, "sửa tin thiếu người thực hiện (audit cần ai)")
	}
	if r.Disposition == DispRetracted || r.Disposition == DispSuperseded {
		return inputErr(ErrDecisionNotAllowed, "bản %s đã %s: không sửa", r.Key, r.Disposition)
	}
	it = it.sanitized()
	it.Key = r.Key
	if err := it.validate(); err != nil {
		return err
	}
	hash, fp := contentHashOf(it.Raw), fingerprintOf(c.tenant, it)
	n := c.newRecord(it, hash, fp)
	if hash == r.ContentHash && groupOf(n) == groupOf(r) && n.SentAt.Equal(r.SentAt) && n.Kind == r.Kind && n.Approx == r.Approx &&
		n.Disposition == r.Disposition && n.Reason == r.Reason && n.Entity == r.Entity {
		return inputErr(ErrDecisionNotAllowed, "nội dung sửa của bản %s không đổi so với phiên bản hiện hành: không thêm phiên bản", r.Key)
	}
	c.cause = "sửa tin bản " + r.Key
	from := r.Disposition
	oldSlot, hadSlot := slotOf(r)
	moved := groupOf(r) != groupOf(n) || !r.SentAt.Equal(n.SentAt)
	c.touch(r)
	n.Version = r.Version + 1
	n.DecidedBy, n.DecidedAt, n.Decision = r.DecidedBy, r.DecidedAt, r.Decision // quyết định cũ của người còn ở bản ghi (BB7)
	if !moved {
		n.ReviewedPairs = append([]string(nil), r.ReviewedPairs...)
	}
	*r = *n
	if moved && hadSlot {
		if err := c.purgePairsAbout(r.Key, oldSlot); err != nil {
			return err
		}
	}
	c.commit(r)
	c.versions = append(c.versions, versionOf(r))
	c.hist(r, EventAmended, from, actor, fmt.Sprintf("sửa tin, phiên bản %d: %s", r.Version, why), "")
	return c.settle()
}
