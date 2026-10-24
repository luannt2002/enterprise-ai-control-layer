package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// banTaiLieu ghi một bản tài liệu có phiên bản vào thư mục docs của dữ liệu tenant và cấp quyền đọc cho director trong acl.json.
func banTaiLieu(t *testing.T, docs, file, id, ver, status, body string) {
	t.Helper()
	text := "---\nid: " + id + "\nversion: " + ver + "\nstatus: " + status + "\neffective_from: \"2025-01-01\"\n---\n# Bảng định mức " + id + "\n" + body
	if err := os.WriteFile(filepath.Join(docs, file), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(docs, "acl.json")
	acl := map[string][]string{}
	if b, err := os.ReadFile(p); err != nil || json.Unmarshal(b, &acl) != nil {
		t.Fatalf("acl.json của seed: %v", err)
	}
	acl[id] = []string{"director"}
	if b, err := json.Marshal(acl); err != nil || os.WriteFile(p, b, 0o644) != nil {
		t.Fatalf("ghi acl.json: %v", err)
	}
}

// Báo cáo nạp nói ra bản bị cổng chất lượng giữ lại (F-018): số bản và lý do của từng bản, cả trong log — trước đây lần nạp báo thành
// công mà tài liệu không được công bố, không ai biết.
func TestIngestBaoBanBiGiu(t *testing.T) {
	tn := uniqTenant("ban-bi-giu")
	packDir, dataRoot := tenantFixtures(t, tn)
	docs := filepath.Join(dataRoot, tn, "docs")
	banTaiLieu(t, docs, "bang-v1.md", "BANG-01", "1", "approved", "| Hạng mục | Định mức |\n|---|---|\n| Đào móng | |\n")
	logs := &syncBuffer{}
	ap, err := newApp(config{Pack: packDir, Data: dataRoot, Now: clockForTest}, newMemStore(), slog.New(slog.NewTextHandler(logs, nil)))
	if err != nil {
		t.Fatal(err)
	}
	rep, err := ap.ingest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Held != 1 || len(rep.Holds) != 1 || !strings.HasPrefix(rep.Holds[0], "BANG-01@v1: ") || !strings.Contains(rep.Holds[0], "ô rỗng") {
		t.Errorf("muốn đúng 1 bản bị giữ (BANG-01@v1, lý do ô rỗng), được held=%d %q", rep.Held, rep.Holds)
	}
	if b, _ := json.Marshal(rep); !strings.Contains(string(b), `"held":1`) {
		t.Errorf("JSON của báo cáo nạp phải có held: %s", b)
	}
	if l := logs.String(); !strings.Contains(l, "BANG-01@v1") || !strings.Contains(l, "level=WARN") {
		t.Errorf("log phải cảnh báo bản bị giữ: %s", l)
	}
}

// Nạp lại không hạ âm thầm bản ĐANG công bố (F-018, đúng kịch bản của phiếu): v3 đã duyệt (cột A, B) được công bố; thêm v2 đã thay thế
// (A, B, C) rồi nạp lại — cổng (không nhớ trạng thái) thấy v3 "mất cột C" so với mốc v2. Kho đang công bố đúng nội dung v3 nên v3 giữ
// nguyên đã duyệt, báo cáo nêu cảnh báo; còn khi nội dung v3 ĐỔI mà vẫn không đạt thì v3 bị giữ lại như cũ, nạp lại vẫn bị giữ.
func TestNapLaiKhongHaBanDaCongBo(t *testing.T) {
	st, admin := openPG(t)
	tn := uniqTenant("nap-lai-cong-bo")
	packDir, dataRoot := tenantFixtures(t, tn)
	docs := filepath.Join(dataRoot, tn, "docs")
	ap, err := newApp(config{Pack: packDir, Data: dataRoot, Now: clockForTest}, st, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	const v3 = "| A | B |\n|---|---|\n| Đào móng | 2 |\n"
	trangThai := func() string {
		t.Helper()
		for _, s := range []string{"approved", "needs_review"} {
			if countRows(t, admin, "documents", tn, "id = 'BANG-02' AND version = 3 AND status = $2", s) == 1 {
				return s
			}
		}
		return "?"
	}
	banTaiLieu(t, docs, "bang-v3.md", "BANG-02", "3", "approved", v3)
	if rep, err := ap.ingest(context.Background()); err != nil || rep.Held != 0 || trangThai() != "approved" {
		t.Fatalf("lần nạp đầu: v3 phải được công bố, được %s, held=%d %v", trangThai(), rep.Held, err)
	}
	banTaiLieu(t, docs, "bang-v2.md", "BANG-02", "2", "superseded", "| A | B | C |\n|---|---|---|\n| Đào móng | 2 | 1 |\n")
	rep, err := ap.ingest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := trangThai(); got != "approved" || rep.Held != 0 {
		t.Errorf("nạp lại cùng nội dung v3: muốn v3 vẫn approved, held=0; được %s, held=%d %q", got, rep.Held, rep.Holds)
	}
	if len(rep.KeptPublished) != 1 || !strings.HasPrefix(rep.KeptPublished[0], "BANG-02@v3: ") || !strings.Contains(rep.KeptPublished[0], "mất cột") {
		t.Errorf("báo cáo phải cảnh báo bản được giữ nguyên công bố (lý do của cổng): %q", rep.KeptPublished)
	}
	banTaiLieu(t, docs, "bang-v3.md", "BANG-02", "3", "approved", v3+"| Lấp đất | 3 |\n")
	if rep, err := ap.ingest(context.Background()); err != nil || trangThai() != "needs_review" || rep.Held != 1 || len(rep.KeptPublished) != 0 {
		t.Errorf("v3 đổi nội dung mà vẫn mất cột: muốn bị giữ (needs_review, held=1); được %s, held=%d, giữ nguyên %q, %v", trangThai(), rep.Held, rep.KeptPublished, err)
	}
	// Bản đang BỊ GIỮ trong kho không phải bản đã công bố: nạp lại y nguyên thì vẫn bị giữ.
	if rep, err := ap.ingest(context.Background()); err != nil || trangThai() != "needs_review" || rep.Held != 1 {
		t.Errorf("nạp lại bản đang bị giữ: muốn vẫn needs_review, held=1; được %s, held=%d, %v", trangThai(), rep.Held, err)
	}
}
