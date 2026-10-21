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
)

// Cổng danh tính: repo chỉ có MỘT danh tính sản phẩm hiện hành là EAIS (Enterprise AI Systems, D-RM-02). Tên dùng từ sáng tới
// tối 21/10/2026 (mẫu dưới đây) chỉ được xuất hiện ở các tệp có lý do ghi trong legacyAllowed, không nhiều hơn số lần đã cho
// phép: bằng chứng đóng băng không được sửa cho đẹp tên, còn tài liệu / mã hiện hành không được để tên cũ quay lại.
var legacyIdentity = regexp.MustCompile(`(?i)eacl|enterprise[- ]ai[- ]control[- ]layer`)

// gitLink là tên repo, cũng là đường dẫn module Go và mọi import. D-RM-02 giữ nguyên link git, nên cổng bỏ đúng chuỗi này
// (chữ thường, gạch nối) trước khi đếm; mọi dạng viết khác của tên dài vẫn bị đếm.
const gitLink = "enterprise-ai-control-layer"

type allowance struct {
	max    int
	reason string // HISTORICAL_EVIDENCE · IMMUTABLE_HASHED_ARTIFACT · COMPATIBILITY_ALIAS · EXTERNAL_REFERENCE · FALSE_POSITIVE · PENDING_RENAME
}

// legacyAllowed: tiền tố đường dẫn (thư mục kết thúc bằng "/") hoặc tệp → số lần tối đa (0 = không giới hạn) + lý do.
// Mọi mục là tệp với hạn mức đúng bằng số lần hiện có: tên cũ không lan thêm được, kể cả trong tài liệu lịch sử. Chỉ chính
// tệp cổng (mẫu + ca âm) không giới hạn.
var legacyAllowed = map[string]allowance{
	"CHANGELOG.md":                                 {15, "HISTORICAL_EVIDENCE: mục ngày 21/10/2026 ghi đúng tên, module, biến và GUC lúc đó"},
	"CLAUDE.md":                                    {17, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"README.md":                                    {9, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"danh_tinh_test.go":                            {0, "FALSE_POSITIVE: chính mẫu và ca âm của cổng này"},
	"db/migrations/0001_win.sql":                   {7, "IMMUTABLE_HASHED_ARTIFACT: migration đã áp; định danh chạy đổi bằng 0002"},
	"db/migrations/0002_dinh_danh_eais.sql":        {4, "COMPATIBILITY_ALIAS: migration gỡ quyền vai eacl_app của 0001, GUC tên cũ hết tác dụng"},
	"docs/01-architecture.md":                      {11, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/02-migration-audit.md":                   {12, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/03-proof-harness.md":                     {5, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/04-core-mvp.md":                          {20, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/CHO-QUYET-DINH.md":                       {25, "HISTORICAL_EVIDENCE: văn bản các quyết định đã chốt (D-V3-*, D-G7-*, D-OP-*, D-RM-*)"},
	"docs/CONVENTIONS.md":                          {2, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/PROMPT-AUDIT-ROADMAP.md":                 {6, "HISTORICAL_EVIDENCE: yêu cầu có ngày (mốc 8280f40)"},
	"docs/PROMPT-BUILD-EACL.md":                    {8, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/PROMPT-DOI-TEN-VA-LUONG-MOI.md":          {18, "HISTORICAL_EVIDENCE: yêu cầu đổi tên D-RM-02, nêu tên cũ để định nghĩa phép đổi"},
	"docs/ROADMAP-DAY-DU.md":                       {3, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/ROADMAP.md":                              {96, "HISTORICAL_EVIDENCE: tài liệu tra cứu theo mốc 8280f40 (khung D-RM-01)"},
	"docs/STATUS.md":                               {14, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/adr/0001-win-mvp.md":                     {5, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/adr/0002-win-quyet-dinh-trien-khai.md":   {6, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/adr/0005-g7-hop-dong-va-sua-d5.md":       {13, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/adr/0006-bo-giu-lai-doc-lap.md":          {7, "IMMUTABLE_HASHED_ARTIFACT: muối băm chia bộ giữ lại đã đóng băng (§4)"},
	"docs/adr/0007-bo-cham-g7-v2.md":               {4, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/bao-cao/2026-10-20-win-mvp.md":           {1, "HISTORICAL_EVIDENCE: báo cáo ngày 20/10"},
	"docs/bao-cao/2026-10-21-ca-ngay.md":           {7, "HISTORICAL_EVIDENCE: báo cáo ca 21/10"},
	"docs/bao-cao/2026-10-21-migration-eacl.md":    {26, "HISTORICAL_EVIDENCE: báo cáo đợt đổi tên sang EACL sáng 21/10"},
	"docs/bao-cao/2026-10-21-overnight.md":         {8, "HISTORICAL_EVIDENCE: báo cáo đêm 20→21/10"},
	"docs/bao-cao/2026-10-21-phan-hoi-roadmap.md":  {4, "HISTORICAL_EVIDENCE: phản hồi audit ROADMAP (bản giữa chừng 21/10)"},
	"docs/bao-cao/TIEN-DO-BUILD.md":                {2, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/debug-handoff/README.md":                 {2, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/g7/BO-CHAM-V2-DAC-TA.md":                 {27, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/g7/CHECKPOINT-A-H.md":                    {9, "HISTORICAL_EVIDENCE: checkpoint G7 20/10"},
	"docs/g7/CHI-PHI-G7.md":                        {11, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/g7/CONTAMINATION-AUDIT.md":               {21, "HISTORICAL_EVIDENCE: kiểm toán nhiễm 20/10"},
	"docs/g7/COST-PLAN.md":                         {6, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/g7/HOP-DONG-G7.md":                       {34, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/g7/REVIEW-D5.md":                         {49, "HISTORICAL_EVIDENCE: bản soát 20/10"},
	"docs/g7/holdout-h3/ket-qua-h3/chay.txt":       {1, "HISTORICAL_EVIDENCE: nhật ký lần chạy H3 duy nhất (ghi tên database lúc chạy)"},
	"docs/g7/holdout/goi-y-lo-cho-phien-chinh.txt": {1, "HISTORICAL_EVIDENCE: ghi chú của người soạn bộ giữ lại"},
	"docs/g7/holdout/holdout-manifest.json":        {8, "IMMUTABLE_HASHED_ARTIFACT: manifest đóng băng H1/H2"},
	"docs/g7/holdout/ket-qua-h1/phan-tich.md":      {3, "HISTORICAL_EVIDENCE: phân tích lần chạy H1 duy nhất"},
	"docs/g7/holdout/kho-soat.md":                  {3, "HISTORICAL_EVIDENCE: bản soát kho ca"},
	"docs/g7/holdout/loai-truoc-dong-bang.txt":     {9, "IMMUTABLE_HASHED_ARTIFACT: danh sách loại trước đóng băng (băm trong manifest)"},
	"docs/ops/BO-NHO.md":                           {4, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/ops/LUONG-TU-CHAY.md":                    {9, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/ops/README.md":                           {31, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/study/CLAUDE-EACL-DRAFT.md":              {14, "HISTORICAL_EVIDENCE: tài liệu học phiên sáng 21/10, câu gắn danh tính lúc đó"},
	"docs/study/README.md":                         {7, "HISTORICAL_EVIDENCE: tài liệu học phiên sáng 21/10"},
	"docs/study/legacy-to-eacl.md":                 {11, "HISTORICAL_EVIDENCE: tài liệu học phiên sáng 21/10, câu gắn danh tính lúc đó"},
	"docs/tenants/construction/01-business-map.md": {13, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/tenants/construction/03-data-map.md":     {21, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/tenants/sports/01-business-map.md":       {2, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/tenants/sports/04-realtime-flow.md":      {2, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/tenants/sports/05-problem-catalog.md":    {1, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/tenants/sports/07-failure-catalog.md":    {1, "PENDING_RENAME: chưa đổi tên trong đợt D-RM-02 (gỡ khi đổi xong)"},
	"docs/v3/00-CONTEXT-AUDIT.md":                  {13, "HISTORICAL_EVIDENCE: kế hoạch V3 20/10 (tên riêng của gói kế hoạch; trỏ nguồn ở máy)"},
	"docs/v3/01-CURRENT-STATE-MATRIX.md":           {9, "HISTORICAL_EVIDENCE: kế hoạch V3 20/10 (tên riêng của gói kế hoạch; trỏ nguồn ở máy)"},
	"docs/v3/02-BASELINE-EVIDENCE.md":              {6, "HISTORICAL_EVIDENCE: kế hoạch V3 20/10 (tên riêng của gói kế hoạch; trỏ nguồn ở máy)"},
	"docs/v3/03-REQUIREMENTS-TRACEABILITY.md":      {4, "HISTORICAL_EVIDENCE: kế hoạch V3 20/10 (tên riêng của gói kế hoạch; trỏ nguồn ở máy)"},
	"docs/v3/04-IMPLEMENTATION-DAG.md":             {2, "HISTORICAL_EVIDENCE: kế hoạch V3 20/10 (tên riêng của gói kế hoạch; trỏ nguồn ở máy)"},
	"docs/v3/05-R0-DEFINITION-FREEZE.md":           {17, "HISTORICAL_EVIDENCE: kế hoạch V3 20/10 (tên riêng của gói kế hoạch; trỏ nguồn ở máy)"},
	"docs/v3/06-MVP-TASK-BACKLOG.md":               {16, "HISTORICAL_EVIDENCE: kế hoạch V3 20/10 (tên riêng của gói kế hoạch; trỏ nguồn ở máy)"},
	"docs/v3/08-RISK-REGISTER.md":                  {7, "HISTORICAL_EVIDENCE: kế hoạch V3 20/10 (tên riêng của gói kế hoạch; trỏ nguồn ở máy)"},
	"docs/v3/09-EVIDENCE-INDEX.md":                 {3, "HISTORICAL_EVIDENCE: kế hoạch V3 20/10 (tên riêng của gói kế hoạch; trỏ nguồn ở máy)"},
	"intelligence/models/messages.go":              {2, "COMPATIBILITY_ALIAS: modelEnv đọc EACL_MODEL_* (tên trước D-RM-02, D-G7-8) khi biến EAIS_ không đặt"},
	"intelligence/models/messages_params_test.go":  {8, "COMPATIBILITY_ALIAS: test của modelEnv"},
	"intelligence/rag/rag_test.go":                 {1, "FALSE_POSITIVE: TestRetrieveACLRongKhongAiDoc (chữ ACL)"},
	"platform/pgstore/env.go":                      {1, "COMPATIBILITY_ALIAS: EnvDSN đọc EACL_PG_* khi biến EAIS_ không đặt"},
	"platform/pgstore/env_test.go":                 {6, "COMPATIBILITY_ALIAS: test của EnvDSN"},
	"platform/pgstore/pgstore_test.go":             {7, "FALSE_POSITIVE (TestRetrieveACL: chữ ACL) + COMPATIBILITY_ALIAS: sau 0002 vai eacl_app và GUC eacl.tenant_id hết tác dụng"},
	"scripts/chay-giu-lai.sh":                      {3, "COMPATIBILITY_ALIAS: đọc DSN EACL_PG_* như pgstore.EnvDSN"},
	"scripts/dong-bang-giu-lai.py":                 {2, "IMMUTABLE_HASHED_ARTIFACT: muối băm chia bộ giữ lại đã đóng băng (SALT)"},
	"scripts/dot-bien/n42-danh-tinh.sh":            {2, "FALSE_POSITIVE: chuỗi đột biến của chính cổng này và của biến môi trường tương thích"},
	"scripts/dot-bien/w4-pg-rls.sh":                {5, "FALSE_POSITIVE: chuỗi đột biến của migration 0002 (GUC / vai tên cũ)"},
	"scripts/du-toan-g7.py":                        {2, "COMPATIBILITY_ALIAS: đọc mã hệ \"eacl\" trong bằng chứng thô đã lưu trước D-RM-02"},
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
		n := len(legacyIdentity.FindAllIndex(bytes.ReplaceAll(b, []byte(gitLink), nil), -1))
		if n == 0 {
			return nil
		}
		for k, a := range allowed {
			if rel == k || (strings.HasSuffix(k, "/") && strings.HasPrefix(rel, k)) {
				if a.max > 0 && n > a.max {
					bad = append(bad, rel+": "+strconv.Itoa(n)+" lần, cho phép "+strconv.Itoa(a.max)+" ("+a.reason+")")
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
		"core/y/y.go":                        "import \"github.com/luannt2002/enterprise-ai-control-layer/core/contracts\"\n",
		"docs/clone.md":                      "git clone https://github.com/luannt2002/enterprise-ai-control-layer.git && cd enterprise-ai-control-layer\n",
		"docs/ten-dai.md":                    "Enterprise-AI-Control-Layer · enterprise ai control layer\n",
		"core/z/z.go":                        "// EACL github.com/luannt2002/enterprise-ai-control-layer\n",
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
	got := strings.Join(legacyHits(t, root, map[string]allowance{"CHANGELOG.md": {2, "thử"}, "docs/bao-cao/": {0, "thử"}}), "\n")
	for _, want := range []string{"README.md: 2 lần", "CHANGELOG.md: 3 lần, cho phép 2", "cmd/x/main.go: 1 lần", ".claude/ghi-chu.md: 1 lần", "stale/f.md: 1 lần",
		"docs/ten-dai.md: 2 lần", "core/z/z.go: 1 lần"} {
		if !strings.Contains(got, want) {
			t.Errorf("thiếu vi phạm %q trong:\n%s", want, got)
		}
	}
	for _, not := range []string{"docs/sources", "docs/bao-cao", "pack/x.yaml", ".local.md", "bin/cong-cu", ".claude/worktrees", "stale/.git",
		"core/y/y.go", "docs/clone.md"} {
		if strings.Contains(got, not) {
			t.Errorf("báo oan %q:\n%s", not, got)
		}
	}
}
