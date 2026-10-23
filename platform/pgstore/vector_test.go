package pgstore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
)

// nhungDem là embedder giả tất định, đếm số chữ đã nhúng; vector = (độ dài chữ, số dấu cách, 1).
type nhungDem struct {
	ten string
	dem int
}

func (e *nhungDem) Name() string { return e.ten }

func (e *nhungDem) Embed(_ context.Context, texts []string, _ bool) ([][]float32, error) {
	e.dem += len(texts)
	out := make([][]float32, len(texts))
	for i, t := range texts {
		out[i] = []float32{float32(len(t)), float32(strings.Count(t, " ")), 1}
	}
	return out, nil
}

// nhungCho chặn lời gọi embedder đầu tiên tới khi đóng cho (boQua = các lời gọi sau không chặn).
type nhungCho struct {
	*nhungDem
	cho, vao chan struct{}
	boQua    bool
}

func (e *nhungCho) Embed(ctx context.Context, texts []string, q bool) ([][]float32, error) {
	if !e.boQua {
		close(e.vao)
		<-e.cho
	}
	return e.nhungDem.Embed(ctx, texts, q)
}

// Vector của tri thức trên PG (migration 0008): chưa nhúng thì lượt hỏi báo lỗi dữ liệu nêu cách sửa; nhúng lại chỉ nhúng chữ mới và
// xoá vector của chữ đã bỏ; chỉ mục giữ sẵn được DÙNG LẠI khi không có gì đổi (dấu nạp nhanh khớp dấu dựng) và dựng lại sau khi nhúng;
// tenant khác và không gian vector khác không dùng được vector này.
func TestVectorTriThucPG(t *testing.T) {
	admin, app := EnvDSN()
	if admin == "" || app == "" {
		t.Skip("chưa có EAIS_PG_ADMIN_DSN / EAIS_PG_DSN — xem deploy/docker-compose.yml")
	}
	ctx := context.Background()
	if err := Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	s, err := Open(ctx, app)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(s.Close)
	e := &nhungDem{ten: "gia/3"}
	s.Semantic = &rag.Semantic{Embedder: e}
	tenant := contracts.TenantID(fmt.Sprintf("vector-%d", time.Now().UnixNano()))
	tctx := contracts.WithTenant(ctx, tenant)
	pr := contracts.Principal{ID: "u", TenantID: tenant, Kind: contracts.PrincipalUser, Roles: []string{"ks"}}
	doc := func(body string) []rag.Document {
		return []rag.Document{
			{ID: "NT", Title: "Quy trình nghiệm thu", ACL: []string{"ks"}, Body: "1. Biên bản nghiệm thu do tư vấn giám sát lập.\n" + body},
			{ID: "GG", Title: "Quy trình giàn giáo", ACL: []string{"ks"}, Body: "1. Kiểm tra giàn giáo trước mỗi ca."},
		}
	}
	t.Cleanup(func() {
		_ = s.Load(tctx, nil, nil, nil)
		_, _, _ = s.EmbedKnowledge(ctx, tenant) // kho rỗng ⇒ xoá mọi vector của tenant thử
	})
	if err := s.Load(tctx, nil, nil, doc("2. Lưu hồ sơ.")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Retrieve(tctx, pr, "biên bản nghiệm thu", 3); contracts.FaultOf(err) != contracts.FaultData || !strings.Contains(err.Error(), "nhúng lại") {
		t.Fatalf("chưa nhúng: muốn lỗi dữ liệu nêu 'nhúng lại', được %v", err)
	}
	if e.dem != 0 {
		t.Errorf("kho chưa nhúng: lượt hỏi không được gửi câu hỏi ra embedder, đã gửi %d chữ", e.dem)
	}
	// nhung nhúng lại kho, trả số chữ embedder thực sự nhận (câu hỏi của các lượt hỏi không tính).
	nhung := func() (added, removed, sent int, err error) {
		truoc := e.dem
		added, removed, err = s.EmbedKnowledge(ctx, tenant)
		return added, removed, e.dem - truoc, err
	}
	if added, removed, sent, err := nhung(); err != nil || added != 3 || removed != 0 || sent != 3 {
		t.Fatalf("nhúng lần đầu: thêm %d, xoá %d, gửi %d chữ, %v (muốn 3, 0, 3)", added, removed, sent, err)
	}
	ev, err := s.Retrieve(tctx, pr, "biên bản nghiệm thu", 3)
	if err != nil || len(ev) == 0 {
		t.Fatalf("sau khi nhúng: %v %v", ev, err)
	}
	a, _ := s.knowledgeIndex(tctx, tenant)
	if b, _ := s.knowledgeIndex(tctx, tenant); a != b {
		t.Error("không có gì đổi mà chỉ mục bị dựng lại: dấu nạp nhanh lệch dấu dựng (mỗi lượt sẽ dựng lại)")
	}
	if added, _, sent, err := nhung(); err != nil || added != 0 || sent != 0 {
		t.Errorf("nhúng lại khi kho không đổi: thêm %d, gửi %d chữ (muốn 0, 0), %v", added, sent, err)
	}
	if b, _ := s.knowledgeIndex(tctx, tenant); a != b {
		t.Error("nhúng lại không thêm / xoá gì mà chỉ mục bị dựng lại")
	}

	// Sửa một bước: lượt hỏi báo thiếu vector cho tới khi nhúng lại; nhúng lại chỉ nhúng chữ mới, xoá vector của chữ cũ.
	if err := s.Load(tctx, nil, nil, doc("2. Lưu hồ sơ tại ban quản lý.")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Retrieve(tctx, pr, "biên bản nghiệm thu", 3); contracts.FaultOf(err) != contracts.FaultData {
		t.Errorf("tài liệu đổi mà chưa nhúng lại: muốn lỗi dữ liệu, được %v", err)
	}
	if added, removed, sent, err := nhung(); err != nil || added != 1 || removed != 1 || sent != 1 {
		t.Errorf("nhúng lại sau khi sửa một bước: thêm %d, xoá %d, gửi %d chữ, %v (muốn 1, 1, 1)", added, removed, sent, err)
	}
	c, _ := s.knowledgeIndex(tctx, tenant)
	if c == a {
		t.Error("nhúng lại xong mà chỉ mục không dựng lại")
	}
	if _, err := s.Retrieve(tctx, pr, "biên bản nghiệm thu", 3); err != nil {
		t.Errorf("sau khi nhúng lại: %v", err)
	}

	// Hai lần nhúng chạy đua: lần 1 chạy trên kho cũ, kẹt lúc gọi embedder; trong lúc đó kho đổi và lần 2 nhúng xong. Lần 1 chạy tiếp
	// không được xoá vector của kho mới (soát độc lập 23/10).
	cham := &nhungCho{nhungDem: e, cho: make(chan struct{}), vao: make(chan struct{})}
	s.Semantic = &rag.Semantic{Embedder: cham}
	if err := s.Load(tctx, nil, nil, doc("2. Lưu hồ sơ bản cứng.")); err != nil { // chữ mới ⇒ lần 1 phải gọi embedder
		t.Fatal(err)
	}
	xong := make(chan error)
	go func() { _, _, err := s.EmbedKnowledge(ctx, tenant); xong <- err }()
	<-cham.vao // lần 1 đang trong embedder, đã chốt "kho cũ"
	if err := s.Load(tctx, nil, nil, doc("2. Lưu hồ sơ bản mềm.")); err != nil {
		t.Fatal(err)
	}
	cham.boQua = true
	if _, _, err := s.EmbedKnowledge(ctx, tenant); err != nil {
		t.Fatalf("lần nhúng 2: %v", err)
	}
	close(cham.cho)
	if err := <-xong; err != nil {
		t.Fatalf("lần nhúng 1: %v", err)
	}
	if _, err := s.Retrieve(tctx, pr, "biên bản nghiệm thu", 3); err != nil {
		t.Errorf("lần nhúng chạy trên kho cũ đã xoá vector của kho mới: %v", err)
	}
	s.Semantic = &rag.Semantic{Embedder: e}

	// Tenant khác cùng tài liệu: không thấy vector của tenant này (RLS) ⇒ báo thiếu, không dùng nhờ.
	other := contracts.TenantID(string(tenant) + "-khac")
	octx := contracts.WithTenant(ctx, other)
	t.Cleanup(func() { _ = s.Load(octx, nil, nil, nil) })
	if err := s.Load(octx, nil, nil, doc("2. Lưu hồ sơ tại ban quản lý.")); err != nil {
		t.Fatal(err)
	}
	opr := pr
	opr.TenantID = other
	if _, err := s.Retrieve(octx, opr, "biên bản nghiệm thu", 3); contracts.FaultOf(err) != contracts.FaultData {
		t.Errorf("tenant khác dùng nhờ vector của tenant này: %v", err)
	}
	var n int
	if err := s.inTenant(ctx, other, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT count(*) FROM document_vectors").Scan(&n)
	}); err != nil || n != 0 {
		t.Errorf("RLS: tenant khác đọc được %d dòng vector, %v", n, err)
	}

	// Đổi model (không gian vector khác): vector cũ không bị so nhầm.
	s.Semantic = &rag.Semantic{Embedder: &nhungDem{ten: "gia-khac/3"}}
	if _, err := s.Retrieve(tctx, pr, "biên bản nghiệm thu", 3); contracts.FaultOf(err) != contracts.FaultData {
		t.Errorf("đổi không gian vector mà không báo thiếu: %v", err)
	}
	s.Semantic = &rag.Semantic{Embedder: e}

	// Kho rỗng: nhúng lại xoá HẾT vector của tenant (soát độc lập 23/10: trước đây ANY(NULL) không xoá gì, để rác).
	if err := s.Load(tctx, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, removed, err := s.EmbedKnowledge(ctx, tenant); err != nil || removed == 0 {
		t.Errorf("kho rỗng: muốn xoá hết vector, xoá %d, %v", removed, err)
	}
	if err := s.inTenant(ctx, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT count(*) FROM document_vectors").Scan(&n)
	}); err != nil || n != 0 {
		t.Errorf("kho rỗng còn %d dòng vector, %v", n, err)
	}

	// Không cấu hình embedding: EmbedKnowledge là lỗi cấu hình, không lặng lẽ không làm gì.
	s2 := &Store{pool: s.pool}
	if _, _, err := s2.EmbedKnowledge(ctx, tenant); contracts.FaultOf(err) != contracts.FaultCode {
		t.Errorf("chưa cấu hình embedding: muốn lỗi cấu hình, được %v", err)
	}
	var nf *contracts.KnowledgeNotice
	if _, err := s2.Retrieve(tctx, pr, "biên bản nghiệm thu", 3); err != nil && !errors.As(err, &nf) {
		t.Errorf("Store không cấu hình embedding vẫn phải chạy đường từ khoá như cũ: %v", err)
	}
}
