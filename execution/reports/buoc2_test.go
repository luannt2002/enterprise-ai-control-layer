package reports

// Bước 2 (HOP-DONG §14.2): kho bản ghi, sổ lần giao, biên nhận lô, bốn kết cục, mã lý do. Các test chạy trên mọi kho có sẵn (forEachStore).
// Tên test mang mã lớp ca (LC) của §10 để bảng "lớp ca → test" ở lc_map_test.go tra được.

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Bảng §3.4 chép nguyên văn thành một bảng độc lập với reasonOutcome của mã: hai bảng phải trùng.
var specReasons = map[Reason]Outcome{
	"missing_value": OutcomeQuarantined, "unknown_entity": OutcomeQuarantined, "ambiguous_entity": OutcomeHeld,
	"out_of_range": OutcomeQuarantined, "unit_dimension": OutcomeQuarantined, "unparseable": OutcomeQuarantined,
	"multiple_items": OutcomeQuarantined, "number_reading_uncertain": OutcomeHeld, "unsupported_media": OutcomeQuarantined,
	"unrecognized_line": OutcomeQuarantined, "unreadable_file": OutcomeQuarantined, "unknown_sender": OutcomeQuarantined,
	"sender_not_authorized": OutcomeHeld, "date_conflict": OutcomeHeld, "correction_unconfirmed": OutcomeHeld,
	"possible_duplicate": OutcomeHeld, "ambiguous_kind": OutcomeHeld, "id_conflict": OutcomeHeld,
	"duplicate_key": OutcomeDuplicate, "duplicate_fingerprint": OutcomeDuplicate,
}

func TestMaLyDo_DanhSachDongKhopBang34(t *testing.T) {
	got := map[Reason]Outcome{}
	for _, r := range AllReasons() {
		o, ok := r.Outcome()
		if !ok {
			t.Fatalf("mã %q trong AllReasons nhưng Outcome() không biết", r)
		}
		got[r] = o
	}
	if !reflect.DeepEqual(got, specReasons) {
		t.Fatalf("danh sách mã lý do lệch bảng §3.4:\n được %v\n muốn %v", got, specReasons)
	}
	if _, ok := Reason("tu_bia_ra").Outcome(); ok {
		t.Error("mã ngoài danh sách đóng phải bị từ chối")
	}
	if len(ReasonsOf(OutcomeDuplicate)) != 2 || len(ReasonsOf(OutcomeQuarantined)) != 10 || len(ReasonsOf(OutcomeHeld)) != 8 {
		t.Errorf("số mã theo kết cục: D=%d Q=%d H=%d, muốn 2, 10, 8", len(ReasonsOf(OutcomeDuplicate)), len(ReasonsOf(OutcomeQuarantined)), len(ReasonsOf(OutcomeHeld)))
	}
}

// LC07, BB8: đầu vào không có đường nào để chọn tenant.
func TestLC07_DauVaoKhongCoTruongTenant(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeOf(Item{}), reflect.TypeOf(Command{})} {
		for i := 0; i < typ.NumField(); i++ {
			if strings.Contains(strings.ToLower(typ.Field(i).Name), "tenant") {
				t.Errorf("%s có trường %s: tenant chỉ đến từ ngữ cảnh", typ.Name(), typ.Field(i).Name)
			}
		}
	}
}

func TestBB8_KhongTenantTrongNguCanh_TuChoiKhongChamDuLieu(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		bare := context.Background()
		if _, err := env.st.Deliver(bare, "b1", 1, mk("x", 0)); contracts.FaultOf(err) != contracts.FaultPolicy {
			t.Errorf("Deliver không tenant: muốn lỗi %s, được %v", contracts.FaultPolicy, err)
		}
		if _, err := env.st.DeliverBatch(bare, []Item{mk("x", 0)}); contracts.FaultOf(err) != contracts.FaultPolicy {
			t.Errorf("DeliverBatch không tenant: muốn lỗi %s, được %v", contracts.FaultPolicy, err)
		}
		if _, err := env.st.Records(bare); contracts.FaultOf(err) != contracts.FaultPolicy {
			t.Errorf("Records không tenant: muốn lỗi %s, được %v", contracts.FaultPolicy, err)
		}
		if _, err := env.st.Totals(bare); contracts.FaultOf(err) != contracts.FaultPolicy {
			t.Errorf("Totals không tenant: muốn lỗi %s, được %v", contracts.FaultPolicy, err)
		}
	})
}

// LC01 (phần kho): lô hỗn hợp, mỗi đơn vị vào khớp đúng một dòng sổ (BB1).
func TestLC01_LoHonHop_SoDonViVaoBangSoDongSoBangTongKetCuc(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc01")
		items := []Item{
			mk("a-1", 0),
			mk("a-2", 5, withEntity("E2")),
			mk("q-1", 10, quarantinedFor(ReasonUnrecognizedLine)),
			mk("q-2", 11, quarantinedFor(ReasonUnreadableFile)),
			mk("h-1", 12, heldFor(ReasonAmbiguousEntity)),
			mk("a-3", 200, withEntity("E3")),
		}
		rc := mustBatch(t, ctx, env.st, items...)
		if err := rc.Conserved(keysOf(items)); err != nil {
			t.Fatal(err)
		}
		if rc.ByOutcome[OutcomeAccepted] != 3 || rc.ByOutcome[OutcomeHeld] != 1 || rc.ByOutcome[OutcomeQuarantined] != 2 || rc.ByOutcome[OutcomeDuplicate] != 0 {
			t.Errorf("biên nhận theo kết cục = %v, muốn A3 H1 Q2 D0", rc.ByOutcome)
		}
		if rc.ByReason[ReasonUnrecognizedLine] != 1 || rc.ByReason[ReasonAmbiguousEntity] != 1 {
			t.Errorf("biên nhận theo mã lý do = %v", rc.ByReason)
		}
		// sổ đọc lại từ kho khớp biên nhận: nguồn sự thật của BB1 là kho, không phải giá trị trả về
		led, err := env.st.Ledger(ctx, rc.BatchID)
		if err != nil || len(led) != len(items) {
			t.Fatalf("Ledger: %d dòng (lỗi %v), muốn %d", len(led), err, len(items))
		}
		recs := mustRecords(t, ctx, env.st)
		if len(recs) != len(items) {
			t.Errorf("kho có %d bản ghi, muốn %d (mỗi đơn vị không-DUPLICATE đúng một bản ghi)", len(recs), len(items))
		}
		for _, e := range led {
			if e.RecordRef == "" {
				t.Errorf("dòng sổ %s không trỏ bản ghi", e.Key)
			}
			if _, ok, _ := env.st.Record(ctx, e.RecordRef); !ok {
				t.Errorf("dòng sổ %s trỏ bản ghi %s không có thật", e.Key, e.RecordRef)
			}
		}
	})
}

// LC02, LC03, LC06 (phần kho): mọi mã QUARANTINED hợp lệ được lưu kèm lý do, chi tiết và người nhận; không vào tổng.
func TestLC02_LC03_LC06_CachLyLuuBenCoLyDoVaNguoiNhan(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc02")
		var items []Item
		for i, r := range ReasonsOf(OutcomeQuarantined) {
			items = append(items, mk("q-"+string(r), i, quarantinedFor(r), func(it *Item) { it.Entity, it.OccurredOn, it.Value = "", "", Quantity{} }))
		}
		for i, r := range ReasonsOf(OutcomeHeld) {
			if r == ReasonPossibleDuplicate {
				continue // do TR3 sở hữu, luật đọc không đề xuất được
			}
			items = append(items, mk("h-"+string(r), 100+i*100, heldFor(r)))
		}
		rc := mustBatch(t, ctx, env.st, items...)
		if err := rc.Conserved(keysOf(items)); err != nil {
			t.Fatal(err)
		}
		for _, it := range items {
			rec := mustRecord(t, ctx, env.st, it.Key)
			if Outcome(rec.Disposition) != it.Verdict.Outcome || rec.Reason != it.Verdict.Reason {
				t.Errorf("%s: được %s/%s, muốn %s/%s", it.Key, rec.Disposition, rec.Reason, it.Verdict.Outcome, it.Verdict.Reason)
			}
			if rec.Detail == "" || rec.Recipient == "" {
				t.Errorf("%s: thiếu detail %q hoặc người nhận %q (BB2)", it.Key, rec.Detail, rec.Recipient)
			}
		}
		if got := totalOf(t, ctx, env.st, "E1"); got != "0" {
			t.Errorf("tổng = %s, muốn 0: HELD và QUARANTINED không bao giờ vào tổng (BB3)", got)
		}
	})
}

// §6.3: người nhận do bên gọi tính theo thực thể; không xác định được thì director.
func TestNguoiNhan_TheoThucTheVaMacDinhDirector(t *testing.T) {
	opt := testOptions()
	opt.Recipient = func(entity string) string {
		if entity == "E1" {
			return "pm-1"
		}
		return ""
	}
	st := NewMemory(opt)
	ctx, _ := newCtx("rcpt")
	mustBatch(t, ctx, st, mk("h1", 0, heldFor(ReasonAmbiguousEntity)), mk("h2", 1, heldFor(ReasonAmbiguousEntity), withEntity("E9")),
		mk("q1", 2, quarantinedFor(ReasonUnknownEntity), withEntity("")))
	for key, want := range map[string]string{"h1": "pm-1", "h2": DefaultRecipient, "q1": DefaultRecipient} {
		if got := mustRecord(t, ctx, st, key).Recipient; got != want {
			t.Errorf("%s: người nhận %q, muốn %q", key, got, want)
		}
	}
}

// LC04, LC05: tham chiếu đính kèm được giữ, kể cả khi chữ rỗng; không bỏ im lặng.
func TestLC04_LC05_GiuThamChieuDinhKem(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc04")
		att := []Attachment{{Kind: "image", Ref: "img/001.jpg"}, {Kind: "file", Ref: "doc/a.pdf"}}
		mustBatch(t, ctx, env.st,
			mk("img-only", 0, quarantinedFor(ReasonUnsupportedMedia), withRaw(""), func(it *Item) { it.Attachments = att; it.Entity, it.OccurredOn, it.Value = "", "", Quantity{} }),
			mk("text+img", 1, func(it *Item) { it.Attachments = att[:1] }))
		if got := mustRecord(t, ctx, env.st, "img-only").Attachments; !reflect.DeepEqual(got, att) {
			t.Errorf("tin chỉ có ảnh: đính kèm %v, muốn %v", got, att)
		}
		rec := wantRec(t, ctx, env.st, "text+img", DispAccepted, "")
		if !reflect.DeepEqual(rec.Attachments, att[:1]) || rec.Raw != "tin text+img" {
			t.Errorf("tin có cả chữ và ảnh: đính kèm %v, raw %q", rec.Attachments, rec.Raw)
		}
	})
}

// LC08: tin rất dài, ký tự điều khiển, ký tự vô hình, UTF-8 hỏng, NUL: vẫn đúng một kết cục, không panic, raw giữ từng byte (BB4).
func TestLC08_KyTuLaVaTinRatDai_RawGiuTungByte(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("lc08")
		raws := map[string]string{
			"dai":     strings.Repeat("máy 03 cấp 1k\n", 80_000),
			"dieukhn": "a\x01b\x07c\x1b[31md\tx\r\ny",
			"vohinh":  "máy" + string(rune(0x200b)) + "03" + string(rune(0x200d)) + " 3k" + string(rune(0xfeff)),
			"hong":    "máy 03 \xff\xfe\xc3\x28 3k",
			"nul":     "a\x00b",
			"rong":    "",
		}
		var items []Item
		i := 0
		for k, raw := range raws {
			items = append(items, mk("lc08-"+k, i, withRaw(raw)))
			i += 200 // cách xa hơn W để không vướng TR3
		}
		rc := mustBatch(t, ctx, env.st, items...)
		if err := rc.Conserved(keysOf(items)); err != nil {
			t.Fatal(err)
		}
		for k, raw := range raws {
			rec := mustRecord(t, ctx, env.st, "lc08-"+k)
			if rec.Raw != raw || !bytes.Equal([]byte(rec.Raw), []byte(raw)) {
				t.Errorf("raw %q bị đổi sau khi lưu (BB4): %d byte còn %d byte, utf8 hợp lệ trước=%v", k, len(raw), len(rec.Raw), utf8.ValidString(raw))
			}
		}
	})
}

// §2.4: khoá đơn vị chứa ký tự không lưu được kiểu text vẫn được xử lý, và khoá được làm sạch tất định.
func TestKhoaDonVi_KyTuKhongLuuDuocDuocLamSach(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("khoa")
		e := mustDeliver(t, ctx, env.st, "b1", 1, mk("f.jsonl#a\x00b\xff", 0))
		if e.Outcome != OutcomeAccepted || !utf8.ValidString(e.RecordRef) || strings.ContainsRune(e.RecordRef, 0) {
			t.Fatalf("khoá bẩn: dòng sổ %+v", e)
		}
		if _, ok, err := env.st.Record(ctx, e.RecordRef); err != nil || !ok {
			t.Fatalf("không đọc lại được bản ghi %q: ok=%v err=%v", e.RecordRef, ok, err)
		}
	})
}

// Đầu vào sai là lỗi của bên gọi, không bao giờ rơi im lặng: bên gọi thấy lỗi, kho không đổi, sổ không có dòng.
func TestDeliver_DauVaoSaiBiTuChoiKhongDoiKho(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("bad")
		bad := map[string]Item{
			"thiếu khoá":              mk("", 0),
			"ACCEPTED thiếu thực thể": mk("b1", 0, withEntity("")),
			"ACCEPTED thiếu lượng":    mk("b2", 0, withValue("")),
			"ACCEPTED mang lý do":     mk("b3", 0, func(it *Item) { it.Verdict.Reason = ReasonUnparseable }),
			"HELD mã của QUARANTINED": mk("b4", 0, func(it *Item) { it.Verdict = Verdict{Outcome: OutcomeHeld, Reason: ReasonUnparseable, Detail: "x"} }),
			"mã ngoài danh sách":      mk("b5", 0, func(it *Item) { it.Verdict = Verdict{Outcome: OutcomeHeld, Reason: "tu_bia_ra", Detail: "x"} }),
			"đề xuất DUPLICATE":       mk("b6", 0, func(it *Item) { it.Verdict = Verdict{Outcome: OutcomeDuplicate, Reason: ReasonDuplicateKey} }),
			"đề xuất possible_dup":    mk("b7", 0, heldFor(ReasonPossibleDuplicate)),
			"ngày sai dạng":           mk("b8", 0, withDay("19/10/2026")),
			"lượng không phải số":     mk("b9", 0, withValue("ba nghìn")),
			"loại tin lạ":             mk("b10", 0, withKind("KHAC")),
		}
		for name, it := range bad {
			_, err := env.st.Deliver(ctx, "b1", 1, it)
			if !errors.Is(err, ErrInvalidItem) || contracts.FaultOf(err) != contracts.FaultInput {
				t.Errorf("%s: muốn ErrInvalidItem nhãn %s, được %v", name, contracts.FaultInput, err)
			}
		}
		if rs := mustRecords(t, ctx, env.st); len(rs) != 0 {
			t.Errorf("đầu vào sai làm kho có %d bản ghi", len(rs))
		}
		if led, _ := env.st.Ledger(ctx, ""); len(led) != 0 {
			t.Errorf("đầu vào sai làm sổ có %d dòng", len(led))
		}
	})
}

// §7.5: lô dừng giữa chừng chỉ có dòng của đơn vị đã xử lý xong; lô chạy lại là lô mới.
func TestLo_DungGiuaChung_ChiCoDongCuaDonViDaXong(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("dung")
		items := []Item{mk("d1", 0), mk("d2", 200, withEntity("E2")), mk("d3", 400, withValue("")), mk("d4", 600, withEntity("E4"))}
		rc, err := env.st.DeliverBatch(ctx, items)
		if !errors.Is(err, ErrInvalidItem) {
			t.Fatalf("muốn lỗi ErrInvalidItem ở đơn vị thứ 3, được %v", err)
		}
		if len(rc.Entries) != 2 || rc.Entries[1].Key != "d2" {
			t.Fatalf("biên nhận của lô dừng: %+v, muốn đúng hai dòng d1, d2", rc.Entries)
		}
		led, _ := env.st.Ledger(ctx, rc.BatchID)
		if len(led) != 2 {
			t.Errorf("sổ có %d dòng cho lô dừng, muốn 2", len(led))
		}
		if _, ok, _ := env.st.Record(ctx, "d4"); ok {
			t.Error("đơn vị sau chỗ dừng không được xử lý")
		}
		// chạy lại là một lô MỚI (mã lô cấp mới, không suy từ nội dung: §3.8)
		rc2, err := env.st.DeliverBatch(ctx, []Item{mk("d5", 800, withEntity("E5"))}) // giao lại cùng khoá là việc của TR1 (bước 3 tiếp theo)
		if err != nil || rc2.BatchID == rc.BatchID {
			t.Errorf("lô chạy lại phải có mã lô mới: %q so với %q (lỗi %v)", rc2.BatchID, rc.BatchID, err)
		}
	})
}

// Receipt.Conserved là chính cái đo BB1: phải đỏ khi thiếu dòng, sai thứ tự, hoặc dòng không trỏ bản ghi nào.
func TestBB1_Conserved_BatDuocThieuDongVaSaiThuTu(t *testing.T) {
	good := []DeliveryEntry{
		{BatchID: "b", Ordinal: 1, Key: "a", Outcome: OutcomeAccepted, RecordRef: "a"},
		{BatchID: "b", Ordinal: 2, Key: "b", Outcome: OutcomeDuplicate, Reason: ReasonDuplicateKey, RecordRef: "a"},
	}
	if err := BuildReceipt("b", good).Conserved([]string{"a", "b"}); err != nil {
		t.Fatalf("lô đúng bị báo lỗi: %v", err)
	}
	cases := map[string]struct {
		entries  []DeliveryEntry
		manifest []string
	}{
		"thiếu một dòng sổ (trùng rơi im lặng)": {good[:1], []string{"a", "b"}},
		"thừa một dòng sổ":                      {good, []string{"a"}},
		"sai thứ tự":                            {good, []string{"b", "a"}},
		"dòng không trỏ bản ghi":                {[]DeliveryEntry{good[0], {BatchID: "b", Ordinal: 2, Key: "b", Outcome: OutcomeDuplicate}}, []string{"a", "b"}},
		"kết cục ngoài bốn giá trị":             {[]DeliveryEntry{good[0], {BatchID: "b", Ordinal: 2, Key: "b", Outcome: "LA", RecordRef: "a"}}, []string{"a", "b"}},
	}
	for name, c := range cases {
		if err := BuildReceipt("b", c.entries).Conserved(c.manifest); err == nil {
			t.Errorf("%s: Conserved phải báo lỗi", name)
		}
	}
}

// BB3, LC57: chỉ ACCEPTED loại OCCURRED vào tổng; HELD, QUARANTINED, PLAN, CORRECTION thì không.
func TestBB3_ChiAcceptedOccurredVaoTong(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctx, _ := newCtx("bb3")
		mustBatch(t, ctx, env.st,
			mk("o1", 0, withValue("1200")),
			mk("o2", 200, withValue("800")),
			mk("plan", 400, withKind(KindPlan), withValue("4000")),
			mk("corr", 600, withKind(KindCorrection), withValue("3000")),
			mk("held", 800, heldFor(ReasonAmbiguousEntity), withValue("9999")),
			mk("quar", 1000, quarantinedFor(ReasonOutOfRange), withValue("77777")))
		if got := totalOf(t, ctx, env.st, "E1"); got != "2000" {
			t.Errorf("tổng = %s, muốn 2000 (1200 + 800)", got)
		}
	})
}

// BB8, LC28 (phần kho): cùng unit_key ở hai tenant là hai bản ghi riêng; không rò sang nhau.
func TestBB8_HaiTenantCungKhoaKhongRoSangNhau(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		ctxA, _ := newCtx("ta")
		ctxB, _ := newCtx("tb")
		mustBatch(t, ctxA, env.st, mk("chung", 0, withValue("1111")))
		mustBatch(t, ctxB, env.st, mk("chung", 0, withValue("2222")))
		if got := mustRecord(t, ctxA, env.st, "chung").Value.Number; got != "1111" {
			t.Errorf("tenant A thấy %s, muốn 1111", got)
		}
		if got := mustRecord(t, ctxB, env.st, "chung").Value.Number; got != "2222" {
			t.Errorf("tenant B thấy %s, muốn 2222", got)
		}
		if rs := mustRecords(t, ctxA, env.st); len(rs) != 1 {
			t.Errorf("tenant A thấy %d bản ghi, muốn 1", len(rs))
		}
		if led, _ := env.st.Ledger(ctxB, ""); len(led) != 1 {
			t.Errorf("tenant B thấy %d dòng sổ, muốn 1", len(led))
		}
	})
}
