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
	// Sender, SentAt: người gửi và thời điểm gửi của nguồn chữ tự do (tin nhắn); rỗng / zero khi nguồn không có.
	Sender string
	SentAt time.Time
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

// entityRecord là một phần tử của một mảng trong entities_file (EntityList); trường mang id cha đọc riêng theo khai của pack.
type entityRecord struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Aliases   []string          `json:"aliases"`
	SourceIDs map[string]string `json:"source_ids"`
}

// index là dữ liệu gốc của tenant: danh mục thực thể + bảng tra mã theo nguồn.
type index struct {
	base      entityresolution.MemoryDirectory
	sourceIDs map[string]map[string]string // namespace → Key(mã) → id thực thể
	codes     map[string]map[string]string // id thực thể → namespace → mã (nguyên văn)
}

// loadIndex đọc entities_file theo lược đồ của pack: mảng nào là loại thực thể nào, trường nào trỏ cha. Khoá không khai thì bỏ qua.
func loadIndex(path string, lists []EntityList) (*index, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f map[string]json.RawMessage
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	ix := &index{sourceIDs: map[string]map[string]string{}, codes: map[string]map[string]string{}}
	for _, l := range lists {
		var raws []json.RawMessage
		if f[l.List] != nil {
			if err := json.Unmarshal(f[l.List], &raws); err != nil {
				return nil, fmt.Errorf("%s: %s: %w", path, l.List, err)
			}
		}
		for _, raw := range raws {
			var r entityRecord
			if err := json.Unmarshal(raw, &r); err != nil {
				return nil, fmt.Errorf("%s: %s: %w", path, l.List, err)
			}
			e := entityresolution.Entity{ID: r.ID, Type: l.Type, Key: r.ID, Name: r.Name}
			if l.ParentField != "" {
				var fields map[string]json.RawMessage
				if err := json.Unmarshal(raw, &fields); err != nil {
					return nil, fmt.Errorf("%s: %s: %w", path, l.List, err)
				}
				if p := fields[l.ParentField]; p != nil {
					if err := json.Unmarshal(p, &e.Parent); err != nil {
						return nil, fmt.Errorf("%s: %s %s.%s: %w", path, l.List, r.ID, l.ParentField, err)
					}
				}
			}
			for _, a := range r.Aliases {
				e.Aliases = append(e.Aliases, entityresolution.Alias{Text: a, Source: "master"})
			}
			ix.base = append(ix.base, e)
			if len(r.SourceIDs) > 0 {
				ix.codes[r.ID] = r.SourceIDs
			}
			for ns, code := range r.SourceIDs {
				if ix.sourceIDs[ns] == nil {
					ix.sourceIDs[ns] = map[string]string{}
				}
				ix.sourceIDs[ns][entityresolution.Key(code)] = r.ID
			}
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
		if code := ix.codes[e.ID][ns]; code != "" {
			e.Aliases = append(append([]entityresolution.Alias(nil), e.Aliases...), entityresolution.Alias{Text: code, Source: ns})
		}
		out = append(out, e)
	}
	return out
}

// Ingest nạp mọi nguồn trong thư mục dữ liệu của MỘT tenant (rt). Tệp không đọc được là lỗi hạ
// tầng (không thể liệt kê bản ghi để cách ly); bản ghi hỏng thì vào cách ly.
func Ingest(ctx context.Context, dir string, rt *tenant.Runtime, sm SourceMap, x TextExtractor) (Result, error) {
	ix, err := loadIndex(filepath.Join(dir, sm.EntitiesFile), sm.EntityLists)
	if err != nil {
		return Result{}, err
	}
	var res Result
	seen := map[string]bool{} // khoá bản ghi đã gặp, chung mọi nguồn: khoá là id của fact (khoá chính facts)
	for _, s := range sm.Sources {
		def, _ := rt.Metrics.Metric(s.Metric)
		c := &conformer{ctx: ctx, rt: rt, spec: s, def: def, ix: ix, dir: ix.directoryFor(s.EntityNamespace), free: sm.FreeTextEntity,
			seen: seen, last: map[string]string{}}
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
	free FreeTextEntity    // loại thực thể mà chữ tự do nhận dạng, và tên gọi của nó
	seen map[string]bool   // khoá bản ghi đã gặp trong lượt nạp
	last map[string]string // người gửi|ngày → thực thể của tin gần nhất (cho CarryEntity)
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
	// Bản sau của một khoá đã gặp không được thành fact thứ hai cùng id (cả lần nạp sẽ hỏng ở khoá chính facts, F-009); bản đầu giữ
	// nguyên kết cục của nó. Chi tiết cố định: lưu cách ly định danh theo (khoá, lý do), nên mọi bản trùng của một khoá là một dòng.
	if c.seen[key] {
		quarantine(ReasonDuplicate, "khoá bản ghi trùng một bản ghi trước đó trong cùng tệp")
		return
	}
	c.seen[key] = true
	var (
		on, at         time.Time
		entity, sender string
		value          contracts.Value
		raw            string
		note           string
	)
	switch {
	case f.Format == "pdf_lines" || c.spec.FreeText:
		raw, sender = c.field(r, "text"), c.field(r, "sender")
		d, t, err := c.freeTextDate(rel, f, r)
		if err != nil {
			quarantine(ReasonUnparseable, err.Error())
			return
		}
		on, at = d, t
		var reason, detail string
		entity, value, note, reason, detail = c.freeText(raw, sender, on)
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
		v, reason, detail := c.quantity(strings.TrimSpace(qty+" "+c.field(r, "unit")), nil, true)
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
		On: on, Value: value, Raw: raw, Note: note, Sender: sender, SentAt: at,
	})
}

// freeTextDate trả ngày của bản ghi chữ tự do, và thời điểm gửi khi nguồn có (trường datetime; zero khi không có).
func (c *conformer) freeTextDate(rel string, f FileSpec, r map[string]string) (day, at time.Time, err error) {
	if f.Format == "pdf_lines" { // báo cáo ngày: ngày nằm ở tên tệp YYYY-MM-DD.pdf
		base := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
		day, err = time.ParseInLocation("2006-01-02", base, c.rt.Location)
		return day, at, err
	}
	if s := c.field(r, "datetime"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return day, at, fmt.Errorf("thời điểm %q: %w", s, err)
		}
		l := t.In(c.rt.Location)
		return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, c.rt.Location), t, nil
	}
	day, err = time.ParseInLocation("2006-01-02", c.field(r, "date"), c.rt.Location)
	return day, at, err
}

// freeText trích thực thể (loại do pack khai) và lượng từ chữ tự do bằng CHÍNH bộ chuẩn hoá + nhận dạng của luồng hỏi.
func (c *conformer) freeText(text, sender string, on time.Time) (entity string, v contracts.Value, note, reason, detail string) {
	in := c.rt.Normalizer.Normalize(text, normalization.Options{Now: on})
	res, err := entityresolution.Resolve(c.ctx, c.dir, in, c.free.Type)
	if err != nil {
		return "", v, "", ReasonUnparseable, err.Error()
	}
	if len(res.Ambiguous) > 0 {
		return "", v, "", ReasonAmbiguousEntity, fmt.Sprintf("cụm %q khớp nhiều %s", res.Ambiguous[0].Mention, c.free.Label)
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
			return "", v, "", ReasonUnknownEntity, "tin không nhắc " + c.free.Label + " nào"
		}
		entity, note = prev, c.free.Label+" kế thừa từ tin trước của "+sender+" cùng ngày"
	default:
		return "", v, "", ReasonAmbiguousEntity, "một tin nhắc nhiều " + c.free.Label
	}
	v, reason, detail = c.quantity(text, exclude, false)
	return entity, v, note, reason, detail
}

// quantity đọc đúng MỘT lượng từ chữ (bỏ qua phần nằm trong cụm thực thể); số trần lấy đơn vị
// mặc định của nguồn; đổi về đơn vị chuẩn của chỉ số. cell: text là MỘT ô lượng của nguồn có cấu trúc (bảng, ERP, phiếu) — lượng viết nhóm nghìn
// trong ô luôn chắc chắn (normalization.Options.Cell); chữ tự do thì nhóm nghìn chưa phân định (không đứng đầu văn bản, hay chồng lên tên thực thể)
// bị cách ly, không ghi một cách đọc vào nguồn sự thật (P5; D-OP-9).
func (c *conformer) quantity(text string, exclude [][2]int, cell bool) (contracts.Value, string, string) {
	in := c.rt.Normalizer.Normalize(text, normalization.Options{Cell: cell})
	var cands []contracts.Span
	for _, sp := range in.Spans {
		if sp.Kind != contracts.ValueQuantity && sp.Kind != contracts.ValueNumber {
			continue
		}
		inside := false
		for _, x := range exclude {
			if normalization.GroupSplitsName(sp, x[0], x[1]) {
				// "máy 3 850 lít": "3" là số hiệu của tên hay phần đầu của lượng? Không loại im lặng rồi lấy số khác ("…, phiếu 912" thành 912 L)
				return contracts.Value{}, ReasonUnparseable, fmt.Sprintf("nhóm nghìn %q chồng lên tên %s %q: số đầu là số hiệu hay phần đầu của lượng", sp.Raw, c.free.Label, in.Raw[x[0]:x[1]])
			}
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
	if normalization.UncertainGroup(cands[0]) {
		// "bể 2 500 lít": 2.500 L, hay bể 2 với 500 L? Không ghi một cách đọc vào nguồn sự thật. Xét SAU đổi đơn vị: nhóm khác chiều đo
		// ("12 000 km") đã là unit_dimension — lý do đúng hơn "chưa phân định".
		return contracts.Value{}, ReasonUnparseable, fmt.Sprintf("nhóm nghìn %q chưa phân định: một lượng, hay số hiệu đứng trước một lượng", cands[0].Raw)
	}
	return v, "", ""
}

func (c *conformer) metricUnit() string {
	d, _ := c.rt.Metrics.Metric(c.spec.Metric)
	return d.Unit
}
