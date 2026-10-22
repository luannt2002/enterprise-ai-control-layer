package traces

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Record là bản ghi vết BỀN của một lượt (kể cả lượt lỗi): bản ghi quyết định phẳng (đủ để phát lại), vết từng bước, thời gian
// từng chặng và lỗi. Đây là đơn vị mà kho vết lưu và đọc lại theo trace id.
type Record struct {
	Decision      Decision `json:"decision"`
	Steps         []Step   `json:"steps"`
	Spans         []Span   `json:"spans"`
	TotalMs       int64    `json:"total_ms"`
	UnaccountedMs int64    `json:"unaccounted_ms"`
	// Fault, Error: nhãn lỗi và thông điệp ĐÃ làm sạch khi lượt lỗi; rỗng ở lượt không lỗi.
	Fault string `json:"fault,omitempty"`
	Error string `json:"error,omitempty"`
}

// NewRecord dựng bản ghi của một lượt hỏi đã chạy xong (Handle không lỗi).
func NewRecord(t Turn) Record {
	return Record{
		Decision: t.Decision, Steps: BuildSteps(t), Spans: append([]Span{}, t.Spans...),
		TotalMs: t.TotalMs, UnaccountedMs: t.UnaccountedMs,
	}
}

// Versions là ba phiên bản gắn vào bản ghi của lượt lỗi (lượt lỗi không có Decision do luồng hỏi dựng).
type Versions struct{ Pack, Code, Data string }

// NewErrorRecord dựng bản ghi của một lượt mà Handle trả lỗi. Handle trả Outcome rỗng khi lỗi — không có Decision, Spans hay
// Plan — nên bản ghi chỉ mang những gì nơi gọi biết (principal, câu hỏi, đồng hồ, phiên bản) cộng nhãn lỗi và thông điệp đã
// làm sạch, trong MỘT bước S? ghi rõ là chưa biết chặng nào lỗi. err phải khác nil. traceID do nơi gọi cấp: lượt lỗi chưa có.
func NewErrorRecord(traceID string, pr contracts.Principal, question string, now time.Time, v Versions, err error) Record {
	fault, msg := SanitizeError(err)
	d := Decision{
		TraceID: traceID, Tenant: string(pr.TenantID), Principal: pr.ID, Kind: string(pr.Kind), Roles: pr.Roles,
		Attributes: pr.Attributes, Now: now, PackVersion: v.Pack, CodeVersion: v.Code, DataVersion: v.Data, Question: question,
	}
	step := Step{
		TraceID: traceID, Stage: StageUnknown, Name: "fault", Status: StatusError, Decision: "lỗi " + fault,
		Reason:      notRecorded("Handle trả Outcome rỗng khi lỗi nên chưa biết chặng nào lỗi"),
		PackVersion: v.Pack, CodeVersion: v.Code, DataVersion: v.Data, Fault: fault, Error: msg,
	}
	return Record{Decision: d, Steps: []Step{step}, Spans: []Span{}, Fault: fault, Error: msg}
}

// Validate kiểm bản ghi trước khi lưu: có trace_id, mọi bước mang mã bước hợp lệ và đúng trace_id, thông điệp lỗi luôn đi kèm
// nhãn fault. Tenant không kiểm ở đây: nó phải khớp tenant của ngữ cảnh, việc của tầng lưu.
func (r Record) Validate() error {
	if r.Decision.TraceID == "" {
		return errors.New("thiếu trace_id")
	}
	if r.Error != "" && r.Fault == "" {
		return errors.New("có thông điệp lỗi mà không có nhãn fault")
	}
	for i, s := range r.Steps {
		if !ValidStage(s.Stage) {
			return fmt.Errorf("bước %d: mã bước %q không hợp lệ", i, s.Stage)
		}
		if s.TraceID != "" && s.TraceID != r.Decision.TraceID {
			return fmt.Errorf("bước %d: trace_id %q khác trace_id của bản ghi %q", i, s.TraceID, r.Decision.TraceID)
		}
	}
	return nil
}

// Digest là băm SHA-256 (hex) của dạng JSON của bản ghi. Tính trên bản ĐÃ làm sạch (SanitizeRecord) thì hai lần ghi y nguyên cho
// cùng một băm, mọi khác biệt dù nhỏ cho băm khác: kho vết dùng nó để phân biệt ghi lại y nguyên với đụng trace_id.
func (r Record) Digest() (string, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// FirstDiff là bước ĐẦU TIÊN mà hai bản ghi cho kết quả khác nhau.
type FirstDiff struct {
	Stage string // S2…S9, hoặc S? khi khác nhau ở chỗ lỗi
	Name  string // tên chặng của Diff (normalize, resolve, intent, route, outcome, evidence, answer, verdict) hoặc "fault"
	A, B  string // giá trị hai bên ở dạng chuỗi để in
}

// firstDiffOrder là thứ tự XÉT của FirstDifferentStep (tên chặng của Diff → bước). KHÔNG theo số bước: S9 đứng trước S6, S7 và S8.
// Lý do: kiểm trượt ở S9 ghi đè cả ba thứ đó — pipeline đổi outcome sang SYSTEM_FAILURE, bỏ trích dẫn và thay câu bằng mẫu thay thế
// ([CODE] core/pipeline/ask.go, nhánh "len(violations) > 0" của answerMetric và answerKnowledge) — nên khi verdict khác thì outcome,
// bằng chứng và câu khác chỉ là hệ quả. Xét outcome trước verdict thì mọi hồi quy ở S8 / S9 (model bịa nguồn, validator đổi) đều bị
// chỉ nhầm vào S6 trong khi chặng truy vấn y hệt ở hai bên (R1-01).
var firstDiffOrder = []struct{ name, stage string }{
	{"normalize", StageNormalize}, {"resolve", StageResolve}, {"intent", StageIntent}, {"route", StageRoute},
	{"verdict", StageValidate},
	{"outcome", StageExecute}, {"evidence", StageContext}, {"answer", StageGenerate},
}

// FirstDifferentStep so hai bản ghi của CÙNG một câu (thường ở hai mốc mã) và trả bước đầu tiên mà kết quả đã ghi khác nhau; ok =
// false khi khớp hoàn toàn. Dựa trên Diff, theo thứ tự xét S2, S3, S4, S5, S9, S6, S7, S8 (firstDiffOrder): normalize→S2,
// resolve→S3, intent→S4, route→S5, verdict→S9, outcome→S6, evidence→S7, answer→S8.
//
// Quy ước, để "bước đầu tiên" không chỉ nhầm:
//   - hai bản ghi khác verdict thì lỗi bắt đầu từ S9; outcome, bằng chứng, câu khác chỉ là hệ quả của kiểm trượt (xem firstDiffOrder).
//     Chỉ khi verdict giữ nguyên mà outcome khác thì mới tính S6, bằng chứng khác thì S7, câu khác thì S8. Bản nháp của S8 chưa được
//     ghi riêng nên không so được trực tiếp;
//   - giới hạn đã biết: dữ liệu truy vấn đổi mà kéo theo kiểm trượt vẫn chỉ S9 — kết quả truy vấn của S6 chưa xuất khỏi Handle (xem
//     NotRecorded) nên không phân biệt được với "S6 y hệt, model bịa". Muốn biết S6 có đổi thật, so Steps của hai bên;
//   - hai lượt khác nhau ở chỗ lỗi (một bên lỗi, một bên không, hoặc nhãn fault khác) thì trả S? trước mọi thứ: lượt lỗi không có
//     Decision từ luồng hỏi, so các chặng còn lại sẽ chỉ ra những khác biệt giả.
func FirstDifferentStep(a, b Record) (FirstDiff, bool) {
	if a.Fault != b.Fault {
		return FirstDiff{Stage: StageUnknown, Name: "fault", A: a.Fault, B: b.Fault}, true
	}
	changed := Diff(a.Decision, b.Decision)
	for _, c := range firstDiffOrder {
		if slices.Contains(changed, c.name) {
			return FirstDiff{Stage: c.stage, Name: c.name, A: fieldText(a.Decision, c.name), B: fieldText(b.Decision, c.name)}, true
		}
	}
	return FirstDiff{}, false
}

// fieldText in giá trị của chặng mà Diff gọi tên.
func fieldText(d Decision, name string) string {
	switch name {
	case "normalize":
		return strings.Join(d.Normalized, "; ")
	case "resolve":
		return strings.Join(d.Entities, "; ")
	case "intent":
		return d.Intent + " [" + strings.Join(d.Slots, "; ") + "]"
	case "route":
		return d.Route
	case "outcome":
		return d.Outcome
	case "evidence":
		return strings.Join(d.Evidence, "; ")
	case "verdict":
		return d.Verdict
	case "answer":
		return d.Answer
	}
	return ""
}
