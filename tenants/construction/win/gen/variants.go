package gen

import (
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// Bản ghi mang biến thể metamorphic (CONTRACT §4).
const (
	slipVariantID = "R-004" // phiếu kho: CSV, XLSX, JSON, CSV gõ tay (số kiểu Việt, đơn vị viết chữ)
	chatVariantID = "R-001" // tin nhắn hiện trường: nhiều cách viết
)

// chatVariantTexts là cùng một tin nhắn viết bốn cách: bỏ dấu, viết đủ chữ, mã khác, đơn vị khác.
// Quan sát chuẩn của cả bốn phải là một (thực thể, nghĩa, lượng, ngày, nguồn).
var chatVariantTexts = []string{
	"máy 03 sáng 2k",
	"may 03 sang 2k",
	"máy xúc 03 sáng lấy 2.000 lít",
	"MX-03 sáng 2 khối",
}

type variantExpect struct {
	Entity   string `json:"entity"`
	Semantic string `json:"semantic"`
	Source   string `json:"source"`
	On       string `json:"on"`
	ValueL   string `json:"value_l"`
}

// variantEntry là một mục của variants/manifest.json. Files là tệp dữ liệu chính của từng biến thể,
// tương đối so với thư mục variants/ (nơi đặt manifest); thư mục chứa tệp là một bố cục đủ như mục 1.
type variantEntry struct {
	WorldID string        `json:"world_id"`
	Tenant  string        `json:"tenant"`
	Source  string        `json:"source"`
	Files   []string      `json:"files"`
	Expect  variantExpect `json:"expect"`
}

// slipJSON là một phần tử của warehouse.json: cùng cột với warehouse.csv, mọi giá trị là chuỗi.
type slipJSON struct {
	RecordID      string `json:"record_id"`
	SlipNo        string `json:"slip_no"`
	EquipmentCode string `json:"equipment_code"`
	IssueDate     string `json:"issue_date"`
	Quantity      string `json:"quantity"`
	Unit          string `json:"unit"`
}

func newVariant(r record) variantEntry {
	return variantEntry{WorldID: r.ID, Tenant: r.Tenant, Source: r.Source,
		Expect: variantExpect{Entity: r.Entity, Semantic: r.Semantic, Source: r.Source, On: r.On, ValueL: r.ValueL}}
}

// writeVariants sinh variants/<world_id>/<n>/ cho phiếu kho và tin nhắn, rồi variants/manifest.json.
// Mỗi thư mục chỉ chứa MỘT bản ghi (các tệp khác chỉ có header) để nạp riêng ra đúng một quan sát.
func writeVariants(w *world, out string) error {
	slip, err := w.record(slipVariantID)
	if err != nil {
		return err
	}
	chat, err := w.record(chatVariantID)
	if err != nil {
		return err
	}
	if chat.Raw != chatVariantTexts[0] {
		return fmt.Errorf("raw của %s đã đổi thành %q: cập nhật chatVariantTexts cho khớp", chat.ID, chat.Raw)
	}

	slipV := newVariant(slip)
	for i, f := range []struct{ file, format string }{{"warehouse.csv", ""}, {"warehouse.xlsx", "xlsx"}, {"warehouse.json", ""}} {
		dir := filepath.Join(out, "variants", slip.ID, strconv.Itoa(i+1))
		r := slip
		r.Format = f.format
		recs := []record{r}
		if f.file == "warehouse.json" {
			recs = nil // JSON không thuộc bố cục chuẩn: bản ghi nằm ở warehouse.json
		}
		if err := writeVariantDir(w, dir, slip.Tenant, recs); err != nil {
			return err
		}
		if f.file == "warehouse.json" {
			if err := writeSlipJSON(w, dir, slip); err != nil {
				return err
			}
		}
		slipV.Files = append(slipV.Files, path.Join(slip.ID, strconv.Itoa(i+1), f.file))
	}
	// Biểu diễn thứ tư: CSV gõ tay — số có dấu chấm nghìn ("3.850") và đơn vị viết chữ ("lít").
	dir4 := filepath.Join(out, "variants", slip.ID, "4")
	if err := writeVariantDir(w, dir4, slip.Tenant, nil); err != nil {
		return err
	}
	if err := writeSlipCSVHand(w, dir4, slip); err != nil {
		return err
	}
	slipV.Files = append(slipV.Files, path.Join(slip.ID, "4", "warehouse.csv"))

	chatV := newVariant(chat)
	for i, text := range chatVariantTexts {
		r := chat
		r.Raw = text
		if err := writeVariantDir(w, filepath.Join(out, "variants", chat.ID, strconv.Itoa(i+1)), chat.Tenant, []record{r}); err != nil {
			return err
		}
		chatV.Files = append(chatV.Files, path.Join(chat.ID, strconv.Itoa(i+1), "messages.jsonl"))
	}
	return writeJSON(filepath.Join(out, "variants", "manifest.json"), []variantEntry{slipV, chatV})
}

// writeVariantDir ghi một thư mục theo bố cục mục 1 chỉ với thiết bị của world và các bản ghi recs.
func writeVariantDir(w *world, dir, tenant string, recs []record) error {
	def, _ := w.tenant(tenant) // tenant của bản ghi đã được validate
	return writeTenant(dir, tenantInput{def: def, records: recs, ref: w.ref}, func(any) {})
}

// writeSlipCSVHand ghi phiếu kho như người gõ tay: lượng có dấu chấm nghìn, đơn vị viết chữ.
func writeSlipCSVHand(w *world, dir string, r record) error {
	def, _ := w.tenant(r.Tenant)
	e, ok := def.find(r.Entity)
	if !ok {
		return fmt.Errorf("bản ghi %s: thiết bị %q không có trong tenant %q", r.ID, r.Entity, r.Tenant)
	}
	code, err := codeOf(e, r.Source)
	if err != nil {
		return err
	}
	c, err := slipColumns(r, code)
	if err != nil {
		return err
	}
	if c[5] != "L" || len(c[4]) < 4 || strings.ContainsAny(c[4], ".,") {
		return fmt.Errorf("bản ghi %s: biến thể gõ tay cần lượng nguyên ≥ 1000 lít, được %q %q", r.ID, c[4], c[5])
	}
	c[4], c[5] = c[4][:len(c[4])-3]+"."+c[4][len(c[4])-3:], "lít"
	return writeCSVRows(filepath.Join(dir, "warehouse.csv"), slipHeader, [][]string{c})
}

func writeSlipJSON(w *world, dir string, r record) error {
	def, _ := w.tenant(r.Tenant)
	e, ok := def.find(r.Entity)
	if !ok {
		return fmt.Errorf("bản ghi %s: thiết bị %q không có trong tenant %q", r.ID, r.Entity, r.Tenant)
	}
	code, err := codeOf(e, r.Source)
	if err != nil {
		return err
	}
	c, err := slipColumns(r, code)
	if err != nil {
		return err
	}
	return writeJSON(filepath.Join(dir, "warehouse.json"), []slipJSON{{c[0], c[1], c[2], c[3], c[4], c[5]}})
}
