package ragcases

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// Đo luồng RAG với ĐỘNG CƠ THẬT (embedding, rerank, LLM) qua đúng đường của ứng dụng: PG → chỉ mục lai → pipeline S0–S10. Chỉ chạy
// khi người chạy bật EAIS_RAG_DONG_CO=1 và nạp cấu hình nhà cung cấp vào môi trường (EAIS_EMBED_*, EAIS_RERANK_*, EAIS_MODEL_*,
// EAIS_RAG_MIN_SCORE, EAIS_RAG_POOL — khuôn ở .env.example). Bộ câu: như TestS7BoPhatTrien (EAIS_S7_BO chọn bộ khác).
// Tài liệu và câu hỏi của bộ được GỬI RA nhà cung cấp đã cấu hình: chỉ chạy với dữ liệu giả lập / công khai.
// Đây là phép ĐO (in số), không phải cổng: vế tuyệt đối duy nhất là 0 đoạn ngoài quyền / không làm căn cứ / thiếu nguồn.
func TestDongCoThat(t *testing.T) {
	if os.Getenv("EAIS_RAG_DONG_CO") != "1" {
		t.Skip("đặt EAIS_RAG_DONG_CO=1 và cấu hình nhà cung cấp để đo với động cơ thật (gửi dữ liệu ra ngoài)")
	}
	admin, app := pgstore.EnvDSN()
	if admin == "" || app == "" {
		t.Fatal("cần EAIS_PG_ADMIN_DSN / EAIS_PG_DSN")
	}
	bo, docs, rt := napS7Do(t)
	sem := semanticTuMoiTruong(t)
	gw, err := models.New(models.ConfigFromEnv())
	if err != nil {
		t.Fatalf("cấu hình model: %v", err)
	}
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	s, err := pgstore.Open(ctx, app)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(s.Close)
	s.Semantic = sem
	tid := contracts.TenantID(fmt.Sprintf("rag-that-%d", time.Now().UnixNano()))
	tctx := contracts.WithTenant(ctx, tid)
	if err := s.Load(tctx, nil, nil, docs); err != nil {
		t.Fatalf("nạp PG: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Load(tctx, nil, nil, nil)
		_, _, _ = s.EmbedKnowledge(ctx, tid) // kho rỗng ⇒ xoá vector của tenant thử
	})
	t0 := time.Now()
	added, _, err := s.EmbedKnowledge(ctx, tid)
	if err != nil {
		t.Fatalf("nhúng kho: %v", err)
	}
	t.Logf("nhúng %d chữ trong %s (không gian %s)", added, time.Since(t0).Round(time.Second), sem.Embedder.Name())

	// S7 — truy hồi.
	kq := doS7(t, bo, docs, tid, rt, s.Retrieve)
	t.Logf("S7 lai: %s", kq)
	for _, x := range [][]string{kq.ngoaiQuyen, kq.khongCanCu, kq.thieuNguon} {
		if len(x) > 0 {
			t.Errorf("vế tuyệt đối hỏng: %v", x)
		}
	}
	for _, c := range bo.Cau {
		if c.Loai == "tra_loi" && !kq.trungTheoCau[c.Ma] {
			t.Logf("  trượt top-3: %s %q → %v", c.Ma, c.Hoi, kq.top3[c.Ma])
		}
		if c.Loai == "khong_co" && len(kq.top3[c.Ma]) > 0 {
			t.Logf("  không có đáp án mà nhận đoạn: %s %q → %v", c.Ma, c.Hoi, kq.top3[c.Ma])
		}
	}

	// Luồng hỏi trọn (S0–S10) với model đã cấu hình. Runtime là bản sao của pack mang tenant thử: S1 đòi tenant người hỏi = tenant của
	// pack, và nạp vào tenant thật của pack sẽ đè dữ liệu của nó trong DB dùng chung.
	rtThu := *rt
	rtThu.Meta.Tenant = string(tid)
	rt = &rtThu
	var dung, trichSai, khongTraLoi, s9, traNham, imDung, loi int
	var tre []time.Duration
	for _, c := range bo.Cau {
		day, _ := ngayS7(bo, c)
		pr := contracts.Principal{ID: "nguoi-hoi", TenantID: tid, Kind: contracts.PrincipalUser, Roles: c.Vai,
			Attributes: map[string]string{"entity_scope": strings.Join(c.PhamVi, ",")}}
		a := &pipeline.Ask{RT: rt, Deps: pipeline.Deps{Directory: s, Facts: s, Knowledge: s, Model: gw, Now: func() time.Time { return day.Add(10 * time.Hour) }}}
		t1 := time.Now()
		out, err := a.Handle(tctx, pr, contracts.AskRequest{ConversationID: "do", MessageID: c.Ma, Text: c.Hoi})
		tre = append(tre, time.Since(t1))
		r := out.Response
		switch {
		case err != nil:
			loi++
			t.Logf("  %s lỗi: %v", c.Ma, err)
		case r.Outcome == contracts.OutcomeSystemFailure: // kể cả câu không có đáp án: bị S9 chặn KHÔNG phải "im đúng"
			s9++
			t.Logf("  %s bị S9 chặn: %+v", c.Ma, r.Verdict.Violations)
		case c.Loai == "khong_co" && r.Outcome == contracts.OutcomeResolved:
			traNham++
			t.Logf("  %s không có đáp án mà trả lời: %q", c.Ma, r.Answer)
		case c.Loai == "khong_co":
			imDung++
		case r.Outcome == contracts.OutcomeResolved && slices.ContainsFunc(r.Citations, func(ci contracts.Citation) bool {
			return slices.ContainsFunc(c.DapAn, func(d s7DapAn) bool { return ci.Source.DocumentID == d.TaiLieu && ci.Source.DocumentVersion == d.Ban })
		}):
			dung++
		case r.Outcome == contracts.OutcomeResolved:
			trichSai++
			t.Logf("  %s trả lời nhưng trích tài liệu khác đáp án: %q %+v", c.Ma, r.Answer, r.Citations)
		default:
			khongTraLoi++
			t.Logf("  %s không trả lời (%s): %q", c.Ma, r.Outcome, c.Hoi)
		}
	}
	slices.Sort(tre)
	t.Logf("luồng hỏi trọn: trả lời trích đúng tài liệu %d · trích tài liệu khác %d · không trả lời %d — câu có đáp án %d; "+
		"câu không có đáp án: im đúng %d · trả nhầm %d; S9 chặn (cả hai loại) %d · lỗi %d; trễ trung vị %s, lớn nhất %s",
		dung, trichSai, khongTraLoi, kq.traLoi, imDung, traNham, s9, loi, tre[len(tre)/2].Round(time.Millisecond), tre[len(tre)-1].Round(time.Millisecond))
}

// semanticTuMoiTruong dựng truy hồi lai từ cấu hình: embedder bắt buộc; reranker khi EAIS_RERANK_PROVIDER khác rỗng.
func semanticTuMoiTruong(t *testing.T) *rag.Semantic {
	t.Helper()
	e, err := models.NewEmbedder(models.ServiceFromEnv("EMBED"))
	if err != nil {
		t.Fatalf("cấu hình embedding: %v", err)
	}
	sem := &rag.Semantic{Embedder: e}
	if os.Getenv("EAIS_RERANK_PROVIDER") != "" {
		r, err := models.NewReranker(models.ServiceFromEnv("RERANK"))
		if err != nil {
			t.Fatalf("cấu hình rerank: %v", err)
		}
		sem.Reranker = r
	}
	if v := os.Getenv("EAIS_RAG_MIN_SCORE"); v != "" {
		if sem.MinScore, err = strconv.ParseFloat(v, 64); err != nil {
			t.Fatalf("EAIS_RAG_MIN_SCORE %q: %v", v, err)
		}
	}
	if v := os.Getenv("EAIS_RAG_POOL"); v != "" {
		if sem.Pool, err = strconv.Atoi(v); err != nil || sem.Pool <= 0 {
			t.Fatalf("EAIS_RAG_POOL %q phải là số nguyên dương", v)
		}
	}
	return sem
}
