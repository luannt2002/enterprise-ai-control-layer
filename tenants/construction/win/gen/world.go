// Package gen là bộ sinh dữ liệu WIN: đọc world.yaml rồi render ra thư mục dữ liệu đúng bố cục
// của CONTRACT.md (cạnh world.yaml), kèm manifest là đáp án cho cổng G1.
//
// Mã này KHÔNG mang tên tenant nào: tên tenant, thiết bị, ngày đều đọc từ world.yaml.
package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// dateLayout là định dạng ngày của mọi tệp (CONTRACT: YYYY-MM-DD).
const dateLayout = "2006-01-02"

// Nguồn bản ghi mà bộ sinh biết render.
var knownSources = map[string]bool{
	"warehouse_slip": true, "excel_summary": true, "erp": true, "fuel_sensor": true,
	"invoice": true, "chat_message": true, "site_report_form": true,
}

// Lý do cách ly hợp lệ (CONTRACT §3).
var knownReasons = map[string]bool{
	"missing_value": true, "unknown_entity": true, "out_of_range": true,
	"unit_dimension": true, "unparseable": true,
}

type project struct {
	ID      string   `yaml:"id" json:"id"`
	Name    string   `yaml:"name" json:"name"`
	Aliases []string `yaml:"aliases" json:"aliases"`
}

type equipment struct {
	ID        string            `yaml:"id" json:"id"`
	Type      string            `yaml:"type" json:"type"`
	Name      string            `yaml:"name" json:"name"`
	Project   string            `yaml:"project" json:"project"`
	Aliases   []string          `yaml:"aliases" json:"aliases"`
	SourceIDs map[string]string `yaml:"source_ids" json:"source_ids"`
}

type tenantDef struct {
	ID         string      `yaml:"id"`
	Projects   []project   `yaml:"projects"`
	Equipment  []equipment `yaml:"equipment"`
	Principals any         `yaml:"principals"` // bộ sinh không dùng, chỉ để bộ đọc không báo trường lạ
}

type record struct {
	ID       string `yaml:"id"`
	Tenant   string `yaml:"tenant"`
	Entity   string `yaml:"entity"`
	On       string `yaml:"on"`
	Semantic string `yaml:"semantic"`
	Source   string `yaml:"source"`
	ValueL   string `yaml:"value_l"`
	Approx   bool   `yaml:"approx"`
	Raw      string `yaml:"raw"`
	By       string `yaml:"by"`
	Ref      string `yaml:"ref"`
	Format   string `yaml:"format"`
	Expect   string `yaml:"expect"`
	Reason   string `yaml:"reason"`
}

func (r record) quarantined() bool { return r.Expect == "quarantine" }

type document struct {
	ID     string   `yaml:"id"`
	Tenant string   `yaml:"tenant"`
	Title  string   `yaml:"title"`
	ACL    []string `yaml:"acl"`
	Body   string   `yaml:"body"`
}

type world struct {
	ReferenceDate string      `yaml:"reference_date"`
	Tenants       []tenantDef `yaml:"tenants"`
	Records       []record    `yaml:"records"`
	Documents     []document  `yaml:"documents"`

	ref time.Time // reference_date đã parse
}

// loadWorld đọc và kiểm world.yaml. Trường lạ bị chặn: bộ sinh không được âm thầm bỏ sót dữ liệu.
func loadWorld(path string) (*world, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("mở world %s: %w", path, err)
	}
	defer f.Close()
	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)
	var w world
	if err := dec.Decode(&w); err != nil {
		return nil, fmt.Errorf("đọc world %s: %w", path, err)
	}
	if err := w.validate(); err != nil {
		return nil, fmt.Errorf("world %s không hợp lệ: %w", path, err)
	}
	w.nonNil()
	return &w, nil
}

// idOK: id được dùng làm tên thư mục / tên tệp nên không được chứa dấu phân cách đường dẫn.
func idOK(id string) bool {
	return id != "" && filepath.IsLocal(id) && !strings.ContainsAny(id, `/\`)
}

func (w *world) validate() error {
	var err error
	if w.ref, err = time.Parse(dateLayout, w.ReferenceDate); err != nil {
		return fmt.Errorf("reference_date %q không phải YYYY-MM-DD", w.ReferenceDate)
	}
	if len(w.Tenants) == 0 {
		return fmt.Errorf("không có tenant nào")
	}
	tenants := map[string]bool{}
	for _, t := range w.Tenants {
		if !idOK(t.ID) || tenants[t.ID] {
			return fmt.Errorf("id tenant %q rỗng, trùng hoặc có dấu phân cách đường dẫn", t.ID)
		}
		tenants[t.ID] = true
	}
	ids := map[string]bool{}
	for _, r := range w.Records {
		if ids[r.ID] || r.ID == "" {
			return fmt.Errorf("id bản ghi %q rỗng hoặc trùng", r.ID)
		}
		ids[r.ID] = true
		if !tenants[r.Tenant] {
			return fmt.Errorf("bản ghi %s: tenant %q không có trong world", r.ID, r.Tenant)
		}
		if _, err := time.Parse(dateLayout, r.On); err != nil {
			return fmt.Errorf("bản ghi %s: ngày %q không phải YYYY-MM-DD", r.ID, r.On)
		}
		if !knownSources[r.Source] {
			return fmt.Errorf("bản ghi %s: nguồn %q bộ sinh chưa biết render", r.ID, r.Source)
		}
		if r.Expect != "" && r.Expect != "observation" && r.Expect != "quarantine" {
			return fmt.Errorf("bản ghi %s: expect %q phải là observation hoặc quarantine", r.ID, r.Expect)
		}
		if r.quarantined() != knownReasons[r.Reason] {
			return fmt.Errorf("bản ghi %s: expect %q đi kèm reason %q không khớp", r.ID, r.Expect, r.Reason)
		}
		if r.Format != "" && r.Format != "xlsx" && r.Format != "pdf" {
			return fmt.Errorf("bản ghi %s: format %q chưa hỗ trợ", r.ID, r.Format)
		}
		if r.Format == "xlsx" && r.Source != "warehouse_slip" {
			return fmt.Errorf("bản ghi %s: format xlsx chỉ dành cho phiếu kho", r.ID)
		}
		if r.Format == "pdf" && r.Source != "site_report_form" {
			return fmt.Errorf("bản ghi %s: format pdf chỉ dành cho báo cáo ngày", r.ID)
		}
	}
	for _, d := range w.Documents {
		if !idOK(d.ID) || !tenants[d.Tenant] {
			return fmt.Errorf("tài liệu %q: id không hợp lệ hoặc tenant %q không có trong world", d.ID, d.Tenant)
		}
	}
	return nil
}

func (d tenantDef) find(id string) (equipment, bool) {
	for _, e := range d.Equipment {
		if e.ID == id {
			return e, true
		}
	}
	return equipment{}, false
}

// nonNil đổi slice/map nil thành rỗng để JSON ra [] và {} chứ không ra null.
func (w *world) nonNil() {
	for i := range w.Tenants {
		for j := range w.Tenants[i].Projects {
			if w.Tenants[i].Projects[j].Aliases == nil {
				w.Tenants[i].Projects[j].Aliases = []string{}
			}
		}
		for j := range w.Tenants[i].Equipment {
			e := &w.Tenants[i].Equipment[j]
			if e.Aliases == nil {
				e.Aliases = []string{}
			}
			if e.SourceIDs == nil {
				e.SourceIDs = map[string]string{}
			}
		}
	}
}

func (w *world) tenant(id string) (tenantDef, bool) {
	for _, t := range w.Tenants {
		if t.ID == id {
			return t, true
		}
	}
	return tenantDef{}, false
}

func (w *world) recordsOf(tenant string) (out []record) {
	for _, r := range w.Records {
		if r.Tenant == tenant {
			out = append(out, r)
		}
	}
	return out
}

func (w *world) docsOf(tenant string) (out []document) {
	for _, d := range w.Documents {
		if d.Tenant == tenant {
			out = append(out, d)
		}
	}
	return out
}

func (w *world) record(id string) (record, error) {
	for _, r := range w.Records {
		if r.ID == id {
			return r, nil
		}
	}
	return record{}, fmt.Errorf("world không có bản ghi %s mà hợp đồng biến thể (CONTRACT §4) cần", id)
}
