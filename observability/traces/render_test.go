package traces

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// renderTurn dựng Turn của một lượt số liệu; route / outcome / verdict do ca chọn.
func renderTurn(route contracts.RouteKind, outcome contracts.Outcome, verdict string, pass bool, action contracts.Action) Turn {
	d := Decision{
		TraceID: "t-render", Tenant: "tenant-x", Principal: "u-1", Kind: "user", Roles: []string{"director"},
		Now:         time.Date(2026, 10, 20, 10, 0, 0, 0, time.FixedZone("ICT", 7*3600)),
		PackVersion: "pack-1", CodeVersion: "rev-1", DataVersion: "data-1", Question: "EX-03 hôm qua cấp bao nhiêu lít?",
		Normalized: []string{"date:hôm qua=2026-10-19"}, Entities: []string{"EX-03"}, Intent: "fuel_issued", Slots: []string{"entity=EX-03"},
		Route: string(route), Outcome: string(outcome), Evidence: []string{"E1"}, Verdict: verdict, Answer: "3.850 L",
	}
	plan := contracts.RoutePlan{Route: route, Template: "metric_total", Reason: "ý định số liệu có đủ khe",
		Intent: contracts.Intent{Name: "fuel_issued", Confidence: 0.9, Rule: "từ khoá: cấp + dầu"}}
	resp := contracts.AskResponse{Answer: d.Answer, Outcome: outcome, Route: route, TraceID: "t-render",
		Verdict:   contracts.Verdict{Pass: pass, Action: action},
		Citations: []contracts.Citation{{EvidenceID: "E1", Source: contracts.SourceRef{SourceType: "warehouse_slip", Name: "Phiếu R-1"}}}}
	spans := []Span{{Name: "normalize", Ms: 1}, {Name: "resolve", Ms: 2}, {Name: "execute", Ms: 3}, {Name: "validate", Ms: 1}}
	return Turn{Decision: d, Spans: spans, TotalMs: 9, UnaccountedMs: 2, Plan: plan, Response: resp}
}

// row trả dòng của bảng bước ứng với mã bước (dòng bắt đầu bằng "<số> <mã bước> "), để kiểm đúng DÒNG chứ không phải đâu đó trong
// chi tiết.
func row(t *testing.T, out, stage string) string {
	t.Helper()
	re := regexp.MustCompile(`(?m)^\d+\s+` + regexp.QuoteMeta(stage) + `\s.*$`)
	m := re.FindString(out)
	if m == "" {
		t.Fatalf("không có dòng bảng cho bước %s trong:\n%s", stage, out)
	}
	return m
}

// CHO PHÉP: lượt thẳng — đủ 11 bước S0…S10 theo thứ tự, không bước nào bị đánh dấu LỆCH / LỖI, dòng chốt nói không có bước lệch.
func TestFormatTableLuotThang(t *testing.T) {
	rec := NewRecord(renderTurn(contracts.RouteSQLQuery, contracts.OutcomeResolved, "true|accept", true, contracts.ActionAccept))
	out := FormatTable(rec)
	last := -1
	for i := 0; i <= 10; i++ {
		stage := "S" + string(rune('0'+i))
		if i == 10 {
			stage = "S10"
		}
		idx := strings.Index(out, "\n"+row(t, out, stage))
		if idx <= last {
			t.Errorf("bước %s không đứng sau bước trước nó trong bảng", stage)
		}
		last = idx
		if strings.Contains(row(t, out, stage), MarkDeviant) || strings.Contains(row(t, out, stage), MarkError) {
			t.Errorf("lượt thẳng mà bước %s bị đánh dấu lệch / lỗi: %s", stage, row(t, out, stage))
		}
	}
	for _, want := range []string{"VẾT t-render", "tenant tenant-x", "câu hỏi: EX-03 hôm qua", "pack pack-1 · mã rev-1 · dữ liệu data-1", "Không có bước lệch."} {
		if !strings.Contains(out, want) {
			t.Errorf("thiếu %q trong bảng vết:\n%s", want, out)
		}
	}
	if strings.Contains(out, "BƯỚC LỆCH ĐẦU TIÊN") {
		t.Errorf("lượt thẳng mà có dòng BƯỚC LỆCH ĐẦU TIÊN:\n%s", out)
	}
	if FormatTable(rec) != out {
		t.Error("FormatTable không tất định: hai lần gọi cho hai chuỗi khác nhau")
	}
}

// CA ÂM của cổng "đánh dấu bước lệch": hỏi lại (CLARIFICATION) làm S5 và S10 lệch — bước đầu tiên là S5 và dòng chốt chỉ đúng nó.
func TestFormatTableBuocLech(t *testing.T) {
	tu := renderTurn(contracts.RouteClarification, contracts.OutcomeClarificationRequired, "true|clarify", true, contracts.ActionClarify)
	rec := NewRecord(tu)
	out := FormatTable(rec)
	if !strings.Contains(row(t, out, "S5"), MarkDeviant) {
		t.Errorf("S5 hỏi lại mà không có dấu %s: %s", MarkDeviant, row(t, out, "S5"))
	}
	if strings.Contains(row(t, out, "S2"), MarkDeviant) {
		t.Errorf("S2 thẳng mà có dấu %s: %s", MarkDeviant, row(t, out, "S2"))
	}
	if !strings.Contains(out, "BƯỚC LỆCH ĐẦU TIÊN: S5 route") {
		t.Errorf("dòng chốt không chỉ S5:\n%s", out)
	}
}

// Phán quyết hỏng: kiểm trượt ở S9 phải có dấu riêng, và bước lệch đầu tiên là S9 chứ không phải S10 (hệ quả).
func TestFormatTablePhanQuyetHong(t *testing.T) {
	tu := renderTurn(contracts.RouteSQLQuery, contracts.OutcomeSystemFailure, "false|fallback|NUMBER_UNSUPPORTED", false, contracts.ActionFallback)
	tu.Response.Verdict.Violations = []contracts.Violation{{Validator: "bound", Code: "NUMBER_UNSUPPORTED", Severity: "high", Detail: "số 4000 không có bằng chứng"}}
	out := FormatTable(NewRecord(tu))
	if !strings.Contains(row(t, out, "S9"), MarkBadVerdict) {
		t.Errorf("S9 kiểm trượt mà thiếu dấu %q: %s", MarkBadVerdict, row(t, out, "S9"))
	}
	if strings.Contains(row(t, out, "S8"), MarkBadVerdict) || strings.Contains(row(t, out, "S10"), MarkBadVerdict) {
		t.Error("dấu PHÁN QUYẾT HỎNG lan sang bước khác S9")
	}
	if !strings.Contains(out, "BƯỚC LỆCH ĐẦU TIÊN: S9 validate") {
		t.Errorf("dòng chốt không chỉ S9:\n%s", out)
	}
	if !strings.Contains(out, "NUMBER_UNSUPPORTED") {
		t.Error("lý do kiểm trượt (mã vi phạm) không hiện trong chi tiết")
	}
}

// Lượt lỗi: có dấu LỖI, nhãn fault và thông điệp đã làm sạch; bước S? là bước lệch đầu tiên.
func TestFormatTableLuotLoi(t *testing.T) {
	pr := contracts.Principal{ID: "u-1", TenantID: "tenant-x", Kind: contracts.PrincipalUser, Roles: []string{"director"}}
	rec := NewErrorRecord("t-loi", pr, "câu hỏi", time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC), Versions{Pack: "p", Code: "c", Data: "d"},
		contracts.Errorf(contracts.FaultInput, "text rỗng"))
	out := FormatTable(rec)
	if !strings.Contains(row(t, out, "S?"), MarkError) {
		t.Errorf("bước S? của lượt lỗi thiếu dấu %s: %s", MarkError, row(t, out, "S?"))
	}
	for _, want := range []string{MarkError + ": input: text rỗng", "BƯỚC LỆCH ĐẦU TIÊN: S? fault"} {
		if !strings.Contains(out, want) {
			t.Errorf("thiếu %q:\n%s", want, out)
		}
	}
	// lỗi không nhãn tính là lỗi của mã, và thông điệp nhạy cảm đã bị che trước khi tới bảng
	rec2 := NewErrorRecord("t-loi2", pr, "q", time.Now(), Versions{}, errors.New("dial postgres://u:hunter2pw@db:5432/x thất bại"))
	if out2 := FormatTable(rec2); strings.Contains(out2, "hunter2pw") || !strings.Contains(out2, "code") {
		t.Errorf("lỗi không nhãn: muốn nhãn code và mật khẩu bị che:\n%s", out2)
	}
}

// Khoảng trống: chỗ luồng hỏi chưa xuất ra được ghi rõ là TRỐNG — và bước có lý do thật thì KHÔNG bị gắn dấu đó (không đánh dấu bừa).
func TestFormatTableKhoangTrong(t *testing.T) {
	out := FormatTable(NewRecord(renderTurn(contracts.RouteSQLQuery, contracts.OutcomeResolved, "true|accept", true, contracts.ActionAccept)))
	if !strings.Contains(row(t, out, "S2"), MarkGap) {
		t.Errorf("S2 chưa xuất lý do mà thiếu dấu %s: %s", MarkGap, row(t, out, "S2"))
	}
	if strings.Contains(row(t, out, "S0"), MarkGap) {
		t.Errorf("S0 có lý do thật mà bị gắn %s: %s", MarkGap, row(t, out, "S0"))
	}
	if !strings.Contains(out, NotRecorded) {
		t.Errorf("chi tiết không in rõ chỗ trống %q", NotRecorded)
	}
}
