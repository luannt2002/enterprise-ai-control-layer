package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Luồng RAG trọn qua ứng dụng (nhánh RAG, docs/viec/2-tra-quy-trinh): tài liệu có phiên bản trong thư mục dữ liệu → nạp (như `eais
// ingest`, rồi POST /v1/ingest) → PostgreSQL → POST /v1/ask bằng token từng vai → GET /v1/trace. Từng khúc đã có test riêng (chỉ
// mục, PG, pipeline); đây là đường người dùng thật chạm vào. Nội dung tài liệu cố ý không chứa con số: test canh luồng RAG, không
// canh lớp chặn số của tài liệu (P4).
func TestRAGTronUngDung(t *testing.T) {
	st, _ := openPG(t)
	tn := uniqTenant("rag-e2e")
	packDir, dataRoot := tenantFixtures(t, tn)
	docs := filepath.Join(dataRoot, tn, "docs")
	viet := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(docs, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ban := func(id, ver, status, from, to, scope, title, line string) string {
		h := "---\nid: " + id + "\nversion: " + ver + "\nstatus: " + status + "\neffective_from: \"" + from + "\"\n"
		if to != "" {
			h += "effective_to: \"" + to + "\"\n"
		}
		if scope != "" {
			h += "scope: [" + scope + "]\n"
		}
		return h + "---\n# " + title + "\n1. " + line + "\n"
	}
	const nt = "Quy trình nghiệm thu khối lượng thi công"
	viet("qt-nt-v1.md", ban("QT-NT-01", "1", "approved", "2025-01-01", "2026-03-01", "", nt, "Biên bản nghiệm thu khối lượng do kỹ sư kỹ thuật lập."))
	viet("qt-nt-v2.md", ban("QT-NT-01", "2", "approved", "2026-03-01", "", "", nt, "Biên bản nghiệm thu khối lượng do tư vấn giám sát lập."))
	viet("qt-nt-v3.md", ban("QT-NT-01", "3", "draft", "2026-10-01", "", "", nt, "Biên bản nghiệm thu khối lượng do chủ đầu tư lập."))
	const gg = "Quy trình kiểm tra an toàn giàn giáo"
	viet("qt-gg-v1.md", ban("QT-GG-01", "1", "approved", "2025-01-01", "", "", gg, "Kiểm tra giàn giáo trước mỗi ca, ghi sổ an toàn."))
	viet("qt-gg-v2.md", ban("QT-GG-01", "2", "approved", "2026-01-01", "", "", gg, "Kiểm tra giàn giáo hằng tuần, ghi sổ an toàn."))
	viet("bp-tru-a.md", ban("BP-TRU-A", "1", "approved", "2025-06-01", "", "PRJ-A", "Biện pháp đổ bê tông trụ cầu",
		"Quy trình đổ bê tông trụ: đổ liên tục, đầm dùi ngay sau mỗi lớp."))
	aclPath := filepath.Join(docs, "acl.json")
	acl := map[string][]string{}
	if b, err := os.ReadFile(aclPath); err != nil || json.Unmarshal(b, &acl) != nil {
		t.Fatalf("acl.json của seed: %v", err)
	}
	for _, id := range []string{"QT-NT-01", "QT-GG-01", "BP-TRU-A"} {
		acl[id] = []string{"director", "site_engineer"}
	}
	if b, err := json.Marshal(acl); err != nil || os.WriteFile(aclPath, b, 0o644) != nil {
		t.Fatalf("ghi acl.json: %v", err)
	}

	logs := &syncBuffer{}
	log := slog.New(slog.NewTextHandler(logs, nil))
	ap, err := newApp(config{Pack: packDir, Data: dataRoot, Now: clockForTest}, st, log)
	if err != nil {
		t.Fatalf("newApp: %v", err)
	}
	if _, err := ap.ingest(context.Background()); err != nil {
		t.Fatalf("nạp: %v", err)
	}
	user := func(id, role, scope string) contracts.Principal {
		p := contracts.Principal{ID: id, TenantID: contracts.TenantID(tn), Kind: contracts.PrincipalUser, Roles: []string{role}}
		if scope != "" {
			p.Attributes = map[string]string{"entity_scope": scope}
		}
		return p
	}
	srv := newServer(ap, map[string]contracts.Principal{
		"t-gd": user("u-gd", "director", ""), "t-ks-a": user("u-ks-a", "site_engineer", "PRJ-A"), "t-ks-b": user("u-ks-b", "site_engineer", "PRJ-B"),
	}, log)
	srv.hostGuard = true
	ts := httptest.NewServer(srv.handler())
	t.Cleanup(ts.Close)
	e := &env{t: t, ts: ts}
	hoi := func(tok, text string) contracts.AskResponse {
		t.Helper()
		r := e.ask(tok, text)
		if r.status != 200 {
			t.Fatalf("%q: HTTP %d %s", text, r.status, r.body)
		}
		return decodeAs[contracts.AskResponse](t, r)
	}
	const cauNT = "Quy trình nghiệm thu khối lượng: biên bản do ai lập?"

	// Bản đang hiệu lực (v2) được trích, kèm số bản; bản cũ và bản nháp không lọt.
	r := hoi("t-gd", cauNT)
	if r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, "tư vấn giám sát") ||
		strings.Contains(r.Answer, "kỹ sư kỹ thuật") || strings.Contains(r.Answer, "chủ đầu tư") {
		t.Errorf("phiên bản hiệu lực: muốn v2 (tư vấn giám sát), được %s → %q", r.Outcome, r.Answer)
	}
	if len(r.Citations) == 0 || r.Citations[0].Source.DocumentVersion != 2 {
		t.Errorf("trích dẫn phải mang bản 2: %+v", r.Citations)
	}
	if tr := e.trace("t-gd", r.TraceID); tr.status != 200 || !strings.Contains(string(tr.body), "QT-NT-01@v2#") {
		t.Errorf("vết S7 phải nêu đoạn QT-NT-01@v2 đã truy xuất: HTTP %d", tr.status)
	}

	// Hai bản cùng hiệu lực: nói rõ, không chọn bản nào; vết S7 ghi lý do.
	r = hoi("t-gd", "Quy trình kiểm tra an toàn giàn giáo thế nào?")
	if r.Outcome != contracts.OutcomeSourceConflict || strings.Contains(r.Answer, "mỗi ca") || strings.Contains(r.Answer, "hằng tuần") {
		t.Errorf("xung đột phiên bản: muốn SOURCE_CONFLICT, không trích bản nào; được %s → %q", r.Outcome, r.Answer)
	}
	if tr := e.trace("t-gd", r.TraceID); !strings.Contains(string(tr.body), "version_conflict") {
		t.Errorf("vết S7 phải ghi thông báo xung đột phiên bản: %s", tr.body)
	}

	// Phạm vi theo token: kỹ sư PRJ-A đọc được biện pháp của PRJ-A, kỹ sư PRJ-B thì như không có tài liệu.
	const cauTru = "Quy trình đổ bê tông trụ quy định thế nào?"
	if r := hoi("t-ks-a", cauTru); r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, "đầm dùi") {
		t.Errorf("kỹ sư PRJ-A: muốn trích biện pháp PRJ-A, được %s → %q", r.Outcome, r.Answer)
	}
	if r := hoi("t-ks-b", cauTru); r.Outcome == contracts.OutcomeResolved || strings.Contains(r.Answer, "đầm dùi") {
		t.Errorf("kỹ sư PRJ-B không được thấy biện pháp PRJ-A: %s → %q", r.Outcome, r.Answer)
	}

	// Kho không nói tới điều được hỏi: nói thiếu, không trích đoạn lạc đề.
	if r := hoi("t-gd", "Thủ tục xin giấy phép xây dựng gồm những gì?"); r.Outcome != contracts.OutcomeInsufficientEvidence {
		t.Errorf("chủ đề ngoài kho: muốn INSUFFICIENT_EVIDENCE, được %s → %q", r.Outcome, r.Answer)
	}

	// Nạp lại qua HTTP: v3 được duyệt và v2 hết hiệu lực từ 01/10 ⇒ lượt sau trích v3 (chỉ mục giữ sẵn phải dựng lại).
	viet("qt-nt-v2.md", ban("QT-NT-01", "2", "approved", "2026-03-01", "2026-10-01", "", nt, "Biên bản nghiệm thu khối lượng do tư vấn giám sát lập."))
	viet("qt-nt-v3.md", ban("QT-NT-01", "3", "approved", "2026-10-01", "", "", nt, "Biên bản nghiệm thu khối lượng do chủ đầu tư lập."))
	if rr := e.post("/v1/ingest", "t-gd", map[string]string{}); rr.status != 200 {
		t.Fatalf("POST /v1/ingest: HTTP %d %s", rr.status, rr.body)
	}
	if r := hoi("t-gd", cauNT); !strings.Contains(r.Answer, "chủ đầu tư") || strings.Contains(r.Answer, "tư vấn giám sát") {
		t.Errorf("sau khi nạp lại: muốn v3 (chủ đầu tư), được %s → %q", r.Outcome, r.Answer)
	}
}
