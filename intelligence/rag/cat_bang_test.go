package rag

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

const bangDinhMuc = "Áp dụng cho mọi công trình.\n" +
	"| Loại máy | Định mức (L/giờ máy) | Dung sai |\n" +
	"|---|---|---|\n" +
	"| Máy xúc bánh xích 1,2 m³ | 17 | +15% |\n" +
	"| Máy ủi bánh xích | 21 | +15% |\n" +
	"Bảng rà soát mỗi năm."

// RAG-2 (CON-T-005): hàng bảng đứng một mình là số không đơn vị. Mỗi hàng thành một đoạn KÈM hàng header; dòng
// phân cách "|---|" không thành đoạn; mã đoạn trỏ đúng dòng của hàng trong tài liệu.
func TestCatBangLapHeader(t *testing.T) {
	ix := NewMemoryIndex("a", []Document{{ID: "SOP-DM", Tenant: "a", Title: "Định mức nhiên liệu", ACL: []string{"r"}, Body: bangDinhMuc}})
	ev, err := ix.Retrieve(context.Background(), principal("a", "r"), "định mức máy xúc bánh xích 1,2 m³", 3)
	if err != nil || len(ev) == 0 {
		t.Fatalf("muốn đoạn, được %v / %v", ev, err)
	}
	if !strings.Contains(ev[0].Text, "Định mức (L/giờ máy)") || !strings.Contains(ev[0].Text, "| Máy xúc bánh xích 1,2 m³ | 17 |") {
		t.Errorf("đoạn hàng phải mang header: %q", ev[0].Text)
	}
	if ev[0].ID != "SOP-DM#3" {
		t.Errorf("mã đoạn phải trỏ dòng của hàng (3), được %s", ev[0].ID)
	}
	for _, c := range ix.chunks {
		if strings.Contains(c.text, "|---|") {
			t.Errorf("dòng phân cách thành đoạn: %q", c.text)
		}
		if strings.Contains(c.text, "Máy xúc") && strings.Contains(c.text, "Máy ủi") {
			t.Errorf("hai hàng chung một đoạn: %q", c.text)
		}
	}
}

// Bất biến (nguồn C): mọi dòng có chữ của tài liệu, trừ dòng phân cách bảng, nằm trong ít nhất một đoạn — cắt không
// được làm mất chữ. Bảng chỉ có header thì header vẫn là một đoạn.
func TestCatBangPhuHetVanBan(t *testing.T) {
	for _, body := range []string{bangDinhMuc, "Mở đầu.\n| A | B |\n|---|---|\nKết.", "| A | B |"} {
		ix := NewMemoryIndex("a", []Document{{ID: "D", Tenant: "a", Title: "T", ACL: []string{"r"}, Body: body}})
		for _, line := range strings.Split(body, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || isTableSeparator(line) {
				continue
			}
			found := false
			for _, c := range ix.chunks {
				found = found || strings.Contains(c.text, line)
			}
			if !found {
				t.Errorf("mất dòng %q khi cắt %q", line, body)
			}
		}
	}
}

// Soát U3 (T4): ghim đúng danh sách đoạn (dòng, chữ) — dòng ngoài bảng giữ nguyên dạng cũ, không bị nuốt vào bảng.
func TestCatBangDanhSachDoanChinhXac(t *testing.T) {
	ix := NewMemoryIndex("a", []Document{{ID: "D", Tenant: "a", Title: "T", ACL: []string{"r"}, Body: bangDinhMuc}})
	want := []string{
		"0:Áp dụng cho mọi công trình.",
		"3:| Loại máy | Định mức (L/giờ máy) | Dung sai |\n| Máy xúc bánh xích 1,2 m³ | 17 | +15% |",
		"4:| Loại máy | Định mức (L/giờ máy) | Dung sai |\n| Máy ủi bánh xích | 21 | +15% |",
		"5:Bảng rà soát mỗi năm.",
	}
	var got []string
	for _, c := range ix.chunks {
		got = append(got, fmt.Sprintf("%d:%s", c.seq, c.text))
	}
	if strings.Join(got, "\n---\n") != strings.Join(want, "\n---\n") {
		t.Errorf("đoạn:\n%s\nmuốn:\n%s", strings.Join(got, "\n---\n"), strings.Join(want, "\n---\n"))
	}
}

// Soát U3 (T2, T3, T5): chỉ là bảng khi dòng sau là dòng phân cách; dòng phân cách đứng một mình bỏ đi; nhiều dòng
// "|" không có phân cách giữ cách cắt cũ; bảng thứ hai viết liền dưới bảng thứ nhất mang header của chính nó; dòng
// phân cách có căn lề, tab, thụt đầu dòng vẫn được nhận.
func TestCatBangCacDangLa(t *testing.T) {
	for name, tc := range map[string]struct{ body, want string }{
		"không phân cách": {"| ghi chú 1\n| ghi chú 2", "0:| ghi chú 1|1:| ghi chú 2"},
		"phân cách lẻ":    {"Mở đầu.\n|---|---|", "0:Mở đầu."},
		"hai bảng liền": {"| Loại máy | Định mức (L/giờ máy) |\n|---|---|\n| Máy xúc | 17 |\n| Loại xe | Định mức (L/100 km) |\n|---|---|\n| Xe tải 10 tấn | 30 |",
			"2:| Loại máy | Định mức (L/giờ máy) |\n| Máy xúc | 17 ||5:| Loại xe | Định mức (L/100 km) |\n| Xe tải 10 tấn | 30 |"},
		"căn lề, tab, thụt":   {"| A | B |\n| :--- |\t---: |\n  | 1 | 2 |", "2:| A | B |\n| 1 | 2 |"},
		"phân cách khác số ô": {"| A | B |\n|---|\n| 1 | 2 |", "0:| A | B ||2:| 1 | 2 |"},
	} {
		ix := NewMemoryIndex("a", []Document{{ID: "D", Tenant: "a", Title: "T", ACL: []string{"r"}, Body: tc.body}})
		var got []string
		for _, c := range ix.chunks {
			got = append(got, fmt.Sprintf("%d:%s", c.seq, c.text))
		}
		if g := strings.Join(got, "|"); g != tc.want {
			t.Errorf("%s:\n được %q\n muốn %q", name, g, tc.want)
		}
	}
}

// Soát U3 (T1): từ của header nằm trong CHỮ của đoạn hàng nhưng không cộng điểm cho mọi hàng — nếu không, câu hỏi
// chạm từ của header kéo mọi hàng của bảng vượt ngưỡng (CON-T-004: hàng "Xe lu" lọt vào ngữ cảnh).
func TestCatBangHeaderKhongDayDiemMoiHang(t *testing.T) {
	ix := NewMemoryIndex("a", []Document{{ID: "D", Tenant: "a", Title: "Định mức nhiên liệu thiết bị", ACL: []string{"r"},
		Body: "| Loại máy | Định mức (L/giờ máy) | Dung sai |\n|---|---|---|\n| Máy xúc bánh xích 1,2 m³ | 17 | +15% |\n| Máy ủi bánh xích | 21 | +15% |\n| Xe lu rung | 12 | +10% |"}})
	ev, _ := ix.Retrieve(context.Background(), principal("a", "r"), "Máy xúc EX-01 hôm nay tiêu thụ dầu có vượt định mức không?", 3)
	for _, e := range ev {
		if strings.Contains(e.Text, "Xe lu") {
			t.Errorf("hàng không chung từ nào với câu hỏi (ngoài tiêu đề) vẫn lọt: %q", e.Text)
		}
	}
}
