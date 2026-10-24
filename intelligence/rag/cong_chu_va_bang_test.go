package rag

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tệp ra 0 chữ (PDF scan chưa có lớp chữ, tệp rỗng, bản chỉ có tiêu đề — F-023) không nạp im lặng: thân không có chữ thì không đoạn
// nào vào chỉ mục, tài liệu biến mất mà không ai biết. Cổng giữ lại kèm lý do — cả tài liệu không phiên bản của nguồn files; tệp có
// chữ thì không bị đụng.
func TestTepKhongChuBiGiu(t *testing.T) {
	dir := t.TempDir()
	viet := func(rel, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	viet("scan.pdf", "giả")
	viet("co-chu.pdf", "giả")
	viet("rong.txt", "")
	viet("chi-tieu-de.md", "---\nid: QT-RONG\nversion: 1\nstatus: approved\n---\n# Quy trình chưa viết\n\n")
	pdf := func(p string) (string, error) {
		if strings.HasSuffix(p, "scan.pdf") {
			return "\f\f\f", nil // pdftotext trên trang ảnh: chỉ còn dấu ngắt trang
		}
		return "Quy chế vật tư\nĐề nghị mua vật tư phải có báo giá.\n", nil
	}
	docs, err := Collect(context.Background(), []SourceSpec{{Kind: "files", Path: "*", ACL: []string{"ks"}}}, FileKinds(pdf), "t", dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range docs {
		giu := d.Status == StatusNeedsReview && strings.Contains(d.Hold, "không có chữ")
		if want := d.ID != "co-chu"; giu != want {
			t.Errorf("%s: muốn bị giữ = %v, được %q / %q", d.ID, want, d.Status, d.Hold)
		}
	}
	if len(docs) != 4 {
		t.Fatalf("muốn 4 tài liệu, được %d", len(docs))
	}
}

// Bảng mất dòng phân cách (hoặc mất header) ở bản ĐẦU TIÊN — chưa có mốc để so cột (F-024): các dòng "|" thành dòng thường, hàng số
// không tên cột được phục vụ (CON-T-005). Cổng giữ lại kèm lý do; bảng lành không bị đụng.
func TestBangMatPhanCachBanDauBiGiu(t *testing.T) {
	ban := func(id, body string) string {
		return "---\nid: " + id + "\nversion: 1\nstatus: approved\n---\n# Định mức " + id + "\n" + body
	}
	docs := napThu(t, map[string]string{
		"mat-phan-cach.md": ban("MAT-PC", "| Loại máy | Định mức |\n| Máy ủi | 21 |\n"),
		"mat-header.md":    ban("MAT-HD", "|---|---|\n| Máy ủi | 21 |\n"),
		"lanh.md":          ban("LANH", "| Loại máy | Định mức |\n|---|---|\n| Máy ủi | 21 |\n"),
	})
	for _, d := range docs {
		giu := d.Status == StatusNeedsReview && strings.Contains(d.Hold, "không thuộc bảng nào")
		if want := d.ID != "LANH"; giu != want {
			t.Errorf("%s: muốn bị giữ = %v, được %q / %q", d.ID, want, d.Status, d.Hold)
		}
	}
}
