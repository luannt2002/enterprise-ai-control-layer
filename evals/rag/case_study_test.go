package ragcases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/ingestion"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
	"go.yaml.in/yaml/v3"
)

// Chạy CASE STUDY qua EAIS (chỉ khi EAIS_RAG_CASE=<tệp cấu hình>): kho tài liệu từ các nguồn (rag.SourceKinds: markdown, files,
// postgres) → PG → chỉ mục → pipeline S0–S10 cho từng câu; ghi vết từng bước (traces), đánh giá, và TẦNG SAI ĐẦU TIÊN của câu trượt.
// Bộ câu ở định dạng bất kỳ (JSON / YAML) đọc qua bảng ánh xạ trường của cấu hình — không mã riêng cho từng định dạng.
// Động cơ: mặc định trên máy (từ khoá + model giả, không gửi gì ra ngoài); động cơ thật chỉ khi EAIS_RAG_DONG_CO=1 VÀ cấu hình không
// khai chi_tren_may (dữ liệu khách không được rời máy). Kết quả ghi vào thư mục "ra" của cấu hình — không vào repo.
func TestCaseStudy(t *testing.T) {
	path := os.Getenv("EAIS_RAG_CASE")
	if path == "" {
		t.Skip("đặt EAIS_RAG_CASE=<tệp cấu hình case study> để chạy")
	}
	admin, app := pgstore.EnvDSN()
	if admin == "" || app == "" {
		t.Fatal("cần EAIS_PG_ADMIN_DSN / EAIS_PG_DSN")
	}
	run, err := docCauHinhCa(path)
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Dir(path)
	rt, err := tenant.Load(resolvePath(base, run.Pack))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	kinds := rag.FileKinds(ingestion.PdfToText{}.PDFText)
	kinds["postgres"] = pgstore.KnowledgeSQL
	tid := contracts.TenantID(fmt.Sprintf("case-%d", time.Now().UnixNano()))
	docs, err := rag.Collect(ctx, run.Nguon, kinds, string(tid), base)
	if err != nil {
		t.Fatalf("nạp nguồn: %v", err)
	}
	var giu []string // tài liệu cổng chất lượng giữ lại: không phục vụ được — báo ra, không coi là đã nạp
	for _, d := range docs {
		if d.Hold != "" {
			giu = append(giu, fmt.Sprintf("%s@v%d: %s", d.ID, d.Version, d.Hold))
		}
	}
	if err := pgstore.Migrate(ctx, admin); err != nil {
		t.Fatal(err)
	}
	s, err := pgstore.Open(ctx, app)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	tctx := contracts.WithTenant(ctx, tid)
	if err := s.Load(tctx, nil, nil, docs); err != nil {
		t.Fatalf("nạp PG: %v", err)
	}
	var gw models.Gateway = &models.Extractive{}
	dongCo := "trên máy (từ khoá + model giả)"
	if os.Getenv("EAIS_RAG_DONG_CO") == "1" && !run.ChiTrenMay {
		s.Semantic = semanticTuMoiTruong(t)
		if gw, err = models.New(models.ConfigFromEnv()); err != nil {
			t.Fatal(err)
		}
		if _, _, err := s.EmbedKnowledge(ctx, tid); err != nil {
			t.Fatalf("nhúng kho: %v", err)
		}
		dongCo = "thật (cấu hình .env: truy hồi lai + model)"
	}
	t.Cleanup(func() {
		_ = s.Load(tctx, nil, nil, nil)
		if s.Semantic != nil {
			_, _, _ = s.EmbedKnowledge(ctx, tid)
		}
	})
	rtThu := *rt
	rtThu.Meta.Tenant = string(tid)
	day, err := time.ParseInLocation("2006-01-02", run.Ngay, rt.Location)
	if err != nil {
		t.Fatalf("ngay %q: %v", run.Ngay, err)
	}
	truy := &truyGhi{inner: s}
	a := &pipeline.Ask{RT: &rtThu, Deps: pipeline.Deps{Directory: s, Facts: s, Knowledge: truy, Model: gw,
		Now: func() time.Time { return day.Add(10 * time.Hour) }}}

	ra, err := filepath.Abs(resolvePath(base, run.Ra))
	if err != nil {
		t.Fatal(err)
	}
	if repo, _ := filepath.Abs("../.."); !strings.HasPrefix(must(filepath.Rel(repo, ra)), "..") {
		t.Fatalf("thư mục ra %s nằm trong repo: kết quả chứa câu hỏi / đoạn tài liệu (có thể của khách) — đặt ở ngoài repo", ra)
	}
	if err := os.MkdirAll(ra, 0o755); err != nil {
		t.Fatal(err)
	}
	jl, err := os.Create(filepath.Join(ra, "ket-qua.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer jl.Close()
	var all []ketQuaCa
	for _, bo := range run.Bo {
		cas, err := docBoCau(base, bo)
		if err != nil {
			t.Fatalf("bộ %s: %v", bo.Ten, err)
		}
		boDir := filepath.Join(ra, "vet", tenTep(bo.Ten))
		if err := os.MkdirAll(boDir, 0o755); err != nil {
			t.Fatal(err)
		}
		for i, c := range cas {
			vai := c.Vai
			if len(vai) == 0 {
				vai = run.Vai
			}
			pr := contracts.Principal{ID: "nguoi-hoi", TenantID: tid, Kind: contracts.PrincipalUser, Roles: vai}
			truy.reset()
			t0 := time.Now()
			out, herr := a.Handle(tctx, pr, contracts.AskRequest{ConversationID: bo.Ten, MessageID: c.Ma, Text: c.Hoi})
			kq := danhGia(bo.Ten, c, out, herr, truy.lan())
			kq.Ms = time.Since(t0).Milliseconds()
			vet := ""
			if herr == nil {
				rec := traces.NewRecord(traces.Turn{Decision: out.Decision, Spans: out.Spans, TotalMs: out.TotalMs,
					UnaccountedMs: out.UnaccountedMs, Plan: out.Plan, Response: out.Response})
				kq.Buoc = rec.Steps
				vet = traces.FormatTable(rec)
			}
			// Số thứ tự đứng trước: hai mã khác nhau làm sạch ra cùng tên ("a/b", "a_b") không ghi đè vết của nhau.
			if err := os.WriteFile(filepath.Join(boDir, fmt.Sprintf("%03d-%s.txt", i+1, tenTep(c.Ma))), []byte(kq.tieuDe()+"\n"+vet), 0o644); err != nil {
				t.Fatal(err)
			}
			b, _ := json.Marshal(kq)
			fmt.Fprintln(jl, string(b))
			all = append(all, kq)
		}
	}
	tong := tongHop(all, dongCo, len(docs), giu)
	if err := os.WriteFile(filepath.Join(ra, "tong-hop.md"), []byte(tong), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("động cơ: %s · %d tài liệu · %d câu · kết quả ở %s\n%s", dongCo, len(docs), len(all), ra, tong)
}

func must(s string, err error) string {
	if err != nil {
		return ".." // không tính được đường dẫn tương đối: coi như ngoài repo là sai — Rel chỉ lỗi khi một bên tương đối, đã Abs cả hai
	}
	return s
}

// cauHinhCa là tệp cấu hình một lần chạy case study.
type cauHinhCa struct {
	Pack       string           `yaml:"pack"`
	Ngay       string           `yaml:"ngay"`
	Vai        []string         `yaml:"vai"`
	Nguon      []rag.SourceSpec `yaml:"nguon"`
	ChiTrenMay bool             `yaml:"chi_tren_may"` // dữ liệu không được rời máy: bỏ qua EAIS_RAG_DONG_CO
	Ra         string           `yaml:"ra"`
	Bo         []boCauSpec      `yaml:"bo"`
}

// boCauSpec ánh xạ MỘT tệp bộ câu (định dạng bất kỳ) vào ca chung: danh_sach là đường dẫn chấm tới mảng câu ("" = gốc là mảng),
// truong là tên trường nguồn cho từng vai trò; loc giữ câu có trường = giá trị.
type boCauSpec struct {
	Ten      string            `yaml:"ten"`
	Tep      string            `yaml:"tep"`
	DanhSach string            `yaml:"danh_sach"`
	Truong   truongCa          `yaml:"truong"`
	Loc      map[string]string `yaml:"loc"`
	// TuChoiKhiRong: tên trường mà khi rỗng / null thì câu là câu KHÔNG có đáp án (phải từ chối).
	TuChoiKhiRong string `yaml:"tu_choi_khi_rong"`
	// TuChoiKhi: câu KHÔNG có đáp án khi mọi trường = giá trị (vd loai: khong_co_dap_an).
	TuChoiKhi map[string]string `yaml:"tu_choi_khi"`
}

type truongCa struct {
	Ma        string `yaml:"ma"`
	Hoi       string `yaml:"hoi"`
	Nhom      string `yaml:"nhom"`
	Chua      string `yaml:"chua"`       // chuỗi hoặc danh sách: câu trả lời phải chứa MỌI phần
	BatKy     string `yaml:"bat_ky"`     // danh sách: câu trả lời chứa ÍT NHẤT một
	KhongChua string `yaml:"khong_chua"` // danh sách: câu trả lời không được chứa
	TuChoi    string `yaml:"tu_choi"`    // trường đúng / sai: đúng = câu không có đáp án
	Vai       string `yaml:"vai"`
}

func docCauHinhCa(path string) (cauHinhCa, error) {
	var c cauHinhCa
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	dec := yaml.NewDecoder(strings.NewReader(string(b)))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil {
		return c, fmt.Errorf("%s: %w", path, err)
	}
	if c.Pack == "" || c.Ngay == "" || c.Ra == "" || len(c.Nguon) == 0 || len(c.Bo) == 0 || len(c.Vai) == 0 {
		return c, fmt.Errorf("%s: cần pack, ngay, vai, nguon, ra, bo", path)
	}
	return c, nil
}

// caChung là một câu hỏi sau khi ánh xạ.
type caChung struct {
	Ma, Hoi, Nhom          string
	Chua, BatKy, KhongChua []string
	TuChoi                 bool
	Vai                    []string
}

// docBoCau đọc tệp bộ câu (.json / .yaml / .yml) và ánh xạ từng mục theo cấu hình.
func docBoCau(base string, bo boCauSpec) ([]caChung, error) {
	b, err := os.ReadFile(resolvePath(base, bo.Tep))
	if err != nil {
		return nil, err
	}
	var root any
	if err := yaml.Unmarshal(b, &root); err != nil { // YAML là tập mẹ của JSON: một bộ đọc cho cả hai
		return nil, err
	}
	node := root
	for _, k := range strings.Split(bo.DanhSach, ".") {
		if k == "" {
			continue
		}
		m, ok := node.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("danh_sach %q: %q không phải đối tượng", bo.DanhSach, k)
		}
		node = m[k]
	}
	items, ok := node.([]any)
	if !ok || len(items) == 0 {
		return nil, fmt.Errorf("danh_sach %q không trỏ tới mảng câu", bo.DanhSach)
	}
	f := bo.Truong
	if f.Hoi == "" {
		return nil, fmt.Errorf("truong cần ít nhất hoi")
	}
	// Tên trường gõ sai thì mọi câu thành "không chấm" hoặc "phải từ chối" mà test vẫn xanh: mỗi tên khai phải có ở ít nhất một mục.
	ten := []string{f.Ma, f.Hoi, f.Nhom, f.Chua, f.BatKy, f.KhongChua, f.TuChoi, f.Vai, bo.TuChoiKhiRong}
	ten = append(append(ten, slices.Collect(maps.Keys(bo.Loc))...), slices.Collect(maps.Keys(bo.TuChoiKhi))...)
	for _, k := range ten {
		if k != "" && !slices.ContainsFunc(items, func(it any) bool { m, ok := it.(map[string]any); _, has := m[k]; return ok && has }) {
			return nil, fmt.Errorf("trường %q không có ở mục nào của %s", k, bo.Tep)
		}
	}
	var out []caChung
	for i, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("mục #%d không phải đối tượng", i)
		}
		if !khopLoc(m, bo.Loc) {
			continue
		}
		c := caChung{Ma: chuoi(m[f.Ma]), Hoi: chuoi(m[f.Hoi]), Nhom: chuoi(m[f.Nhom]), Chua: danhSach(m[f.Chua]),
			BatKy: danhSach(m[f.BatKy]), KhongChua: danhSach(m[f.KhongChua]), TuChoi: dung(m[f.TuChoi]), Vai: danhSach(m[f.Vai])}
		if c.Ma == "" {
			c.Ma = fmt.Sprintf("%s-%03d", bo.Ten, i+1)
		}
		if bo.TuChoiKhiRong != "" && len(danhSach(m[bo.TuChoiKhiRong])) == 0 || len(bo.TuChoiKhi) > 0 && khopLoc(m, bo.TuChoiKhi) {
			c.TuChoi = true
		}
		if strings.TrimSpace(c.Hoi) == "" {
			return nil, fmt.Errorf("mục %s: câu hỏi rỗng", c.Ma)
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("lọc %v không giữ câu nào trong %d mục", bo.Loc, len(items))
	}
	return out, nil
}

func khopLoc(m map[string]any, loc map[string]string) bool {
	for k, v := range loc {
		if chuoi(m[k]) != v {
			return false
		}
	}
	return true
}

func chuoi(v any) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func danhSach(v any) []string {
	switch x := v.(type) {
	case nil:
		return nil
	case []any:
		var out []string
		for _, e := range x {
			if s := chuoi(e); s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		if s := chuoi(x); s != "" {
			return []string{s}
		}
		return nil
	}
}

// dung đọc trường đúng / sai: bool, hoặc chuỗi "true" / "1" / "yes"… (strconv.ParseBool).
func dung(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		b, _ := strconv.ParseBool(strings.TrimSpace(x))
		return b
	}
	return false
}

// truyGhi bọc kho tri thức để ghi lại bằng chứng của lượt (đánh giá truy hồi tách khỏi đánh giá câu trả lời).
type truyGhi struct {
	inner  pipeline.KnowledgeRetriever
	ev     []contracts.Evidence
	calls  int
	notice *contracts.KnowledgeNotice
}

func (r *truyGhi) reset() { r.ev, r.calls, r.notice = nil, 0, nil }

func (r *truyGhi) Retrieve(ctx context.Context, pr contracts.Principal, q string, k int) ([]contracts.Evidence, error) {
	ev, err := r.inner.Retrieve(ctx, pr, q, k)
	r.calls++
	r.ev = append(r.ev, ev...)
	var n *contracts.KnowledgeNotice
	if errors.As(err, &n) {
		r.notice = n
	}
	return ev, err
}

// lanTruy là những gì truy hồi để lại trong MỘT lượt: tách "không gọi truy hồi" / "gọi mà rỗng" / "thông báo phiên bản".
type lanTruy struct {
	ev     []contracts.Evidence
	calls  int
	notice *contracts.KnowledgeNotice
}

func (r *truyGhi) lan() lanTruy { return lanTruy{r.ev, r.calls, r.notice} }

// ketQuaCa là kết quả của một câu: đủ để đọc lại mà không chạy lại.
type ketQuaCa struct {
	Bo, Ma, Nhom, Hoi string
	KyVong            string
	Outcome           contracts.Outcome
	TraLoi            string
	TrichDan          []string
	TruyHoi           []bangChungNgan
	TruyHoiTrung      bool
	Dat               bool
	KhongCham         bool   // ca không khai kỳ vọng nào: ghi kết cục, không tính đạt / trượt
	TangSai           string // rỗng khi đạt
	LyDo              string
	Loi               string `json:",omitempty"`
	Ms                int64
	Buoc              []traces.Step
}

type bangChungNgan struct {
	ID   string
	Diem float64
	Doan string
}

// khop: so khớp theo RANH GIỚI TỪ, không phân biệt hoa thường. Kỳ vọng gõ không dấu thì so bỏ dấu ("ha noi" khớp "Hà Nội"); kỳ vọng
// có dấu thì so giữ dấu ("phạt" không khớp "phát"). Kỳ vọng toàn chữ số so cả với chữ đã bỏ dấu phân cách NHÓM BA ("1250000" khớp
// "1.250.000"; "12" không khớp "1 2"). shortcut: không hiểu phủ định ("không có X" vẫn chứa "có X") — bộ ca cần khuong_chua cho ý đó.
func khop(text string, want string) bool {
	t, w := strings.ToLower(normalization.NFC(text)), strings.ToLower(normalization.NFC(strings.TrimSpace(want)))
	if w == normalization.Fold(w) {
		t = normalization.Fold(t)
	}
	if w == "" {
		return false
	}
	if chuaTu(t, w) {
		return true
	}
	return strings.Trim(w, "0123456789") == "" && chuaTu(boNhomSo(t), w)
}

// chuaTu: w xuất hiện trong t mà hai bên không dính chữ / số ("24 giờ" không khớp "124 giờ").
func chuaTu(t, w string) bool {
	for i := 0; ; {
		j := strings.Index(t[i:], w)
		if j < 0 {
			return false
		}
		a, b := i+j, i+j+len(w)
		before, _ := utf8.DecodeLastRuneInString(t[:a])
		after, _ := utf8.DecodeRuneInString(t[b:])
		if (a == 0 || !chuSo(before)) && (b == len(t) || !chuSo(after)) {
			return true
		}
		i = a + 1
	}
}

func chuSo(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// nhomSo: dấu chấm / phẩy / khoảng trắng giữa một chữ số và ĐÚNG ba chữ số (nhóm nghìn).
var nhomSo = regexp.MustCompile(`(\d)[.,\s](\d{3})(\D|$)`)

func boNhomSo(t string) string {
	for {
		u := nhomSo.ReplaceAllString(t, "$1$2$3")
		if u == t {
			return t
		}
		t = u
	}
}

// kiemCa là bảng kiểm (strategy): tên → phép kiểm trên một đoạn chữ. Ca không khai trường nào thì không kiểm trường đó.
var kiemCa = map[string]func(text string, want []string) bool{
	"chua": func(t string, w []string) bool {
		return !slices.ContainsFunc(w, func(x string) bool { return !khop(t, x) })
	},
	"bat_ky": func(t string, w []string) bool {
		return len(w) == 0 || slices.ContainsFunc(w, func(x string) bool { return khop(t, x) })
	},
	"khong_chua": func(t string, w []string) bool {
		return !slices.ContainsFunc(w, func(x string) bool { return khop(t, x) })
	},
}

// tuChoiDung: kết cục được tính là "không trả lời" ĐÚNG cho câu không có đáp án. Bị S9 chặn (model đã bịa), từ chối vì quyền / chuyển
// người / xung đột phiên bản đều KHÔNG phải — chúng là đúng lý do khác, không chứng minh hệ biết im.
var tuChoiDung = map[contracts.Outcome]bool{
	contracts.OutcomeInsufficientEvidence: true, contracts.OutcomeOutOfScope: true, contracts.OutcomeClarificationRequired: true,
}

// danhGia chấm một câu và chỉ ra TẦNG SAI ĐẦU TIÊN (CLAUDE.md P10: tìm tầng hỏng trước khi đổi model).
func danhGia(bo string, c caChung, out pipeline.Outcome, herr error, tr lanTruy) ketQuaCa {
	r, ev := out.Response, tr.ev
	kq := ketQuaCa{Bo: bo, Ma: c.Ma, Nhom: c.Nhom, Hoi: c.Hoi, Outcome: r.Outcome, TraLoi: r.Answer}
	switch {
	case c.TuChoi:
		kq.KyVong = "không có đáp án — phải từ chối"
	default:
		kq.KyVong = fmt.Sprintf("chứa %v · bất kỳ %v · không chứa %v", c.Chua, c.BatKy, c.KhongChua)
	}
	for _, ci := range r.Citations {
		kq.TrichDan = append(kq.TrichDan, ci.EvidenceID)
	}
	var doan strings.Builder
	for _, e := range ev {
		kq.TruyHoi = append(kq.TruyHoi, bangChungNgan{ID: e.ID, Diem: e.Confidence, Doan: catNgan(e.Text, 240)})
		doan.WriteString(e.Text + "\n")
	}
	kq.TruyHoiTrung = (len(c.Chua) > 0 || len(c.BatKy) > 0) && kiemCa["chua"](doan.String(), c.Chua) && kiemCa["bat_ky"](doan.String(), c.BatKy)
	if herr != nil {
		kq.Loi, kq.TangSai, kq.LyDo = herr.Error(), "lỗi hệ thống", contracts.FaultOf(herr)
		return kq
	}
	if !c.TuChoi && len(c.Chua) == 0 && len(c.BatKy) == 0 && len(c.KhongChua) == 0 {
		kq.KhongCham, kq.KyVong = true, "không khai kỳ vọng — KHÔNG CHẤM"
		return kq
	}
	route := out.Plan.Route
	tangKhac := func() (string, string) { // tầng của kết cục KHÔNG do truy hồi / model tri thức quyết
		switch o := r.Outcome; {
		case o == contracts.OutcomeOutOfScope || o == contracts.OutcomeClarificationRequired:
			return "S4 ý định / định tuyến", string(o) + ": " + out.Plan.Reason
		case o == contracts.OutcomePermissionDenied:
			return "S5 quyền", out.Plan.Reason
		case o == contracts.OutcomeHumanReviewRequired:
			return "S5 / S9 chuyển người", out.Plan.Reason
		case route != contracts.RouteRAG && route != contracts.RouteHybrid:
			return "S5 route khác tri thức (" + string(route) + ")", string(o) + ": " + out.Plan.Reason
		case tr.notice != nil:
			return "S7 phiên bản (" + string(tr.notice.Kind) + ")", "tài liệu khớp nhất không có đúng một bản làm căn cứ: " + tr.notice.DocumentID
		}
		return "", ""
	}
	if c.TuChoi {
		if tuChoiDung[r.Outcome] && tr.notice == nil {
			kq.Dat = true
			return kq
		}
		switch tang, lyDo := tangKhac(); {
		case r.Outcome == contracts.OutcomeSystemFailure:
			kq.TangSai, kq.LyDo = "S8 trả lời câu không có đáp án (S9 đã chặn)", fmt.Sprint(r.Verdict.Violations)
		case r.Outcome == contracts.OutcomeResolved && len(ev) > 0:
			kq.TangSai, kq.LyDo = "S6/S8 trả lời câu không có đáp án", "truy hồi nhận đoạn và model diễn đạt thành câu trả lời"
		case r.Outcome == contracts.OutcomeResolved:
			kq.TangSai, kq.LyDo = "S8 trả lời câu không có đáp án", "trả lời qua route "+string(route)
		default:
			kq.TangSai, kq.LyDo = "từ chối vì lý do khác — "+tang, lyDo
		}
		return kq
	}
	tl := r.Answer
	kq.Dat = r.Outcome == contracts.OutcomeResolved && kiemCa["chua"](tl, c.Chua) && kiemCa["bat_ky"](tl, c.BatKy) && kiemCa["khong_chua"](tl, c.KhongChua)
	if kq.Dat {
		return kq
	}
	if tang, lyDo := tangKhac(); tang != "" {
		kq.TangSai, kq.LyDo = tang, lyDo
		return kq
	}
	switch o := r.Outcome; {
	case o == contracts.OutcomeSystemFailure:
		kq.TangSai, kq.LyDo = "S9 kiểm đầu ra chặn", fmt.Sprint(r.Verdict.Violations)
	case tr.calls == 0:
		kq.TangSai, kq.LyDo = "S6 không gọi truy hồi", string(o)+" trên route "+string(route)+" mà chưa hỏi kho (vd luật câu hỏi lượng)"
	case len(ev) == 0:
		kq.TangSai, kq.LyDo = "S6 truy hồi: im", "không đoạn nào qua ngưỡng / luật biết im"
	case (len(c.Chua) > 0 || len(c.BatKy) > 0) && !kq.TruyHoiTrung:
		kq.TangSai, kq.LyDo = "S6 truy hồi: trượt", "có đoạn nhưng không đoạn nào chứa đáp án"
	case o != contracts.OutcomeResolved:
		kq.TangSai, kq.LyDo = "S8 model từ chối", string(o)+" dù đoạn chứa đáp án đã được truy hồi"
	case !kiemCa["khong_chua"](tl, c.KhongChua):
		kq.TangSai, kq.LyDo = "S8 nói điều cấm", "câu trả lời chứa cụm không được có"
	default:
		kq.TangSai, kq.LyDo = "S8 diễn đạt thiếu ý", "đoạn đúng đã truy hồi nhưng câu trả lời thiếu phần kỳ vọng"
	}
	return kq
}

func (k ketQuaCa) tieuDe() string {
	dat := "ĐẠT"
	switch {
	case k.KhongCham:
		dat = "KHÔNG CHẤM"
	case !k.Dat:
		dat = "TRƯỢT — tầng sai đầu tiên: " + k.TangSai
	}
	var b strings.Builder
	fmt.Fprintf(&b, "CA %s/%s [%s] — %s\n  hỏi: %s\n  kỳ vọng: %s\n  kết cục: %s · %d ms\n  trả lời: %s\n  trích dẫn: %v\n",
		k.Bo, k.Ma, k.Nhom, dat, k.Hoi, k.KyVong, k.Outcome, k.Ms, catNgan(k.TraLoi, 600), k.TrichDan)
	if k.LyDo != "" {
		fmt.Fprintf(&b, "  lý do: %s\n", k.LyDo)
	}
	fmt.Fprintf(&b, "  truy hồi (%d đoạn, có đoạn chứa đáp án: %v):\n", len(k.TruyHoi), k.TruyHoiTrung)
	for _, e := range k.TruyHoi {
		fmt.Fprintf(&b, "    %.3f  %s  %s\n", e.Diem, e.ID, e.Doan)
	}
	if k.Loi != "" {
		fmt.Fprintf(&b, "  LỖI: %s\n", k.Loi)
	}
	return b.String()
}

// tongHop: theo bộ, theo nhóm câu, theo tầng sai đầu tiên; kèm danh sách câu trượt để mở vết.
func tongHop(all []ketQuaCa, dongCo string, nDocs int, giu []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Tổng hợp case study\n\nĐộng cơ: %s · kho %d tài liệu · %d câu.\n\n", dongCo, nDocs, len(all))
	fmt.Fprintf(&b, "Tài liệu bị cổng chất lượng GIỮ LẠI (không phục vụ): %d\n", len(giu))
	for _, g := range giu {
		fmt.Fprintf(&b, "- %s\n", g)
	}
	b.WriteString("\n")
	type dem struct{ dat, tong int }
	var cham []ketQuaCa
	khongCham := 0
	for _, k := range all {
		if k.KhongCham {
			khongCham++
		} else {
			cham = append(cham, k)
		}
	}
	fmt.Fprintf(&b, "Không chấm (ca không khai kỳ vọng): %d câu — không tính vào bảng dưới.\n\n", khongCham)
	theo := func(key func(ketQuaCa) string) ([]string, map[string]*dem) {
		m := map[string]*dem{}
		var keys []string
		for _, k := range cham {
			x := key(k)
			if m[x] == nil {
				m[x] = &dem{}
				keys = append(keys, x)
			}
			m[x].tong++
			if k.Dat {
				m[x].dat++
			}
		}
		return keys, m
	}
	keys, m := theo(func(k ketQuaCa) string { return k.Bo })
	b.WriteString("| Bộ | Đạt | Tổng |\n|---|---|---|\n")
	for _, x := range keys {
		fmt.Fprintf(&b, "| %s | %d | %d |\n", x, m[x].dat, m[x].tong)
	}
	keys, m = theo(func(k ketQuaCa) string { return k.Bo + " · " + orDash(k.Nhom) })
	b.WriteString("\n| Bộ · nhóm câu | Đạt | Tổng |\n|---|---|---|\n")
	for _, x := range keys {
		fmt.Fprintf(&b, "| %s | %d | %d |\n", x, m[x].dat, m[x].tong)
	}
	tang := map[string]int{}
	var tangs []string
	for _, k := range cham {
		if !k.Dat {
			if tang[k.TangSai] == 0 {
				tangs = append(tangs, k.TangSai)
			}
			tang[k.TangSai]++
		}
	}
	slices.SortFunc(tangs, func(a, c string) int { return tang[c] - tang[a] })
	b.WriteString("\n| Tầng sai đầu tiên | Số câu |\n|---|---|\n")
	for _, x := range tangs {
		fmt.Fprintf(&b, "| %s | %d |\n", x, tang[x])
	}
	b.WriteString("\n## Câu trượt (vết ở vet/<bộ>/<mã>.txt)\n\n")
	for _, k := range cham {
		if !k.Dat {
			fmt.Fprintf(&b, "- %s/%s [%s] %s — %s\n", k.Bo, k.Ma, orDash(k.Nhom), k.TangSai, catNgan(k.Hoi, 120))
		}
	}
	return b.String()
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func catNgan(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

func tenTep(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ' ' || r == ':' {
			return '_'
		}
		return r
	}, s)
}

func resolvePath(base, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(base, p)
}
