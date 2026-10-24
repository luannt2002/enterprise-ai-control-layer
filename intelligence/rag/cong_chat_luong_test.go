package rag

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/unicode/norm"
)

func napThu(t *testing.T, files map[string]string) []Document {
	t.Helper()
	dir := t.TempDir()
	for f, s := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	docs, err := LoadMarkdown(dir, "a")
	if err != nil {
		t.Fatal(err)
	}
	return docs
}

func banSo(docs []Document, v int) Document {
	for _, d := range docs {
		if d.Version == v {
			return d
		}
	}
	return Document{}
}

const dauBan = "---\nid: SOP-NL-ĐM-01\nversion: %d\nstatus: approved\n---\n# Định mức nhiên liệu thiết bị\n"

// BR-DOC-03 (CON-T-032): bản mới khai "đã duyệt" mà bảng mất cột so với bản trước (parser làm rơi cột định mức) thì
// không được công bố — tự chuyển "chờ soát", ghi lý do; bản trước giữ nguyên.
func TestCongChatLuongBangMatCot(t *testing.T) {
	v2 := strings.Replace(dauBan, "%d", "2", 1) + "| Loại máy | Định mức (L/giờ máy) | Dung sai |\n|---|---|---|\n| Máy xúc | 16 | +15% |\n"
	v3 := strings.Replace(dauBan, "%d", "3", 1) + "| Loại máy | Dung sai |\n|---|---|\n| Máy xúc | +15% |\n"
	docs := napThu(t, map[string]string{"v2.md": v2, "v3.md": v3})
	if d := banSo(docs, 3); d.Status != StatusNeedsReview || !strings.Contains(d.Hold, "Định mức (L/giờ máy)") {
		t.Errorf("v3 mất cột phải chờ soát kèm lý do nêu cột, được %q / %q", d.Status, d.Hold)
	}
	if d := banSo(docs, 2); d.Status != StatusApproved || d.Hold != "" {
		t.Errorf("v2 không đổi, được %q / %q", d.Status, d.Hold)
	}
	// v4 cũng thiếu cột đó: phải so với v2 (bản đã duyệt gần nhất), không so với v3 đã bị giữ.
	v4 := strings.Replace(dauBan, "%d", "4", 1) + "| Loại máy | Dung sai |\n|---|---|\n| Máy xúc | +10% |\n"
	docs = napThu(t, map[string]string{"v2.md": v2, "v3.md": v3, "v4.md": v4})
	if d := banSo(docs, 4); d.Status != StatusNeedsReview {
		t.Errorf("v4 thiếu cột so với v2 phải chờ soát, được %q", d.Status)
	}
}

// BR-DOC-03: hàng thiếu ô so với header ("bảng đủ ô") cũng không được công bố.
func TestCongChatLuongHangThieuO(t *testing.T) {
	v1 := strings.Replace(dauBan, "%d", "1", 1) + "| Loại máy | Định mức (L/giờ máy) | Dung sai |\n|---|---|---|\n| Máy xúc | 17 |\n"
	if d := banSo(napThu(t, map[string]string{"v1.md": v1}), 1); d.Status != StatusNeedsReview || !strings.Contains(d.Hold, "thiếu ô") {
		t.Errorf("hàng thiếu ô phải chờ soát, được %q / %q", d.Status, d.Hold)
	}
}

// Bảng đủ ô, không mất cột; tài liệu không phiên bản; bản nháp: cổng không đụng tới.
func TestCongChatLuongKhongDungBanTot(t *testing.T) {
	v1 := strings.Replace(dauBan, "%d", "1", 1) + "| Loại máy | Định mức (L/giờ máy) |\n|---|---|\n| Máy xúc | 17 |\n"
	v2 := strings.Replace(dauBan, "%d", "2", 1) + "| Loại máy | Định mức (L/giờ máy) | Ghi chú |\n|---|---|---|\n| Máy xúc | 18 | mới |\n"
	nhap := "---\nid: SOP-Y\nversion: 1\nstatus: draft\n---\n# Y\n| A | B |\n|---|---|\n| 1 |\n"
	docs := napThu(t, map[string]string{"v1.md": v1, "v2.md": v2, "nhap.md": nhap, "SOP-CU.md": "# Cũ\n| A | B |\n|---|---|\n| 1 |\n"})
	for _, d := range docs {
		want := StatusApproved
		switch d.ID {
		case "SOP-Y":
			want = StatusDraft
		case "SOP-CU":
			want = ""
		}
		if d.Status != want || d.Hold != "" {
			t.Errorf("%s bản %d: muốn %q không lý do giữ, được %q / %q", d.ID, d.Version, want, d.Status, d.Hold)
		}
	}
}

func banPhanDau(v int, status, extra, body string) string {
	return fmt.Sprintf("---\nid: SOP-NL-ĐM-01\nversion: %d\nstatus: %s\n%s---\n# Định mức nhiên liệu thiết bị\n%s", v, status, extra, body)
}

const bang3 = "| Loại máy | Định mức (L/giờ máy) | Dung sai |\n|---|---|---|\n| Máy xúc | 16 | +15% |\n"
const bang2 = "| Loại máy | Dung sai |\n|---|---|\n| Máy xúc | +15% |\n"

// Soát U5a: mỗi dòng là một ca; muốn = trạng thái của bản cuối (bản số lớn nhất) sau khi nạp.
func TestCongChatLuongCacCaSoat(t *testing.T) {
	for name, tc := range map[string]struct {
		files map[string]string
		v     int
		want  string
	}{
		// G1: v2 đã chuyển "đã thay thế" đúng vòng đời; v3 mất cột vẫn phải bị giữ.
		"mốc là bản đã thay thế": {map[string]string{"a.md": banPhanDau(2, "superseded", "", bang3), "b.md": banPhanDau(3, "approved", "", bang2)}, 3, StatusNeedsReview},
		// G2: hai bảng cùng tên cột; bảng máy mất cột, bảng xe giữ nguyên.
		"mất cột ở một trong hai bảng": {map[string]string{
			"a.md": banPhanDau(2, "approved", "", bang3+"\n| Loại xe | Định mức (L/giờ máy) | Dung sai |\n|---|---|---|\n| Xe tải | 9 | +5% |\n"),
			"b.md": banPhanDau(3, "approved", "", bang2+"\n| Loại xe | Định mức (L/giờ máy) | Dung sai |\n|---|---|---|\n| Xe tải | 9 | +5% |\n")}, 3, StatusNeedsReview},
		// G3: "số đủ" — ô rỗng, hoặc bảng chỉ còn header.
		"ô rỗng":          {map[string]string{"a.md": banPhanDau(1, "approved", "", "| Loại máy | Định mức (L/giờ máy) | Dung sai |\n|---|---|---|\n| Máy xúc |  | +15% |\n")}, 1, StatusNeedsReview},
		"chỉ còn header":  {map[string]string{"a.md": banPhanDau(1, "approved", "", "| Loại máy | Định mức (L/giờ máy) | Dung sai |\n|---|---|---|\n")}, 1, StatusNeedsReview},
		"hàng thừa ô":     {map[string]string{"a.md": banPhanDau(1, "approved", "", "| Loại máy | Định mức |\n|---|---|\n| Máy xúc | 17 | dư |\n")}, 1, StatusNeedsReview},
		"thứ tự theo bản": {map[string]string{"dm-v9.md": banPhanDau(9, "approved", "", bang3), "dm-v10.md": banPhanDau(10, "approved", "", bang2)}, 10, StatusNeedsReview},
		// G6: mốc so theo mã + phạm vi; bản của dự án khác không phải bản trước.
		"khác phạm vi": {map[string]string{"a.md": banPhanDau(2, "approved", "scope: [PRJ-A]\n", bang3), "b.md": banPhanDau(3, "approved", "scope: [PRJ-B]\n", bang2)}, 3, StatusApproved},
		// G7: "\|" là chữ trong ô; tên cột NFD và NFC là một.
		"pipe có escape": {map[string]string{"a.md": banPhanDau(1, "approved", "", "| Ống | Số |\n|---|---|\n| D50 \\| D63 | 3 |\n")}, 1, StatusApproved},
		"tên cột NFD":    {map[string]string{"a.md": banPhanDau(2, "approved", "", bang3), "b.md": banPhanDau(3, "approved", "", norm.NFD.String(bang3))}, 3, StatusApproved},
		// G8: chủ tài liệu khai rõ cột bỏ có chủ đích.
		"bỏ cột có khai": {map[string]string{"a.md": banPhanDau(2, "approved", "", bang3), "b.md": banPhanDau(3, "approved", "removed_columns: [\"Định mức (L/giờ máy)\"]\n", bang2)}, 3, StatusApproved},
	} {
		d := banSo(napThu(t, tc.files), tc.v)
		if d.Status != tc.want {
			t.Errorf("%s: muốn %q, được %q (lý do %q)", name, tc.want, d.Status, d.Hold)
		}
	}
}

// G4: dòng phân cách phải đúng luật GFM; "| | |" là hàng (rỗng), không phải dòng phân cách.
func TestDongPhanCachDungLuat(t *testing.T) {
	for line, want := range map[string]bool{"|---|---|": true, "| :--- |\t---: |": true, "|:-:|": true, "| | |": false, "|": false, "||": false, "| a | --- |": false} {
		if isTableSeparator(line) != want {
			t.Errorf("isTableSeparator(%q) = %v, muốn %v", line, !want, want)
		}
	}
}

// G9: lý do giữ bị cắt gọn (không dài theo số hàng lỗi) và chỉ hàng bằng #vị trí như mã đoạn.
func TestLyDoGiuGon(t *testing.T) {
	body := "| A | B |\n|---|---|\n"
	for i := 0; i < 50; i++ {
		body += "| x |\n"
	}
	d := banSo(napThu(t, map[string]string{"a.md": banPhanDau(1, "approved", "", body)}), 1)
	if d.Status != StatusNeedsReview || len(d.Hold) > 300 || !strings.Contains(d.Hold, "#2") || !strings.Contains(d.Hold, "50") {
		t.Errorf("lý do phải gọn, nêu #2 và tổng 50: %q", d.Hold)
	}
}

// Bảng chỉ còn header (parser làm rơi cả thân bảng) phải được nói đúng tên lỗi cho chủ tài liệu, không lẫn với
// "hàng thiếu ô".
func TestBangChiConHeaderNoiDungLoi(t *testing.T) {
	d := banSo(napThu(t, map[string]string{"a.md": banPhanDau(1, "approved", "", "| Loại máy | Định mức (L/giờ máy) | Dung sai |\n|---|---|---|\n")}), 1)
	if d.Status != StatusNeedsReview || !strings.Contains(d.Hold, "chỉ có header") || strings.Contains(d.Hold, "thiếu ô") {
		t.Errorf("muốn lý do 'chỉ có header', được %q / %q", d.Status, d.Hold)
	}
}

// Soát vòng 2 (N1, G2, N2, G6, N4, G8): mỗi dòng là một ca; muốn = trạng thái của bản cuối sau khi nạp.
func TestCongChatLuongSoatVong2(t *testing.T) {
	hai := func(c string) string { // bảng 3 cột, 2 hàng
		return "| Loại máy | Định mức (L/giờ máy) | Dung sai |\n|---|---|---|\n| Máy xúc | 16 | +15% |\n| Máy ủi | 21 | +15% |\n" + c
	}
	xe := func(rows int) string {
		s := "\n| Loại xe | Định mức (L/giờ máy) | Dung sai |\n|---|---|---|\n"
		for i := 0; i < rows; i++ {
			s += "| Xe tải | 9 | +5% |\n"
		}
		return s
	}
	for name, tc := range map[string]struct {
		files map[string]string
		v     int
		want  string
	}{
		// N1: bớt một hàng, giữ đủ cột → không bị giữ oan.
		"bớt hàng": {map[string]string{"a.md": banPhanDau(2, "approved", "", hai("")), "b.md": banPhanDau(3, "approved", "", bang3)}, 3, StatusApproved},
		// G2: bảng máy mất cột, bảng xe thêm hàng — số hàng không được che việc mất cột.
		"thêm hàng che mất cột": {map[string]string{"a.md": banPhanDau(2, "approved", "", bang3+xe(1)), "b.md": banPhanDau(3, "approved", "", bang2+xe(2))}, 3, StatusNeedsReview},
		// N2: bản đã thay thế mà bảng hỏng không được làm mốc; mốc là bản tốt trước nó.
		"mốc hỏng bị bỏ qua": {map[string]string{"a.md": banPhanDau(1, "superseded", "", bang3), "b.md": banPhanDau(2, "superseded", "", bang2+"| Xe | |\n"), "c.md": banPhanDau(3, "approved", "", bang2)}, 3, StatusNeedsReview},
		// G6: thu hẹp phạm vi (không phạm vi → [PRJ-A]) vẫn phải so với bản trước.
		"thu hẹp phạm vi": {map[string]string{"a.md": banPhanDau(2, "superseded", "", bang3), "b.md": banPhanDau(3, "approved", "scope: [PRJ-A]\n", bang2)}, 3, StatusNeedsReview},
		// N4: hàng toàn dấu gạch ("không áp dụng") là hàng dữ liệu, không phải dòng phân cách.
		"hàng toàn gạch": {map[string]string{"a.md": banPhanDau(1, "approved", "", "| Loại máy | Định mức | Dung sai |\n|---|---|---|\n| Máy xúc | 17 | +15% |\n| - | - | - |\n| Máy ủi | 21 | +15% |\n")}, 1, StatusApproved},
		// G8: removed_columns khai tên không có ở bản trước → giữ lại để chủ tài liệu sửa, không im lặng.
		"khai bỏ cột sai tên":       {map[string]string{"a.md": banPhanDau(2, "approved", "", bang3), "b.md": banPhanDau(3, "approved", "removed_columns: [\"Định mức (L/giờ)\"]\n", bang2)}, 3, StatusNeedsReview},
		"khai bỏ cột không tồn tại": {map[string]string{"a.md": banPhanDau(2, "approved", "", bang3), "b.md": banPhanDau(3, "approved", "removed_columns: [\"Ghi chú\"]\n", bang3)}, 3, StatusNeedsReview},
	} {
		d := banSo(napThu(t, tc.files), tc.v)
		if d.Status != tc.want {
			t.Errorf("%s: muốn %q, được %q (lý do %q)", name, tc.want, d.Status, d.Hold)
		}
	}
}

// Soát vòng 2 (G4, N4): các dạng dòng phân cách GFM hợp lệ / không hợp lệ.
func TestDongPhanCachCacDang(t *testing.T) {
	for line, want := range map[string]bool{"|-|": true, "| ---- |": true, "|---|---": true, "|:-|-:|": true, "|:|": false, "|::|": false} {
		if isTableSeparator(line) != want {
			t.Errorf("isTableSeparator(%q) = %v, muốn %v", line, !want, want)
		}
	}
}
