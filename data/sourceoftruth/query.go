package sourceoftruth

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
	"github.com/luannt2002/enterprise-ai-systems/core/decimal"
)

// Observation là điều một nguồn đã nói/ghi: nguyên văn, nguồn, thời điểm.
type Observation struct {
	ID         string
	SourceType string // warehouse_slip, chat_message, fuel_sensor, invoice, erp…
	SourceRef  string // mã phiếu, mã tin nhắn
	Raw        string // nguyên văn
	ObservedAt time.Time
	ReportedBy string
}

// Fact là một lần đo đã gắn chỉ số ngữ nghĩa, thực thể và ngày.
type Fact struct {
	ID         string
	Metric     string
	EntityID   string
	OccurredOn time.Time       // ngày xảy ra, 00:00 theo múi giờ tenant
	Value      contracts.Value // ĐÃ ở đơn vị chuẩn của chỉ số
	Source     Observation
}

// Store đọc fact của MỘT tenant. Cài bằng PostgreSQL khi chạy thật, bằng bộ nhớ trong test.
type Store interface {
	Facts(ctx context.Context, metric string, entityIDs []string, from, to time.Time) ([]Fact, error)
}

// MetricQuery hỏi một chỉ số trên một tập thực thể trong một khoảng ngày (gồm hai đầu).
// EntityIDs ĐÃ được lọc theo quyền của người hỏi trước khi tới đây.
type MetricQuery struct {
	Metric    string
	EntityIDs []string
	From, To  time.Time
}

// Result là kết quả một truy vấn chỉ số.
type Result struct {
	Metric string
	// Total là tổng đã hoà giải; nil khi không có dữ liệu hoặc còn xung đột chưa giải được
	// — khi đó KHÔNG được trả một con số như thể nó là sự thật.
	Total       *contracts.Value
	Approx      bool // có ước lượng ("khoảng 4k") trong số đã dùng
	NeedsReview bool // còn xung đột không tự giải được
	Evidence    []contracts.Evidence
	Conflicts   []contracts.Conflict
}

// Query trả tổng đã hoà giải của một chỉ số.
func (c *Catalog) Query(ctx context.Context, store Store, q MetricQuery) (Result, error) {
	def, ok := c.metrics[q.Metric]
	if !ok {
		return Result{}, contracts.Errorf(contracts.FaultCode, "chỉ số chưa khai báo: "+q.Metric)
	}
	facts, err := store.Facts(ctx, q.Metric, q.EntityIDs, q.From, q.To)
	if err != nil {
		return Result{}, contracts.Label(contracts.Infra("facts"), err)
	}
	res := Result{Metric: q.Metric}
	if len(facts) == 0 {
		return res, nil
	}

	// Gom theo (thực thể, ngày) rồi theo loại nguồn.
	type groupKey struct {
		entity string
		day    string
	}
	groups := map[groupKey]map[string][]Fact{}
	var keys []groupKey
	for _, f := range facts {
		k := groupKey{f.EntityID, f.OccurredOn.Format("2006-01-02")}
		if groups[k] == nil {
			groups[k] = map[string][]Fact{}
			keys = append(keys, k)
		}
		groups[k][f.Source.SourceType] = append(groups[k][f.Source.SourceType], f)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].day != keys[j].day {
			return keys[i].day < keys[j].day
		}
		return keys[i].entity < keys[j].entity
	})

	total := "0"
	var used []string
	for _, k := range keys {
		bySource := groups[k]
		type candidate struct {
			source    string
			authority int
			value     string
			ids       []string
		}
		var cands []candidate
		// Duyệt nguồn theo thứ tự cố định: thứ tự bằng chứng (và trích dẫn) phải tái lập được
		// giữa hai lần chạy — duyệt map thẳng thì mỗi lần một thứ tự (bản ghi quyết định G6 lệch).
		for _, src := range slices.Sorted(maps.Keys(bySource)) {
			fs := bySource[src]
			v, ids, approx, err := aggregate(def, fs)
			if err != nil {
				return Result{}, contracts.Label(contracts.FaultData, err)
			}
			if approx {
				res.Approx = true
			}
			cands = append(cands, candidate{src, def.authorityOf(src), v, ids})
			for _, f := range fs {
				res.Evidence = append(res.Evidence, factEvidence(def, f))
			}
		}
		sort.Slice(cands, func(i, j int) bool {
			if cands[i].authority != cands[j].authority {
				return cands[i].authority < cands[j].authority
			}
			return cands[i].source < cands[j].source
		})
		best := cands[0]
		conflictKey := q.Metric + "|" + k.entity + "|" + k.day
		unresolved := false
		for _, other := range cands[1:] {
			within, err := decimal.Within(best.value, other.value, def.Tolerance)
			if err != nil {
				return Result{}, contracts.Label(contracts.FaultData, err)
			}
			if within {
				continue
			}
			conf := contracts.Conflict{
				Key:         conflictKey,
				EvidenceIDs: append(append([]string{}, best.ids...), other.ids...),
				Policy:      "authority",
				Reason: fmt.Sprintf("%s=%s %s, %s=%s %s, lệch quá dung sai %s",
					best.source, best.value, def.Unit, other.source, other.value, def.Unit, def.Tolerance),
				Others: []contracts.SourceValue{{SourceType: other.source, Number: other.value, Unit: def.Unit, EvidenceIDs: other.ids}},
			}
			if other.authority == best.authority {
				// Ngang thẩm quyền: không có căn cứ để chọn — người quyết.
				conf.NeedsReview = true
				unresolved = true
			} else {
				// Trỏ về bằng chứng có thật của nguồn được chọn (nhiều fact thì nối bằng dấu phẩy).
				conf.ChosenID = strings.Join(best.ids, ",")
			}
			res.Conflicts = append(res.Conflicts, conf)
		}
		if unresolved {
			res.NeedsReview = true
			continue
		}
		if def.Aggregate == "last" {
			total = best.value
		} else {
			t, err := decimal.Add(total, best.value)
			if err != nil {
				return Result{}, contracts.Label(contracts.FaultData, err)
			}
			total = t
		}
		used = append(used, best.ids...)
	}

	if res.NeedsReview {
		return res, nil
	}
	v := contracts.Value{Kind: contracts.ValueQuantity, Number: total, Unit: def.Unit, Approx: res.Approx}
	res.Total = &v
	res.Evidence = append(res.Evidence, contracts.Evidence{
		ID:      "calc:" + q.Metric,
		Kind:    contracts.EvidenceCalculation,
		Text:    fmt.Sprintf("%s = %s %s", def.Label, total, def.Unit),
		Values:  []contracts.Value{v},
		Source:  contracts.SourceRef{SourceType: "calculation", Name: def.Aggregate},
		Metric:  q.Metric,
		Period:  PeriodKey(q.From, q.To),
		Formula: def.Aggregate + "(" + strings.Join(used, ", ") + ")",
	})
	return res, nil
}

// aggregate gộp các fact CÙNG loại nguồn trong một (thực thể, ngày): nhiều lần cấp trong
// ngày là nhiều sự kiện, không phải nhiều phiên bản của một số.
func aggregate(def MetricDef, fs []Fact) (value string, ids []string, approx bool, err error) {
	sort.Slice(fs, func(i, j int) bool { return fs[i].Source.ObservedAt.Before(fs[j].Source.ObservedAt) })
	value = "0"
	for _, f := range fs {
		if f.Value.Unit != def.Unit {
			return "", nil, false, fmt.Errorf("fact %s đơn vị %s, chỉ số %s đòi %s", f.ID, f.Value.Unit, def.ID, def.Unit)
		}
		if f.Value.Approx {
			approx = true
		}
		ids = append(ids, "fact:"+f.ID)
		if def.Aggregate == "last" {
			value = f.Value.Number
			continue
		}
		if value, err = decimal.Add(value, f.Value.Number); err != nil {
			return "", nil, false, err
		}
	}
	return value, ids, approx, nil
}

func factEvidence(def MetricDef, f Fact) contracts.Evidence {
	return contracts.Evidence{
		ID:   "fact:" + f.ID,
		Kind: contracts.EvidenceFact,
		Text: fmt.Sprintf("%s %s ngày %s: %s %s (nguồn %s %s)",
			def.Label, f.EntityID, f.OccurredOn.Format("02/01/2006"), f.Value.Number, f.Value.Unit, f.Source.SourceType, f.Source.SourceRef),
		Values:     []contracts.Value{f.Value},
		Source:     contracts.SourceRef{SourceID: f.Source.ID, SourceType: f.Source.SourceType, Name: f.Source.SourceRef},
		Authority:  def.authorityOf(f.Source.SourceType),
		ObservedAt: f.Source.ObservedAt,
		Confidence: 1,
		Subject:    f.EntityID,
		Metric:     def.ID,
		Period:     PeriodKey(f.OccurredOn, f.OccurredOn),
	}
}

// PeriodKey là dạng chuẩn của một kỳ trong bằng chứng và claim: "YYYY-MM-DD" cho một ngày,
// "YYYY-MM-DD/YYYY-MM-DD" cho một khoảng.
func PeriodKey(from, to time.Time) string {
	f, t := from.Format(time.DateOnly), to.Format(time.DateOnly)
	if f == t {
		return f
	}
	return f + "/" + t
}

// Derive tính một chỉ số dẫn xuất. Kết quả luôn ghi công thức và bằng chứng đầu vào.
func (c *Catalog) Derive(ctx context.Context, store Store, id string, entityIDs []string, from, to time.Time) (Result, error) {
	d, ok := c.derived[id]
	if !ok {
		return Result{}, contracts.Errorf(contracts.FaultCode, "chỉ số dẫn xuất chưa khai báo: "+id)
	}
	return c.combine(ctx, store, id, d.Label, d.Op, d.Left, d.Right, entityIDs, from, to)
}

// CompareID là mã của phép so hai chỉ số (trái − phải) — mã bằng chứng phép tính là "calc:" + mã này.
func CompareID(left, right string) string { return "compare:" + left + ":" + right }

// Compare tính hiệu hai chỉ số ĐÃ KHAI BÁO cùng đơn vị (trái − phải) trên cùng thực thể và kỳ — câu
// hỏi "đội báo nhiều hơn kho xuất bao nhiêu". Mỗi vế vẫn theo thẩm quyền trong chính nghĩa của nó.
func (c *Catalog) Compare(ctx context.Context, store Store, left, right string, entityIDs []string, from, to time.Time) (Result, error) {
	l, okL := c.metrics[left]
	r, okR := c.metrics[right]
	switch {
	case !okL || !okR:
		return Result{}, contracts.Errorf(contracts.FaultCode, "so hai chỉ số chưa khai báo: "+left+", "+right)
	case l.Unit != r.Unit:
		return Result{}, contracts.Errorf(contracts.FaultCode, "không so được hai chỉ số khác đơn vị: "+left+", "+right)
	}
	return c.combine(ctx, store, CompareID(left, right), l.Label+" − "+r.Label, "subtract", left, right, entityIDs, from, to)
}

// combine tính trái (op) phải trên hai truy vấn chỉ số, ghi bằng chứng phép tính có công thức.
func (c *Catalog) combine(ctx context.Context, store Store, id, label, opName, leftID, rightID string, entityIDs []string, from, to time.Time) (Result, error) {
	left, err := c.Query(ctx, store, MetricQuery{Metric: leftID, EntityIDs: entityIDs, From: from, To: to})
	if err != nil {
		return Result{}, err
	}
	right, err := c.Query(ctx, store, MetricQuery{Metric: rightID, EntityIDs: entityIDs, From: from, To: to})
	if err != nil {
		return Result{}, err
	}
	res := Result{
		Metric:      id,
		Approx:      left.Approx || right.Approx,
		NeedsReview: left.NeedsReview || right.NeedsReview,
		Evidence:    append(append([]contracts.Evidence{}, left.Evidence...), right.Evidence...),
		Conflicts:   append(append([]contracts.Conflict{}, left.Conflicts...), right.Conflicts...),
	}
	if res.NeedsReview || left.Total == nil || right.Total == nil {
		return res, nil // thiếu một vế thì không có chênh lệch nào để báo
	}
	op, sym := decimal.Sub, "−"
	if opName == "add" {
		op, sym = decimal.Add, "+"
	}
	n, err := op(left.Total.Number, right.Total.Number)
	if err != nil {
		return Result{}, contracts.Label(contracts.FaultData, err)
	}
	v := contracts.Value{Kind: contracts.ValueQuantity, Number: n, Unit: left.Total.Unit, Approx: res.Approx}
	res.Total = &v
	res.Evidence = append(res.Evidence, contracts.Evidence{
		ID:      "calc:" + id,
		Kind:    contracts.EvidenceCalculation,
		Text:    fmt.Sprintf("%s = %s %s %s = %s %s", label, left.Total.Number, sym, right.Total.Number, n, v.Unit),
		Values:  []contracts.Value{v},
		Source:  contracts.SourceRef{SourceType: "calculation", Name: opName},
		Formula: fmt.Sprintf("calc:%s %s calc:%s", leftID, sym, rightID),
		Metric:  id,
		Period:  PeriodKey(from, to),
	})
	return res, nil
}

// MemoryStore là Store trong bộ nhớ — cho test và dữ liệu giả lập.
type MemoryStore []Fact

// Facts lọc theo chỉ số, thực thể (rỗng = không lọc) và ngày.
func (m MemoryStore) Facts(_ context.Context, metric string, entityIDs []string, from, to time.Time) ([]Fact, error) {
	allowed := map[string]bool{}
	for _, id := range entityIDs {
		allowed[id] = true
	}
	var out []Fact
	for _, f := range m {
		if f.Metric != metric || (len(allowed) > 0 && !allowed[f.EntityID]) {
			continue
		}
		if f.OccurredOn.Before(from) || f.OccurredOn.After(to) {
			continue
		}
		out = append(out, f)
	}
	return out, nil
}
