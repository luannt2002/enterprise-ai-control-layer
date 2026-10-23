package pipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// V8a. Các hàm ghi S8 (bản nháp như được sinh): đếm theo KÝ TỰ, băm chữ đã làm sạch, đoạn đầu cắt đúng 80 ký tự — khoá từng chữ của dòng.

func sha12(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:12]
}

func TestTextDigestDemKyTuVaLamSachTruoc(t *testing.T) {
	if n, h := textDigest("  ệệệ \n"); n != 3 || h != sha12("ệệệ") {
		t.Errorf("khoảng trắng đầu / cuối không tính, đếm ký tự (không phải byte): %d %s", n, h)
	}
	if n, h := textDigest(""); n != 0 || h != sha12("") {
		t.Errorf("chữ rỗng: %d %s", n, h)
	}
	// bí mật được che TRƯỚC khi đếm và băm: số ký tự và băm khớp chữ nằm trong vết bền
	masked := "password=" + traces.Masked
	if n, h := textDigest("password=hunter2pw"); n != len([]rune(masked)) || h != sha12(masked) {
		t.Errorf("chữ có bí mật: %d %s, muốn %d %s", n, h, len([]rune(masked)), sha12(masked))
	}
}

func TestDraftHeadCat80KyTu(t *testing.T) {
	eighty := strings.Repeat("ệ", 80)
	for _, c := range []struct{ in, want string }{
		{eighty, eighty},
		{eighty + "x", eighty + "…"},
		{"  ngắn  ", "ngắn"},
		{"", ""},
	} {
		if got := draftHead(c.in); got != c.want {
			t.Errorf("draftHead(%.12q…) = %.20q…, muốn %.20q…", c.in, got, c.want)
		}
	}
	if got := draftHead(strings.Repeat("a", 60) + " mật khẩu: hunter2pw " + strings.Repeat("b", 30)); strings.Contains(got, "hunter2pw") {
		t.Errorf("đoạn đầu phải qua bộ làm sạch: %q", got)
	}
}

func TestNoteDraftTungChu(t *testing.T) {
	a := &Ask{}
	var why traces.Why
	text := "Dầu kho xuất của Máy xúc EX-03 ngày 19/10/2026: 3.850 L."
	ctx := context.Background()
	a.noteTemplateDraft(ctx, &why, "", "metric_total", text)
	a.noteModelDraft(ctx, &why, "tri thức: ", "extractive", text)
	a.noteModelDraft(ctx, &why, "", "", text)
	a.noteTemplateDraft(ctx, &why, "số liệu: ", "metric_total + confirm_withheld", text)
	h := sha12(text)
	want := []string{
		"bản nháp dựng bằng mẫu câu metric_total (không gọi model): 56 ký tự · băm " + h + " · đầu “" + text + "”",
		"tri thức: bản nháp do model extractive diễn đạt: 56 ký tự · băm " + h + " · đầu “" + text + "”",
		"bản nháp do model (không khai tên) diễn đạt: 56 ký tự · băm " + h + " · đầu “" + text + "”",
		"số liệu: bản nháp dựng bằng mẫu câu metric_total + confirm_withheld (không gọi model): 56 ký tự · băm " + h + " · đầu “" + text + "”",
	}
	if got := len([]rune(text)); got != 56 {
		t.Fatalf("tiền đề: chữ mẫu dài %d ký tự, muốn 56", got)
	}
	if !slices.Equal(why.Generate.Reasons, want) {
		t.Errorf("S8:\n  được:\n    %s\n  muốn:\n    %s", strings.Join(why.Generate.Reasons, "\n    "), strings.Join(want, "\n    "))
	}
	if len(why.Generate.Rejected) != 0 {
		t.Errorf("S8 không có phương án bị loại: %q", why.Generate.Rejected)
	}
	// công tắc noWhy: không ghi gì
	var off traces.Why
	(&Ask{noWhy: true}).noteTemplateDraft(ctx, &off, "", "metric_total", text)
	if off.Generate.Recorded() {
		t.Errorf("noWhy phải tắt ghi S8: %+v", off.Generate)
	}
}

// Dòng S8 đi qua MỘT chỗ che phạm vi (finalWhy): chữ bản nháp nhắc khoá thực thể ngoài phạm vi thì khoá bị che; câu hỏi mang bí mật thì đoạn đầu (trong
// cặp “…”) bị che nguyên, còn số ký tự và băm giữ nguyên.
func TestDongS8QuaChoChePhamViVaWithholdQuoted(t *testing.T) {
	a := &Ask{}
	var why traces.Why
	a.noteModelDraft(context.Background(), &why, "", "extractive", "Máy ủi DZ-03 cần đối soát theo quy trình")
	got := finalWhy(why, scopeRedactor{tokens: []string{"DZ-03"}}).Generate.Reasons
	if len(got) != 1 || strings.Contains(got[0], "DZ-03") || !strings.Contains(got[0], "Máy ủi "+traces.OutOfScope+" cần đối soát") {
		t.Errorf("khoá ngoài phạm vi phải bị che trong dòng S8: %q", got)
	}
	if !strings.Contains(got[0], "40 ký tự · băm "+sha12("Máy ủi DZ-03 cần đối soát theo quy trình")) {
		t.Errorf("số ký tự và băm tính trên chữ gốc, không đổi khi che: %q", got)
	}
	w := why.WithholdQuoted().Generate.Reasons[0]
	if strings.Contains(w, "Máy ủi") || !strings.Contains(w, "40 ký tự · băm "+sha12("Máy ủi DZ-03 cần đối soát theo quy trình")+" · đầu “đã che”") {
		t.Errorf("WithholdQuoted phải che đoạn đầu và giữ số ký tự, băm: %q", w)
	}
}

// V8a, S10. noteDelivered phân biệt đủ năm hình: trùng, chứa nguyên văn, KHÔNG chứa (một bước sau S9 đã đổi chữ), bản nháp bị S9 loại, và không có sổ.
func TestNoteDeliveredPhanBietNamHinh(t *testing.T) {
	a := &Ask{}
	draft := "1. Khi lượng xuất kho lệch quá dung sai, tổ trưởng kho lập biên bản đối soát."
	dr, dh := textDigest(draft)
	if dr != 77 || len(draft) <= dr {
		t.Fatalf("tiền đề: bản nháp dài %d ký tự (%d byte), muốn 77 ký tự và nhiều byte hơn (chữ có dấu)", dr, len(draft))
	}
	cut := string([]rune(draft)[:40])
	ledger := func(prefix string, rejected bool) *draftLedger {
		return &draftLedger{drafts: []ledgerDraft{{prefix: prefix, text: draft, rejected: rejected}}}
	}
	run := func(answer string, l *draftLedger) []string {
		var why traces.Why
		a.noteDelivered(&why, answer, l)
		return why.Respond.Reasons
	}
	ah := sha12(draft)
	if dh != ah {
		t.Fatalf("băm bản nháp %s ≠ %s", dh, ah)
	}
	wrapped := "Trích văn bản quy trình (SOP): " + draft
	cases := []struct {
		name   string
		answer string
		l      *draftLedger
		want   []string
	}{
		{"trùng (kể cả khoảng trắng đầu / cuối)", "\n" + draft + " ", ledger("", false),
			[]string{"câu giao: 77 ký tự · băm " + ah, "câu giao trùng bản nháp ở S8 (băm " + ah + "): không bước nào sau S8 đổi chữ"}},
		{"chứa nguyên văn", wrapped, ledger("", false),
			[]string{"câu giao: 108 ký tự · băm " + sha12(wrapped), "câu giao chứa nguyên văn bản nháp ở S8 (77 ký tự · băm " + ah + "), phần còn lại do mẫu câu của tenant dựng"}},
		{"KHÔNG chứa: câu giao bị cắt", cut, ledger("", false),
			[]string{"câu giao: 40 ký tự · băm " + sha12(cut), "câu giao KHÔNG chứa nguyên văn bản nháp ở S8 (77 ký tự · băm " + ah + ") dù S9 không loại nó"}},
		{"S9 đã loại bản nháp", "Hệ thống chưa kiểm được.", ledger("", true),
			[]string{"câu giao: 24 ký tự · băm " + sha12("Hệ thống chưa kiểm được."), "bản nháp ở S8 (77 ký tự · băm " + ah + ") đã bị S9 loại nên không vào câu giao; câu giao là mẫu thay thế của tenant"}},
		{"tiền tố HYBRID", wrapped, ledger("tri thức: ", false),
			[]string{"câu giao: 108 ký tự · băm " + sha12(wrapped), "tri thức: câu giao chứa nguyên văn bản nháp ở S8 (77 ký tự · băm " + ah + "), phần còn lại do mẫu câu của tenant dựng"}},
		{"không có sổ", draft, nil, []string{"câu giao: 77 ký tự · băm " + ah}},
	}
	for _, c := range cases {
		if got := run(c.answer, c.l); !slices.Equal(got, c.want) {
			t.Errorf("%s:\n  được:\n    %s\n  muốn:\n    %s", c.name, strings.Join(got, "\n    "), strings.Join(c.want, "\n    "))
		}
	}
	var off traces.Why
	(&Ask{noWhy: true}).noteDelivered(&off, draft, ledger("", false))
	if off.Respond.Recorded() {
		t.Errorf("noWhy phải tắt ghi S10: %+v", off.Respond)
	}
}

// Sổ bản nháp: ghi theo thứ tự dựng; loại đúng bản nháp gần nhất cùng tiền tố và chữ, mỗi bản nháp một lần; nhận nil.
func TestDraftLedgerLoaiDungBan(t *testing.T) {
	ctx, l := withLedger(context.Background())
	if ledgerOf(ctx) != l || ledgerOf(context.Background()) != nil {
		t.Fatalf("ledgerOf phải trả đúng sổ của ctx và nil khi không có")
	}
	a := &Ask{}
	var why traces.Why
	a.noteTemplateDraft(ctx, &why, "số liệu: ", "metric_total", "A")
	a.noteModelDraft(ctx, &why, "tri thức: ", "m", "B")
	a.noteModelDraft(ctx, &why, "tri thức: ", "m", "B")
	a.noteRejected(ctx, "tri thức: ", "B")
	a.noteRejected(ctx, "số liệu: ", "khác")
	got := []bool{l.drafts[0].rejected, l.drafts[1].rejected, l.drafts[2].rejected}
	if want := []bool{false, false, true}; !slices.Equal(got, want) {
		t.Errorf("loại bản nháp gần nhất cùng tiền tố và chữ: %v, muốn %v", got, want)
	}
	var none *draftLedger
	none.add("", "x")
	none.reject("", "x") // không panic
	(&Ask{noWhy: true}).noteRejected(ctx, "tri thức: ", "B")
	if !l.drafts[1].rejected == false && l.drafts[1].rejected {
		t.Errorf("noWhy không được loại bản nháp trong sổ")
	}
}

// V8a, nhánh thoát sớm. quantityTrigger là bản đọc lại của asksFactQuantity chỉ để giải thích: trên mọi câu đã công bố (bộ 117, H1, H3, H4) cộng vài câu dựng
// thêm, "có cụm hỏi lượng VÀ có chiều đo nghiệp vụ" của nó phải đúng bằng kết luận của asksFactQuantity — hai hàm lệch nhau thì dòng S8 nói sai lý do thoát.
func TestQuantityTriggerKhopAsksFactQuantity(t *testing.T) {
	rt, err := tenant.Load("../../tenants/construction/pack")
	if err != nil {
		t.Fatalf("nạp pack: %v", err)
	}
	a := &Ask{RT: rt}
	now := time.Date(2026, 10, 19, 10, 0, 0, 0, time.FixedZone("ICT", 7*3600))
	var questions []string
	for _, f := range []string{"../../tenants/construction/win/cases-independent.yaml", "../../docs/g7/holdout/h1.yaml", "../../docs/g7/holdout-h3/h3.yaml", "../../docs/g7/holdout-h4/h4.yaml"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("đọc %s: %v", f, err)
		}
		var doc struct {
			Cases []struct {
				Question string `yaml:"question"`
			} `yaml:"cases"`
		}
		if err := yaml.Unmarshal(b, &doc); err != nil {
			t.Fatalf("đọc %s: %v", f, err)
		}
		for _, c := range doc.Cases {
			questions = append(questions, c.Question)
		}
	}
	questions = append(questions, "Theo quy trình thì một máy xúc tiêu thụ bao nhiêu lít dầu mỗi giờ?", "Định mức cấp phát cho máy xúc theo quy trình là bao nhiêu?",
		"Quy trình xử lý chênh lệch nhiên liệu là gì?")
	fires, markerOnly := 0, 0
	for _, q := range questions {
		in := rt.Normalizer.Normalize(q, normalization.Options{Now: now})
		marker, dims := a.quantityTrigger(in, now)
		got := marker != "" && len(dims) > 0
		if got != a.asksFactQuantity(in, now) {
			t.Errorf("%q: quantityTrigger (cụm %q, chiều %v) và asksFactQuantity (%v) kết luận khác nhau", q, marker, dims, a.asksFactQuantity(in, now))
		}
		if got {
			fires++
		} else if marker != "" {
			markerOnly++
		}
	}
	if fires < 5 || markerOnly < 1 {
		t.Errorf("bộ câu không còn khoá được điều kiện: nhánh kích hoạt %d lần (muốn ≥ 5), có cụm hỏi lượng mà không có chiều đo %d lần (muốn ≥ 1)", fires, markerOnly)
	}
	in := rt.Normalizer.Normalize("Theo quy trình thì một máy xúc tiêu thụ bao nhiêu lít dầu mỗi giờ?", normalization.Options{Now: now})
	if marker, dims := a.quantityTrigger(in, now); marker != "bao nhiêu" || !slices.Equal(dims, []string{"volume"}) {
		t.Errorf("câu mẫu: cụm %q, chiều %v; muốn \"bao nhiêu\" và [volume]", marker, dims)
	}
}

// noteExit: không ghi gì khi noWhy, và không ghi khi đây là S9 loại bản nháp (rejected) — chỉ nhánh thoát sớm mới có dòng "kết cục nhánh đã chọn".
func TestNoteExitChiGhiNhanhThoatSom(t *testing.T) {
	rt, err := tenant.Load("../../tenants/construction/pack")
	if err != nil {
		t.Fatalf("nạp pack: %v", err)
	}
	resp := &contracts.AskResponse{Outcome: contracts.OutcomeInsufficientEvidence, Answer: "Chưa có tài liệu bạn được xem trả lời được câu hỏi này."}
	run := func(a *Ask, hybrid, rejected bool) traces.Notes {
		var why traces.Why
		a.noteExit(context.Background(), &why, hybrid, "no_knowledge", contracts.OutcomeInsufficientEvidence, resp, rejected)
		return why.Generate
	}
	a := &Ask{RT: rt}
	if n := run(a, false, false); len(n.Reasons) != 2 || n.Reasons[0] != "kết cục nhánh đã chọn: INSUFFICIENT_EVIDENCE (mẫu câu no_knowledge)" {
		t.Errorf("thoát sớm: %q", n.Reasons)
	}
	if n := run(a, true, false); len(n.Reasons) != 2 || !strings.HasPrefix(n.Reasons[0], "tri thức: phần quy trình bỏ, kết cục giữ nguyên INSUFFICIENT_EVIDENCE") {
		t.Errorf("HYBRID: %q", n.Reasons)
	}
	if n := run(a, false, true); n.Recorded() {
		t.Errorf("S9 loại bản nháp không phải thoát sớm: %q", n.Reasons)
	}
	if n := run(&Ask{RT: rt, noWhy: true}, false, false); n.Recorded() {
		t.Errorf("noWhy phải tắt ghi: %q", n.Reasons)
	}
}
