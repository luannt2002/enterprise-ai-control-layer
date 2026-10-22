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

// FirstDifferentStep so hai bản ghi của CÙNG một câu (thường ở hai mốc mã) và trả bước đầu tiên theo thứ tự S2→S9 mà kết quả
// đã ghi khác nhau; ok = false khi khớp hoàn toàn. Dựa trên Diff, ánh xạ tên chặng sang bước: normalize→S2, resolve→S3,
// intent→S4, route→S5, outcome→S6, evidence→S7, answer→S8, verdict→S9.
//
// Hai quy ước, để "bước đầu tiên" không chỉ nhầm:
//   - câu trả lời cuối là sản phẩm của S8, nhưng S9 có thể thay nó bằng câu thay thế khi kiểm trượt. Hai bản ghi khác verdict thì
//     lỗi bắt đầu từ S9 (câu khác chỉ là hệ quả); chỉ khi verdict giữ nguyên mà câu khác thì mới tính S8. Bản nháp của S8 chưa
//     được ghi riêng nên không so được trực tiếp;
//   - hai lượt khác nhau ở chỗ lỗi (một bên lỗi, một bên không, hoặc nhãn fault khác) thì trả S? trước mọi thứ: lượt lỗi không có
//     Decision từ luồng hỏi, so các chặng còn lại sẽ chỉ ra những khác biệt giả.
func FirstDifferentStep(a, b Record) (FirstDiff, bool) {
	if a.Fault != b.Fault {
		return FirstDiff{Stage: StageUnknown, Name: "fault", A: a.Fault, B: b.Fault}, true
	}
	changed := Diff(a.Decision, b.Decision)
	pick := func(stage, name string) (FirstDiff, bool) {
		return FirstDiff{Stage: stage, Name: name, A: fieldText(a.Decision, name), B: fieldText(b.Decision, name)}, true
	}
	for _, c := range []struct{ name, stage string }{
		{"normalize", StageNormalize}, {"resolve", StageResolve}, {"intent", StageIntent},
		{"route", StageRoute}, {"outcome", StageExecute}, {"evidence", StageContext},
	} {
		if slices.Contains(changed, c.name) {
			return pick(c.stage, c.name)
		}
	}
	switch verdictChanged, answerChanged := slices.Contains(changed, "verdict"), slices.Contains(changed, "answer"); {
	case verdictChanged:
		return pick(StageValidate, "verdict")
	case answerChanged:
		return pick(StageGenerate, "answer")
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
