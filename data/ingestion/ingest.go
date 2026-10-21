package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
)

// Observation là một bản ghi nguồn đã chuẩn hoá: ai (thực thể), nghĩa gì (chỉ số), nguồn nào,
// ngày nào, bao nhiêu (đơn vị chuẩn của chỉ số) — kèm nguyên văn để truy.
type Observation struct {
	Key    string // <tệp tương đối>#<record_id | số dòng> — khớp khoá manifest của bộ sinh
	Tenant string
	Entity string
	Metric string
	Source string
	On     time.Time
	Value  contracts.Value
	Raw    string
	Note   string // suy luận đã áp, ví dụ thực thể kế thừa từ tin trước
}

// Quarantine là bản ghi không thành quan sát được — luôn kèm lý do, không bao giờ rơi im lặng.
type Quarantine struct {
	Key, Tenant, Source, Reason, Detail string
}

// Result là đầu ra của một lượt nạp.
type Result struct {
	Observations []Observation
	Quarantined  []Quarantine
}

type entitiesFile struct {
	Projects []struct {
		ID      string   `json:"id"`
		Name    string   `json:"name"`
		Aliases []string `json:"aliases"`
	} `json:"projects"`
	Equipment []struct {
		ID        string            `json:"id"`
		Type      string            `json:"type"`
		Name      string            `json:"name"`
		Project   string            `json:"project"`
		Aliases   []string          `json:"aliases"`
		SourceIDs map[string]string `json:"source_ids"`
	} `json:"equipment"`
}

// index là dữ liệu gốc của tenant: danh mục thực thể + bảng tra mã theo nguồn.
type index struct {
	base      entityresolution.MemoryDirectory
	sourceIDs map[string]map[string]string // namespace → Key(mã) → id thực thể
	raw       entitiesFile
}

func loadIndex(path string) (*index, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f entitiesFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	ix := &index{sourceIDs: map[string]map[string]string{}, raw: f}
	for _, p := range f.Projects {
		e := entityresolution.Entity{ID: p.ID, Type: "project", Key: p.ID, Name: p.Name}
		for _, a := range p.Aliases {
			e.Aliases = append(e.Aliases, entityresolution.Alias{Text: a, Source: "master"})
		}
		ix.base = append(ix.base, e)
	}
	for _, q := range f.Equipment {
		e := entityresolution.Entity{ID: q.ID, Type: "equipment", Key: q.ID, Name: q.Name, Parent: q.Project}
		for _, a := range q.Aliases {
			e.Aliases = append(e.Aliases, entityresolution.Alias{Text: a, Source: "master"})
		}
		ix.base = append(ix.base, e)
		for ns, code := range q.SourceIDs {
			if ix.sourceIDs[ns] == nil {
				ix.sourceIDs[ns] = map[string]string{}
			}
			ix.sourceIDs[ns][entityresolution.Key(code)] = q.ID
		}
	}
	return ix, nil
}

// directoryFor trả danh mục dùng cho nguồn chữ tự do: bí danh chung + mã của chính nguồn đó.
func (ix *index) directoryFor(ns string) entityresolution.MemoryDirectory {
	if ns == "" {
		return ix.base
	}
	out := make(entityresolution.MemoryDirectory, 0, len(ix.base))
	for _, e := range ix.base {
		for _, q := range ix.raw.Equipment {
			if q.ID == e.ID && q.SourceIDs[ns] != "" {
				e.Aliases = append(append([]entityresolution.Alias(nil), e.Aliases...),
					entityresolution.Alias{Text: q.SourceIDs[ns], Source: ns})
			}
		}
		out = append(out, e)
	}
	return out
}

// Ingest nạp mọi nguồn trong thư mục dữ liệu của MỘT tenant (rt). Tệp không đọc được là lỗi hạ
// tầng (không thể liệt kê bản ghi để cách ly); bản ghi hỏng thì vào cách ly.
func Ingest(ctx context.Context, dir string, rt *tenant.Runtime, sm SourceMap, x TextExtractor) (Result, error) {
	ix, err := loadIndex(filepath.Join(dir, sm.EntitiesFile))
	if err != nil {
		return Result{}, err
	}
	var res Result
	for _, s := range sm.Sources {
		def, _ := rt.Metrics.Metric(s.Metric)
		c := &conformer{ctx: ctx, rt: rt, spec: s, def: def, ix: ix, dir: ix.directoryFor(s.EntityNamespace), last: map[string]string{}}
		for _, f := range s.Files {
			paths, err := filepath.Glob(filepath.Join(dir, f.Path))
			if err != nil {
				return res, err
			}
			sort.Strings(paths)
			for _, p := range paths {
				rel, _ := filepath.Rel(dir, p)
				rows, err := readRows(p, f, x)
				if err != nil {
					return res, fmt.Errorf("%s: %w", rel, err)
				}
				for _, r := range rows {
					c.add(&res, rel, f, r)
				}
			}
		}
	}
	sort.Slice(res.Observations, func(i, j int) bool { return res.Observations[i].Key < res.Observations[j].Key })
	sort.Slice(res.Quarantined, func(i, j int) bool { return res.Quarantined[i].Key < res.Quarantined[j].Key })
	return res, nil
}

type conformer struct {
	ctx  context.Context
	rt   *tenant.Runtime
	spec SourceSpec
	def  interface {
		InRange(string) error
	}
	ix   *index
	dir  entityresolution.MemoryDirectory
	last map[string]string // người gửi|ngày → thiết bị của tin gần nhất (cho CarryEntity)
}

func (c *conformer) field(r map[string]string, k string) string {
	if col := c.spec.Fields[k]; col != "" {
		return strings.TrimSpace(r[col])
	}
	return strings.TrimSpace(r[k])
}

// add biến một dòng thành quan sát hoặc cách ly.
func (c *conformer) add(res *Result, rel string, f FileSpec, r map[string]string) {
	key := rel + "#" + c.field(r, "record_id")
	quarantine := func(reason, detail string) {
		res.Quarantined = append(res.Quarantined, Quarantine{Key: key, Tenant: c.rt.Meta.Tenant, Source: c.spec.Type, Reason: reason, Detail: detail})
	}
	var (
		on     time.Time
		entity string
		value  contracts.Value
		raw    string
		note   string
	)
	switch {
	case f.Format == "pdf_lines" || c.spec.FreeText:
		raw = c.field(r, "text")
		d, err := c.freeTextDate(rel, f, r)
		if err != nil {
			quarantine(ReasonUnparseable, err.Error())
			return
		}
		on = d
		var reason, detail string
		entity, value, note, reason, detail = c.freeText(raw, c.field(r, "sender"), on)
		if reason != "" {
			quarantine(reason, detail)
			return
		}
	default:
		code := c.field(r, "entity")
		raw = strings.TrimSpace(c.field(r, "quantity") + " " + c.field(r, "unit"))
		id, ok := c.ix.sourceIDs[c.spec.EntityNamespace][entityresolution.Key(code)]
		if !ok {
			quarantine(ReasonUnknownEntity, fmt.Sprintf("mã %q không có trong %s", code, c.spec.EntityNamespace))
			return
		}
		entity = id
		d, err := time.ParseInLocation("2006-01-02", c.field(r, "date"), c.rt.Location)
		if err != nil {
			quarantine(ReasonUnparseable, "ngày "+c.field(r, "date"))
			return
		}
		on = d
		qty := c.field(r, "quantity")
		if qty == "" {
			quarantine(ReasonMissingValue, "ô lượng trống")
			return
		}
		v, reason, detail := c.quantity(strings.TrimSpace(qty+" "+c.field(r, "unit")), nil)
		if reason == ReasonMissingValue { // ô có chữ nhưng không đọc ra lượng: không phải "thiếu"
			reason = ReasonUnparseable
		}
		if reason != "" {
			quarantine(reason, detail)
			return
		}
		value = v
	}
	if err := c.def.InRange(value.Number); err != nil {
		quarantine(ReasonOutOfRange, err.Error())
		return
	}
	res.Observations = append(res.Observations, Observation{
		Key: key, Tenant: c.rt.Meta.Tenant, Entity: entity, Metric: c.spec.Metric, Source: c.spec.Type,
		On: on, Value: value, Raw: raw, Note: note,
	})
}

func (c *conformer) freeTextDate(rel string, f FileSpec, r map[string]string) (time.Time, error) {
	if f.Format == "pdf_lines" { // báo cáo ngày: ngày nằm ở tên tệp YYYY-MM-DD.pdf
		base := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
		return time.ParseInLocation("2006-01-02", base, c.rt.Location)
	}
	if s := c.field(r, "datetime"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}, fmt.Errorf("thời điểm %q: %w", s, err)
		}
		t = t.In(c.rt.Location)
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, c.rt.Location), nil
	}
	return time.ParseInLocation("2006-01-02", c.field(r, "date"), c.rt.Location)
}

// freeText trích thiết bị và lượng từ chữ tự do bằng CHÍNH bộ chuẩn hoá + nhận dạng của luồng hỏi.
func (c *conformer) freeText(text, sender string, on time.Time) (entity string, v contracts.Value, note, reason, detail string) {
	in := c.rt.Normalizer.Normalize(text, normalization.Options{Now: on})
	res, err := entityresolution.Resolve(c.ctx, c.dir, in, "equipment")
	if err != nil {
		return "", v, "", ReasonUnparseable, err.Error()
	}
	if len(res.Ambiguous) > 0 {
		return "", v, "", ReasonAmbiguousEntity, fmt.Sprintf("cụm %q khớp nhiều thiết bị", res.Ambiguous[0].Mention)
	}
	ids := map[string]bool{}
	var exclude [][2]int
	for _, e := range res.Entities {
		ids[e.ID] = true
		exclude = append(exclude, [2]int{e.Start, e.End}) // "03" của "máy 03" thuộc thực thể, không phải lượng
	}
	carryKey := sender + "|" + on.Format("2006-01-02")
	switch len(ids) {
	case 1:
		for id := range ids {
			entity = id
		}
		if c.spec.CarryEntity && sender != "" {
			c.last[carryKey] = entity
		}
	case 0:
		prev, ok := c.last[carryKey]
		if !c.spec.CarryEntity || sender == "" || !ok {
			return "", v, "", ReasonUnknownEntity, "tin không nhắc thiết bị nào"
		}
		entity, note = prev, "thiết bị kế thừa từ tin trước của "+sender+" cùng ngày"
	default:
		return "", v, "", ReasonAmbiguousEntity, "một tin nhắc nhiều thiết bị"
	}
	v, reason, detail = c.quantity(text, exclude)
	return entity, v, note, reason, detail
}

// quantity đọc đúng MỘT lượng từ chữ (bỏ qua phần nằm trong cụm thực thể); số trần lấy đơn vị
// mặc định của nguồn; đổi về đơn vị chuẩn của chỉ số.
func (c *conformer) quantity(text string, exclude [][2]int) (contracts.Value, string, string) {
	in := c.rt.Normalizer.Normalize(text, normalization.Options{})
	var cands []contracts.Span
	for _, sp := range in.Spans {
		if sp.Kind != contracts.ValueQuantity && sp.Kind != contracts.ValueNumber {
			continue
		}
		inside := false
		for _, x := range exclude {
			inside = inside || (sp.Start < x[1] && sp.End > x[0])
		}
		if !inside {
			cands = append(cands, sp)
		}
	}
	switch len(cands) {
	case 0:
		return contracts.Value{}, ReasonMissingValue, "không thấy lượng trong " + fmt.Sprintf("%q", text)
	case 1:
	default:
		return contracts.Value{}, ReasonUnparseable, fmt.Sprintf("có %d lượng trong %q", len(cands), text)
	}
	v := cands[0].Value
	if v.Kind == contracts.ValueNumber || v.Unit == "" {
		if c.spec.DefaultUnit == "" {
			return contracts.Value{}, ReasonUnparseable, "số trần không có đơn vị: " + text
		}
		v.Kind, v.Unit = contracts.ValueQuantity, c.spec.DefaultUnit
	}
	unit := c.metricUnit()
	if v.Unit != unit {
		cv, err := c.rt.Normalizer.Convert(v, unit)
		if err != nil {
			return contracts.Value{}, ReasonUnitDimension, err.Error()
		}
		v = cv
	}
	return v, "", ""
}

func (c *conformer) metricUnit() string {
	d, _ := c.rt.Metrics.Metric(c.spec.Metric)
	return d.Unit
}
