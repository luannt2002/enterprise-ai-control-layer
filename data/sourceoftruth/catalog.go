// Package sourceoftruth giữ sự thật nghiệp vụ theo bốn tầng (docs/01-architecture.md §5.0):
//
//	OBSERVATION → MEASUREMENT → FACT → DERIVED FACT
//
// Hai luật trung tâm:
//  1. Thẩm quyền nguồn chỉ áp trong CÙNG một chỉ số ngữ nghĩa. "Kho xuất 3.850 L" không
//     thắng "đội trưởng báo 4.000 L" — đó là hai chỉ số khác nhau (issued_fuel và
//     reported_fuel). Chênh lệch giữa chúng là một DERIVED FACT có công thức.
//  2. Authority ≠ recency ≠ similarity ≠ số lần lặp. Hai nguồn ngang thẩm quyền mà nói
//     khác nhau thì không tự chọn — trả về cần người duyệt.
//
// Chỉ số, nguồn và thẩm quyền là DỮ LIỆU của pack (metrics.yaml); package này chỉ có luật.
package sourceoftruth

import (
	"fmt"
	"sort"
	"strings"

	"github.com/luannt2002/enterprise-ai-control-layer/core/decimal"
)

// MetricDef là một chỉ số ngữ nghĩa.
type MetricDef struct {
	ID          string   `yaml:"id"`    // reported_fuel, issued_fuel…
	Label       string   `yaml:"label"` // cách gọi trong câu trả lời
	Unit        string   `yaml:"unit"`  // đơn vị chuẩn của chỉ số (đã khai báo trong lexicon)
	Aggregate   string   `yaml:"aggregate"`
	EntityTypes []string `yaml:"entity_types"`
	// Sources: thẩm quyền của từng loại nguồn trong CHỈ SỐ NÀY (1 = mạnh nhất).
	Sources []SourceRank `yaml:"sources"`
	// Tolerance: chênh lệch tuyệt đối cho phép giữa hai nguồn cùng chỉ số, cùng thực thể,
	// cùng ngày. Vượt thì ghi xung đột. Rỗng = "0" (phải khớp tuyệt đối).
	Tolerance string `yaml:"tolerance"`
	Min       string `yaml:"min"` // miền giá trị hợp lệ của MỘT lần đo (kiểm lúc nạp)
	Max       string `yaml:"max"`
}

// SourceRank là thẩm quyền của một loại nguồn.
type SourceRank struct {
	Type      string `yaml:"type"`
	Authority int    `yaml:"authority"`
	Label     string `yaml:"label"` // cách gọi nguồn trong câu trả lời ("phiếu xuất kho")
}

// SourceLabel trả cách gọi một loại nguồn trong một chỉ số; không khai thì trả mã nguồn.
func (c *Catalog) SourceLabel(metric, source string) string {
	for _, r := range c.metrics[metric].Sources {
		if r.Type == source && r.Label != "" {
			return r.Label
		}
	}
	return source
}

// DerivedDef là một fact dẫn xuất từ hai chỉ số cùng đơn vị.
type DerivedDef struct {
	ID    string `yaml:"id"`
	Label string `yaml:"label"`
	Op    string `yaml:"op"` // subtract | add
	Left  string `yaml:"left"`
	Right string `yaml:"right"`
}

// Catalog là danh mục chỉ số đã kiểm.
type Catalog struct {
	metrics map[string]MetricDef
	derived map[string]DerivedDef
}

// Compile kiểm danh mục. unitKnown báo một đơn vị có trong lexicon của tenant không.
func Compile(metrics []MetricDef, derived []DerivedDef, unitKnown func(string) bool) (*Catalog, error) {
	var errs []string
	fail := func(f string, a ...any) { errs = append(errs, fmt.Sprintf(f, a...)) }
	c := &Catalog{metrics: map[string]MetricDef{}, derived: map[string]DerivedDef{}}

	for _, m := range metrics {
		if m.ID == "" {
			fail("chỉ số thiếu id: %+v", m)
			continue
		}
		if _, dup := c.metrics[m.ID]; dup {
			fail("chỉ số %s khai báo hai lần", m.ID)
		}
		if m.Unit == "" || !unitKnown(m.Unit) {
			fail("chỉ số %s: đơn vị %q chưa khai báo trong lexicon", m.ID, m.Unit)
		}
		if m.Aggregate == "" {
			m.Aggregate = "sum"
		}
		if m.Aggregate != "sum" && m.Aggregate != "last" {
			fail("chỉ số %s: aggregate phải là sum hoặc last, đang là %q", m.ID, m.Aggregate)
		}
		if len(m.Sources) == 0 {
			fail("chỉ số %s không có nguồn nào", m.ID)
		}
		seen := map[string]bool{}
		for _, s := range m.Sources {
			if s.Type == "" || s.Authority < 1 {
				fail("chỉ số %s: nguồn %+v thiếu type hoặc authority < 1", m.ID, s)
			}
			if seen[s.Type] {
				fail("chỉ số %s: nguồn %s khai báo hai lần", m.ID, s.Type)
			}
			seen[s.Type] = true
		}
		if m.Tolerance == "" {
			m.Tolerance = "0"
		}
		for name, v := range map[string]string{"tolerance": m.Tolerance, "min": m.Min, "max": m.Max} {
			if v == "" {
				continue
			}
			if _, ok := decimal.Parse(v); !ok {
				fail("chỉ số %s: %s %q không phải số", m.ID, name, v)
			}
		}
		c.metrics[m.ID] = m
	}

	for _, d := range derived {
		l, okL := c.metrics[d.Left]
		r, okR := c.metrics[d.Right]
		switch {
		case d.ID == "":
			fail("chỉ số dẫn xuất thiếu id: %+v", d)
		case d.Op != "subtract" && d.Op != "add":
			fail("chỉ số dẫn xuất %s: op phải là subtract hoặc add", d.ID)
		case !okL || !okR:
			fail("chỉ số dẫn xuất %s: %s hoặc %s chưa khai báo", d.ID, d.Left, d.Right)
		case l.Unit != r.Unit:
			fail("chỉ số dẫn xuất %s: %s (%s) và %s (%s) khác đơn vị", d.ID, d.Left, l.Unit, d.Right, r.Unit)
		}
		c.derived[d.ID] = d
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("metrics không hợp lệ:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return c, nil
}

// InRange kiểm một lần đo (đã ở đơn vị chuẩn của chỉ số) nằm trong miền min/max — "400000000 L
// dầu một ngày" là dữ liệu hợp lệ về kiểu nhưng vô lý về nghiệp vụ.
func (d MetricDef) InRange(n string) error {
	if d.Min != "" {
		if c, err := decimal.Cmp(n, d.Min); err != nil || c < 0 {
			return fmt.Errorf("giá trị %s dưới miền %s của %s", n, d.Min, d.ID)
		}
	}
	if d.Max != "" {
		if c, err := decimal.Cmp(n, d.Max); err != nil || c > 0 {
			return fmt.Errorf("giá trị %s vượt miền %s của %s", n, d.Max, d.ID)
		}
	}
	return nil
}

// MetricIDs trả id mọi chỉ số gốc (không gồm dẫn xuất), sắp xếp để kết quả tất định.
func (c *Catalog) MetricIDs() []string {
	ids := make([]string, 0, len(c.metrics))
	for id := range c.metrics {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Metric trả định nghĩa một chỉ số.
func (c *Catalog) Metric(id string) (MetricDef, bool) {
	m, ok := c.metrics[id]
	return m, ok
}

// Derived trả định nghĩa một chỉ số dẫn xuất.
func (c *Catalog) Derived(id string) (DerivedDef, bool) {
	d, ok := c.derived[id]
	return d, ok
}

// authorityOf trả thẩm quyền của một loại nguồn trong chỉ số; nguồn không khai báo xếp
// cuối cùng — nó không bao giờ thắng nguồn đã khai báo.
func (m MetricDef) authorityOf(sourceType string) int {
	for _, s := range m.Sources {
		if s.Type == sourceType {
			return s.Authority
		}
	}
	return 1 << 20
}
