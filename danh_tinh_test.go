package eacl_test

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Cổng danh tính: repo chỉ có MỘT danh tính sản phẩm hiện hành là EACL (Enterprise AI Control Layer). Mẫu dưới đây
// (tên sản phẩm, tên repo, tiền tố định danh trước 21/10/2026) chỉ được xuất hiện ở các tệp có lý do ghi trong
// legacyAllowed, và không nhiều hơn số lần đã cho phép — bằng chứng đóng băng không được sửa cho đẹp tên, còn tài
// liệu / mã hiện hành không được để tên cũ quay lại.
var legacyIdentity = regexp.MustCompile(`(?i)eais|enterprise[- ]ai[- ]systems`)

type allowance struct {
	max    int
	reason string // HISTORICAL_EVIDENCE · IMMUTABLE_HASHED_ARTIFACT · COMPATIBILITY_ALIAS · EXTERNAL_REFERENCE · FALSE_POSITIVE
}

// legacyAllowed: tiền tố đường dẫn (thư mục kết thúc bằng "/") hoặc tệp → số lần tối đa (0 = không giới hạn) + lý do.
var legacyAllowed = map[string]allowance{
	"docs/bao-cao/":                               {0, "HISTORICAL_EVIDENCE: báo cáo theo ngày ghi đúng tên tại thời điểm đó"},
	"docs/g7/holdout/":                            {0, "IMMUTABLE_HASHED_ARTIFACT: bộ giữ lại và kết quả chạy đã đóng băng có băm"},
	"docs/g7/REVIEW-D5.md":                        {0, "HISTORICAL_EVIDENCE: bản soát độc lập 20/10"},
	"docs/g7/CONTAMINATION-AUDIT.md":              {0, "HISTORICAL_EVIDENCE: kiểm toán nhiễm 20/10"},
	"docs/ops/NHAT-KY-2026-10-20.md":              {0, "HISTORICAL_EVIDENCE: nhật ký vận hành 20/10"},
	"docs/requirements/":                          {0, "HISTORICAL_EVIDENCE + EXTERNAL_REFERENCE: sổ yêu cầu V1/V2 đông cứng; tên tệp brief nguồn nằm ngoài repo"},
	"docs/05-master-roadmap.md":                   {0, "HISTORICAL_EVIDENCE: roadmap V1/V2, ghi nhãn Historical artifact"},
	"docs/v3/00-CONTEXT-AUDIT.md":                 {0, "HISTORICAL_EVIDENCE: kiểm toán nguồn 20/10, ghi nhãn Historical artifact"},
	"CHANGELOG.md":                                {3, "HISTORICAL_EVIDENCE: mục 19–20/10 ghi module, GUC và vai lúc đó"},
	"docs/CHO-QUYET-DINH.md":                      {3, "HISTORICAL_EVIDENCE: văn bản quyết định D-V3-00, D-V3-10 của chủ dự án"},
	"docs/v3/06-MVP-TASK-BACKLOG.md":              {2, "FALSE_POSITIVE: mẫu grep trong tiêu chí nghiệm thu TASK-R0-03 / R0-05"},
	"intelligence/models/messages.go":             {2, "COMPATIBILITY_ALIAS: modelEnv đọc biến model cùng tên tiền tố cũ khi EACL_ không đặt"},
	"intelligence/models/messages_params_test.go": {9, "COMPATIBILITY_ALIAS: test của modelEnv"},
	"platform/pgstore/pgstore_test.go":            {3, "COMPATIBILITY_ALIAS: test PG báo lỗi thay vì bỏ qua khi chỉ còn DSN tiền tố cũ"},
	"scripts/dot-bien/n42-danh-tinh.sh":           {2, "FALSE_POSITIVE: chuỗi đột biến của chính mẫu cổng này"},
	"danh_tinh_test.go":                           {0, "FALSE_POSITIVE: chính mẫu của cổng này"},
}

// skipped: thư mục / tệp không thuộc repo (git bỏ qua) — nguồn riêng của chủ dự án, sổ ca ở máy.
func skipped(rel string, d fs.DirEntry) bool {
	if d.IsDir() {
		return rel == ".git" || rel == "docs/sources"
	}
	return strings.HasSuffix(rel, ".local.md")
}

// legacyHits trả các vi phạm của cổng danh tính dưới root: tệp ngoài danh sách cho phép mà có mẫu, hoặc tệp trong danh
// sách mà vượt số lần cho phép.
func legacyHits(t *testing.T, root string, allowed map[string]allowance) []string {
	t.Helper()
	var bad []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if skipped(rel, d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if bytes.IndexByte(b, 0) >= 0 { // tệp nhị phân
			return nil
		}
		n := len(legacyIdentity.FindAllIndex(b, -1))
		if n == 0 {
			return nil
		}
		for k, a := range allowed {
			if rel == k || (strings.HasSuffix(k, "/") && strings.HasPrefix(rel, k)) {
				if a.max > 0 && n > a.max {
					bad = append(bad, rel+": "+strconv.Itoa(n)+" lần, cho phép "+strconv.Itoa(a.max)+" ("+a.reason+")")
				}
				return nil
			}
		}
		bad = append(bad, rel+": "+strconv.Itoa(n)+" lần, không có lý do được ghi")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(bad)
	return bad
}

// Repo hiện tại: mọi lần xuất hiện tên cũ đều có lý do.
func TestDanhTinhChiConEACL(t *testing.T) {
	for _, v := range legacyHits(t, ".", legacyAllowed) {
		t.Error(v)
	}
}

// Ca âm: cổng bắt được tên cũ ở tệp hiện hành và tệp được phép mà vượt số lần.
func TestDanhTinhBatDuocTenCu(t *testing.T) {
	root := t.TempDir()
	for p, body := range map[string]string{
		"README.md":                          "# EAIS — Enterprise AI Systems\n",
		"docs/sources/brief.md":              "EAIS trong nguồn riêng: bỏ qua\n",
		"CHANGELOG.md":                       "EAIS EAIS EAIS\n",
		"docs/bao-cao/2026-10-20-ngay.md":    "EAIS lúc đó\n",
		"cmd/x/main.go":                      "package main // github.com/luannt2002/enterprise-ai-systems\n",
		"tenants/construction/pack/x.yaml":   "name: ok\n",
		"docs/sources/holdout-h3/kit/kiemca": "\x00EAIS",
		"CA-2026-10-21.local.md":             "sổ ca ở máy nhắc EAIS\n",
		"bin/cong-cu":                        "\x00EAIS nhị phân",
	} {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := strings.Join(legacyHits(t, root, map[string]allowance{"CHANGELOG.md": {2, "thử"}, "docs/bao-cao/": {0, "thử"}}), "\n")
	for _, want := range []string{"README.md: 2 lần", "CHANGELOG.md: 3 lần, cho phép 2", "cmd/x/main.go: 1 lần"} {
		if !strings.Contains(got, want) {
			t.Errorf("thiếu vi phạm %q trong:\n%s", want, got)
		}
	}
	for _, not := range []string{"docs/sources", "docs/bao-cao", "pack/x.yaml", ".local.md", "bin/cong-cu"} {
		if strings.Contains(got, not) {
			t.Errorf("báo oan %q:\n%s", not, got)
		}
	}
}
