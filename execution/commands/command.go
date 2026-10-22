// Package commands là pipeline lệnh C0–C7 của mặt phẳng Hành động (P4): nhận Principal đã xác thực và một yêu cầu CÓ CẤU TRÚC
// (không chữ tự do), tự tính giá trị từ số liệu có thẩm quyền, rồi ghi ĐÚNG MỘT bản ghi và đọc lại trước khi báo xong.
//
// Hợp đồng: docs/viec/1-hanh-dong/HOP-DONG.md. Nơi đặt mã và ràng buộc import: docs/adr/0011-phase1-hanh-dong-co-kiem-soat.md,
// mục "Nơi đặt mã". Gói này KHÔNG import core/tenant, core/pipeline và KHÔNG có model nào: mọi thứ nó cần (chính sách quyền,
// danh mục chỉ số, kho số liệu, danh mục thực thể, múi giờ, phiên bản pack) do cmd/eais truyền vào lúc dựng (Deps).
//
// Mã dùng chung nên chỉ nói "chỉ số dẫn xuất" và "khoá thực thể": tên tenant, ngành, chỉ số cụ thể nằm ở pack (tools.yaml,
// metrics.yaml, policy.yaml), không nằm ở đây (CLAUDE.md §16, §17).
package commands

import (
	"net/http"
	"strings"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// CommandOpenDiscrepancyTicket là tên lệnh duy nhất của Phase 1: tên tool trong tools.yaml, tên ở đường dẫn route lệnh và
// giá trị cột "hành động" của audit.
const CommandOpenDiscrepancyTicket = "open_discrepancy_ticket"

// TicketOpen là trạng thái duy nhất của phiếu ở Phase 1 (HOP-DONG §8): mọi chuyển trạng thái sau OPEN thuộc Phase 2.
const TicketOpen = "OPEN"

// CapMetricsRead là capability đọc số liệu. Phiếu chứa giá trị của hai vế và kết quả lệnh trả lại chúng, nên người tạo phiếu
// phải đọc được cả hai vế: phiếu không được là đường vòng quyền đọc (HOP-DONG §4.1 lớp 2). Tên chung của nền tảng, không
// phải của tenant nào; capability của chính lệnh (tickets.create) thì do tools.yaml khai.
const CapMetricsRead = "metrics.read"

// Status là một trong ba trị kết quả (HOP-DONG §2.3). Không biết thì UNKNOWN: không làm tròn thành FAILED hay DONE.
type Status string

const (
	StatusDone    Status = "DONE"    // phiếu ĐÃ ghi và ĐÃ đọc lại thấy đúng trong chính lượt này (kể cả phát lại)
	StatusUnknown Status = "UNKNOWN" // không biết phiếu đã ghi hay chưa, hoặc ghi rồi mà đọc lại không xác nhận được
	StatusFailed  Status = "FAILED"  // biết CHẮC lượt này không để lại phiếu mới
)

// ReasonCode là mã lý do (HOP-DONG §2.4). Ba mã đầu dùng lại từ vựng sẵn có của contracts.Outcome khi nghĩa trùng.
type ReasonCode string

const (
	ReasonInvalidRequest       ReasonCode = "INVALID_REQUEST"
	ReasonIdempotencyKeyReused ReasonCode = "IDEMPOTENCY_KEY_REUSED"
	ReasonPermissionDenied     ReasonCode = ReasonCode(contracts.OutcomePermissionDenied)
	ReasonInsufficientEvidence ReasonCode = ReasonCode(contracts.OutcomeInsufficientEvidence)
	ReasonHumanReviewRequired  ReasonCode = ReasonCode(contracts.OutcomeHumanReviewRequired)
	ReasonNoDiscrepancy        ReasonCode = "NO_DISCREPANCY"
	ReasonDuplicateOpenTicket  ReasonCode = "DUPLICATE_OPEN_TICKET"
	ReasonStoreUnavailable     ReasonCode = "STORE_UNAVAILABLE"
	ReasonCommitUnknown        ReasonCode = "COMMIT_UNKNOWN"
	ReasonReadbackUnavailable  ReasonCode = "READBACK_UNAVAILABLE"
	ReasonVerifyMismatch       ReasonCode = "VERIFY_MISMATCH"
	ReasonInternalError        ReasonCode = "INTERNAL_ERROR"
)

// Retry nói bên gọi nên làm gì tiếp (HOP-DONG §2.3).
type Retry string

const (
	RetryNone       Retry = "none"        // không thử lại
	RetrySameKey    Retry = "same_key"    // gửi lại nguyên văn với cùng command_id
	RetryFixRequest Retry = "fix_request" // sửa yêu cầu rồi gửi
	RetryLater      Retry = "later"       // dữ liệu có thể đổi, gửi lại sau
)

// Request là yêu cầu tạo phiếu (HOP-DONG §2.1). Đúng bốn trường, không có trường nào khác: KHÔNG có tenant, người gọi, vai,
// phạm vi, giá trị chênh lệch, trạng thái, mã phiếu, thời gian hay chữ tự do — những thứ đó đến từ Principal, pack và hệ.
// Tên kiểu kết thúc bằng "Request" nên cổng kiến trúc TestRequestTypesHaveNoTenantField tự phủ nó.
type Request struct {
	CommandID  string `json:"command_id"`  // UUID 8-4-4-4-12, không kiểm phiên bản; bên gọi sinh mỗi ý định một lần
	SubjectKey string `json:"subject_key"` // khoá chuẩn của thực thể trong danh mục: không bí danh, không so khớp mờ
	Metric     string `json:"metric"`      // mã chỉ số dẫn xuất, phải thuộc danh sách của tool trong pack
	Day        string `json:"day"`         // YYYY-MM-DD theo múi giờ tenant, phải là ngày đã qua
}

// Operand là một vế của phép tính: chỉ số, giá trị thập phân dạng chuỗi chuẩn, đơn vị.
type Operand struct {
	Metric string `json:"metric"`
	Value  string `json:"value"`
	Unit   string `json:"unit"`
}

// Ticket là phiếu trong kết quả. Khi DONE có đủ trường; khi DUPLICATE_OPEN_TICKET chỉ có ID và Status (mã phiếu cũ).
// Approx là con trỏ để "không có thông tin" (nil) khác "không ước lượng" (false): R6, cờ không được mất.
type Ticket struct {
	ID         string   `json:"id"`
	Status     string   `json:"status"`
	SubjectKey string   `json:"subject_key,omitempty"`
	Metric     string   `json:"metric,omitempty"`
	Day        string   `json:"day,omitempty"`
	Left       *Operand `json:"left,omitempty"`
	Right      *Operand `json:"right,omitempty"`
	Diff       string   `json:"diff,omitempty"`
	Approx     *bool    `json:"approx,omitempty"`
}

// Result là kết quả của một lượt lệnh (HOP-DONG §2.3). Bên gọi dựa vào Status và ReasonCode, không dựa vào mã HTTP.
type Result struct {
	Status     Status     `json:"status"`
	ReasonCode ReasonCode `json:"reason_code"`
	// Detail là câu CỐ ĐỊNH theo mã lý do, hệ đặt, bên gọi không đưa; lỗi hạ tầng và mã chỉ có câu chung, không chi tiết nội bộ.
	Detail string `json:"detail"`
	// InvalidFields chỉ có khi INVALID_REQUEST: tên trường của Request mà chặng từ chối thấy là sai, theo thứ tự command_id,
	// subject_key, metric, day.
	InvalidFields []string `json:"invalid_fields,omitempty"`
	// Fault là nhãn đầy đủ của docs/CONVENTIONS.md §4.3, ví dụ "infra:postgres". Qua HTTP chỉ còn lớp lỗi: FaultClass.
	Fault    string  `json:"fault,omitempty"`
	Retry    Retry   `json:"retry"`
	Replayed bool    `json:"replayed"` // chỉ có nghĩa khi DONE
	Ticket   *Ticket `json:"ticket,omitempty"`
	// TraceID để tra vết bằng eais trace. Pipeline luôn cấp một mã ở C0; nơi LƯU vết (trace.go, cmd/eais) phải xoá nó khi
	// không lưu được (HOP-DONG §7.5, "Mất vết"): kết quả đã ghi không đổi, chỉ trace_id rỗng.
	TraceID string `json:"trace_id"`
}

// HTTPStatus là mã HTTP của kết quả: DONE (kể cả phát lại) là 200; còn lại suy từ nhãn fault qua contracts.HTTPStatus, không lập
// bảng riêng (HOP-DONG §2.6.2): input 400, policy 403, data 409, infra:* 503, code 500.
func (r Result) HTTPStatus() int {
	if r.Status == StatusDone {
		return http.StatusOK
	}
	return contracts.HTTPStatus(contracts.Errorf(r.Fault, string(r.ReasonCode)))
}

// FaultClass là phần trước dấu hai chấm của Fault ("infra:postgres" → "infra"): thứ duy nhất được ra khỏi tiến trình qua HTTP.
func (r Result) FaultClass() string {
	class, _, _ := strings.Cut(r.Fault, ":")
	return class
}

// withFields trả bản sao có invalid_fields; danh sách rỗng giữ nil để JSON không in trường này.
func (r Result) withFields(fields []string) Result {
	if len(fields) > 0 {
		r.InvalidFields = fields
	}
	return r
}

// reasonInfo là một dòng của bảng mã lý do (HOP-DONG §2.4): status, nhãn fault đầy đủ, retry và câu detail cố định.
type reasonInfo struct {
	status Status
	fault  string
	retry  Retry
	detail string
}

// reasons là bảng §2.4, một nguồn sự thật cho status, fault, retry và detail của mọi kết cục không phải DONE. Detail viết cho bên
// gọi: nói điều cần sửa hay lý do từ chối; PERMISSION_DENIED chỉ có MỘT câu cho mọi nguyên nhân (§4.4); không tên thành phần,
// lỗi trình điều khiển, đường dẫn hay SQL; KHÔNG dùng publicMessage của /v1/ask (nó gộp mọi lớp ngoài input, policy thành
// "lỗi hệ thống", sai với từ chối nghiệp vụ lớp data).
var reasons = map[ReasonCode]reasonInfo{
	ReasonInvalidRequest: {StatusFailed, "input", RetryFixRequest,
		"Yêu cầu không hợp lệ: sửa các trường sai (nếu có nêu ở invalid_fields) rồi gửi lại; tenant, người gọi, vai và phạm vi chỉ lấy từ token, không đưa vào thân yêu cầu."},
	ReasonIdempotencyKeyReused: {StatusFailed, "input", RetryFixRequest,
		"command_id này đã dùng cho một yêu cầu khác nội dung: gửi lại đúng nội dung cũ, hoặc dùng command_id mới cho yêu cầu mới."},
	ReasonPermissionDenied: {StatusFailed, "policy", RetryNone,
		"Bạn không có quyền tạo phiếu cho yêu cầu này."},
	ReasonInsufficientEvidence: {StatusFailed, "data", RetryLater,
		"Chưa đủ số liệu có thẩm quyền để tính chênh lệch của thực thể trong ngày này (thiếu một vế); thử lại khi dữ liệu đã bổ sung."},
	ReasonHumanReviewRequired: {StatusFailed, "data", RetryLater,
		"Một vế có hai nguồn ngang thẩm quyền nói khác nhau quá dung sai; cần người quyết trước khi tạo phiếu."},
	ReasonNoDiscrepancy: {StatusFailed, "data", RetryNone,
		"Chênh lệch không vượt ngưỡng nên không tạo phiếu."},
	ReasonDuplicateOpenTicket: {StatusFailed, "data", RetryNone,
		"Đã có phiếu đang mở cho cùng thực thể, chỉ số và ngày; xem mã phiếu ở trường ticket."},
	ReasonStoreUnavailable: {StatusFailed, contracts.Infra("postgres"), RetrySameKey,
		"Hệ thống lưu trữ tạm thời không dùng được và yêu cầu chưa được ghi; gửi lại với cùng command_id."},
	ReasonCommitUnknown: {StatusUnknown, contracts.Infra("postgres"), RetrySameKey,
		"Chưa biết yêu cầu đã được ghi hay chưa; gửi lại với cùng command_id để biết kết quả."},
	ReasonReadbackUnavailable: {StatusUnknown, contracts.Infra("postgres"), RetrySameKey,
		"Yêu cầu đã được ghi nhưng chưa đọc lại để xác nhận được; gửi lại với cùng command_id."},
	ReasonVerifyMismatch: {StatusUnknown, contracts.FaultCode, RetryNone,
		"Phiếu đã được ghi nhưng đọc lại không khớp; cần người xem audit, không tạo lại bằng command_id mới."},
	// INTERNAL_ERROR mặc định FAILED (lỗi trước BEGIN, chắc chắn chưa ghi); internalResult đổi sang UNKNOWN khi đã sau BEGIN.
	ReasonInternalError: {StatusFailed, contracts.FaultCode, RetryNone,
		"Lỗi hệ thống không lường trước; yêu cầu chưa được ghi. Báo người vận hành kèm trace_id."},
}

// internalUnknownDetail là câu của INTERNAL_ERROR khi đã sau BEGIN mà không xác nhận được rollback: không được nói "chưa ghi".
const internalUnknownDetail = "Lỗi hệ thống không lường trước và chưa biết yêu cầu đã được ghi hay chưa. Báo người vận hành kèm trace_id."

// resultFor dựng kết quả không phải DONE từ bảng mã lý do.
func resultFor(code ReasonCode) Result {
	ri := reasons[code]
	return Result{Status: ri.status, ReasonCode: code, Detail: ri.detail, Fault: ri.fault, Retry: ri.retry}
}

// internalResult dựng INTERNAL_ERROR: FAILED nếu lỗi trước BEGIN, UNKNOWN nếu sau BEGIN mà không xác nhận được rollback.
func internalResult(status Status) Result {
	r := resultFor(ReasonInternalError)
	if status == StatusUnknown {
		r.Status, r.Detail = StatusUnknown, internalUnknownDetail
	}
	return r
}
