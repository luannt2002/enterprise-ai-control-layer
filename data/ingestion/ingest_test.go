package ingestion_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/luannt2002/enterprise-ai-systems/core/tenant"
	"github.com/luannt2002/enterprise-ai-systems/data/ingestion"
	"github.com/xuri/excelize/v2"
)

const packDir = "../../tenants/construction/pack"

const fontDir = "/usr/share/fonts/truetype/dejavu"

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// dataset dựng tay một thư mục tenant đủ 5 định dạng, có bản ghi bẩn và chữ tự do khó.
func dataset(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("thiếu pdftotext (poppler) — bộ nạp PDF cần nó (ADR-0001 D4)")
	}
	if _, err := os.Stat(filepath.Join(fontDir, "DejaVuSans.ttf")); err != nil {
		t.Skip("thiếu font DejaVuSans để dựng PDF thử")
	}
	dir := t.TempDir()
	write(t, dir, "entities.json", `{"projects":[{"id":"PRJ-A","name":"Cầu A","aliases":["dự án A"]},{"id":"PRJ-B","name":"Đường B","aliases":["dự án B"]}],
"equipment":[
 {"id":"EX-03","type":"excavator","name":"Máy xúc EX-03","project":"PRJ-A","aliases":["máy 03","máy xúc 03","MX-03"],
  "source_ids":{"erp":"asset-8392","fuel_sensor":"EX003","site_report_form":"Machine 03","warehouse_slip":"EX-03"}},
 {"id":"RL-02","type":"roller","name":"Lu rung RL-02","project":"PRJ-A","aliases":["lu 02","lu rung 02"],"source_ids":{"warehouse_slip":"RL-02"}},
 {"id":"EX-07","type":"excavator","name":"Máy xúc EX-07","project":"PRJ-A","aliases":["máy xúc 07","máy 07"],"source_ids":{"warehouse_slip":"EX-07"}},
 {"id":"RL-07","type":"roller","name":"Lu rung RL-07","project":"PRJ-B","aliases":["lu 07","máy 07"],"source_ids":{"warehouse_slip":"RL-07"}}]}`)
	write(t, dir, "warehouse.csv", "record_id,slip_no,equipment_code,issue_date,quantity,unit\n"+
		"R-004,PX-0912,EX-03,2026-10-19,3850,L\n"+
		"R-016,PX-0913,RL-02,2026-10-19,\"1,2\",m3\n"+
		"R-021,PX-0999,EX-03,2026-10-19,,L\n"+
		"R-022,PX-0998,EX-99,2026-10-19,500,L\n"+
		"R-023,PX-0997,EX-03,2026-10-19,400000000,L\n"+
		"R-024,PX-0996,EX-03,2026-10-19,5,kg\n"+
		"R-025,PX-0995,EX03,2026-10-19,abc,L\n")
	write(t, dir, "erp.json", `[{"record_id":"R-005","doc_no":"ERP-77812","asset_id":"asset-8392","posting_date":"2026-10-19","material":"DIESEL","quantity":"3900","uom":"L"}]`)
	write(t, dir, "sensor.csv", "record_id,device_id,date,consumed_l\nR-006,EX003,2026-10-19,3782\n")
	write(t, dir, "messages.jsonl", strings.Join([]string{
		`{"record_id":"R-001","sent_at":"2026-10-19T07:30:00+07:00","sender":"Hùng","text":"máy 03 sáng 2k"}`,
		`{"record_id":"R-002","sent_at":"2026-10-19T15:10:00+07:00","sender":"Hùng","text":"chiều thêm 2k"}`,
		`{"record_id":"R-017","sent_at":"2026-10-19T16:00:00+07:00","sender":"Lan","text":"lu 02 khoảng 1k2"}`,
		`{"record_id":"R-030","sent_at":"2026-10-19T17:00:00+07:00","sender":"Minh","text":"lấy thêm 3k"}`,
		`{"record_id":"R-031","sent_at":"2026-10-19T17:30:00+07:00","sender":"Lan","text":"máy 07 lấy 500 lít"}`,
		`{"record_id":"R-032","sent_at":"2026-10-19T18:00:00+07:00","sender":"Tuấn","text":"sáng nay 2k cho máy 03"}`,
		`{"record_id":"R-033","sent_at":"2026-10-19T18:10:00+07:00","sender":"Tuấn","text":"máy 03 đổ bể 2 500 lít"}`,
		`{"record_id":"R-034","sent_at":"2026-10-19T18:20:00+07:00","sender":"Tuấn","text":"máy 03 lô A 1 200 lít"}`,
		`{"record_id":"R-035","sent_at":"2026-10-19T18:30:00+07:00","sender":"Tuấn","text":"máy 03 sửa chữa 500 000 đồng"}`,
		`{"record_id":"R-036","sent_at":"2026-10-19T18:40:00+07:00","sender":"Tuấn","text":"máy 03 đổ bể #2 500 lít"}`,
	}, "\n")+"\n")

	x := excelize.NewFile()
	x.SetSheetName("Sheet1", "PhieuXuat")
	for i, row := range [][]any{{"record_id", "slip_no", "equipment_code", "issue_date", "quantity", "unit"}, {"R-040", "PX-0901", "EX-03", "2026-10-18", "2000", "L"}} {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		if err := x.SetSheetRow("PhieuXuat", cell, &row); err != nil {
			t.Fatal(err)
		}
	}
	if err := x.SaveAs(filepath.Join(dir, "warehouse.xlsx")); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(dir, "reports"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := fpdf.New("P", "mm", "A4", fontDir)
	p.AddUTF8Font("DejaVu", "", "DejaVuSans.ttf")
	p.AddPage()
	p.SetFont("DejaVu", "", 12)
	for _, l := range []string{"Báo cáo ngày 19/10/2026", "1. Machine 03 ≈ 4 m³", "2. Lu rung 02 khoảng 1 khối"} {
		p.Cell(0, 8, l)
		p.Ln(8)
	}
	if err := p.OutputFileAndClose(filepath.Join(dir, "reports", "2026-10-19.pdf")); err != nil {
		t.Fatal(err)
	}
	return dir
}

func ingest(t *testing.T, dir string) ingestion.Result {
	t.Helper()
	rt, err := tenant.Load(packDir)
	if err != nil {
		t.Fatal(err)
	}
	sm, err := ingestion.LoadSources(filepath.Join(packDir, "sources.yaml"), rt)
	if err != nil {
		t.Fatal(err)
	}
	res, err := ingestion.Ingest(context.Background(), dir, rt, sm, ingestion.PdfToText{})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestIngestFiveFormatsToObservationsOrQuarantine(t *testing.T) {
	res := ingest(t, dataset(t))
	type want struct{ entity, metric, source, on, value string }
	wantObs := map[string]want{
		"warehouse.csv#R-004":      {"EX-03", "issued_fuel", "warehouse_slip", "2026-10-19", "3850"},
		"warehouse.csv#R-016":      {"RL-02", "issued_fuel", "warehouse_slip", "2026-10-19", "1200"}, // 1,2 m3
		"warehouse.xlsx#R-040":     {"EX-03", "issued_fuel", "warehouse_slip", "2026-10-18", "2000"},
		"erp.json#R-005":           {"EX-03", "issued_fuel", "erp", "2026-10-19", "3900"},                  // asset-8392
		"sensor.csv#R-006":         {"EX-03", "measured_consumption", "fuel_sensor", "2026-10-19", "3782"}, // EX003
		"messages.jsonl#R-001":     {"EX-03", "reported_fuel", "chat_message", "2026-10-19", "2000"},       // "2k"
		"messages.jsonl#R-002":     {"EX-03", "reported_fuel", "chat_message", "2026-10-19", "2000"},       // kế thừa thiết bị
		"messages.jsonl#R-017":     {"RL-02", "reported_fuel", "chat_message", "2026-10-19", "1200"},       // "1k2"
		"messages.jsonl#R-032":     {"EX-03", "reported_fuel", "chat_message", "2026-10-19", "2000"},       // số đứng TRƯỚC tên máy
		"reports/2026-10-19.pdf#1": {"EX-03", "reported_fuel", "site_report_form", "2026-10-19", "4000"},   // "Machine 03 ≈ 4 m³"
		"reports/2026-10-19.pdf#2": {"RL-02", "reported_fuel", "site_report_form", "2026-10-19", "1000"},
	}
	wantQ := map[string]string{
		"warehouse.csv#R-021":  ingestion.ReasonMissingValue,
		"warehouse.csv#R-022":  ingestion.ReasonUnknownEntity,
		"warehouse.csv#R-023":  ingestion.ReasonOutOfRange,
		"warehouse.csv#R-024":  ingestion.ReasonUnitDimension,
		"warehouse.csv#R-025":  ingestion.ReasonUnparseable,
		"messages.jsonl#R-030": ingestion.ReasonUnknownEntity,
		"messages.jsonl#R-031": ingestion.ReasonAmbiguousEntity,
		// "bể 2 500 lít": 2.500 L, hay bể 2 với 500 L? Không ghi một cách đọc vào nguồn sự thật — cách ly (soát C3, C3-02; fd4d19c F5)
		"messages.jsonl#R-033": ingestion.ReasonUnparseable,
		"messages.jsonl#R-034": ingestion.ReasonUnparseable,
		"messages.jsonl#R-036": ingestion.ReasonUnparseable,
		// tiền viết nhóm nghìn không bao giờ thành lượng dầu (soát fd4d19c, F2: bản không gộp từng nạp thành 500 L)
		"messages.jsonl#R-035": ingestion.ReasonMissingValue,
	}
	seen := map[string]int{}
	for _, o := range res.Observations {
		seen[o.Key]++
		w, ok := wantObs[o.Key]
		if !ok {
			t.Errorf("quan sát thừa %s (%s %s)", o.Key, o.Entity, o.Value.Number)
			continue
		}
		got := want{o.Entity, o.Metric, o.Source, o.On.Format("2006-01-02"), o.Value.Number}
		if got != w || o.Value.Unit != "L" {
			t.Errorf("%s = %+v %s, muốn %+v L", o.Key, got, o.Value.Unit, w)
		}
	}
	for _, q := range res.Quarantined {
		seen[q.Key]++
		if w, ok := wantQ[q.Key]; !ok || w != q.Reason {
			t.Errorf("cách ly %s lý do %s (%s), muốn %q", q.Key, q.Reason, q.Detail, w)
		}
	}
	// Không bản ghi nào rơi im lặng, không bản ghi nào bị đếm hai lần.
	for k := range wantObs {
		if seen[k] != 1 {
			t.Errorf("%s xuất hiện %d lần, muốn đúng 1", k, seen[k])
		}
	}
	for k := range wantQ {
		if seen[k] != 1 {
			t.Errorf("%s xuất hiện %d lần, muốn đúng 1", k, seen[k])
		}
	}
	for _, o := range res.Observations {
		if o.Key == "messages.jsonl#R-002" && !strings.Contains(o.Note, "kế thừa") {
			t.Errorf("R-002 phải ghi dấu thiết bị kế thừa, Note = %q", o.Note)
		}
		if o.Key == "messages.jsonl#R-017" && !o.Value.Approx {
			t.Errorf("R-017 'khoảng 1k2' phải là số ước lượng")
		}
	}
}

func TestLoadSourcesRejectsBrokenMaps(t *testing.T) {
	rt, err := tenant.Load(packDir)
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"chỉ số lạ":        "entities_file: e.json\nsources:\n  - {type: erp, metric: magic, files: [{path: a.json, format: json}], fields: {record_id: r, entity: e, date: d, quantity: q}, entity_namespace: erp}\n",
		"nguồn không hạng": "entities_file: e.json\nsources:\n  - {type: gossip, metric: issued_fuel, files: [{path: a.json, format: json}], fields: {record_id: r, entity: e, date: d, quantity: q}, entity_namespace: erp}\n",
		"thiếu trường":     "entities_file: e.json\nsources:\n  - {type: erp, metric: issued_fuel, files: [{path: a.json, format: json}], fields: {record_id: r}, entity_namespace: erp}\n",
		"định dạng lạ":     "entities_file: e.json\nsources:\n  - {type: erp, metric: issued_fuel, files: [{path: a.doc, format: doc}], fields: {record_id: r, entity: e, date: d, quantity: q}, entity_namespace: erp}\n",
		"xlsx thiếu sheet": "entities_file: e.json\nsources:\n  - {type: erp, metric: issued_fuel, files: [{path: a.xlsx, format: xlsx}], fields: {record_id: r, entity: e, date: d, quantity: q}, entity_namespace: erp}\n",
		"khoá gõ nhầm":     "entities_file: e.json\nsources:\n  - {type: erp, metrik: issued_fuel}\n",
		"thiếu entities":   "sources: []\n",
	} {
		p := filepath.Join(t.TempDir(), "sources.yaml")
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := ingestion.LoadSources(p, rt); err == nil {
			t.Errorf("%s: LoadSources phải lỗi", name)
		}
	}
}

var _ = time.Now
