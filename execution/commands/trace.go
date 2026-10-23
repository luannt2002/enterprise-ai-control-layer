package commands

// Vết của lượt lệnh (HOP-DONG §7.5): dựng traces.Record chín bước C0…C7 từ Outcome của pipeline. Pipeline chỉ trả dữ liệu thô (Outcome, Step) và không
// import observability/traces; tệp này là chỗ DUY NHẤT của gói nói tiếng của kho vết, để kho vết lưu và `eais trace` đọc được lượt lệnh như lượt hỏi.
//
// Vết khác audit: audit là bản ghi nghiệp vụ chỉ-thêm của quyết định (EXECUTED nằm trong đơn vị ghi phiếu); vết là bản ghi gỡ lỗi từng bước của MỘT
// lượt, kể cả lượt bị từ chối ở C0 đến C3 (không có audit). Hai thứ nối nhau bằng trace_id (cột của audit). Nguồn sự thật về phiếu là bảng phiếu cộng
// audit, không phải vết. Vết không lộ điều mà Result không lộ: bước dựng sẵn từ dữ liệu pipeline đã giấu (§4.4), tệp này không thêm gì ngoài chúng.

import (
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// NewRecord dựng bản ghi vết của một lượt lệnh. req là yêu cầu đã giải mã (rỗng và bị bỏ qua khi thân không đọc được: Outcome.Question rỗng); pr là
// Principal lấy từ token (lượt dừng ở C1 vẫn mang đúng Principal đã gọi); now là đồng hồ của lượt; v là ba phiên bản lúc chạy; totalMs là thời gian cả
// lượt do nơi gọi đo. Gói không đo từng chặng nên mọi bước "chưa đo" (Timed = false) và phần "ngoài mọi chặng" bằng cả lượt, không bịa thời gian.
//
// Trường quyết định dùng đúng kiểu traces.Decision, trường nào luồng lệnh không có nghĩa thì để rỗng (Normalized, Verdict). Question là dạng chuẩn
// của yêu cầu; Intent là tên lệnh; Route là TOOL; Entities là khoá máy chỉ khi đã qua C4; Evidence là mã bằng chứng của bản chụp (từ C5); Outcome là
// DONE hoặc mã lý do; Answer là "status=… reason_code=… replayed=… ticket=…". Bản ghi phải qua traces.Sanitize trước khi lưu: kho vết làm việc đó,
// và Error của từng bước (lỗi gốc chưa làm sạch của hạ tầng hoặc mã) được làm sạch ngay ở đây.
func NewRecord(req Request, out Outcome, pr contracts.Principal, now time.Time, v traces.Versions, totalMs int64) traces.Record {
	res := out.Result
	d := traces.Decision{
		TraceID: res.TraceID, Tenant: string(pr.TenantID), Principal: pr.ID, Kind: string(pr.Kind), Roles: slices.Clone(pr.Roles),
		Attributes: maps.Clone(pr.Attributes), Now: now, PackVersion: v.Pack, CodeVersion: v.Code, DataVersion: v.Data,
		Question: out.Question, Normalized: []string{}, Entities: []string{}, Slots: []string{}, Evidence: slices.Clone(out.Evidence),
		Intent: CommandOpenDiscrepancyTicket, Route: string(contracts.RouteTool), Outcome: outcomeText(res), Answer: answerText(res),
	}
	if d.Evidence == nil {
		d.Evidence = []string{}
	}
	if out.Question != "" { // thân đọc được: có các tham số của lệnh
		d.Slots = []string{"metric=" + clean(req.Metric), "day=" + clean(req.Day)}
	}
	if out.Entity != "" {
		d.Entities = []string{out.Entity}
	}
	rec := traces.Record{Decision: d, Steps: make([]traces.Step, 0, len(out.Steps)), Spans: []traces.Span{}, TotalMs: totalMs, UnaccountedMs: totalMs}
	for i, s := range out.Steps {
		ts := traces.Step{
			TraceID: res.TraceID, Seq: i, Stage: s.Code, Name: s.Name, Status: s.Status, Input: s.Input, Output: s.Output, Decision: s.Decision,
			Reason: s.Reason, Rejected: s.Rejected, PackVersion: v.Pack, CodeVersion: v.Code, DataVersion: v.Data, Fault: s.Fault,
			Error: traces.Sanitize(s.Error),
		}
		if ts.Decision == "" && s.Status == StepSkipped {
			ts.Decision = "không chạy ở lượt này"
		}
		for _, id := range s.Evidence {
			ts.Evidence = append(ts.Evidence, traces.EvidenceRef{ID: id})
		}
		rec.Steps = append(rec.Steps, ts)
		if rec.Fault == "" && s.Status == StepError { // lượt hạ tầng hoặc mã hỏng: nhãn và thông điệp đã làm sạch cũng nằm ở cấp bản ghi (cột fault của kho vết)
			rec.Fault, rec.Error = ts.Fault, ts.Error
		}
	}
	return rec
}

func outcomeText(r Result) string {
	if r.Status == StatusDone {
		return string(StatusDone)
	}
	return string(r.ReasonCode)
}

func answerText(r Result) string {
	ticket := ""
	if r.Ticket != nil {
		ticket = r.Ticket.ID
	}
	return fmt.Sprintf("status=%s reason_code=%s replayed=%t ticket=%s", r.Status, r.ReasonCode, r.Replayed, ticket)
}
