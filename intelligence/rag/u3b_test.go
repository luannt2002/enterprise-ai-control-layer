package rag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Thân có cấu trúc mục (dòng tiêu đề markdown) và danh sách có câu dẫn — dạng văn bản quy định thật (U3b, docs/ROADMAP-RAG.md §6.3).
const thanQuyDinh = `## Chương I
QUY ĐỊNH CHUNG
#### Điều 2. Giải thích từ ngữ
Trong quy chế này, các từ ngữ dưới đây được hiểu như sau:
1. Tường lửa là tập hợp thiết bị kiểm soát luồng mạng.
2. Thiết bị di động là thiết bị số có thể mang theo người.
## Chương II
#### Điều 18. Yêu cầu đối với trung tâm dữ liệu
1. Cổng ra vào tòa nhà có người kiểm soát suốt ngày đêm.
2. Hệ thống cấp độ 2 là hệ thống có một trong các tiêu chí sau:
a) Phục vụ hoạt động nội bộ của đơn vị;
b) Phục vụ khách hàng không yêu cầu vận hành liên tục.
3. Dữ liệu camera lưu tối thiểu chín mươi ngày.
4. Kiểm tra định kỳ như sau:
5. Không có mục con nên câu dẫn ở trên đứng một mình.`

func chiMucQuyDinh(t *testing.T, body string) *MemoryIndex {
	t.Helper()
	ix := NewMemoryIndex("t", []Document{{ID: "QC", Tenant: "t", Title: "Quy chế an toàn", ACL: []string{"ks"}, Body: body}})
	ix.Now = func() time.Time { return time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC) }
	return ix
}

func TestDuongDanMucVaGomDanhSach(t *testing.T) {
	ix := chiMucQuyDinh(t, thanQuyDinh)
	by := map[int]string{}
	for _, c := range ix.chunks {
		by[c.seq] = c.text
		if strings.HasPrefix(strings.TrimSpace(strings.SplitN(c.text, "\n", 2)[len(strings.SplitN(c.text, "\n", 2))-1]), "#") {
			t.Errorf("dòng tiêu đề không được thành đoạn riêng: %q", c.text)
		}
	}
	for seq, want := range map[int]string{
		1:  "Chương I\nQUY ĐỊNH CHUNG",
		3:  "Chương I › Điều 2. Giải thích từ ngữ\nTrong quy chế này, các từ ngữ dưới đây được hiểu như sau:\n1. Tường lửa là tập hợp thiết bị kiểm soát luồng mạng.\n2. Thiết bị di động là thiết bị số có thể mang theo người.",
		8:  "Chương II › Điều 18. Yêu cầu đối với trung tâm dữ liệu\n1. Cổng ra vào tòa nhà có người kiểm soát suốt ngày đêm.",
		9:  "Chương II › Điều 18. Yêu cầu đối với trung tâm dữ liệu\n2. Hệ thống cấp độ 2 là hệ thống có một trong các tiêu chí sau:\na) Phục vụ hoạt động nội bộ của đơn vị;\nb) Phục vụ khách hàng không yêu cầu vận hành liên tục.",
		12: "Chương II › Điều 18. Yêu cầu đối với trung tâm dữ liệu\n3. Dữ liệu camera lưu tối thiểu chín mươi ngày.",
		13: "Chương II › Điều 18. Yêu cầu đối với trung tâm dữ liệu\n4. Kiểm tra định kỳ như sau:", // câu dẫn không có mục con SÂU hơn: đứng một mình
		14: "Chương II › Điều 18. Yêu cầu đối với trung tâm dữ liệu\n5. Không có mục con nên câu dẫn ở trên đứng một mình.",
	} {
		if by[seq] != want {
			t.Errorf("đoạn #%d:\nđược %q\nmuốn %q", seq, by[seq], want)
		}
	}
	if len(ix.chunks) != 7 {
		t.Errorf("muốn 7 đoạn, có %d", len(ix.chunks))
	}

	// Hỏi theo cấu trúc văn bản: khoản của Điều 18 không có chữ "Điều 18" nhưng đường dẫn có.
	ctx := context.Background()
	pr := contracts.Principal{ID: "u", TenantID: "t", Kind: contracts.PrincipalUser, Roles: []string{"ks"}}
	ev, err := ix.Retrieve(ctx, pr, "Điều 18 quy định gì?", 3)
	if err != nil || len(ev) == 0 || !strings.Contains(ev[0].Text, "Điều 18") {
		t.Errorf("hỏi Điều 18: %v %v", ids(ev), err)
	}
	// Câu liệt kê: MỘT đoạn mang đủ câu dẫn + các tiêu chí.
	ev, err = ix.Retrieve(ctx, pr, "Hệ thống cấp độ 2 gồm những tiêu chí nào?", 3)
	if err != nil || len(ev) == 0 || ev[0].ID != "QC#9" || !strings.Contains(ev[0].Text, "a) ") || !strings.Contains(ev[0].Text, "b) ") {
		t.Errorf("câu liệt kê phải trúng đoạn câu dẫn + a), b): %v %v", ids(ev), err)
	}
}

// Danh sách dài vượt trần: tách thành nhiều đoạn, đoạn nào cũng mở bằng câu dẫn; mã đoạn không trùng.
func TestGomDanhSachTachTheoTran(t *testing.T) {
	var b strings.Builder
	b.WriteString("#### Điều 6. Nội dung quy chế\nQuy chế gồm các nội dung sau:\n")
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&b, "%c) %s\n", 'a'+rune(i%14), strings.Repeat("nội dung dài về quản lý an toàn ", 4))
	}
	ix := chiMucQuyDinh(t, b.String())
	seen := map[int]bool{}
	for _, c := range ix.chunks {
		if seen[c.seq] {
			t.Errorf("trùng vị trí đoạn %d", c.seq)
		}
		seen[c.seq] = true
		if !strings.HasPrefix(c.text, "Điều 6. Nội dung quy chế\nQuy chế gồm các nội dung sau:\n") || len([]rune(c.text)) > maxGroup+200 {
			t.Errorf("đoạn tách phải mở bằng đường dẫn + câu dẫn và không vượt trần nhiều: %d ký tự, %q…", len([]rune(c.text)), c.text[:60])
		}
	}
	if len(ix.chunks) < 2 {
		t.Errorf("danh sách vượt trần phải tách, có %d đoạn", len(ix.chunks))
	}
}

// Luật tiêu đề ở tầng nguồn: dòng khớp mẫu thành tiêu đề markdown đúng cấp (số dòng không đổi); mẫu hỏng / cấp ngoài 1..5 là lỗi;
// không áp cho bảng tính.
func TestLuatTieuDeCuaNguon(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "qc.txt"), []byte("Quy chế an toàn\nChương I\nĐiều 1. Phạm vi\n1. Quy chế áp dụng cho mọi đơn vị.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	luat := []HeadingRule{{Mau: `^Chương [IVXLC]+$`, Cap: 1}, {Mau: `^Điều \d+\.`, Cap: 3}}
	docs, err := Collect(context.Background(), []SourceSpec{{Kind: "files", Path: "*.txt", ACL: []string{"ks"}, TieuDe: luat}}, FileKinds(nil), "t", dir)
	if err != nil {
		t.Fatal(err)
	}
	if docs[0].Body != "## Chương I\n#### Điều 1. Phạm vi\n1. Quy chế áp dụng cho mọi đơn vị.\n" {
		t.Errorf("thân sau luật tiêu đề: %q", docs[0].Body)
	}
	ix := NewMemoryIndex("t", docs)
	if len(ix.chunks) != 1 || ix.chunks[0].text != "Chương I › Điều 1. Phạm vi\n1. Quy chế áp dụng cho mọi đơn vị." || ix.chunks[0].seq != 2 {
		t.Errorf("đoạn sau luật tiêu đề: %+v", ix.chunks)
	}
	for name, l := range map[string][]HeadingRule{"mẫu hỏng": {{Mau: `(`, Cap: 1}}, "cấp 0": {{Mau: `^Điều`, Cap: 0}}, "cấp 6": {{Mau: `^Điều`, Cap: 6}}, "mẫu rỗng": {{Mau: ``, Cap: 1}}} {
		if _, err := Collect(context.Background(), []SourceSpec{{Kind: "files", Path: "*.txt", ACL: []string{"ks"}, TieuDe: l}}, FileKinds(nil), "t", dir); err == nil {
			t.Errorf("%s: muốn lỗi cấu hình", name)
		}
	}
}

// Tiêu đề không có nội dung bên dưới (tới tiêu đề cùng cấp / cao hơn, hoặc hết thân) thì chính nó là đoạn — "Điều 10. (Đã bãi bỏ)"
// không được biến mất khỏi chỉ mục; tiêu đề cha có mục con thì chỉ nằm trong đường dẫn. Bất biến: mọi tiêu đề có mặt trong ít nhất
// một đoạn.
func TestTieuDeTrongVanLaDoan(t *testing.T) {
	body := "## Chương I\n#### Điều 9. Phạm vi\n1. Nội dung.\n#### Điều 10. (Đã bãi bỏ)\n#### Điều 11. Hiệu lực\n1. Có hiệu lực.\n" +
		"## Chương II\n#### Điều 12. (Đã bãi bỏ)"
	ix := chiMucQuyDinh(t, body)
	var got []string
	for _, c := range ix.chunks {
		got = append(got, c.text)
	}
	want := []string{"Chương I › Điều 9. Phạm vi\n1. Nội dung.", "Chương I\nĐiều 10. (Đã bãi bỏ)", "Chương I › Điều 11. Hiệu lực\n1. Có hiệu lực.",
		"Chương II\nĐiều 12. (Đã bãi bỏ)"}
	if !slices.Equal(got, want) {
		t.Errorf("đoạn:\nđược %q\nmuốn %q", got, want)
	}
	for _, h := range []string{"Chương I", "Chương II", "Điều 9. Phạm vi", "Điều 10. (Đã bãi bỏ)", "Điều 11. Hiệu lực", "Điều 12. (Đã bãi bỏ)"} {
		if !slices.ContainsFunc(got, func(s string) bool { return strings.Contains(s, h) }) {
			t.Errorf("tiêu đề %q biến mất khỏi chỉ mục", h)
		}
	}
}

// Soát độc lập 23/10: danh sách lồng ở lại cùng câu dẫn; tiêu đề chỉ có bảng bên dưới vẫn tìm được bằng chữ của nó; la mã từ (xi);
// "+" sâu hơn "-"; luật tiêu đề không chạm hàng bảng.
func TestGomDanhSachLongVaTieuDeTrenBang(t *testing.T) {
	texts := func(body string) []string {
		var out []string
		for _, c := range chiMucQuyDinh(t, body).chunks {
			out = append(out, c.text)
		}
		return out
	}
	for name, c := range map[string]struct{ body, want string }{
		"danh sách lồng": {"Các hành vi bị nghiêm cấm gồm:\n1. Hành vi A gồm:\na) A1;\nb) A2.\n2. Hành vi B.",
			"Các hành vi bị nghiêm cấm gồm:\n1. Hành vi A gồm:\na) A1;\nb) A2.\n2. Hành vi B."},
		"la mã từ (xi)":     {"Gồm:\n(i) một\n(ii) hai\n(ix) chín\n(x) mười\n(xi) mười một\n(xii) mười hai", "Gồm:\n(i) một\n(ii) hai\n(ix) chín\n(x) mười\n(xi) mười một\n(xii) mười hai"},
		"cộng sâu hơn gạch": {"- Nhóm A gồm:\n+ a1\n+ a2", "- Nhóm A gồm:\n+ a1\n+ a2"},
	} {
		if got := texts(c.body); len(got) != 1 || got[0] != c.want {
			t.Errorf("%s: muốn MỘT đoạn %q, được %q", name, c.want, got)
		}
	}
	// Tách theo trần chỉ ở mục con cấp đầu: nhóm con (câu dẫn "1. … gồm:" + a), b)) không bị cắt đôi.
	var b strings.Builder
	b.WriteString("Quy chế gồm:\n")
	for i := 1; i <= 6; i++ {
		fmt.Fprintf(&b, "%d. Nội dung %d gồm:\na) %s\nb) %s\n", i, i, strings.Repeat("chi tiết dài ", 12), strings.Repeat("chi tiết dài ", 12))
	}
	for _, s := range texts(b.String()) {
		if !strings.HasPrefix(s, "Quy chế gồm:\n") || strings.HasPrefix(strings.SplitN(s, "\n", 3)[1], "a)") || strings.HasPrefix(strings.SplitN(s, "\n", 3)[1], "b)") {
			t.Errorf("đoạn tách phải mở bằng câu dẫn rồi tới mục cấp đầu: %q", s[:min(len(s), 80)])
		}
	}

	ix := chiMucQuyDinh(t, "## Phụ lục 2. Biểu phí dịch vụ\n| Dịch vụ | Phí |\n|---|---|\n| Cấp lại thẻ | 50.000 |")
	pr := contracts.Principal{ID: "u", TenantID: "t", Kind: contracts.PrincipalUser, Roles: []string{"ks"}}
	ev, err := ix.Retrieve(context.Background(), pr, "Phụ lục 2 biểu phí gồm gì?", 3)
	if err != nil || len(ev) == 0 {
		t.Errorf("tiêu đề chỉ có bảng bên dưới phải tìm được bằng chữ của nó: %v %v", ids(ev), err)
	}
	if got := texts("## Phụ lục 2. Biểu phí dịch vụ\n| Dịch vụ | Phí |\n|---|---|\n| Cấp lại thẻ | 50.000 |"); len(got) != 2 || got[0] != "Phụ lục 2. Biểu phí dịch vụ" {
		t.Errorf("tiêu đề trên bảng: đoạn riêng đứng TRƯỚC hàng bảng, được %q", got)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bp.txt"), []byte("Biểu phí\n| Mục | Điều 5 |\n|---|---|\n| a | Điều 6 áp dụng |\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	docs, err := Collect(context.Background(), []SourceSpec{{Kind: "files", Path: "*.txt", ACL: []string{"ks"}, TieuDe: []HeadingRule{{Mau: `Điều \d+`, Cap: 3}}}}, FileKinds(nil), "t", dir)
	if err != nil || docs[0].Body != "| Mục | Điều 5 |\n|---|---|\n| a | Điều 6 áp dụng |\n" {
		t.Errorf("luật tiêu đề không được chạm hàng bảng: %q %v", docs[0].Body, err)
	}
}
