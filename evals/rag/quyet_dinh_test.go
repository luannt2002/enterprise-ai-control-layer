package ragcases

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/intent"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
)

// Thí nghiệm model quyết định có kiểu (models.ProtocolDecide) ở hai chỗ, docs/ROADMAP-RAG.md §10.3:
//   - S4: B2 = luật + model phân loại ở câu luật không ra kế hoạch; B3 = model phân loại mọi câu; B4 = B1 trước, model chỉ khi chỉ mục im.
//     Code quyết: chỉ đổi sang ý định tri thức khi nhãn thắng là quy trình / cả hai VÀ xác suất ≥ ngưỡng; chốt quyền và câu hỏi lượng
//     vẫn ở pipeline.
//   - Chấm đoạn trước khi sinh câu trả lời: A = đường gốc · B = gốc rồi chấm "đoạn có trả lời được không", bỏ đoạn dưới ngưỡng (BỔ SUNG
//     cho xếp hạng) · C = lấy nhóm lớn hơn từ đường gốc, xếp lại bằng xác suất chấm (THAY xếp hạng). Chấm chỉ chạy trên đoạn ĐÃ qua
//     lọc quyền / phiên bản: đoạn người hỏi không được đọc không bao giờ ra nhà cung cấp. Chấm lỗi ⇒ trả đường gốc và đếm lỗi.

type quyetDinh interface {
	Choice(ctx context.Context, state any, instructions string, labels map[string]string) (map[string]float64, string, error)
	Propositions(ctx context.Context, state any, props map[string]string) (map[string]float64, string, error)
}

// Nhãn và hướng dẫn gửi cho model phân loại S4 (tên nhãn và mô tả ĐỀU được model đọc). Chỉnh trên bộ phát triển, không trên bộ giữ lại.
const s4HuongDan = "A construction-site employee asked this question in Vietnamese, possibly typed without diacritics or with site " +
	"abbreviations. Classify what they want."

var s4Nhan = map[string]string{
	"operational_fact": "Asks for a recorded business number: fuel issued, consumed, reported, invoiced, or the difference, for a machine or project over a period",
	"procedure":        "Asks how something must be done or what the rule is: procedure, regulation, responsibility, deadline, condition, required documents, technical specification. Answered by internal documents",
	"combined":         "Asks both for a recorded business number and for the procedure related to it",
	"other":            "Greeting, small talk, or anything unrelated to the company's operations and documents",
}

// moHinhS4 là dự phòng S4 bằng model phân loại.
type moHinhS4 struct {
	qd     quyetDinh
	moi    bool                     // B3: hỏi mọi câu (pipeline vẫn chặn câu hỏi lượng / thiếu quyền)
	truoc  *pipeline.KnowledgeProbe // B4: dò chỉ mục trước
	nguong float64
	dem    *demGoi
}

func (m moHinhS4) Intent() string { return s4YDinhTriThuc }

func (m moHinhS4) Fallback(ctx context.Context, pr contracts.Principal, in contracts.NormalizedInput, cls intent.Result) (bool, float64, string, error) {
	ket := cls.Intent.Name == "" || len(cls.Intent.Missing) > 0
	if !m.moi && (!ket || len(cls.Ambiguous) > 0) {
		return false, 0, "", nil
	}
	if m.truoc != nil {
		if ok, c, why, err := m.truoc.Fallback(ctx, pr, in, cls); ok || err != nil {
			return ok, c, why, err
		}
	}
	m.dem.mu.Lock()
	m.dem.goi++
	m.dem.mu.Unlock()
	p, model, err := m.qd.Choice(ctx, in.Raw, s4HuongDan, s4Nhan)
	if err != nil {
		m.dem.mu.Lock()
		m.dem.loi++
		m.dem.mu.Unlock()
		return false, 0, "", err
	}
	best := slices.MaxFunc(slices.Collect(maps.Keys(p)), func(a, b string) int { return cmp.Or(cmp.Compare(p[a], p[b]), cmp.Compare(b, a)) })
	why := fmt.Sprintf("model phân loại %s: %s %.2f", model, best, p[best])
	m.dem.mu.Lock()
	m.dem.cuoi = why
	m.dem.mu.Unlock()
	return (best == "procedure" || best == "combined") && p[best] >= m.nguong, p[best], why, nil
}

// chamDoan bọc đường truy hồi gốc bằng bộ chấm "đoạn có trả lời được câu hỏi không".
type chamDoan struct {
	goc    s7Truy
	qd     quyetDinh
	kieu   string // A | B | C
	nguong float64
	pool   int // C: số đoạn lấy từ đường gốc để xếp lại
	dem    *demGoi
}

// Mệnh đề chấm đoạn (Noul). Chỉ "tra_loi_duoc" quyết; "lien_quan" ghi để đọc (liên quan ≠ đủ căn cứ).
var chamMenhDe = map[string]string{
	"tra_loi_duoc": "Does the passage directly state information that answers the query (not merely discuss the same topic)?",
	"lien_quan":    "Is the passage about the subject of the query?",
}

func (c *chamDoan) Retrieve(ctx context.Context, pr contracts.Principal, q string, k int) ([]contracts.Evidence, error) {
	n := k
	if c.kieu == "C" {
		n = max(c.pool, k)
	}
	ev, err := c.goc(ctx, pr, q, n)
	if err != nil || c.kieu == "A" || len(ev) == 0 {
		return ev, err
	}
	p := make([]float64, len(ev))
	var wg sync.WaitGroup
	var loi error
	var mu sync.Mutex
	sem := make(chan struct{}, 4)
	for i, e := range ev {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			r, _, err := c.qd.Propositions(ctx, map[string]any{"query": q, "passage": map[string]string{"title": e.Source.Name, "text": e.Text}}, chamMenhDe)
			mu.Lock()
			defer mu.Unlock()
			c.dem.goi++
			if err != nil {
				loi = err
				return
			}
			p[i] = r["tra_loi_duoc"]
		}()
	}
	wg.Wait()
	if loi != nil {
		c.dem.mu.Lock()
		c.dem.loi++
		c.dem.mu.Unlock()
		return ev[:min(k, len(ev))], nil // chấm lỗi: về đường gốc
	}
	idx := make([]int, len(ev))
	for i := range idx {
		idx[i] = i
	}
	if c.kieu == "C" {
		slices.SortStableFunc(idx, func(a, b int) int { return cmp.Compare(p[b], p[a]) })
	}
	var out []contracts.Evidence
	for _, i := range idx {
		if p[i] >= c.nguong && len(out) < k {
			out = append(out, ev[i])
		}
	}
	return out, nil
}

// nhoQuyetDinh nhớ kết quả theo đầu vào trong MỘT lần chạy: quét nhiều ngưỡng không gọi lại. Model không tất định — số đo dùng đúng
// một lần gọi cho mỗi đầu vào; độ lệch giữa các lần gọi đo riêng.
type nhoQuyetDinh struct {
	qd quyetDinh
	mu sync.Mutex
	m  map[string]map[string]float64
}

func (n *nhoQuyetDinh) lay(key string, goi func() (map[string]float64, string, error)) (map[string]float64, string, error) {
	n.mu.Lock()
	if p, ok := n.m[key]; ok {
		n.mu.Unlock()
		return p, "nhớ", nil
	}
	n.mu.Unlock()
	p, model, err := goi()
	if err == nil {
		n.mu.Lock()
		n.m[key] = p
		n.mu.Unlock()
	}
	return p, model, err
}

func (n *nhoQuyetDinh) Choice(ctx context.Context, state any, ins string, labels map[string]string) (map[string]float64, string, error) {
	return n.lay(fmt.Sprint("c", state, ins, labels), func() (map[string]float64, string, error) { return n.qd.Choice(ctx, state, ins, labels) })
}

func (n *nhoQuyetDinh) Propositions(ctx context.Context, state any, props map[string]string) (map[string]float64, string, error) {
	return n.lay(fmt.Sprint("p", state, props), func() (map[string]float64, string, error) { return n.qd.Propositions(ctx, state, props) })
}

// nguongDanhSach đọc một danh sách ngưỡng "0.3,0.5,0.7" (trống = mac).
func nguongDanhSach(t *testing.T, ten string, mac float64) []float64 {
	t.Helper()
	v := os.Getenv(ten)
	if v == "" {
		return []float64{mac}
	}
	var out []float64
	for _, x := range strings.Split(v, ",") {
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if err != nil || f < 0 || f > 1 {
			t.Fatalf("%s %q: mỗi ngưỡng phải là số trong [0, 1]", ten, v)
		}
		out = append(out, f)
	}
	return out
}

// quyetDinhTuMoiTruong dựng client từ EAIS_DECIDE_*; nil (phép đo bỏ qua) khi chưa bật hoặc chưa có khoá — bật mà thiếu khoá thì nói rõ.
func quyetDinhTuMoiTruong(t *testing.T) quyetDinh {
	t.Helper()
	if os.Getenv("EAIS_DECIDE_PROVIDER") == "" {
		return nil
	}
	if os.Getenv("EAIS_DECIDE_API_KEY") == "" {
		t.Log("EAIS_DECIDE_PROVIDER đã đặt nhưng EAIS_DECIDE_API_KEY trống — bỏ qua phép đo model quyết định")
		return nil
	}
	d, err := models.NewDecider(models.ServiceFromEnv("DECIDE"))
	if err != nil {
		t.Fatalf("cấu hình model quyết định: %v", err)
	}
	quyetDinhChung.Do(func() { quyetDinhNho = &nhoQuyetDinh{qd: d, m: map[string]map[string]float64{}} })
	return quyetDinhNho
}

var (
	quyetDinhChung sync.Once
	quyetDinhNho   *nhoQuyetDinh
)

// TestChamDoan đo A / B / C trên bộ S7 (EAIS_S7_BO). Đường gốc chọn bằng EAIS_CHAM_DUONG: "tu_khoa" (mặc định, đường của ứng dụng) hoặc
// "lai" (từ khoá + vector + xếp lại, cần EAIS_EMBED_* / EAIS_RERANK_*): A = lai có xếp lại · B = A rồi chấm, bỏ đoạn dưới ngưỡng
// (BỔ SUNG xếp lại) · C = lai KHÔNG xếp lại, nhóm EAIS_CHAM_POOL đoạn đầu theo RRF, chấm và xếp bằng xác suất (THAY xếp lại). Chỉ chạy
// khi cấu hình EAIS_DECIDE_* — câu hỏi và đoạn ĐÃ qua lọc quyền được GỬI RA nhà cung cấp: chỉ bộ giả lập / công khai. Ngưỡng:
// EAIS_CHAM_NGUONG, có thể là danh sách "0.3,0.5,0.7" (mặc định 0.5).
func TestChamDoan(t *testing.T) {
	qd := quyetDinhTuMoiTruong(t)
	if qd == nil {
		t.Skip("đặt EAIS_DECIDE_* (khuôn ở .env.example) để đo bộ chấm đoạn với model thật (gửi dữ liệu ra ngoài)")
	}
	bo, docs, rt := napS7Do(t)
	if os.Getenv("EAIS_S4_BO") != "" { // bộ của thí nghiệm S4: dùng phần có đáp án / không có đáp án
		b4, d4, _ := napBoS4(t)
		bo, docs = boTruyHoiTuS4(b4, d4), d4
	}
	pool, _ := strconv.Atoi(cmp.Or(os.Getenv("EAIS_CHAM_POOL"), "10"))
	duong := cmp.Or(os.Getenv("EAIS_CHAM_DUONG"), "tu_khoa")
	goc := map[string]s7Truy{}
	switch duong {
	case "tu_khoa":
		ix := rag.NewMemoryIndex("abc-infra", docs)
		goc["A"], goc["B"], goc["C"] = ix.Retrieve, ix.Retrieve, ix.Retrieve
	case "lai":
		sem := semanticTuMoiTruong(t)
		ix := rag.NewMemoryIndex("abc-infra", docs)
		texts := ix.EmbedTexts()
		vec, err := sem.Embedder.Embed(context.Background(), texts, false)
		if err != nil {
			t.Fatalf("nhúng kho: %v", err)
		}
		sem.Vectors = map[string][]float32{}
		for i, x := range texts {
			sem.Vectors[rag.EmbedKey(x)] = vec[i]
		}
		ix.Semantic = sem
		khong := *sem // C: cùng vector, không xếp lại, không ngưỡng cosine (ngưỡng của C là ngưỡng chấm)
		khong.Reranker, khong.MinScore = nil, 0
		ixC := rag.NewMemoryIndex("abc-infra", docs)
		ixC.Semantic = &khong
		goc["A"], goc["B"], goc["C"] = ix.Retrieve, ix.Retrieve, ixC.Retrieve
		t.Logf("đường lai: %d chữ đã nhúng, ngưỡng xếp lại %.3f, nhóm xếp lại %d", len(texts), sem.MinScore, sem.Pool)
	default:
		t.Fatalf("EAIS_CHAM_DUONG %q: tu_khoa | lai", duong)
	}
	for lan, nguong := range nguongDanhSach(t, "EAIS_CHAM_NGUONG", 0.5) {
		for _, kieu := range []string{"A", "B", "C"} {
			if kieu == "A" && lan > 0 {
				continue // A không phụ thuộc ngưỡng: đo một lần
			}
			dem := &demGoi{}
			c := &chamDoan{goc: goc[kieu], qd: qd, kieu: kieu, nguong: nguong, pool: pool, dem: dem}
			t0 := time.Now()
			kq := doS7(t, bo, docs, "abc-infra", rt, c.Retrieve)
			t.Logf("[%s] %s (ngưỡng %.2f, nhóm C %d): %s · lời gọi chấm %d (lỗi %d) · %s", duong, kieu, nguong, pool, kq, dem.goi, dem.loi, time.Since(t0).Round(time.Millisecond))
			for _, x := range [][]string{kq.ngoaiQuyen, kq.khongCanCu, kq.thieuNguon} {
				if len(x) > 0 {
					t.Errorf("%s: vế tuyệt đối hỏng: %v", kieu, x)
				}
			}
		}
	}
}

// boTruyHoiTuS4 lấy phần đo truy hồi của bộ S4: tra_loi và ca_hai thành câu có đáp án, khong_co giữ, so_lieu bỏ; đáp án ghi ban 1 cho tài
// liệu không phiên bản đổi về 0 (quy ước khai trước lần chạy, dungBan).
func boTruyHoiTuS4(b s4Bo, docs []rag.Document) s7Bo {
	out := s7Bo{NgayMacDinh: b.NgayMacDinh, K: b.K}
	for _, c := range b.Cau {
		switch c.Loai {
		case "so_lieu":
			continue
		case "ca_hai":
			c.Loai = "tra_loi"
		}
		x := c.s7Cau
		x.DapAn = slices.Clone(x.DapAn)
		for i, d := range x.DapAn {
			if d.Ban == 1 && slices.ContainsFunc(docs, func(r rag.Document) bool { return r.ID == d.TaiLieu && r.Version == 0 }) {
				x.DapAn[i].Ban = 0
			}
		}
		out.Cau = append(out.Cau, x)
	}
	return out
}

// Logic của B2 / B3 / B4 và A / B / C với model GIẢ (trên máy, không gọi ra ngoài).
func TestQuyetDinhGia(t *testing.T) {
	ctx := context.Background()
	gia := &quyetDinhGia{}
	dem := &demGoi{}
	ket := intent.Result{}
	du := intent.Result{Intent: contracts.Intent{Name: "fuel_issued"}}
	in := contracts.NormalizedInput{Raw: "máy chạy mấy tiếng thì thay nhớt?"}
	for _, c := range []struct {
		ten  string
		m    moHinhS4
		cls  intent.Result
		p    float64
		muon bool
		goi  int
	}{
		{"B2 câu kẹt, quy trình 0.8", moHinhS4{qd: gia, nguong: 0.5, dem: dem}, ket, 0.8, true, 1},
		{"B2 câu kẹt, dưới ngưỡng", moHinhS4{qd: gia, nguong: 0.9, dem: dem}, ket, 0.8, false, 1},
		{"B2 câu đã có ý định: không hỏi", moHinhS4{qd: gia, nguong: 0.5, dem: dem}, du, 0.8, false, 0},
		{"B3 câu đã có ý định: vẫn hỏi", moHinhS4{qd: gia, moi: true, nguong: 0.5, dem: dem}, du, 0.8, true, 1},
	} {
		gia.pQuyTrinh, dem.goi = c.p, 0
		ok, _, _, err := c.m.Fallback(ctx, contracts.Principal{}, in, c.cls)
		if err != nil || ok != c.muon || dem.goi != c.goi {
			t.Errorf("%s: được %v (gọi %d, lỗi %v), muốn %v (gọi %d)", c.ten, ok, dem.goi, err, c.muon, c.goi)
		}
	}
	// B4: chỉ mục có đoạn thì không gọi model.
	ix := rag.NewMemoryIndex("t", []rag.Document{{ID: "BD", Tenant: "t", Title: "Bảo dưỡng", ACL: []string{"ks"}, Body: "1. Thay nhớt máy sau 250 giờ chạy."}})
	pr := contracts.Principal{ID: "u", TenantID: "t", Kind: contracts.PrincipalUser, Roles: []string{"ks"}}
	dem.goi = 0
	b4 := moHinhS4{qd: gia, truoc: &pipeline.KnowledgeProbe{Knowledge: ix, Name: s4YDinhTriThuc}, nguong: 0.5, dem: dem}
	if ok, _, _, _ := b4.Fallback(ctx, pr, contracts.NormalizedInput{Raw: "thay nhớt sau bao nhiêu giờ chạy?"}, ket); !ok || dem.goi != 0 {
		t.Errorf("B4: chỉ mục có đoạn ⇒ dùng, 0 lời gọi model; được %v, %d", ok, dem.goi)
	}

	// Chấm đoạn: giả chấm "trả lời được" theo đoạn có chữ "250 giờ".
	ev := func(ids ...string) []contracts.Evidence {
		var out []contracts.Evidence
		for _, id := range ids {
			out = append(out, contracts.Evidence{ID: id, Text: id})
		}
		return out
	}
	goc := func(_ context.Context, _ contracts.Principal, _ string, k int) ([]contracts.Evidence, error) {
		return ev("kiểm tra nhớt hằng ngày", "lọc nhớt 500 giờ", "thay nhớt 250 giờ", "đoạn 4")[:k], nil
	}
	for _, c := range []struct {
		kieu string
		muon []string
	}{{"A", []string{"kiểm tra nhớt hằng ngày", "lọc nhớt 500 giờ", "thay nhớt 250 giờ"}}, {"B", []string{"thay nhớt 250 giờ"}}, {"C", []string{"thay nhớt 250 giờ"}}} {
		ch := &chamDoan{goc: goc, qd: gia, kieu: c.kieu, nguong: 0.5, pool: 4, dem: &demGoi{}}
		got, err := ch.Retrieve(ctx, pr, "bao nhiêu giờ thay nhớt?", 3)
		if ids := evIDs(got); err != nil || !slices.Equal(ids, c.muon) {
			t.Errorf("%s: được %v %v, muốn %v", c.kieu, ids, err, c.muon)
		}
	}
	// C xếp lại từ nhóm lớn hơn: đoạn đúng nằm ngoài top-3 của đường gốc vẫn vào.
	goc4 := func(_ context.Context, _ contracts.Principal, _ string, k int) ([]contracts.Evidence, error) {
		return ev("a", "b", "c", "thay nhớt 250 giờ")[:k], nil
	}
	if got, _ := (&chamDoan{goc: goc4, qd: gia, kieu: "C", nguong: 0.5, pool: 4, dem: &demGoi{}}).Retrieve(ctx, pr, "q", 3); !slices.Equal(evIDs(got), []string{"thay nhớt 250 giờ"}) {
		t.Errorf("C phải kéo đoạn hạng 4 lên: %v", evIDs(got))
	}
	// Chấm lỗi: về đường gốc, đếm lỗi.
	gia.loi = true
	d := &demGoi{}
	if got, err := (&chamDoan{goc: goc, qd: gia, kieu: "B", nguong: 0.5, dem: d}).Retrieve(ctx, pr, "q", 3); err != nil || len(got) != 3 || d.loi != 1 {
		t.Errorf("chấm lỗi: muốn 3 đoạn gốc + 1 lỗi, được %v %v %d", evIDs(got), err, d.loi)
	}
}

type quyetDinhGia struct {
	pQuyTrinh float64
	loi       bool
}

func (g *quyetDinhGia) Choice(context.Context, any, string, map[string]string) (map[string]float64, string, error) {
	return map[string]float64{"procedure": g.pQuyTrinh, "operational_fact": 1 - g.pQuyTrinh, "combined": 0, "other": 0}, "gia", nil
}

func (g *quyetDinhGia) Propositions(_ context.Context, state any, _ map[string]string) (map[string]float64, string, error) {
	if g.loi {
		return nil, "", contracts.Errorf(contracts.ThirdParty("gia", contracts.KindRateLimit), "429")
	}
	text := state.(map[string]any)["passage"].(map[string]string)["text"]
	p := 0.1
	if text == "thay nhớt 250 giờ" {
		p = 0.9
	}
	return map[string]float64{"tra_loi_duoc": p, "lien_quan": 0.8}, "gia", nil
}

func evIDs(ev []contracts.Evidence) []string {
	var out []string
	for _, e := range ev {
		out = append(out, e.ID)
	}
	return out
}
