package reports

// R6-01 và R6-02 (giấy khoá ở vòng 6; ADR-0012 "Ghi chú soát", HOP-DONG đầu tệp): hai nợ có mã, chặn bằng test đỏ.
//
//	R6-01  bất biến theo SỰ KIỆN: bản được giữ k THÔI được tính vì BẤT KỲ lý do gì thì mọi bản đang REJECTED bởi MARK_DUPLICATE trỏ tới k quay
//	       lại HELD possible_duplicate trong CÙNG giao dịch. Kiểm bằng phép kiểm TÍNH CHẤT chạy sau MỌI sự kiện (oracle bên dưới) và bộ sinh chuỗi sự
//	       kiện ngẫu nhiên, hạt giống cố định.
//	R6-02  các câu "không bản nào rơi khỏi tổng" đúng cho MỌI đường rời nhóm mà mã có: bảng (đường rời nhóm × trạng thái của bản được giữ).
//
// Phép kiểm viết theo chữ của HOP-DONG §5.1 và §5.6, KHÔNG gọi hàm nào của engine (groupOf, inGroup, before, settle): để người viết engine không tự
// chấm bài của mình bằng chính cách hiểu của mình.

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/decimal"
)

// ---------------------------------------------------------------------------------------------------------------------
// Phép kiểm tính chất (oracle)

func containsKey(xs []string, k string) bool {
	for _, x := range xs {
		if x == k {
			return true
		}
	}
	return false
}

func oracleGroupKey(r *Record) string {
	return strings.Join([]string{r.Entity, string(r.Kind), r.OccurredOn, r.Value.Number, r.Value.Unit}, "\x1f")
}

// oracleMember: bản thuộc nhóm nghi trùng (§5.1): có sent_at, đang ACCEPTED hoặc HELD possible_duplicate.
func oracleMember(r *Record) bool {
	return !r.SentAt.IsZero() && r.Entity != "" && r.OccurredOn != "" && r.Value.Number != "" &&
		(r.Disposition == DispAccepted || (r.Disposition == DispHeld && r.Reason == ReasonPossibleDuplicate))
}

// oracle trả mọi chỗ vi phạm của một trạng thái kho (rỗng = đúng). w là W.
func oracle(recs []Record, w time.Duration) []string {
	byKey := map[string]*Record{}
	for i := range recs {
		byKey[recs[i].Key] = &recs[i]
	}
	groups := map[string][]*Record{}
	for i := range recs {
		if r := &recs[i]; oracleMember(r) {
			groups[oracleGroupKey(r)] = append(groups[oracleGroupKey(r)], r)
		}
	}
	var bad []string
	for _, list := range groups {
		sort.Slice(list, func(i, j int) bool {
			if !list[i].SentAt.Equal(list[j].SentAt) {
				return list[i].SentAt.Before(list[j].SentAt)
			}
			return list[i].Key < list[j].Key
		})
		for i, r := range list {
			var p *Record
			if i > 0 {
				p = list[i-1]
			}
			if r.Disposition == DispAccepted && p != nil && r.SentAt.Sub(p.SentAt) <= w && !containsKey(r.ReviewedPairs, p.Key) {
				bad = append(bad, fmt.Sprintf("CỘNG ĐÔI: %s và %s liền kề trong nhóm, cách %v ≤ W, người chưa xét cặp, cùng ACCEPTED", p.Key, r.Key, r.SentAt.Sub(p.SentAt)))
			}
			if r.Disposition == DispHeld {
				want := ""
				if p != nil {
					want = p.Key
				}
				if r.DuplicateOf != want {
					bad = append(bad, fmt.Sprintf("DUPLICATE_OF SAI: %s trỏ %q, bản đứng ngay trước trong nhóm là %q", r.Key, r.DuplicateOf, want))
				}
			}
		}
	}
	for i := range recs {
		r := &recs[i]
		if r.Disposition == DispRejected && r.Decision == DecMarkDuplicate {
			cur, seen := r, map[string]bool{}
			ok := false
			for step := 0; step < len(recs)+1; step++ {
				if seen[cur.Key] {
					break
				}
				seen[cur.Key] = true
				tgt := byKey[cur.DuplicateOf]
				if tgt == nil {
					break
				}
				if oracleMember(tgt) && oracleGroupKey(tgt) == oracleGroupKey(r) {
					ok = true
					break
				}
				if tgt.Disposition == DispRejected && tgt.Decision == DecMarkDuplicate {
					cur = tgt
					continue
				}
				break
			}
			if !ok {
				bad = append(bad, fmt.Sprintf("RƠI KHỎI TỔNG: %s REJECTED theo %q nhưng chuỗi bản được giữ không còn bản nào được tính (tổng hay hàng chờ)", r.Key, r.DuplicateOf))
			}
		}
		switch r.Disposition {
		case DispHeld, DispQuarantined:
			if o, known := r.Reason.Outcome(); !known || o != Outcome(r.Disposition) {
				bad = append(bad, fmt.Sprintf("MÃ LÝ DO: %s %s mang mã %q", r.Key, r.Disposition, r.Reason))
			}
			if r.Recipient == "" || r.Detail == "" {
				bad = append(bad, fmt.Sprintf("BB2/CL2: %s %s thiếu người nhận hoặc chi tiết", r.Key, r.Disposition))
			}
		case DispAccepted:
			if r.Reason != "" || r.Entity == "" || r.OccurredOn == "" || r.Value.Number == "" || r.Value.Unit == "" {
				bad = append(bad, fmt.Sprintf("BB6: %s ACCEPTED thiếu trường hoặc mang lý do: %+v", r.Key, *r))
			}
		}
	}
	sort.Strings(bad)
	return bad
}

// oracleTotals tính lại tổng đọc từ bản ghi (BB3): chỉ ACCEPTED loại OCCURRED.
func oracleTotals(recs []Record) map[string]string {
	out := map[string]string{}
	for i := range recs {
		r := &recs[i]
		if r.Disposition != DispAccepted || r.Kind != KindOccurred {
			continue
		}
		k := r.Entity + "|" + r.OccurredOn + "|" + r.Value.Unit
		cur := out[k]
		if cur == "" {
			cur = "0"
		}
		sum, _ := decimal.Add(cur, r.Value.Number)
		out[k] = sum
	}
	return out
}

func checkTotals(t *testing.T, ctx context.Context, st Store, recs []Record, where string) {
	t.Helper()
	ts, err := st.Totals(ctx)
	if err != nil {
		t.Fatalf("%s: Totals: %v", where, err)
	}
	got := map[string]string{}
	for _, x := range ts {
		got[x.Entity+"|"+x.OccurredOn+"|"+x.Unit] = x.Number
	}
	if want := oracleTotals(recs); !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: tổng đọc %v khác tổng tính lại từ bản ghi %v (BB3)", where, got, want)
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// Bộ sinh chuỗi sự kiện ngẫu nhiên (hạt giống cố định)

type genRun struct {
	rng     *rand.Rand
	st      Store
	ctx     context.Context
	n       int // số khoá đã cấp
	amends  int
	minutes []int
	log     []string
}

func newGenRun(seed int64, st Store, ctx context.Context) *genRun {
	return &genRun{rng: rand.New(rand.NewSource(seed)), st: st, ctx: ctx, minutes: []int{0, 15, 30, 45, 60, 75, 90, 150}}
}

func (g *genRun) pick(xs []string) string { return xs[g.rng.Intn(len(xs))] }

func (g *genRun) newItem() Item {
	g.n++
	key := fmt.Sprintf("k%d", g.n)
	mods := []func(*Item){withValue(g.pick([]string{"1000", "1000", "1000", "2000"}))}
	if g.rng.Intn(100) < 15 {
		mods = append(mods, withDay("2026-10-20"))
	}
	switch p := g.rng.Intn(100); {
	case p < 62: // ACCEPTED
	case p < 80:
		reason := []Reason{ReasonAmbiguousEntity, ReasonSenderNotAuthorized, ReasonNumberReadingUncertain, ReasonDateConflict, ReasonAmbiguousKind, ReasonIDConflict}[g.rng.Intn(6)]
		mods = append(mods, heldFor(reason))
		if reason == ReasonAmbiguousEntity {
			mods = append(mods, withEntity(""))
		}
		if reason == ReasonNumberReadingUncertain {
			mods = append(mods, withValue(""))
		}
	case p < 90:
		mods = append(mods, heldFor(ReasonCorrectionUnconfirmed), withKind(KindCorrection))
	default:
		mods = append(mods, quarantinedFor(ReasonUnparseable), withEntity(""), withDay(""), withValue(""))
	}
	return mk(key, g.minutes[g.rng.Intn(len(g.minutes))], mods...)
}

func (g *genRun) records(t *testing.T) []Record {
	t.Helper()
	rs, err := g.st.Records(g.ctx)
	if err != nil {
		t.Fatalf("Records: %v", err)
	}
	return rs
}

func (g *genRun) choose(rs []Record, pred func(*Record) bool) *Record {
	var c []*Record
	for i := range rs {
		if pred(&rs[i]) {
			c = append(c, &rs[i])
		}
	}
	if len(c) == 0 {
		return nil
	}
	return c[g.rng.Intn(len(c))]
}

// step chạy MỘT sự kiện ngẫu nhiên. Trả mô tả, tập khoá được phép chuyển sang ACCEPTED trong sự kiện này (quyết định của người, sửa tin, nạp mới;
// mọi chuyển khác sang ACCEPTED là máy nhả: vi phạm BB6, BB7) và lỗi của sự kiện (từ chối).
func (g *genRun) step(t *testing.T) (string, map[string]bool, error) {
	rs := g.records(t)
	allow := map[string]bool{}
	heldPD := func(r *Record) bool { return r.Disposition == DispHeld && r.Reason == ReasonPossibleDuplicate }
	heldOther := func(r *Record) bool {
		return r.Disposition == DispHeld && r.Reason != ReasonPossibleDuplicate && r.Reason != ReasonCorrectionUnconfirmed
	}
	live := func(r *Record) bool { return r.Disposition != DispRetracted && r.Disposition != DispSuperseded }
	roll := g.rng.Intn(100)
	switch {
	case roll < 30 || len(rs) == 0:
		it := g.newItem()
		allow[it.Key] = true
		g.log = append(g.log, "deliver "+it.Key)
		_, err := g.st.Deliver(g.ctx, fmt.Sprintf("lo-%d", g.n), 1, it)
		return fmt.Sprintf("deliver %s (min %d, %s %s)", it.Key, int(it.SentAt.Sub(t0)/time.Minute), it.Verdict.Outcome, it.Verdict.Reason), allow, err
	case roll < 48: // KEEP_BOTH hoặc MARK_DUPLICATE
		r := g.choose(rs, heldPD)
		if r == nil {
			return "bỏ qua (không có bản HELD possible_duplicate)", allow, nil
		}
		d := DecKeepBoth
		if g.rng.Intn(2) == 0 {
			d = DecMarkDuplicate
		}
		seen := r.DuplicateOf
		if g.rng.Intn(100) < 12 { // màn hình cũ
			seen = g.pick([]string{"", "k1", "k2", "k3", "k4"})
		}
		allow[r.Key] = d == DecKeepBoth
		_, err := g.st.Decide(g.ctx, Command{Key: r.Key, Decision: d, Actor: "pm-1", Why: "ngẫu nhiên", SeenDuplicateOf: &seen})
		return fmt.Sprintf("%s %s (thấy %q, hiện tại %q)", d, r.Key, seen, r.DuplicateOf), allow, err
	case roll < 58: // CHOOSE, CORRECT, AUTHORIZE, REJECT cho bản HELD vì lý do khác
		r := g.choose(rs, heldOther)
		if r == nil {
			return "bỏ qua (không có bản HELD vì lý do khác)", allow, nil
		}
		cmd := Command{Key: r.Key, Actor: "pm-1", Why: "ngẫu nhiên"}
		switch r.Reason {
		case ReasonAmbiguousEntity:
			cmd.Decision, cmd.Entity = g.pickDecision(DecChoose, DecReject), g.pick([]string{"E1", "E1", "E2"})
		case ReasonSenderNotAuthorized:
			cmd.Decision = g.pickDecision(DecAuthorize, DecReject)
		default:
			cmd.Decision = g.pickDecision(DecCorrect, DecReject)
			cmd.Value = &Quantity{Number: g.pick([]string{"1000", "2000"}), Unit: "L"}
			cmd.Entity, cmd.OccurredOn = "E1", "2026-10-19"
		}
		allow[r.Key] = cmd.Decision != DecReject
		_, err := g.st.Decide(g.ctx, cmd)
		return fmt.Sprintf("%s %s", cmd.Decision, r.Key), allow, err
	case roll < 64: // APPLY / REJECT đính chính
		q := g.choose(rs, func(r *Record) bool { return r.Disposition == DispHeld && r.Reason == ReasonCorrectionUnconfirmed })
		if q == nil {
			return "bỏ qua (không có đính chính chờ)", allow, nil
		}
		tgt := g.choose(rs, func(r *Record) bool { return r.Key != q.Key && live(r) })
		cmd := Command{Key: q.Key, Decision: DecApply, Actor: "pm-1", Why: "ngẫu nhiên"}
		if tgt != nil {
			cmd.Target = tgt.Key
		}
		if tgt == nil || g.rng.Intn(5) == 0 {
			cmd.Decision = DecReject
		}
		allow[q.Key] = cmd.Decision == DecApply
		_, err := g.st.Decide(g.ctx, cmd)
		return fmt.Sprintf("%s %s → %s", cmd.Decision, q.Key, cmd.Target), allow, err
	case roll < 76: // thu hồi (nguyên thủy áp dụng: cũng là cách người APPLY một lần giao thu hồi)
		r := g.choose(rs, live)
		if r == nil {
			return "bỏ qua (không có bản còn sống)", allow, nil
		}
		return "retract " + r.Key, allow, g.st.Retract(g.ctx, r.Key, "ingest", "thu hồi ngẫu nhiên")
	default: // sửa tin
		r := g.choose(rs, live)
		if r == nil {
			return "bỏ qua (không có bản còn sống)", allow, nil
		}
		g.amends++
		it := mk(r.Key, g.minutes[g.rng.Intn(len(g.minutes))], withRaw(fmt.Sprintf("sửa lần %d", g.amends)))
		how := "giờ gửi"
		switch g.rng.Intn(6) {
		case 0:
			it.Value.Number, how = g.pick([]string{"1000", "2000", "3000"}), "giá trị"
		case 1:
			it.Verdict = Verdict{Outcome: OutcomeHeld, Reason: ReasonAmbiguousEntity, Detail: "sửa thành chờ"}
			how = "HELD vì lý do khác"
		case 2:
			it.Entity, how = "E2", "thực thể"
		case 3:
			it.OccurredOn, how = "2026-10-20", "ngày"
		case 4:
			it.Kind, how = KindPlan, "loại"
		}
		allow[r.Key] = true
		return fmt.Sprintf("amend %s (%s)", r.Key, how), allow, g.st.Amend(g.ctx, r.Key, it, "ingest", "sửa ngẫu nhiên")
	}
}

func (g *genRun) pickDecision(a, b Decision) Decision {
	if g.rng.Intn(4) == 0 {
		return b
	}
	return a
}

// checkStep kiểm trạng thái sau MỘT sự kiện: phép kiểm tính chất, không máy nhả, lỗi thì không đổi, không bản nào biến mất.
func checkStep(t *testing.T, where, desc string, before, after []Record, allow map[string]bool, err error, w time.Duration) {
	t.Helper()
	if err != nil && !reflect.DeepEqual(before, after) {
		t.Fatalf("%s: sau %q (lỗi %v) bản ghi đổi: lệnh bị từ chối không được đổi bản ghi", where, desc, err)
	}
	if bad := oracle(after, w); len(bad) > 0 {
		t.Fatalf("%s: sau %q vi phạm bất biến:\n  %s", where, desc, strings.Join(bad, "\n  "))
	}
	prev := map[string]Record{}
	for _, r := range before {
		prev[r.Key] = r
	}
	if len(after) < len(before) {
		t.Fatalf("%s: sau %q kho mất bản ghi (%d → %d)", where, desc, len(before), len(after))
	}
	for _, r := range after {
		p, had := prev[r.Key]
		if !had {
			continue
		}
		if p.Disposition != DispAccepted && r.Disposition == DispAccepted && !allow[r.Key] {
			t.Fatalf("%s: sau %q bản %s đi từ %s sang ACCEPTED mà không do quyết định của người, sửa tin hay nạp mới: máy nhả (BB6, BB7)", where, desc, r.Key, p.Disposition)
		}
		if r.Version < p.Version {
			t.Fatalf("%s: sau %q version của %s lùi %d → %d", where, desc, r.Key, p.Version, r.Version)
		}
	}
}

func runRandomSequences(t *testing.T, st Store, seed int64, sequences int) {
	t.Helper()
	w := 60 * time.Minute
	for s := 0; s < sequences; s++ {
		ctx, _ := newCtx("prop")
		g := newGenRun(seed+int64(s), st, ctx)
		steps := 4 + g.rng.Intn(10)
		var trail []string
		for i := 0; i < steps; i++ {
			before := g.records(t)
			desc, allow, err := g.step(t)
			trail = append(trail, desc)
			after := g.records(t)
			where := fmt.Sprintf("hạt giống %d, bước %d (%s)", seed+int64(s), i+1, strings.Join(trail, " ; "))
			checkStep(t, where, desc, before, after, allow, err, w)
			checkTotals(t, ctx, st, after, where)
		}
	}
}

// R6-01: sau MỌI sự kiện của MỌI chuỗi ngẫu nhiên: 0 cộng đôi, 0 bản bị bác mà bản được giữ không còn được tính (rơi khỏi tổng), duplicate_of đúng bản đứng
// ngay trước, không máy nhả, lệnh bị từ chối không đổi bản ghi, tổng khớp tính lại.
func TestR601_PhepKiemTinhChat_SauMoiSuKien_BoSinhNgauNhien(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		n := 3000
		if env.name != "bonho" {
			n = 80
		}
		runRandomSequences(t, env.st, 20261023, n)
	})
}

// Bộ sinh phải thật sự chạm các đường mà R6-01 nhắm: đếm, trên cùng hạt giống, số chuỗi có bản REJECTED bởi MARK_DUPLICATE quay lại HELD, và số chuỗi có
// bản được giữ bị APPLY đính chính (SUPERSEDED). Nếu bộ sinh không bao giờ dựng được chuỗi đó thì phép kiểm sau mọi sự kiện chưa chứng minh gì.
func TestR601_BoSinhCoChamDuongRoiNhom(t *testing.T) {
	st := NewMemory(testOptions())
	returned, superseded, rejectedChain := 0, 0, 0
	for s := 0; s < 3000; s++ {
		ctx, _ := newCtx("cover")
		g := newGenRun(20261023+int64(s), st, ctx)
		for i, steps := 0, 4+g.rng.Intn(10); i < steps; i++ {
			if _, _, err := g.step(t); err != nil {
				continue
			}
		}
		for _, r := range g.records(t) {
			hs, _ := st.History(ctx, r.Key)
			for _, h := range hs {
				switch h.Event {
				case EventReturnedToHeld:
					returned++
				case EventSuperseded:
					superseded++
				}
			}
			if r.Disposition == DispRejected && r.Decision == DecMarkDuplicate {
				if tgt, ok, _ := st.Record(ctx, r.DuplicateOf); ok && tgt.Disposition == DispRejected {
					rejectedChain++
				}
			}
		}
	}
	t.Logf("trên 3000 chuỗi: %d lần bản bị bác quay lại HELD, %d lần bản bị SUPERSEDED, %d chuỗi bị bác nối hai mắt", returned, superseded, rejectedChain)
	if returned < 100 || superseded < 20 || rejectedChain < 10 {
		t.Fatalf("bộ sinh quá ít chạm đường rời nhóm: returned=%d superseded=%d chuỗi hai mắt=%d (muốn ≥ 100, 20, 10)", returned, superseded, rejectedChain)
	}
}

// Tất định (BB5): cùng hạt giống cho cùng trạng thái cuối, hai lần chạy độc lập.
func TestBoSinh_TatDinh_CungHatGiongCungKetQua(t *testing.T) {
	run := func() map[string]string {
		st := NewMemory(testOptions())
		ctx, _ := newCtx("det")
		out := map[string]string{}
		for s := 0; s < 200; s++ {
			g := newGenRun(777+int64(s), st, ctx)
			for i, steps := 0, 4+g.rng.Intn(10); i < steps; i++ {
				_, _, _ = g.step(t)
			}
			for k, v := range dispOf(g.records(t)) {
				out[fmt.Sprintf("%d/%s", s, k)] = v
			}
			// dọn: tenant dùng chung cho cả 200 chuỗi nên khoá phải khác nhau; bộ sinh dùng k1, k2… nên mỗi chuỗi dùng tenant riêng bên dưới
			ctx, _ = newCtx("det")
		}
		return out
	}
	a, b := run(), run()
	if !reflect.DeepEqual(a, b) {
		t.Fatal("hai lần chạy cùng hạt giống cho trạng thái cuối khác nhau (không tất định)")
	}
}

// Đánh giá lại là điểm bất động: chạy settle lần nữa trên trạng thái đã xong không đổi bản ghi nào và không thêm dòng lịch sử.
func TestSettle_LaDiemBatDong(t *testing.T) {
	m := NewMemory(testOptions())
	for s := 0; s < 600; s++ {
		ctx, tenant := newCtx("fix")
		g := newGenRun(555+int64(s), m, ctx)
		for i, steps := 0, 4+g.rng.Intn(10); i < steps; i++ {
			_, _, _ = g.step(t)
		}
		before := g.records(t)
		var histBefore int
		for _, r := range before {
			hs, _ := m.History(ctx, r.Key)
			histBefore += len(hs)
		}
		if err := m.run(ctx, func(c *opctx) error {
			for _, r := range before {
				rec := r
				c.touch(&rec)
			}
			return c.settle()
		}); err != nil {
			t.Fatalf("tenant %s: settle lần hai: %v", tenant, err)
		}
		after := g.records(t)
		if !reflect.DeepEqual(before, after) {
			t.Fatalf("hạt giống %d: settle lần hai đổi bản ghi (không phải điểm bất động)", 555+int64(s))
		}
		var histAfter int
		for _, r := range after {
			hs, _ := m.History(ctx, r.Key)
			histAfter += len(hs)
		}
		if histBefore != histAfter {
			t.Fatalf("hạt giống %d: settle lần hai thêm %d dòng lịch sử", 555+int64(s), histAfter-histBefore)
		}
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// R6-02: mọi đường rời nhóm mà mã có × trạng thái của bản được giữ

type exitWay struct {
	name string
	// returns: bản bị bác theo k phải quay lại HELD (k thôi được tính); false: bản bị bác vẫn REJECTED (k vẫn được tính hoặc đi theo bản còn sống)
	returns func(kState string) bool
	do      func(t *testing.T, ctx context.Context, st Store, key string)
}

func exitWays() []exitWay {
	allKStates := func(string) bool { return true }
	return []exitWay{
		{"thu hồi", allKStates, func(t *testing.T, ctx context.Context, st Store, k string) { mustRetract(t, ctx, st, k) }},
		{"sửa sang giá trị khác", allKStates, func(t *testing.T, ctx context.Context, st Store, k string) {
			mustAmend(t, ctx, st, k, mk(k, 0, withValue("2000"), withRaw("sửa giá trị")))
		}},
		{"sửa sang loại khác", allKStates, func(t *testing.T, ctx context.Context, st Store, k string) {
			mustAmend(t, ctx, st, k, mk(k, 0, withKind(KindPlan), withRaw("sửa loại")))
		}},
		{"sửa sang thực thể khác", allKStates, func(t *testing.T, ctx context.Context, st Store, k string) {
			mustAmend(t, ctx, st, k, mk(k, 0, withEntity("E2"), withRaw("sửa thực thể")))
		}},
		{"sửa sang ngày khác", allKStates, func(t *testing.T, ctx context.Context, st Store, k string) {
			mustAmend(t, ctx, st, k, mk(k, 0, withDay("2026-10-20"), withRaw("sửa ngày")))
		}},
		{"sửa thành HELD vì lý do khác", allKStates, func(t *testing.T, ctx context.Context, st Store, k string) {
			mustAmend(t, ctx, st, k, mk(k, 0, heldFor(ReasonAmbiguousEntity), withRaw("sửa thành chờ")))
		}},
		{"sửa thành QUARANTINED", allKStates, func(t *testing.T, ctx context.Context, st Store, k string) {
			mustAmend(t, ctx, st, k, mk(k, 0, quarantinedFor(ReasonUnparseable), withEntity(""), withDay(""), withValue(""), withRaw("sửa thành cách ly")))
		}},
		{"APPLY đính chính (SUPERSEDED)", allKStates, func(t *testing.T, ctx context.Context, st Store, k string) {
			mustDeliver(t, ctx, st, "lo-q", 1, mk("q", 900, heldFor(ReasonCorrectionUnconfirmed), withKind(KindCorrection), withValue("3000")))
			mustDecide(t, ctx, st, Command{Key: "q", Decision: DecApply, Actor: "pm-1", Why: "báo nhầm", Target: k})
		}},
		// bản được giữ rời nhóm bằng chính MARK_DUPLICATE: bản bị bác đi theo, KHÔNG quay lại (chỉ áp cho k đang HELD possible_duplicate)
		{"MARK_DUPLICATE trên bản được giữ", func(s string) bool { return false }, func(t *testing.T, ctx context.Context, st Store, k string) { markDup(t, ctx, st, k) }},
	}
}

// R6-02: với MỌI đường rời nhóm và MỌI trạng thái của bản được giữ k (ACCEPTED, HELD possible_duplicate, REJECTED bởi MARK_DUPLICATE): sau khi k rời đi,
// bản b bị bác theo k hoặc quay lại HELD (hàng chờ có người nhận) hoặc vẫn REJECTED theo một bản còn được tính; phép kiểm tính chất đúng; tổng không cộng đôi.
func TestR602_MoiDuongRoiNhom_KhongBanNaoRoiKhoiTong(t *testing.T) {
	type kState struct {
		name  string
		build func(t *testing.T, ctx context.Context, st Store) // dựng z (nếu cần), k, b: b REJECTED theo k
	}
	states := []kState{
		{"k ACCEPTED", func(t *testing.T, ctx context.Context, st Store) {
			mustBatch(t, ctx, st, mk("k", 0), mk("b", 30))
			markDup(t, ctx, st, "b")
		}},
		{"k HELD possible_duplicate", func(t *testing.T, ctx context.Context, st Store) {
			mustBatch(t, ctx, st, mk("z", 0), mk("k", 30), mk("b", 60))
			markDup(t, ctx, st, "b") // b REJECTED theo k (k HELD theo z)
		}},
		{"k REJECTED bởi MARK_DUPLICATE", func(t *testing.T, ctx context.Context, st Store) {
			mustBatch(t, ctx, st, mk("z", 0), mk("k", 30), mk("b", 60))
			markDup(t, ctx, st, "b") // b theo k
			markDup(t, ctx, st, "k") // k theo z; b đi theo k
		}},
	}
	forEachStore(t, func(t *testing.T, env storeEnv) {
		for _, ks := range states {
			for _, w := range exitWays() {
				if w.name == "MARK_DUPLICATE trên bản được giữ" && ks.name != "k HELD possible_duplicate" {
					continue // chỉ bản HELD possible_duplicate mới MARK_DUPLICATE được
				}
				ctx, _ := newCtx("r602")
				name := ks.name + " / " + w.name
				ks.build(t, ctx, env.st)
				w.do(t, ctx, env.st, "k")
				recs := mustRecords(t, ctx, env.st)
				if bad := oracle(recs, 60*time.Minute); len(bad) > 0 {
					t.Fatalf("%s: vi phạm:\n  %s", name, strings.Join(bad, "\n  "))
				}
				checkTotals(t, ctx, env.st, recs, name)
				b := mustRecord(t, ctx, env.st, "b")
				if w.returns(ks.name) {
					if b.Disposition != DispHeld || b.Reason != ReasonPossibleDuplicate {
						t.Errorf("%s: b = %s/%s, muốn quay lại HELD possible_duplicate (k thôi được tính)", name, b.Disposition, b.Reason)
					}
				} else if b.Disposition != DispRejected || b.DuplicateOf != "k" {
					t.Errorf("%s: b = %s dup=%q, muốn vẫn REJECTED theo k (k rời nhóm bằng chính MARK_DUPLICATE: b đi theo)", name, b.Disposition, b.DuplicateOf)
				}
			}
		}
	})
}
