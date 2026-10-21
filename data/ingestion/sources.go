// Package ingestion là cửa nạp bản ghi nguồn (docs/01 §5, docs/03 §5): đọc từng nguồn theo ÁNH XẠ
// NGUỒN của pack (sources.yaml), rồi biến mỗi bản ghi thành QUAN SÁT có nghĩa nghiệp vụ + nguồn
// gốc, hoặc đưa vào CÁCH LY kèm lý do — không bản ghi nào được rơi im lặng (cổng G1).
//
// Không có nhánh theo tenant: tệp nào, cột nào, nghĩa gì, tra mã ở đâu đều do pack khai báo.
package ingestion

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"go.yaml.in/yaml/v3"
)

// Lý do cách ly.
const (
	ReasonMissingValue    = "missing_value"
	ReasonUnknownEntity   = "unknown_entity"
	ReasonAmbiguousEntity = "ambiguous_entity"
	ReasonOutOfRange      = "out_of_range"
	ReasonUnitDimension   = "unit_dimension"
	ReasonUnparseable     = "unparseable"
)

// FileSpec là một tệp của một nguồn.
type FileSpec struct {
	Path   string `yaml:"path"`   // tương đối trong thư mục tenant; cho phép glob (reports/*.pdf)
	Format string `yaml:"format"` // csv | xlsx | json | jsonl | pdf_lines
	Sheet  string `yaml:"sheet"`  // chỉ cho xlsx
}

// SourceSpec là ánh xạ của một loại nguồn: đọc ở đâu, cột nào là trường chuẩn nào, mang nghĩa
// nghiệp vụ (chỉ số) nào.
type SourceSpec struct {
	Type   string     `yaml:"type"`   // loại nguồn — phải có thẩm quyền trong chỉ số Metric
	Metric string     `yaml:"metric"` // nghĩa nghiệp vụ
	Files  []FileSpec `yaml:"files"`
	// Fields: trường chuẩn → tên cột. Trường chuẩn: record_id, entity, date, datetime, quantity,
	// unit, text, sender.
	Fields      map[string]string `yaml:"fields"`
	DefaultUnit string            `yaml:"default_unit"` // đơn vị khi nguồn ghi số trần
	// EntityNamespace: nguồn có cấu trúc tra mã trong source_ids.<namespace>; nguồn chữ tự do dùng
	// các mã đó làm bí danh thêm.
	EntityNamespace string `yaml:"entity_namespace"`
	FreeText        bool   `yaml:"free_text"` // thực thể + lượng trích từ chữ
	// CarryEntity: tin không nhắc thiết bị thì kế thừa thiết bị của tin ngay trước CÙNG người gửi
	// trong CÙNG ngày — luật tất định, có ghi dấu trong quan sát.
	CarryEntity bool `yaml:"carry_entity"`
}

// SourceMap là toàn bộ ánh xạ nguồn của một pack.
type SourceMap struct {
	EntitiesFile string       `yaml:"entities_file"`
	Sources      []SourceSpec `yaml:"sources"`
}

var formats = map[string]bool{"csv": true, "xlsx": true, "json": true, "jsonl": true, "pdf_lines": true}

// LoadSources đọc sources.yaml và kiểm nó với runtime của tenant: sai thì chặn lúc khởi động.
func LoadSources(path string, rt *tenant.Runtime) (SourceMap, error) {
	var sm SourceMap
	b, err := os.ReadFile(path)
	if err != nil {
		return sm, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&sm); err != nil {
		return sm, fmt.Errorf("%s: %w", path, err)
	}
	var errs []string
	fail := func(f string, a ...any) { errs = append(errs, fmt.Sprintf(f, a...)) }
	if sm.EntitiesFile == "" {
		fail("thiếu entities_file")
	}
	for _, s := range sm.Sources {
		def, ok := rt.Metrics.Metric(s.Metric)
		if !ok {
			fail("nguồn %s: chỉ số %q chưa khai báo", s.Type, s.Metric)
			continue
		}
		ranked := false
		for _, r := range def.Sources {
			ranked = ranked || r.Type == s.Type
		}
		if !ranked {
			fail("nguồn %s chưa có thẩm quyền trong chỉ số %s", s.Type, s.Metric)
		}
		if s.DefaultUnit != "" {
			if _, ok := rt.Normalizer.Unit(s.DefaultUnit); !ok {
				fail("nguồn %s: default_unit %q chưa khai báo trong lexicon", s.Type, s.DefaultUnit)
			}
		}
		if len(s.Files) == 0 {
			fail("nguồn %s không có tệp nào", s.Type)
		}
		pdfOnly := true
		for _, f := range s.Files {
			if !formats[f.Format] {
				fail("nguồn %s: định dạng %q không hỗ trợ", s.Type, f.Format)
			}
			if f.Format == "xlsx" && f.Sheet == "" {
				fail("nguồn %s: tệp xlsx %s thiếu sheet", s.Type, f.Path)
			}
			pdfOnly = pdfOnly && f.Format == "pdf_lines"
		}
		need := []string{"record_id", "entity", "date", "quantity"}
		switch {
		case pdfOnly: // số dòng là record_id, ngày lấy từ tên tệp, chữ là cả dòng
			need = nil
		case s.FreeText:
			need = []string{"record_id", "text"}
		}
		for _, k := range need {
			if s.Fields[k] == "" {
				fail("nguồn %s thiếu ánh xạ trường %s", s.Type, k)
			}
		}
		if !s.FreeText && !pdfOnly && s.EntityNamespace == "" {
			fail("nguồn có cấu trúc %s phải khai entity_namespace", s.Type)
		}
	}
	if len(errs) > 0 {
		return sm, fmt.Errorf("%s không hợp lệ:\n  - %s", path, strings.Join(errs, "\n  - "))
	}
	return sm, nil
}
