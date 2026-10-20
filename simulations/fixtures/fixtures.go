// Package fixtures nạp dữ liệu giả lập ĐÃ ĐÓNG BĂNG thành tệp (thực thể, fact) vào kho
// trong bộ nhớ. Dữ liệu sinh một lần rồi lưu tệp có phiên bản — bộ ca đánh giá không
// được phụ thuộc vào số ngẫu nhiên sinh lúc chạy.
//
// Đây là cửa nạp I2 rút gọn: mỗi giá trị được đổi về đơn vị chuẩn của chỉ số và kiểm miền
// giá trị như dữ liệu thật — dữ liệu giả lập không được đi đường tắt qua các cổng.
package fixtures

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
	"github.com/luannt2002/enterprise-ai-systems/core/decimal"
	"github.com/luannt2002/enterprise-ai-systems/core/tenant"
	"github.com/luannt2002/enterprise-ai-systems/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-systems/data/sourceoftruth"
	"go.yaml.in/yaml/v3"
)

type entitiesFile struct {
	Entities []struct {
		ID      string   `yaml:"id"`
		Type    string   `yaml:"type"`
		Key     string   `yaml:"key"`
		Name    string   `yaml:"name"`
		Parent  string   `yaml:"parent"`
		Aliases []string `yaml:"aliases"`
	} `yaml:"entities"`
}

type factsFile struct {
	Facts []struct {
		ID     string `yaml:"id"`
		Metric string `yaml:"metric"`
		Entity string `yaml:"entity"`
		On     string `yaml:"on"` // YYYY-MM-DD theo múi giờ tenant
		Value  string `yaml:"value"`
		Unit   string `yaml:"unit"`
		Approx bool   `yaml:"approx"`
		Source string `yaml:"source"`
		Ref    string `yaml:"ref"`
		Raw    string `yaml:"raw"`
		At     string `yaml:"at"` // RFC3339
		By     string `yaml:"by"`
	} `yaml:"facts"`
}

func decode(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := yaml.NewDecoder(strings.NewReader(string(b)))
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// LoadDirectory nạp danh mục thực thể.
func LoadDirectory(path string) (entityresolution.MemoryDirectory, error) {
	var f entitiesFile
	if err := decode(path, &f); err != nil {
		return nil, err
	}
	var dir entityresolution.MemoryDirectory
	ids := map[string]bool{}
	for _, e := range f.Entities {
		if e.ID == "" || e.Type == "" || e.Key == "" {
			return nil, fmt.Errorf("%s: thực thể thiếu id/type/key: %+v", path, e)
		}
		if ids[e.ID] {
			return nil, fmt.Errorf("%s: thực thể %s trùng id", path, e.ID)
		}
		ids[e.ID] = true
		ent := entityresolution.Entity{ID: e.ID, Type: e.Type, Key: e.Key, Name: e.Name, Parent: e.Parent}
		for _, a := range e.Aliases {
			ent.Aliases = append(ent.Aliases, entityresolution.Alias{Text: a, Source: "seed"})
		}
		dir = append(dir, ent)
	}
	for _, e := range dir {
		if e.Parent != "" && !ids[e.Parent] {
			return nil, fmt.Errorf("%s: thực thể %s trỏ cha %s không tồn tại", path, e.ID, e.Parent)
		}
	}
	return dir, nil
}

// LoadFacts nạp fact, đổi về đơn vị chuẩn của chỉ số và kiểm miền giá trị.
func LoadFacts(path string, rt *tenant.Runtime, dir entityresolution.MemoryDirectory) (sourceoftruth.MemoryStore, error) {
	var f factsFile
	if err := decode(path, &f); err != nil {
		return nil, err
	}
	known := map[string]bool{}
	for _, e := range dir {
		known[e.ID] = true
	}
	var store sourceoftruth.MemoryStore
	for _, r := range f.Facts {
		def, ok := rt.Metrics.Metric(r.Metric)
		if !ok {
			return nil, fmt.Errorf("fact %s: chỉ số %s chưa khai báo", r.ID, r.Metric)
		}
		if !known[r.Entity] {
			return nil, fmt.Errorf("fact %s: thực thể %s không có trong danh mục", r.ID, r.Entity)
		}
		on, err := time.ParseInLocation("2006-01-02", r.On, rt.Location)
		if err != nil {
			return nil, fmt.Errorf("fact %s: ngày %q: %w", r.ID, r.On, err)
		}
		at := on
		if r.At != "" {
			if at, err = time.Parse(time.RFC3339, r.At); err != nil {
				return nil, fmt.Errorf("fact %s: thời điểm %q: %w", r.ID, r.At, err)
			}
		}
		v := contracts.Value{Kind: contracts.ValueQuantity, Number: r.Value, Unit: r.Unit, Approx: r.Approx}
		if v.Unit != def.Unit {
			if v, err = rt.Normalizer.Convert(v, def.Unit); err != nil {
				return nil, fmt.Errorf("fact %s: %w", r.ID, err)
			}
		}
		if err := inRange(v.Number, def); err != nil {
			return nil, fmt.Errorf("fact %s: %w", r.ID, err)
		}
		store = append(store, sourceoftruth.Fact{
			ID: r.ID, Metric: r.Metric, EntityID: r.Entity, OccurredOn: on, Value: v,
			Source: sourceoftruth.Observation{ID: "obs-" + r.ID, SourceType: r.Source, SourceRef: r.Ref, Raw: r.Raw, ObservedAt: at, ReportedBy: r.By},
		})
	}
	return store, nil
}

// inRange kiểm một lần đo nằm trong miền giá trị của chỉ số — "400000000 L dầu một ngày"
// là JSON hợp lệ nhưng vô lý về nghiệp vụ.
func inRange(n string, def sourceoftruth.MetricDef) error {
	if def.Min != "" {
		if c, err := decimal.Cmp(n, def.Min); err != nil || c < 0 {
			return fmt.Errorf("giá trị %s dưới miền %s của %s", n, def.Min, def.ID)
		}
	}
	if def.Max != "" {
		if c, err := decimal.Cmp(n, def.Max); err != nil || c > 0 {
			return fmt.Errorf("giá trị %s vượt miền %s của %s", n, def.Max, def.ID)
		}
	}
	return nil
}
