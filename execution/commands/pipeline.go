package commands

// Pipeline lệnh C0–C7 (HOP-DONG §3). Tất cả chặng tất định, 0 lời gọi model. Mỗi lượt trả Result và đủ chín bước C0, C1, C2, C3,
// C4, C4b, C5, C6, C7 (§7.5) dưới dạng dữ liệu thô: gói này KHÔNG import observability/traces, một bước khác (trace.go) dựng bản
// ghi vết từ Outcome. Điểm chèn lỗi F1–F6, F8 và rào F7 chỉ là móc test: chúng nằm ở lớp bọc quanh Store trong _test.go, đường
// chạy thường không có móc nào.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"runtime/debug"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/decimal"
	"github.com/luannt2002/enterprise-ai-control-layer/core/policy"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
)

// ---------------------------------------------------------------------------
// Vết thô: chín bước
// ---------------------------------------------------------------------------

// Trạng thái của một bước (cùng nghĩa với traces.Step, HOP-DONG §7.5).
const (
	StepOK      = "ok"      // đi qua
	StepSkipped = "skipped" // không chạy ở lượt này, kèm lý do
	StepDeviant = "deviant" // từ chối theo luật (kèm nhãn fault)
	StepError   = "error"   // hạ tầng hoặc mã (kèm nhãn fault)
)

// Chỉ số chín bước, theo thứ tự cố định của vết.
const (
	stAccept = iota
	stTenant
	stUnderstand
	stParams
	stPermission
	stLedger
	stRules
	stWrite
	stVerify
	numStages
)

var stages = [numStages]struct{ code, name string }{
	{"C0", "accept"}, {"C1", "tenant"}, {"C2", "understand"}, {"C3", "params"}, {"C4", "permission"},
	{"C4b", "ledger"}, {"C5", "rules"}, {"C6", "write"}, {"C7", "verify"},
}

// Step là dữ liệu thô của MỘT bước, đủ để trace.go dựng traces.Step. Không chứa điều mà Result không lộ: vết của lượt
// PERMISSION_DENIED không tách "không tồn tại" với "ngoài phạm vi" (HOP-DONG §4.4, §7.5).
type Step struct {
	Code     string // C0…C7, C4b
	Name     string // accept, tenant, understand, params, permission, ledger, rules, write, verify
	Status   string // StepOK, StepSkipped, StepDeviant, StepError
	Input    []string
	Output   []string
	Decision string
	Reason   string
	Rejected []string // phương án đã xét và bị loại (C5: nguồn và số bị loại khi hai nguồn ngang thẩm quyền lệch)
	Evidence []string // mã bằng chứng (C5)
	// Fault là nhãn fault của bước dừng. Bước lỗi hạ tầng mang nhãn THẬT của thành phần hỏng khi tầng dưới đã gắn (contracts.FaultOf, ví dụ
	// infra:directory), không thì nhãn của bảng §2.4 như ở Result: Result giữ nhãn của bảng, còn bảng vết nói đúng thành phần.
	Fault string
	// Error là lỗi gốc CHƯA làm sạch của bước lỗi. Chỉ trace.go được ghi nó vào vết, và phải qua traces.Sanitize trước khi lưu.
	Error string
}

// Outcome là mọi thứ một lượt để lại.
type Outcome struct {
	Result Result
	Steps  []Step // luôn đủ chín bước theo thứ tự
	// Question là dạng chuẩn của yêu cầu đã làm sạch ("open_discrepancy_ticket subject_key=… metric=… day=… command_id=…");
	// rỗng khi thân không đọc được (Malformed). Entity là khoá thực thể, chỉ khi đã qua C4. Evidence là mã bằng chứng của bản
	// chụp ý định (từ C5): cùng ba thứ dùng cho traces.Decision (HOP-DONG §7.5).
	Question string
	Entity   string
	Evidence []string
	// Cause là lỗi gốc của hạ tầng hoặc mã, CHỈ để ghi log: không bao giờ đưa ra ngoài qua Result.
	Cause error
}

// ---------------------------------------------------------------------------
// Giao diện lưu trữ (khai tại chỗ dùng) và các bản ghi của nó
// ---------------------------------------------------------------------------

// Caller là người gọi, lấy từ Principal.
type Caller struct {
	ID          string
	Roles       []string
	EntityScope string
}

// EvidenceSnap là một bằng chứng của kết quả Derive tại LÚC tính (HOP-DONG §5, "Bản chụp bằng chứng"): Load thay toàn bộ fact
// của tenant nên mã fact cũ có thể không còn; phiếu giữ cả nguồn, giá trị, ngày và hạng thẩm quyền, không chỉ mã.
type EvidenceSnap struct {
	ID         string
	Kind       string
	SourceType string
	SourceRef  string
	Value      string
	Unit       string
	Day        string
	Authority  int
}

// TicketRecord là một dòng của bảng phiếu (HOP-DONG §7.4). Thời gian tạo do cơ sở dữ liệu gán nên không nằm ở đây và không
// được so khi đọc lại. Không có khoá ngoại tới danh mục hay fact: phiếu giữ khoá và mã thực thể dạng chữ.
type TicketRecord struct {
	ID          string
	Status      string // TicketOpen
	EntityID    string // mã thực thể trong danh mục lúc tính
	SubjectKey  string // khoá thực thể: khoá nghiệp vụ chống trùng dùng khoá này, không dùng mã
	Metric      string
	Day         string
	Left, Right Operand
	Diff        string
	Approx      bool
	Formula     string
	Evidence    []EvidenceSnap
	CreatedBy   string // Principal.ID
	PackVersion string
	CodeVersion string
	CommandID   string // cột thường, không có khoá ngoại tới sổ lệnh (§7.1)
}

// LedgerRecord là một dòng sổ lệnh: khoá chính (tenant, Caller, CommandID); khoá ngoại thường tới phiếu (TicketID).
type LedgerRecord struct {
	Caller      string
	CommandID   string
	Fingerprint string
	TicketID    string
}

// AuditOutcome là kết cục của một dòng audit (HOP-DONG §7.3).
type AuditOutcome string

const (
	AuditExecuted     AuditOutcome = "EXECUTED"      // nằm TRONG đơn vị ghi phiếu
	AuditReplayed     AuditOutcome = "REPLAYED"      // giao dịch riêng
	AuditRefused      AuditOutcome = "REFUSED"       // giao dịch riêng, kèm ReasonCode
	AuditVerifyFailed AuditOutcome = "VERIFY_FAILED" // giao dịch riêng
)

// Audit là một dòng audit_log. ReasonDetail là lý do THẬT (ví dụ phân biệt "không tồn tại" với "ngoài phạm vi") và chỉ nằm
// ở đây, không bao giờ ở Result hay vết. Không có bí mật, token hay thứ gì ngoài các cột này.
type Audit struct {
	Outcome      AuditOutcome
	ReasonCode   ReasonCode
	ReasonDetail string
	Caller       Caller
	CommandID    string
	Action       string
	SubjectKey   string
	Metric       string
	Day          string
	TicketID     string
	Fingerprint  string
	PackVersion  string
	CodeVersion  string
	TraceID      string
}

// WriteUnit là thứ đơn vị ghi phiếu ghi: ba dòng, đúng thứ tự phiếu, sổ lệnh, audit EXECUTED, trong MỘT giao dịch.
type WriteUnit struct {
	Ticket TicketRecord
	Ledger LedgerRecord
	Audit  Audit
}

// Stored là điều đọc lại thấy ở nơi ghi, bằng giao dịch mới: phiếu theo mã, sổ lệnh theo khoá, audit EXECUTED theo command_id.
type Stored struct {
	Tickets  []TicketRecord
	Ledger   []LedgerRecord
	Executed []Audit
}

// TicketRef là phiếu OPEN tìm thấy theo khoá nghiệp vụ.
type TicketRef struct{ ID, Status string }

// Ba loại lỗi mà Store PHẢI phân biệt ở đơn vị ghi phiếu (HOP-DONG §7.1). Store bọc lỗi gốc bằng %w; lỗi không mang một trong
// ba loại này bị coi là không phân loại được và cho ra UNKNOWN INTERNAL_ERROR (không bao giờ FAILED).
var (
	// ErrBeforeCommit: biết CHẮC chưa ghi gì — PG báo lỗi câu lệnh và giao dịch đã huỷ, hoặc lỗi kết nối / hết hạn TRƯỚC khi gửi
	// COMMIT (kể cả hết hạn khi đang chờ khoá của giao dịch khác).
	ErrBeforeCommit = errors.New("lỗi trước khi gửi COMMIT, chắc chắn chưa ghi")
	// ErrCommitUnknown: lỗi hoặc hết hạn từ lúc gửi COMMIT, không biết kết quả.
	ErrCommitUnknown = errors.New("lỗi từ lúc gửi COMMIT, không biết kết quả")
	// ErrUniqueViolation: vi phạm khoá duy nhất (SQLSTATE 23505); giao dịch đã huỷ. Pipeline đọc lại để phân loại (§6.4).
	ErrUniqueViolation = errors.New("vi phạm khoá duy nhất")
)

// Store là lớp lưu trữ của phiếu, do gói khai tại chỗ dùng. Mọi hàm lấy tenant từ ngữ cảnh (contracts.TenantFrom, do C1 gắn)
// và từ chối khi thiếu; không hàm nào nhận tenant qua tham số.
type Store interface {
	// FindLedger tra sổ lệnh theo (người gọi, command_id) — C4b và §6.4.
	FindLedger(ctx context.Context, caller, commandID string) (LedgerRecord, bool, error)
	// FindOpenTicket tìm phiếu OPEN theo khoá nghiệp vụ (khoá thực thể, chỉ số, ngày) — R5 ở C5 và §6.4.
	FindOpenTicket(ctx context.Context, subjectKey, metric, day string) (TicketRef, bool, error)
	// WriteTicket là đơn vị ghi phiếu, lời gọi DUY NHẤT của lớp lưu trữ ở C6: một giao dịch đã đặt tenant, ba INSERT theo thứ tự
	// phiếu, sổ lệnh, audit EXECUTED, rồi COMMIT. Lỗi trả theo ba loại ở trên.
	WriteTicket(ctx context.Context, u WriteUnit) error
	// ReadBack đọc lại bằng giao dịch MỚI từ nơi ghi (không replica), không dùng RETURNING của lệnh INSERT (§7.2).
	ReadBack(ctx context.Context, caller, commandID, ticketID string) (Stored, error)
	// WriteAudit ghi MỘT dòng audit bằng giao dịch riêng (REFUSED, REPLAYED, VERIFY_FAILED), cố gắng tối đa.
	WriteAudit(ctx context.Context, a Audit) error
}

// ---------------------------------------------------------------------------
// Pipeline
// ---------------------------------------------------------------------------

// Deps là mọi thứ pipeline cần; cmd/eais truyền vào lúc dựng.
type Deps struct {
	Tenant    string         // tenant của runtime: Principal.TenantID phải bằng nó (C1)
	Location  *time.Location // múi giờ tenant: "hôm nay" và ngày của yêu cầu
	Tool      Tool           // cấu hình tool (tools.yaml)
	Policy    policy.Policy
	Catalog   *sourceoftruth.Catalog
	Facts     sourceoftruth.Store
	Directory entityresolution.Directory
	Store     Store
	// PackVersion, CodeVersion ghi vào phiếu và audit.
	PackVersion string
	CodeVersion string
	// Now nil = time.Now. NewID nil = UUID v4; dùng cho mã phiếu và trace_id.
	Now   func() time.Time
	NewID func() string
}

// Pipeline là pipeline lệnh của một tenant. An toàn cho nhiều lượt chạy đồng thời: không có trạng thái riêng.
type Pipeline struct{ d Deps }

// New kiểm các phụ thuộc bắt buộc và dựng pipeline. Nó KHÔNG kiểm ba luật nạp của §4.3 (việc của LoadTools lúc khởi động):
// lớp kiểm quyền lúc chạy vẫn đóng khi cấu hình tay thiếu metrics.read (phòng thủ chiều sâu, ca A09).
func New(d Deps) (*Pipeline, error) {
	var missing []string
	add := func(bad bool, name string) {
		if bad {
			missing = append(missing, name)
		}
	}
	add(d.Tenant == "", "Tenant")
	add(d.Location == nil, "Location")
	add(d.Tool.Capability == "", "Tool.Capability")
	add(len(d.Tool.Metrics) == 0, "Tool.Metrics")
	add(d.Catalog == nil, "Catalog")
	add(d.Facts == nil, "Facts")
	add(d.Directory == nil, "Directory")
	add(d.Store == nil, "Store")
	if len(missing) > 0 {
		return nil, fmt.Errorf("commands.New: thiếu %s", strings.Join(missing, ", "))
	}
	if d.Tool.MinAbsDiff == "" {
		d.Tool.MinAbsDiff = "0"
	}
	if msg := minAbsDiffProblem(d.Tool.MinAbsDiff); msg != "" { // cùng phép kiểm với LoadTools: ngưỡng âm vô hiệu hoá R4
		return nil, fmt.Errorf("commands.New: Tool.MinAbsDiff %s", msg)
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.NewID == nil {
		d.NewID = uuid.NewString
	}
	return &Pipeline{d: d}, nil
}

// Run chạy một lượt lệnh C0 → C7. Luôn trả Outcome: mọi thất bại là một Result, không có lỗi riêng; panic của kho, danh mục, số liệu hay hàm
// của Deps cũng được bắt lại và thành INTERNAL_ERROR (panicked). Lượt chạy dưới một hạn chót: cái đến trước giữa ngữ cảnh của bên gọi và
// Timeout của tool (nếu có).
func (p *Pipeline) Run(ctx context.Context, pr contracts.Principal, req Request) (out Outcome) {
	var r *run
	defer func() {
		if rec := recover(); rec != nil {
			out = p.recovered(r, pr, req, rec)
		}
	}()
	if p.d.Tool.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.d.Tool.Timeout)
		defer cancel()
	}
	r = p.newRun(pr, req)
	ctx = r.exec(ctx)
	r.conclude(ctx)
	return r.outcome()
}

// recovered dựng Outcome cho lượt bị panic. r nil nghĩa là panic ngay lúc dựng lượt (Now hay NewID của Deps): dựng lượt tối thiểu, mã vết lấy
// thẳng từ uuid vì chính NewID có thể là thứ vừa panic.
func (p *Pipeline) recovered(r *run, pr contracts.Principal, req Request, rec any) Outcome {
	if r == nil {
		r = &run{p: p, pr: pr, req: req, traceID: uuid.NewString(), question: question(req), stop: -1}
	}
	r.panicked(rec)
	return r.outcome()
}

// panicked là lưới đỡ cuối của Run: một panic không được thoát ra ngoài làm mất Result, vết và câu trả lời "chưa biết" của một phiếu có thể đã
// commit. Chưa gọi đơn vị ghi phiếu (và chưa thấy dòng sổ lệnh nào của khoá này) thì chắc chắn chưa ghi: INTERNAL_ERROR FAILED. Từ lúc gọi nó,
// hoặc khi lượt đang phát lại một phiếu có thật mà chưa xác nhận được, thì không biết: INTERNAL_ERROR UNKNOWN (§2.4, §9.2.1). Bước đang chạy
// là bước error; C7 vẫn trả kết quả (không đọc lại, không audit: kho có thể chính là thứ vừa panic). Lỗi gốc ở Cause kèm ngăn xếp để ghi log;
// Step.Error chỉ một dòng.
func (r *run) panicked(rec any) {
	err := fmt.Errorf("panic: %v", rec)
	if e, ok := rec.(error); ok {
		err = fmt.Errorf("panic: %w", e)
	}
	res, noRead := internalResult(StatusFailed), "lỗi mã (panic) trước khi ghi"
	if r.unit != nil || r.replay != nil {
		res, noRead = internalResult(StatusUnknown), "lỗi mã (panic) sau khi ghi hoặc phát lại, chưa xác nhận được"
	}
	st := r.at
	r.setStep(st, Step{Status: StepError, Fault: res.Fault, Error: err.Error(), Decision: "lỗi mã", Reason: "panic không lường trước"})
	r.res, r.noRead = res, noRead
	r.cause = fmt.Errorf("%w\n%s", err, debug.Stack())
	if st != stVerify {
		r.stop = st
		r.setStep(stVerify, Step{Status: StepOK, Output: []string{"không đọc lại: " + noRead}, Decision: "trả kết quả", Reason: "không đọc lại: " + noRead})
	}
}

// Malformed dựng kết quả cho thân yêu cầu mà lớp HTTP không giải mã được (JSON hỏng, trường lạ, nội dung thừa): INVALID_REQUEST
// dừng ở C0 với đủ chín bước, để lượt đó vẫn có vết (HOP-DONG §2.6.2, A28 (b)). fields là tên trường lạ đầu tiên, hoặc rỗng khi
// lỗi không quy về một trường.
func (p *Pipeline) Malformed(fields ...string) Outcome {
	r := p.newRun(contracts.Principal{}, Request{})
	r.question = ""
	r.stopStep(stAccept, StepDeviant, resultFor(ReasonInvalidRequest).withFields(fields), Step{
		Output:   []string{"invalid_fields=" + strings.Join(fields, ",")},
		Decision: "từ chối", Reason: "thân yêu cầu không giải mã được hoặc có trường lạ",
	})
	r.noRead = "lượt dừng ở C0"
	r.conclude(context.Background())
	return r.outcome()
}

// run là trạng thái của MỘT lượt.
type run struct {
	p        *Pipeline
	pr       contracts.Principal
	req      Request
	caller   Caller
	traceID  string
	fp       string
	today    string
	question string

	steps [numStages]Step
	done  [numStages]bool
	stop  int // chỉ số chặng dừng; -1 khi lượt chưa dừng

	res    Result
	cause  error
	noRead string // vì sao C7 không đọc lại

	ent       entityresolution.Entity
	entFound  bool          // đã qua C4
	draft     *TicketRecord // bản chụp ý định dựng ở C5, chưa có mã phiếu
	evidence  []string
	unit      *WriteUnit
	committed bool
	replay    *LedgerRecord
	audit     *Audit // dòng audit riêng phải ghi ở C7 (REFUSED)
	at        int    // chặng đang chạy (enter): nếu chặng đó panic thì đây là bước lỗi (panicked)
}

func (p *Pipeline) newRun(pr contracts.Principal, req Request) *run {
	return &run{
		p: p, pr: pr, req: req,
		caller:   Caller{ID: pr.ID, Roles: slices.Clone(pr.Roles), EntityScope: pr.Attributes["entity_scope"]},
		traceID:  p.d.NewID(),
		fp:       fingerprint(req),
		today:    p.d.Now().In(p.d.Location).Format(time.DateOnly),
		question: question(req),
		stop:     -1,
	}
}

// exec chạy C0 → C6 và trả ngữ cảnh đã gắn tenant để C7 dùng tiếp. Mỗi chặng trả true khi lượt đi tiếp.
func (r *run) exec(ctx context.Context) context.Context {
	if !r.enter(stAccept).accept() || !r.enter(stTenant).tenant() {
		return ctx
	}
	ctx = contracts.WithTenant(ctx, r.pr.TenantID) // từ đây mọi truy vấn mang tenant của Principal
	r.setStep(stUnderstand, Step{Status: StepSkipped, Reason: "Phase 1 không dùng C2: đầu vào đã có cấu trúc, không cần hiểu lệnh"})
	if r.enter(stParams).params() && r.enter(stPermission).permission(ctx) && r.enter(stLedger).ledger(ctx) && r.enter(stRules).rules(ctx) {
		r.enter(stWrite).write(ctx)
	}
	return ctx
}

// enter ghi lại chặng sắp chạy, để panic ở chặng đó được đánh dấu đúng bước (panicked).
func (r *run) enter(st int) *run {
	r.at = st
	return r
}

// ---------------------------------------------------------------------------
// C0 accept
// ---------------------------------------------------------------------------

// accept kiểm hình dạng: đủ bốn trường, command_id đúng dạng UUID, không trường nào có ký tự NUL hay chuỗi UTF-8 hỏng (PostgreSQL từ chối cả
// hai ở câu INSERT phiếu và audit). Không audit.
func (r *run) accept() bool {
	in := []string{
		"command_id=" + echo(r.req.CommandID), "subject_key=" + echo(r.req.SubjectKey),
		"metric=" + echo(r.req.Metric), "day=" + echo(r.req.Day),
	}
	if bad := shapeProblems(r.req); len(bad) > 0 {
		r.stopStep(stAccept, StepDeviant, resultFor(ReasonInvalidRequest).withFields(bad), Step{
			Input: in, Output: []string{"invalid_fields=" + strings.Join(bad, ",")},
			Decision: "từ chối", Reason: "trường thiếu, sai dạng, quá dài, có ký tự NUL hoặc chuỗi UTF-8 hỏng",
		})
		r.noRead = "lượt dừng ở C0"
		return false
	}
	r.req.CommandID = strings.ToLower(r.req.CommandID) // UUID không phân biệt hoa thường: một khoá, một cách viết
	r.setStep(stAccept, Step{Status: StepOK, Input: in, Output: []string{"accepted"},
		Decision: "chấp nhận", Reason: "đủ bốn trường, command_id đúng dạng UUID, không có ký tự NUL"})
	return true
}

// maxSubjectKeyBytes là trần độ dài (byte UTF-8) của subject_key kiểm ở C0 (C5, lệnh ca chiều 23/10). Khoá danh mục là mã ngắn (EX-03);
// không có trần thì một khoá dài tuỳ ý đi vào câu hỏi chuẩn, vết, dòng audit và sổ lệnh. Đây là giới hạn cấu trúc, không phải giá trị
// nghiệp vụ.
const maxSubjectKeyBytes = 128

// shapeProblems trả tên các trường sai hình dạng theo thứ tự command_id, subject_key, metric, day.
func shapeProblems(req Request) []string {
	var bad []string
	if !validUUID(req.CommandID) {
		bad = append(bad, "command_id")
	}
	for _, f := range []struct{ name, value string }{{"subject_key", req.SubjectKey}, {"metric", req.Metric}, {"day", req.Day}} {
		if f.value == "" || strings.ContainsRune(f.value, 0) || !utf8.ValidString(f.value) ||
			(f.name == "subject_key" && len(f.value) > maxSubjectKeyBytes) {
			bad = append(bad, f.name)
		}
	}
	return bad
}

// validUUID kiểm đúng dạng 8-4-4-4-12 chữ số hex, không kiểm phiên bản (uuid.Parse còn nhận dạng khác nên không dùng).
func validUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if c != '-' {
				return false
			}
		case !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'):
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// C1 tenant
// ---------------------------------------------------------------------------

// tenant: Principal phải có mã và thuộc đúng tenant của runtime. Không audit (chưa có tenant tin cậy để gắn).
func (r *run) tenant() bool {
	in := []string{
		"principal id=" + clean(r.pr.ID) + " kind=" + string(r.pr.Kind) + " roles=" + clean(strings.Join(r.pr.Roles, ",")) +
			" entity_scope=" + clean(r.caller.EntityScope),
	}
	if r.pr.ID == "" || string(r.pr.TenantID) != r.p.d.Tenant {
		r.stopStep(stTenant, StepDeviant, resultFor(ReasonPermissionDenied), Step{
			Input: in, Output: []string{"tenant của người gọi không khớp runtime"},
			Decision: "từ chối", Reason: "tenant lấy từ Principal và phải khớp tenant của runtime",
		})
		r.noRead = "lượt dừng ở C1"
		return false
	}
	r.setStep(stTenant, Step{Status: StepOK, Input: in, Output: []string{"tenant=" + r.p.d.Tenant},
		Decision: "cho qua", Reason: "tenant lấy từ Principal, khớp tenant của runtime"})
	return true
}

// ---------------------------------------------------------------------------
// C3 params
// ---------------------------------------------------------------------------

// params: metric thuộc danh sách của tool; day đúng dạng và là ngày đã qua theo múi giờ tenant. Không audit.
func (r *run) params() bool {
	// echo(): metric và day chưa được kiểm ở đây — giá trị dài hay bẩn không đi nguyên vào vết (soát C1–C7, C17S-04)
	in := []string{"metric=" + echo(r.req.Metric), "day=" + echo(r.req.Day), "today=" + r.today, "metrics_allowed=" + strings.Join(r.p.d.Tool.Metrics, ",")}
	var bad []string
	if !slices.Contains(r.p.d.Tool.Metrics, r.req.Metric) {
		bad = append(bad, "metric")
	}
	if !pastDay(r.req.Day, r.today) {
		bad = append(bad, "day")
	}
	if len(bad) > 0 {
		r.stopStep(stParams, StepDeviant, resultFor(ReasonInvalidRequest).withFields(bad), Step{
			Input: in, Output: []string{"invalid_fields=" + strings.Join(bad, ",")},
			Decision: "từ chối", Reason: "chỉ số ngoài danh sách của tool, hoặc ngày sai dạng hay không phải ngày đã qua",
		})
		r.noRead = "lượt dừng ở C3"
		return false
	}
	r.setStep(stParams, Step{Status: StepOK, Input: in, Output: []string{"valid"},
		Decision: "hợp lệ", Reason: "chỉ số thuộc danh sách của tool, ngày là ngày đã qua"})
	return true
}

// pastDay báo day là ngày YYYY-MM-DD CHÍNH XÁC (đúng hai chữ số tháng và ngày, có trên lịch) và đứng trước today. Năm 0000 bị loại: Go nhận nó
// nhưng kiểu date của PostgreSQL không có năm 0, nên truy vấn số liệu của ngày đó lỗi tất định và thành STORE_UNAVAILABLE "gửi lại cùng khoá"
// mãi. Không có cận nghiệp vụ nào khác: ngày xa mà PostgreSQL biểu diễn được đi tiếp và dừng vì thiếu số liệu.
func pastDay(day, today string) bool {
	t, err := time.Parse(time.DateOnly, day)
	return err == nil && t.Year() >= 1 && t.Format(time.DateOnly) == day && day < today
}

// ---------------------------------------------------------------------------
// C4 permission
// ---------------------------------------------------------------------------

// permission kiểm ba lớp, đóng mặc định (HOP-DONG §4.1): capability của lệnh, capability đọc hai vế, phạm vi thực thể. Máy không
// tồn tại, máy ngoài phạm vi và khoá có ký tự lạ cùng ra PERMISSION_DENIED với cùng một thông báo và cùng một vết (§4.4); lý do
// thật chỉ nằm ở reason_detail của audit.
func (r *run) permission(ctx context.Context) bool {
	d := r.p.d
	in := []string{
		"cần: " + d.Tool.Capability + ", " + CapMetricsRead + " (hai vế), phạm vi thực thể của người gọi",
		"subject_key=" + r.req.SubjectKey,
	}
	deny := func(real string) bool {
		r.refuse(stPermission, resultFor(ReasonPermissionDenied), Step{
			Input: in, Output: []string{"denied"},
			Decision: "từ chối", Reason: "không có quyền thực hiện yêu cầu này",
		}, real)
		return false
	}
	if !d.Policy.Allows(r.pr, d.Tool.Capability) {
		return deny("thiếu capability " + d.Tool.Capability)
	}
	if !d.Policy.Allows(r.pr, CapMetricsRead) {
		return deny("thiếu capability " + CapMetricsRead)
	}
	// Danh mục chỉ được đọc MỘT lần: tổ tiên tính trên chính ảnh chụp này (Lineage tự đọc danh mục nên đưa cho nó một danh mục trong bộ nhớ).
	// Số lần đọc không phụ thuộc máy có tồn tại hay không, nên độ trễ không lộ điều §4.4 giấu; và Load nạp lại xen giữa hai lần đọc không còn
	// làm phạm vi tính trên một danh mục khác với danh mục đã tìm thấy máy.
	ents, err := d.Directory.Entities(ctx)
	if err != nil {
		r.infra(stPermission, err, in, "không đọc được danh mục thực thể nên không kết luận được quyền")
		return false
	}
	i := slices.IndexFunc(ents, func(e entityresolution.Entity) bool { return e.Key == r.req.SubjectKey })
	if i < 0 {
		return deny("khoá thực thể không có trong danh mục")
	}
	lineage, err := entityresolution.Lineage(ctx, entityresolution.MemoryDirectory(ents), ents[i].ID)
	if err != nil {
		r.infra(stPermission, err, in, "không đọc được danh mục thực thể nên không kết luận được quyền")
		return false
	}
	if !d.Policy.InScope(r.pr, lineage) {
		return deny("thực thể ngoài phạm vi của người gọi")
	}
	r.ent, r.entFound = ents[i], true
	r.setStep(stPermission, Step{Status: StepOK, Input: in,
		Output:   []string{"allowed", "entity=" + r.ent.Key + " type=" + r.ent.Type},
		Decision: "cho phép", Reason: "đủ capability và thực thể nằm trong phạm vi"})
	return true
}

// ---------------------------------------------------------------------------
// C4b ledger
// ---------------------------------------------------------------------------

// ledger là nửa đọc của Idempotency, đặt ngay sau quyền (HOP-DONG §3, Q-P1-7): thấy cùng nội dung thì phát lại; thấy khác nội
// dung thì từ chối; không thấy thì đi tiếp. Trả true khi lượt đi tiếp sang C5.
func (r *run) ledger(ctx context.Context) bool { return r.consultLedger(ctx, false) }

// consultLedger tra sổ lệnh và ghi kết cục vào bước C4b. late = false là lần tra của C4b; late = true là lần tra LẠI mà R5 làm khi thấy phiếu
// OPEN (§6.3 hàng 3, I5): giữa C4b và R5 một lượt gửi trước CÙNG khoá có thể đã ghi xong, và phiếu OPEN R5 vừa thấy chính là của nó, nên bên
// gọi phải nhận phát lại (hoặc IDEMPOTENCY_KEY_REUSED nếu nội dung khác), không phải DUPLICATE_OPEN_TICKET. Mọi kết cục của việc tra sổ lệnh, kể
// cả lỗi hạ tầng, đều ở C4b. Lần tra lại không thấy dòng nào thì để nguyên bước C4b "mới": phiếu mở là của người khác. Trả true khi lượt đi tiếp.
func (r *run) consultLedger(ctx context.Context, late bool) bool {
	in := []string{"caller=" + r.caller.ID, "command_id=" + r.req.CommandID, "fingerprint=" + r.fp}
	led, found, err := r.p.d.Store.FindLedger(ctx, r.caller.ID, r.req.CommandID)
	switch {
	case err != nil:
		r.infra(stLedger, err, in, "không đọc được sổ lệnh")
		return false
	case !found:
		if !late {
			r.setStep(stLedger, Step{Status: StepOK, Input: in, Output: []string{"new"},
				Decision: "mới", Reason: "chưa có trong sổ lệnh"})
		}
		return true
	case led.Fingerprint != r.fp:
		r.refuse(stLedger, resultFor(ReasonIdempotencyKeyReused), Step{
			Input: in, Output: []string{"key_reused"}, Decision: "từ chối", Reason: "cùng command_id, khác nội dung",
		}, "cùng (người gọi, command_id) nhưng khác fingerprint")
		return false
	}
	r.replay = &led
	reason := "cùng command_id, cùng nội dung: không tính lại, không ghi"
	if late {
		reason += " (thấy ở lần tra lại của R5: lượt gửi trước cùng khoá đã ghi xong sau lần tra đầu)"
	}
	r.setStep(stLedger, Step{Status: StepOK, Input: in, Output: []string{"replay ticket=" + led.TicketID},
		Decision: "phát lại", Reason: reason})
	r.stop = stLedger
	return false
}

// fingerprint là sha256 của dạng chuẩn {subject_key, metric, day}, không gồm command_id (HOP-DONG §6.2). Ký tự NUL ngăn cách
// vì C0 đã cấm nó trong mọi trường, nên hai bộ ba khác nhau không bao giờ ra cùng một dạng chuẩn.
func fingerprint(req Request) string {
	sum := sha256.Sum256([]byte(req.SubjectKey + "\x00" + req.Metric + "\x00" + req.Day))
	return hex.EncodeToString(sum[:])
}

// ---------------------------------------------------------------------------
// C5 rules
// ---------------------------------------------------------------------------

// rules: loại thực thể hợp lệ cho chỉ số (kiểm SAU quyền để không lộ loại của máy ngoài phạm vi), tính chênh lệch bằng Derive,
// rồi R3 (đủ bằng chứng; xung đột ngang thẩm quyền thắng thiếu vế), R4 (có chênh lệch), R5 (không trùng), R6 (cờ ước lượng).
func (r *run) rules(ctx context.Context) bool {
	d := r.p.d
	in := []string{"subject_key=" + r.ent.Key, "metric=" + r.req.Metric, "day=" + r.req.Day}
	internal := func(err error, why string) bool {
		r.fail(stRules, internalResult(StatusFailed), err, Step{Input: in, Decision: "lỗi mã", Reason: why}, "chưa mở giao dịch ghi")
		return false
	}
	def, ok := d.Catalog.Derived(r.req.Metric)
	if !ok {
		return internal(fmt.Errorf("chỉ số %q không phải chỉ số dẫn xuất đã khai", r.req.Metric), "chỉ số không có trong danh mục dẫn xuất")
	}
	for _, side := range []string{def.Left, def.Right} {
		m, ok := d.Catalog.Metric(side)
		if !ok {
			return internal(fmt.Errorf("vế %q của %q chưa khai", side, def.ID), "vế của chỉ số dẫn xuất không có trong danh mục")
		}
		if !slices.Contains(m.EntityTypes, r.ent.Type) {
			r.refuse(stRules, resultFor(ReasonInvalidRequest).withFields([]string{"subject_key"}), Step{
				Input: in, Output: []string{"entity_type=" + r.ent.Type},
				Decision: "từ chối", Reason: "loại thực thể không hợp lệ cho chỉ số",
			}, fmt.Sprintf("loại thực thể %s không nằm trong entity_types của %s", r.ent.Type, side))
			return false
		}
	}
	day, err := time.ParseInLocation(time.DateOnly, r.req.Day, d.Location) // đã kiểm ở C3
	if err != nil {
		return internal(err, "ngày đã qua C3 mà không đọc được")
	}
	res, err := d.Catalog.Derive(ctx, d.Facts, r.req.Metric, []string{r.ent.ID}, day, day)
	if err != nil {
		if strings.HasPrefix(contracts.FaultOf(err), "infra:") {
			r.infra(stRules, err, in, "không đọc được số liệu")
			return false
		}
		return internal(err, "số liệu hỏng hoặc chỉ số khai sai, không tính được chênh lệch")
	}
	switch {
	case res.NeedsReview: // xung đột cần người quyết, dù vế kia có hay chưa: KHÔNG chọn số nào
		var rejected []string
		for _, c := range res.Conflicts {
			if c.NeedsReview {
				rejected = append(rejected, c.Reason)
			}
		}
		r.refuse(stRules, resultFor(ReasonHumanReviewRequired), Step{
			Input: in, Output: []string{"không chọn số nào"}, Rejected: rejected,
			Decision: "từ chối", Reason: "hai nguồn ngang thẩm quyền lệch quá dung sai ở một vế",
		}, "xung đột ngang thẩm quyền: "+strings.Join(rejected, "; "))
		return false
	case res.Total == nil:
		r.refuse(stRules, resultFor(ReasonInsufficientEvidence), Step{
			Input: in, Output: []string{"thiếu ít nhất một vế"}, Decision: "từ chối", Reason: "không có đủ số liệu có thẩm quyền cho hai vế",
		}, "thiếu số liệu có thẩm quyền ở ít nhất một vế")
		return false
	}
	// R4 đứng trước R5: khi chênh lệch không vượt ngưỡng mà còn phiếu OPEN, NO_DISCREPANCY thắng DUPLICATE_OPEN_TICKET. Hợp đồng §5 chưa đặc
	// tả thứ tự này (chỉ cố định R3: xung đột trước thiếu vế); TestR4DungTruocR5KhiCaHaiCungDung ghim lựa chọn để đổi thứ tự phải có chủ ý.
	diff := res.Total.Number
	cmp, err := decimal.Cmp(strings.TrimPrefix(diff, "-"), d.Tool.MinAbsDiff)
	if err != nil {
		return internal(err, "so sánh với ngưỡng không được")
	}
	if cmp <= 0 {
		r.refuse(stRules, resultFor(ReasonNoDiscrepancy), Step{
			Input: in, Output: []string{"diff=" + diff, "min_abs_diff=" + d.Tool.MinAbsDiff},
			Decision: "từ chối", Reason: "|chênh lệch| không vượt ngưỡng",
		}, "|chênh lệch| = "+strings.TrimPrefix(diff, "-")+" không vượt ngưỡng "+d.Tool.MinAbsDiff)
		return false
	}
	ref, found, err := d.Store.FindOpenTicket(ctx, r.ent.Key, r.req.Metric, r.req.Day)
	if err != nil {
		r.infra(stRules, err, in, "không tìm được phiếu đang mở")
		return false
	}
	if found {
		// Giữa C4b và R5 một lượt gửi trước cùng khoá có thể đã ghi xong, và phiếu OPEN vừa thấy chính là của nó: tra lại sổ lệnh trước khi
		// kết luận "người khác đã tạo" (§6.3 hàng 3, I5).
		if !r.consultLedger(ctx, true) {
			if r.replay != nil { // C6 (lệnh ca chiều 23/10): C5 ĐÃ chạy (Derive, R3, R4) trước khi R5 chuyển sang phát lại — vết ghi đúng điều đó
				r.setStep(stRules, Step{Status: StepOK, Input: in,
					Output:   []string{"diff=" + diff, fmt.Sprintf("approx=%t", res.Approx), "open_ticket=" + ref.ID},
					Decision: "phát lại",
					Reason: "Derive đã chạy, chênh lệch vượt ngưỡng; R5 thấy phiếu OPEN của chính khoá lệnh này (lượt gửi trước cùng khoá ghi xong " +
						"sau C4b) nên phát lại, không ghi"})
			}
			return false // phát lại, khoá bị dùng lại cho nội dung khác, hoặc không đọc được sổ lệnh: kết cục đã nằm ở C4b
		}
		r.refuse(stRules, duplicateResult(ref), Step{
			Input: in, Output: []string{"open_ticket=" + ref.ID}, Decision: "từ chối", Reason: "đã có phiếu OPEN cùng khoá nghiệp vụ",
		}, "đã có phiếu OPEN "+ref.ID)
		return false
	}
	left, okL := calcValue(res.Evidence, def.Left)
	right, okR := calcValue(res.Evidence, def.Right)
	if !okL || !okR {
		return internal(fmt.Errorf("kết quả Derive thiếu bằng chứng phép tính của hai vế"), "không đọc được giá trị hai vế")
	}
	snap := snapshot(res.Evidence)
	for _, e := range snap {
		r.evidence = append(r.evidence, e.ID)
	}
	r.draft = &TicketRecord{
		Status: TicketOpen, EntityID: r.ent.ID, SubjectKey: r.ent.Key, Metric: r.req.Metric, Day: r.req.Day,
		Left:  Operand{Metric: def.Left, Value: left.Number, Unit: left.Unit},
		Right: Operand{Metric: def.Right, Value: right.Number, Unit: right.Unit},
		Diff:  diff, Approx: res.Approx, Formula: calcFormula(res.Evidence, def.ID), Evidence: snap,
		CreatedBy: r.caller.ID, PackVersion: d.PackVersion, CodeVersion: d.CodeVersion, CommandID: r.req.CommandID,
	}
	r.setStep(stRules, Step{Status: StepOK, Input: in,
		Output: []string{
			fmt.Sprintf("left %s=%s %s", def.Left, left.Number, left.Unit), fmt.Sprintf("right %s=%s %s", def.Right, right.Number, right.Unit),
			"diff=" + diff, fmt.Sprintf("approx=%t", res.Approx),
		},
		Evidence: r.evidence, Decision: "đủ điều kiện tạo phiếu",
		Reason: "hai vế có số liệu có thẩm quyền, chênh lệch vượt ngưỡng, chưa có phiếu mở cùng khoá nghiệp vụ"})
	return true
}

func duplicateResult(ref TicketRef) Result {
	res := resultFor(ReasonDuplicateOpenTicket)
	status := ref.Status
	if status == "" {
		status = TicketOpen
	}
	res.Ticket = &Ticket{ID: ref.ID, Status: status}
	return res
}

// calcValue lấy giá trị đã hoà giải của một vế từ bằng chứng phép tính "calc:<chỉ số>" mà Query của mỗi vế để lại trong Derive.
func calcValue(ev []contracts.Evidence, metric string) (contracts.Value, bool) {
	for _, e := range ev {
		if e.ID == "calc:"+metric && len(e.Values) == 1 {
			return e.Values[0], true
		}
	}
	return contracts.Value{}, false
}

func calcFormula(ev []contracts.Evidence, metric string) string {
	for _, e := range ev {
		if e.ID == "calc:"+metric {
			return e.Formula
		}
	}
	return ""
}

func snapshot(ev []contracts.Evidence) []EvidenceSnap {
	out := make([]EvidenceSnap, 0, len(ev))
	for _, e := range ev {
		s := EvidenceSnap{ID: e.ID, Kind: string(e.Kind), SourceType: e.Source.SourceType, SourceRef: e.Source.Name, Day: e.Period, Authority: e.Authority}
		if len(e.Values) > 0 {
			s.Value, s.Unit = e.Values[0].Number, e.Values[0].Unit
		}
		out = append(out, s)
	}
	return out
}

// ---------------------------------------------------------------------------
// C6 write
// ---------------------------------------------------------------------------

// write gọi đơn vị ghi phiếu DUY NHẤT MỘT lần và phân loại lỗi (HOP-DONG §7.1, §9): chắc chắn chưa ghi thì FAILED; từ lúc gửi
// COMMIT thì UNKNOWN; vi phạm khoá duy nhất thì đọc lại để phân loại (§6.4); không phân loại được thì UNKNOWN, không bao giờ FAILED.
func (r *run) write(ctx context.Context) {
	rec := *r.draft
	rec.ID = r.p.d.NewID()
	u := WriteUnit{
		Ticket: rec,
		Ledger: LedgerRecord{Caller: r.caller.ID, CommandID: r.req.CommandID, Fingerprint: r.fp, TicketID: rec.ID},
		Audit:  r.auditRow(AuditExecuted, "", "", rec.ID),
	}
	r.unit = &u
	in := []string{"ticket=" + rec.ID, "command_id=" + r.req.CommandID}
	err := r.p.d.Store.WriteTicket(ctx, u)
	switch {
	case err == nil:
		r.committed = true
		r.setStep(stWrite, Step{Status: StepOK, Input: in, Output: []string{"commit"},
			Decision: "ghi phiếu", Reason: "một giao dịch: phiếu, sổ lệnh, audit EXECUTED"})
	case errors.Is(err, ErrCommitUnknown):
		// Xét TRƯỚC hai loại kia: lỗi còn mang dấu "từ lúc gửi COMMIT" thì không được báo FAILED dù nó mang thêm dấu khác (§9.2.1: nghi ngờ thì
		// UNKNOWN). Kho chỉ nên trả đúng một loại; đây là lưới đỡ khi kho trả nhiều.
		r.fail(stWrite, resultFor(ReasonCommitUnknown), err, Step{
			Input: in, Output: []string{"commit_unknown"}, Decision: "không biết", Reason: "lỗi hoặc hết hạn từ lúc gửi COMMIT",
		}, "commit chưa xác nhận")
	case errors.Is(err, ErrUniqueViolation):
		r.reclassify(ctx, err, in)
	case errors.Is(err, ErrBeforeCommit):
		r.fail(stWrite, resultFor(ReasonStoreUnavailable), err, Step{
			Input: in, Output: []string{"rollback"}, Decision: "không ghi", Reason: "lỗi trước khi gửi COMMIT, chắc chắn chưa ghi",
		}, "đã xác nhận chưa ghi (lỗi trước COMMIT)")
	default:
		r.fail(stWrite, internalResult(StatusUnknown), err, Step{
			Input: in, Decision: "không biết", Reason: "lỗi của đơn vị ghi không thuộc loại nào đã biết",
		}, "không xác nhận được rollback")
	}
}

// reclassify là §6.4: sau vi phạm khoá duy nhất giao dịch đã huỷ; pipeline KHÔNG trả lỗi thô mà đọc lại. Có dòng sổ lệnh cùng
// khoá: cùng fingerprint thì phát lại, khác thì IDEMPOTENCY_KEY_REUSED. Không có, nhưng có phiếu OPEN cùng khoá nghiệp vụ thì
// DUPLICATE_OPEN_TICKET. Không thấy gì thì INTERNAL_ERROR. Giao dịch đã huỷ chắc chắn nên mọi kết cục không phải phát lại là FAILED.
func (r *run) reclassify(ctx context.Context, cause error, in []string) {
	d := r.p.d
	readFailed := func(err error) {
		r.fail(stWrite, resultFor(ReasonStoreUnavailable), err, Step{
			Input: in, Decision: "lỗi hạ tầng", Reason: "vi phạm khoá duy nhất rồi không đọc lại để phân loại được",
		}, "đã xác nhận chưa ghi (giao dịch đã huỷ)")
	}
	led, found, err := d.Store.FindLedger(ctx, r.caller.ID, r.req.CommandID)
	if err != nil {
		readFailed(err)
		return
	}
	if found {
		if led.Fingerprint != r.fp {
			r.refuse(stWrite, resultFor(ReasonIdempotencyKeyReused), Step{
				Input: in, Output: []string{"phân loại lại: cùng khoá, khác nội dung"}, Decision: "từ chối",
				Reason: "vi phạm khoá duy nhất của sổ lệnh, đọc lại thấy cùng command_id khác nội dung",
			}, "vi phạm khoá duy nhất rồi phân loại lại: cùng (người gọi, command_id), khác fingerprint")
			return
		}
		// Phát lại là kết cục DONE nên C6 là ok (A29 (b): lượt DONE, kể cả phát lại, không có bước lệch); chỉ phân loại lại thành từ chối
		// (khoá dùng lại, trùng phiếu mở) mới là deviant.
		r.replay = &led
		r.setStep(stWrite, Step{Status: StepOK, Input: in, Output: []string{"phân loại lại: phát lại ticket=" + led.TicketID},
			Decision: "phát lại", Reason: "phát lại sau vi phạm khoá duy nhất: đọc lại thấy cùng command_id cùng nội dung"})
		r.stop = stWrite
		return
	}
	ref, found, err := d.Store.FindOpenTicket(ctx, r.ent.Key, r.req.Metric, r.req.Day)
	if err != nil {
		readFailed(err)
		return
	}
	if found {
		r.refuse(stWrite, duplicateResult(ref), Step{
			Input: in, Output: []string{"phân loại lại: open_ticket=" + ref.ID}, Decision: "từ chối",
			Reason: "vi phạm chỉ mục duy nhất của phiếu mở, đọc lại thấy phiếu OPEN cùng khoá nghiệp vụ",
		}, "vi phạm khoá duy nhất rồi phân loại lại: đã có phiếu OPEN "+ref.ID)
		return
	}
	r.fail(stWrite, internalResult(StatusFailed), cause, Step{
		Input: in, Decision: "lỗi mã", Reason: "vi phạm khoá duy nhất nhưng đọc lại không thấy sổ lệnh hay phiếu mở nào",
	}, "đã xác nhận chưa ghi (giao dịch đã huỷ)")
}

// ---------------------------------------------------------------------------
// C7 verify
// ---------------------------------------------------------------------------

// conclude là C7, luôn chạy: đọc lại khi có gì để đọc lại (đã ghi, hoặc phát lại), ghi audit riêng của lượt nếu lượt có audit,
// rồi chốt kết quả. DONE chỉ khi đọc lại thấy đúng phiếu, kể cả phát lại (I3).
func (r *run) conclude(ctx context.Context) {
	r.at = stVerify
	switch {
	case r.committed:
		r.verify(ctx, r.unit.Ticket.ID, false)
	case r.replay != nil:
		r.verify(ctx, r.replay.TicketID, true)
	default:
		out := []string{"không đọc lại: " + r.noRead}
		if r.audit != nil {
			out = append(out, r.writeAudit(ctx, *r.audit))
		}
		r.setStep(stVerify, Step{Status: StepOK, Output: out, Decision: "trả kết quả", Reason: "không đọc lại: " + r.noRead})
	}
}

func (r *run) verify(ctx context.Context, ticketID string, replayed bool) {
	in := []string{"ticket=" + ticketID, "command_id=" + r.req.CommandID}
	st, err := r.p.d.Store.ReadBack(ctx, r.caller.ID, r.req.CommandID, ticketID)
	if err != nil {
		r.cause, r.res = err, resultFor(ReasonReadbackUnavailable)
		r.setStep(stVerify, Step{Status: StepError, Fault: faultLabel(err, r.res.Fault), Error: err.Error(), Input: in,
			Decision: "chưa xác nhận", Reason: "đọc lại lỗi hoặc hết hạn"})
		return
	}
	var bad []string
	if replayed {
		bad = verifyReplay(st, *r.replay, r.req, r.caller.ID)
	} else {
		bad = verifyFresh(st, *r.unit)
	}
	if len(bad) > 0 {
		r.res = resultFor(ReasonVerifyMismatch)
		note := r.writeAudit(ctx, r.auditRow(AuditVerifyFailed, ReasonVerifyMismatch, "đọc lại không khớp: "+strings.Join(bad, ", "), ticketID))
		r.setStep(stVerify, Step{Status: StepError, Fault: r.res.Fault, Input: in, Output: []string{"mismatch=" + strings.Join(bad, ", "), note},
			Decision: "không xác nhận", Reason: "đọc lại không thấy hoặc lệch trường: cần người xem audit"})
		return
	}
	rec := st.Tickets[0]
	r.res = doneResult(rec, replayed)
	// Mã bằng chứng lấy từ chính phiếu đã đọc lại, nên phát lại (không qua C5) cũng có. Mảng MỚI, không cắt r.evidence về độ dài 0: bước C5 đang
	// trỏ vào mảng nền của nó, ghi đè sẽ đổi luôn Steps[C5].Evidence.
	r.evidence = make([]string, 0, len(rec.Evidence))
	for _, e := range rec.Evidence {
		r.evidence = append(r.evidence, e.ID)
	}
	out := []string{"verified ticket=" + rec.ID}
	if replayed {
		out = append(out, r.writeAudit(ctx, r.auditRow(AuditReplayed, "", "phát lại: cùng khoá, cùng nội dung", rec.ID)))
	} else {
		out = append(out, "audit=EXECUTED nằm trong đơn vị ghi phiếu")
	}
	r.setStep(stVerify, Step{Status: StepOK, Input: in, Output: out, Evidence: slices.Clone(r.evidence),
		Decision: string(StatusDone), Reason: "đọc lại thấy đúng phiếu"})
}

func doneResult(rec TicketRecord, replayed bool) Result {
	left, right, approx := rec.Left, rec.Right, rec.Approx
	return Result{
		Status: StatusDone, Retry: RetryNone, Replayed: replayed,
		Ticket: &Ticket{ID: rec.ID, Status: rec.Status, SubjectKey: rec.SubjectKey, Metric: rec.Metric, Day: rec.Day,
			Left: &left, Right: &right, Diff: rec.Diff, Approx: &approx},
	}
}

// verifyFresh so MỌI trường đọc lại được với bản chụp ý định dựng trong bộ nhớ trước khi ghi (không với kết quả RETURNING của
// INSERT, vì làm vậy thì đọc lại không kiểm gì). Trả tên các chỗ lệch; rỗng là khớp.
func verifyFresh(st Stored, u WriteUnit) []string {
	var bad []string
	if len(st.Tickets) != 1 {
		bad = append(bad, fmt.Sprintf("phiếu: có %d dòng", len(st.Tickets)))
	} else {
		bad = append(bad, diffTicket(u.Ticket, st.Tickets[0])...)
	}
	if len(st.Ledger) != 1 {
		bad = append(bad, fmt.Sprintf("sổ lệnh: có %d dòng", len(st.Ledger)))
	} else if st.Ledger[0] != u.Ledger {
		bad = append(bad, "sổ lệnh: lệch trường")
	}
	if !slices.ContainsFunc(st.Executed, func(a Audit) bool { return executedMatches(a, u.Audit) }) {
		bad = append(bad, "audit EXECUTED: không thấy")
	}
	return bad
}

// verifyReplay kiểm phát lại: không tính lại (Load có thể đã đổi số liệu) nên chỉ so với dòng sổ lệnh đã thấy và với chính yêu cầu.
func verifyReplay(st Stored, led LedgerRecord, req Request, caller string) []string {
	var bad []string
	if len(st.Ledger) != 1 || st.Ledger[0] != led {
		bad = append(bad, "sổ lệnh: không còn như lúc tra")
	}
	if len(st.Tickets) != 1 {
		bad = append(bad, fmt.Sprintf("phiếu: có %d dòng", len(st.Tickets)))
	} else if t := st.Tickets[0]; t.ID != led.TicketID || t.Status != TicketOpen || t.SubjectKey != req.SubjectKey || t.Metric != req.Metric ||
		t.Day != req.Day || t.CommandID != req.CommandID || t.CreatedBy != caller {
		bad = append(bad, "phiếu: lệch so với yêu cầu đã phát lại")
	}
	// Không có bản chụp ý định của lượt gốc: so các cột suy ra được từ dòng sổ lệnh, chính yêu cầu và phiếu đọc lại (phiên bản pack / mã).
	// Vai, phạm vi và trace_id là của lượt gốc (vai có thể đã đổi từ đó) nên không so ở đây.
	want := Audit{Outcome: AuditExecuted, Caller: Caller{ID: led.Caller}, CommandID: led.CommandID, Action: CommandOpenDiscrepancyTicket,
		SubjectKey: req.SubjectKey, Metric: req.Metric, Day: req.Day, TicketID: led.TicketID, Fingerprint: led.Fingerprint}
	// phiếu và dòng EXECUTED cùng một đơn vị ghi nên phiên bản pack / mã phải bằng nhau: so với phiếu vừa đọc lại khi có đúng một phiếu (số
	// phiếu khác một đã là chỗ lệch riêng ở trên) (C17S-07)
	sameVersions := func(a Audit) bool {
		return len(st.Tickets) != 1 || a.PackVersion == st.Tickets[0].PackVersion && a.CodeVersion == st.Tickets[0].CodeVersion
	}
	if !slices.ContainsFunc(st.Executed, func(a Audit) bool { return replayedExecutedMatches(a, want) && sameVersions(a) }) {
		bad = append(bad, "audit EXECUTED: không thấy")
	}
	return bad
}

// executedMatches báo a là đúng dòng audit EXECUTED của bản chụp ý định want: so MỌI cột đọc lại được (HOP-DONG §7.2; C2, lệnh ca chiều
// 23/10), trừ reason_detail — cột vai ứng dụng không được đọc (C1); dòng EXECUTED luôn ghi reason_detail rỗng.
func executedMatches(a, want Audit) bool {
	return replayedExecutedMatches(a, want) && slices.Equal(a.Caller.Roles, want.Caller.Roles) &&
		a.Caller.EntityScope == want.Caller.EntityScope && a.PackVersion == want.PackVersion && a.CodeVersion == want.CodeVersion &&
		a.TraceID == want.TraceID
}

// replayedExecutedMatches so các cột của dòng EXECUTED mà lượt phát lại suy ra được: kết cục, mã lý do, người gọi, khoá lệnh, hành động,
// khoá thực thể, chỉ số, ngày, mã phiếu, vân tay.
func replayedExecutedMatches(a, want Audit) bool {
	return a.Outcome == want.Outcome && a.ReasonCode == want.ReasonCode && a.Caller.ID == want.Caller.ID && a.CommandID == want.CommandID &&
		a.Action == want.Action && a.SubjectKey == want.SubjectKey && a.Metric == want.Metric && a.Day == want.Day &&
		a.TicketID == want.TicketID && a.Fingerprint == want.Fingerprint
}

// diffTicket trả tên các trường lệch giữa bản chụp ý định và dòng đọc lại; chỉ tên, không giá trị.
func diffTicket(w, g TicketRecord) []string {
	var bad []string
	for _, c := range []struct {
		name string
		same bool
	}{
		{"id", w.ID == g.ID}, {"status", w.Status == g.Status}, {"entity_id", w.EntityID == g.EntityID},
		{"subject_key", w.SubjectKey == g.SubjectKey}, {"metric", w.Metric == g.Metric}, {"day", w.Day == g.Day},
		{"left", w.Left == g.Left}, {"right", w.Right == g.Right}, {"diff", w.Diff == g.Diff}, {"approx", w.Approx == g.Approx},
		{"formula", w.Formula == g.Formula}, {"evidence", slices.Equal(w.Evidence, g.Evidence)},
		{"created_by", w.CreatedBy == g.CreatedBy}, {"pack_version", w.PackVersion == g.PackVersion},
		{"code_version", w.CodeVersion == g.CodeVersion}, {"command_id", w.CommandID == g.CommandID},
	} {
		if !c.same {
			bad = append(bad, "phiếu."+c.name)
		}
	}
	return bad
}

// ---------------------------------------------------------------------------
// Ghi vết thô, audit, kết quả
// ---------------------------------------------------------------------------

// setStep ghi bước st (mã và tên do bảng stages điền).
func (r *run) setStep(st int, s Step) {
	s.Code, s.Name = stages[st].code, stages[st].name
	r.steps[st], r.done[st] = s, true
}

// stopStep ghi bước dừng (deviant hoặc error, kèm nhãn fault của kết quả) và kết quả của lượt.
func (r *run) stopStep(st int, status string, res Result, s Step) {
	s.Status = status
	if s.Fault == "" { // bước lỗi có thể đã mang nhãn thật của thành phần hỏng (fail)
		s.Fault = res.Fault
	}
	r.setStep(st, s)
	r.stop, r.res = st, res
}

// refuse là từ chối theo luật. Từ C4 trở đi lượt có đúng một dòng audit REFUSED kèm lý do THẬT (real); C0, C1, C3 thì không
// (chưa có yêu cầu hợp lệ hoặc chưa có tenant tin cậy để gắn).
func (r *run) refuse(st int, res Result, s Step, real string) {
	r.stopStep(st, StepDeviant, res, s)
	r.noRead = "lượt dừng ở " + stages[st].code
	if st >= stPermission {
		a := r.auditRow(AuditRefused, res.ReasonCode, real, "")
		r.audit = &a
	}
}

// fail là dừng vì hạ tầng hoặc mã: bước là error, giữ lỗi gốc ở Cause (log) và Step.Error (vết, phải làm sạch). Không audit riêng:
// kho có thể đang hỏng, và lần gửi lại cùng khoá để lại dòng của kết cục thật (HOP-DONG §7.3).
func (r *run) fail(st int, res Result, err error, s Step, noRead string) {
	s.Error = err.Error()
	s.Fault = faultLabel(err, res.Fault)
	r.cause = err
	r.stopStep(st, StepError, res, s)
	r.noRead = noRead
}

// faultLabel là nhãn fault của một bước lỗi: nhãn thật của thành phần hỏng nếu tầng dưới đã gắn (contracts.FaultOf, ví dụ infra:directory),
// không thì nhãn của bảng §2.4. Result luôn giữ nhãn của bảng; chỉ bước vết mang nhãn thật.
func faultLabel(err error, table string) string {
	if label := contracts.FaultOf(err); label != "" {
		return label
	}
	return table
}

// infra là dừng vì hạ tầng TRƯỚC khi mở giao dịch ghi (đọc danh mục, sổ lệnh, số liệu, tìm phiếu mở lỗi hoặc hết hạn): biết
// chắc chưa ghi gì nên FAILED STORE_UNAVAILABLE, bên gọi gửi lại cùng khoá.
// Lỗi mà tầng dưới đã gắn nhãn MÃ hay DỮ LIỆU (pgstore sau 5b: ràng buộc, quyền, sai kiểu — lỗi tất định, gửi lại cũng hỏng như cũ) không phải
// lỗi hạ tầng: FAILED INTERNAL_ERROR (bảng §2.4: lỗi mã, FAILED khi trước BEGIN), như nhánh Derive. Lỗi nhãn infra:*, chưa gắn nhãn hay hết hạn
// vẫn là STORE_UNAVAILABLE (soát lại 5abc, 5b-R1).
func (r *run) infra(st int, err error, in []string, why string) {
	if f := contracts.FaultOf(err); f == contracts.FaultCode || f == contracts.FaultData {
		r.fail(st, internalResult(StatusFailed), err, Step{Input: in, Decision: "lỗi mã", Reason: why}, "chưa mở giao dịch ghi")
		return
	}
	r.fail(st, resultFor(ReasonStoreUnavailable), err, Step{Input: in, Decision: "lỗi hạ tầng", Reason: why}, "chưa mở giao dịch ghi")
}

// auditRow dựng một dòng audit của lượt; ticketID rỗng thì lấy mã phiếu của bản ghi đang ghi (nếu có).
func (r *run) auditRow(outcome AuditOutcome, code ReasonCode, detail string, ticketID ...string) Audit {
	a := Audit{
		Outcome: outcome, ReasonCode: code, ReasonDetail: detail, Caller: r.caller, CommandID: r.req.CommandID,
		Action: CommandOpenDiscrepancyTicket, SubjectKey: r.req.SubjectKey, Metric: r.req.Metric, Day: r.req.Day,
		Fingerprint: r.fp, PackVersion: r.p.d.PackVersion, CodeVersion: r.p.d.CodeVersion, TraceID: r.traceID,
	}
	if len(ticketID) > 0 {
		a.TicketID = ticketID[0]
	}
	return a
}

// auditTimeout là hạn riêng của một lần ghi audit ngoài đơn vị ghi phiếu (chặn treo khi kho không trả lời).
const auditTimeout = 5 * time.Second

// writeAudit ghi một dòng audit bằng giao dịch riêng, cố gắng tối đa: hỏng thì kết quả KHÔNG đổi, chỉ để lại ghi chú ở vết
// (nhãn infra:postgres) và lỗi gốc ở Cause. Ghi bằng ngữ cảnh KHÔNG huỷ theo bên gọi, có hạn riêng: quyết định của lượt đã có, bên gọi
// ngắt kết nối không được xoá dấu kiểm toán của nó (soát K3, K3S-01: 300 lượt bị từ chối rồi ngắt RST chỉ còn 2 dòng REFUSED). Tenant vẫn
// đi theo ngữ cảnh (WithoutCancel giữ các giá trị).
func (r *run) writeAudit(ctx context.Context, a Audit) string {
	actx, cancel := context.WithTimeout(context.WithoutCancel(ctx), auditTimeout)
	defer cancel()
	if err := r.p.d.Store.WriteAudit(actx, a); err != nil {
		if r.cause == nil {
			r.cause = err
		}
		return fmt.Sprintf("audit=%s không ghi được (%s)", a.Outcome, contracts.Infra("postgres"))
	}
	return fmt.Sprintf("audit=%s đã ghi", a.Outcome)
}

// outcome chốt lượt: điền bước skipped cho chặng chưa chạy rồi đóng gói.
func (r *run) outcome() Outcome {
	// chụp TRƯỚC vòng lặp: vòng lặp tự đặt done[] cho bước nó vừa điền skipped, nên đọc r.done[stRules] trong vòng sẽ coi nhầm mọi phát lại
	// thường là phát lại do R5 (soát độc lập C1–C7, C17S-01)
	rulesRan := r.done[stRules]
	for i := range r.steps {
		if r.done[i] {
			continue
		}
		reason := "lượt dừng ở " + stages[max(r.stop, 0)].code
		switch {
		case r.replay != nil && r.stop == stLedger && rulesRan: // phát lại do R5 (C6): C5 đã tính, chỉ không ghi
			reason = "phát lại (R5 thấy phiếu của chính khoá lệnh này): không ghi"
		case r.replay != nil && r.stop == stLedger:
			reason = "phát lại ở C4b: không tính lại, không ghi"
		}
		r.setStep(i, Step{Status: StepSkipped, Reason: reason})
	}
	res := r.res
	res.TraceID = r.traceID
	out := Outcome{Result: res, Steps: r.steps[:], Question: r.question, Evidence: r.evidence, Cause: r.cause}
	if r.entFound {
		out.Entity = r.ent.Key
	}
	return out
}

// question là dạng chuẩn của yêu cầu đã làm sạch (HOP-DONG §7.5).
func question(req Request) string {
	return CommandOpenDiscrepancyTicket + " subject_key=" + echo(req.SubjectKey) + " metric=" + echo(req.Metric) +
		" day=" + echo(req.Day) + " command_id=" + echo(req.CommandID)
}

// echo là dạng một trường thô của yêu cầu ghi vào vết: làm sạch như clean, rồi cắt ở maxSubjectKeyBytes byte (trên ranh giới ký tự) kèm
// số byte gốc, để trường quá dài bị C0 từ chối không mang nguyên giá trị vào câu hỏi chuẩn và đầu vào của bước (C5).
func echo(s string) string {
	c := clean(s)
	if len(c) <= maxSubjectKeyBytes {
		return c
	}
	cut := maxSubjectKeyBytes
	for cut > 0 && !utf8.RuneStart(c[cut]) {
		cut--
	}
	return c[:cut] + fmt.Sprintf("…(%d byte)", len(s))
}

// clean làm một chuỗi lấy từ yêu cầu lưu được vào dòng vết: đổi ký tự NUL thành chữ và byte UTF-8 hỏng thành ký tự thay thế, vì kho vết
// (PostgreSQL) không nhận cả hai. Lượt bị C0 từ chối vì đúng hai lý do đó vẫn phải có vết.
func clean(s string) string {
	return strings.ToValidUTF8(strings.ReplaceAll(s, "\x00", "<NUL>"), string(utf8.RuneError))
}
