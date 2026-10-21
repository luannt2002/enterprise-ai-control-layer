package eais_test

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

// Cổng danh tính: repo chỉ có MỘT danh tính sản phẩm hiện hành là EAIS (Enterprise AI Systems, D-RM-02). Tên dùng từ chiều
// 20/10 (kế hoạch V3) tới tối 21/10/2026 (mẫu dưới đây) chỉ được xuất hiện ở các tệp có lý do ghi trong legacyAllowed, đúng bằng
// số lần đã cho phép: bằng chứng đóng băng không được sửa cho đẹp tên, còn tài liệu / mã hiện hành không được để tên cũ quay lại.
// Tên viết tắt: mọi kiểu hoa thường. Tên dài: dò trên bản chỉ giữ chữ cái / chữ số (legacyLongName), nên mọi cách nối từ đều bị
// bắt — dấu cách, xuống dòng kể cả qua tiền tố comment / trích dẫn, gạch nối và gạch ngang Unicode, gạch dưới, dấu chấm, "/",
// ký tự vô hình, in đậm từng từ, viết liền. Giá phải trả: một từ có chuỗi "eacl" (ví dụ treacle) bị đếm oan; gặp thì ghi mục
// FALSE_POSITIVE.
var legacyIdentity = regexp.MustCompile(`(?i)eacl`)

// legacyLongName là tên dài cũ sau khi bỏ mọi ký tự không phải chữ cái / chữ số và viết thường.
const legacyLongName = "enterpriseaicontrollayer"

// gitLink là tên repo, cũng là đường dẫn module Go và mọi import. D-RM-02 giữ nguyên link git, nên cổng thay đúng chuỗi này
// (chữ thường, gạch nối) bằng một byte NUL trước khi đếm — thay chứ không xoá, để chữ hai bên không dính thành một lần khớp giả.
const gitLink = "enterprise-ai-control-layer"

// legacyCount đếm số lần tên cũ trong nội dung b (đã biết không phải tệp nhị phân). Không đếm link git, và không đếm "eACL" viết
// đúng kiểu đó ngay sau một chữ cái: đó là chữ ACL (access control list) cuối một từ, ví dụ RetrieveACL, không phải tên sản phẩm.
func legacyCount(b []byte) int {
	b = bytes.ReplaceAll(b, []byte(gitLink), []byte{0})
	n := 0
	for _, m := range legacyIdentity.FindAllIndex(b, -1) {
		if r, _ := utf8.DecodeLastRune(b[:m[0]]); string(b[m[0]:m[1]]) == "eACL" && unicode.IsLetter(r) {
			continue
		}
		n++
	}
	return n + strings.Count(lettersOnly(b), legacyLongName)
}

// lettersOnly trả các chữ cái / chữ số của b, viết thường; mọi ký tự khác bị bỏ, trừ NUL (chỗ của link git) được giữ làm điểm ngắt
// để chữ hai bên link không dính thành tên dài giả.
func lettersOnly(b []byte) string {
	var sb strings.Builder
	for _, r := range string(b) {
		switch {
		case r == 0:
			sb.WriteByte(0)
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			sb.WriteRune(unicode.ToLower(r))
		}
	}
	return sb.String()
}

type allowance struct {
	max    int
	reason string // HISTORICAL_EVIDENCE · IMMUTABLE_HASHED_ARTIFACT · COMPATIBILITY_ALIAS · EXTERNAL_REFERENCE · FALSE_POSITIVE
}

// legacyAllowed: tiền tố đường dẫn (thư mục kết thúc bằng "/") hoặc tệp → hạn mức (0 = không giới hạn) + lý do. Hạn mức khác 0
// phải ĐÚNG BẰNG số lần hiện có: nhiều hơn là tên cũ lan thêm, ít hơn là hạn mức thừa để tên cũ quay lại mà không bị bắt — cả
// hai đều đỏ. Chỉ chính tệp cổng (mẫu + ca âm) không giới hạn.
var legacyAllowed = map[string]allowance{
	"CHANGELOG.md":                                 {20, "HISTORICAL_EVIDENCE: các mục 21/10/2026 ghi đúng tên, module, biến và GUC lúc đó; mục đổi tên lại tối 21/10 nêu tên cũ"},
	"danh_tinh_test.go":                            {0, "FALSE_POSITIVE: chính mẫu và ca âm của cổng này"},
	"db/migrations/0001_win.sql":                   {7, "HISTORICAL_EVIDENCE: migration đã áp (không băm, nhưng không sửa); định danh chạy đổi bằng 0002"},
	"db/migrations/0002_dinh_danh_eais.sql":        {4, "COMPATIBILITY_ALIAS: migration gỡ quyền vai eacl_app của 0001, GUC tên cũ hết tác dụng"},
	"docs/CHO-QUYET-DINH.md":                       {25, "HISTORICAL_EVIDENCE: văn bản các quyết định đã chốt (D-V3-*, D-G7-*, D-OP-*, D-RM-*)"},
	"docs/PROMPT-AUDIT-ROADMAP.md":                 {6, "HISTORICAL_EVIDENCE: yêu cầu có ngày (mốc 8280f40)"},
	"docs/PROMPT-BUILD-EAIS.md":                    {6, "HISTORICAL_EVIDENCE: §4 mục 1 của brief nêu tên cũ để định nghĩa phép đổi D-RM-02"},
	"docs/PROMPT-DOI-TEN-VA-LUONG-MOI.md":          {19, "HISTORICAL_EVIDENCE: yêu cầu đổi tên D-RM-02, nêu tên cũ để định nghĩa phép đổi"},
	"docs/ROADMAP.md":                              {96, "HISTORICAL_EVIDENCE: tài liệu tra cứu theo mốc 8280f40 (khung D-RM-01)"},
	"docs/adr/0006-bo-giu-lai-doc-lap.md":          {1, "HISTORICAL_EVIDENCE: muối băm chia bộ giữ lại (§4); đổi chuỗi là đổi cách chia H1 / H2 đã đóng băng"},
	"docs/bao-cao/2026-10-20-win-mvp.md":           {1, "HISTORICAL_EVIDENCE: báo cáo ngày 20/10"},
	"docs/bao-cao/2026-10-21-ca-ngay.md":           {7, "HISTORICAL_EVIDENCE: báo cáo ca 21/10"},
	"docs/bao-cao/2026-10-21-migration-eacl.md":    {27, "HISTORICAL_EVIDENCE: báo cáo đợt đổi tên sang EACL sáng 21/10"},
	"docs/bao-cao/2026-10-21-overnight.md":         {8, "HISTORICAL_EVIDENCE: báo cáo đêm 20→21/10"},
	"docs/bao-cao/2026-10-21-phan-hoi-roadmap.md":  {4, "HISTORICAL_EVIDENCE: phản hồi audit ROADMAP (bản giữa chừng 21/10)"},
	"docs/bao-cao/TIEN-DO-BUILD.md":                {1, "HISTORICAL_EVIDENCE: các dòng nhật ký đã ghi"},
	"docs/g7/CHECKPOINT-A-H.md":                    {9, "HISTORICAL_EVIDENCE: checkpoint G7 20/10"},
	"docs/g7/CONTAMINATION-AUDIT.md":               {21, "HISTORICAL_EVIDENCE: kiểm toán nhiễm 20/10"},
	"docs/g7/REVIEW-D5.md":                         {49, "HISTORICAL_EVIDENCE: bản soát 20/10"},
	"docs/g7/holdout-h3/ket-qua-h3/chay.txt":       {1, "HISTORICAL_EVIDENCE: nhật ký lần chạy H3 duy nhất (ghi tên database lúc chạy)"},
	"docs/g7/holdout/goi-y-lo-cho-phien-chinh.txt": {1, "HISTORICAL_EVIDENCE: ghi chú của người soạn bộ giữ lại"},
	"docs/g7/holdout/holdout-manifest.json":        {8, "IMMUTABLE_HASHED_ARTIFACT: manifest đóng băng H1/H2"},
	"docs/g7/holdout/ket-qua-h1/phan-tich.md":      {3, "HISTORICAL_EVIDENCE: phân tích lần chạy H1 duy nhất"},
	"docs/g7/holdout/kho-soat.md":                  {3, "HISTORICAL_EVIDENCE: bản soát kho ca"},
	"docs/g7/holdout/loai-truoc-dong-bang.txt":     {9, "IMMUTABLE_HASHED_ARTIFACT: danh sách loại trước đóng băng (băm trong manifest)"},
	"docs/ops/README.md":                           {9, "COMPATIBILITY_ALIAS: ghi bí danh EACL_*, cách gỡ container tên cũ; tên tệp báo cáo lịch sử (EXTERNAL_REFERENCE)"},
	"docs/study/CLAUDE-EACL-DRAFT.md":              {15, "HISTORICAL_EVIDENCE: tài liệu học phiên sáng 21/10, câu gắn danh tính lúc đó"},
	"docs/study/README.md":                         {7, "HISTORICAL_EVIDENCE: tài liệu học phiên sáng 21/10"},
	"docs/study/legacy-to-eacl.md":                 {12, "HISTORICAL_EVIDENCE: tài liệu học phiên sáng 21/10, câu gắn danh tính lúc đó"},
	"docs/v3/00-CONTEXT-AUDIT.md":                  {13, "HISTORICAL_EVIDENCE: kế hoạch V3 20/10 (tên riêng 'EACL V3', nguồn ở máy docs/sources/eacl-v3/)"},
	"docs/v3/01-CURRENT-STATE-MATRIX.md":           {9, "HISTORICAL_EVIDENCE: kế hoạch V3 20/10 (tên riêng 'EACL V3', nguồn ở máy docs/sources/eacl-v3/) + chỗ đợt đổi tên sáng 21/10 (35c8e61) đổi máy móc từ tên gốc"},
	"docs/v3/02-BASELINE-EVIDENCE.md":              {6, "HISTORICAL_EVIDENCE: kế hoạch V3 20/10 (tên riêng 'EACL V3', nguồn ở máy docs/sources/eacl-v3/) + chỗ đợt đổi tên sáng 21/10 (35c8e61) đổi máy móc từ tên gốc"},
	"docs/v3/03-REQUIREMENTS-TRACEABILITY.md":      {4, "HISTORICAL_EVIDENCE: kế hoạch V3 20/10 (tên riêng 'EACL V3', nguồn ở máy docs/sources/eacl-v3/) + chỗ đợt đổi tên sáng 21/10 (35c8e61) đổi máy móc từ tên gốc"},
	"docs/v3/04-IMPLEMENTATION-DAG.md":             {2, "HISTORICAL_EVIDENCE: kế hoạch V3 20/10 (tên riêng 'EACL V3', nguồn ở máy docs/sources/eacl-v3/)"},
	"docs/v3/05-R0-DEFINITION-FREEZE.md":           {17, "HISTORICAL_EVIDENCE: kế hoạch V3 20/10 (tên riêng 'EACL V3', nguồn ở máy docs/sources/eacl-v3/) + chỗ đợt đổi tên sáng 21/10 (35c8e61) đổi máy móc từ tên gốc"},
	"docs/v3/06-MVP-TASK-BACKLOG.md":               {16, "HISTORICAL_EVIDENCE: kế hoạch V3 20/10 (tên riêng 'EACL V3', nguồn ở máy docs/sources/eacl-v3/) + chỗ đợt đổi tên sáng 21/10 (35c8e61) đổi máy móc từ tên gốc"},
	"docs/v3/08-RISK-REGISTER.md":                  {7, "HISTORICAL_EVIDENCE: kế hoạch V3 20/10 (tên riêng 'EACL V3', nguồn ở máy docs/sources/eacl-v3/)"},
	"docs/v3/09-EVIDENCE-INDEX.md":                 {3, "HISTORICAL_EVIDENCE: kế hoạch V3 20/10 (tên riêng 'EACL V3', nguồn ở máy docs/sources/eacl-v3/) + chỗ đợt đổi tên sáng 21/10 (35c8e61) đổi máy móc từ tên gốc"},
	"intelligence/models/messages.go":              {2, "COMPATIBILITY_ALIAS: modelEnv đọc EACL_MODEL_* (tên trước D-RM-02, D-G7-8) khi biến EAIS_ không đặt"},
	"intelligence/models/messages_params_test.go":  {8, "COMPATIBILITY_ALIAS: test của modelEnv"},
	"platform/pgstore/env.go":                      {1, "COMPATIBILITY_ALIAS: EnvDSN đọc EACL_PG_* khi biến EAIS_ không đặt"},
	"platform/pgstore/env_test.go":                 {6, "COMPATIBILITY_ALIAS: test của EnvDSN"},
	"platform/pgstore/pgstore_test.go":             {6, "COMPATIBILITY_ALIAS: test 0002 — vai eacl_app không còn quyền, GUC eacl.tenant_id không mở dòng"},
	"scripts/chay-giu-lai.sh":                      {3, "COMPATIBILITY_ALIAS: đọc DSN EACL_PG_* như pgstore.EnvDSN"},
	"scripts/dong-bang-giu-lai.py":                 {2, "HISTORICAL_EVIDENCE: muối băm chia bộ giữ lại (SALT); đổi chuỗi là đổi cách chia H1 / H2 đã đóng băng"},
	"scripts/dot-bien/n42-danh-tinh.sh":            {4, "FALSE_POSITIVE: chuỗi đột biến của chính luật đếm của cổng này (DT17, DT18)"},
	"scripts/dot-bien/w4-pg-rls.sh":                {5, "FALSE_POSITIVE: chuỗi đột biến của migration 0002 (GUC / vai tên cũ)"},
	"scripts/du-toan-g7.py":                        {2, "COMPATIBILITY_ALIAS: đọc mã hệ \"eacl\" trong bằng chứng thô đã lưu trước D-RM-02"},
	"scripts/kiem-ke-doi-ten.py":                   {24, "FALSE_POSITIVE: mẫu đếm và luật xếp nhóm của công cụ kiểm kê đổi tên (PROMPT-DOI-TEN §2.1)"},
	"tenants/construction/win/d5probe/main.go":     {2, "HISTORICAL_EVIDENCE: chương trình đo của bản soát D5 (20/10), giữ nguyên văn nhãn in ra"},
}

// skipped: thư mục / tệp không thuộc nội dung repo. Git không bao giờ theo dõi mục tên .git (thư mục hay tệp); nguồn riêng của chủ
// dự án, sổ ca ở máy và worktree mà harness dựng trong repo (.claude/worktrees/) đều bị git bỏ qua (.gitignore). Chỉ đúng các đường
// này: thư mục có mục .git hỏng / rỗng vẫn bị quét, vì git vẫn commit nội dung của nó.
func skipped(rel string, d fs.DirEntry) bool {
	if d.Name() == ".git" {
		return true
	}
	if d.IsDir() {
		return rel == "docs/sources" || rel == ".claude/worktrees"
	}
	return strings.HasSuffix(rel, ".local.md")
}

// legacyHits trả các vi phạm của cổng danh tính dưới root: tệp ngoài danh sách cho phép mà có mẫu, hoặc tệp trong danh
// sách mà vượt số lần cho phép.
func legacyHits(t *testing.T, root string, allowed map[string]allowance) []string {
	t.Helper()
	var bad []string
	seen := map[string]bool{}
	for k, a := range allowed { // hạn mức đúng-bằng chỉ có nghĩa theo từng tệp: mục thư mục phải là 0 (không giới hạn)
		if strings.HasSuffix(k, "/") && a.max > 0 {
			bad = append(bad, k+": mục thư mục có hạn mức "+strconv.Itoa(a.max)+": thư mục chỉ được 0 (không giới hạn)")
		}
	}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if skipped(rel, d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if bytes.IndexByte(b, 0) >= 0 { // tệp nhị phân
			return nil
		}
		n := legacyCount(b) + legacyCount([]byte(rel)) // nội dung và chính đường dẫn (tên tệp có tên cũ cũng là một lần)
		if n == 0 {
			return nil
		}
		for k, a := range allowed {
			if rel == k || (strings.HasSuffix(k, "/") && strings.HasPrefix(rel, k)) {
				seen[k] = true
				if a.max > 0 && n > a.max {
					bad = append(bad, rel+": "+strconv.Itoa(n)+" lần, cho phép "+strconv.Itoa(a.max)+" ("+a.reason+")")
				}
				if a.max > 0 && n < a.max {
					bad = append(bad, rel+": "+strconv.Itoa(n)+" lần, hạn mức "+strconv.Itoa(a.max)+" thừa: hạ xuống "+strconv.Itoa(n)+" ("+a.reason+")")
				}
				return nil
			}
		}
		bad = append(bad, rel+": "+strconv.Itoa(n)+" lần, không có lý do được ghi")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for k, a := range allowed { // mục cho tệp không còn lần nào: hạn mức thừa nhất
		if a.max > 0 && !strings.HasSuffix(k, "/") && !seen[k] {
			bad = append(bad, k+": 0 lần, hạn mức "+strconv.Itoa(a.max)+" thừa: bỏ mục ("+a.reason+")")
		}
	}
	sort.Strings(bad)
	return bad
}

// Repo hiện tại: mọi lần xuất hiện tên cũ đều có lý do.
func TestDanhTinhChiConEAIS(t *testing.T) {
	for _, v := range legacyHits(t, ".", legacyAllowed) {
		t.Error(v)
	}
}

// Ca âm: cổng bắt được tên cũ ở tệp hiện hành và tệp được phép mà vượt số lần.
func TestDanhTinhBatDuocTenCu(t *testing.T) {
	root := t.TempDir()
	for p, body := range map[string]string{
		"README.md":                        "# EACL — Enterprise AI Control Layer\n",
		"docs/sources/brief.md":            "EACL trong nguồn riêng: bỏ qua\n",
		"CHANGELOG.md":                     "EACL EACL EACL\n",
		"docs/bao-cao/2026-10-20-ngay.md":  "EACL lúc đó\n",
		"cmd/x/main.go":                    "package main // vẫn gọi eacl\n",
		"tenants/construction/pack/x.yaml": "name: ok\n",
		// link git (D-RM-02 giữ nguyên): đường dẫn module, import, URL clone không tính; tên dài viết kiểu khác vẫn tính
		"core/y/y.go":     "import \"github.com/luannt2002/enterprise-ai-control-layer/core/contracts\"\n",
		"docs/clone.md":   "git clone https://github.com/luannt2002/enterprise-ai-control-layer.git && cd enterprise-ai-control-layer\n",
		"docs/ten-dai.md": "Enterprise-AI-Control-Layer · enterprise ai control layer\n",
		"core/z/z.go":     "// EACL github.com/luannt2002/enterprise-ai-control-layer\n",
		// tên dài ở mọi cách nối từ (soát C-01): gạch dưới, CamelCase / viết liền, dấu chấm, hai dấu cách, xuống dòng, khoảng trắng Unicode
		"docs/noi-tu.md": "enterprise_ai_control_layer · EnterpriseAIControlLayer · enterpriseaicontrollayer · Enterprise.AI.Control.Layer · " +
			"Enterprise  AI  Control  Layer · Enterprise\nAI Control\nLayer · Enterprise\u00a0AI\u00a0Control\u00a0Layer\n",
		// thêm các cách nối từ người soát lượt 2 tìm ra (SUA-01): ngắt dòng qua tiền tố comment / trích dẫn, gạch nối Unicode, gạch
		// ngang dài, "/", ký tự vô hình, in đậm từng từ, dấu phẩy — mỗi dòng một lần
		"docs/noi-tu-2.md": "// Enterprise AI\n// Control Layer\n" + "> Enterprise AI\n> Control Layer\n" + "Enterprise\u2010AI\u2010Control\u2010Layer\n" +
			"Enterprise\u2014AI\u2014Control\u2014Layer\n" + "Enterprise/AI/Control/Layer\n" + "Enterprise\u200bAI\u200bControl\u200bLayer\n" +
			"**Enterprise** **AI** **Control** **Layer**\n" + "Enterprise AI, Control Layer\n",
		// link git bị thay bằng NUL, không bị xoá: chữ hai bên không được dính thành "eacl" giả (soát C-04), kể cả với tên dài
		"docs/ghep.md":     "xeacenterprise-ai-control-layerlx\n",
		"docs/ghep-dai.md": "Enterpriseenterprise-ai-control-layerAIControlLayer\n",
		// tên tệp có tên cũ cũng là một lần (SUA-03)
		"docs/ghi-chu-eacl.md": "nội dung không nhắc tên\n",
		// chữ ACL cuối một từ không phải tên sản phẩm (soát C-06); tên sản phẩm dính sau chữ cái vẫn bị đếm
		"core/acl/acl.go": "func TestRetrieveACL() {} // ReleaseACL, CreateACL\n",
		"core/he/he.go":   "sysEACL, EACLPass, eaclAsker\n",
		// hạn mức thừa: tệp có 1 lần mà mục cho phép ghi 3 (soát C-05)
		"docs/thua.md":                       "EACL một lần\n",
		"docs/sources/holdout-h3/kit/kiemca": "\x00EACL",
		"CA-2026-10-21.local.md":             "sổ ca ở máy nhắc EACL\n",
		"bin/cong-cu":                        "\x00EACL nhị phân",
		// gốc là một repo thật (có .git): luật bỏ qua checkout lồng không được bỏ qua chính gốc
		".git/HEAD": "ref: refs/heads/main\n",
		// worktree mà harness dựng trong repo (git bỏ qua .claude/worktrees/): bản chép của cây khác — bỏ qua cả cây con
		".claude/worktrees/wf-x/.git":      "gitdir: /noi-khac/.git/worktrees/wf-x\n",
		".claude/worktrees/wf-x/README.md": "# EACL — bản chép cây khác\n",
		// thư mục bắt đầu bằng dấu chấm KHÔNG phải nơi git bỏ qua: vẫn là nội dung repo, vẫn bị quét
		".claude/ghi-chu.md": "ghi chú nhắc EACL\n",
		// mục .git hỏng (gitdir trỏ nơi không tồn tại): git vẫn commit nội dung thư mục — vẫn bị quét; chính mục .git thì git không
		// bao giờ theo dõi — bỏ qua (kể cả khi đường dẫn bên trong mang tên cũ)
		"stale/.git": "gitdir: /noi-khac/eacl/.git/worktrees/x\n",
		"stale/f.md": "EACL trong thư mục có .git hỏng\n",
	} {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := strings.Join(legacyHits(t, root, map[string]allowance{"CHANGELOG.md": {2, "thử"}, "docs/bao-cao/": {0, "thử"}, "docs/thua.md": {3, "thử"}, "docs/da-sua.md": {2, "thử"}, "docs/lich-su/": {4, "thử"}}), "\n")
	for _, want := range []string{"README.md: 2 lần", "CHANGELOG.md: 3 lần, cho phép 2", "cmd/x/main.go: 1 lần", ".claude/ghi-chu.md: 1 lần", "stale/f.md: 1 lần",
		"docs/ten-dai.md: 2 lần", "core/z/z.go: 1 lần", "docs/noi-tu.md: 7 lần", "core/he/he.go: 3 lần", "docs/thua.md: 1 lần, hạn mức 3 thừa: hạ xuống 1",
		"docs/da-sua.md: 0 lần, hạn mức 2 thừa: bỏ mục", "docs/noi-tu-2.md: 8 lần", "docs/ghi-chu-eacl.md: 1 lần",
		"docs/lich-su/: mục thư mục có hạn mức 4"} {
		if !strings.Contains(got, want) {
			t.Errorf("thiếu vi phạm %q trong:\n%s", want, got)
		}
	}
	for _, not := range []string{"docs/sources", "docs/bao-cao", "pack/x.yaml", ".local.md", "bin/cong-cu", ".claude/worktrees", "stale/.git",
		"core/y/y.go", "docs/clone.md", "docs/ghep.md", "core/acl/acl.go", "docs/ghep-dai.md"} {
		if strings.Contains(got, not) {
			t.Errorf("báo oan %q:\n%s", not, got)
		}
	}
}
