package ingestion

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
	"github.com/xuri/excelize/v2"
)

// TextExtractor trích chữ từ PDF.
type TextExtractor interface {
	PDFText(path string) (string, error)
}

// PdfToText dùng `pdftotext` (poppler). Đo 20/10/2026: thư viện Go thuần đọc sai dấu tiếng Việt
// trong PDF nhúng font TrueType, pdftotext đọc đúng (ADR-0001 D4).
type PdfToText struct{}

// PDFText trả chữ của cả tệp, giữ bố cục dòng.
func (PdfToText) PDFText(path string) (string, error) {
	out, err := exec.Command("pdftotext", "-layout", "-enc", "UTF-8", path, "-").Output()
	if err != nil {
		return "", contracts.Label(contracts.Infra("pdftotext"), fmt.Errorf("pdftotext %s: %w", path, err))
	}
	return string(out), nil
}

// readRows đọc mọi bản ghi của một tệp thành bảng cột → giá trị (chuỗi).
func readRows(path string, f FileSpec, x TextExtractor) ([]map[string]string, error) {
	switch f.Format {
	case "csv":
		return readCSV(path)
	case "xlsx":
		return readXLSX(path, f.Sheet)
	case "json":
		return readJSON(path)
	case "jsonl":
		return readJSONL(path)
	case "pdf_lines":
		return readPDFLines(path, x)
	}
	return nil, fmt.Errorf("định dạng %q không hỗ trợ", f.Format)
}

func table(header []string, rows [][]string) []map[string]string {
	var out []map[string]string
	for _, r := range rows {
		m := make(map[string]string, len(header))
		for i, h := range header {
			if i < len(r) {
				m[strings.TrimSpace(h)] = r[i]
			} else {
				m[strings.TrimSpace(h)] = "" // ô cuối trống: excelize/CSV cắt bớt — vẫn là một ô rỗng
			}
		}
		out = append(out, m)
	}
	return out
}

func readCSV(path string) ([]map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	r := csv.NewReader(bytes.NewReader(b))
	r.FieldsPerRecord = -1
	recs, err := r.ReadAll()
	if err != nil || len(recs) == 0 {
		return nil, err
	}
	return table(recs[0], recs[1:]), nil
}

func readXLSX(path, sheet string) ([]map[string]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return table(rows[0], rows[1:]), nil
}

func stringify(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case json.Number:
		return t.String()
	case bool:
		return strconv.FormatBool(t)
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

func objects(raw []map[string]any) []map[string]string {
	out := make([]map[string]string, 0, len(raw))
	for _, o := range raw {
		m := make(map[string]string, len(o))
		for k, v := range o {
			m[k] = stringify(v)
		}
		out = append(out, m)
	}
	return out
}

func readJSON(path string) ([]map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var raw []map[string]any
	if err := dec.Decode(&raw); err != nil {
		return nil, err
	}
	return objects(raw), nil
}

func readJSONL(path string) ([]map[string]string, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	var raw []map[string]any
	sc := bufio.NewScanner(fh)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		dec := json.NewDecoder(strings.NewReader(line))
		dec.UseNumber()
		var o map[string]any
		if err := dec.Decode(&o); err != nil {
			return nil, fmt.Errorf("dòng %d: %w", n, err)
		}
		raw = append(raw, o)
	}
	return objects(raw), sc.Err()
}

// pdfLine là một dòng bản ghi của báo cáo ngày: "<n>. <chữ>".
var pdfLine = regexp.MustCompile(`^\s*(\d+)\.\s+(.+?)\s*$`)

// readPDFLines trả mỗi dòng bản ghi thành {record_id: n, text: chữ}.
func readPDFLines(path string, x TextExtractor) ([]map[string]string, error) {
	txt, err := x.PDFText(path)
	if err != nil {
		return nil, err
	}
	var out []map[string]string
	for _, l := range strings.Split(txt, "\n") {
		if m := pdfLine.FindStringSubmatch(l); m != nil {
			out = append(out, map[string]string{"record_id": m[1], "text": m[2]})
		}
	}
	return out, nil
}
