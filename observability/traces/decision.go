package traces

import (
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
