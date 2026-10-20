// Kiểm bộ sinh dữ liệu WIN qua API công khai (Generate) và đọc lại tệp đầu ra như một bộ nạp độc
// lập: không import bộ nạp, chỉ đọc CSV/XLSX/JSON/JSONL/PDF bằng thư viện chuẩn, excelize, pdftotext.
package gen_test

import (
	"bufio"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-systems/tenants/construction/win/gen"
	"github.com/xuri/excelize/v2"
)

const worldPath = "../world.yaml"

type entry struct {
	Tenant   string `json:"tenant"`
	Key      string `json:"key"`
	WorldID  string `json:"world_id"`
	Expect   string `json:"expect"`
	Reason   string `json:"reason"`
	Entity   string `json:"entity"`
	Semantic string `json:"semantic"`
	Source   string `json:"source"`
	On       string `json:"on"`
	ValueL   string `json:"value_l"`
	Approx   bool   `json:"approx"`
}

type manifest struct {
	Version       int     `json:"version"`
	ReferenceDate string  `json:"reference_date"`
	Profile       string  `json:"profile"`
	Seed          uint64  `json:"seed"`
	Records       []entry `json:"records"`
}

func build(t *testing.T, profile string, seed uint64) string {
	t.Helper()
	dir := t.TempDir()
	if err := gen.Generate(worldPath, dir, profile, seed); err != nil {
		t.Fatalf("Generate(%s, seed %d): %v", profile, seed, err)
	}
	return dir
}

func readJSONFile(t *testing.T, path string, v any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("%s không phải JSON hợp lệ: %v", path, err)
	}
}

func readManifest(t *testing.T, root string) manifest {
	t.Helper()
	var m manifest
	readJSONFile(t, filepath.Join(root, "manifest.json"), &m)
	return m
}

func readCSV(t *testing.T, path string) [][]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	if len(rows) == 0 {
		t.Fatalf("%s thiếu header", path)
	}
	return rows
}

func readXLSX(t *testing.T, path, sheet string) [][]string {
	t.Helper()
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := f.GetRows(sheet)
	if err != nil {
		t.Fatalf("%s sheet %s: %v", path, sheet, err)
	}
	if len(rows) == 0 {
		t.Fatalf("%s sheet %s thiếu header", path, sheet)
	}
	return rows
}

func readJSONL(t *testing.T, path string) []map[string]string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []map[string]string
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if line == "" {
			continue
		}
		var o map[string]string
		if err := json.Unmarshal([]byte(line), &o); err != nil {
			t.Fatalf("%s: dòng %q: %v", path, line, err)
		}
		out = append(out, o)
	}
	return out
}

// pdfText đọc PDF bằng pdftotext -layout, trả các dòng không rỗng.
func pdfText(t *testing.T, path string) []string {
	t.Helper()
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Fatalf("cần pdftotext (poppler-utils) để đọc PDF: %v", err)
	}
	out, err := exec.Command("pdftotext", "-layout", "-enc", "UTF-8", path, "-").Output()
	if err != nil {
		t.Fatalf("pdftotext %s: %v", path, err)
	}
	var lines []string
	for _, l := range strings.Split(string(out), "\n") {
		if l = strings.TrimSpace(strings.Trim(l, "\f")); l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

var pdfRecordLine = regexp.MustCompile(`^(\d+)\. (.+)$`)

// tenantKeys trả tập khoá "đường dẫn#id" của mọi bản ghi thật có trong một thư mục tenant.
func tenantKeys(t *testing.T, dir string) map[string]bool {
	t.Helper()
	keys := map[string]bool{}
	add := func(rel, id string) {
		k := rel + "#" + id
		if keys[k] {
			t.Errorf("%s: khoá %s xuất hiện hai lần", dir, k)
		}
		keys[k] = true
	}
	for _, f := range []string{"warehouse.csv", "sensor.csv", "invoices.csv"} {
		for _, r := range readCSV(t, filepath.Join(dir, f))[1:] {
			add(f, r[0])
		}
	}
	for f, sheet := range map[string]string{"warehouse.xlsx": "PhieuXuat", "excel_summary.xlsx": "TongHop"} {
		for _, r := range readXLSX(t, filepath.Join(dir, f), sheet)[1:] {
			add(f, r[0])
		}
	}
	var erp []map[string]string
	readJSONFile(t, filepath.Join(dir, "erp.json"), &erp)
	for _, r := range erp {
		add("erp.json", r["record_id"])
	}
	for _, r := range readJSONL(t, filepath.Join(dir, "messages.jsonl")) {
		add("messages.jsonl", r["record_id"])
	}
	pdfs, _ := filepath.Glob(filepath.Join(dir, "reports", "*.pdf"))
	for _, p := range pdfs {
		lines := pdfText(t, p)
		day := strings.TrimSuffix(filepath.Base(p), ".pdf") // YYYY-MM-DD
		if want := "Báo cáo ngày " + day[8:] + "/" + day[5:7] + "/" + day[:4]; lines[0] != want {
			t.Errorf("%s: dòng đầu %q, cần %q", p, lines[0], want)
		}
		n := 0
		for _, l := range lines[1:] {
			m := pdfRecordLine.FindStringSubmatch(l)
			if m == nil {
				t.Errorf("%s: dòng %q không có dạng '<n>. <chữ>'", p, l)
				continue
			}
			n++
			if m[1] != strconv.Itoa(n) {
				t.Errorf("%s: số thứ tự %s, cần %d", p, m[1], n)
			}
			add("reports/"+filepath.Base(p), m[1])
		}
	}
	return keys
}

// diffKeys trả khoá chỉ có ở a và khoá chỉ có ở b.
func diffKeys(a, b map[string]bool) (onlyA, onlyB []string) {
	for k := range a {
		if !b[k] {
			onlyA = append(onlyA, k)
		}
	}
	for k := range b {
		if !a[k] {
			onlyB = append(onlyB, k)
		}
	}
	sort.Strings(onlyA)
	sort.Strings(onlyB)
	return onlyA, onlyB
}

func hashTree(t *testing.T, root string) map[string]string {
	t.Helper()
	sums := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		h := sha256.Sum256(b)
		rel, _ := filepath.Rel(root, p)
		sums[rel] = hex.EncodeToString(h[:])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return sums
}

func sameTree(t *testing.T, a, b map[string]string) {
	t.Helper()
	if len(a) != len(b) {
		t.Errorf("số tệp lệch: %d và %d", len(a), len(b))
	}
	for rel, h := range a {
		if b[rel] != h {
			t.Errorf("tệp %s lệch sha256: %s và %s", rel, h, b[rel])
		}
	}
}

func TestDeterministic(t *testing.T) {
	a, b := build(t, "functional", 42), build(t, "functional", 42)
	ha, hb := hashTree(t, a), hashTree(t, b)
	if len(ha) < 30 {
		t.Fatalf("chỉ có %d tệp, bộ sinh ra thiếu", len(ha))
	}
	sameTree(t, ha, hb)

	// PDF phải mang ngày tạo/sửa cố định (nếu không, hai lần chạy khác giây sẽ ra byte khác nhau).
	pdf, err := os.ReadFile(filepath.Join(a, "abc-infra", "reports", "2026-10-19.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"/CreationDate (D:20261020000000)", "/ModDate (D:20261020000000)"} {
		if !strings.Contains(string(pdf), field) {
			t.Errorf("PDF thiếu %s", field)
		}
	}

	// Chạy lại vào thư mục đã có dữ liệu cũ vẫn ra đúng cùng một cây tệp.
	if err := gen.Generate(worldPath, a, "functional", 42); err != nil {
		t.Fatal(err)
	}
	sameTree(t, ha, hashTree(t, a))

	// Đối chứng: seed khác thì dữ liệu nền phải khác, để phép so ở trên không rỗng.
	c := hashTree(t, build(t, "functional", 43))
	if c[filepath.Join("abc-infra", "warehouse.csv")] == ha[filepath.Join("abc-infra", "warehouse.csv")] {
		t.Error("seed 43 ra cùng warehouse.csv với seed 42: seed không có tác dụng")
	}
}

func TestManifestCoversEveryRecordBothWays(t *testing.T) {
	root := build(t, "functional", 42)
	m := readManifest(t, root)
	if m.Version != 1 || m.ReferenceDate != "2026-10-20" || m.Profile != "functional" || m.Seed != 42 {
		t.Errorf("đầu manifest sai: %+v", m)
	}

	inManifest, tenants := map[string]bool{}, map[string]bool{}
	for _, r := range m.Records {
		k := r.Tenant + "|" + r.Key
		if inManifest[k] {
			t.Errorf("manifest có khoá trùng %s", k)
		}
		inManifest[k] = true
		tenants[r.Tenant] = true
	}
	inFiles := map[string]bool{}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range entries {
		if d.IsDir() && d.Name() != "variants" {
			for k := range tenantKeys(t, filepath.Join(root, d.Name())) {
				inFiles[d.Name()+"|"+k] = true
			}
		}
	}
	if len(tenants) != 2 {
		t.Errorf("manifest có %d tenant, cần 2", len(tenants))
	}
	missing, extra := diffKeys(inManifest, inFiles)
	if len(missing) > 0 {
		t.Errorf("%d khoá có trong manifest mà không có bản ghi trong tệp, ví dụ %v", len(missing), head(missing))
	}
	if len(extra) > 0 {
		t.Errorf("%d bản ghi trong tệp mà không có trong manifest, ví dụ %v", len(extra), head(extra))
	}
	if len(inFiles) < 2500 || len(inFiles) > 3500 {
		t.Errorf("%d bản ghi, hồ sơ functional phải cỡ 3.000", len(inFiles))
	}
}

// cell là một hàng bản ghi có cấu trúc rút từ tệp: mã thiết bị, lượng và đơn vị đúng như nguồn ghi.
type cell struct{ code, qty, unit string }

// structuredRows đọc các tệp có cấu trúc của một tenant thành "đường dẫn#id" → hàng, kèm bảng tra
// mã thiết bị hợp lệ theo từng bảng mã (warehouse_slip, erp, fuel_sensor) từ entities.json.
func structuredRows(t *testing.T, dir string) (map[string]cell, map[string]map[string]bool) {
	t.Helper()
	var ents struct {
		Equipment []struct {
			SourceIDs map[string]string `json:"source_ids"`
		} `json:"equipment"`
	}
	readJSONFile(t, filepath.Join(dir, "entities.json"), &ents)
	known := map[string]map[string]bool{"warehouse_slip": {}, "erp": {}, "fuel_sensor": {}}
	for _, e := range ents.Equipment {
		for ns, set := range known {
			set[e.SourceIDs[ns]] = true
		}
	}
	at := func(r []string, i int) string {
		if i < len(r) {
			return r[i]
		}
		return ""
	}
	rows := map[string]cell{}
	slips := func(f string, data [][]string) {
		for _, r := range data[1:] {
			rows[f+"#"+r[0]] = cell{at(r, 2), at(r, 4), at(r, 5)}
		}
	}
	slips("warehouse.csv", readCSV(t, filepath.Join(dir, "warehouse.csv")))
	slips("warehouse.xlsx", readXLSX(t, filepath.Join(dir, "warehouse.xlsx"), "PhieuXuat"))
	for _, r := range readXLSX(t, filepath.Join(dir, "excel_summary.xlsx"), "TongHop")[1:] {
		rows["excel_summary.xlsx#"+r[0]] = cell{at(r, 1), at(r, 3), "L"}
	}
	for _, r := range readCSV(t, filepath.Join(dir, "invoices.csv"))[1:] {
		rows["invoices.csv#"+r[0]] = cell{at(r, 2), at(r, 4), "L"}
	}
	for _, r := range readCSV(t, filepath.Join(dir, "sensor.csv"))[1:] {
		rows["sensor.csv#"+r[0]] = cell{at(r, 1), at(r, 3), "L"}
	}
	var erp []map[string]string
	readJSONFile(t, filepath.Join(dir, "erp.json"), &erp)
	for _, r := range erp {
		rows["erp.json#"+r["record_id"]] = cell{r["asset_id"], r["quantity"], r["uom"]}
	}
	return rows, known
}

// Mục manifest phải khớp NỘI DUNG tệp, không chỉ khớp khoá: quan sát mang đúng lượng, và lý do cách ly
// thấy được ngay trong ô (ô trống, mã lạ, đơn vị kg, số quá lớn). Tin nhắn và PDF là chữ tự do nên
// chỉ do bộ nạp đọc, không kiểm ở đây.
func TestManifestAgreesWithFileContent(t *testing.T) {
	root := build(t, "functional", 42)
	checked := 0
	for _, tn := range []string{"abc-infra", "def-build"} {
		rows, known := structuredRows(t, filepath.Join(root, tn))
		ns := map[string]string{"warehouse.csv": "warehouse_slip", "warehouse.xlsx": "warehouse_slip", "excel_summary.xlsx": "warehouse_slip",
			"invoices.csv": "warehouse_slip", "sensor.csv": "fuel_sensor", "erp.json": "erp"}
		for _, e := range readManifest(t, root).Records {
			file := e.Key[:strings.Index(e.Key, "#")]
			r, ok := rows[e.Key]
			if e.Tenant != tn || !ok {
				continue
			}
			checked++
			qty, err := strconv.ParseFloat(strings.ReplaceAll(r.qty, ",", "."), 64)
			switch e.Expect {
			case "observation":
				if r.unit == "m3" {
					qty *= 1000 // phiếu ghi theo khối: 1,2 m3 → 1200 L
				}
				want, _ := strconv.ParseFloat(e.ValueL, 64)
				if err != nil || (r.unit != "L" && r.unit != "m3") || !known[ns[file]][r.code] || qty != want {
					t.Errorf("%s: quan sát %s lượng %s đơn vị %s mã %s không khớp ô %+v", tn, e.Key, e.ValueL, r.unit, r.code, r)
				}
			case "quarantine":
				switch e.Reason {
				case "missing_value":
					ok = r.qty == ""
				case "unknown_entity":
					ok = !known[ns[file]][r.code]
				case "unit_dimension":
					ok = r.unit == "kg"
				case "out_of_range":
					ok = err == nil && qty > 20000
				default:
					ok = false
				}
				if !ok {
					t.Errorf("%s: cách ly %s lý do %s mà ô là %+v", tn, e.Key, e.Reason, r)
				}
			}
		}
	}
	if checked < 2500 {
		t.Errorf("chỉ đối chiếu nội dung %d bản ghi có cấu trúc", checked)
	}
}

// Đối chứng cho phép so tập khoá: bớt một khoá hay thêm một khoá thừa đều phải bị bắt.
func TestKeyComparerCatchesDrift(t *testing.T) {
	root := build(t, "functional", 42)
	files := tenantKeys(t, filepath.Join(root, "abc-infra"))
	man := map[string]bool{}
	for _, r := range readManifest(t, root).Records {
		if r.Tenant == "abc-infra" {
			man[r.Key] = true
		}
	}
	if a, b := diffKeys(man, files); len(a)+len(b) != 0 {
		t.Fatalf("đáng lẽ khớp: thiếu %v, thừa %v", head(a), head(b))
	}
	delete(man, "warehouse.csv#R-004")
	man["warehouse.csv#R-999"] = true
	a, b := diffKeys(man, files)
	if len(a) != 1 || a[0] != "warehouse.csv#R-999" || len(b) != 1 || b[0] != "warehouse.csv#R-004" {
		t.Errorf("không bắt đúng độ lệch: chỉ manifest %v, chỉ tệp %v", a, b)
	}
}

func head(s []string) []string {
	if len(s) > 3 {
		return s[:3]
	}
	return s
}

func find(t *testing.T, m manifest, tenant, worldID string) entry {
	t.Helper()
	for _, r := range m.Records {
		if r.Tenant == tenant && r.WorldID == worldID {
			return r
		}
	}
	t.Fatalf("manifest không có %s của tenant %s", worldID, tenant)
	return entry{}
}

func TestGoldenCases(t *testing.T) {
	root := build(t, "functional", 42)
	m := readManifest(t, root)

	r4 := find(t, m, "abc-infra", "R-004")
	want := entry{Tenant: "abc-infra", Key: "warehouse.csv#R-004", WorldID: "R-004", Expect: "observation", Entity: "EX-03",
		Semantic: "issued_fuel", Source: "warehouse_slip", On: "2026-10-19", ValueL: "3850"}
	if r4 != want {
		t.Errorf("R-004 = %+v, cần %+v", r4, want)
	}

	r16 := find(t, m, "abc-infra", "R-016")
	if r16.ValueL != "1200" || r16.Expect != "observation" || r16.Key != "warehouse.xlsx#R-016" {
		t.Errorf("R-016 = %+v, cần value_l 1200 ở warehouse.xlsx", r16)
	}
	x := readXLSX(t, filepath.Join(root, "abc-infra", "warehouse.xlsx"), "PhieuXuat")
	if got := strings.Join(x[1], "|"); got != "R-016|PX-0913|RL-02|2026-10-19|1,2|m3" {
		t.Errorf("hàng R-016 trong xlsx = %q (nguồn phải ghi 1,2 m3)", got)
	}

	// Bốn bản ghi bẩn: lý do đúng, và tệp render ĐÚNG như raw gợi ý.
	dirty := map[string]struct{ reason, row string }{
		"R-021": {"missing_value", "R-021,PX-0999,EX-03,2026-10-19,,L"},
		"R-022": {"unknown_entity", "R-022,PX-0998,EX-99,2026-10-19,500,L"},
		"R-023": {"out_of_range", "R-023,PX-0997,EX-03,2026-10-19,400000000,L"},
		"R-024": {"unit_dimension", "R-024,PX-0996,EX-03,2026-10-19,5,kg"},
	}
	rows := map[string]string{}
	for _, r := range readCSV(t, filepath.Join(root, "abc-infra", "warehouse.csv"))[1:] {
		rows[r[0]] = strings.Join(r, ",")
	}
	for id, c := range dirty {
		e := find(t, m, "abc-infra", id)
		if e.Expect != "quarantine" || e.Reason != c.reason || e.Entity != "" || e.ValueL != "" {
			t.Errorf("%s = %+v, cần quarantine/%s không kèm quan sát", id, e, c.reason)
		}
		if rows[id] != c.row {
			t.Errorf("%s render %q, cần %q", id, rows[id], c.row)
		}
	}

	if b := find(t, m, "def-build", "R-B01"); b.ValueL != "9999" || b.Expect != "observation" || b.Entity != "EX-03" {
		t.Errorf("R-B01 = %+v, cần value_l 9999 của EX-03", b)
	}
	if got := readCSV(t, filepath.Join(root, "def-build", "warehouse.csv")); len(got) != 2 || got[1][0] != "R-B01" {
		t.Errorf("canary chỉ được có R-B01 trong phiếu kho, có %v", got)
	}

	// Tin nhắn: ngày quan sát = ngày của sent_at theo +07:00; text = raw.
	msgs := readJSONL(t, filepath.Join(root, "abc-infra", "messages.jsonl"))
	if len(msgs) != 3 || msgs[0]["text"] != "máy 03 sáng 2k" || !strings.HasPrefix(msgs[0]["sent_at"], "2026-10-19T") || !strings.HasSuffix(msgs[0]["sent_at"], "+07:00") {
		t.Errorf("messages.jsonl = %v", msgs)
	}
	if r3 := find(t, m, "abc-infra", "R-003"); r3.Key != "reports/2026-10-19.pdf#1" || r3.ValueL != "4000" || !r3.Approx {
		t.Errorf("R-003 = %+v", r3)
	}
}

func TestPDFVietnameseReadable(t *testing.T) {
	root := build(t, "functional", 42)
	lines := pdfText(t, filepath.Join(root, "abc-infra", "reports", "2026-10-19.pdf"))
	found := false
	for _, l := range lines {
		found = found || strings.Contains(l, "Machine 03 ≈ 4 m³")
	}
	if !found {
		t.Errorf("pdftotext không ra dòng 'Machine 03 ≈ 4 m³', ra %q", lines)
	}
}

// Tệp không có bản ghi vẫn phải tạo: chỉ header, hoặc [] / rỗng / {}.
func TestEmptyFilesStillCreated(t *testing.T) {
	dir := filepath.Join(build(t, "functional", 42), "def-build")
	for f, header := range map[string]string{
		"sensor.csv":   "record_id,device_id,date,consumed_l",
		"invoices.csv": "record_id,invoice_no,equipment_code,date,quantity_l",
	} {
		if rows := readCSV(t, filepath.Join(dir, f)); len(rows) != 1 || strings.Join(rows[0], ",") != header {
			t.Errorf("%s = %v, cần đúng header %q", f, rows, header)
		}
	}
	for f, sheet := range map[string]string{"excel_summary.xlsx": "TongHop"} {
		if rows := readXLSX(t, filepath.Join(dir, f), sheet); len(rows) != 1 {
			t.Errorf("%s có %d hàng, cần chỉ header", f, len(rows))
		}
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "erp.json")); strings.TrimSpace(string(b)) != "[]" {
		t.Errorf("erp.json = %q, cần []", b)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "messages.jsonl")); err != nil || len(b) != 0 {
		t.Errorf("messages.jsonl phải tồn tại và rỗng: %q %v", b, err)
	}
	if st, err := os.Stat(filepath.Join(dir, "reports")); err != nil || !st.IsDir() {
		t.Errorf("thư mục reports phải tồn tại: %v", err)
	}
}

// Dữ liệu nền: 40 thiết bị FX thuộc 4 dự án nền riêng × các ngày của cửa sổ 30 ngày trừ ngày của
// world, ERP khoảng 50% ngày lệch 0–20 L, lỗi bơm theo tỉ lệ, và không chạm thiết bị hay ngày của world.
func TestBaseData(t *testing.T) {
	root := build(t, "functional", 42)
	dir := filepath.Join(root, "abc-infra")
	var ents struct {
		Projects []struct {
			ID      string   `json:"id"`
			Name    string   `json:"name"`
			Aliases []string `json:"aliases"`
		} `json:"projects"`
		Equipment []struct {
			ID        string            `json:"id"`
			Project   string            `json:"project"`
			SourceIDs map[string]string `json:"source_ids"`
		} `json:"equipment"`
	}
	readJSONFile(t, filepath.Join(dir, "entities.json"), &ents)
	worldIDs := map[string]bool{"EX-03": true, "DZ-05": true, "EX-07": true, "RL-07": true, "RL-02": true}
	base, perProject, erpAsset := 0, map[string]int{}, map[string]string{}
	for _, e := range ents.Equipment {
		if strings.HasPrefix(e.ID, "FX-") {
			base++
			perProject[e.Project]++
			erpAsset[e.SourceIDs["erp"]] = e.ID
		} else if !worldIDs[e.ID] {
			t.Errorf("thiết bị lạ trong entities.json: %s", e.ID)
		}
	}
	want := map[string]int{"PRJ-F01": 10, "PRJ-F02": 10, "PRJ-F03": 10, "PRJ-F04": 10, "PRJ-A": 3, "PRJ-B": 2}
	if base != 40 || len(perProject) != 4 || perProject["PRJ-F01"] != 10 || perProject["PRJ-F04"] != 10 || perProject["PRJ-A"] != 0 || perProject["PRJ-B"] != 0 {
		t.Errorf("thiết bị nền = %d, chia %v; cần 40 chia 10/10/10/10 cho PRJ-F01…F04, không có ở PRJ-A / PRJ-B", base, perProject)
	}
	names := map[string]string{}
	for _, p := range ents.Projects {
		names[p.ID] = p.Name
		if strings.HasPrefix(p.ID, "PRJ-F") && len(p.Aliases) != 0 {
			t.Errorf("dự án nền %s không được có bí danh (tránh trùng bí danh của world): %v", p.ID, p.Aliases)
		}
	}
	for id, name := range map[string]string{"PRJ-F01": "Dự án nền 01", "PRJ-F02": "Dự án nền 02", "PRJ-F03": "Dự án nền 03", "PRJ-F04": "Dự án nền 04", "PRJ-A": "Cầu A", "PRJ-B": "Đường B"} {
		if names[id] != name {
			t.Errorf("entities.json: dự án %s tên %q, cần %q", id, names[id], name)
		}
	}
	for _, e := range ents.Equipment {
		if !strings.HasPrefix(e.ID, "FX-") {
			want[e.Project]-- // thiết bị world vẫn ở đúng dự án của nó
		}
	}
	if want["PRJ-A"] != 0 || want["PRJ-B"] != 0 {
		t.Errorf("thiết bị của world không còn ở đúng PRJ-A / PRJ-B: %v", want)
	}

	// Ngày của world.yaml; cửa sổ functional là 2026-09-21…2026-10-20 (30 ngày).
	worldDates := map[string]bool{"2026-09-15": true, "2026-09-20": true, "2026-09-22": true, "2026-10-19": true, "2026-10-20": true}
	days := 0
	for d := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC); !d.After(time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)); d = d.AddDate(0, 0, 1) {
		if !worldDates[d.Format("2006-01-02")] {
			days++
		}
	}
	if days != 27 {
		t.Fatalf("test hỏng: cửa sổ còn %d ngày, cần 27", days)
	}

	m := readManifest(t, root)
	reasons, slips, erps, sensors := map[string]int{}, 0, 0, 0
	slipQty := map[string]int{} // "FX-001|2026-10-19" → lượng phiếu sạch
	for _, r := range m.Records {
		if r.WorldID != "" || r.Tenant != "abc-infra" {
			continue
		}
		if !strings.HasPrefix(r.Key[strings.Index(r.Key, "#")+1:], "F-") {
			t.Errorf("bản ghi nền có id lạ: %s", r.Key)
		}
		if r.Expect == "observation" && !strings.HasPrefix(r.Entity, "FX-") {
			t.Errorf("bản ghi nền chạm thiết bị ngoài FX: %+v", r)
		}
		if r.Expect == "observation" && worldDates[r.On] {
			t.Errorf("bản ghi nền chạm ngày của world: %+v", r)
		}
		switch {
		case strings.HasPrefix(r.Key, "warehouse.csv#"):
			slips++
			reasons[r.Reason]++
			if r.Expect == "observation" {
				q, _ := strconv.Atoi(r.ValueL)
				slipQty[r.Entity+"|"+r.On] = q
			}
		case strings.HasPrefix(r.Key, "erp.json#"):
			erps++
		case strings.HasPrefix(r.Key, "sensor.csv#"):
			sensors++
		}
	}
	if slips != 40*days || sensors != 40*days {
		t.Errorf("phiếu nền %d, cảm biến %d; cần 40×%d = %d mỗi loại", slips, sensors, days, 40*days)
	}
	if erps*100 < slips*40 || erps*100 > slips*60 {
		t.Errorf("ERP nền %d trên %d phiếu, cần khoảng 50%%", erps, slips)
	}
	for reason, per1000 := range map[string][2]int{"missing_value": {5, 40}, "unknown_entity": {2, 25}, "unit_dimension": {1, 15}} {
		if n := reasons[reason]; n*1000 < per1000[0]*slips || n*1000 > per1000[1]*slips {
			t.Errorf("lỗi %s: %d/%d phiếu nằm ngoài khoảng %d–%d‰", reason, n, slips, per1000[0], per1000[1])
		}
	}

	var erp []map[string]string
	readJSONFile(t, filepath.Join(dir, "erp.json"), &erp)
	checked := 0
	for _, r := range erp {
		id, ok := erpAsset[r["asset_id"]]
		if !ok {
			continue // ERP của world
		}
		if r["material"] != "DIESEL" || r["uom"] != "L" {
			t.Errorf("ERP nền sai material/uom: %v", r)
		}
		if q, ok := slipQty[id+"|"+r["posting_date"]]; ok {
			e, _ := strconv.Atoi(r["quantity"])
			if gap := e - q; gap < 0 || gap > 20 {
				t.Errorf("ERP %s lệch phiếu %d L, cần 0–20", r["record_id"], gap)
			}
			checked++
		}
	}
	if checked*100 < slips*35 {
		t.Errorf("chỉ đối chiếu được %d cặp ERP–phiếu", checked)
	}
}

// Hồ sơ smoke: cùng seed vẫn ra cùng byte ở quy mô lớn, đủ cỡ, và tỉ lệ lỗi bơm đúng như đã ghi
// (2% thiếu lượng, 1% mã lạ, 0,5% đơn vị kg trên phiếu kho nền).
func TestSmokeProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("hồ sơ smoke ghi ≈ 90 MB mỗi lần; bỏ qua khi chạy -short")
	}
	a, b := build(t, "smoke", 42), build(t, "smoke", 42)
	sameTree(t, hashTree(t, a), hashTree(t, b))

	f, err := os.Open(filepath.Join(a, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	total, reasons := 0, map[string]int{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, `{"tenant"`) {
			continue
		}
		total++
		if i := strings.Index(line, `"reason":"`); i >= 0 {
			reasons[strings.SplitN(line[i+len(`"reason":"`):], `"`, 2)[0]]++
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if total < 300000 {
		t.Errorf("smoke chỉ có %d bản ghi, cần ≥ 300.000", total)
	}
	slips := 400 * 360 // 400 thiết bị × (365 ngày trừ 5 ngày của world)
	for reason, per10000 := range map[string][2]int{"missing_value": {150, 250}, "unknown_entity": {75, 125}, "unit_dimension": {38, 62}} {
		if n := reasons[reason]; n*10000 < per10000[0]*slips || n*10000 > per10000[1]*slips {
			t.Errorf("smoke: lỗi %s %d/%d phiếu nằm ngoài %d–%d phần vạn", reason, n, slips, per10000[0], per10000[1])
		}
	}
}

// Dữ liệu nền không được làm lệch tổng theo dự án của world: không bản ghi nền nào thuộc dự án của
// world, nên "Cầu A tháng 9" vẫn đúng 3.500 + 1.100 + 800 = 5.400 L như world.yaml.
func TestBaseDataLeavesWorldProjectTotalsUntouched(t *testing.T) {
	root := build(t, "functional", 42)
	var ents struct {
		Equipment []struct {
			ID      string `json:"id"`
			Project string `json:"project"`
		} `json:"equipment"`
	}
	readJSONFile(t, filepath.Join(root, "abc-infra", "entities.json"), &ents)
	project := map[string]string{}
	for _, e := range ents.Equipment {
		project[e.ID] = e.Project
	}
	september := 0
	for _, r := range readManifest(t, root).Records {
		if r.Tenant != "abc-infra" || r.Expect != "observation" {
			continue
		}
		if p := project[r.Entity]; r.WorldID == "" && (p == "PRJ-A" || p == "PRJ-B") {
			t.Errorf("bản ghi nền %s thuộc dự án của world %s", r.Key, p)
		}
		if r.Semantic == "issued_fuel" && r.Source == "warehouse_slip" && project[r.Entity] == "PRJ-A" && strings.HasPrefix(r.On, "2026-09") {
			n, _ := strconv.Atoi(r.ValueL)
			september += n
		}
	}
	if september != 5400 {
		t.Errorf("Cầu A tháng 9 (phiếu kho, nghĩa issued_fuel) = %d L, world.yaml cho 5400", september)
	}
}

func TestVariants(t *testing.T) {
	root := build(t, "functional", 42)
	var vs []struct {
		WorldID string   `json:"world_id"`
		Tenant  string   `json:"tenant"`
		Source  string   `json:"source"`
		Files   []string `json:"files"`
		Expect  struct {
			Entity, Semantic, Source, On string
			ValueL                       string `json:"value_l"`
		} `json:"expect"`
	}
	readJSONFile(t, filepath.Join(root, "variants", "manifest.json"), &vs)
	if len(vs) != 2 {
		t.Fatalf("variants/manifest.json có %d mục, cần 2 (R-004, R-001)", len(vs))
	}
	for _, v := range vs {
		want := map[string]int{"R-004": 3, "R-001": 4}[v.WorldID]
		if len(v.Files) < want || v.Expect.Entity != "EX-03" || v.Expect.On != "2026-10-19" || v.Expect.Source != v.Source {
			t.Errorf("mục biến thể %s sai: %+v", v.WorldID, v)
		}
		texts := map[string]bool{}
		for _, f := range v.Files {
			p := filepath.Join(root, "variants", filepath.FromSlash(f))
			dir := filepath.Dir(p)
			if _, err := os.Stat(filepath.Join(dir, "entities.json")); err != nil {
				t.Errorf("%s: thiếu entities.json trong thư mục biến thể", f)
			}
			// Mỗi thư mục biến thể chứa đúng MỘT bản ghi, theo bố cục chuẩn.
			n := len(tenantKeys(t, dir))
			switch filepath.Base(p) {
			case "warehouse.json":
				var rows []map[string]string
				readJSONFile(t, p, &rows)
				n += len(rows)
				if len(rows) == 1 && (rows[0]["record_id"] != "R-004" || rows[0]["quantity"] != "3850" || rows[0]["unit"] != "L" || rows[0]["equipment_code"] != "EX-03") {
					t.Errorf("%s = %v", f, rows)
				}
			case "messages.jsonl":
				msgs := readJSONL(t, p)
				if len(msgs) == 1 {
					texts[msgs[0]["text"]] = true
				}
			}
			if n != 1 {
				t.Errorf("%s: thư mục biến thể có %d bản ghi, cần đúng 1", f, n)
			}
		}
		if v.WorldID == "R-001" && len(texts) < 4 {
			t.Errorf("R-001 chỉ có %d cách viết khác nhau, cần ≥ 4: %v", len(texts), texts)
		}
	}
}

func TestGenerateRejectsBadInput(t *testing.T) {
	if err := gen.Generate(worldPath, t.TempDir(), "huge", 1); err == nil || !strings.Contains(err.Error(), "hồ sơ") {
		t.Errorf("hồ sơ lạ phải bị từ chối bằng thông báo tiếng Việt, được %v", err)
	}
	if err := gen.Generate("không-có-tệp.yaml", t.TempDir(), "functional", 1); err == nil {
		t.Error("world không tồn tại phải báo lỗi")
	}
	for name, tenantYAML := range map[string]string{
		"id dự án nền":      "projects: [{id: PRJ-F01, name: X, aliases: []}], equipment: []",
		"bí danh trùng tên": "projects: [{id: P, name: X, aliases: [dự án nền 02]}], equipment: []",
		"id thiết bị nền":   "projects: [{id: P, name: X, aliases: []}], equipment: [{id: FX-007, type: t, name: n, project: P, aliases: [], source_ids: {}}]",
		"mã nguồn nền":      "projects: [{id: P, name: X, aliases: []}], equipment: [{id: E, type: t, name: n, project: P, aliases: [], source_ids: {erp: asset-9003}}]",
	} {
		w := filepath.Join(t.TempDir(), "world.yaml")
		if err := os.WriteFile(w, []byte("reference_date: 2026-10-20\ntenants:\n  - {id: t1, "+tenantYAML+"}\nrecords: []\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := gen.Generate(w, t.TempDir(), "functional", 1); err == nil || !strings.Contains(err.Error(), "dữ liệu nền") {
			t.Errorf("%s: phải bị chặn vì trùng với dữ liệu nền, được %v", name, err)
		}
	}
	bad := filepath.Join(t.TempDir(), "world.yaml")
	if err := os.WriteFile(bad, []byte("reference_date: 2026-10-20\ntenants: [{id: a}]\nrecords: [{id: R-1, tenant: a, on: 2026-10-19, source: nguon_la}]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gen.Generate(bad, t.TempDir(), "functional", 1); err == nil || !strings.Contains(err.Error(), "nguồn") {
		t.Errorf("nguồn lạ phải bị từ chối, được %v", err)
	}
}
