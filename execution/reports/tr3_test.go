package reports

// Bước 3b (HOP-DONG §14.2 mục 3): TR3, nghi trùng trong cửa sổ W. Thứ tự chuẩn, đánh giá lại, quyết định của người, bản rời nhóm
// (§5.1, §5.6, §6.2). Mọi test chạy trên mọi kho có sẵn (forEachStore). Tên test mang mã lớp ca LC của §10; lc_map_test.go tra bảng
// "lớp ca → test" từ tên. W = 60 phút (mặc định); giờ gửi = t0 + số phút.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// ---------------------------------------------------------------------------------------------------------------------
// Đồ nghề của người quyết

func str(s string) *string { return &s }

// screen là cái người quyết đang thấy trên màn hình: duplicate_of hiện tại của bản ghi (rỗng khi bản đứng trước đã rời nhóm).
func screen(t *testing.T, ctx context.Context, st Store, key string) *string {
	t.Helper()
	return str(mustRecord(t, ctx, st, key).DuplicateOf)
}

func mustDecide(t *testing.T, ctx context.Context, st Store, cmd Command) Record {
	t.Helper()
	r, err := st.Decide(ctx, cmd)
	if err != nil {
		t.Fatalf("Decide(%s %s): %v", cmd.Decision, cmd.Key, err)
	}
	return r
}

func keepBoth(t *testing.T, ctx context.Context, st Store, key string) Record {
	t.Helper()
	return mustDecide(t, ctx, st, Command{Key: key, Decision: DecKeepBoth, Actor: "pm-1", Why: "hai lần cấp thật", SeenDuplicateOf: screen(t, ctx, st, key)})
}

func markDup(t *testing.T, ctx context.Context, st Store, key string) Record {
	t.Helper()
	return mustDecide(t, ctx, st, Command{Key: key, Decision: DecMarkDuplicate, Actor: "pm-1", Why: "gửi lại", SeenDuplicateOf: screen(t, ctx, st, key)})
}

func mustRetract(t *testing.T, ctx context.Context, st Store, key string) {
	t.Helper()
	if err := st.Retract(ctx, key, "ingest", "thu hồi tin"); err != nil {
		t.Fatalf("Retract(%s): %v", key, err)
	}
}

func mustAmend(t *testing.T, ctx context.Context, st Store, key string, it Item) {
	t.Helper()
	if err := st.Amend(ctx, key, it, "ingest", "sửa tin"); err != nil {
		t.Fatalf("Amend(%s): %v", key, err)
	}
}

// wantHeldPD khẳng định bản ghi đang HELD possible_duplicate và trỏ đúng bản đứng trước.
func wantHeldPD(t *testing.T, ctx context.Context, st Store, key, dupOf string) Record {
	t.Helper()
	r := wantRec(t, ctx, st, key, DispHeld, ReasonPossibleDuplicate)
	if r.DuplicateOf != dupOf {
		t.Fatalf("bản %s: duplicate_of = %q, muốn %q (detail %q)", key, r.DuplicateOf, dupOf, r.Detail)
	}
	if r.Recipient == "" || r.Detail == "" {
		t.Fatalf("bản %s HELD thiếu người nhận %q hoặc chi tiết %q (BB2, CL2)", key, r.Recipient, r.Detail)
	}
	return r
}

func events(t *testing.T, ctx context.Context, st Store, key string) []string {
	t.Helper()
	hs, err := st.History(ctx, key)
	if err != nil {
		t.Fatalf("History(%s): %v", key, err)
	}
	out := make([]string, len(hs))
	for i, h := range hs {
		out[i] = h.Event
	}
	return out
}

func hasEvent(evs []string, want string) bool {
	for _, e := range evs {
		if e == want {
			return true
		}
	}
	return false
}

func wantTotal(t *testing.T, ctx context.Context, st Store, want string) {
	t.Helper()
	if got := totalOf(t, ctx, st, "E1"); got != want {
		t.Errorf("tổng (E1, 2026-10-19, L) = %s, muốn %s", got, want)
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// LC24 đến LC27: nghi trùng trong cửa sổ W

// LC24: gửi lại sau vài phút, cùng người, cùng chữ, khác mã tin: TR1 và TR2 không bắt (khác khoá, khác giờ); TR3: bản gửi sau HELD
// possible_duplicate trỏ bản trước; không vào tổng; dòng sổ ghi kết cục lúc xử lý.
func TestLC24_GuiLaiSauVaiPhut_CungNguoiCungChu_BanSauHeldPossibleDuplicate(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc24")
		same := func(it *Item) { it.SenderID, it.Raw = "S1", "máy 03 cấp 1k" }
		first := mustDeliver(t, ctx, env.st, "b1", 1, mk("M-1", 0, same))
		again := mustDeliver(t, ctx, env.st, "b1", 2, mk("M-2", 5, same))
		if first.Outcome != OutcomeAccepted {
			t.Fatalf("bản đầu: %+v", first)
		}
		if again.Outcome != OutcomeHeld || again.Reason != ReasonPossibleDuplicate || again.RecordRef != "M-2" || again.Detail == "" {
			t.Fatalf("bản gửi sau: dòng sổ %+v, muốn HELD possible_duplicate có chi tiết", again)
		}
		rec := wantHeldPD(t, ctx, env.st, "M-2", "M-1")
		if !strings.Contains(rec.Detail, "M-1") {
			t.Errorf("chi tiết %q phải nêu bản đứng trước M-1", rec.Detail)
		}
		wantRec(t, ctx, env.st, "M-1", DispAccepted, "")
		wantTotal(t, ctx, env.st, "1000")
	})
}

// LC25: người khác báo cùng (máy, ngày, giá trị, đơn vị) trong W: bản gửi sau HELD. Biên: đúng W vẫn HELD (≤ W); quá W thì cả hai vào.
func TestLC25_NguoiKhacBaoCungTrongW_BanGuiSauHeld_BienW(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		for _, c := range []struct {
			gap    int
			held   bool
			wantTt string
		}{{30, true, "1000"}, {60, true, "1000"}, {61, false, "2000"}} {
			ctx, _ := newCtx("lc25")
			mustDeliver(t, ctx, env.st, "b1", 1, mk("A", 0, withSender("S1"), withRaw("a")))
			e := mustDeliver(t, ctx, env.st, "b1", 2, mk("B", c.gap, withSender("S2"), withRaw("b")))
			if c.held {
				if e.Outcome != OutcomeHeld || e.Reason != ReasonPossibleDuplicate {
					t.Errorf("cách %d phút: dòng sổ %+v, muốn HELD possible_duplicate", c.gap, e)
				}
				wantHeldPD(t, ctx, env.st, "B", "A")
			} else {
				if e.Outcome != OutcomeAccepted {
					t.Errorf("cách %d phút (> W): dòng sổ %+v, muốn ACCEPTED", c.gap, e)
				}
				wantRec(t, ctx, env.st, "B", DispAccepted, "")
			}
			wantTotal(t, ctx, env.st, c.wantTt)
		}
	})
}

// LC26: không gộp oan. Hai tin cùng giá trị cách nhau hơn W; hoặc trong W nhưng khác giá trị, thực thể, ngày, đơn vị, loại: cả hai vào.
func TestLC26_KhongGopOan_HaiTinKhacNhomHoacXaHonW_CaHaiAccepted(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc26")
		items := []Item{
			mk("far-1", 0), mk("far-2", 90, withRaw("khác chữ")), // cùng giá trị, cách 90 phút > W
			mk("val-1", 300), mk("val-2", 301, withValue("1001")), // khác giá trị
			mk("ent-1", 400), mk("ent-2", 401, withEntity("E2")), // khác thực thể
			mk("day-1", 500), mk("day-2", 501, withDay("2026-10-20")), // khác ngày
			mk("uni-1", 600), mk("uni-2", 601, func(it *Item) { it.Value.Unit = "m3" }), // khác đơn vị
			mk("knd-1", 700), mk("knd-2", 701, withKind(KindPlan)), // khác loại
		}
		rc := mustBatch(t, ctx, env.st, items...)
		if rc.ByOutcome[OutcomeAccepted] != len(items) {
			t.Fatalf("biên nhận %v, muốn %d ACCEPTED (không bản nào bị gộp oan)", rc.ByOutcome, len(items))
		}
		for _, it := range items {
			wantRec(t, ctx, env.st, it.Key, DispAccepted, "")
		}
	})
}

// LC27: hai lần cấp thật cách nhau dưới W, cùng giá trị: HELD; người KEEP_BOTH thì cả hai vào tổng; MARK_DUPLICATE thì bản bị giữ REJECTED.
func TestLC27_HaiLanCapThat_KeepBothVaMarkDuplicate(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc27a")
		mustBatch(t, ctx, env.st, mk("M-1", 0), mk("M-2", 20))
		wantHeldPD(t, ctx, env.st, "M-2", "M-1")
		wantTotal(t, ctx, env.st, "1000")
		got := keepBoth(t, ctx, env.st, "M-2")
		if got.Disposition != DispAccepted || got.Reason != "" || got.Decision != DecKeepBoth || got.DecidedBy != "pm-1" || got.DecidedAt.IsZero() {
			t.Fatalf("sau KEEP_BOTH: %+v", got)
		}
		if len(got.ReviewedPairs) != 1 || got.ReviewedPairs[0] != "M-1" {
			t.Errorf("cặp đã xét = %v, muốn [M-1]", got.ReviewedPairs)
		}
		wantTotal(t, ctx, env.st, "2000")
		wantRec(t, ctx, env.st, "M-1", DispAccepted, "")
		if evs := events(t, ctx, env.st, "M-2"); !hasEvent(evs, EventDecision) {
			t.Errorf("lịch sử %v thiếu %s (ai, khi nào, vì sao)", evs, EventDecision)
		}
		// cặp đã xét không bị giữ lại lần nữa dù có sự kiện khác chạm cùng nhóm
		mustDeliver(t, ctx, env.st, "b2", 1, mk("far", 500))
		wantRec(t, ctx, env.st, "M-2", DispAccepted, "")

		ctx2, _ := newCtx("lc27b")
		mustBatch(t, ctx2, env.st, mk("M-1", 0), mk("M-2", 20))
		rej := markDup(t, ctx2, env.st, "M-2")
		if rej.Disposition != DispRejected || rej.DuplicateOf != "M-1" || rej.Decision != DecMarkDuplicate {
			t.Fatalf("sau MARK_DUPLICATE: %+v", rej)
		}
		wantTotal(t, ctx2, env.st, "1000")
	})
}

// ---------------------------------------------------------------------------------------------------------------------
// LC63, LC75: thứ tự chuẩn, đánh giá lại khi bản gửi sớm đến muộn

func permutations(n int) [][]int {
	var out [][]int
	var rec func(cur []int, used []bool)
	rec = func(cur []int, used []bool) {
		if len(cur) == n {
			out = append(out, append([]int(nil), cur...))
			return
		}
		for i := 0; i < n; i++ {
			if !used[i] {
				used[i] = true
				rec(append(cur, i), used)
				used[i] = false
			}
		}
	}
	rec(nil, make([]bool, n))
	return out
}

// LC63 (chỉ đơn vị thường): đảo thứ tự đến của cả lô và giữa các lô không đổi tổng, cờ, tập HELD. Mọi hoán vị của sáu đơn vị gồm một chuỗi
// ba bản nối tiếp (0, 50, 100: mỗi cặp liền kề dưới W, hai đầu cách hơn W), hai nhóm khác và một thực thể khác.
func TestLC63_DaoThuTuDen_CungTongCungTapHeld(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		units := []Item{mk("p1", 0), mk("p2", 50), mk("p3", 100), mk("p4", 110, withEntity("E2")), mk("p5", 300, withValue("2000")), mk("p6", 330, withValue("2000"))}
		want := map[string]string{
			"p1": "ACCEPTED//", "p2": "HELD/possible_duplicate/p1", "p3": "HELD/possible_duplicate/p2",
			"p4": "ACCEPTED//", "p5": "ACCEPTED//", "p6": "HELD/possible_duplicate/p5",
		}
		step := 1
		if env.name != "bonho" {
			step = 17 // PG: lấy mẫu hoán vị (720 hoán vị trên PG tốn nhiều giờ giao dịch); kho bộ nhớ chạy đủ
		}
		for i, perm := range permutations(len(units)) {
			if i%step != 0 {
				continue
			}
			ctx, _ := newCtx("lc63")
			var order []Item
			for _, k := range perm {
				order = append(order, units[k])
			}
			if i%2 == 0 {
				for j, it := range order {
					mustDeliver(t, ctx, env.st, "b-"+fmt.Sprint(j/2), j%2+1, it) // chia lô hai đơn vị
				}
			} else {
				mustBatch(t, ctx, env.st, order...) // một lô
			}
			got := dispOf(mustRecords(t, ctx, env.st))
			for k, w := range want {
				if got[k] != w {
					t.Fatalf("thứ tự đến %v: bản %s = %s, muốn %s", perm, k, got[k], w)
				}
			}
			if tt := totalOf(t, ctx, env.st, "E1"); tt != "3000" {
				t.Fatalf("thứ tự đến %v: tổng E1 = %s, muốn 3000 (p1 + p5)", perm, tt)
			}
		}
	})
}

// LC75 (i): bản gửi sớm hơn ĐẾN MUỘN. b (gửi 30) đã ACCEPTED; a (gửi 0) đến sau và đứng trước nó: a ACCEPTED, b chuyển HELD (có lịch sử),
// tổng một lần giá trị; dòng sổ cũ của b vẫn ACCEPTED (sổ chỉ thêm, ghi kết cục lúc xử lý).
func TestLC75_BanGuiSomDenMuon_BanDaAcceptedChuyenHeldCoLichSu(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc75a")
		first := mustDeliver(t, ctx, env.st, "b1", 1, mk("b", 30))
		wantRec(t, ctx, env.st, "b", DispAccepted, "")
		wantTotal(t, ctx, env.st, "1000")
		late := mustDeliver(t, ctx, env.st, "b2", 1, mk("a", 0))
		if late.Outcome != OutcomeAccepted {
			t.Fatalf("bản gửi sớm đến muộn: dòng sổ %+v, muốn ACCEPTED", late)
		}
		wantRec(t, ctx, env.st, "a", DispAccepted, "")
		rb := wantHeldPD(t, ctx, env.st, "b", "a")
		if !hasEvent(events(t, ctx, env.st, "b"), EventTR3Hold) {
			t.Errorf("lịch sử của b %v thiếu %s", events(t, ctx, env.st, "b"), EventTR3Hold)
		}
		if rb.Detail == "" {
			t.Error("bản chuyển HELD phải có chi tiết (CL2)")
		}
		wantTotal(t, ctx, env.st, "1000")
		led, _ := env.st.Ledger(ctx, first.BatchID)
		if len(led) != 1 || led[0].Outcome != OutcomeAccepted {
			t.Errorf("dòng sổ cũ của b = %+v: sổ chỉ thêm, kết cục lúc xử lý là ACCEPTED", led)
		}
	})
}

// LC75 (ii): chuỗi ba bản, mỗi cặp liền kề dưới W nhưng hai đầu cách hơn W, đến theo MỌI thứ tự: kết cục cuối giống nhau.
func TestLC75_ChuoiBaBan_MoiThuTuDen_CungKetCuc(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		units := []Item{mk("a", 0), mk("b", 50), mk("c", 100)}
		for _, perm := range permutations(3) {
			ctx, _ := newCtx("lc75b")
			for j, k := range perm {
				mustDeliver(t, ctx, env.st, fmt.Sprintf("lo-%d", j), 1, units[k])
			}
			wantRec(t, ctx, env.st, "a", DispAccepted, "")
			wantHeldPD(t, ctx, env.st, "b", "a")
			wantHeldPD(t, ctx, env.st, "c", "b")
			wantTotal(t, ctx, env.st, "1000")
		}
	})
}

// LC75 (iii): cặp mà người đã KEEP_BOTH không bị giữ lại lần nữa; cặp mới (bản đến muộn chen vào) thì chưa xét nên vẫn bị giữ.
func TestLC75_CapDaKeepBoth_KhongBiGiuLai_CapMoiThiGiu(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc75c")
		mustBatch(t, ctx, env.st, mk("a", 0), mk("b", 30))
		keepBoth(t, ctx, env.st, "b") // cặp đã xét: (b, a)
		wantTotal(t, ctx, env.st, "2000")
		mustDeliver(t, ctx, env.st, "b2", 1, mk("z", 500)) // chạm lại ô nhưng ở xa: không ảnh hưởng
		wantRec(t, ctx, env.st, "b", DispAccepted, "")
		wantTotal(t, ctx, env.st, "3000")
		// d gửi 10 phút, đến muộn, chen giữa a và b: d cách a dưới W nên HELD; b đứng ngay sau d, cặp (b, d) chưa ai xét nên b bị giữ
		mustDeliver(t, ctx, env.st, "b3", 1, mk("d", 10))
		wantHeldPD(t, ctx, env.st, "d", "a")
		wantHeldPD(t, ctx, env.st, "b", "d")
		wantTotal(t, ctx, env.st, "2000") // a và z
	})
}

// ---------------------------------------------------------------------------------------------------------------------
// LC77, LC78: quyết định của người không miễn TR3

// LC77: bản A đã ACCEPTED; bản B cùng nhóm, gửi sau A trong W, HELD vì lý do khác; người quyết cho B (CHOOSE, AUTHORIZE, CORRECT); cả hai
// thứ tự đến của A và B. B KHÔNG ACCEPTED: HELD possible_duplicate trỏ A; lý do trước và quyết định ghi ở lịch sử; tổng chỉ gồm A.
func TestLC77_QuyetDinhCuaNguoiKhongMienTR3(t *testing.T) {
	type variant struct {
		name string
		held Item
		cmd  Command
	}
	variants := []variant{
		{"CHOOSE sau ambiguous_entity", mk("B", 30, heldFor(ReasonAmbiguousEntity), withEntity("")), Command{Key: "B", Decision: DecChoose, Actor: "pm-1", Why: "máy này", Entity: "E1"}},
		{"AUTHORIZE sau sender_not_authorized", mk("B", 30, heldFor(ReasonSenderNotAuthorized)), Command{Key: "B", Decision: DecAuthorize, Actor: "pm-1", Why: "người gửi thay"}},
		{"CORRECT sau number_reading_uncertain", mk("B", 30, heldFor(ReasonNumberReadingUncertain), withValue("")), Command{Key: "B", Decision: DecCorrect, Actor: "pm-1", Why: "đọc lại", Value: &Quantity{Number: "1000", Unit: "L"}}},
	}
	forEachStore(t, func(t *testing.T, env storeEnv) {
		for _, v := range variants {
			for _, bFirst := range []bool{false, true} {
				for _, finish := range []string{"hold", "keep", "mark"} {
					ctx, _ := newCtx("lc77")
					a := mk("A", 0)
					if bFirst {
						mustDeliver(t, ctx, env.st, "b1", 1, v.held)
						mustDeliver(t, ctx, env.st, "b2", 1, a)
					} else {
						mustDeliver(t, ctx, env.st, "b1", 1, a)
						mustDeliver(t, ctx, env.st, "b2", 1, v.held)
					}
					name := fmt.Sprintf("%s/B-đến-trước=%v/%s", v.name, bFirst, finish)
					got := mustDecide(t, ctx, env.st, v.cmd)
					if got.Disposition != DispHeld || got.Reason != ReasonPossibleDuplicate || got.DuplicateOf != "A" {
						t.Fatalf("%s: sau quyết định B = %s/%s dup=%q, muốn HELD possible_duplicate trỏ A (cộng đôi im lặng)", name, got.Disposition, got.Reason, got.DuplicateOf)
					}
					if got.Recipient == "" || got.DecidedBy != "pm-1" || got.Decision != v.cmd.Decision {
						t.Errorf("%s: người nhận %q, quyết định %q/%q: quyết định của người phải còn ở bản ghi", name, got.Recipient, got.DecidedBy, got.Decision)
					}
					if evs := events(t, ctx, env.st, "B"); !hasEvent(evs, EventDecision) || !hasEvent(evs, EventTR3Hold) {
						t.Errorf("%s: lịch sử %v phải có cả quyết định và việc TR3 giữ lại", name, evs)
					}
					wantTotal(t, ctx, env.st, "1000")
					switch finish {
					case "keep":
						keepBoth(t, ctx, env.st, "B")
						wantRec(t, ctx, env.st, "B", DispAccepted, "")
						wantTotal(t, ctx, env.st, "2000")
					case "mark":
						markDup(t, ctx, env.st, "B")
						wantRec(t, ctx, env.st, "B", DispRejected, ReasonPossibleDuplicate)
						wantTotal(t, ctx, env.st, "1000")
					}
				}
			}
		}
	})
}

// LC77 (phần không nghi trùng): quyết định cho ra ACCEPTED khi không có bản đứng trước cách ≤ W thì ACCEPTED, và vào tổng.
func TestLC77_QuyetDinhKhongCoBanDungTruoc_Accepted(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc77x")
		mustBatch(t, ctx, env.st, mk("A", 0), mk("B", 200, heldFor(ReasonAmbiguousEntity), withEntity("")))
		got := mustDecide(t, ctx, env.st, Command{Key: "B", Decision: DecChoose, Actor: "pm-1", Why: "máy này", Entity: "E1"})
		if got.Disposition != DispAccepted || got.Entity != "E1" || got.Version != 2 {
			t.Fatalf("sau CHOOSE: %+v, muốn ACCEPTED ở E1, version 2 (bản máy đọc giữ ở lịch sử phiên bản)", got)
		}
		wantTotal(t, ctx, env.st, "2000")
	})
}

// LC78: bản B đã ACCEPTED sau quyết định (lúc quyết chưa có bản nghi trùng); sau đó bản A gửi sớm hơn B, cách ≤ W, đến muộn: A ACCEPTED, B
// chuyển HELD possible_duplicate trỏ A, quyết định cũ còn ở bản ghi; B vào hàng chờ có người nhận (không đổi im lặng); tổng một lần.
func TestLC78_BanDaQuyetBiDanhGiaLai_BanGuiSomDenMuon(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc78")
		mustDeliver(t, ctx, env.st, "b1", 1, mk("B", 20, heldFor(ReasonAmbiguousEntity), withEntity("")))
		mustDecide(t, ctx, env.st, Command{Key: "B", Decision: DecChoose, Actor: "pm-1", Why: "máy này", Entity: "E1"})
		wantRec(t, ctx, env.st, "B", DispAccepted, "")
		wantTotal(t, ctx, env.st, "1000")
		mustDeliver(t, ctx, env.st, "b2", 1, mk("A", 0))
		rb := wantHeldPD(t, ctx, env.st, "B", "A")
		if rb.Decision != DecChoose || rb.DecidedBy != "pm-1" {
			t.Errorf("quyết định cũ phải còn ở bản ghi: %q/%q", rb.Decision, rb.DecidedBy)
		}
		if !hasEvent(events(t, ctx, env.st, "B"), EventTR3Hold) {
			t.Errorf("lịch sử %v thiếu %s", events(t, ctx, env.st, "B"), EventTR3Hold)
		}
		wantTotal(t, ctx, env.st, "1000")
		keepBoth(t, ctx, env.st, "B")
		wantTotal(t, ctx, env.st, "2000")
	})
}

// ---------------------------------------------------------------------------------------------------------------------
// LC81: bản rời nhóm (R6-01 phần a, b)

type leaveWay struct {
	name string
	do   func(t *testing.T, ctx context.Context, st Store, key string)
	// totalWithout là tổng (E1, ngày, L) còn lại khi chỉ bản "kia" (giá trị 1000) rời đi: giá trị của bản rời đi, nếu còn ACCEPTED ở nhóm khác, tính thêm.
	stays string
}

func leaveWays() []leaveWay {
	return []leaveWay{
		{"thu hồi", func(t *testing.T, ctx context.Context, st Store, key string) { mustRetract(t, ctx, st, key) }, "0"},
		{"sửa sang giá trị khác", func(t *testing.T, ctx context.Context, st Store, key string) {
			mustAmend(t, ctx, st, key, mk(key, 0, withValue("2000"), withRaw("sửa "+key)))
		}, "2000"},
		{"sửa thành HELD vì lý do khác", func(t *testing.T, ctx context.Context, st Store, key string) {
			mustAmend(t, ctx, st, key, mk(key, 0, heldFor(ReasonAmbiguousEntity), withRaw("sửa "+key)))
		}, "0"},
	}
}

// LC81 (i): a, b rồi a rời nhóm. b VẪN HELD (chỉ siết, không tự nới), duplicate_of không trỏ bản đã rời mà không có lý do ở detail, tổng
// không gồm a (giá trị cũ); không bản nào rơi khỏi tổng hay hàng chờ; người KEEP_BOTH thì b vào tổng.
func TestLC81_i_BanRoiNhomSauKhiCapDaDen_BanSauVanHeld(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		for _, w := range leaveWays() {
			ctx, _ := newCtx("lc81i")
			mustBatch(t, ctx, env.st, mk("a", 0), mk("b", 30))
			wantHeldPD(t, ctx, env.st, "b", "a")
			w.do(t, ctx, env.st, "a")
			rb := wantHeldPD(t, ctx, env.st, "b", "")
			if !strings.Contains(rb.Detail, "rời nhóm") {
				t.Errorf("%s: chi tiết %q phải nói bản đứng trước đã rời nhóm", w.name, rb.Detail)
			}
			if got := totalOf(t, ctx, env.st, "E1"); got != w.stays {
				t.Errorf("%s: tổng = %s, muốn %s (không gồm a ở nhóm cũ, b chưa vào)", w.name, got, w.stays)
			}
			keepBoth(t, ctx, env.st, "b")
			wantRec(t, ctx, env.st, "b", DispAccepted, "")
			if got, want := totalOf(t, ctx, env.st, "E1"), map[string]string{"0": "1000", "2000": "3000"}[w.stays]; got != want {
				t.Errorf("%s: sau KEEP_BOTH tổng = %s, muốn %s", w.name, got, want)
			}
		}
	})
}

// LC81 (i, người bác): bản đứng trước a (đang HELD) bị người MARK_DUPLICATE: bản đứng sau nối lại duplicate_of sang bản đứng trước mới.
func TestLC81_i_NguoiBacBanDungTruoc_BanSauNoiLaiDuplicateOf(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc81b")
		mustBatch(t, ctx, env.st, mk("z", 0), mk("a", 30), mk("b", 60))
		wantHeldPD(t, ctx, env.st, "a", "z")
		wantHeldPD(t, ctx, env.st, "b", "a")
		markDup(t, ctx, env.st, "a")
		wantRec(t, ctx, env.st, "a", DispRejected, ReasonPossibleDuplicate)
		wantHeldPD(t, ctx, env.st, "b", "z") // a rời nhóm: bản đứng trước mới của b là z (cách đúng W)
		wantTotal(t, ctx, env.st, "1000")
	})
}

// LC81 (ii), (iii): a rời nhóm TRƯỚC khi b đến (a, a rời nhóm, rồi b): b ACCEPTED; tổng không gồm cả a và b.
func TestLC81_ii_iii_BanRoiNhomTruocKhiBanSauDen_BanSauAccepted(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc81ii")
		mustDeliver(t, ctx, env.st, "b1", 1, mk("a", 0))
		mustRetract(t, ctx, env.st, "a")
		e := mustDeliver(t, ctx, env.st, "b2", 1, mk("b", 30))
		if e.Outcome != OutcomeAccepted {
			t.Fatalf("b đến sau khi a đã thu hồi: %+v, muốn ACCEPTED", e)
		}
		wantRec(t, ctx, env.st, "a", DispRetracted, "")
		wantTotal(t, ctx, env.st, "1000")
	})
}

// LC81 (iv): a, m, c nối tiếp; m cách a ≤ W, c cách m ≤ W (m, c đều HELD); người KEEP_BOTH cho c; rồi m rời nhóm. Hai biến thể: c cách a ≤ W
// (c chuyển HELD trỏ a, vào hàng chờ, tổng không gồm cả a và c) và c cách a hơn W (c giữ ACCEPTED).
func TestLC81_iv_BaBanNoiTiep_KeepBothRoiMRoiNhom(t *testing.T) {
	ways := append(leaveWays(), leaveWay{"MARK_DUPLICATE", func(t *testing.T, ctx context.Context, st Store, key string) { markDup(t, ctx, st, key) }, "0"})
	forEachStore(t, func(t *testing.T, env storeEnv) {
		for _, w := range ways {
			for _, near := range []bool{true, false} {
				cMin := 60 // cách a đúng W: vẫn ≤ W
				if !near {
					cMin = 70 // cách a 70 phút (> W), cách m 40 phút (≤ W)
				}
				ctx, _ := newCtx("lc81iv")
				mustBatch(t, ctx, env.st, mk("a", 0), mk("m", 30), mk("c", cMin))
				wantHeldPD(t, ctx, env.st, "m", "a")
				wantHeldPD(t, ctx, env.st, "c", "m")
				keepBoth(t, ctx, env.st, "c") // cặp đã xét: (c, m)
				wantRec(t, ctx, env.st, "c", DispAccepted, "")
				w.do(t, ctx, env.st, "m")
				name := fmt.Sprintf("%s/c cách a ≤ W: %v", w.name, near)
				if near {
					rc := wantHeldPD(t, ctx, env.st, "c", "a")
					if rc.Decision != DecKeepBoth {
						t.Errorf("%s: quyết định KEEP_BOTH cũ phải còn ở bản ghi", name)
					}
					// tổng = a (1000) cộng giá trị của m nếu m còn ACCEPTED ở nhóm khác (w.stays); không gồm c
					if got, want := totalOf(t, ctx, env.st, "E1"), map[string]string{"0": "1000", "2000": "3000"}[w.stays]; got != want {
						t.Errorf("%s: tổng = %s, muốn %s (không gồm cả a và c)", name, got, want)
					}
					keepBoth(t, ctx, env.st, "c") // cặp (c, a) mới được xét
					wantRec(t, ctx, env.st, "c", DispAccepted, "")
				} else {
					wantRec(t, ctx, env.st, "c", DispAccepted, "")
				}
			}
		}
	})
}

// LC81 (iv), số cụ thể: tổng không gồm cả a và c khi c cách a ≤ W, và gồm cả hai khi c cách a hơn W.
func TestLC81_iv_TongKhongGomCaAVaC(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		for _, near := range []bool{true, false} {
			cMin := 70
			if near {
				cMin = 60
			}
			ctx, _ := newCtx("lc81tt")
			mustBatch(t, ctx, env.st, mk("a", 0), mk("m", 30), mk("c", cMin))
			keepBoth(t, ctx, env.st, "c")
			mustRetract(t, ctx, env.st, "m")
			want := "2000" // c cách a hơn W: a + c
			if near {
				want = "1000" // c bị giữ: chỉ a
			}
			wantTotal(t, ctx, env.st, want)
		}
	})
}

// ---------------------------------------------------------------------------------------------------------------------
// LC83: quyết định trên màn hình cũ

func wantStale(t *testing.T, name string, err error, seen, current string) {
	t.Helper()
	var se *StaleDecisionError
	if !errors.Is(err, ErrStaleDecision) || !errors.As(err, &se) || se.Seen != seen || se.Current != current {
		t.Fatalf("%s: muốn StaleDecisionError (thấy %q, hiện tại %q), được %v", name, seen, current, err)
	}
	if contracts.FaultOf(err) != contracts.FaultInput {
		t.Errorf("%s: lỗi quyết định trên màn hình cũ phải mang nhãn %s, được %q", name, contracts.FaultInput, contracts.FaultOf(err))
	}
}

// LC83 (i): b HELD, duplicate_of = a; người quyết đã thấy a; a bị thu hồi (b còn HELD, duplicate_of rỗng); lệnh MARK_DUPLICATE và (ở ca
// riêng) KEEP_BOTH mang duplicate_of đã thấy (a): bị từ chối kèm audit và lý do "màn hình cũ"; b vẫn HELD, không REJECTED, không cặp mới.
func TestLC83_i_BanKiaDaBiThuHoi_LenhTrenManHinhCuBiTuChoi(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		for _, d := range []Decision{DecMarkDuplicate, DecKeepBoth} {
			ctx, _ := newCtx("lc83i")
			mustBatch(t, ctx, env.st, mk("a", 0), mk("b", 30))
			seen := screen(t, ctx, env.st, "b") // "a"
			mustRetract(t, ctx, env.st, "a")
			before := mustRecord(t, ctx, env.st, "b")
			_, err := env.st.Decide(ctx, Command{Key: "b", Decision: d, Actor: "pm-1", Why: "trùng", SeenDuplicateOf: seen})
			wantStale(t, string(d), err, "a", "")
			after := wantHeldPD(t, ctx, env.st, "b", "")
			if len(after.ReviewedPairs) != len(before.ReviewedPairs) || after.Decision != before.Decision {
				t.Errorf("%s: bản ghi đổi sau lệnh bị từ chối: %+v → %+v", d, before, after)
			}
			hs, _ := env.st.History(ctx, "b")
			last := hs[len(hs)-1]
			if last.Event != EventRefused || !strings.Contains(last.Detail, `"a"`) {
				t.Errorf("%s: dòng audit cuối %+v, muốn %s nêu giá trị đã thấy", d, last, EventRefused)
			}
			wantTotal(t, ctx, env.st, "0")
			// lệnh gửi lại mang duplicate_of hiện tại (rỗng): MARK_DUPLICATE bị từ chối vì không có bản kia; KEEP_BOTH được nhận
			_, err = env.st.Decide(ctx, Command{Key: "b", Decision: DecMarkDuplicate, Actor: "pm-1", SeenDuplicateOf: str("")})
			if !errors.Is(err, ErrNoCounterpart) {
				t.Errorf("MARK_DUPLICATE khi duplicate_of rỗng: muốn ErrNoCounterpart, được %v", err)
			}
			wantHeldPD(t, ctx, env.st, "b", "")
			got := mustDecide(t, ctx, env.st, Command{Key: "b", Decision: DecKeepBoth, Actor: "pm-1", Why: "bản kia đã thu hồi", SeenDuplicateOf: str("")})
			if got.Disposition != DispAccepted || len(got.ReviewedPairs) != 0 {
				t.Errorf("KEEP_BOTH khi duplicate_of rỗng đã khớp màn hình: %+v, muốn ACCEPTED, không thêm cặp", got)
			}
			wantTotal(t, ctx, env.st, "1000")
		}
	})
}

// LC83 (ii): một bản d gửi sớm hơn b nhưng muộn hơn a, đến muộn, chen giữa: duplicate_of của b đổi sang d; lệnh mang a bị từ chối.
func TestLC83_ii_BanKhacChenVaoGiua_LenhTrenManHinhCuBiTuChoi(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		for _, d := range []Decision{DecMarkDuplicate, DecKeepBoth} {
			ctx, _ := newCtx("lc83ii")
			mustBatch(t, ctx, env.st, mk("a", 0), mk("b", 50))
			seen := screen(t, ctx, env.st, "b")
			mustDeliver(t, ctx, env.st, "b2", 1, mk("d", 20)) // đến muộn, chen giữa
			wantHeldPD(t, ctx, env.st, "b", "d")
			_, err := env.st.Decide(ctx, Command{Key: "b", Decision: d, Actor: "pm-1", SeenDuplicateOf: seen})
			wantStale(t, string(d), err, "a", "d")
			wantHeldPD(t, ctx, env.st, "b", "d")
			if len(mustRecord(t, ctx, env.st, "b").ReviewedPairs) != 0 {
				t.Errorf("%s: lệnh bị từ chối không được ghi cặp đã xét", d)
			}
		}
		ctx, _ := newCtx("lc83ii2")
		mustBatch(t, ctx, env.st, mk("a", 0), mk("b", 50), mk("d", 20))
		got := markDup(t, ctx, env.st, "b") // màn hình mới: duplicate_of = d
		if got.Disposition != DispRejected || got.DuplicateOf != "d" {
			t.Errorf("lệnh mang duplicate_of hiện tại phải được nhận: %+v", got)
		}
	})
}

// LC83 (thiếu trường, R4-03): lệnh không mang seen_duplicate_of bị từ chối (đóng mặc định); "có mặt nhưng rỗng" khác "thiếu".
func TestLC83_LenhThieuSeenDuplicateOf_BiTuChoi(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		for _, d := range []Decision{DecMarkDuplicate, DecKeepBoth} {
			ctx, _ := newCtx("lc83m")
			mustBatch(t, ctx, env.st, mk("a", 0), mk("b", 30))
			_, err := env.st.Decide(ctx, Command{Key: "b", Decision: d, Actor: "pm-1"})
			if !errors.Is(err, ErrMissingSeen) || contracts.FaultOf(err) != contracts.FaultInput {
				t.Errorf("%s thiếu seen_duplicate_of: muốn ErrMissingSeen nhãn input, được %v", d, err)
			}
			wantHeldPD(t, ctx, env.st, "b", "a")
		}
	})
}

// ---------------------------------------------------------------------------------------------------------------------
// LC84: bản được giữ của MARK_DUPLICATE rời nhóm SAU quyết định (R6-01: bất biến theo sự kiện)

// bắt đầu dựng cặp a, b: b HELD (duplicate_of = a) rồi MARK_DUPLICATE b: b REJECTED theo a.
func markedPair(t *testing.T, ctx context.Context, st Store) {
	t.Helper()
	mustBatch(t, ctx, st, mk("a", 0), mk("b", 30))
	rej := markDup(t, ctx, st, "b")
	if rej.Disposition != DispRejected {
		t.Fatalf("b sau MARK_DUPLICATE: %+v", rej)
	}
	wantTotal(t, ctx, st, "1000")
}

// wantReturned khẳng định b đã quay lại HELD possible_duplicate vào hàng chờ có người nhận, quyết định MARK_DUPLICATE cũ còn ở bản ghi và
// lịch sử, rồi người KEEP_BOTH thì b vào tổng.
func wantReturned(t *testing.T, ctx context.Context, st Store, key, dupOf string) {
	t.Helper()
	rb := wantHeldPD(t, ctx, st, key, dupOf)
	if rb.Decision != DecMarkDuplicate || rb.DecidedBy == "" {
		t.Errorf("bản %s: quyết định MARK_DUPLICATE cũ phải còn ở bản ghi, được %q/%q", key, rb.Decision, rb.DecidedBy)
	}
	if !hasEvent(events(t, ctx, st, key), EventReturnedToHeld) {
		t.Errorf("lịch sử của %s %v thiếu %s", key, events(t, ctx, st, key), EventReturnedToHeld)
	}
	if dupOf == "" && !strings.Contains(rb.Detail, "đã rời nhóm") {
		t.Errorf("bản %s: duplicate_of rỗng phải kèm chi tiết nói bản được giữ đã rời nhóm, được %q", key, rb.Detail)
	}
}

// LC84 (i), (ii), (iii): bản được giữ a bị thu hồi / sửa sang nhóm khác / sửa thành HELD vì lý do khác: b quay lại HELD, duplicate_of rỗng kèm
// chi tiết; tổng không gồm a ở nhóm cũ; không bản nào rơi khỏi cả tổng lẫn hàng chờ; người KEEP_BOTH thì b vào tổng.
func TestLC84_i_ii_iii_BanDuocGiuRoiNhomSauQuyetDinh_BanBiBacQuayLaiHeld(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		for _, w := range leaveWays() {
			ctx, _ := newCtx("lc84a")
			markedPair(t, ctx, env.st)
			w.do(t, ctx, env.st, "a")
			wantReturned(t, ctx, env.st, "b", "")
			if got := totalOf(t, ctx, env.st, "E1"); got != w.stays {
				t.Errorf("%s: tổng = %s, muốn %s", w.name, got, w.stays)
			}
			keepBoth(t, ctx, env.st, "b")
			wantRec(t, ctx, env.st, "b", DispAccepted, "")
		}
	})
}

// LC84 (iv): bản được giữ rời nhóm bằng chính MARK_DUPLICATE thì bản bị bác đi theo bản được giữ mới, vẫn REJECTED. z, a, b mỗi bản cách bản đứng
// trước ≤ W; b REJECTED theo a, rồi a REJECTED theo z: b vẫn REJECTED (không quay lại). z bị thu hồi: a quay lại HELD, b vẫn REJECTED theo a.
func TestLC84_iv_BanDuocGiuRoiBangChinhMarkDuplicate_BanBiBacDiTheo(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc84iv")
		mustBatch(t, ctx, env.st, mk("z", 0), mk("a", 30), mk("b", 60))
		wantHeldPD(t, ctx, env.st, "a", "z")
		wantHeldPD(t, ctx, env.st, "b", "a")
		markDup(t, ctx, env.st, "b") // b REJECTED theo a
		markDup(t, ctx, env.st, "a") // a REJECTED theo z
		wantRec(t, ctx, env.st, "a", DispRejected, ReasonPossibleDuplicate)
		rb := wantRec(t, ctx, env.st, "b", DispRejected, ReasonPossibleDuplicate)
		if rb.DuplicateOf != "a" {
			t.Errorf("b phải vẫn trỏ a (đi theo bản được giữ mới), được %q", rb.DuplicateOf)
		}
		if hasEvent(events(t, ctx, env.st, "b"), EventReturnedToHeld) {
			t.Error("b không được quay lại HELD khi bản được giữ rời nhóm bằng chính MARK_DUPLICATE")
		}
		wantTotal(t, ctx, env.st, "1000") // z
		mustRetract(t, ctx, env.st, "z")
		wantReturned(t, ctx, env.st, "a", "")
		wantRec(t, ctx, env.st, "b", DispRejected, ReasonPossibleDuplicate) // theo a, đang ở hàng chờ
		wantTotal(t, ctx, env.st, "0")
	})
}

// LC84 (v): a, b, c nối tiếp; MARK_DUPLICATE b rồi c (cả hai theo a); a bị thu hồi: b và c cùng quay lại HELD, duplicate_of của b rỗng, của c là b;
// người KEEP_BOTH cho b rồi cho c thì cả hai vào tổng.
func TestLC84_v_HaiBanBiBacCungTheoMotBan_CungQuayLai(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc84v")
		mustBatch(t, ctx, env.st, mk("a", 0), mk("b", 30), mk("c", 60))
		markDup(t, ctx, env.st, "b")
		wantHeldPD(t, ctx, env.st, "c", "a") // b rời nhóm: bản đứng trước mới của c là a
		markDup(t, ctx, env.st, "c")
		mustRetract(t, ctx, env.st, "a")
		wantReturned(t, ctx, env.st, "b", "")
		wantReturned(t, ctx, env.st, "c", "b")
		wantTotal(t, ctx, env.st, "0")
		keepBoth(t, ctx, env.st, "b")
		keepBoth(t, ctx, env.st, "c")
		wantTotal(t, ctx, env.st, "2000")
	})
}

// LC84 (vi): a đã có quyết định của người (BB7); thu hồi a (nguyên thủy áp dụng, như khi người APPLY lần giao thu hồi): b quay lại HELD.
func TestLC84_vi_BanDuocGiuDaCoQuyetDinhCuaNguoi_ThuHoiThiBanBiBacQuayLai(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc84vi")
		mustDeliver(t, ctx, env.st, "b1", 1, mk("a", 0, heldFor(ReasonAmbiguousEntity), withEntity("")))
		mustDecide(t, ctx, env.st, Command{Key: "a", Decision: DecChoose, Actor: "pm-1", Why: "máy này", Entity: "E1"})
		mustDeliver(t, ctx, env.st, "b2", 1, mk("b", 30))
		wantHeldPD(t, ctx, env.st, "b", "a")
		markDup(t, ctx, env.st, "b")
		wantTotal(t, ctx, env.st, "1000")
		mustRetract(t, ctx, env.st, "a")
		wantRec(t, ctx, env.st, "a", DispRetracted, "")
		wantReturned(t, ctx, env.st, "b", "")
		wantTotal(t, ctx, env.st, "0")
	})
}

// LC84 (vii): bản được giữ đang REJECTED rời nhóm. a, b, c nối tiếp; MARK_DUPLICATE c (theo b) rồi b (theo a); b bị thu hồi / sửa sang giá trị khác / sửa thành
// HELD vì lý do khác (nguyên thủy áp dụng, như khi người APPLY phiên bản chờ): c quay lại HELD (duplicate_of = a), không muộn hơn lúc áp
// dụng; ở mọi thời điểm tổng hoặc hàng chờ còn chứa giá trị của c; a bị thu hồi thì c vẫn HELD (duplicate_of rỗng); KEEP_BOTH thì c vào tổng.
func TestLC84_vii_BanDuocGiuDangRejectedRoiNhom_BanTrungQuayLai(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		for _, w := range leaveWays() {
			ctx, _ := newCtx("lc84vii")
			mustBatch(t, ctx, env.st, mk("a", 0), mk("b", 30), mk("c", 60))
			markDup(t, ctx, env.st, "c") // c REJECTED theo b
			markDup(t, ctx, env.st, "b") // b REJECTED theo a
			wantRec(t, ctx, env.st, "c", DispRejected, ReasonPossibleDuplicate)
			wantTotal(t, ctx, env.st, "1000")
			w.do(t, ctx, env.st, "b")
			wantReturned(t, ctx, env.st, "c", "a")
			mustRetract(t, ctx, env.st, "a")
			wantReturned(t, ctx, env.st, "c", "")
			if got := totalOf(t, ctx, env.st, "E1"); got != w.stays {
				t.Errorf("%s: tổng = %s, muốn %s", w.name, got, w.stays)
			}
			keepBoth(t, ctx, env.st, "c")
			wantRec(t, ctx, env.st, "c", DispAccepted, "")
		}
	})
}

// R6-01 (đường APPLY): quyết định APPLY một đính chính làm bản được giữ thành SUPERSEDED là một đường rời nhóm mà điều kiện cũ chưa kể: bản bị bác theo
// nó phải quay lại HELD trong CÙNG giao dịch, không rơi khỏi tổng im lặng.
func TestR601_ApplyDinhChinhLamBanDuocGiuSuperseded_BanBiBacQuayLaiHeld(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("r601")
		markedPair(t, ctx, env.st)
		mustDeliver(t, ctx, env.st, "b2", 1, mk("q", 200, heldFor(ReasonCorrectionUnconfirmed), withKind(KindCorrection), withValue("800")))
		wantRec(t, ctx, env.st, "q", DispHeld, ReasonCorrectionUnconfirmed)
		wantTotal(t, ctx, env.st, "1000")
		got := mustDecide(t, ctx, env.st, Command{Key: "q", Decision: DecApply, Actor: "pm-1", Why: "báo nhầm", Target: "a"})
		if got.Disposition != DispAccepted || got.Kind != KindOccurred || got.Corrects != "a" {
			t.Fatalf("đính chính sau APPLY: %+v, muốn ACCEPTED OCCURRED sửa a", got)
		}
		ra := mustRecord(t, ctx, env.st, "a")
		if ra.Disposition != DispSuperseded || ra.SupersededBy != "q" {
			t.Fatalf("bản gốc sau APPLY: %s/%q, muốn SUPERSEDED bởi q", ra.Disposition, ra.SupersededBy)
		}
		wantReturned(t, ctx, env.st, "b", "")
		wantTotal(t, ctx, env.st, "800") // chỉ q; b chờ ở hàng chờ
	})
}

// ---------------------------------------------------------------------------------------------------------------------
// LC15, LC16: APPLY và REJECT đính chính; quyết định cho ra ACCEPTED cũng chạy TR3

// LC15: người quyết APPLY đính chính: bản gốc SUPERSEDED, bản đính chính thành OCCURRED ACCEPTED, tổng đổi đúng, có audit.
func TestLC15_ApplyDinhChinh_BanGocSupersededTongDoiDung(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc15")
		mustDeliver(t, ctx, env.st, "b1", 1, mk("o", 0, withValue("3000")))
		mustDeliver(t, ctx, env.st, "b2", 1, mk("q", 400, heldFor(ReasonCorrectionUnconfirmed), withKind(KindCorrection), withValue("2000")))
		wantTotal(t, ctx, env.st, "3000") // chưa quyết: tổng ngày gốc không đổi
		got := mustDecide(t, ctx, env.st, Command{Key: "q", Decision: DecApply, Actor: "pm-1", Why: "báo nhầm 3k", Target: "o"})
		if got.Disposition != DispAccepted || got.Kind != KindOccurred || got.DecidedBy != "pm-1" || got.Decision != DecApply || got.Version != 2 {
			t.Fatalf("đính chính sau APPLY: %+v", got)
		}
		ro := mustRecord(t, ctx, env.st, "o")
		if ro.Disposition != DispSuperseded || ro.SupersededBy != "q" {
			t.Fatalf("bản gốc: %s/%q, muốn SUPERSEDED bởi q", ro.Disposition, ro.SupersededBy)
		}
		wantTotal(t, ctx, env.st, "2000")
		for _, k := range []string{"o", "q"} {
			if evs := events(t, ctx, env.st, k); len(evs) < 2 {
				t.Errorf("lịch sử của %s %v: phải có dòng audit của quyết định", k, evs)
			}
		}
		led, _ := env.st.Ledger(ctx, "")
		if len(led) != 2 {
			t.Errorf("sổ có %d dòng: quyết định của người không phải lần giao nên không thêm dòng sổ (§3.8)", len(led))
		}
	})
}

// LC16: người quyết REJECT đính chính: tổng không đổi; đính chính REJECTED.
func TestLC16_RejectDinhChinh_TongKhongDoi(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc16")
		mustDeliver(t, ctx, env.st, "b1", 1, mk("o", 0, withValue("3000")))
		mustDeliver(t, ctx, env.st, "b2", 1, mk("q", 400, heldFor(ReasonCorrectionUnconfirmed), withKind(KindCorrection), withValue("2000")))
		got := mustDecide(t, ctx, env.st, Command{Key: "q", Decision: DecReject, Actor: "pm-1", Why: "nhầm máy"})
		if got.Disposition != DispRejected || got.Decision != DecReject {
			t.Fatalf("đính chính sau REJECT: %+v", got)
		}
		wantRec(t, ctx, env.st, "o", DispAccepted, "")
		wantTotal(t, ctx, env.st, "3000")
	})
}

// APPLY cho ra ACCEPTED nên cũng chạy TR3 (§6.2): đính chính 2000 mà đã có bản OCCURRED 2000 cách ≤ W thì HELD possible_duplicate.
func TestApply_ChayTR3_DinhChinhNghiTrungBanDaCo(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("applytr3")
		mustDeliver(t, ctx, env.st, "b1", 1, mk("o", 0, withValue("3000")))
		mustDeliver(t, ctx, env.st, "b2", 1, mk("x", 20, withValue("2000"), withRaw("khác")))
		mustDeliver(t, ctx, env.st, "b3", 1, mk("q", 30, heldFor(ReasonCorrectionUnconfirmed), withKind(KindCorrection), withValue("2000")))
		got := mustDecide(t, ctx, env.st, Command{Key: "q", Decision: DecApply, Actor: "pm-1", Why: "báo nhầm", Target: "o"})
		if got.Disposition != DispHeld || got.Reason != ReasonPossibleDuplicate || got.DuplicateOf != "x" || got.Kind != KindOccurred {
			t.Fatalf("đính chính sau APPLY: %s/%s dup=%q kind=%s, muốn HELD possible_duplicate trỏ x", got.Disposition, got.Reason, got.DuplicateOf, got.Kind)
		}
		wantRec(t, ctx, env.st, "o", DispSuperseded, "")
		wantTotal(t, ctx, env.st, "2000") // chỉ x
	})
}

// ---------------------------------------------------------------------------------------------------------------------
// Bảng sự thật của quyết định (§6.2): đúng quyết định cho đúng mã lý do; sai thì từ chối, bản ghi không đổi, audit vẫn ghi.

func TestQuyetDinh_BangSuThat_SaiQuyetDinhBiTuChoiBanGhiKhongDoi(t *testing.T) {
	allowed := map[Reason][]Decision{
		ReasonAmbiguousEntity:        {DecChoose, DecReject},
		ReasonNumberReadingUncertain: {DecCorrect, DecReject},
		ReasonCorrectionUnconfirmed:  {DecApply, DecReject},
		ReasonPossibleDuplicate:      {DecKeepBoth, DecMarkDuplicate},
		ReasonSenderNotAuthorized:    {DecAuthorize, DecReject},
		ReasonDateConflict:           {DecCorrect, DecReject},
		ReasonAmbiguousKind:          {DecCorrect, DecReject},
		ReasonIDConflict:             {DecCorrect, DecReject},
	}
	all := []Decision{DecKeepBoth, DecMarkDuplicate, DecReject, DecChoose, DecCorrect, DecAuthorize, DecApply}
	forEachStore(t, func(t *testing.T, env storeEnv) {
		for reason, ok := range allowed {
			for _, d := range all {
				permitted := false
				for _, a := range ok {
					permitted = permitted || a == d
				}
				if permitted {
					continue
				}
				ctx, _ := newCtx("bang")
				items := []Item{mk("o", 0, withValue("500")), mk("h", 300, heldFor(reason), withEntity("E1"))}
				if reason == ReasonPossibleDuplicate {
					items = []Item{mk("o", 0), mk("h", 20)}
				}
				mustBatch(t, ctx, env.st, items...)
				before := mustRecord(t, ctx, env.st, "h")
				_, err := env.st.Decide(ctx, Command{Key: "h", Decision: d, Actor: "pm-1", Why: "thử", Entity: "E1", Value: &Quantity{Number: "7", Unit: "L"}, Target: "o", SeenDuplicateOf: str(before.DuplicateOf)})
				if !errors.Is(err, ErrDecisionNotAllowed) || contracts.FaultOf(err) != contracts.FaultInput {
					t.Errorf("%s cho bản HELD %s: muốn ErrDecisionNotAllowed nhãn input, được %v", d, reason, err)
				}
				after := mustRecord(t, ctx, env.st, "h")
				after2, before2 := after, before
				if after2.Disposition != before2.Disposition || after2.Reason != before2.Reason || after2.Entity != before2.Entity || after2.Value != before2.Value || after2.Decision != before2.Decision {
					t.Errorf("%s cho bản HELD %s làm bản ghi đổi: %+v → %+v", d, reason, before, after)
				}
			}
		}
		// bản không tồn tại, thiếu người quyết
		ctx, _ := newCtx("bang2")
		if _, err := env.st.Decide(ctx, Command{Key: "khong-co", Decision: DecReject, Actor: "pm-1"}); !errors.Is(err, ErrNotFound) {
			t.Errorf("quyết định cho bản không có: muốn ErrNotFound, được %v", err)
		}
		mustDeliver(t, ctx, env.st, "b", 1, mk("h", 0, heldFor(ReasonAmbiguousEntity)))
		if _, err := env.st.Decide(ctx, Command{Key: "h", Decision: DecReject}); !errors.Is(err, ErrDecisionNotAllowed) {
			t.Errorf("quyết định không nêu người quyết: muốn bị từ chối (audit cần ai), được %v", err)
		}
		// bản ACCEPTED và bản QUARANTINED: chỉ CORRECT cho QUARANTINED
		mustDeliver(t, ctx, env.st, "b", 2, mk("ok", 600))
		mustDeliver(t, ctx, env.st, "b", 3, mk("q", 700, quarantinedFor(ReasonUnreadableFile), withEntity(""), withDay(""), withValue("")))
		if _, err := env.st.Decide(ctx, Command{Key: "ok", Decision: DecReject, Actor: "pm-1"}); !errors.Is(err, ErrDecisionNotAllowed) {
			t.Errorf("REJECT bản ACCEPTED: muốn bị từ chối, được %v", err)
		}
		if _, err := env.st.Decide(ctx, Command{Key: "q", Decision: DecReject, Actor: "pm-1"}); !errors.Is(err, ErrDecisionNotAllowed) {
			t.Errorf("REJECT bản QUARANTINED: muốn bị từ chối (chỉ CORRECT; DISMISS chưa có), được %v", err)
		}
		got := mustDecide(t, ctx, env.st, Command{Key: "q", Decision: DecCorrect, Actor: "pm-1", Why: "nhập tay", Entity: "E1", OccurredOn: "2026-10-19", Value: &Quantity{Number: "900", Unit: "L"}})
		if got.Disposition != DispAccepted || got.Version != 2 || got.Value.Number != "900" {
			t.Errorf("CORRECT bản QUARANTINED: %+v, muốn ACCEPTED, version 2, 900 L", got)
		}
	})
}

// CORRECT thiếu trường bắt buộc cho ACCEPTED (BB6: không nhận bản đoán) bị từ chối, bản ghi không đổi.
func TestCorrect_ThieuTruongBatBuocChoAccepted_BiTuChoi(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("corr")
		mustDeliver(t, ctx, env.st, "b", 1, mk("q", 0, quarantinedFor(ReasonUnparseable), withEntity(""), withDay(""), withValue("")))
		_, err := env.st.Decide(ctx, Command{Key: "q", Decision: DecCorrect, Actor: "pm-1", Entity: "E1"}) // thiếu ngày và lượng
		if !errors.Is(err, ErrInvalidItem) {
			t.Fatalf("muốn ErrInvalidItem, được %v", err)
		}
		wantRec(t, ctx, env.st, "q", DispQuarantined, ReasonUnparseable)
	})
}

// ---------------------------------------------------------------------------------------------------------------------
// Nguyên thủy Retract và Amend

func TestRetract_BanGhiThanhRetractedRoiKhoiTongVaNhom(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("rtr")
		mustDeliver(t, ctx, env.st, "b", 1, mk("a", 0))
		mustRetract(t, ctx, env.st, "a")
		wantRec(t, ctx, env.st, "a", DispRetracted, "")
		wantTotal(t, ctx, env.st, "0")
		if !hasEvent(events(t, ctx, env.st, "a"), EventRetracted) {
			t.Error("lịch sử thiếu dòng thu hồi (ai, khi nào, vì sao)")
		}
		if err := env.st.Retract(ctx, "a", "ingest", "lần hai"); !errors.Is(err, ErrDecisionNotAllowed) {
			t.Errorf("thu hồi bản đã RETRACTED: muốn bị từ chối, được %v", err)
		}
		if err := env.st.Retract(ctx, "khong-co", "ingest", "x"); !errors.Is(err, ErrNotFound) {
			t.Errorf("thu hồi bản không có: muốn ErrNotFound, được %v", err)
		}
	})
}

func TestAmend_PhienBanMoiVersionTangLichSuGiu(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("amd")
		mustDeliver(t, ctx, env.st, "b", 1, mk("a", 0, withValue("1000")))
		mustAmend(t, ctx, env.st, "a", mk("ignored", 0, withValue("1500"), withRaw("sửa a")))
		ra := wantRec(t, ctx, env.st, "a", DispAccepted, "")
		if ra.Version != 2 || ra.Value.Number != "1500" || ra.Raw != "sửa a" {
			t.Fatalf("sau sửa tin: %+v, muốn version 2, 1500, raw mới", ra)
		}
		wantTotal(t, ctx, env.st, "1500")
		if !hasEvent(events(t, ctx, env.st, "a"), EventAmended) {
			t.Error("lịch sử thiếu dòng sửa tin")
		}
		if err := env.st.Amend(ctx, "a", mk("a", 0, withValue("1500"), withRaw("sửa a")), "ingest", "y nguyên"); !errors.Is(err, ErrDecisionNotAllowed) {
			t.Errorf("sửa với nội dung không đổi: muốn bị từ chối (nguyên thủy không im lặng bỏ qua), được %v", err)
		}
		if err := env.st.Amend(ctx, "a", mk("a", 0, withKind("LA")), "ingest", "x"); !errors.Is(err, ErrInvalidItem) {
			t.Errorf("sửa với nội dung sai: muốn ErrInvalidItem, được %v", err)
		}
		mustRetract(t, ctx, env.st, "a")
		if err := env.st.Amend(ctx, "a", mk("a", 0, withRaw("sau thu hồi")), "ingest", "x"); !errors.Is(err, ErrDecisionNotAllowed) {
			t.Errorf("sửa bản đã RETRACTED: muốn bị từ chối, được %v", err)
		}
	})
}

// Sửa tin làm bản đổi nhóm xoá các cặp đã xét liên quan (chỉ siết): cặp người đã xét là về nội dung cũ, không phủ nội dung mới.
func TestAmend_DoiNhom_XoaCapDaXet(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("amdp")
		mustBatch(t, ctx, env.st, mk("a", 0), mk("c", 30))
		keepBoth(t, ctx, env.st, "c") // cặp (c, a)
		// c và a cùng được sửa sang 2000, vẫn cách 30 phút: cặp mới chưa ai xét thì phải bị giữ, không được miễn theo cặp cũ
		mustAmend(t, ctx, env.st, "a", mk("a", 0, withValue("2000"), withRaw("a 2000")))
		mustAmend(t, ctx, env.st, "c", mk("c", 30, withValue("2000"), withRaw("c 2000")))
		rc := wantHeldPD(t, ctx, env.st, "c", "a")
		if len(rc.ReviewedPairs) != 0 {
			t.Errorf("cặp đã xét còn sót sau khi nội dung đổi: %v", rc.ReviewedPairs)
		}
		wantTotal(t, ctx, env.st, "2000")
	})
}
