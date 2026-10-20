package gen

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/xuri/excelize/v2"
)

// create mở tệp ghi mới (cắt nội dung cũ) với quyền 0644 cho mọi tệp đầu ra.
func create(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
}

// csvFile ghi CSV theo từng dòng; lỗi ghi được giữ lại và trả ở close.
type csvFile struct {
	f *os.File
	w *csv.Writer
}

func newCSV(path string, header []string) (*csvFile, error) {
	f, err := create(path)
	if err != nil {
		return nil, err
	}
	c := &csvFile{f: f, w: csv.NewWriter(f)}
	c.row(header...)
	return c, nil
}

func (c *csvFile) row(cols ...string) { _ = c.w.Write(cols) } // lỗi nằm trong c.w.Error()

func (c *csvFile) close() error {
	c.w.Flush()
	err := c.w.Error()
	if cerr := c.f.Close(); err == nil {
		err = cerr
	}
	return err
}

// jsonArray ghi mảng JSON mỗi phần tử một dòng (hợp lệ JSON, ghi được hàng trăm nghìn phần tử mà
// không giữ hết trong bộ nhớ). prefix/suffix bọc quanh mảng, dùng cho manifest.json.
type jsonArray struct {
	f      *os.File
	out    bytes.Buffer // phần chờ ghi xuống đĩa
	one    bytes.Buffer // phần tử đang mã hoá
	enc    *json.Encoder
	suffix string
	n      int
	err    error
}

func newJSONArray(path, prefix, suffix string) (*jsonArray, error) {
	f, err := create(path)
	if err != nil {
		return nil, err
	}
	j := &jsonArray{f: f, suffix: suffix}
	j.enc = json.NewEncoder(&j.one)
	j.enc.SetEscapeHTML(false)
	j.out.WriteString(prefix + "[")
	return j, nil
}

func (j *jsonArray) add(v any) {
	if j.err != nil {
		return
	}
	j.one.Reset()
	if err := j.enc.Encode(v); err != nil {
		j.err = err
		return
	}
	if j.n > 0 {
		j.out.WriteString(",")
	}
	j.out.WriteString("\n")
	j.out.Write(bytes.TrimRight(j.one.Bytes(), "\n"))
	j.n++
	if j.out.Len() > 1<<16 { // xả định kỳ để bộ nhớ không phình
		j.flush()
	}
}

func (j *jsonArray) flush() {
	if j.err == nil {
		_, j.err = j.f.Write(j.out.Bytes())
	}
	j.out.Reset()
}

func (j *jsonArray) close() error {
	if j.n > 0 {
		j.out.WriteString("\n")
	}
	j.out.WriteString("]" + j.suffix)
	j.flush()
	err := j.err
	if cerr := j.f.Close(); err == nil {
		err = cerr
	}
	return err
}

// writeJSON ghi một giá trị JSON thụt lề; không escape HTML để chữ <, >, & giữ nguyên.
func writeJSON(path string, v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// writeJSONL ghi mỗi phần tử một dòng; không có phần tử thì tệp rỗng (JSONL không có header).
func writeJSONL[T any](path string, rows []T) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for _, r := range rows {
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// writeCSVRows ghi CSV nhỏ từ bộ nhớ.
func writeCSVRows(path string, header []string, rows [][]string) error {
	c, err := newCSV(path, header)
	if err != nil {
		return err
	}
	for _, r := range rows {
		c.row(r...)
	}
	return c.close()
}

// writeXLSX ghi một sheet; MỌI ô là chuỗi, đúng "như nguồn ghi" (ngày YYYY-MM-DD, "1,2" không bị
// Excel hiểu thành số). excelize giữ ngày tạo cố định và xếp tệp trong zip theo tên nên byte ổn định.
func writeXLSX(path, sheet string, header []string, rows [][]string) error {
	x := excelize.NewFile()
	defer x.Close()
	if err := x.SetSheetName("Sheet1", sheet); err != nil {
		return err
	}
	for i, cols := range append([][]string{header}, rows...) {
		cell, err := excelize.CoordinatesToCellName(1, i+1)
		if err != nil {
			return err
		}
		vals := make([]any, len(cols))
		for j, c := range cols {
			vals[j] = c
		}
		if err := x.SetSheetRow(sheet, cell, &vals); err != nil {
			return err
		}
	}
	buf, err := x.WriteToBuffer()
	if err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// fontDir là thư mục chứa DejaVuSans.ttf (font có tiếng Việt, ≈, ³). Đổi bằng WIN_FONT_DIR.
func fontDir() string {
	if d := os.Getenv("WIN_FONT_DIR"); d != "" {
		return d
	}
	return "/usr/share/fonts/truetype/dejavu"
}

// writePDF ghi báo cáo ngày: dòng đầu là tiêu đề, mỗi dòng sau là "<n>. <raw>". Ngày tạo/sửa đặt cố
// định, bỏ nén và sắp danh mục để cùng đầu vào ra cùng byte.
func writePDF(path, heading string, lines []string, stamp time.Time) error {
	pdf := fpdf.New("P", "mm", "A4", fontDir())
	pdf.SetCompression(false)
	pdf.SetCatalogSort(true)
	pdf.SetCreationDate(stamp)
	pdf.SetModificationDate(stamp)
	pdf.AddUTF8Font("DejaVu", "", "DejaVuSans.ttf") // chỉ tên tệp: fpdf tự ghép với fontDir
	pdf.SetFont("DejaVu", "", 12)
	pdf.AddPage()
	for _, l := range append([]string{heading}, lines...) {
		pdf.CellFormat(0, 8, l, "", 1, "L", false, 0, "")
	}
	if err := pdf.Error(); err != nil {
		return fmt.Errorf("dựng PDF %s: %w", path, err)
	}
	return pdf.OutputFileAndClose(path)
}
