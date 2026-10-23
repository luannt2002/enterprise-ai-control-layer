package ragcases

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/router"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
	"go.yaml.in/yaml/v3"
)

// Dòng S7 của CỔNG TRƯỚC AI (docs/bao-cao/CONG-TRUOC-AI.md): đoạn chứa đáp án nằm trong top-3 ≥ 90%; 0 đoạn ngoài quyền;
// 100% bằng chứng có nguồn gốc. Bộ ở đây là bộ PHÁT TRIỂN (người làm RAG soạn): số của nó không phải số của cổng — xem
// docs/viec/2-tra-quy-trinh/S7-BO-CA-DOC-LAP.md.
//
// Quyền đọc và "làm căn cứ được" được tính LẠI ở đây từ ACL / phạm vi / trạng thái / hiệu lực của kho và policy của pack, không
// gọi mã của gói rag: bộ đo không chấm bằng chính thứ nó đo.
const (
	s7Dir       = "../../tenants/construction/knowledge/s7"
	s7Pack      = "../../tenants/construction/pack"
	s7MinTraLoi = 30 // ít hơn thì tỉ lệ top-3 không đủ nghĩa
)

type s7Bo struct {
	NgayMacDinh string  `yaml:"ngay_mac_dinh"`
	K           int     `yaml:"k"`
	Cau         []s7Cau `yaml:"cau"`
}

type s7Cau struct {
	Ma             string    `yaml:"ma"`
	Hoi            string    `yaml:"hoi"`
	Vai            []string  `yaml:"vai"`
	PhamVi         []string  `yaml:"pham_vi"`
	Ngay           string    `yaml:"ngay"`
	Loai           string    `yaml:"loai"` // tra_loi | khong_co
	DapAn          []s7DapAn `yaml:"dap_an"`
	Bay            string    `yaml:"bay"` // với khong_co: ngoai_quyen | ngoai_pham_vi | khong_co_tai_lieu
	TruotBietTruoc string    `yaml:"truot_biet_truoc"`
}

type s7DapAn struct {
	TaiLieu string `yaml:"tai_lieu"`
	Ban     int    `yaml:"ban"` // 0 = tài liệu không phiên bản
	Chua    string `yaml:"chua"`
}

// docS7 đọc bộ câu (khoá lạ là lỗi) và kho tài liệu.
func docS7(path string) (s7Bo, []rag.Document, error) {
	var bo s7Bo
	b, err := os.ReadFile(path)
	if err != nil {
		return bo, nil, err
	}
	dec := yaml.NewDecoder(strings.NewReader(string(b)))
	dec.KnownFields(true)
	if err := dec.Decode(&bo); err != nil {
		return bo, nil, fmt.Errorf("%s: %w", path, err)
	}
	docs, err := rag.LoadMarkdown(filepath.Join(filepath.Dir(path), "kho"), "abc-infra")
	return bo, docs, err
}

// kiemS7 kiểm bộ câu trước khi đo: một đáp án viết sai (mã, bản, chuỗi) làm ca không bao giờ trúng mà không ai biết.
func kiemS7(bo s7Bo, docs []rag.Document) error {
	if bo.K <= 0 {
		return errors.New("k phải > 0")
	}
	seen, traLoi := map[string]bool{}, 0
	for _, c := range bo.Cau {
		switch {
		case c.Ma == "" || seen[c.Ma]:
			return fmt.Errorf("mã câu rỗng hoặc trùng: %q", c.Ma)
		case strings.TrimSpace(c.Hoi) == "" || len(c.Vai) == 0:
			return fmt.Errorf("%s: thiếu câu hỏi hoặc vai", c.Ma)
		case c.Loai == "tra_loi" && (len(c.DapAn) == 0 || c.Bay != ""):
			return fmt.Errorf("%s: câu tra_loi phải có dap_an, không có bay", c.Ma)
		case c.Loai == "khong_co" && (len(c.DapAn) != 0 || c.Bay == "" || c.TruotBietTruoc != ""):
			return fmt.Errorf("%s: câu khong_co phải có bay, không có dap_an / truot_biet_truoc", c.Ma)
		case c.Loai != "tra_loi" && c.Loai != "khong_co":
			return fmt.Errorf("%s: loai lạ %q", c.Ma, c.Loai)
		}
		seen[c.Ma] = true
		if _, err := ngayS7(bo, c); err != nil {
			return fmt.Errorf("%s: %w", c.Ma, err)
		}
		if c.Loai == "tra_loi" {
			traLoi++
		}
		for _, d := range c.DapAn {
			if !slices.ContainsFunc(docs, func(x rag.Document) bool {
				return x.ID == d.TaiLieu && x.Version == d.Ban && strings.Contains(x.Body, d.Chua)
			}) {
				return fmt.Errorf("%s: đáp án %s@v%d %q không có trong kho", c.Ma, d.TaiLieu, d.Ban, d.Chua)
			}
		}
	}
	if traLoi < s7MinTraLoi {
		return fmt.Errorf("có %d câu tra_loi, cần ít nhất %d", traLoi, s7MinTraLoi)
	}
	return nil
}

func ngayS7(bo s7Bo, c s7Cau) (time.Time, error) {
	s := c.Ngay
	if s == "" {
		s = bo.NgayMacDinh
	}
	return time.Parse("2006-01-02", s)
}

// duocDoc: có một vai của người hỏi vừa nằm trong ACL vừa thoả phạm vi của tài liệu (vai scope "all" theo policy, hoặc
// thuộc tính entity_scope giao phạm vi). Kho một tenant, nên không xét tenant (cách ly tenant canh ở TestCaTriThuc / pgstore).
func duocDoc(d rag.Document, pr contracts.Principal, pol router.Policy) bool {
	attr := strings.Split(pr.Attributes["entity_scope"], ",")
	for _, r := range pr.Roles {
		if !slices.Contains(d.ACL, r) {
			continue
		}
		if len(d.Scope) == 0 || pol.Roles[r].Scope == "all" ||
			slices.ContainsFunc(attr, func(k string) bool { return slices.Contains(d.Scope, strings.TrimSpace(k)) }) {
			return true
		}
	}
	return false
}

// canCuDuoc: bản đã duyệt (hoặc không khai), đang hiệu lực vào ngày hỏi, và là bản DUY NHẤT như thế của mã đó mà người hỏi đọc được.
func canCuDuoc(d rag.Document, day time.Time, docs []rag.Document, pr contracts.Principal, pol router.Policy) bool {
	today := day.Format("2006-01-02")
	hieuLuc := func(x rag.Document) bool {
		return (x.Status == "" || x.Status == rag.StatusApproved) && (x.EffectiveFrom == "" || today >= x.EffectiveFrom) &&
			(x.EffectiveTo == "" || today < x.EffectiveTo)
	}
	n := 0
	for _, x := range docs {
		if x.ID == d.ID && duocDoc(x, pr, pol) && hieuLuc(x) {
			n++
		}
	}
	return hieuLuc(d) && n == 1
}

type s7Truy func(ctx context.Context, pr contracts.Principal, q string, k int) ([]contracts.Evidence, error)

// s7KetQua là số đo của một đường truy xuất trên một bộ câu.
type s7KetQua struct {
	traLoi, trung, traNham             int
	ngoaiQuyen, khongCanCu, thieuNguon []string // "<mã câu> <mã đoạn>"
	trungTheoCau                       map[string]bool
	top3                               map[string][]string // mã câu → mã đoạn top-3, để đọc ca trượt
}

func doS7(t *testing.T, bo s7Bo, docs []rag.Document, tenantID contracts.TenantID, rt *tenant.Runtime, truy s7Truy) s7KetQua {
	t.Helper()
	pol := rt.Policy
	kq := s7KetQua{trungTheoCau: map[string]bool{}, top3: map[string][]string{}}
	for _, c := range bo.Cau {
		day, _ := ngayS7(bo, c)
		pr := contracts.Principal{ID: "nguoi-hoi", TenantID: tenantID, Kind: contracts.PrincipalUser, Roles: c.Vai,
			Attributes: map[string]string{"entity_scope": strings.Join(c.PhamVi, ",")}}
		// Chủ đề tính đúng như pipeline (S2 của pack + pipeline.KnowledgeTopic), để bộ đo đo đúng lời gọi truy xuất của lượt hỏi.
		topic := pipeline.KnowledgeTopic(rt.Normalizer.Normalize(c.Hoi, normalization.Options{Now: day}))
		ctx := contracts.WithKnowledgeQuery(contracts.WithTenant(context.Background(), tenantID),
			contracts.KnowledgeQuery{AsOf: day, InScope: pol.InScope, Topic: topic})
		ev, err := truy(ctx, pr, c.Hoi, bo.K)
		var n *contracts.KnowledgeNotice
		if err != nil && !errors.As(err, &n) {
			t.Fatalf("%s: lỗi truy xuất: %v", c.Ma, err)
		}
		for _, e := range ev {
			key := c.Ma + " " + e.ID
			i := slices.IndexFunc(docs, func(x rag.Document) bool {
				return x.ID == e.Source.DocumentID && x.Version == e.Source.DocumentVersion
			})
			if i < 0 || e.Source.Name == "" || !strings.Contains(e.ID, "#") || (e.Source.DocumentVersion > 0) != strings.Contains(e.ID, "@v") {
				kq.thieuNguon = append(kq.thieuNguon, key)
				continue
			}
			switch {
			case !duocDoc(docs[i], pr, pol):
				kq.ngoaiQuyen = append(kq.ngoaiQuyen, key)
			case !canCuDuoc(docs[i], day, docs, pr, pol):
				kq.khongCanCu = append(kq.khongCanCu, key)
			}
		}
		if c.Loai == "khong_co" {
			if len(ev) > 0 {
				kq.traNham++
				for _, e := range ev[:min(3, len(ev))] {
					kq.top3[c.Ma] = append(kq.top3[c.Ma], e.ID)
				}
			}
			continue
		}
		kq.traLoi++
		for _, e := range ev[:min(3, len(ev))] {
			kq.top3[c.Ma] = append(kq.top3[c.Ma], e.ID)
		}
		for _, e := range ev[:min(3, len(ev))] {
			if slices.ContainsFunc(c.DapAn, func(d s7DapAn) bool {
				return e.Source.DocumentID == d.TaiLieu && e.Source.DocumentVersion == d.Ban && strings.Contains(e.Text, d.Chua)
			}) {
				kq.trung++
				kq.trungTheoCau[c.Ma] = true
				break
			}
		}
	}
	return kq
}

func (kq s7KetQua) String() string {
	dat := "KHÔNG ĐẠT"
	if 10*kq.trung >= 9*kq.traLoi && len(kq.ngoaiQuyen)+len(kq.khongCanCu)+len(kq.thieuNguon) == 0 {
		dat = "ĐẠT"
	}
	return fmt.Sprintf("top-3 trúng %d/%d (%.1f%%) · đoạn ngoài quyền %d · đoạn không làm căn cứ %d · thiếu nguồn %d · câu không có đáp án mà trả đoạn %d ⇒ dòng S7: %s",
		kq.trung, kq.traLoi, 100*float64(kq.trung)/float64(max(kq.traLoi, 1)), len(kq.ngoaiQuyen), len(kq.khongCanCu), len(kq.thieuNguon), kq.traNham, dat)
}

// kiemKetQua: hai vế tuyệt đối (0 đoạn ngoài quyền / không làm căn cứ, 100% có nguồn) và ratchet ca trượt đã biết.
func kiemKetQua(t *testing.T, ten string, bo s7Bo, kq s7KetQua) {
	t.Helper()
	t.Logf("%s: %s", ten, kq)
	for _, x := range [][]string{kq.ngoaiQuyen, kq.khongCanCu, kq.thieuNguon} {
		if len(x) > 0 {
			t.Errorf("%s: %v", ten, x)
		}
	}
	for _, c := range bo.Cau {
		switch hit := kq.trungTheoCau[c.Ma]; {
		case c.Loai == "khong_co" && len(kq.top3[c.Ma]) > 0:
			t.Logf("%s: %s (%s) không có đáp án mà nhận đoạn %v", ten, c.Ma, c.Bay, kq.top3[c.Ma])
		case c.Loai != "tra_loi":
		case !hit && c.TruotBietTruoc == "":
			t.Errorf("%s: %s trượt top-3 (%q) — top-3 %v", ten, c.Ma, c.Hoi, kq.top3[c.Ma])
		case hit && c.TruotBietTruoc != "":
			t.Errorf("%s: %s đã trúng, gỡ truot_biet_truoc", ten, c.Ma)
		}
	}
}

// napS7 đọc bộ phát triển, hoặc bộ đặt ở EAIS_S7_BO (tệp câu hỏi, kho nằm ở kho/ cạnh nó) — lần chạy bộ độc lập của cổng
// (docs/viec/2-tra-quy-trinh/S7-BO-CA-DOC-LAP.md §3).
func napS7(t *testing.T) (s7Bo, []rag.Document, *tenant.Runtime) {
	t.Helper()
	path := filepath.Join(s7Dir, "phat-trien.yaml")
	if p := os.Getenv("EAIS_S7_BO"); p != "" {
		path = p
	}
	t.Logf("bộ câu: %s", path)
	bo, docs, err := docS7(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := kiemS7(bo, docs); err != nil {
		t.Fatal(err)
	}
	rt, err := tenant.Load(s7Pack)
	if err != nil {
		t.Fatal(err)
	}
	return bo, docs, rt
}

func TestS7BoPhatTrien(t *testing.T) {
	bo, docs, rt := napS7(t)
	ix := rag.NewMemoryIndex("abc-infra", docs)
	kiemKetQua(t, "bộ nhớ", bo, doS7(t, bo, docs, "abc-infra", rt, ix.Retrieve))
}

// Đường PG của ứng dụng trên cùng bộ: cùng số, cùng câu trúng như bộ nhớ.
func TestS7BoPhatTrienQuaPG(t *testing.T) {
	admin, app := pgstore.EnvDSN()
	if admin == "" || app == "" {
		t.Skip("chưa có EAIS_PG_ADMIN_DSN / EAIS_PG_DSN — xem deploy/docker-compose.yml")
	}
	bo, docs, rt := napS7(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	s, err := pgstore.Open(ctx, app)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(s.Close)
	tid := contracts.TenantID(fmt.Sprintf("ragpg-s7-%d", time.Now().UnixNano()))
	tctx := contracts.WithTenant(ctx, tid)
	if err := s.Load(tctx, nil, nil, docs); err != nil {
		t.Fatalf("nạp PG: %v", err)
	}
	t.Cleanup(func() { _ = s.Load(tctx, nil, nil, nil) })
	pg := doS7(t, bo, docs, tid, rt, s.Retrieve)
	kiemKetQua(t, "PG", bo, pg)
	mem := doS7(t, bo, docs, "abc-infra", rt, rag.NewMemoryIndex("abc-infra", docs).Retrieve)
	for _, c := range bo.Cau {
		if mem.trungTheoCau[c.Ma] != pg.trungTheoCau[c.Ma] {
			t.Errorf("%s: bộ nhớ trúng %v, PG trúng %v", c.Ma, mem.trungTheoCau[c.Ma], pg.trungTheoCau[c.Ma])
		}
	}
}

// Phép đo phải phân biệt được: bộ đếm đỏ đúng khi đường truy xuất lộ tài liệu ngoài quyền, trả bản nháp, hay thiếu nguồn;
// bộ kiểm chặn đáp án viết sai. Đo trên một bộ mà mọi thứ đều "sạch" thì không biết bộ đếm có cắn hay không.
func TestS7PhepDoPhanBiet(t *testing.T) {
	bo, docs, rt := napS7(t)
	var moToang []rag.Document // bỏ ACL, giữ phạm vi: ai cũng đọc được mọi tài liệu không khai phạm vi
	for _, d := range docs {
		d.ACL = nil
		for _, c := range bo.Cau {
			d.ACL = append(d.ACL, c.Vai...)
		}
		moToang = append(moToang, d)
	}
	if kq := doS7(t, bo, docs, "abc-infra", rt, rag.NewMemoryIndex("abc-infra", moToang).Retrieve); len(kq.ngoaiQuyen) == 0 {
		t.Errorf("chỉ mục bỏ ACL (phạm vi giữ nguyên) mà bộ đếm 'ngoài quyền' = 0: %s", kq)
	}
	var boPhamVi []rag.Document // giữ ACL, bỏ phạm vi: kỹ sư PRJ-B thấy biện pháp của PRJ-A (S7D-37)
	for _, d := range docs {
		d.Scope = nil
		boPhamVi = append(boPhamVi, d)
	}
	if kq := doS7(t, bo, docs, "abc-infra", rt, rag.NewMemoryIndex("abc-infra", boPhamVi).Retrieve); len(kq.ngoaiQuyen) == 0 {
		t.Errorf("chỉ mục bỏ phạm vi (ACL giữ nguyên) mà bộ đếm 'ngoài quyền' = 0: %s", kq)
	}
	nhap := func(_ context.Context, _ contracts.Principal, _ string, _ int) ([]contracts.Evidence, error) {
		return []contracts.Evidence{{ID: "SOP-NL-01@v3#3", Text: "x", Source: contracts.SourceRef{Name: "x", DocumentID: "SOP-NL-01", DocumentVersion: 3}}}, nil
	}
	if kq := doS7(t, bo, docs, "abc-infra", rt, nhap); len(kq.khongCanCu) == 0 {
		t.Errorf("trả bản nháp mà bộ đếm 'không làm căn cứ' = 0: %s", kq)
	}
	khongNguon := func(_ context.Context, _ contracts.Principal, _ string, _ int) ([]contracts.Evidence, error) {
		return []contracts.Evidence{{ID: "SOP-NL-02#1", Text: "x", Source: contracts.SourceRef{DocumentID: "SOP-NL-02"}}}, nil
	}
	if kq := doS7(t, bo, docs, "abc-infra", rt, khongNguon); len(kq.thieuNguon) == 0 {
		t.Errorf("đoạn thiếu tên nguồn mà bộ đếm 'thiếu nguồn' = 0: %s", kq)
	}
	sai := bo
	sai.Cau = slices.Clone(bo.Cau)
	sai.Cau[0].DapAn = []s7DapAn{{TaiLieu: "SOP-NL-01", Ban: 2, Chua: "trong vòng 36 giờ"}}
	if err := kiemS7(sai, docs); err == nil {
		t.Error("đáp án không có trong kho mà bộ kiểm không chặn")
	}
}
