package reports

// Bước 3 (HOP-DONG §14.2), phần TR1 và TR2: chống trùng bằng khoá nguồn và vân tay. DUPLICATE chỉ có ở sổ lần giao; không bao giờ tạo
// bản ghi thứ hai của cùng unit_key.

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// LC21: cùng tin giao hai lần (cùng khoá, cùng nội dung): lần hai DUPLICATE trỏ bản đã có; không thêm bản ghi; tổng không đổi.
func TestLC21_TR1_CungKhoaCungNoiDung_Duplicate(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc21")
		first := mustDeliver(t, ctx, env.st, "b1", 1, mk("M-1", 0, withValue("1200")))
		again := mustDeliver(t, ctx, env.st, "b2", 1, mk("M-1", 0, withValue("1200")))
		if first.Outcome != OutcomeAccepted || again.Outcome != OutcomeDuplicate || again.Reason != ReasonDuplicateKey || again.RecordRef != "M-1" {
			t.Fatalf("lần 1 = %+v, lần 2 = %+v; muốn ACCEPTED rồi DUPLICATE duplicate_key trỏ M-1", first, again)
		}
		if again.Detail == "" {
			t.Error("dòng DUPLICATE phải có chi tiết (CL2)")
		}
		if rs := mustRecords(t, ctx, env.st); len(rs) != 1 {
			t.Errorf("kho có %d bản ghi, muốn 1 (DUPLICATE không tạo bản ghi)", len(rs))
		}
		if got := totalOf(t, ctx, env.st, "E1"); got != "1200" {
			t.Errorf("tổng = %s, muốn 1200 (không cộng đôi)", got)
		}
		if led, _ := env.st.Ledger(ctx, ""); len(led) != 2 {
			t.Errorf("sổ có %d dòng, muốn 2 (mỗi lần giao đúng một dòng)", len(led))
		}
	})
}

// LC22 (TR2): cùng nội dung, giờ, người gửi nhưng KHÁC mã tin (xuất chồng lấp): DUPLICATE theo vân tay; không thêm bản ghi.
func TestLC22_TR2_KhacMaTinCungNoiDungGioNguoiGui_Duplicate(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc22")
		same := func(it *Item) { it.SenderID, it.Raw = "S7", "lu 02 1200 lít" }
		mustDeliver(t, ctx, env.st, "b1", 1, mk("M-7", 0, same, withValue("1200")))
		e := mustDeliver(t, ctx, env.st, "b2", 1, mk("M-8", 0, same, withValue("1200")))
		if e.Outcome != OutcomeDuplicate || e.Reason != ReasonDuplicateFingerprint || e.RecordRef != "M-7" || e.Key != "M-8" {
			t.Fatalf("dòng sổ %+v: muốn DUPLICATE duplicate_fingerprint trỏ M-7 cho đơn vị M-8", e)
		}
		if _, ok, _ := env.st.Record(ctx, "M-8"); ok {
			t.Error("DUPLICATE không được tạo bản ghi cho M-8")
		}
		if got := totalOf(t, ctx, env.st, "E1"); got != "1200" {
			t.Errorf("tổng = %s, muốn 1200", got)
		}
	})
}

// LC23 (§5.4): nạp lại nguyên một lô đã nạp: mọi đơn vị DUPLICATE ở lô mới; kho bản ghi, lịch sử, tổng giống hệt; sổ thêm đúng N dòng.
func TestLC23_NapLaiNguyenLo_MoiDonViDuplicate_TrangThaiCuoiGiongHet(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc23")
		items := []Item{
			mk("a1", 0), mk("a2", 300, withEntity("E2"), withValue("250")),
			mk("q1", 301, quarantinedFor(ReasonUnrecognizedLine), withEntity("")),
			mk("h1", 302, heldFor(ReasonAmbiguousEntity)),
			mk("f1", 303, withNoSentAt("3"), withEntity("E3")),
		}
		mustBatch(t, ctx, env.st, items...)
		beforeRecs := mustRecords(t, ctx, env.st)
		beforeHist := map[string][]HistoryEntry{}
		for _, it := range items {
			h, _ := env.st.History(ctx, it.Key)
			beforeHist[it.Key] = h
		}
		beforeTotals, _ := env.st.Totals(ctx)
		beforeLedger, _ := env.st.Ledger(ctx, "")

		rc := mustBatch(t, ctx, env.st, items...)
		if err := rc.Conserved(keysOf(items)); err != nil {
			t.Fatal(err)
		}
		if rc.ByOutcome[OutcomeDuplicate] != len(items) || rc.ByReason[ReasonDuplicateKey] != len(items) {
			t.Fatalf("nạp lại: biên nhận %v / %v, muốn %d DUPLICATE duplicate_key", rc.ByOutcome, rc.ByReason, len(items))
		}
		if afterRecs := mustRecords(t, ctx, env.st); !reflect.DeepEqual(beforeRecs, afterRecs) {
			t.Errorf("kho bản ghi đổi sau khi nạp lại:\n trước %+v\n sau   %+v", beforeRecs, afterRecs)
		}
		for _, it := range items {
			if h, _ := env.st.History(ctx, it.Key); !reflect.DeepEqual(beforeHist[it.Key], h) {
				t.Errorf("lịch sử của %s đổi sau khi nạp lại", it.Key)
			}
		}
		if afterTotals, _ := env.st.Totals(ctx); !reflect.DeepEqual(beforeTotals, afterTotals) {
			t.Errorf("tổng đổi sau khi nạp lại: %v → %v", beforeTotals, afterTotals)
		}
		if afterLedger, _ := env.st.Ledger(ctx, ""); len(afterLedger) != len(beforeLedger)+len(items) {
			t.Errorf("sổ có %d dòng, muốn %d (thêm đúng N dòng cho N đơn vị giao lại)", len(afterLedger), len(beforeLedger)+len(items))
		}
	})
}

// LC01 (phần trùng): lô có đơn vị giao trùng ngay trong lô: số đơn vị vào = số dòng sổ = tổng kết cục, kể cả DUPLICATE (BB1).
func TestLC01_LoCoDonViGiaoTrung_BaoToanKeCaDuplicate(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc01d")
		items := []Item{mk("a", 0), mk("b", 300, withEntity("E2")), mk("a", 0), mk("q", 301, quarantinedFor(ReasonUnreadableFile)), mk("b", 300, withEntity("E2"))}
		rc := mustBatch(t, ctx, env.st, items...)
		if err := rc.Conserved(keysOf(items)); err != nil {
			t.Fatal(err)
		}
		if rc.ByOutcome[OutcomeAccepted] != 2 || rc.ByOutcome[OutcomeDuplicate] != 2 || rc.ByOutcome[OutcomeQuarantined] != 1 {
			t.Errorf("biên nhận %v, muốn A2 D2 Q1", rc.ByOutcome)
		}
		if rs := mustRecords(t, ctx, env.st); len(rs) != 3 {
			t.Errorf("kho có %d bản ghi, muốn 3", len(rs))
		}
	})
}

// LC66, §2.4, §3.8: cùng mã nguồn, khác nội dung, không edited_of: bản đến sau HELD id_conflict mang khoá có hậu tố băm; bản đến trước
// giữ nguyên; không hai bản chung unit_key. Giao lại đúng nội dung xung đột thì gặp chính khoá hậu tố và DUPLICATE.
func TestLC66_TR1_CungKhoaKhacNoiDung_IdConflictKhoaHauTo(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		for _, order := range [][2]string{{"A", "B"}, {"B", "A"}} {
			ctx, _ := newCtx("lc66")
			content := map[string]Item{
				"A": mk("X-1", 0, withRaw("máy 03 3k"), withValue("3000")),
				"B": mk("X-1", 400, withRaw("máy 03 4k"), withValue("4000")),
			}
			first := mustDeliver(t, ctx, env.st, "b1", 1, content[order[0]])
			second := mustDeliver(t, ctx, env.st, "b1", 2, content[order[1]])
			if first.Outcome != OutcomeAccepted || second.Outcome != OutcomeHeld || second.Reason != ReasonIDConflict {
				t.Fatalf("thứ tự %v: %+v rồi %+v; muốn ACCEPTED rồi HELD id_conflict", order, first, second)
			}
			if second.Key != "X-1" || !strings.HasPrefix(second.RecordRef, "X-1~") || len(second.RecordRef) != len("X-1~")+8 {
				t.Fatalf("khoá bản xung đột %q: muốn X-1~<8 ký tự băm>, dòng sổ giữ khoá đã giao %q", second.RecordRef, second.Key)
			}
			keep := mustRecord(t, ctx, env.st, "X-1")
			if keep.Raw != content[order[0]].Raw || keep.Disposition != DispAccepted {
				t.Errorf("bản đến trước phải giữ nguyên: %+v", keep)
			}
			conflict := wantRec(t, ctx, env.st, second.RecordRef, DispHeld, ReasonIDConflict)
			if conflict.Recipient == "" || conflict.Raw != content[order[1]].Raw {
				t.Errorf("bản xung đột thiếu người nhận hoặc đổi raw: %+v", conflict)
			}
			again := mustDeliver(t, ctx, env.st, "b2", 1, content[order[1]])
			if again.Outcome != OutcomeDuplicate || again.Reason != ReasonDuplicateKey || again.RecordRef != second.RecordRef {
				t.Errorf("giao lại nội dung xung đột: %+v, muốn DUPLICATE trỏ %s", again, second.RecordRef)
			}
			if rs := mustRecords(t, ctx, env.st); len(rs) != 2 {
				t.Errorf("kho có %d bản ghi, muốn 2", len(rs))
			}
		}
	})
}

// LC70, HT-8: tin thiếu hoặc trùng record_id: không hai đơn vị KHÁC nội dung chung một unit_key.
func TestLC70_KhoaDonVi_ThieuHoacTrungMaNguon_KhongChungKhoa(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc70")
		f := "tin.jsonl"
		items := []Item{
			mk(MakeUnitKey(f, "", 1), 0, withRaw("máy 03 1k"), withEntity("E1")),
			mk(MakeUnitKey(f, "", 2), 300, withRaw("máy 03 2k"), withEntity("E2")),
			mk(MakeUnitKey(f, "X-1", 3), 600, withRaw("máy 03 3k"), withEntity("E3")),
			mk(MakeUnitKey(f, "X-1", 4), 900, withRaw("máy 03 4k"), withEntity("E4")), // cùng record_id, khác nội dung
		}
		rc := mustBatch(t, ctx, env.st, items...)
		if err := rc.Conserved(keysOf(items)); err != nil {
			t.Fatal(err)
		}
		recs := mustRecords(t, ctx, env.st)
		if len(recs) != 4 {
			t.Fatalf("kho có %d bản ghi, muốn 4 (không đơn vị nào rơi hay gộp)", len(recs))
		}
		seen := map[string]string{}
		for _, r := range recs {
			if prev, dup := seen[r.Key]; dup {
				t.Errorf("khoá %q chung hai đơn vị (%q và %q)", r.Key, prev, r.Raw)
			}
			seen[r.Key] = r.Raw
		}
		if rc.Entries[3].Outcome != OutcomeHeld || rc.Entries[3].Reason != ReasonIDConflict {
			t.Errorf("đơn vị trùng record_id khác nội dung: %+v, muốn HELD id_conflict", rc.Entries[3])
		}
	})
}

// LC79 (§5.1 TR2 vế b, §5.3): đơn vị không có sent_at. (i) cùng dòng chữ, cùng số dòng ở hai tệp ngày khác nhau; (ii) hai dòng cùng chữ,
// khác số dòng, trong một tệp: cả hai vào ở mọi trường hợp (không gộp oan). Đơn vị không có sent_at không vào nhóm TR3.
func TestLC79_TR2_DonViKhongCoSentAt_KhongGopOan(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc79")
		line := func(file, day, seq string) Item {
			return mk(MakeUnitKey(file, seq, 0), 0, withNoSentAt(seq), withDay(day), withRaw(seq+". máy 03 cấp 1000 lít"), func(it *Item) { it.SourceType = "bao_cao_ngay" })
		}
		items := []Item{
			line("bc/2026-10-19.pdf", "2026-10-19", "3"), // (i) cùng dòng chữ, cùng số dòng
			line("bc/2026-10-20.pdf", "2026-10-20", "3"),
			line("bc/2026-10-21.pdf", "2026-10-21", "4"), // (ii) hai dòng cùng chữ trong một tệp
			mk(MakeUnitKey("bc/2026-10-21.pdf", "5", 0), 0, withNoSentAt("5"), withDay("2026-10-21"), withRaw("4. máy 03 cấp 1000 lít"), func(it *Item) { it.SourceType = "bao_cao_ngay" }),
		}
		rc := mustBatch(t, ctx, env.st, items...)
		if rc.ByOutcome[OutcomeAccepted] != 4 || rc.ByOutcome[OutcomeDuplicate] != 0 {
			t.Fatalf("biên nhận %v, muốn 4 ACCEPTED: không gộp oan", rc.ByOutcome)
		}
		ts, _ := env.st.Totals(ctx)
		sum := map[string]string{}
		for _, x := range ts {
			sum[x.OccurredOn] = x.Number
		}
		if sum["2026-10-19"] != "1000" || sum["2026-10-20"] != "1000" || sum["2026-10-21"] != "2000" {
			t.Errorf("tổng theo ngày %v, muốn 1000, 1000, 2000", sum)
		}
	})
}

// LC80 (TR2 vế b): cùng một tệp báo cáo (cùng nội dung, cùng ngày) giao lại ở đường dẫn khác: mọi đơn vị DUPLICATE theo vân tay, không thêm
// bản ghi; tổng và báo cáo không đổi.
func TestLC80_TR2_CungTepDuongDanKhac_MoiDonViDuplicateTheoVanTay(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc80")
		file := func(path string) []Item {
			var out []Item
			for i, v := range []string{"1000", "250", "75"} {
				seq := string(rune('1' + i))
				out = append(out, mk(MakeUnitKey(path, seq, 0), 0, withNoSentAt(seq), withDay("2026-10-19"), withEntity("E"+seq), withValue(v),
					withRaw(seq+". dòng báo cáo "+v), func(it *Item) { it.SourceType = "bao_cao_ngay" }))
			}
			return out
		}
		mustBatch(t, ctx, env.st, file("bc/a/2026-10-19.pdf")...)
		before := mustRecords(t, ctx, env.st)
		beforeTotals, _ := env.st.Totals(ctx)
		rc := mustBatch(t, ctx, env.st, file("bc/b/2026-10-19.pdf")...)
		if rc.ByOutcome[OutcomeDuplicate] != 3 || rc.ByReason[ReasonDuplicateFingerprint] != 3 {
			t.Fatalf("biên nhận %v / %v, muốn 3 DUPLICATE duplicate_fingerprint", rc.ByOutcome, rc.ByReason)
		}
		for _, e := range rc.Entries {
			if !strings.HasPrefix(e.RecordRef, "bc/a/") {
				t.Errorf("dòng %s phải trỏ bản đã có ở đường dẫn cũ, được %q", e.Key, e.RecordRef)
			}
		}
		if after := mustRecords(t, ctx, env.st); !reflect.DeepEqual(before, after) {
			t.Error("kho bản ghi đổi sau khi giao lại tệp ở đường dẫn khác")
		}
		if afterTotals, _ := env.st.Totals(ctx); !reflect.DeepEqual(beforeTotals, afterTotals) {
			t.Errorf("tổng đổi: %v → %v", beforeTotals, afterTotals)
		}
	})
}

// §5.3 không khử trùng theo giá trị: hai lần cấp thật cùng số lít, cách xa nhau, chữ khác nhau: cả hai vào, tổng gồm cả hai.
func TestKhongKhuTrungTheoGiaTri(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("giatri")
		mustBatch(t, ctx, env.st,
			mk("sang", 0, withRaw("sáng cấp 1k"), withSender("S1")),
			mk("chieu", 8*60, withRaw("chiều cấp thêm 1k"), withSender("S2")))
		if got := totalOf(t, ctx, env.st, "E1"); got != "2000" {
			t.Errorf("tổng = %s, muốn 2000 (hai lần cấp thật, không gộp oan)", got)
		}
	})
}

// LC28 (BB8): tin giống hệt ở hai tenant: mỗi tenant một bản ghi; không DUPLICATE xuyên tenant.
func TestLC28_HaiTenantCungTin_KhongDuplicateXuyenTenant(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctxA, _ := newCtx("lc28a")
		ctxB, _ := newCtx("lc28b")
		it := mk("M-1", 0, withSender("S1"), withRaw("máy 03 3k"))
		a := mustDeliver(t, ctxA, env.st, "b1", 1, it)
		b := mustDeliver(t, ctxB, env.st, "b1", 1, it)
		if a.Outcome != OutcomeAccepted || b.Outcome != OutcomeAccepted {
			t.Fatalf("tenant A %+v, tenant B %+v: cả hai phải ACCEPTED", a, b)
		}
		// cùng nội dung khác khoá ở tenant B: vân tay chỉ trùng trong CÙNG tenant
		c := mustDeliver(t, ctxB, env.st, "b2", 1, mk("M-2", 0, withSender("S1"), withRaw("máy 03 3k")))
		if c.Outcome != OutcomeDuplicate || c.RecordRef != "M-1" {
			t.Errorf("trong cùng tenant B phải DUPLICATE theo vân tay: %+v", c)
		}
		if rs := mustRecords(t, ctxA, env.st); len(rs) != 1 {
			t.Errorf("tenant A thấy %d bản ghi, muốn 1", len(rs))
		}
	})
}

// §3.7 trường của hệ: giao lại không ghi đè trace_id của bản ghi có sẵn.
func TestGiaoLai_KhongGhiDeTraceIdCuaBanCoSan(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("trace")
		mustDeliver(t, ctx, env.st, "b1", 1, mk("M-1", 0, func(it *Item) { it.TraceID = "vet-lan-1" }))
		mustDeliver(t, ctx, env.st, "b2", 1, mk("M-1", 0, func(it *Item) { it.TraceID = "vet-lan-2" }))
		if got := mustRecord(t, ctx, env.st, "M-1").TraceID; got != "vet-lan-1" {
			t.Errorf("trace_id = %q, muốn vet-lan-1", got)
		}
		led, _ := env.st.Ledger(ctx, "b2")
		if len(led) != 1 || led[0].TraceID != "vet-lan-2" {
			t.Errorf("dòng sổ của lần giao lại phải mang vết của chính lần giao đó: %+v", led)
		}
	})
}

var _ = time.Second // giữ import khi test chạy trên khung tạm
