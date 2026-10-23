package reports

// Đồ nghề dùng chung của test: đồng hồ cố định, dựng Item, tenant riêng cho từng test, chạy cùng một kịch bản trên mọi kho có sẵn.

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// t0 là gốc thời gian của mọi test: giờ gửi = t0 + số phút.
var t0 = time.Date(2026, 10, 19, 8, 0, 0, 0, time.UTC)

// fakeClock là đồng hồ của hệ: mỗi lần hỏi tiến một giây, tất định và an toàn cho nhiều goroutine.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (f *fakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.t = f.t.Add(time.Second)
	return f.t
}

func testOptions() Options {
	clk := &fakeClock{t: t0}
	return Options{Now: clk.Now}
}

var tenantSeq atomic.Int64

// newCtx cấp một ngữ cảnh mang tenant duy nhất cho test: các test chạy chung một kho (PG) không giẫm nhau.
func newCtx(name string) (context.Context, string) {
	id := fmt.Sprintf("t-%s-%d-%d", name, time.Now().UnixNano()%1_000_000, tenantSeq.Add(1))
	return contracts.WithTenant(context.Background(), contracts.TenantID(id)), id
}

// mk dựng một Item ACCEPTED hợp lệ: cùng (thực thể E1, ngày 2026-10-19, 1000 L, OCCURRED), gửi lúc t0 + min phút, mỗi khoá một
// người gửi và một nguyên văn riêng (để vân tay TR2 không trùng ngoài ý muốn). mods chỉnh tiếp.
func mk(key string, min int, mods ...func(*Item)) Item {
	it := Item{
		Key: key, Channel: "chat", SourceType: "chat_message", SenderID: "S-" + key,
		SentAt: t0.Add(time.Duration(min) * time.Minute), SourceSeq: key,
		OccurredOn: "2026-10-19", Entity: "E1", Metric: "m1", Value: Quantity{Number: "1000", Unit: "L"}, Kind: KindOccurred,
		Raw: "tin " + key, Verdict: Verdict{Outcome: OutcomeAccepted}, PackVersion: "pk1", TraceID: "tr-" + key,
	}
	for _, f := range mods {
		f(&it)
	}
	return it
}

func heldFor(r Reason) func(*Item) {
	return func(it *Item) {
		it.Verdict = Verdict{Outcome: OutcomeHeld, Reason: r, Detail: "chi tiết " + string(r)}
	}
}

func quarantinedFor(r Reason) func(*Item) {
	return func(it *Item) {
		it.Verdict = Verdict{Outcome: OutcomeQuarantined, Reason: r, Detail: "chi tiết " + string(r)}
	}
}

func withRaw(s string) func(*Item)    { return func(it *Item) { it.Raw = s } }
func withSender(s string) func(*Item) { return func(it *Item) { it.SenderID = s } }
func withValue(n string) func(*Item)  { return func(it *Item) { it.Value.Number = n } }
func withEntity(e string) func(*Item) { return func(it *Item) { it.Entity = e } }
func withDay(d string) func(*Item)    { return func(it *Item) { it.OccurredOn = d } }
func withKind(k Kind) func(*Item)     { return func(it *Item) { it.Kind = k } }
func withNoSentAt(seq string) func(*Item) {
	return func(it *Item) { it.SentAt = time.Time{}; it.SourceSeq = seq }
}

// storeEnv là một kho kèm cách cấp ngữ cảnh tenant mới.
type storeEnv struct {
	name  string
	st    Store
	extra func(t *testing.T) // kiểm thêm riêng của kho (rỗng với kho bộ nhớ)
}

// forEachStore chạy fn trên mọi kho có sẵn: kho bộ nhớ luôn có; kho PG được thêm ở pg_test.go khi có PostgreSQL.
func forEachStore(t *testing.T, fn func(t *testing.T, env storeEnv)) {
	t.Helper()
	t.Run("bonho", func(t *testing.T) { fn(t, storeEnv{name: "bonho", st: NewMemory(testOptions())}) })
	for _, mkPG := range pgStoreMakers {
		mkPG := mkPG
		t.Run("pg", func(t *testing.T) { fn(t, mkPG(t)) })
	}
}

// pgStoreMakers do pg_test.go đăng ký; rỗng khi chưa có.
var pgStoreMakers []func(t *testing.T) storeEnv

// mustDeliver giao một đơn vị vào lô cho trước; lỗi là test hỏng.
func mustDeliver(t *testing.T, ctx context.Context, st Store, batch string, ord int, it Item) DeliveryEntry {
	t.Helper()
	e, err := st.Deliver(ctx, batch, ord, it)
	if err != nil {
		t.Fatalf("Deliver(%s): %v", it.Key, err)
	}
	return e
}

// mustBatch giao các đơn vị vào một lô mới; lỗi là test hỏng.
func mustBatch(t *testing.T, ctx context.Context, st Store, items ...Item) Receipt {
	t.Helper()
	r, err := st.DeliverBatch(ctx, items)
	if err != nil {
		t.Fatalf("DeliverBatch: %v", err)
	}
	return r
}

func mustRecord(t *testing.T, ctx context.Context, st Store, key string) Record {
	t.Helper()
	r, ok, err := st.Record(ctx, key)
	if err != nil || !ok {
		t.Fatalf("Record(%s): ok=%v err=%v", key, ok, err)
	}
	return r
}

func mustRecords(t *testing.T, ctx context.Context, st Store) []Record {
	t.Helper()
	rs, err := st.Records(ctx)
	if err != nil {
		t.Fatalf("Records: %v", err)
	}
	return rs
}

// wantRec khẳng định trạng thái và mã lý do của một bản ghi.
func wantRec(t *testing.T, ctx context.Context, st Store, key string, d Disposition, r Reason) Record {
	t.Helper()
	rec := mustRecord(t, ctx, st, key)
	if rec.Disposition != d || rec.Reason != r {
		t.Fatalf("bản %s: được %s/%q, muốn %s/%q (detail %q)", key, rec.Disposition, rec.Reason, d, r, rec.Detail)
	}
	return rec
}

func keysOf(items []Item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Key
	}
	return out
}

// dispOf trả bản đồ unit_key → "TRẠNG THÁI/lý do/duplicate_of" của mọi bản ghi, để so hai kho hoặc hai thứ tự đến.
func dispOf(rs []Record) map[string]string {
	m := map[string]string{}
	for _, r := range rs {
		m[r.Key] = fmt.Sprintf("%s/%s/%s", r.Disposition, r.Reason, r.DuplicateOf)
	}
	return m
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// totalOf trả tổng đã đọc cho (E1, 2026-10-19, L); "0" khi chưa có dòng.
func totalOf(t *testing.T, ctx context.Context, st Store, entity string) string {
	t.Helper()
	ts, err := st.Totals(ctx)
	if err != nil {
		t.Fatalf("Totals: %v", err)
	}
	for _, x := range ts {
		if x.Entity == entity && x.OccurredOn == "2026-10-19" && x.Unit == "L" {
			return x.Number
		}
	}
	return "0"
}
