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
