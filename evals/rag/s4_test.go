package ragcases

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/router"
	"github.com/luannt2002/enterprise-ai-control-layer/simulations/fixtures"
	"go.yaml.in/yaml/v3"
)

// Đo các phương án S4 trên luồng hỏi TRỌN (docs/ROADMAP-RAG.md §10.2): B0 luật hiện tại · B1 luật + dò chỉ mục tri thức khi luật không
// ra kế hoạch trả lời (tất định, 0 lời gọi model) · B2 luật + model phân loại ở đúng các câu đó · B3 model phân loại MỌI câu, code vẫn giữ
// quyền / phạm vi / khe và chốt câu hỏi lượng; B4 = B1 trước, model chỉ khi chỉ mục im. B2 / B3 / B4 chỉ chạy khi cấu hình model phân loại (EAIS_DECIDE_*, khuôn ở .env.example) — câu
// hỏi được GỬI RA nhà cung cấp: chỉ bộ giả lập / công khai.
//
// Bộ câu: EAIS_S4_BO (mặc định bộ S7 phát triển), kho ở EAIS_S4_KHO (mặc định kho/ cạnh tệp). Ngoài loai của S7 (tra_loi, khong_co) bộ
// có thể có so_lieu (câu số liệu: KHÔNG được tới route tri thức) và ca_hai (số + quy trình). Phương án: EAIS_S4_PHUONG_AN (mặc định
// "B0,B1"). EAIS_S4_RA: tệp JSONL ghi kết quả thô từng câu (đặt NGOÀI repo) — chấm lại không cần chạy lại. Đây là phép ĐO (in số);
// vế tuyệt đối: 0 đoạn ngoài quyền được trích, 0 câu so_lieu tới route tri thức ở B1.
func TestS4PhuongAn(t *testing.T) {
	bo, docs, kho := napBoS4(t)
	t.Logf("bộ %s · kho %s · %d câu · %d tài liệu", os.Getenv("EAIS_S4_BO"), kho, len(bo.Cau), len(docs))
	phuongAn := strings.Split(cmp.Or(os.Getenv("EAIS_S4_PHUONG_AN"), "B0,B1"), ",")
	if !slices.Contains(phuongAn, "B0") || phuongAn[0] != "B0" { // B0 là mốc của "đổi kết quả": luôn chạy trước
		phuongAn = append([]string{"B0"}, slices.DeleteFunc(phuongAn, func(x string) bool { return x == "B0" })...)
	}
	var ra *json.Encoder
	if p := os.Getenv("EAIS_S4_RA"); p != "" {
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { f.Close() })
		ra = json.NewEncoder(f)
	}
	// Hai chính sách: policy NGUYÊN của pack, và policy phủ thêm documents.read (phạm vi theo thuộc tính) cho mọi vai có trong ACL của kho
	// mà policy không khai — đo 24/10: pack khai 5 vai, ACL kho dùng 16 ⇒ câu tri thức của 12 vai bị S5 chặn dù S4 đúng. Lớp phủ chỉ có
	// trong phép đo này, để tách lỗi S4 khỏi lỗi cấu hình vai; không sửa pack.
	for _, phu := range []bool{false, true} {
		rt := napPack(t)
		ten := "policy của pack"
		if phu {
			them := phuVaiTaiLieu(rt, docs)
			ten = fmt.Sprintf("policy phủ %d vai của ACL kho", len(them))
		}
		dir, err := fixtures.LoadDirectory("../../tenants/construction/seed/entities.yaml")
		if err != nil {
			t.Fatal(err)
		}
		facts, err := fixtures.LoadFacts("../../tenants/construction/seed/facts.yaml", rt, dir)
		if err != nil {
			t.Fatal(err)
		}
		moc := map[string]contracts.Outcome{} // kết quả B0 theo mã câu: đo hồi quy của câu so_lieu
		for _, pa := range phuongAn {
			ix := rag.NewMemoryIndex("abc-infra", docs)
			fb, dem, ok := duPhongS4(t, pa, ix)
			if !ok {
				t.Logf("[%s] %s: bỏ qua — cần model phân loại (EAIS_DECIDE_*, khuôn ở .env.example; câu hỏi được gửi ra ngoài)", ten, pa)
				continue
			}
			var kq s4KetQua
			for _, c := range bo.Cau {
				day, _ := ngayS4(bo, c)
				ix.Now = func() time.Time { return day.Add(10 * time.Hour) }
				pr := contracts.Principal{ID: "nguoi-hoi", TenantID: "abc-infra", Kind: contracts.PrincipalUser, Roles: c.Vai,
					Attributes: map[string]string{"entity_scope": strings.Join(c.PhamVi, ",")}}
				a := &pipeline.Ask{RT: rt, Deps: pipeline.Deps{Directory: dir, Facts: facts, Knowledge: ix, Model: &models.Extractive{},
					IntentFallback: fb, Now: func() time.Time { return day.Add(10 * time.Hour) }}}
				t0 := time.Now()
				o, err := a.Handle(context.Background(), pr, contracts.AskRequest{ConversationID: "s4", MessageID: c.Ma, Text: c.Hoi})
				kq.tre = append(kq.tre, time.Since(t0))
				if pa == "B0" {
					moc[c.Ma] = o.Response.Outcome
				}
				kq.cham(t, pa, c, o, err, docs, moc, rt.Policy, day)
				moHinh := dem.cuoi
				dem.cuoi = ""
				if ra != nil {
					_ = ra.Encode(map[string]any{"policy": ten, "pa": pa, "ma": c.Ma, "loai": c.Loai, "bay": c.Bay, "hoi": c.Hoi, "loi": fmt.Sprint(err), "mo_hinh": moHinh,
						"route": o.Response.Route, "outcome": o.Response.Outcome, "intent": o.Plan.Intent.Name, "ly_do": o.Plan.Reason,
						"s4": o.Decision.Why.Intent.Reasons, "trich": o.Response.Citations, "tra_loi": o.Response.Answer})
				}
			}
			t.Logf("[%s] %s: %s · số câu phải hỏi model %d (lỗi %d)", ten, pa, kq, dem.goi, dem.loi)
			if len(kq.ro) > 0 {
				t.Errorf("%s: trích tài liệu người hỏi không được đọc / không làm căn cứ / không có trong kho: %v", pa, kq.ro)
			}
			if kq.soLieuSangTriThuc > 0 {
				t.Errorf("%s: %d câu so_lieu tới route tri thức", pa, kq.soLieuSangTriThuc)
			}
		}
	}
}

// phuVaiTaiLieu thêm vào policy mọi vai có trong ACL của kho mà policy chưa khai, với đúng documents.read và phạm vi theo thuộc tính.
func phuVaiTaiLieu(rt *tenant.Runtime, docs []rag.Document) []string {
	var them []string
	for _, d := range docs {
		for _, v := range d.ACL {
			if _, ok := rt.Policy.Roles[v]; !ok {
				rt.Policy.Roles[v] = router.RoleDef{Capabilities: []string{"documents.read"}, Scope: "attribute"}
				them = append(them, v)
			}
		}
	}
	return them
}

const s4YDinhTriThuc = "sop_question"

type s4Cau struct {
	s7Cau     `yaml:",inline"`
	YDinhDung string `yaml:"y_dinh_dung"`
}

type s4Bo struct {
	NgayMacDinh string  `yaml:"ngay_mac_dinh"`
	K           int     `yaml:"k"`
	Cau         []s4Cau `yaml:"cau"`
}

func napBoS4(t *testing.T) (s4Bo, []rag.Document, string) {
	t.Helper()
	path := cmp.Or(os.Getenv("EAIS_S4_BO"), filepath.Join(s7Dir, "phat-trien.yaml"))
	kho := cmp.Or(os.Getenv("EAIS_S4_KHO"), filepath.Join(filepath.Dir(path), "kho"))
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var bo s4Bo
	dec := yaml.NewDecoder(strings.NewReader(string(b)))
	dec.KnownFields(true)
	if err := dec.Decode(&bo); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	docs, err := rag.LoadMarkdown(kho, "abc-infra")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range bo.Cau {
		switch c.Loai {
		case "tra_loi", "ca_hai", "khong_co", "so_lieu":
		default:
			t.Fatalf("%s: loai lạ %q", c.Ma, c.Loai)
		}
		for _, d := range c.DapAn {
			if !slices.ContainsFunc(docs, func(x rag.Document) bool { return dungBan(x, d) && strings.Contains(x.Body, d.Chua) }) {
				t.Fatalf("%s: đáp án %s@v%d %q không có trong kho", c.Ma, d.TaiLieu, d.Ban, d.Chua)
			}
		}
	}
	return bo, docs, kho
}

// dungBan: tài liệu không có phần đầu phiên bản (Version 0) khớp đáp án ghi ban 0 hoặc 1 — quy ước khai TRƯỚC lần chạy bộ độc lập S4
// (người soạn ghi ban 1 cho tài liệu không phiên bản).
func dungBan(x rag.Document, d s7DapAn) bool {
	return x.ID == d.TaiLieu && (x.Version == d.Ban || (x.Version == 0 && d.Ban == 1))
}

func ngayS4(bo s4Bo, c s4Cau) (time.Time, error) {
	return ngayS7(s7Bo{NgayMacDinh: bo.NgayMacDinh}, c.s7Cau)
}

func napPack(t *testing.T) *tenant.Runtime {
	t.Helper()
	rt, err := tenant.Load(s7Pack)
	if err != nil {
		t.Fatal(err)
	}
	return rt
}

type s4KetQua struct {
	n                                                       map[string]int // số câu theo loai
	s4Dung, trichDung                                       map[string]int // theo loai
	hoiLai, ngoaiPhamVi, thieuQuyenCauHinh, loi, traLoiNham int
	soLieuSangTriThuc, soLieuDoi                            int
	ro                                                      []string
	tre                                                     []time.Duration
}

// cham chấm một câu. "Trích đúng" = trả lời, trích đúng tài liệu + bản của đáp án VÀ câu trả lời chứa chuỗi đáp án (soát độc lập 24/10: chỉ
// so tài liệu thì tính đúng cả câu trích nhầm đoạn). Rò = trích tài liệu người hỏi không được đọc / không làm căn cứ ở ngày hỏi / không có
// trong kho — cùng luật với dòng S7 (duocDoc, canCuDuoc).
func (k *s4KetQua) cham(t *testing.T, pa string, c s4Cau, o pipeline.Outcome, err error, docs []rag.Document, moc map[string]contracts.Outcome,
	pol router.Policy, day time.Time) {
	t.Helper()
	if k.n == nil {
		k.n, k.s4Dung, k.trichDung = map[string]int{}, map[string]int{}, map[string]int{}
	}
	loai := c.Loai
	if c.Loai == "khong_co" {
		loai += "/" + c.Bay
	}
	k.n[loai]++
	if err != nil {
		k.loi++
		t.Logf("  %s %s lỗi: %v", pa, c.Ma, err)
		return
	}
	r, plan := o.Response, o.Plan
	triThuc := plan.Intent.Name == s4YDinhTriThuc || r.Route == contracts.RouteHybrid
	pr := contracts.Principal{ID: "nguoi-hoi", TenantID: "abc-infra", Kind: contracts.PrincipalUser, Roles: c.Vai,
		Attributes: map[string]string{"entity_scope": strings.Join(c.PhamVi, ",")}}
	for _, ci := range r.Citations {
		switch d := timTaiLieu(docs, ci.Source.DocumentID, ci.Source.DocumentVersion); {
		case ci.Source.DocumentID == "": // bằng chứng số liệu (SQL / phép tính), không phải tài liệu
		case d == nil:
			k.ro = append(k.ro, c.Ma+"→"+ci.Source.DocumentID+" (không có trong kho)")
		case !duocDoc(*d, pr, pol):
			k.ro = append(k.ro, c.Ma+"→"+d.ID+" (ngoài quyền)")
		case !canCuDuoc(*d, day, docs, pr, pol):
			k.ro = append(k.ro, c.Ma+"→"+d.ID+" (không làm căn cứ ở ngày hỏi)")
		}
	}
	gon := strings.Join(strings.Fields(r.Answer), " ")
	trich := r.Outcome == contracts.OutcomeResolved && slices.ContainsFunc(r.Citations, func(ci contracts.Citation) bool {
		return slices.ContainsFunc(c.DapAn, func(d s7DapAn) bool {
			x := timTaiLieu(docs, ci.Source.DocumentID, ci.Source.DocumentVersion)
			return x != nil && dungBan(*x, d) && strings.Contains(gon, strings.Join(strings.Fields(d.Chua), " "))
		})
	})
	switch c.Loai {
	case "tra_loi", "ca_hai":
		if triThuc {
			k.s4Dung[loai]++
		}
		if trich {
			k.trichDung[loai]++
		}
		if triThuc && r.Outcome == contracts.OutcomePermissionDenied && strings.Contains(plan.Reason, "thiếu capability") {
			k.thieuQuyenCauHinh++ // S4 đúng, S5 chặn vì vai không có capability trong policy — lỗi cấu hình pack, không phải S4
		}
		if !triThuc {
			switch r.Outcome {
			case contracts.OutcomeClarificationRequired:
				k.hoiLai++
			case contracts.OutcomeOutOfScope:
				k.ngoaiPhamVi++
			}
			t.Logf("  %s %s S4 trượt (%s %s: %s): %q", pa, c.Ma, r.Route, r.Outcome, plan.Reason, c.Hoi)
		}
	case "so_lieu":
		if r.Route == contracts.RouteRAG || plan.Intent.Name == s4YDinhTriThuc {
			k.soLieuSangTriThuc++
			t.Logf("  %s %s câu số liệu tới route tri thức: %q", pa, c.Ma, c.Hoi)
		} else {
			k.s4Dung[loai]++
		}
		if b0, ok := moc[c.Ma]; ok && pa != "B0" && b0 != r.Outcome {
			k.soLieuDoi++
			t.Logf("  %s %s câu số liệu đổi kết quả so với B0: %s → %s", pa, c.Ma, b0, r.Outcome)
		}
	case "khong_co":
		if r.Outcome == contracts.OutcomeResolved {
			k.traLoiNham++
			t.Logf("  %s %s không có đáp án / ngoài phạm vi mà trả lời: %q → %q", pa, c.Ma, c.Hoi, r.Answer)
		} else {
			k.s4Dung[loai]++ // "đúng" ở đây = không trả lời
		}
	}
}

func (k s4KetQua) String() string {
	var b strings.Builder
	for _, l := range slices.Sorted(maps.Keys(k.n)) {
		fmt.Fprintf(&b, "%s %d: S4 đúng %d", l, k.n[l], k.s4Dung[l])
		if l == "tra_loi" || l == "ca_hai" {
			fmt.Fprintf(&b, ", trích đúng đoạn %d", k.trichDung[l])
		}
		b.WriteString(" · ")
	}
	tre := slices.Clone(k.tre)
	slices.Sort(tre)
	fmt.Fprintf(&b, "câu tri thức S4 trượt: hỏi lại %d, ngoài phạm vi %d · S5 chặn vì cấu hình vai %d · không đáp án mà trả lời %d · "+
		"so_lieu tới tri thức %d, đổi kết quả %d · lỗi %d · trễ p50 %s p95 %s", k.hoiLai, k.ngoaiPhamVi, k.thieuQuyenCauHinh, k.traLoiNham,
		k.soLieuSangTriThuc, k.soLieuDoi, k.loi, tre[len(tre)/2].Round(time.Microsecond), tre[len(tre)*95/100].Round(time.Microsecond))
	return b.String()
}

func timTaiLieu(docs []rag.Document, id string, v int) *rag.Document {
	for i := range docs {
		if docs[i].ID == id && docs[i].Version == v {
			return &docs[i]
		}
	}
	return nil
}

// demGoi đếm lời gọi model của một phương án.
type demGoi struct {
	mu       sync.Mutex
	goi, loi int
	cuoi     string // quyết định của model ở câu vừa chạy (ghi JSONL)
}

// duPhongS4 dựng móc S4 của phương án; B2 / B3 / B4 cần model phân loại — chưa cấu hình thì ok = false (chỉ bỏ phương án đó).
func duPhongS4(t *testing.T, pa string, ix *rag.MemoryIndex) (pipeline.IntentFallback, *demGoi, bool) {
	t.Helper()
	dem := &demGoi{}
	switch pa {
	case "B0":
		return nil, dem, true
	case "B1":
		return pipeline.KnowledgeProbe{Knowledge: ix, Name: s4YDinhTriThuc}, dem, true
	}
	// B2 / B3 / B4, có thể kèm ngưỡng: "B2@0.6" (không kèm = EAIS_S4_NGUONG, trống = 0.5)
	ten, so, co := strings.Cut(pa, "@")
	nguong := nguongDanhSach(t, "EAIS_S4_NGUONG", 0.5)[0]
	if co {
		f, err := strconv.ParseFloat(so, 64)
		if err != nil || f < 0 || f > 1 {
			t.Fatalf("phương án %q: ngưỡng phải là số trong [0, 1]", pa)
		}
		nguong = f
	}
	switch ten {
	case "B2", "B3", "B4":
		qd := quyetDinhTuMoiTruong(t)
		if qd == nil {
			return nil, dem, false
		}
		m := moHinhS4{qd: qd, moi: ten == "B3", nguong: nguong, dem: dem}
		if ten == "B4" {
			m.truoc = &pipeline.KnowledgeProbe{Knowledge: ix, Name: s4YDinhTriThuc}
		}
		return m, dem, true
	}
	t.Fatalf("phương án lạ %q", pa)
	return nil, dem, false
}

// Bộ chấm S4 (soát độc lập 24/10): rò bắt tài liệu ngoài quyền và trích dẫn không có trong kho, KHÔNG tính bằng chứng số liệu; "trích
// đúng đoạn" đòi câu trả lời chứa đáp án, không chỉ đúng tài liệu.
func TestChamS4(t *testing.T) {
	day := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	docs := []rag.Document{{ID: "SOP", Tenant: "abc-infra", Title: "SOP", ACL: []string{"ks"}, Body: "1. Thay nhớt sau 250 giờ."},
		{ID: "MAT", Tenant: "abc-infra", Title: "Mật", ACL: []string{"gd"}, Body: "1. Két sắt."}}
	pol := router.Policy{Roles: map[string]router.RoleDef{"ks": {Capabilities: []string{"documents.read"}, Scope: "all"}, "gd": {Capabilities: []string{"documents.read"}, Scope: "all"}}}
	c := s4Cau{s7Cau: s7Cau{Ma: "X", Vai: []string{"ks"}, Loai: "tra_loi", DapAn: []s7DapAn{{TaiLieu: "SOP", Chua: "250 giờ"}}}}
	tl := func(answer string, ids ...string) pipeline.Outcome {
		var ci []contracts.Citation
		for _, id := range ids {
			ci = append(ci, contracts.Citation{Source: contracts.SourceRef{DocumentID: id}})
		}
		return pipeline.Outcome{Response: contracts.AskResponse{Outcome: contracts.OutcomeResolved, Answer: answer, Citations: ci}}
	}
	for _, x := range []struct {
		ten       string
		o         pipeline.Outcome
		ro, trich int
	}{
		{"đúng đoạn", tl("Trích: 1. Thay nhớt sau 250   giờ.", "SOP"), 0, 1},
		{"đúng tài liệu, sai đoạn", tl("Trích: 2. Kiểm tra nhớt hằng ngày.", "SOP"), 0, 0},
		{"tài liệu ngoài quyền", tl("Két sắt.", "MAT"), 1, 0},
		{"không có trong kho", tl("x", "LA"), 1, 0},
		{"bằng chứng số liệu", tl("3.850 L", ""), 0, 0},
	} {
		var k s4KetQua
		k.cham(t, "B?", c, x.o, nil, docs, nil, pol, day)
		if len(k.ro) != x.ro || k.trichDung["tra_loi"] != x.trich {
			t.Errorf("%s: rò %v (muốn %d), trích đúng đoạn %d (muốn %d)", x.ten, k.ro, x.ro, k.trichDung["tra_loi"], x.trich)
		}
	}
}
