package rag

import (
	"context"
	"encoding/csv"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Source là MỘT nguồn tài liệu tri thức của tenant (strategy): thư mục markdown, tệp import ngoài, database có sẵn… Quyền đọc (ACL),
// phiên bản, hiệu lực đến từ chính nguồn hoặc từ cấu hình của nguồn — không nguồn nào được mặc định "ai cũng đọc".
type Source interface {
	Load(ctx context.Context, tenant string) ([]Document, error)
}

// SourceFunc biến một hàm thành Source.
type SourceFunc func(ctx context.Context, tenant string) ([]Document, error)

func (f SourceFunc) Load(ctx context.Context, tenant string) ([]Document, error) {
	return f(ctx, tenant)
}

// SourceSpec là cấu hình MỘT nguồn. Kind chọn chiến lược; mỗi chiến lược chỉ dùng các trường của nó.
type SourceSpec struct {
	Kind string `yaml:"kind"`
	// Path: thư mục (markdown) hoặc mẫu glob (files); đường dẫn tương đối tính theo thư mục của tệp cấu hình.
	Path string `yaml:"path"`
	// DSNEnv: TÊN biến môi trường chứa chuỗi kết nối (postgres) — không ghi chuỗi kết nối / mật khẩu vào cấu hình.
	DSNEnv string `yaml:"dsn_env"`
	Query  string `yaml:"query"`
	// Giá trị cho tài liệu mà nguồn không tự khai (files; cột thiếu của postgres).
	ACL           []string `yaml:"acl"`
	Version       int      `yaml:"version"`
	Status        string   `yaml:"status"`
	EffectiveFrom string   `yaml:"effective_from"`
	EffectiveTo   string   `yaml:"effective_to"`
	Scope         []string `yaml:"scope"`
}

// SourceFactory dựng chiến lược của một kind từ cấu hình; baseDir là thư mục của tệp cấu hình.
type SourceFactory func(spec SourceSpec, baseDir string) (Source, error)

// SourceKinds là bảng kind → chiến lược. Nơi dựng app / bộ đo ghép thêm chiến lược ở tầng khác (vd database ở platform/pgstore)
// bằng cách thêm khoá — không có nhánh if / switch theo loại nguồn.
type SourceKinds map[string]SourceFactory

// FileKinds là các chiến lược đọc tệp của gói này. pdfText đọc chữ của tệp PDF (vd ingestion.PdfToText); nil = không nhận .pdf.
func FileKinds(pdfText func(path string) (string, error)) SourceKinds {
	return SourceKinds{"markdown": markdownSource, "files": filesSource(pdfText)}
}

// LoadSourceSpecs đọc tệp cấu hình nguồn: {sources: [{kind: …, …}, …]}. Khoá lạ là lỗi.
func LoadSourceSpecs(path string) ([]SourceSpec, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg struct {
		Sources []SourceSpec `yaml:"sources"`
	}
	dec := yaml.NewDecoder(strings.NewReader(string(b)))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(cfg.Sources) == 0 {
		return nil, fmt.Errorf("%s: không có nguồn nào", path)
	}
	return cfg.Sources, nil
}

// Collect đọc mọi nguồn rồi kiểm CHUNG cho cả kho: mã@bản không trùng giữa các nguồn, một mã không vừa có bản không phiên bản vừa có
// bản khai phiên bản, mọi bản của một mã dùng CHUNG quyền (như acl.json — không thì người này nhận bản cũ, người kia bản mới, không
// ai thấy xung đột), trạng thái / ngày hợp lệ; rồi chạy cổng chất lượng đọc MỘT lần trên cả kho (mốc so cột có thể ở nguồn khác).
// Tài liệu bị cổng giữ lại vẫn được trả về (Status chờ soát, Hold nêu lý do) — nơi gọi phải báo chúng ra, không coi là đã nạp xong.
func Collect(ctx context.Context, specs []SourceSpec, kinds SourceKinds, tenant, baseDir string) ([]Document, error) {
	var out []Document
	from := map[string]string{} // "mã@bản" → nguồn
	versioned := map[string]bool{}
	acls := map[string][]string{} // mã → quyền (đã sắp) của bản gặp đầu tiên
	for i, spec := range specs {
		build, ok := kinds[spec.Kind]
		if !ok {
			return nil, fmt.Errorf("nguồn #%d: kind %q chưa hỗ trợ (có: %s)", i+1, spec.Kind, strings.Join(slices.Sorted(maps.Keys(kinds)), ", "))
		}
		src, err := build(spec, baseDir)
		if err != nil {
			return nil, fmt.Errorf("nguồn #%d (%s): %w", i+1, spec.Kind, err)
		}
		docs, err := src.Load(ctx, tenant)
		if err != nil {
			return nil, fmt.Errorf("nguồn #%d (%s): %w", i+1, spec.Kind, err)
		}
		where := fmt.Sprintf("nguồn #%d (%s)", i+1, spec.Kind)
		for _, d := range docs {
			if err := validDoc(d); err != nil {
				return nil, fmt.Errorf("%s: tài liệu %s: %w", where, d.ID, err)
			}
			key := fmt.Sprintf("%s@%d", d.ID, d.Version)
			if prev, dup := from[key]; dup {
				return nil, fmt.Errorf("%s và %s cùng có tài liệu %s bản %d", prev, where, d.ID, d.Version)
			}
			from[key] = where
			if legacy, ok := versioned[d.ID]; ok && legacy != (d.Version == 0) {
				return nil, fmt.Errorf("%s: tài liệu %s vừa có bản không phiên bản vừa có bản khai phiên bản", where, d.ID)
			}
			versioned[d.ID] = d.Version == 0
			acl := slices.Sorted(slices.Values(d.ACL))
			if prev, ok := acls[d.ID]; ok && !slices.Equal(prev, acl) {
				return nil, fmt.Errorf("%s: các bản của tài liệu %s khai quyền khác nhau (%v và %v) — mọi bản của một mã dùng chung quyền", where, d.ID, prev, acl)
			}
			acls[d.ID] = acl
			d.Tenant = tenant
			out = append(out, d)
		}
	}
	qualityGate(out)
	return out, nil
}

// validDoc áp đúng luật của phần đầu tệp markdown cho tài liệu từ mọi nguồn.
func validDoc(d Document) error {
	if strings.TrimSpace(d.ID) == "" {
		return fmt.Errorf("thiếu mã")
	}
	if d.Version < 0 {
		return fmt.Errorf("bản %d âm", d.Version)
	}
	switch d.Status {
	case "", StatusApproved, StatusDraft, StatusNeedsReview, StatusSuperseded:
	default:
		return fmt.Errorf("trạng thái %q không hợp lệ", d.Status)
	}
	if (d.Version == 0) != (d.Status == "") {
		return fmt.Errorf("bản %d với trạng thái %q: tài liệu khai phiên bản phải có trạng thái và ngược lại", d.Version, d.Status)
	}
	for _, s := range []string{d.EffectiveFrom, d.EffectiveTo} {
		if s != "" && !validDate(s) {
			return fmt.Errorf("ngày %q phải có dạng %s", s, dateLayout)
		}
	}
	if d.EffectiveFrom != "" && d.EffectiveTo != "" && d.EffectiveTo <= d.EffectiveFrom {
		return fmt.Errorf("hết hiệu lực (%s) không sau ngày bắt đầu (%s)", d.EffectiveTo, d.EffectiveFrom)
	}
	return nil
}

// markdownSource: thư mục theo đúng luật của LoadMarkdown (phần đầu + acl.json). Quyền / phiên bản thuộc về tệp, nên cấu hình khai
// thêm là lỗi (không lặng lẽ bỏ qua).
func markdownSource(spec SourceSpec, baseDir string) (Source, error) {
	if spec.Path == "" {
		return nil, fmt.Errorf("thiếu path")
	}
	if spec.hasDefaults() || spec.DSNEnv != "" || spec.Query != "" {
		return nil, fmt.Errorf("kind markdown lấy quyền / phiên bản từ acl.json và phần đầu tệp — bỏ các trường khác path")
	}
	dir := resolve(baseDir, spec.Path)
	// loadMarkdown (không cổng): Collect chạy cổng chất lượng MỘT lần trên cả kho.
	return SourceFunc(func(_ context.Context, tenant string) ([]Document, error) { return loadMarkdown(dir, tenant) }), nil
}

// filesSource: tệp import ngoài theo mẫu glob, mỗi tệp một tài liệu (mã = tên tệp). CHỈ tệp .md được khai phần đầu (mã / bản / trạng
// thái) như LoadMarkdown; chữ của .txt / .pdf / .csv là nội dung nguyên văn — một tệp xuất từ nơi khác không tự phong được mã và bản của
// tài liệu chính thức. Đọc chữ theo đuôi tệp qua bảng readers — đuôi không có trong bảng là lỗi. Quyền bắt buộc khai ở cấu hình.
func filesSource(pdfText func(string) (string, error)) SourceFactory {
	readText := func(p string) (string, error) {
		b, err := os.ReadFile(p)
		return string(b), err
	}
	readers := map[string]func(string) (string, error){".md": readText, ".txt": readText, ".csv": csvAsTable}
	if pdfText != nil {
		readers[".pdf"] = pdfText
	}
	return func(spec SourceSpec, baseDir string) (Source, error) {
		if spec.Path == "" || len(spec.ACL) == 0 {
			return nil, fmt.Errorf("kind files cần path (mẫu glob) và acl (vai được đọc)")
		}
		pattern := spec.Path
		if !filepath.IsAbs(pattern) { // thư mục cấu hình là chữ, không phải mẫu: "[", "*" trong tên thư mục không được khớp thư mục khác
			pattern = filepath.Join(globEscape.Replace(baseDir), pattern)
		}
		return SourceFunc(func(_ context.Context, tenant string) ([]Document, error) {
			paths, err := filepath.Glob(pattern)
			if err != nil {
				return nil, err
			}
			if len(paths) == 0 {
				return nil, fmt.Errorf("mẫu %s không khớp tệp nào", spec.Path)
			}
			slices.Sort(paths)
			out := make([]Document, 0, len(paths))
			for _, p := range paths {
				read, ok := readers[strings.ToLower(filepath.Ext(p))]
				if !ok {
					return nil, fmt.Errorf("%s: đuôi tệp chưa hỗ trợ (có: %s)", p, strings.Join(slices.Sorted(maps.Keys(readers)), ", "))
				}
				text, err := read(p)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", p, err)
				}
				d, fm, err := parseDocText(p, text, tenant, strings.EqualFold(filepath.Ext(p), ".md"))
				if err != nil {
					return nil, err
				}
				if !fm {
					d.Version, d.Status, d.EffectiveFrom, d.EffectiveTo, d.Scope = spec.Version, spec.Status, spec.EffectiveFrom, spec.EffectiveTo, spec.Scope
				}
				d.ACL = spec.ACL
				out = append(out, d)
			}
			return out, nil
		}), nil
	}
}

// csvAsTable đọc bảng tính CSV thành bảng markdown (dòng đầu = header): chỉ mục lặp header ở mỗi hàng, nên một hàng đứng một mình
// vẫn mang tên cột (hàng "Đào đất,150000" không thành số trơ trọi). Tiêu đề tài liệu = tên tệp. Hàng thiếu ô được đệm ô RỖNG — tệp khai
// bản "approved" sẽ bị cổng chất lượng giữ lại vì ô rỗng (đúng luật: ô trống có chủ đích ghi "-"). BOM của Excel bị bỏ; hàng chỉ có
// gạch / hai chấm là đường kẻ trang trí, bị bỏ (giữ thì nó thành dòng phân cách và hàng trước nó thành header của bảng giả).
func csvAsTable(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord, r.LazyQuotes = -1, true
	rows, err := r.ReadAll()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("# " + strings.TrimSuffix(filepath.Base(p), filepath.Ext(p)) + "\n")
	width := 0
	for _, row := range rows {
		width = max(width, len(row))
	}
	cell := strings.NewReplacer("|", "\\|", "\n", " ", "\r", " ")
	if len(rows) > 0 && len(rows[0]) > 0 {
		rows[0][0] = strings.TrimPrefix(rows[0][0], "\ufeff")
	}
	for i, row := range rows {
		if i > 0 && !slices.ContainsFunc(row, func(c string) bool { return strings.Trim(c, " \t:-") != "" }) {
			continue
		}
		cells := make([]string, width)
		for j := range cells {
			if j < len(row) {
				cells[j] = strings.TrimSpace(cell.Replace(row[j]))
			}
		}
		b.WriteString("| " + strings.Join(cells, " | ") + " |\n")
		if i == 0 {
			b.WriteString(strings.Repeat("|---", width) + "|\n")
		}
	}
	return b.String(), nil
}

func (s SourceSpec) hasDefaults() bool {
	return len(s.ACL) > 0 || s.Version != 0 || s.Status != "" || s.EffectiveFrom != "" || s.EffectiveTo != "" || len(s.Scope) > 0
}

// globEscape thoát ký tự mẫu của filepath.Match trong phần đường dẫn là chữ.
var globEscape = strings.NewReplacer(`\`, `\\`, `*`, `\*`, `?`, `\?`, `[`, `\[`)

func resolve(baseDir, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(baseDir, p)
}
