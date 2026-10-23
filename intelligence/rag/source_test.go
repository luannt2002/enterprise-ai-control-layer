package rag

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Nguồn tri thức theo strategy: kind chọn chiến lược từ bảng, nhiều nguồn ghép vào MỘT kho và kiểm chung (trùng mã@bản giữa hai nguồn,
// trạng thái / ngày, cổng chất lượng xuyên nguồn); thiếu quyền, kind lạ, đuôi tệp lạ là lỗi — không lặng lẽ bỏ qua hay mặc định
// "ai cũng đọc".
func TestNguonTriThuc(t *testing.T) {
	dir := t.TempDir()
	viet := func(rel, body string) {
		t.Helper()
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	viet("md/qt.md", "---\nid: QT\nversion: 1\nstatus: approved\n---\n# Quy trình nghiệm thu\n| Hạng mục | Thời hạn |\n|---|---|\n| Biên bản | 24 giờ |\n")
	viet("md/acl.json", `{"QT": ["ks"]}`)
	viet("nhap/qt-v2.md", "---\nid: QT\nversion: 2\nstatus: approved\n---\n# Quy trình nghiệm thu\n| Hạng mục |\n|---|\n| Biên bản |\n")
	viet("nhap/so-tay.txt", "Sổ tay an toàn\nĐội mũ bảo hộ khi vào công trường.\n")
	viet("nhap/quy-che.pdf", "giả")
	pdf := func(p string) (string, error) {
		return "Quy chế vật tư\nĐề nghị mua vật tư phải có báo giá.\n", nil
	}
	ctx := context.Background()
	specs := []SourceSpec{
		{Kind: "markdown", Path: "md"},
		{Kind: "files", Path: "nhap/*", ACL: []string{"ks"}},
	}
	docs, err := Collect(ctx, specs, FileKinds(pdf), "t", dir)
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]Document{}
	for _, d := range docs {
		by[d.ID+"@"+string(rune('0'+d.Version))] = d
		if d.Tenant != "t" || len(d.ACL) == 0 {
			t.Errorf("%s: tenant %q, ACL %v", d.ID, d.Tenant, d.ACL)
		}
	}
	if len(docs) != 4 || by["so-tay@0"].Title != "Sổ tay an toàn" || !strings.Contains(by["quy-che@0"].Body, "báo giá") {
		t.Fatalf("kho ghép sai: %+v", docs)
	}
	// Cổng chất lượng chạy trên CẢ kho: bản 2 (nguồn files) mất cột so với bản 1 (nguồn markdown) ⇒ chờ soát.
	if v2 := by["QT@2"]; v2.Status != StatusNeedsReview || !strings.Contains(v2.Hold, "mất cột") {
		t.Errorf("cổng chất lượng xuyên nguồn: QT@2 = %s %q", v2.Status, v2.Hold)
	}

	for name, c := range map[string]struct {
		specs []SourceSpec
		want  string
	}{
		"kind lạ":                 {[]SourceSpec{{Kind: "s3", Path: "x"}}, "chưa hỗ trợ (có: files, markdown)"},
		"files thiếu quyền":       {[]SourceSpec{{Kind: "files", Path: "nhap/*.txt"}}, "acl"},
		"markdown khai quyền":     {[]SourceSpec{{Kind: "markdown", Path: "md", ACL: []string{"ks"}}}, "acl.json"},
		"không khớp tệp nào":      {[]SourceSpec{{Kind: "files", Path: "khong/*.md", ACL: []string{"ks"}}}, "không khớp"},
		"đuôi tệp lạ":             {[]SourceSpec{{Kind: "files", Path: "md/*.json", ACL: []string{"ks"}}}, "đuôi tệp chưa hỗ trợ"},
		"trùng mã@bản 2 nguồn":    {[]SourceSpec{{Kind: "markdown", Path: "md"}, {Kind: "files", Path: "md/qt.md", ACL: []string{"ks"}}}, "cùng có tài liệu QT bản 1"},
		"bản không có trạng thái": {[]SourceSpec{{Kind: "files", Path: "nhap/*.txt", ACL: []string{"ks"}, Version: 1}}, "phải có trạng thái"},
		"ngày sai dạng":           {[]SourceSpec{{Kind: "files", Path: "nhap/*.txt", ACL: []string{"ks"}, EffectiveFrom: "01/01/2026"}}, "phải có dạng"},
	} {
		if _, err := Collect(ctx, c.specs, FileKinds(pdf), "t", dir); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: muốn lỗi chứa %q, được %v", name, c.want, err)
		}
	}
	if _, err := Collect(ctx, []SourceSpec{{Kind: "files", Path: "nhap/*.pdf", ACL: []string{"ks"}}}, FileKinds(nil), "t", dir); err == nil {
		t.Error("không có bộ đọc PDF mà nhận tệp .pdf")
	}

	// CSV (bảng tính) → bảng markdown: mỗi hàng là một đoạn MANG header; ô có "|" không vỡ bảng; hàng thiếu ô được đệm.
	// BOM của Excel bị bỏ; hàng chỉ có gạch là đường kẻ trang trí, bị bỏ (không thành dòng phân cách của bảng giả).
	viet("bang/gia.csv", "\ufeffHạng mục,Đơn giá (VND),Ghi chú\nĐào đất,150000,m3 | thủ công\n---,---,---\nĐắp cát,90000\n")
	docs, err = Collect(ctx, []SourceSpec{{Kind: "files", Path: "bang/*.csv", ACL: []string{"ks"}}}, FileKinds(nil), "t", dir)
	if err != nil || len(docs) != 1 {
		t.Fatalf("csv: %+v %v", docs, err)
	}
	ix := NewMemoryIndex("t", docs)
	if len(ix.chunks) != 2 || ix.chunks[0].text != "| Hạng mục | Đơn giá (VND) | Ghi chú |\n| Đào đất | 150000 | m3 \\| thủ công |" ||
		ix.chunks[1].text != "| Hạng mục | Đơn giá (VND) | Ghi chú |\n| Đắp cát | 90000 |  |" || docs[0].Title != "gia" {
		t.Errorf("csv thành bảng: tiêu đề %q, đoạn %q", docs[0].Title, func() (s []string) {
			for _, c := range ix.chunks {
				s = append(s, c.text)
			}
			return
		}())
	}

	// Tệp .txt mở đầu bằng "---" không được tự khai mã / bản / trạng thái: chỉ .md có phần đầu.
	viet("txt/gia-mao.txt", "---\nid: QT\nversion: 9\nstatus: approved\n---\nQuy trình nghiệm thu\nBiên bản trong 72 giờ.\n")
	if docs, err := Collect(ctx, []SourceSpec{{Kind: "files", Path: "txt/*.txt", ACL: []string{"kho"}}}, FileKinds(nil), "t", dir); err != nil ||
		len(docs) != 1 || docs[0].ID != "gia-mao" || docs[0].Version != 0 || docs[0].Status != "" {
		t.Errorf("txt tự khai phần đầu phải bị bỏ qua: %+v %v", docs, err)
	}
	// Mọi bản của một mã dùng chung quyền (như acl.json): bản 9 khai quyền khác bản 1 ⇒ lỗi, không lặng lẽ cho mỗi vai một bản.
	viet("md9/qt-v9.md", "---\nid: QT\nversion: 9\nstatus: approved\n---\n# Quy trình nghiệm thu\nBiên bản trong 72 giờ.\n")
	if _, err := Collect(ctx, []SourceSpec{{Kind: "markdown", Path: "md"}, {Kind: "files", Path: "md9/*.md", ACL: []string{"kho"}}}, FileKinds(nil), "t", dir); err == nil ||
		!strings.Contains(err.Error(), "khai quyền khác nhau") {
		t.Errorf("các bản cùng mã khác quyền: muốn lỗi, được %v", err)
	}
	// Thư mục cấu hình có ký tự mẫu ("[", "]") là chữ: không được khớp sang thư mục bên cạnh.
	viet("cfg[1]/nhap/dung.txt", "Đúng thư mục\nnội dung.\n")
	viet("cfg1/nhap/la.txt", "Thư mục lạ\nnội dung.\n")
	if docs, err := Collect(ctx, []SourceSpec{{Kind: "files", Path: "nhap/*", ACL: []string{"ks"}}}, FileKinds(nil), "t", filepath.Join(dir, "cfg[1]")); err != nil ||
		len(docs) != 1 || docs[0].ID != "dung" {
		t.Errorf("ký tự mẫu trong thư mục cấu hình: %+v %v", docs, err)
	}
	// Cổng chất lượng chạy MỘT lần trên cả kho: bản 2 (nguồn khác) bỏ cột C có chủ đích, bản 3 so với bản 2 nên được công bố.
	viet("gate/qt1.md", "---\nid: QG\nversion: 1\nstatus: approved\n---\n# Q\n| A | B | C |\n|---|---|---|\n| 1 | 2 | 3 |\n")
	viet("gate/qt3.md", "---\nid: QG\nversion: 3\nstatus: approved\n---\n# Q\n| A | B |\n|---|---|\n| 1 | 2 |\n")
	viet("gate/acl.json", `{"QG": ["ks"]}`)
	viet("gate2/qt2.md", "---\nid: QG\nversion: 2\nstatus: approved\nremoved_columns: [C]\n---\n# Q\n| A | B |\n|---|---|\n| 1 | 2 |\n")
	docs, err = Collect(ctx, []SourceSpec{{Kind: "markdown", Path: "gate"}, {Kind: "files", Path: "gate2/*.md", ACL: []string{"ks"}}}, FileKinds(nil), "t", dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range docs {
		if d.ID == "QG" && d.Status != StatusApproved {
			t.Errorf("cổng chạy hai lần: QG@%d = %s %q", d.Version, d.Status, d.Hold)
		}
	}

	viet("nguon.yaml", "sources:\n  - kind: markdown\n    path: md\n")
	if s, err := LoadSourceSpecs(filepath.Join(dir, "nguon.yaml")); err != nil || len(s) != 1 || s[0].Kind != "markdown" {
		t.Errorf("đọc cấu hình nguồn: %+v %v", s, err)
	}
	viet("sai.yaml", "sources:\n  - kind: markdown\n    duong_dan: md\n")
	if _, err := LoadSourceSpecs(filepath.Join(dir, "sai.yaml")); err == nil {
		t.Error("khoá lạ trong cấu hình nguồn phải là lỗi")
	}
}
