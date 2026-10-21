package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/xuri/excelize/v2"
	"go.yaml.in/yaml/v3"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/ingestion"
	"github.com/luannt2002/enterprise-ai-control-layer/evals"
	"github.com/luannt2002/enterprise-ai-control-layer/evals/baseline"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
)

// Các hệ của G7 (ADR-0005 §2). Baseline cổng: B2+ và B1 — G7 PASS cần đạt với CẢ hai. Chỉ báo cáo:
// B2 (baseline cũ), B2″ và B2-oracle.
const (
	sysEACL   = "eacl"
	sysB1     = "b1"     // LLM trực tiếp trên bản ghi thô trong cửa sổ ngày của ca
	sysB2Plus = "b2plus" // RAG đơn giản có năng lực: đoạn có dự án + bí danh + mã nguồn; ngày hiện tại
	sysB2     = "b2"     // baseline cũ: đoạn thiếu bí danh
	sysB2PP   = "b2pp"   // B2″ = B2+ cộng luật thẩm quyền / dung sai của pack viết thành chữ
	// B2-oracle = B2+ nhưng bằng chứng là đúng đoạn của bản ghi ca trỏ tới (truth.record_ids)
	sysB2Oracle = "b2oracle"
)

var sysLabel = map[string]string{sysEACL: "EACL", sysB1: "B1", sysB2Plus: "B2+", sysB2: "B2", sysB2PP: "B2″", sysB2Oracle: "B2-oracle"}

// gateSystems là các baseline cổng, theo thứ tự in.
var gateSystems = []string{sysB2Plus, sysB1}

// parseSystems đọc danh sách hệ của cờ -systems; EACL luôn có và luôn đứng đầu.
func parseSystems(s string) ([]string, error) {
	out := []string{sysEACL}
	seen := map[string]bool{sysEACL: true}
	for _, x := range strings.Split(s, ",") {
		x = strings.TrimSpace(x)
		if x == "" || seen[x] {
			continue
		}
		if _, ok := sysLabel[x]; !ok {
			return nil, fmt.Errorf("hệ %q không có; dùng: %s, %s, %s, %s, %s", x, sysB2Plus, sysB1, sysB2, sysB2PP, sysB2Oracle)
		}
		seen[x] = true
		out = append(out, x)
	}
	return out, nil
}

// todayLine là phần mở đầu cho model của B2+ / B2″ / B1: ngày hiện tại của thế giới dữ liệu. Thiếu nó
// thì "hôm qua" không trả lời được, và baseline trượt vì đồng hồ chứ không vì thiếu tầng kiểm soát.
func todayLine(ref time.Time) string {
	return fmt.Sprintf("Hôm nay là ngày %s (giờ Việt Nam).", ref.Format("02/01/2006"))
}

// entityInfo viết dữ liệu gốc của một thực thể: tên, mã chuẩn, dự án (tên + bí danh), bí danh, mã theo
// từng nguồn — đúng thứ entities.json có. Dùng cho đoạn của B2+.
func entityInfo(byID map[string]entityresolution.Entity, id string) string {
	e, ok := byID[id]
	if !ok {
		return id
	}
	var parts []string
	if p, ok := byID[e.Parent]; ok {
		parts = append(parts, fmt.Sprintf("dự án %s (%s%s)", p.Name, p.ID, aliasList(p.Aliases, "; ")))
	} else if e.Parent != "" {
		parts = append(parts, "dự án "+e.Parent)
	}
	var manual, codes []string
	for _, a := range e.Aliases {
		if a.Source == "" || a.Source == "manual" || a.Source == "master" { // bí danh của dữ liệu gốc
			manual = append(manual, a.Text)
		} else {
			codes = append(codes, a.Source+" "+a.Text)
		}
	}
	sort.Strings(codes)
	if len(manual) > 0 {
		parts = append(parts, "bí danh: "+strings.Join(manual, ", "))
	}
	if len(codes) > 0 {
		parts = append(parts, "mã theo nguồn: "+strings.Join(codes, ", "))
	}
	name := e.Name
	if name == "" {
		name = e.ID
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%s (%s)", name, e.ID)
	}
	return fmt.Sprintf("%s (%s; %s)", name, e.ID, strings.Join(parts, "; "))
}

func aliasList(as []entityresolution.Alias, sep string) string {
	var xs []string
	for _, a := range as {
		xs = append(xs, a.Text)
	}
	if len(xs) == 0 {
		return ""
	}
	return sep + "bí danh: " + strings.Join(xs, ", ")
}

// paragraphPlus là đoạn của B2+: như paragraph, nhưng thực thể được viết bằng đủ dữ liệu gốc (dự án,
// bí danh, mã theo nguồn). Không có thẩm quyền, dung sai hay luật nào (biên B2+, ADR-0005 §2).
func paragraphPlus(rt *tenant.Runtime, byID map[string]entityresolution.Entity, o ingestion.Observation) string {
	label := o.Metric
	if d, ok := rt.Metrics.Metric(o.Metric); ok && d.Label != "" {
		label = d.Label
	}
	return fmt.Sprintf("Ngày %s, %s: %s %s (%s).", o.On.Format("2006-01-02"), entityInfo(byID, o.Entity), label, o.Raw, rt.Metrics.SourceLabel(o.Metric, o.Source))
}

// domainRules viết luật thẩm quyền + dung sai của pack (metrics.yaml) thành chữ cho B2″, sinh máy móc
// từ tệp để tái lập được. Giới hạn đã biết: chưa gồm các mặc định (nghĩa của câu hỏi chung chung,
// kỳ mặc định) — chúng nằm ở intents / ADR-0002, không ở metrics.yaml.
func domainRules(packDir string) (string, error) {
	b, err := os.ReadFile(filepath.Join(packDir, "metrics.yaml"))
	if err != nil {
		return "", err
	}
	var f struct {
		Metrics []struct {
			ID        string `yaml:"id"`
			Label     string `yaml:"label"`
			Unit      string `yaml:"unit"`
			Tolerance string `yaml:"tolerance"`
			Sources   []struct {
				Type      string `yaml:"type"`
				Authority int    `yaml:"authority"`
				Label     string `yaml:"label"`
			} `yaml:"sources"`
		} `yaml:"metrics"`
	}
	if err := yaml.Unmarshal(b, &f); err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString("Luật nghiệp vụ của doanh nghiệp này:\n")
	sb.WriteString("- Mỗi chỉ số dưới đây là một thứ khác nhau; chỉ so các nguồn của CÙNG một chỉ số. Chênh lệch giữa hai chỉ số không phải xung đột nguồn.\n")
	for _, m := range f.Metrics {
		byRank := map[int][]string{}
		var ranks []int
		for _, s := range m.Sources {
			if _, ok := byRank[s.Authority]; !ok {
				ranks = append(ranks, s.Authority)
			}
			byRank[s.Authority] = append(byRank[s.Authority], s.Label)
		}
		sort.Ints(ranks)
		var order []string
		for _, r := range ranks {
			order = append(order, fmt.Sprintf("%s (hạng %d)", strings.Join(byRank[r], ", "), r))
		}
		fmt.Fprintf(&sb, "- %s (đơn vị %s): nguồn theo thẩm quyền, hạng nhỏ mạnh hơn: %s.", m.Label, m.Unit, strings.Join(order, "; "))
		if m.Tolerance != "" {
			fmt.Fprintf(&sb, " Dung sai %s %s: nguồn hạng thấp lệch nguồn hạng cao hơn quá dung sai thì vẫn dùng số của nguồn hạng cao nhất và nêu xung đột; hai nguồn cùng hạng lệch quá dung sai thì không chọn số nào, cần người duyệt.", m.Tolerance, m.Unit)
		}
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

// ---------------------------------------------------------------------------
// B1 — LLM trực tiếp trên bản ghi thô
// ---------------------------------------------------------------------------

// rawRow là một bản ghi thô của tenant: nguyên giá trị ô như nguồn ghi, không chuẩn hoá.
type rawRow struct {
	ID   string    // <tệp tương đối>#<thứ tự bản ghi trong tệp, từ 1>
	On   time.Time // ngày của bản ghi; zero = không đọc được ngày (bản ghi bẩn) → luôn đưa vào
	Text string
}

// rawSpec là bố cục một tệp nguồn theo tenants/construction/win/CONTRACT.md §1–2: cột ngày nằm ở đâu.
type rawSpec struct {
	path, format, sheet, dateField string
}

var rawSpecs = []rawSpec{
	{"warehouse.csv", "csv", "", "issue_date"},
	{"warehouse.json", "json", "", "issue_date"},
	{"warehouse.xlsx", "xlsx", "PhieuXuat", "issue_date"},
	{"excel_summary.xlsx", "xlsx", "TongHop", "date"},
	{"erp.json", "json", "", "posting_date"},
	{"sensor.csv", "csv", "", "date"},
	{"invoices.csv", "csv", "", "date"},
	{"messages.jsonl", "jsonl", "", "sent_at"},
}

// readRawRows đọc MỌI bản ghi thô của thư mục một tenant (kể cả bản ghi bẩn) + báo cáo PDF (một bản ghi
// mỗi báo cáo, ngày lấy từ tên tệp). Không import bộ nạp của EACL: B1 không được nhận dữ liệu đã chuẩn hoá.
func readRawRows(dir string, loc *time.Location, pdf ingestion.TextExtractor) ([]rawRow, error) {
	var out []rawRow
	for _, s := range rawSpecs {
		p := filepath.Join(dir, s.path)
		if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
			continue
		}
		recs, err := readTable(p, s)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", s.path, err)
		}
		for i, r := range recs {
			out = append(out, rawRow{ID: fmt.Sprintf("%s#%d", s.path, i+1), On: rawDate(r.fields[s.dateField], loc), Text: s.path + " | " + r.text})
		}
	}
	reports, _ := filepath.Glob(filepath.Join(dir, "reports", "*.pdf"))
	sort.Strings(reports)
	for _, p := range reports {
		txt, err := pdf.PDFText(p)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		day := strings.TrimSuffix(filepath.Base(p), ".pdf")
		rel := "reports/" + filepath.Base(p)
		out = append(out, rawRow{ID: rel, On: rawDate(day, loc), Text: rel + " | " + strings.Join(strings.Fields(txt), " ")})
	}
	return out, nil
}

type rawRecord struct {
	fields map[string]string
	text   string // "cột=giá trị, …" theo thứ tự cột của tệp; JSON thì theo khoá đã sắp
}

func readTable(path string, s rawSpec) ([]rawRecord, error) {
	switch s.format {
	case "csv":
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		r := csv.NewReader(f)
		r.FieldsPerRecord = -1
		rows, err := r.ReadAll()
		if err != nil {
			return nil, err
		}
		return tableRecords(rows), nil
	case "xlsx":
		f, err := excelize.OpenFile(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		rows, err := f.GetRows(s.sheet)
		if err != nil {
			return nil, err
		}
		return tableRecords(rows), nil
	case "json":
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var objs []map[string]any
		if err := json.Unmarshal(b, &objs); err != nil {
			return nil, err
		}
		var out []rawRecord
		for _, o := range objs {
			out = append(out, objectRecord(o))
		}
		return out, nil
	case "jsonl":
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		var out []rawRecord
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 4<<20)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			var o map[string]any
			if err := json.Unmarshal([]byte(line), &o); err != nil {
				out = append(out, rawRecord{fields: map[string]string{}, text: line}) // dòng hỏng: vẫn là bản ghi thô
				continue
			}
			out = append(out, objectRecord(o))
		}
		return out, sc.Err()
	}
	return nil, fmt.Errorf("định dạng %q không hỗ trợ", s.format)
}

func tableRecords(rows [][]string) []rawRecord {
	if len(rows) == 0 {
		return nil
	}
	head := rows[0]
	var out []rawRecord
	for _, row := range rows[1:] {
		rec := rawRecord{fields: map[string]string{}}
		var cells []string
		for i, h := range head {
			v := ""
			if i < len(row) {
				v = row[i]
			}
			rec.fields[strings.TrimSpace(h)] = v
			cells = append(cells, strings.TrimSpace(h)+"="+v)
		}
		rec.text = strings.Join(cells, ", ")
		out = append(out, rec)
	}
	return out
}

func objectRecord(o map[string]any) rawRecord {
	keys := make([]string, 0, len(o))
	for k := range o {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	rec := rawRecord{fields: map[string]string{}}
	var cells []string
	for _, k := range keys {
		v := fmt.Sprint(o[k])
		rec.fields[k] = v
		cells = append(cells, k+"="+v)
	}
	rec.text = strings.Join(cells, ", ")
	return rec
}

// rawDate đọc ngày của bản ghi: YYYY-MM-DD, hoặc RFC3339 (đổi về múi giờ của tenant). Không đọc được thì
// trả zero — bản ghi đó luôn được đưa vào (B1 thấy cả bản ghi bẩn, như người đọc tệp thô).
func rawDate(s string, loc *time.Location) time.Time {
	s = strings.TrimSpace(s)
	if t, err := time.ParseInLocation("2006-01-02", s, loc); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		y, m, d := t.In(loc).Date()
		return time.Date(y, m, d, 0, 0, 0, 0, loc)
	}
	return time.Time{}
}

// window là cửa sổ ngày B1 nhận cho một ca (b1_window của hồ sơ ca); nil = không có bản ghi ngày nào.
type window struct{ from, to time.Time }

// loadWindows đọc b1_window của từng ca từ hồ sơ ca (holdout-manifest.json / kho-manifest.json): mảng
// ca, hoặc object có khoá "cases". Trả ánh xạ id ca → cửa sổ; ca có b1_window null → nil.
func loadWindows(path string, loc *time.Location) (map[string]*window, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	type entry struct {
		CaseID   string `json:"case_id"`
		B1Window *struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"b1_window"`
	}
	list, err := caseEntries[entry](path, b, func(e entry) string { return e.CaseID })
	if err != nil {
		return nil, err
	}
	out := map[string]*window{}
	for _, e := range list {
		if e.B1Window == nil {
			out[e.CaseID] = nil
			continue
		}
		from, err1 := time.ParseInLocation("2006-01-02", e.B1Window.From, loc)
		to, err2 := time.ParseInLocation("2006-01-02", e.B1Window.To, loc)
		if err1 != nil || err2 != nil || to.Before(from) {
			return nil, fmt.Errorf("%s: b1_window của %s sai: %+v", path, e.CaseID, *e.B1Window)
		}
		out[e.CaseID] = &window{from: from, to: to}
	}
	return out, nil
}

// caseEntries giải mã hồ sơ ca (holdout-manifest.json / kho-manifest.json): mảng ca, hoặc object có
// khoá "cases". Mục thiếu case_id hay case_id trùng là lỗi.
func caseEntries[T any](path string, b []byte, id func(T) string) ([]T, error) {
	var list []T
	if err := json.Unmarshal(b, &list); err != nil {
		var wrapped struct {
			Cases []T `json:"cases"`
		}
		if err2 := json.Unmarshal(b, &wrapped); err2 != nil {
			return nil, fmt.Errorf("%s: không phải mảng ca hay object có \"cases\": %w", path, err)
		}
		list = wrapped.Cases
	}
	seen := map[string]bool{}
	for _, e := range list {
		switch k := id(e); {
		case k == "":
			return nil, fmt.Errorf("%s: có mục thiếu case_id", path)
		case seen[k]:
			return nil, fmt.Errorf("%s: case_id %s xuất hiện hai lần", path, k)
		default:
			seen[k] = true
		}
	}
	return list, nil
}

// caseTruth là truth của một ca trong hồ sơ ca (người soạn ca ghi, đóng băng cùng ca).
type caseTruth struct {
	Entities []string `json:"entities"`
	Period   *struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"period"`
	Value       string `json:"value"`
	OtherValues []struct {
		Value string `json:"value"`
	} `json:"other_values"`
	RecordIDs []string `json:"record_ids"`
}

// loadTruths đọc truth của từng ca từ hồ sơ ca; ca không có truth thì không có mục. Kỳ phải ở dạng
// YYYY-MM-DD (so chuỗi với ngày của bộ sinh).
func loadTruths(path string) (map[string]*caseTruth, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	type entry struct {
		CaseID string     `json:"case_id"`
		Truth  *caseTruth `json:"truth"`
	}
	list, err := caseEntries[entry](path, b, func(e entry) string { return e.CaseID })
	if err != nil {
		return nil, err
	}
	out := map[string]*caseTruth{}
	for _, e := range list {
		if e.Truth == nil {
			continue
		}
		if p := e.Truth.Period; p != nil {
			_, err1 := time.Parse("2006-01-02", p.From)
			_, err2 := time.Parse("2006-01-02", p.To)
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("%s: truth.period của %s không ở dạng YYYY-MM-DD", path, e.CaseID)
			}
		}
		out[e.CaseID] = e.Truth
	}
	return out, nil
}

// genRecord là một bản ghi trong manifest.json của BỘ SINH: giá trị chuẩn (lít) của bản ghi, độc lập
// với bộ nạp của EACL. Bản ghi bị cách ly không có value_l. Key trùng Observation.Key của bộ nạp.
type genRecord struct {
	Tenant  string `json:"tenant"`
	Key     string `json:"key"`
	WorldID string `json:"world_id"`
	Entity  string `json:"entity"`
	On      string `json:"on"`
	ValueL  string `json:"value_l"`
}

// loadGenRecords đọc bản ghi của manifest.json bộ sinh; không có bản ghi nào là lỗi.
func loadGenRecords(path string) ([]genRecord, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var gm struct {
		Records []genRecord `json:"records"`
	}
	if err := json.Unmarshal(b, &gm); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(gm.Records) == 0 {
		return nil, fmt.Errorf("%s: không có bản ghi nào — bộ đếm sẽ đếm oan mọi số của dữ liệu nền", path)
	}
	return gm.Records, nil
}

// perCaseNumbers tính G7CounterConfig.PerCase: số được phép RIÊNG cho từng ca của bộ đếm claim không
// căn cứ. Nguồn: truth của hồ sơ ca và manifest.json của BỘ SINH — không qua bộ nạp của EACL, nên độc
// lập với hệ được chấm. Số riêng của một ca: truth.value, other_values[].value, và value_l của mọi bản
// ghi thuộc tenant được chấm mà ca trỏ tới bằng truth.record_ids (khớp key hoặc world_id) hoặc có thực
// thể thuộc truth.entities với ngày trong truth.period. Ca không có truth thì không có số riêng (chặt hơn).
func perCaseNumbers(casesPath, genManifest, tenantID string) (map[string][]string, error) {
	truths, err := loadTruths(casesPath)
	if err != nil {
		return nil, err
	}
	recs, err := loadGenRecords(genManifest)
	if err != nil {
		return nil, err
	}
	out := map[string][]string{}
	for id, t := range truths {
		var from, to string
		if t.Period != nil {
			from, to = t.Period.From, t.Period.To
		}
		ids, ents := map[string]bool{}, map[string]bool{}
		for _, x := range t.RecordIDs {
			ids[x] = true
		}
		for _, x := range t.Entities {
			ents[x] = true
		}
		var nums []string
		if t.Value != "" {
			nums = append(nums, t.Value)
		}
		for _, o := range t.OtherValues {
			if o.Value != "" {
				nums = append(nums, o.Value)
			}
		}
		for _, r := range recs {
			if r.Tenant != tenantID || r.ValueL == "" {
				continue
			}
			inPeriod := t.Period != nil && ents[r.Entity] && r.On >= from && r.On <= to
			if ids[r.Key] || (r.WorldID != "" && ids[r.WorldID]) || inPeriod {
				nums = append(nums, r.ValueL)
			}
		}
		out[id] = nums
	}
	return out, nil
}

// oracleKeys trả bằng chứng vàng của B2-oracle: với mỗi ca có truth.record_ids, tập khoá bản ghi bộ sinh
// (trùng Observation.Key) mà ca trỏ tới — record_id là key của bộ sinh, hoặc world_id của bản ghi world.
// Chỉ bản ghi của tenant được chấm.
func oracleKeys(truths map[string]*caseTruth, recs []genRecord, tenantID string) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for id, t := range truths {
		ids := map[string]bool{}
		for _, x := range t.RecordIDs {
			ids[x] = true
		}
		keys := map[string]bool{}
		for _, r := range recs {
			if r.Tenant == tenantID && (ids[r.Key] || (r.WorldID != "" && ids[r.WorldID])) {
				keys[r.Key] = true
			}
		}
		if len(keys) > 0 {
			out[id] = keys
		}
	}
	return out
}

// oracleRAG là B2-oracle (ADR-0005 §2, D-G7-9 mục 5; CHỈ BÁO CÁO): B2+ với truy xuất được thay bằng ĐÚNG
// các đoạn của bản ghi mà ca trỏ tới (truth.record_ids), cùng id và chữ chỉ mục B2+ đặt cho đoạn. Ca
// không có bằng chứng vàng (câu hỏi tài liệu, ngoài phạm vi, thiếu dữ liệu) thì hỏi B2+ y nguyên. Vì vậy
// chênh lệch B2-oracle với B2+ là phần do truy xuất; chỗ B2-oracle vẫn trượt là thiếu tầng kiểm soát.
type oracleRAG struct {
	inner    *baseline.RAG                   // B2+ cho ca không có bằng chứng vàng
	model    models.Gateway                  // cùng gateway với mọi hệ
	evidence map[string][]contracts.Evidence // khoá askKey(người hỏi, câu hỏi)
	preamble string
	calls    atomic.Int64
}

// newOracleRAG dựng B2-oracle cho các ca đã nạp. obs và paras cùng thứ tự (paras[i] là đoạn B2+ của
// obs[i]), đúng thứ tự B2+ dựng chỉ mục, nên id "P-%06d#0" khớp id của chỉ mục B2+: baseline.New gộp
// khoảng trắng nên mỗi đoạn là MỘT dòng, và chỉ mục đặt id đoạn "<tài liệu>#<số dòng>" (rag.go).
func newOracleRAG(inner *baseline.RAG, gw models.Gateway, cases []evals.WinCase, keys map[string]map[string]bool, obs []ingestion.Observation, paras []string, preamble string) (*oracleRAG, error) {
	ids := make([]string, len(paras))
	n := 0
	for i, p := range paras {
		if strings.Join(strings.Fields(p), " ") != "" {
			n++
			ids[i] = fmt.Sprintf("P-%06d#0", n)
		}
	}
	o := &oracleRAG{inner: inner, model: gw, evidence: map[string][]contracts.Evidence{}, preamble: preamble}
	type seen struct {
		id   string
		gold bool
		ev   []contracts.Evidence
	}
	byQ := map[string]seen{}
	for _, c := range cases {
		ks := keys[c.ID]
		var ev []contracts.Evidence
		for i, ob := range obs {
			if ks[ob.Key] && ids[i] != "" {
				ev = append(ev, contracts.Evidence{ID: ids[i], Text: strings.Join(strings.Fields(paras[i]), " ")})
			}
		}
		k := askKey(c.Principal, c.Question)
		cur := seen{id: c.ID, gold: len(ks) > 0, ev: ev}
		if old, dup := byQ[k]; dup && (old.gold != cur.gold || !slices.EqualFunc(old.ev, cur.ev, func(a, b contracts.Evidence) bool { return a.ID == b.ID })) {
			return nil, fmt.Errorf("B2-oracle: %s và %s cùng người hỏi + câu hỏi nhưng khác bằng chứng vàng", old.id, c.ID)
		}
		byQ[k] = cur
		if cur.gold {
			o.evidence[k] = ev
		}
	}
	return o, nil
}

func (o *oracleRAG) Ask(ctx context.Context, pr contracts.Principal, question string) (contracts.AskResponse, error) {
	ev, ok := o.evidence[askKey(pr.ID, question)]
	if !ok {
		return o.inner.Ask(ctx, pr, question)
	}
	q := question
	if o.preamble != "" {
		q = o.preamble + "\n" + question
	}
	o.calls.Add(1)
	draft, err := o.model.Generate(ctx, models.Request{Question: q, Evidence: ev})
	if err != nil {
		return contracts.AskResponse{}, err
	}
	return contracts.AskResponse{Route: contracts.RouteRAG, Outcome: contracts.OutcomeResolved, Answer: draft.Text}, nil
}

func (o *oracleRAG) ModelCalls() int { return int(o.calls.Load()) + o.inner.ModelCalls() }
func (o *oracleRAG) ResetModelCalls() {
	o.calls.Store(0)
	o.inner.ResetModelCalls()
}

// directLLM là B1 (ADR-0005 §2): mỗi câu hỏi MỘT lời gọi model. Bằng chứng = entities.json + mọi tài
// liệu của tenant + mọi bản ghi thô có ngày trong cửa sổ của ca (bản ghi không đọc được ngày luôn có).
// Không chuẩn hoá, không truy xuất, không nhận dạng thực thể, không lọc quyền.
type directLLM struct {
	model    models.Gateway
	rows     []rawRow
	fixed    []contracts.Evidence // entities.json + tài liệu: luôn có
	windows  map[string]*window   // khoá: principal + "\x00" + câu hỏi
	preamble string
	calls    atomic.Int64
}

func askKey(principal, question string) string { return principal + "\x00" + question }

// newDirectLLM dựng B1 cho các ca đã nạp. Ca nào thiếu cửa sổ trong hồ sơ là lỗi: cửa sổ do người soạn
// ca gán, không đoán.
func newDirectLLM(gw models.Gateway, rows []rawRow, fixed []contracts.Evidence, cases []evals.WinCase, byCase map[string]*window, preamble string) (*directLLM, error) {
	d := &directLLM{model: gw, rows: rows, fixed: fixed, windows: map[string]*window{}, preamble: preamble}
	var missing []string
	for _, c := range cases {
		w, ok := byCase[c.ID]
		if !ok {
			missing = append(missing, c.ID)
			continue
		}
		k := askKey(c.Principal, c.Question)
		if old, dup := d.windows[k]; dup && !sameWindow(old, w) {
			return nil, fmt.Errorf("B1: hai ca cùng người hỏi + câu hỏi nhưng khác cửa sổ: %s", c.ID)
		}
		d.windows[k] = w
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("B1: %d ca thiếu b1_window trong hồ sơ ca, ví dụ %s", len(missing), missing[0])
	}
	return d, nil
}

func sameWindow(a, b *window) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.from.Equal(b.from) && a.to.Equal(b.to)
}

func (d *directLLM) Ask(ctx context.Context, pr contracts.Principal, question string) (contracts.AskResponse, error) {
	w, ok := d.windows[askKey(pr.ID, question)]
	if !ok {
		return contracts.AskResponse{}, fmt.Errorf("B1: câu hỏi không có trong bảng cửa sổ: %q", question)
	}
	ev := append([]contracts.Evidence(nil), d.fixed...)
	for _, r := range d.rows {
		if r.On.IsZero() || (w != nil && !r.On.Before(w.from) && !r.On.After(w.to)) {
			ev = append(ev, contracts.Evidence{ID: r.ID, Text: r.Text})
		}
	}
	q := question
	if d.preamble != "" {
		q = d.preamble + "\n" + question
	}
	d.calls.Add(1)
	draft, err := d.model.Generate(ctx, models.Request{Question: q, Evidence: ev})
	if err != nil {
		return contracts.AskResponse{}, err
	}
	return contracts.AskResponse{Route: contracts.RouteRAG, Outcome: contracts.OutcomeResolved, Answer: draft.Text}, nil
}

func (d *directLLM) ModelCalls() int  { return int(d.calls.Load()) }
func (d *directLLM) ResetModelCalls() { d.calls.Store(0) }

// fixedEvidence là phần B1 luôn nhận: entities.json nguyên văn + mọi tài liệu (tiêu đề + thân).
func fixedEvidence(dir string, docs []rag.Document) ([]contracts.Evidence, error) {
	b, err := os.ReadFile(filepath.Join(dir, "entities.json"))
	if err != nil {
		return nil, err
	}
	out := []contracts.Evidence{{ID: "entities.json", Text: strings.Join(strings.Fields(string(b)), " ")}}
	for _, d := range docs {
		out = append(out, contracts.Evidence{ID: "docs/" + d.ID, Text: d.Title + ": " + strings.Join(strings.Fields(d.Body), " ")})
	}
	return out, nil
}
