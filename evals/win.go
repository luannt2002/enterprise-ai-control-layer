package evals

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Bộ ca WIN (docs/04-core-mvp.md §6 G6): chỉ QUERY; mỗi ca gắn lớp hỏng (§5) và người soạn.
// Mã ca mang luôn người soạn: WIN-I-### do phiên KHÔNG viết mã soạn, WIN-A-### do người viết
// mã soạn — hai tệp không bao giờ trùng mã và G6 đếm được tỉ lệ ca độc lập.

// FailureClasses là 8 lớp hỏng của docs/04 §5.
var FailureClasses = []string{"representation", "entity", "ambiguity", "semantic", "source_conflict", "permission", "unsupported_answer", "routing"}

// WinRoutes là 7 route của WIN (docs/04 §4), tên theo hợp đồng mã.
var WinRoutes = []string{"SQL_QUERY", "CALCULATION", "RAG", "HYBRID_RETRIEVAL", "CLARIFICATION", "REJECT", "HUMAN_ESCALATION"}

// Outcomes là các kết quả có kiểu (docs/04 §4). Trong WIN, xung đột giải được bằng thẩm quyền
// vẫn là RESOLVED (kèm conflict: true); ngang thẩm quyền là HUMAN_REVIEW_REQUIRED; SOURCE_CONFLICT
// để dành cho policy khác.
var Outcomes = []string{"RESOLVED", "CLARIFICATION_REQUIRED", "INSUFFICIENT_EVIDENCE", "SOURCE_CONFLICT", "PERMISSION_DENIED", "HUMAN_REVIEW_REQUIRED", "OUT_OF_SCOPE", "SYSTEM_FAILURE"}

var winID = regexp.MustCompile(`^WIN-([IA])-\d{3}$`)

// WinSuite là một tệp ca WIN.
type WinSuite struct {
	Suite         string    `yaml:"suite"`
	Tenant        string    `yaml:"tenant"`
	ReferenceDate string    `yaml:"reference_date"`
	Cases         []WinCase `yaml:"cases"`
}

// WinCase là một ca: câu hỏi của một principal và điều phải đúng ở kết quả.
type WinCase struct {
	ID           string    `yaml:"id"`
	FailureClass string    `yaml:"failure_class"`
	AuthoredBy   string    `yaml:"authored_by"` // independent | author
	Principal    string    `yaml:"principal"`   // id principal trong world.yaml
	Question     string    `yaml:"question"`
	Expect       WinExpect `yaml:"expect"`
	Note         string    `yaml:"note"`
}

// WinExpect là kỳ vọng nhị phân của một ca; trường để trống thì không kiểm.
type WinExpect struct {
	Route        string    `yaml:"route"`   // route router chọn
	Outcome      string    `yaml:"outcome"` // kết quả cuối
	Value        *WinValue `yaml:"value"`   // con số có thẩm quyền phải có trong câu trả lời
	Contains     []string  `yaml:"contains"`
	NotContains  []string  `yaml:"not_contains"`
	Candidates   []string  `yaml:"candidates"` // ứng viên khi hỏi lại
	CitedSources []string  `yaml:"cited_sources"`
	ModelCalls   *int      `yaml:"model_calls"` // số lời gọi gateway model (G4)
	Conflict     *bool     `yaml:"conflict"`    // có xung đột (nguồn thấp hơn lệch quá dung sai) được lộ ra không
}

// WinValue là một lượng ở đơn vị chuẩn.
type WinValue struct {
	Number string `yaml:"number"`
	Unit   string `yaml:"unit"`
}

// LoadWinSuite đọc một tệp ca WIN, chặn khoá lạ.
func LoadWinSuite(path string) (WinSuite, error) {
	var s WinSuite
	b, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}
	dec := yaml.NewDecoder(strings.NewReader(string(b)))
	dec.KnownFields(true)
	if err := dec.Decode(&s); err != nil {
		return s, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

// ValidateWinSuite trả mọi lỗi hợp đồng của một tệp ca WIN (rỗng = đạt). principals là tập id
// principal hợp lệ của tenant (từ world.yaml).
func ValidateWinSuite(s WinSuite, tenant string, principals map[string]bool) []string {
	var out []string
	if s.Suite != "win" {
		out = append(out, fmt.Sprintf("suite = %q, muốn win", s.Suite))
	}
	if s.Tenant != tenant {
		out = append(out, fmt.Sprintf("tenant = %q, muốn %q", s.Tenant, tenant))
	}
	classes, routes, outcomes := setOf(FailureClasses...), setOf(WinRoutes...), setOf(Outcomes...)
	seen := map[string]bool{}
	for i, c := range s.Cases {
		bad := func(f string, a ...any) {
			out = append(out, fmt.Sprintf("ca #%d (%s): %s", i+1, c.ID, fmt.Sprintf(f, a...)))
		}
		m := winID.FindStringSubmatch(c.ID)
		switch {
		case m == nil:
			bad("id sai mẫu WIN-I-### / WIN-A-###")
		case seen[c.ID]:
			bad("id trùng")
		case (m[1] == "I") != (c.AuthoredBy == "independent") || (m[1] == "A") != (c.AuthoredBy == "author"):
			bad("authored_by %q không khớp tiền tố id", c.AuthoredBy)
		}
		seen[c.ID] = true
		if !classes[c.FailureClass] {
			bad("failure_class lạ %q", c.FailureClass)
		}
		if !principals[c.Principal] {
			bad("principal %q không có trong world.yaml", c.Principal)
		}
		if strings.TrimSpace(c.Question) == "" {
			bad("không có câu hỏi")
		}
		if !routes[c.Expect.Route] {
			bad("expect.route lạ %q", c.Expect.Route)
		}
		if !outcomes[c.Expect.Outcome] {
			bad("expect.outcome lạ %q", c.Expect.Outcome)
		}
		if v := c.Expect.Value; v != nil && (v.Number == "" || v.Unit == "") {
			bad("expect.value thiếu number/unit")
		}
	}
	return out
}

// WinGateCounts là phần ĐẾM của cổng G6 trên CẢ bộ ca WIN (mọi tệp gộp lại): ≥ 100 ca, mỗi lớp
// hỏng ≥ 5 ca, ≥ 40% ca do phiên không viết mã soạn, không trùng mã giữa các tệp. Trả mọi điều
// chưa đạt (rỗng = đạt).
func WinGateCounts(cases []WinCase) []string {
	var out []string
	if len(cases) < 100 {
		out = append(out, fmt.Sprintf("chỉ có %d ca, cần ≥ 100", len(cases)))
	}
	perClass, independent, seen := map[string]int{}, 0, map[string]bool{}
	for _, c := range cases {
		perClass[c.FailureClass]++
		if c.AuthoredBy == "independent" {
			independent++
		}
		if seen[c.ID] {
			out = append(out, "mã ca trùng giữa các tệp: "+c.ID)
		}
		seen[c.ID] = true
	}
	for _, fc := range FailureClasses {
		if perClass[fc] < 5 {
			out = append(out, fmt.Sprintf("lớp %s có %d ca, cần ≥ 5", fc, perClass[fc]))
		}
	}
	if len(cases) > 0 && independent*100 < 40*len(cases) {
		out = append(out, fmt.Sprintf("ca độc lập %d/%d < 40%%", independent, len(cases)))
	}
	return out
}
