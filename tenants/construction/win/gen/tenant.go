package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Cột của các tệp bảng (CONTRACT §2).
var (
	slipHeader    = []string{"record_id", "slip_no", "equipment_code", "issue_date", "quantity", "unit"}
	summaryHeader = []string{"record_id", "equipment_code", "date", "quantity_l"}
	sensorHeader  = []string{"record_id", "device_id", "date", "consumed_l"}
	invoiceHeader = []string{"record_id", "invoice_no", "equipment_code", "date", "quantity_l"}
)

type erpRow struct {
	RecordID    string `json:"record_id"`
	DocNo       string `json:"doc_no"`
	AssetID     string `json:"asset_id"`
	PostingDate string `json:"posting_date"`
	Material    string `json:"material"`
	Quantity    string `json:"quantity"` // chuỗi, đúng hợp đồng
	UOM         string `json:"uom"`
}

type msgRow struct {
	RecordID string `json:"record_id"`
	SentAt   string `json:"sent_at"`
	Sender   string `json:"sender"`
	Text     string `json:"text"`
}

type entitiesFile struct {
	Projects  []project   `json:"projects"`
	Equipment []equipment `json:"equipment"`
}

// tenantInput là mọi thứ cần để ghi một thư mục tenant.
type tenantInput struct {
	def     tenantDef
	records []record
	docs    []document
	base    *baseSpec // nil: không sinh dữ liệu nền
	ref     time.Time // reference_date; cũng là dấu thời gian cố định của PDF
}

// streams là ba tệp có dữ liệu nền nên ghi tuần tự thay vì giữ trong bộ nhớ.
type streams struct {
	slip   *csvFile
	erp    *jsonArray
	sensor *csvFile
}

func openStreams(dir string) (*streams, error) {
	slip, err := newCSV(filepath.Join(dir, "warehouse.csv"), slipHeader)
	if err != nil {
		return nil, err
	}
	erp, err := newJSONArray(filepath.Join(dir, "erp.json"), "", "\n")
	if err != nil {
		_ = slip.close()
		return nil, err
	}
	sensor, err := newCSV(filepath.Join(dir, "sensor.csv"), sensorHeader)
	if err != nil {
		_ = slip.close()
		_ = erp.close()
		return nil, err
	}
	return &streams{slip: slip, erp: erp, sensor: sensor}, nil
}

func (s *streams) close() error {
	var first error
	for _, err := range []error{s.slip.close(), s.erp.close(), s.sensor.close()} {
		if first == nil {
			first = err
		}
	}
	return first
}

// small gom các tệp chỉ có bản ghi world (vài dòng) nên giữ trong bộ nhớ rồi ghi một lần.
type small struct {
	slipXLSX, summary, invoices [][]string
	msgs                        []msgRow
	msgPerDay                   map[string]int
	pdfLines                    map[string][]string // ngày → raw của từng dòng báo cáo, theo thứ tự world
}

// writeTenant ghi toàn bộ thư mục một tenant theo CONTRACT §1–§2 và đẩy từng mục manifest qua emit.
func writeTenant(dir string, in tenantInput, emit func(any)) error {
	for _, sub := range []string{"reports", "docs"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			return err
		}
	}
	projects := append([]project(nil), in.def.Projects...)
	eqs := append([]equipment(nil), in.def.Equipment...)
	if in.base != nil {
		projects = append(projects, baseProjects()...)
		eqs = append(eqs, baseEquipment(in.base.devices)...)
	}
	if err := writeJSON(filepath.Join(dir, "entities.json"), entitiesFile{Projects: projects, Equipment: eqs}); err != nil {
		return err
	}

	s, err := openStreams(dir)
	if err != nil {
		return err
	}
	sm, werr := renderWorld(in, s, emit)
	if werr == nil && in.base != nil {
		genBase(in.base, eqs[len(in.def.Equipment):], s, emit)
	}
	if cerr := s.close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		return werr
	}
	return writeSmall(dir, in, sm)
}

// renderWorld render từng bản ghi world theo nguồn của nó. Bản ghi cách ly render ĐÚNG như raw gợi ý
// (ô trống, mã lạ, kg, số khổng lồ) và vào manifest cùng reason của world.
func renderWorld(in tenantInput, s *streams, emit func(any)) (*small, error) {
	sm := &small{msgPerDay: map[string]int{}, pdfLines: map[string][]string{}}
	for _, r := range in.records {
		e, known := in.def.find(r.Entity)
		if !known && !(r.quarantined() && r.Reason == "unknown_entity") {
			return nil, fmt.Errorf("bản ghi %s: thiết bị %q không có trong tenant %q", r.ID, r.Entity, in.def.ID)
		}
		code := r.Entity // thiết bị lạ ghi nguyên mã của world
		if known {
			var err error
			if code, err = codeOf(e, r.Source); err != nil {
				return nil, fmt.Errorf("bản ghi %s: %w", r.ID, err)
			}
		}
		key, err := sm.render(r, code, s)
		if err != nil {
			return nil, err
		}
		emit(entryFor(in.def.ID, key, r))
	}
	return sm, nil
}

// render ghi một bản ghi world và trả khoá manifest (đường dẫn trong thư mục tenant + "#" + id/số dòng).
func (sm *small) render(r record, code string, s *streams) (string, error) {
	switch r.Source {
	case "warehouse_slip":
		cols, err := slipColumns(r, code)
		if err != nil {
			return "", err
		}
		if r.Format == "xlsx" {
			sm.slipXLSX = append(sm.slipXLSX, cols)
			return "warehouse.xlsx#" + r.ID, nil
		}
		s.slip.row(cols...)
		return "warehouse.csv#" + r.ID, nil
	case "excel_summary":
		sm.summary = append(sm.summary, []string{r.ID, code, r.On, r.ValueL})
		return "excel_summary.xlsx#" + r.ID, nil
	case "erp":
		s.erp.add(erpRow{RecordID: r.ID, DocNo: r.Ref, AssetID: code, PostingDate: r.On, Material: "DIESEL", Quantity: r.ValueL, UOM: "L"})
		return "erp.json#" + r.ID, nil
	case "fuel_sensor":
		s.sensor.row(r.ID, code, r.On, r.ValueL)
		return "sensor.csv#" + r.ID, nil
	case "invoice":
		sm.invoices = append(sm.invoices, []string{r.ID, r.Ref, code, r.On, r.ValueL})
		return "invoices.csv#" + r.ID, nil
	}
	if r.Raw == "" {
		return "", fmt.Errorf("bản ghi %s: nguồn %s cần trường raw", r.ID, r.Source)
	}
	if r.Source == "chat_message" {
		k := sm.msgPerDay[r.On]
		sm.msgPerDay[r.On]++
		sender := r.By
		if sender == "" {
			sender = "hiện trường"
		}
		sm.msgs = append(sm.msgs, msgRow{RecordID: r.ID, SentAt: sentAt(r.On, k), Sender: sender, Text: r.Raw})
		return "messages.jsonl#" + r.ID, nil
	}
	sm.pdfLines[r.On] = append(sm.pdfLines[r.On], r.Raw) // site_report_form
	return fmt.Sprintf("reports/%s.pdf#%d", r.On, len(sm.pdfLines[r.On])), nil
}

// sentAt cho tin thứ k trong ngày một giờ trong ngày (09:15, 13:15, 17:15, …) theo +07:00, luôn
// nằm trong đúng ngày đó để "ngày quan sát = ngày của sent_at" đúng.
func sentAt(on string, k int) string {
	d, _ := time.Parse(dateLayout, on) // on đã được validate
	tz := time.FixedZone("", 7*3600)
	return time.Date(d.Year(), d.Month(), d.Day(), 9+4*(k%3), 15+k/3, 0, 0, tz).Format(time.RFC3339)
}

// codeOf trả mã thiết bị theo bảng mã của nguồn: bảng tổng hợp và hoá đơn dùng mã phiếu kho (khớp
// entity_namespace của pack); ERP và cảm biến bắt buộc có mã riêng trong source_ids.
func codeOf(e equipment, source string) (string, error) {
	ns := source
	if source == "excel_summary" || source == "invoice" {
		ns = "warehouse_slip"
	}
	if c, ok := e.SourceIDs[ns]; ok {
		return c, nil
	}
	if ns == "erp" || ns == "fuel_sensor" {
		return "", fmt.Errorf("thiết bị %s không có mã %s trong source_ids", e.ID, ns)
	}
	return e.ID, nil
}

// slipColumns dựng sáu cột phiếu kho. Ba dạng raw:
//   - "PX-0999 | EX-03 | ??? | L": bản ghi bẩn, tách bốn phần; value_l rỗng thì ô lượng để trống;
//   - "1,2 m3": lượng ghi theo khối, tách ở dấu cách cuối;
//   - rỗng: lượng = value_l, đơn vị L.
func slipColumns(r record, code string) ([]string, error) {
	slipNo, equip, qty, unit := r.Ref, code, r.ValueL, "L"
	switch {
	case strings.Contains(r.Raw, "|"):
		p := strings.Split(r.Raw, "|")
		if len(p) != 4 {
			return nil, fmt.Errorf("bản ghi %s: raw %q phải có dạng 'số phiếu | mã | lượng | đơn vị'", r.ID, r.Raw)
		}
		for i := range p {
			p[i] = strings.TrimSpace(p[i])
		}
		slipNo, equip, qty, unit = p[0], p[1], p[2], p[3]
		if r.ValueL == "" {
			qty = ""
		}
	case r.Raw != "":
		i := strings.LastIndex(r.Raw, " ")
		if i <= 0 {
			return nil, fmt.Errorf("bản ghi %s: raw %q phải có dạng 'lượng đơn_vị'", r.ID, r.Raw)
		}
		qty, unit = strings.TrimSpace(r.Raw[:i]), r.Raw[i+1:]
	}
	return []string{r.ID, slipNo, equip, r.On, qty, unit}, nil
}

// writeSmall ghi các tệp còn lại: hai xlsx, hoá đơn, tin nhắn, báo cáo PDF, tài liệu SOP + acl.json.
func writeSmall(dir string, in tenantInput, sm *small) error {
	if err := writeXLSX(filepath.Join(dir, "warehouse.xlsx"), "PhieuXuat", slipHeader, sm.slipXLSX); err != nil {
		return err
	}
	if err := writeXLSX(filepath.Join(dir, "excel_summary.xlsx"), "TongHop", summaryHeader, sm.summary); err != nil {
		return err
	}
	if err := writeCSVRows(filepath.Join(dir, "invoices.csv"), invoiceHeader, sm.invoices); err != nil {
		return err
	}
	if err := writeJSONL(filepath.Join(dir, "messages.jsonl"), sm.msgs); err != nil {
		return err
	}
	dates := make([]string, 0, len(sm.pdfLines))
	for d := range sm.pdfLines {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	for _, d := range dates {
		day, _ := time.Parse(dateLayout, d) // d đã được validate
		lines := make([]string, len(sm.pdfLines[d]))
		for i, raw := range sm.pdfLines[d] {
			lines[i] = fmt.Sprintf("%d. %s", i+1, raw)
		}
		if err := writePDF(filepath.Join(dir, "reports", d+".pdf"), "Báo cáo ngày "+day.Format("02/01/2006"), lines, in.ref); err != nil {
			return err
		}
	}
	acl := map[string][]string{}
	for _, d := range in.docs {
		body := strings.TrimRight(d.Body, "\n") + "\n"
		if err := os.WriteFile(filepath.Join(dir, "docs", d.ID+".md"), []byte("# "+d.Title+"\n\n"+body), 0o644); err != nil {
			return err
		}
		acl[d.ID] = append([]string{}, d.ACL...)
	}
	return writeJSON(filepath.Join(dir, "docs", "acl.json"), acl)
}
