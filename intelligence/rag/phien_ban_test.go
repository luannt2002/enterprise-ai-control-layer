package rag

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Bảng định mức của SOP-NL-ĐM-01 theo các ca CON-T-004/005/032/061 (tenants/construction/evals/cases.yaml).
// Số của máy xúc 1,2 m³ lấy đúng từ ca; số của bản v2 và của máy ủi không có trong ca — đặt khác đi để
// bắt được trích nhầm (giả lập).
func dinhMuc(id string, version int, status, from, to, mayXuc string) Document {
	return Document{ID: id, Tenant: "a", Title: "Định mức nhiên liệu thiết bị", ACL: []string{"site_supervisor"},
		Version: version, Status: status, EffectiveFrom: from, EffectiveTo: to,
		Body: "Máy xúc bánh xích 1,2 m³: định mức " + mayXuc + " L/giờ máy, dung sai +15%.\n" +
			"Máy ủi: định mức 21 L/giờ máy (giả lập, chỉ để kiểm loại trừ)."}
}

func ngay(s string) func() time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return func() time.Time { return t }
}

const cauDinhMuc = "Định mức nhiên liệu của máy xúc bánh xích 1,2 m³ theo quy định nội bộ là bao nhiêu?"

// CON-T-005: v2 hết hiệu lực từ 2026-03-01, v3 hiệu lực từ 2026-03-01 → chỉ v3 được làm căn cứ.
func TestRetrieveChiBanDangHieuLuc(t *testing.T) {
	ix := NewMemoryIndex("a", []Document{
		dinhMuc("SOP-NL-ĐM-01", 2, StatusApproved, "2025-01-01", "2026-03-01", "16"),
		dinhMuc("SOP-NL-ĐM-01", 3, StatusApproved, "2026-03-01", "", "17"),
	})
	ix.Now = ngay("2026-10-19")
	ev, err := ix.Retrieve(context.Background(), principal("a", "site_supervisor"), cauDinhMuc, 3)
	if err != nil || len(ev) == 0 {
		t.Fatalf("muốn đoạn của v3, được %v (lỗi %v)", ev, err)
	}
	for _, e := range ev {
		if e.Source.DocumentVersion != 3 || !strings.HasPrefix(e.ID, "SOP-NL-ĐM-01@v3#") {
			t.Errorf("đoạn không phải v3: %s (bản %d)", e.ID, e.Source.DocumentVersion)
		}
		if strings.Contains(e.Text, "16 L") {
			t.Errorf("số của v2 hết hiệu lực lọt vào bằng chứng: %q", e.Text)
		}
	}
	if !strings.Contains(ev[0].Text, "17 L/giờ") {
		t.Errorf("đoạn đầu phải là hàng máy xúc của v3, được %q", ev[0].Text)
	}
	// Hỏi đúng ngày cũ thì bản cũ mới là bản hiệu lực: hiệu lực xét theo ngày HỎI, không theo ngày nạp.
	ix.Now = ngay("2026-02-28")
	ev, err = ix.Retrieve(context.Background(), principal("a", "site_supervisor"), cauDinhMuc, 3)
	if err != nil || len(ev) == 0 || ev[0].Source.DocumentVersion != 2 {
		t.Errorf("ngày 2026-02-28: muốn v2, được %v (lỗi %v)", ev, err)
	}
}

// CON-T-061: hai bản cùng nhãn "đang hiệu lực" (nhập sai) → không chọn bản nào, báo xung đột.
func TestRetrieveHaiBanCungHieuLucLaXungDot(t *testing.T) {
	ix := NewMemoryIndex("a", []Document{
		dinhMuc("SOP-NL-ĐM-01", 2, StatusApproved, "2025-01-01", "", "17"),
		dinhMuc("SOP-NL-ĐM-01", 3, StatusApproved, "2026-03-01", "", "18"),
	})
	ix.Now = ngay("2026-10-19")
	ev, err := ix.Retrieve(context.Background(), principal("a", "site_supervisor"), "Định mức nhiên liệu máy xúc 1,2 m³ là bao nhiêu?", 3)
	var n *VersionNotice
	if !errors.As(err, &n) || n.Kind != NoticeConflict || n.DocumentID != "SOP-NL-ĐM-01" {
		t.Fatalf("muốn thông báo xung đột của SOP-NL-ĐM-01, được %v (đoạn %v)", err, ev)
	}
	if len(ev) != 0 {
		t.Errorf("xung đột mà vẫn trả đoạn: %v", ev)
	}
	if got := versionsOf(n); got != "2,3" {
		t.Errorf("thông báo phải nêu cả hai bản, được %q", got)
	}
}

// CON-T-032: v3 bị cổng chất lượng giữ lại (needs_review), v2 đã hết hiệu lực → không có bản nào phục vụ
// được; báo rõ chứ không trả số của v2 hay của bảng hỏng.
func TestRetrieveKhongCoBanHieuLuc(t *testing.T) {
	ix := NewMemoryIndex("a", []Document{
		dinhMuc("SOP-NL-ĐM-01", 2, StatusApproved, "2025-01-01", "2026-03-01", "16"),
		dinhMuc("SOP-NL-ĐM-01", 3, StatusNeedsReview, "2026-03-01", "", "17"),
	})
	ix.Now = ngay("2026-10-19")
	ev, err := ix.Retrieve(context.Background(), principal("a", "site_supervisor"), cauDinhMuc, 3)
	var n *VersionNotice
	if !errors.As(err, &n) || n.Kind != NoticeNoEffective {
		t.Fatalf("muốn thông báo không có bản hiệu lực, được %v (đoạn %v)", err, ev)
	}
	if len(ev) != 0 {
		t.Errorf("không có bản hiệu lực mà vẫn trả đoạn: %v", ev)
	}
	if got := versionsOf(n); got != "2,3" {
		t.Errorf("thông báo phải nêu v2 (hết hiệu lực) và v3 (chờ soát), được %q", got)
	}
}

// Nháp, chờ soát, đã thay thế: không bản nào được làm căn cứ, kể cả khi ngày hiệu lực khớp. Nháp / chờ soát thì
// báo "không có bản hiệu lực"; đã thay thế thì im lặng như không tồn tại (soát vòng 1, F16: tài liệu đã được thay
// bằng tài liệu khác, báo thì che mất tài liệu mới).
func TestRetrieveTrangThaiChuaDuyetKhongPhucVu(t *testing.T) {
	for _, st := range []string{StatusDraft, StatusNeedsReview, StatusSuperseded} {
		ix := NewMemoryIndex("a", []Document{dinhMuc("SOP-X", 1, st, "2026-01-01", "", "17")})
		ix.Now = ngay("2026-10-19")
		ev, err := ix.Retrieve(context.Background(), principal("a", "site_supervisor"), cauDinhMuc, 3)
		var n *VersionNotice
		baoDuoc := errors.As(err, &n) && n.Kind == NoticeNoEffective
		if len(ev) != 0 || baoDuoc == (st == StatusSuperseded) || (st == StatusSuperseded && err != nil) {
			t.Errorf("trạng thái %q: 0 đoạn, thông báo = %v; được %v / %v", st, st != StatusSuperseded, ev, err)
		}
	}
}

// Phạm vi: tài liệu khai phạm vi PRJ-B thì người chỉ được giao PRJ-A không thấy — và thấy ĐÚNG như khi tài
// liệu không tồn tại: không đoạn, không thông báo, kể cả khi tài liệu đó đang xung đột phiên bản.
func TestRetrievePhamViKhongLo(t *testing.T) {
	a := dinhMuc("SOP-NL-ĐM-01", 2, StatusApproved, "2025-01-01", "", "17")
	b := dinhMuc("SOP-NL-ĐM-01", 3, StatusApproved, "2026-03-01", "", "18")
	a.Scope, b.Scope = []string{"PRJ-B"}, []string{"PRJ-B"}
	ix := NewMemoryIndex("a", []Document{a, b})
	ix.Now, ix.InScope = ngay("2026-10-19"), theoThuocTinh
	pA := principal("a", "site_supervisor")
	pA.Attributes = map[string]string{"entity_scope": "PRJ-A"}
	ev, err := ix.Retrieve(context.Background(), pA, cauDinhMuc, 3)
	if err != nil || len(ev) != 0 {
		t.Errorf("ngoài phạm vi phải giống không tồn tại (0 đoạn, không lỗi), được %v / %v", ev, err)
	}
	empty, err2 := NewMemoryIndex("a", nil).Retrieve(context.Background(), pA, cauDinhMuc, 3)
	if len(empty) != len(ev) || (err == nil) != (err2 == nil) {
		t.Errorf("ngoài phạm vi khác với kho rỗng: %v/%v so với %v/%v", ev, err, empty, err2)
	}
	pB := principal("a", "site_supervisor")
	pB.Attributes = map[string]string{"entity_scope": "PRJ-A, PRJ-B"}
	var n *VersionNotice
	if _, err := ix.Retrieve(context.Background(), pB, cauDinhMuc, 3); !errors.As(err, &n) {
		t.Errorf("trong phạm vi: muốn thấy xung đột, được %v", err)
	}
	// Bên gọi đưa luật phạm vi của mình (ví dụ router.Policy.InScope): được dùng, nhưng xét theo TỪNG vai có trong ACL —
	// director (ngoài ACL) không cho site_supervisor mượn phạm vi "xem tất cả" (soát độc lập U6a).
	ix.InScope = func(pr contracts.Principal, keys []string) bool { return pr.HasRole("director") }
	if ev, err := ix.Retrieve(context.Background(), principal("a", "site_supervisor", "director"), cauDinhMuc, 3); len(ev) != 0 || err != nil {
		t.Errorf("vai ngoài ACL cho mượn phạm vi: %v / %v", ev, err)
	}
	ix.InScope = func(pr contracts.Principal, keys []string) bool { return pr.HasRole("site_supervisor") }
	if _, err := ix.Retrieve(context.Background(), principal("a", "site_supervisor"), cauDinhMuc, 3); !errors.As(err, &n) {
		t.Errorf("InScope do bên gọi đưa không được dùng: %v", err)
	}
}

// Tài liệu cũ không khai phiên bản chạy y như trước: mã đoạn giữ dạng "<id>#<dòng>", không có bản.
func TestRetrieveTaiLieuKhongPhienBanGiuNguyen(t *testing.T) {
	ix := NewMemoryIndex("a", testDocs())
	ev, err := ix.Retrieve(context.Background(), principal("a", "site_engineer"), "Biên bản đối soát phải lập trong bao lâu?", 3)
	if err != nil || len(ev) == 0 || ev[0].ID != "SOP-1#0" || ev[0].Source.DocumentVersion != 0 {
		t.Errorf("tài liệu không phiên bản: muốn SOP-1#0 bản 0, được %v (lỗi %v)", ev, err)
	}
}

// Nạp markdown có phần đầu (frontmatter): id, bản, trạng thái, hiệu lực, phạm vi. Tệp không có phần đầu nạp
// như cũ. Phần đầu sai (khoá lạ, trạng thái lạ, ngày sai, trùng id@bản) thì báo lỗi chứ không công bố.
func TestLoadMarkdownPhanDau(t *testing.T) {
	dir := t.TempDir()
	write := func(name, s string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("acl.json", `{"SOP-NL-ĐM-01": ["site_supervisor"], "SOP-FUEL-01": ["director"]}`)
	write("SOP-FUEL-01.md", "# Quy trình xử lý chênh lệch\n1. Lập biên bản.")
	write("dm-v3.md", "---\nid: SOP-NL-ĐM-01\nversion: 3\nstatus: approved\neffective_from: 2026-03-01\nscope: [PRJ-A]\n---\n# Định mức nhiên liệu\nMáy xúc: 17 L/giờ.")
	docs, err := LoadMarkdown(dir, "a")
	if err != nil || len(docs) != 2 {
		t.Fatalf("muốn 2 tài liệu, được %d (lỗi %v)", len(docs), err)
	}
	var dm, sop Document
	for _, d := range docs {
		if d.ID == "SOP-NL-ĐM-01" {
			dm = d
		} else {
			sop = d
		}
	}
	if dm.Version != 3 || dm.Status != StatusApproved || dm.EffectiveFrom != "2026-03-01" || dm.EffectiveTo != "" ||
		strings.Join(dm.Scope, ",") != "PRJ-A" || dm.Title != "Định mức nhiên liệu" || strings.Join(dm.ACL, ",") != "site_supervisor" {
		t.Errorf("phần đầu đọc sai: %+v", dm)
	}
	if sop.ID != "SOP-FUEL-01" || sop.Version != 0 || sop.Status != "" || sop.Title != "Quy trình xử lý chênh lệch" {
		t.Errorf("tệp không phần đầu phải nạp như cũ: %+v", sop)
	}
	for name, bad := range map[string]string{
		"khoá lạ":               "---\nid: X\nversion: 1\nstatus: approved\nowner: ai\n---\n# T\nb",
		"trạng thái lạ":         "---\nid: X\nversion: 1\nstatus: approvd\n---\n# T\nb",
		"ngày sai":              "---\nid: X\nversion: 1\nstatus: approved\neffective_from: 01/03/2026\n---\n# T\nb",
		"thiếu bản":             "---\nid: X\nstatus: approved\n---\n# T\nb",
		"hết trước khi bắt đầu": "---\nid: X\nversion: 1\nstatus: approved\neffective_from: 2026-03-01\neffective_to: 2026-01-01\n---\n# T\nb",
		"không đóng":            "---\nid: X\nversion: 1\nstatus: approved\n# T\nb",
	} {
		d := t.TempDir()
		if err := os.WriteFile(filepath.Join(d, "x.md"), []byte(bad), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadMarkdown(d, "a"); err == nil {
			t.Errorf("%s: phải báo lỗi", name)
		}
	}
	d := t.TempDir()
	for _, f := range []string{"a.md", "b.md"} {
		if err := os.WriteFile(filepath.Join(d, f), []byte("---\nid: X\nversion: 1\nstatus: approved\n---\n# T\nb"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := LoadMarkdown(d, "a"); err == nil {
		t.Errorf("trùng id@bản: phải báo lỗi")
	}
}

func versionsOf(n *VersionNotice) string {
	var s []string
	for _, v := range n.Versions {
		s = append(s, strconv.Itoa(v.Version))
	}
	return strings.Join(s, ",")
}

// Soát vòng 1 (F1): tệp lưu CRLF hoặc có BOM vẫn phải đọc được phần đầu — nếu không, bản nháp nạp như tài liệu
// không phiên bản (coi như đã duyệt) và bị phục vụ.
func TestLoadMarkdownCRLFvaBOM(t *testing.T) {
	for name, s := range map[string]string{
		"crlf.md": "---\r\nid: SOP-X\r\nversion: 1\r\nstatus: draft\r\n---\r\n# Định mức\r\nMáy xúc 99 L/giờ\r\n",
		"bom.md":  "\ufeff---\nid: SOP-X\nversion: 1\nstatus: draft\n---\n# Định mức\nMáy xúc 99 L/giờ\n",
	} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "SOP-X.md"), []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
		docs, err := LoadMarkdown(dir, "a")
		if err != nil || len(docs) != 1 || docs[0].Status != StatusDraft || docs[0].Version != 1 || docs[0].Title != "Định mức" {
			t.Errorf("%s: phần đầu không được đọc: %+v (lỗi %v)", name, docs, err)
		}
	}
}

// Soát vòng 1 (F6, F7, F8, F11): khai sai thì báo lỗi lúc nạp, không để lộ ra lúc hỏi.
func TestLoadMarkdownKhaiSaiKhac(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"vừa không phiên bản vừa có bản": {"SOP-X.md": "# T\nb", "v3.md": "---\nid: SOP-X\nversion: 3\nstatus: approved\n---\n# T\nb"},
		"bản không nguyên":               {"x.md": "---\nid: X\nversion: 2.7\nstatus: approved\n---\n# T\nb"},
		"từ = đến":                       {"x.md": "---\nid: X\nversion: 1\nstatus: approved\neffective_from: \"2026-03-01\"\neffective_to: \"2026-03-01\"\n---\n# T\nb"},
	} {
		dir := t.TempDir()
		for f, s := range files {
			if err := os.WriteFile(filepath.Join(dir, f), []byte(s), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := LoadMarkdown(dir, "a"); err == nil {
			t.Errorf("%s: phải báo lỗi", name)
		}
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x.md"), []byte("---\n---\n# T\nb"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadMarkdown(dir, "a"); err == nil || strings.Contains(err.Error(), "dòng đóng") {
		t.Errorf("phần đầu rỗng: phải báo thiếu version, không phải thiếu dòng đóng: %v", err)
	}
}

// Soát vòng 1 (F2): mỗi BẢN có phạm vi riêng. Người chỉ thấy v2 không được biết v3 tồn tại.
func TestRetrievePhamViTheoTungBan(t *testing.T) {
	v2 := dinhMuc("SOP-NL-ĐM-01", 2, StatusApproved, "2025-01-01", "", "16")
	v3 := dinhMuc("SOP-NL-ĐM-01", 3, StatusApproved, "2026-03-01", "", "17")
	v2.Scope, v3.Scope = []string{"PRJ-A"}, []string{"PRJ-B"}
	pA := principal("a", "site_supervisor")
	pA.Attributes = map[string]string{"entity_scope": "PRJ-A"}
	ix := NewMemoryIndex("a", []Document{v2, v3})
	ix.Now, ix.InScope = ngay("2026-10-19"), theoThuocTinh
	ev, err := ix.Retrieve(context.Background(), pA, cauDinhMuc, 3)
	if err != nil || len(ev) == 0 || ev[0].Source.DocumentVersion != 2 {
		t.Errorf("chỉ thấy v2: muốn đoạn v2, không xung đột; được %v / %v", ev, err)
	}
	v2.EffectiveTo = "2026-03-01"
	ix = NewMemoryIndex("a", []Document{v2, v3})
	ix.Now, ix.InScope = ngay("2026-10-19"), theoThuocTinh
	_, err = ix.Retrieve(context.Background(), pA, cauDinhMuc, 3)
	var n *VersionNotice
	if !errors.As(err, &n) || n.Kind != NoticeNoEffective || versionsOf(n) != "2" {
		t.Errorf("v2 hết hạn: thông báo chỉ được nêu v2, được %v", err)
	}
}

// Soát vòng 1 (F3): câu hỏi khớp nhiều tài liệu. Thông báo chỉ thắng khi tài liệu có vấn đề khớp NGANG hoặc hơn
// đoạn tốt nhất; hai tài liệu cùng có vấn đề và hoà điểm thì chọn theo mã, không theo thứ tự nạp.
func TestRetrieveThongBaoGiuaHaiTaiLieu(t *testing.T) {
	doc := func(id string, v int, st, to, title, body string) Document {
		return Document{ID: id, Tenant: "a", Title: title, ACL: []string{"r"}, Version: v, Status: st, EffectiveFrom: "2025-01-01", EffectiveTo: to, Body: body}
	}
	tot := doc("SOP-A", 1, StatusApproved, "", "Quy trình đối soát nhiên liệu", "Kho đối soát phiếu xuất với nhật ký máy.")
	xd2 := doc("SOP-B", 1, StatusApproved, "", "Quy trình cấp phát nhiên liệu", "Kho cấp phát khi có phiếu đề nghị.")
	xd3 := doc("SOP-B", 2, StatusApproved, "", "Quy trình cấp phát nhiên liệu", "Kho cấp phát khi có phiếu đề nghị.")
	ask := func(docs []Document, q string) ([]contracts.Evidence, error) {
		ix := NewMemoryIndex("a", docs)
		ix.Now = ngay("2026-10-19")
		return ix.Retrieve(context.Background(), principal("a", "r"), q, 3)
	}
	var n *VersionNotice
	if ev, err := ask([]Document{tot, xd2, xd3}, "đối soát phiếu xuất với nhật ký máy"); err != nil || len(ev) == 0 || ev[0].Source.DocumentID != "SOP-A" {
		t.Errorf("SOP-A khớp hơn: muốn đoạn SOP-A, được %v / %v", ev, err)
	}
	if _, err := ask([]Document{tot, xd2, xd3}, "cấp phát khi có phiếu đề nghị"); !errors.As(err, &n) || n.DocumentID != "SOP-B" {
		t.Errorf("SOP-B (xung đột) khớp hơn: muốn thông báo SOP-B, được %v", err)
	}
	ngang := doc("SOP-C", 1, StatusApproved, "", xd2.Title, xd2.Body) // cùng tiêu đề + nội dung → cùng điểm
	if _, err := ask([]Document{ngang, xd2, xd3}, "cấp phát khi có phiếu đề nghị"); !errors.As(err, &n) || n.DocumentID != "SOP-B" {
		t.Errorf("hoà điểm: thông báo phải thắng đoạn, được %v", err)
	}
	het := doc("SOP-0", 1, StatusApproved, "2026-01-01", xd2.Title, xd2.Body) // hết hạn, cùng điểm, mã nhỏ hơn
	for _, docs := range [][]Document{{xd2, xd3, het}, {het, xd3, xd2}} {
		if _, err := ask(docs, "cấp phát khi có phiếu đề nghị"); !errors.As(err, &n) || n.DocumentID != "SOP-0" {
			t.Errorf("hai tài liệu cùng có vấn đề, hoà điểm: muốn SOP-0 (mã nhỏ) bất kể thứ tự nạp, được %v", err)
		}
	}
}

// Soát vòng 1 (F4): ngày chuyển bản. v2 hết hiệu lực từ 2026-03-01, v3 có hiệu lực từ 2026-03-01 → đúng ngày đó
// chỉ v3 (khoảng [từ, đến)), không xung đột, không khoảng trống.
func TestRetrieveNgayChuyenBan(t *testing.T) {
	ix := NewMemoryIndex("a", []Document{
		dinhMuc("SOP-NL-ĐM-01", 2, StatusApproved, "2025-01-01", "2026-03-01", "16"),
		dinhMuc("SOP-NL-ĐM-01", 3, StatusApproved, "2026-03-01", "", "17"),
	})
	ix.Now = ngay("2026-03-01")
	ev, err := ix.Retrieve(context.Background(), principal("a", "site_supervisor"), cauDinhMuc, 3)
	if err != nil || len(ev) == 0 || ev[0].Source.DocumentVersion != 3 {
		t.Errorf("ngày 2026-03-01: muốn v3, được %v / %v", ev, err)
	}
}

// Soát vòng 1 (F16): tài liệu mà mọi bản đều "đã thay thế" là tài liệu đã được thay bằng tài liệu khác — không báo
// "không có bản hiệu lực", để không che tài liệu thay thế có cùng nội dung.
func TestRetrieveBanDaThayTheKhongCheTaiLieuMoi(t *testing.T) {
	cu := dinhMuc("SOP-CU", 1, StatusSuperseded, "2025-01-01", "", "17")
	moi := dinhMuc("SOP-MOI", 1, StatusApproved, "2026-03-01", "", "17")
	ix := NewMemoryIndex("a", []Document{cu, moi})
	ix.Now = ngay("2026-10-19")
	ev, err := ix.Retrieve(context.Background(), principal("a", "site_supervisor"), cauDinhMuc, 3)
	if err != nil || len(ev) == 0 || ev[0].Source.DocumentID != "SOP-MOI" {
		t.Errorf("muốn đoạn của SOP-MOI, được %v / %v", ev, err)
	}
}
