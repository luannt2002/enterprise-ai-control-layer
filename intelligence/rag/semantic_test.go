package rag

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
)

// nhungGia là embedder giả tất định: mỗi chiều là một KHÁI NIỆM (nhóm từ đồng nghĩa); chữ có từ của khái niệm thì chiều đó = 1.
// Đủ để dựng ca "người hỏi dùng chữ khác tài liệu" mà đường từ khoá không bắt được.
type nhungGia struct {
	khaiNiem [][]string
	goi      int
	loi      error
}

func (e *nhungGia) Name() string { return "gia/khai-niem" }

func (e *nhungGia) Embed(_ context.Context, texts []string, _ bool) ([][]float32, error) {
	e.goi++
	if e.loi != nil {
		return nil, e.loi
	}
	out := make([][]float32, len(texts))
	for i, t := range texts {
		t = normalization.Fold(strings.ToLower(t))
		v := make([]float32, len(e.khaiNiem)+1)
		v[len(e.khaiNiem)] = 0.05 // chiều nền: vector không bao giờ bằng 0
		for d, ws := range e.khaiNiem {
			for _, w := range ws {
				if strings.Contains(t, normalization.Fold(w)) {
					v[d] = 1
				}
			}
		}
		out[i] = v
	}
	return out, nil
}

// xepGia là reranker giả: điểm cao cho đoạn chứa uuTien.
type xepGia struct{ uuTien string }

func (r xepGia) Rerank(_ context.Context, _ string, docs []string) ([]float64, error) {
	out := make([]float64, len(docs))
	for i, d := range docs {
		out[i] = 0.1
		if strings.Contains(d, r.uuTien) {
			out[i] = 0.9
		}
	}
	return out, nil
}

var khaiNiemThu = [][]string{{"nghiệm thu", "bàn giao"}, {"giàn giáo", "dàn giáo"}, {"bê tông", "vữa"}}

func khoLai() []Document {
	d := func(id string, v int, status, from, to string, acl, scope []string, title, body string) Document {
		return Document{ID: id, Tenant: "t", Version: v, Status: status, EffectiveFrom: from, EffectiveTo: to, ACL: acl, Scope: scope, Title: title, Body: body}
	}
	ks := []string{"ks"}
	return []Document{
		d("NT", 1, StatusApproved, "2025-01-01", "2026-03-01", ks, nil, "Quy trình nghiệm thu", "1. Biên bản nghiệm thu do kỹ sư lập."),
		d("NT", 2, StatusApproved, "2026-03-01", "", ks, nil, "Quy trình nghiệm thu", "1. Biên bản nghiệm thu do tư vấn giám sát lập."),
		d("NT", 3, StatusDraft, "2026-10-01", "", ks, nil, "Quy trình nghiệm thu", "1. Biên bản nghiệm thu do chủ đầu tư lập."),
		d("GG", 1, StatusApproved, "2025-01-01", "", ks, nil, "Quy trình giàn giáo", "1. Kiểm tra giàn giáo trước mỗi ca."),
		d("GG", 2, StatusApproved, "2026-01-01", "", ks, nil, "Quy trình giàn giáo", "1. Kiểm tra giàn giáo hằng tuần."),
		d("BT", 1, StatusApproved, "2025-01-01", "", ks, []string{"PRJ-A"}, "Biện pháp bê tông", "1. Đổ bê tông liên tục, đầm dùi sau mỗi lớp."),
		d("MAT", 1, StatusApproved, "2025-01-01", "", []string{"giam-doc"}, nil, "Quy chế bàn giao mật", "1. Hồ sơ bàn giao lưu két sắt."),
		// Cùng mã, bản 1 cho kỹ sư, bản 2 chỉ giám đốc đọc: mỗi người thấy đúng MỘT bản làm căn cứ, không ai thấy xung đột.
		d("QC", 1, StatusApproved, "2025-01-01", "", ks, nil, "Quy chế cấp phối", "1. Vữa trộn tại chỗ theo mác thiết kế."),
		d("QC", 2, StatusApproved, "2026-01-01", "", []string{"giam-doc"}, nil, "Quy chế cấp phối", "1. Vữa mác đặc biệt, chỉ giám đốc duyệt."),
	}
}

func chiMucLai(t *testing.T, docs []Document, e Embedder) *MemoryIndex {
	t.Helper()
	ix := NewMemoryIndex("t", docs)
	ix.Now = func() time.Time { return time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC) }
	texts := ix.EmbedTexts()
	vs, err := e.Embed(context.Background(), texts, false)
	if err != nil {
		t.Fatal(err)
	}
	sem := &Semantic{Embedder: e, Vectors: map[string][]float32{}}
	for i, s := range texts {
		sem.Vectors[EmbedKey(s)] = vs[i]
	}
	ix.Semantic = sem
	return ix
}

func ksLai(scope string) contracts.Principal {
	pr := contracts.Principal{ID: "u", TenantID: "t", Kind: contracts.PrincipalUser, Roles: []string{"ks"}}
	if scope != "" {
		pr.Attributes = map[string]string{"entity_scope": scope}
	}
	return pr
}

// Người hỏi dùng chữ khác tài liệu ("bàn giao" ↔ "nghiệm thu"): đường từ khoá không tìm ra, đường lai tìm ra bản ĐANG HIỆU LỰC (v2) —
// bản cũ, bản nháp không lọt; tài liệu cùng khái niệm mà người hỏi không được đọc (MAT, vai khác) không lọt.
func TestLaiTimDuocChuKhacTaiLieu(t *testing.T) {
	ctx := context.Background()
	const cau = "Ai ký nhận khi bàn giao?" // không chung từ nào với đoạn kỹ sư được đọc
	tu := NewMemoryIndex("t", khoLai())
	tu.Now = func() time.Time { return time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC) }
	if ev, err := tu.Retrieve(ctx, ksLai(""), cau, 3); err != nil || len(ev) != 0 {
		t.Fatalf("tiền đề: đường từ khoá phải không tìm ra (chữ khác tài liệu), được %v %v", ids(ev), err)
	}
	ix := chiMucLai(t, khoLai(), &nhungGia{khaiNiem: khaiNiemThu})
	ev, err := ix.Retrieve(ctx, ksLai(""), cau, 3)
	if err != nil || len(ev) == 0 || ev[0].ID != "NT@v2#0" {
		t.Fatalf("đường lai: muốn NT@v2#0 đứng đầu, được %v %v", ids(ev), err)
	}
	for _, e := range ev {
		if strings.HasPrefix(e.ID, "NT@v1") || strings.HasPrefix(e.ID, "NT@v3") || strings.HasPrefix(e.ID, "MAT") || strings.HasPrefix(e.ID, "BT") {
			t.Errorf("đoạn không được làm căn cứ / ngoài quyền / ngoài phạm vi lọt vào: %s", e.ID)
		}
	}
	// Người có vai giam-doc thấy MAT: cùng câu, tài liệu được đọc thì được trả (ca cho phép của cùng luật).
	gd := contracts.Principal{ID: "g", TenantID: "t", Kind: contracts.PrincipalUser, Roles: []string{"giam-doc"}}
	if ev, err := ix.Retrieve(ctx, gd, cau, 3); err != nil || len(ev) == 0 || !strings.HasPrefix(ev[0].ID, "MAT") {
		t.Errorf("giám đốc phải thấy MAT: %v %v", ids(ev), err)
	}
}

// Phạm vi trên đường lai: kỹ sư PRJ-B không thấy biện pháp của PRJ-A dù vector khớp tuyệt đối; kỹ sư PRJ-A thấy.
func TestLaiPhamVi(t *testing.T) {
	ix := chiMucLai(t, khoLai(), &nhungGia{khaiNiem: khaiNiemThu})
	const cau = "Đổ vữa thế nào?"
	// Không đặt ngưỡng: câu lạc đề với PRJ-B vẫn có đoạn đứng đầu (có thể là thông báo phiên bản của tài liệu khác) — bất biến là BT
	// không lọt, cả trong đoạn lẫn trong thông báo.
	ev, err := ix.Retrieve(context.Background(), ksLai("PRJ-B"), cau, 3)
	var n *VersionNotice
	if containsPrefix(ev, "BT") || errors.As(err, &n) && n.DocumentID == "BT" || err != nil && n == nil {
		t.Errorf("PRJ-B không được thấy BT: %v %v", ids(ev), err)
	}
	if ev, err := ix.Retrieve(context.Background(), ksLai("PRJ-A"), cau, 3); err != nil || len(ev) == 0 || !strings.HasPrefix(ev[0].ID, "BT") {
		t.Errorf("PRJ-A phải thấy BT đứng đầu: %v %v", ids(ev), err)
	}
}

// Cùng mã, bản người hỏi không được đọc KHÔNG lọt dù bản người hỏi đọc được đang làm căn cứ (lọc quyền theo từng BẢN, không theo mã).
func TestLaiQuyenTheoTungBan(t *testing.T) {
	ix := chiMucLai(t, khoLai(), &nhungGia{khaiNiem: khaiNiemThu})
	const cau = "Vữa trộn thế nào?"
	if ev, err := ix.Retrieve(context.Background(), ksLai(""), cau, 3); err != nil || containsPrefix(ev, "QC@v2") || !containsPrefix(ev, "QC@v1") {
		t.Errorf("kỹ sư: muốn QC@v1, không QC@v2; được %v %v", ids(ev), err)
	}
	gd := contracts.Principal{ID: "g", TenantID: "t", Kind: contracts.PrincipalUser, Roles: []string{"giam-doc"}}
	if ev, err := ix.Retrieve(context.Background(), gd, cau, 3); err != nil || containsPrefix(ev, "QC@v1") || !containsPrefix(ev, "QC@v2") {
		t.Errorf("giám đốc: muốn QC@v2, không QC@v1; được %v %v", ids(ev), err)
	}
}

// Tài liệu khớp nhất có hai bản cùng hiệu lực: báo xung đột như đường từ khoá, không chọn bản nào.
func TestLaiXungDotPhienBan(t *testing.T) {
	ix := chiMucLai(t, khoLai(), &nhungGia{khaiNiem: khaiNiemThu})
	_, err := ix.Retrieve(context.Background(), ksLai(""), "Kiểm tra dàn giáo khi nào?", 3)
	var n *VersionNotice
	if !errors.As(err, &n) || n.Kind != NoticeConflict || n.DocumentID != "GG" {
		t.Errorf("muốn thông báo xung đột của GG, được %v", err)
	}
}

// Tài liệu xung đột đứng SAU đoạn làm căn cứ: không thông báo (tài liệu khớp nhất làm căn cứ được), và đoạn của tài liệu xung đột
// không lọt vào bằng chứng.
func TestLaiXungDotDungSau(t *testing.T) {
	ix := chiMucLai(t, khoLai(), &nhungGia{khaiNiem: khaiNiemThu})
	ev, err := ix.Retrieve(context.Background(), ksLai(""), "Nghiệm thu dàn giáo thế nào?", 3)
	if err != nil || len(ev) == 0 || ev[0].ID != "NT@v2#0" || containsPrefix(ev, "GG") {
		t.Errorf("muốn NT@v2#0 đứng đầu, không đoạn GG nào; được %v %v", ids(ev), err)
	}
}

// Biết im: câu ngoài kho có cosine thấp dưới ngưỡng ⇒ không đoạn nào; với reranker, ngưỡng áp lên điểm xếp lại, và reranker đổi
// được thứ tự trong nhóm đầu.
func TestLaiBietImVaXepLai(t *testing.T) {
	ctx := context.Background()
	ix := chiMucLai(t, khoLai(), &nhungGia{khaiNiem: khaiNiemThu})
	ix.Semantic.MinScore = 0.5
	if ev, err := ix.Retrieve(ctx, ksLai(""), "Thủ tục xin giấy phép là gì?", 3); err != nil || len(ev) != 0 {
		t.Errorf("câu ngoài kho phải im, được %v %v", ids(ev), err)
	}
	if ev, err := ix.Retrieve(ctx, ksLai(""), "Ai lập văn bản bàn giao?", 3); err != nil || len(ev) == 0 {
		t.Errorf("câu trong kho không được im: %v %v", ids(ev), err)
	}
	ix.Semantic.Reranker = xepGia{uuTien: "tư vấn giám sát"}
	ix.Semantic.MinScore = 0.95
	if ev, err := ix.Retrieve(ctx, ksLai(""), "Ai lập văn bản bàn giao?", 3); err != nil || len(ev) != 0 {
		t.Errorf("điểm xếp lại 0,9 < ngưỡng 0,95 phải im, được %v %v", ids(ev), err)
	}
	ix.Semantic.MinScore = 0.5
	ix.Semantic.Reranker = xepGia{uuTien: "không đoạn nào có chữ này"}
	ev, err := ix.Retrieve(ctx, ksLai("PRJ-A"), "Đổ vữa thế nào?", 3)
	if err != nil || len(ev) != 0 {
		t.Errorf("reranker chấm mọi đoạn 0,1 < 0,5: phải im, được %v %v", ids(ev), err)
	}
	ix.Semantic.Reranker = xepGia{uuTien: "Kiểm tra giàn giáo trước"}
	ix.Semantic.MinScore = 0
	if _, err := ix.Retrieve(ctx, ksLai(""), "Ai lập văn bản bàn giao?", 3); err == nil {
		// GG (xung đột) được reranker đẩy lên đầu ⇒ thông báo xung đột, không trả đoạn nào.
		t.Errorf("reranker đẩy tài liệu xung đột lên đầu: muốn thông báo, được không lỗi")
	}
}

// Thiếu vector (nạp tài liệu mới / đổi model mà chưa nhúng lại) là LỖI dữ liệu nêu cách sửa, không âm thầm chỉ dùng từ khoá;
// lỗi của embedder đi nguyên nhãn.
func TestLaiThieuVectorVaLoiEmbedder(t *testing.T) {
	ctx := context.Background()
	e := &nhungGia{khaiNiem: khaiNiemThu}
	ix := chiMucLai(t, khoLai(), e)
	delete(ix.Semantic.Vectors, EmbedKey("Quy trình nghiệm thu: 1. Biên bản nghiệm thu do tư vấn giám sát lập.")) // đoạn ứng viên NT@v2#0
	if _, err := ix.Retrieve(ctx, ksLai(""), "Ai lập văn bản bàn giao?", 3); contracts.FaultOf(err) != contracts.FaultData || !strings.Contains(err.Error(), "nhúng lại") {
		t.Errorf("thiếu vector: muốn lỗi dữ liệu nêu 'nhúng lại', được %v", err)
	}
	ix = chiMucLai(t, khoLai(), e)
	e.loi = contracts.Errorf(contracts.ThirdParty("embed", contracts.KindRateLimit), "hết hạn mức")
	if _, err := ix.Retrieve(ctx, ksLai(""), "Ai lập văn bản bàn giao?", 3); contracts.FaultOf(err) != contracts.ThirdParty("embed", contracts.KindRateLimit) {
		t.Errorf("lỗi embedder phải giữ nhãn, được %v", err)
	}
	// Người hỏi không thấy tài liệu nào: không gọi embedder (không gửi câu hỏi ra ngoài khi chẳng có gì để so).
	e.loi, e.goi = nil, 0
	la := contracts.Principal{ID: "x", TenantID: "t", Kind: contracts.PrincipalUser, Roles: []string{"khach"}}
	if ev, err := ix.Retrieve(ctx, la, "Ai lập văn bản bàn giao?", 3); err != nil || len(ev) != 0 || e.goi != 0 {
		t.Errorf("không thấy tài liệu nào: muốn rỗng và 0 lời gọi embedder, được %v %v %d", ids(ev), err, e.goi)
	}
}

func ids(ev []contracts.Evidence) []string {
	var out []string
	for _, e := range ev {
		out = append(out, e.ID)
	}
	return out
}

// Chưa đặt ngưỡng điểm (MinScore 0): đường lai vẫn "biết im" bằng luật chủ đề của đường từ khoá — chủ đề ngoài kho thì không đoạn nào,
// và không gửi câu hỏi ra embedder (soát độc lập 23/10: không ngưỡng thì trước đây không bao giờ im).
func TestLaiKhongNguongDungLuatChuDe(t *testing.T) {
	e := &nhungGia{khaiNiem: khaiNiemThu}
	ix := chiMucLai(t, khoLai(), e)
	e.goi = 0
	ctx := contracts.WithKnowledgeQuery(context.Background(), contracts.KnowledgeQuery{AsOf: time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC),
		Topic: "thủ tục xin giấy phép xây dựng"})
	if ev, err := ix.Retrieve(ctx, ksLai(""), "Thủ tục xin giấy phép xây dựng gồm những gì?", 3); err != nil || len(ev) != 0 || e.goi != 0 {
		t.Errorf("chủ đề ngoài kho, không ngưỡng: muốn rỗng và 0 lời gọi embedder, được %v %v %d", ids(ev), err, e.goi)
	}
	ctx = contracts.WithKnowledgeQuery(context.Background(), contracts.KnowledgeQuery{AsOf: time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC),
		Topic: "biên bản nghiệm thu"})
	if ev, err := ix.Retrieve(ctx, ksLai(""), "Biên bản nghiệm thu do ai lập?", 3); err != nil || len(ev) == 0 || e.goi != 1 {
		t.Errorf("chủ đề trong kho: muốn có đoạn, 1 lời gọi embedder; được %v %v %d", ids(ev), err, e.goi)
	}
}

// Hai bản cùng mã có dòng giống hệt nhau (hoà điểm cả từ khoá lẫn cosine): kết quả không phụ thuộc thứ tự nạp tài liệu (đường bộ nhớ
// và đường PG phải như nhau — S7D-25; soát độc lập 23/10 bắt 7/3000 kho lật).
func TestLaiHoaDiemTheoBan(t *testing.T) {
	docs := []Document{
		{ID: "A", Tenant: "t", Version: 2, Status: StatusApproved, EffectiveFrom: "2025-01-01", ACL: []string{"ks"}, Title: "Quy trình nghiệm thu", Body: "1. Biên bản nghiệm thu."},
		{ID: "A", Tenant: "t", Version: 1, Status: StatusApproved, EffectiveFrom: "2025-01-01", ACL: []string{"ks"}, Title: "Quy trình nghiệm thu", Body: "1. Biên bản nghiệm thu."},
		{ID: "B", Tenant: "t", Version: 1, Status: StatusApproved, EffectiveFrom: "2025-01-01", ACL: []string{"ks"}, Title: "Quy trình bàn giao", Body: "1. Biên bản bàn giao."},
	}
	var want string
	for i, ds := range [][]Document{docs, {docs[1], docs[0], docs[2]}, {docs[2], docs[1], docs[0]}} {
		ix := chiMucLai(t, ds, &nhungGia{khaiNiem: khaiNiemThu})
		ev, err := ix.Retrieve(context.Background(), ksLai(""), "Biên bản bàn giao?", 3)
		got := fmt.Sprint(ids(ev), err)
		if i == 0 {
			want = got
		} else if got != want {
			t.Errorf("thứ tự nạp #%d đổi kết quả: %s, lần đầu %s", i, got, want)
		}
	}
}

func containsPrefix(ev []contracts.Evidence, p string) bool {
	for _, e := range ev {
		if strings.HasPrefix(e.ID, p) {
			return true
		}
	}
	return false
}
