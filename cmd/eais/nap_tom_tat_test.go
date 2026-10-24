package main

import (
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

// Dòng tóm tắt của lần nạp — log "đã nạp lại dữ liệu" của POST /v1/ingest và dòng NẠP của `eais ingest` — nói số bản tài liệu bị cổng
// chất lượng giữ lại và số bản giữ nguyên công bố (F-018): người nạp chỉ đọc dòng tóm tắt không được thấy "nạp xong" mà không biết có
// tài liệu không được công bố.
func TestNapLaiQuaHTTPLogSoBanBiGiu(t *testing.T) {
	e := newEnv(t, newMemStore())
	banTaiLieu(t, filepath.Join(e.dataRoot, e.tenantA, "docs"), "bang-v1.md", "BANG-01", "1", "approved", "| Hạng mục | Định mức |\n|---|---|\n| Đào móng | |\n")
	if r := e.post("/v1/ingest", tokDirector, struct{}{}); r.status != http.StatusOK {
		t.Fatalf("POST /v1/ingest = %d: %s", r.status, r.body)
	}
	for _, l := range strings.Split(e.serverLog(), "\n") {
		if strings.Contains(l, "đã nạp lại dữ liệu") {
			if !strings.Contains(l, " held=1 ") || !strings.Contains(l, " kept_published=0 ") {
				t.Errorf("log nạp lại phải có held=1 và kept_published=0: %s", l)
			}
			return
		}
	}
	t.Fatalf("không thấy dòng log nạp lại:\n%s", e.serverLog())
}

func TestCLIIngestInSoBanBiGiu(t *testing.T) {
	openPG(t)
	tn := uniqTenant("cli-ban-bi-giu")
	packDir, dataRoot := tenantFixtures(t, tn)
	banTaiLieu(t, filepath.Join(dataRoot, tn, "docs"), "bang-v1.md", "BANG-01", "1", "approved", "| Hạng mục | Định mức |\n|---|---|\n| Đào móng | |\n")
	code, out, errs := runCLI("ingest", "-pack", packDir, "-data", dataRoot)
	if code != 0 || !strings.Contains(out, "đã có từ lần nạp cùng mã), 1 bản tài liệu bị giữ (không công bố), 0 bản giữ nguyên công bố\n") {
		t.Fatalf("ingest CLI = %d, dòng NẠP phải nói số bản bị giữ và số bản giữ nguyên công bố:\n%s%s", code, out, errs)
	}
}
