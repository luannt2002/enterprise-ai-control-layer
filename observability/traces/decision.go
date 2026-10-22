package traces

import (
	"fmt"
	"slices"
	"time"
)

// Decision là bản ghi QUYẾT ĐỊNH của một lượt hỏi (G6): hệ đã quyết gì ở từng chặng, trên phiên
// bản nào. Đủ để phát lại các chặng tất định và so từng chặng. Mỗi chặng ghi ở dạng chuỗi phẳng,
// ổn định — so bằng chuỗi, không phụ thuộc cấu trúc nội bộ.
type Decision struct {
	TraceID    string            `json:"trace_id"`
	Tenant     string            `json:"tenant"`
	Principal  string            `json:"principal"`
	Kind       string            `json:"kind"`
	Roles      []string          `json:"roles"`
	Attributes map[string]string `json:"attributes,omitempty"`
	// Now là đồng hồ của lượt: "hôm nay", "hôm qua" phụ thuộc nó, phát lại phải dùng đúng mốc này.
	Now         time.Time `json:"now"`
	PackVersion string    `json:"pack_version"`
	CodeVersion string    `json:"code_version"`
	DataVersion string    `json:"data_version"`
	Question    string    `json:"question"`

	Normalized []string `json:"normalized"` // S2: "kind:raw=giá trị" mỗi đoạn
	Entities   []string `json:"entities"`   // S3: id đã nhận; cụm mơ hồ ghi "?id|id"
	Intent     string   `json:"intent"`     // S4 sau khi router gỡ mơ hồ
	Slots      []string `json:"slots"`      // S4: "tên=giá trị"
	Route      string   `json:"route"`      // S5
	Outcome    string   `json:"outcome"`
	Evidence   []string `json:"evidence"` // S6–S7: id bằng chứng được trích dẫn
	Verdict    string   `json:"verdict"`  // S9: "pass|hành động" hoặc "fail|hành động|MÃ,MÃ"
	Answer     string   `json:"answer"`   // S10

	// Why là phần GIẢI THÍCH từng chặng (R2-03): đầu vào quyết định, phương án đã xét và lý do bị loại, do luồng hỏi ghi ngay tại chỗ
	// quyết định. Chỉ để ĐỌC vết: Diff không so nó (giải thích khác mà quyết định như nhau vẫn là cùng một quyết định).
	Why Why `json:"why,omitzero"`
}

// Notes là phần giải thích của MỘT chặng, do luồng hỏi ghi tại đúng chỗ chặng đó quyết (không suy lại từ kết quả về sau). Reasons là đầu
// vào quyết định và luật đã chọn; Rejected là phương án đã xét và bị loại kèm lý do. Mỗi phần tử là một ý, viết sẵn cho người đọc vết.
type Notes struct {
	Reasons  []string `json:"reasons,omitempty"`
	Rejected []string `json:"rejected,omitempty"`
}

// Recorded báo luồng hỏi ĐÃ ghi giải thích cho chặng. Chặng đã ghi luôn có ít nhất một ý ở Reasons (kể cả khi chỉ để nói "không có gì
// để ghi"); Rejected có thể rỗng và khi đó rỗng thật: không có phương án nào bị loại. Reasons rỗng nghĩa là chưa ghi — BuildSteps hiện NotRecorded.
func (n Notes) Recorded() bool { return len(n.Reasons) > 0 }

// Reason thêm một ý vào Reasons (định dạng như fmt.Sprintf).
func (n *Notes) Reason(format string, a ...any) {
	n.Reasons = append(n.Reasons, fmt.Sprintf(format, a...))
}

// Reject thêm một phương án đã xét và bị loại vào Rejected (định dạng như fmt.Sprintf).
func (n *Notes) Reject(format string, a ...any) {
	n.Rejected = append(n.Rejected, fmt.Sprintf(format, a...))
}

// Why gom giải thích theo chặng. Chặng nào chưa có (lượt cũ, bản dựng tay, chặng chưa được móc) thì để trống: BuildSteps ghi NotRecorded
// cho chặng đó, không bịa lý do.
type Why struct {
	Normalize Notes `json:"normalize,omitzero"` // S2: luật sinh từng đoạn (Span.Rule), cách đọc khác của chỗ mơ hồ, ngày hỏng
	Resolve   Notes `json:"resolve,omitzero"`   // S3: thực thể rõ, cụm mơ hồ, ứng viên trước / sau lọc phạm vi
	Intent    Notes `json:"intent,omitzero"`    // S4: ý định khớp đủ / một phần, ý định chung, lý do hỏi lại, nguồn ứng viên
	Route     Notes `json:"route,omitzero"`     // S5: các bậc kiểm của router, route đã loại, ghi đè của luồng hỏi
	Context   Notes `json:"context,omitzero"`   // S7: bằng chứng được dùng / bị loại (thẩm quyền, quyền đọc tài liệu)
	Validate  Notes `json:"validate,omitzero"`  // S9: cổng phán quyết của câu xác nhận
}

// Diff trả tên các chặng mà hai bản ghi quyết định khác nhau (rỗng = khớp hoàn toàn).
func Diff(a, b Decision) []string {
	var out []string
	add := func(name string, same bool) {
		if !same {
			out = append(out, name)
		}
	}
	add("normalize", slices.Equal(a.Normalized, b.Normalized))
	add("resolve", slices.Equal(a.Entities, b.Entities))
	add("intent", a.Intent == b.Intent && slices.Equal(a.Slots, b.Slots))
	add("route", a.Route == b.Route)
	add("outcome", a.Outcome == b.Outcome)
	add("evidence", slices.Equal(a.Evidence, b.Evidence))
	add("verdict", a.Verdict == b.Verdict)
	add("answer", a.Answer == b.Answer)
	return out
}
