// Package evals nạp và kiểm bộ ca kiểm thử của tenant (docs/CONVENTIONS.md §5).
//
// Đây mới là viên gạch đầu của runner eval: chưa chạy ca nào qua pipeline, chỉ bảo đảm bộ ca
// là DỮ LIỆU máy đọc được và đúng hợp đồng. Bộ ca không ai kiểm thì chỉ là văn xuôi.
package evals

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
	"go.yaml.in/yaml/v3"
)

// Keys là 23 khoá bắt buộc của một ca, theo thứ tự trong CONVENTIONS §5. Khoá không áp dụng
// thì ghi null, không được xoá.
var Keys = []string{"id", "tenant", "domain", "business_process", "scenario", "entry_type", "category",
	"input", "preconditions", "sources", "expected_route", "expected_tools", "expected_context",
	"expected_output_properties", "expected_state", "must_not_do", "business_rules", "permissions",
	"failure_injection", "expected_fallback", "metrics", "severity", "acceptance_criteria"}

var (
	entryTypes = setOf("QUERY", "COMMAND", "EVENT", "WORKFLOW", "RECONCILIATION", "SECURITY")
	categories = setOf("happy_path", "dirty_data", "ambiguity", "conflict", "permission", "infrastructure", "adversarial")
	severities = setOf("S1", "S2", "S3", "S4")
	caseID     = regexp.MustCompile(`^(CON|SPT|PLT|RTE)-T-\d{3}$`)
)

func setOf(xs ...string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}

// Case là một ca đã nạp, giữ dạng khoá → giá trị để kiểm được cả khoá thiếu lẫn khoá lạ.
type Case map[string]any

func (c Case) str(key string) string { s, _ := c[key].(string); return s }

// Load đọc tệp ca: một danh sách YAML các ca.
func Load(path string) ([]Case, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cases []Case
	if err := yaml.Unmarshal(b, &cases); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cases, nil
}

// Validate trả MỌI lỗi hợp đồng của bộ ca thuộc một tenant (rỗng = đạt).
func Validate(cases []Case, tenant string) []string {
	var out []string
	bad := func(i int, c Case, format string, a ...any) {
		out = append(out, fmt.Sprintf("ca #%d (%s): %s", i+1, c.str("id"), fmt.Sprintf(format, a...)))
	}
	known := setOf(Keys...)
	seen := map[string]bool{}
	for i, c := range cases {
		var missing, unknown []string
		for _, k := range Keys {
			if _, ok := c[k]; !ok {
				missing = append(missing, k)
			}
		}
		for k := range c {
			if !known[k] {
				unknown = append(unknown, k)
			}
		}
		if len(missing) > 0 {
			bad(i, c, "thiếu khoá %s", strings.Join(missing, ", "))
		}
		if len(unknown) > 0 {
			bad(i, c, "khoá lạ %s", strings.Join(unknown, ", "))
		}
		id := c.str("id")
		switch {
		case !caseID.MatchString(id):
			bad(i, c, "id sai mẫu <T>-T-NNN")
		case seen[id]:
			bad(i, c, "id trùng")
		}
		seen[id] = true
		if c.str("tenant") != tenant {
			bad(i, c, "tenant %q, muốn %q", c.str("tenant"), tenant)
		}
		if !entryTypes[c.str("entry_type")] {
			bad(i, c, "entry_type lạ %q", c.str("entry_type"))
		}
		if !categories[c.str("category")] {
			bad(i, c, "category lạ %q", c.str("category"))
		}
		if !severities[c.str("severity")] {
			bad(i, c, "severity lạ %q", c.str("severity"))
		}
		if r, ok := c["expected_route"]; ok && r != nil && !contracts.RouteKind(c.str("expected_route")).Valid() {
			bad(i, c, "expected_route lạ %v", r)
		}
		if c["input"] == nil && c.str("entry_type") == "QUERY" {
			bad(i, c, "ca QUERY không có input")
		}
	}
	return out
}

// Categories đếm số ca theo category (để báo cáo độ phủ, không phải cổng).
func Categories(cases []Case) map[string]int {
	m := map[string]int{}
	for _, c := range cases {
		m[c.str("category")]++
	}
	return m
}
