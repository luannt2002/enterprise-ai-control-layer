// Package knowledgebase_test là ví dụ kho tri thức TRUNG TÍNH (F-028): sổ tay quy trình nội bộ giả lập — nghỉ phép, mua sắm, bảo mật —
// chạy qua động cơ RAG, qua PostgreSQL (nạp chỉ tài liệu) và qua trọn luồng hỏi S0–S10 với một pack chỉ có tài liệu. Không đọc gì của
// pack tham chiếu ở tenants/: RAG không cần dữ liệu của miền nào để chạy.
package knowledgebase_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// ngayHoi là ngày hỏi cố định của ví dụ: bản 2 của quy trình nghỉ phép đang hiệu lực.
var ngayHoi = time.Date(2026, 10, 20, 10, 0, 0, 0, time.FixedZone("ICT", 7*3600))

func nguoi(tn, vai string) contracts.Principal {
	return contracts.Principal{ID: "u-" + vai, TenantID: contracts.TenantID(tn), Kind: contracts.PrincipalUser, Roles: []string{vai}}
}

// ca là một câu hỏi của ví dụ và kết cục truy xuất mong đợi: tài liệu của đoạn đứng đầu (mã@bản) và chữ mà một đoạn trong top-3 phải
// chứa, hoặc thông báo phiên bản; cam = tài liệu không được có đoạn nào trong kết quả (người hỏi không được đọc).
type ca struct {
	ten, vai, hoi string
	ngay          time.Time
	dau, chua     string // dau và cam cùng rỗng = không đoạn nào
	cam           string
	thongBao      contracts.KnowledgeNoticeKind
}

var cacCa = []ca{
	{ten: "bản đang hiệu lực", vai: "nhan-vien", hoi: "Xin nghỉ phép năm phải gửi đơn trước bao lâu?", ngay: ngayHoi, dau: "QT-NGHI-PHEP@v2", chua: "5 ngày làm việc"},
	{ten: "bản của ngày hỏi cũ", vai: "nhan-vien", hoi: "Xin nghỉ phép năm phải gửi đơn trước bao lâu?", ngay: time.Date(2025, 6, 1, 10, 0, 0, 0, ngayHoi.Location()), dau: "QT-NGHI-PHEP@v1", chua: "3 ngày làm việc"},
	{ten: "hàng bảng mang tên cột", vai: "quan-ly", hoi: "Khoản mua trên 50 triệu đồng do ai phê duyệt?", ngay: ngayHoi, dau: "QT-MUA-SAM@v1", chua: "| Giá trị khoản mua | Người phê duyệt |\n| Trên 50 triệu đồng | Giám đốc điều hành |"},
	{ten: "ngoài quyền đọc", vai: "nhan-vien", hoi: "Mật khẩu tài khoản quản trị phải đổi sau bao lâu?", ngay: ngayHoi, cam: "QT-BAO-MAT"},
	{ten: "trong quyền đọc", vai: "quan-tri", hoi: "Mật khẩu tài khoản quản trị phải đổi sau bao lâu?", ngay: ngayHoi, dau: "QT-BAO-MAT@v1", chua: "90 ngày"},
	{ten: "bản nháp không phục vụ", vai: "nhan-vien", hoi: "Quy trình đăng ký làm thêm giờ thế nào?", ngay: ngayHoi, thongBao: contracts.NoticeNoEffectiveVersion},
	{ten: "ngoài kho", vai: "nhan-vien", hoi: "Căng tin mở cửa vào cuối tuần không?", ngay: ngayHoi},
}

// chay chạy mọi ca trên một đường truy xuất, trả kết cục từng ca dạng chữ (để so hai đường).
func chay(t *testing.T, tn string, truy func(ctx context.Context, pr contracts.Principal, q string, k int) ([]contracts.Evidence, error)) []string {
	t.Helper()
	var out []string
	for _, c := range cacCa {
		ctx := contracts.WithKnowledgeQuery(contracts.WithTenant(context.Background(), contracts.TenantID(tn)), contracts.KnowledgeQuery{AsOf: c.ngay})
		ev, err := truy(ctx, nguoi(tn, c.vai), c.hoi, 3)
		var n *contracts.KnowledgeNotice
		switch {
		case errors.As(err, &n):
			if n.Kind != c.thongBao {
				t.Errorf("%s: thông báo %s, muốn %q", c.ten, n.Kind, c.thongBao)
			}
			out = append(out, c.ten+": "+string(n.Kind))
			continue
		case err != nil:
			t.Fatalf("%s: %v", c.ten, err)
		case c.thongBao != "":
			t.Errorf("%s: muốn thông báo %s, được %d đoạn", c.ten, c.thongBao, len(ev))
		case c.cam != "" && slices.ContainsFunc(ev, func(e contracts.Evidence) bool { return e.Source.DocumentID == c.cam }):
			t.Errorf("%s: lọt đoạn của %s: %+v", c.ten, c.cam, ev)
		case c.dau == "" && c.cam == "" && len(ev) > 0:
			t.Errorf("%s: muốn 0 đoạn, được %s", c.ten, ev[0].ID)
		case c.dau != "" && (len(ev) == 0 || !strings.HasPrefix(ev[0].ID, c.dau+"#") ||
			!slices.ContainsFunc(ev, func(e contracts.Evidence) bool { return strings.Contains(e.Text, c.chua) })):
			t.Errorf("%s: muốn đoạn đầu thuộc %s, một đoạn chứa %q; được %+v", c.ten, c.dau, c.chua, ev)
		}
		var ids []string
		for _, e := range ev {
			ids = append(ids, e.ID)
		}
		out = append(out, fmt.Sprintf("%s: %v", c.ten, ids))
	}
	return out
}

func napKho(t *testing.T, tn string) []rag.Document {
	t.Helper()
	docs, err := rag.LoadMarkdown("docs", tn)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 5 {
		t.Fatalf("kho ví dụ có 5 bản tài liệu, nạp được %d", len(docs))
	}
	for _, d := range docs {
		if d.Hold != "" {
			t.Errorf("%s@v%d bị cổng chất lượng giữ: %s", d.ID, d.Version, d.Hold)
		}
	}
	return docs
}

// Động cơ RAG trên kho trung tính, trong bộ nhớ: phiên bản theo ngày hỏi, quyền đọc theo vai, bản nháp không phục vụ, hàng bảng mang
// tên cột, câu ngoài kho không nhận đoạn nào.
func TestKhoTriThucTrungTinh(t *testing.T) {
	const tn = "so-tay-mau"
	chay(t, tn, rag.NewMemoryIndex(tn, napKho(t, tn)).Retrieve)
}

// Cùng kho qua PostgreSQL: nạp CHỈ tài liệu (LoadDocuments, không fact, không danh mục) rồi truy xuất qua RLS — cùng kết cục từng ca
// như đường bộ nhớ.
func TestKhoTriThucTrungTinhQuaPG(t *testing.T) {
	admin, app := pgstore.EnvDSN()
	if admin == "" || app == "" {
		t.Skip("chưa có EAIS_PG_ADMIN_DSN / EAIS_PG_DSN — xem deploy/docker-compose.yml")
	}
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	s, err := pgstore.Open(ctx, app)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	tn := fmt.Sprintf("so-tay-mau-%d", time.Now().UnixNano())
	tctx := contracts.WithTenant(ctx, contracts.TenantID(tn))
	docs := napKho(t, tn)
	if err := s.LoadDocuments(tctx, pgstore.LoadInfo{}, docs); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.LoadDocuments(tctx, pgstore.LoadInfo{}, nil) })
	pg := chay(t, tn, s.Retrieve)
	mem := chay(t, tn, rag.NewMemoryIndex(tn, docs).Retrieve)
	if fmt.Sprint(pg) != fmt.Sprint(mem) {
		t.Errorf("PG khác bộ nhớ:\n PG:      %v\n bộ nhớ: %v", pg, mem)
	}
}

// Trọn luồng hỏi S0–S10 với pack CHỈ có tài liệu (pack/: không chỉ số, không thực thể, model trích tất định): câu hỏi quy trình được
// trả lời kèm trích dẫn bản đang hiệu lực; vai không có quyền đọc tài liệu bị từ chối; câu ngoài phạm vi, câu về bản nháp không được
// trả lời từ tài liệu.
func TestLuongHoiTrungTinh(t *testing.T) {
	rt, err := tenant.Load("pack")
	if err != nil {
		t.Fatalf("pack trung tính: %v", err)
	}
	tn := rt.Meta.Tenant
	m := &models.Extractive{}
	a := &pipeline.Ask{RT: rt, Deps: pipeline.Deps{Directory: entityresolution.MemoryDirectory{}, Facts: sourceoftruth.MemoryStore{},
		Knowledge: rag.NewMemoryIndex(tn, napKho(t, tn)), Model: m, Now: func() time.Time { return ngayHoi }}}
	hoi := func(vai, q string) contracts.AskResponse {
		t.Helper()
		out, err := a.Handle(context.Background(), nguoi(tn, vai), contracts.AskRequest{ConversationID: "c", MessageID: "m", Text: q})
		if err != nil {
			t.Fatalf("%q: %v", q, err)
		}
		return out.Response
	}
	r := hoi("nhan-vien", "Quy trình xin nghỉ phép năm phải gửi đơn trước bao lâu?")
	if r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, "5 ngày làm việc") || strings.Contains(r.Answer, "3 ngày") ||
		len(r.Citations) == 0 || r.Citations[0].Source.DocumentID != "QT-NGHI-PHEP" || r.Citations[0].Source.DocumentVersion != 2 {
		t.Errorf("câu quy trình: muốn trích QT-NGHI-PHEP bản 2, được %s → %q %+v", r.Outcome, r.Answer, r.Citations)
	}
	if r := hoi("khach", "Quy trình xin nghỉ phép năm thế nào?"); r.Outcome != contracts.OutcomePermissionDenied || len(r.Citations) != 0 {
		t.Errorf("vai không có documents.read: muốn PERMISSION_DENIED, được %s → %q", r.Outcome, r.Answer)
	}
	if r := hoi("nhan-vien", "Thời tiết hôm nay thế nào?"); r.Outcome != contracts.OutcomeOutOfScope {
		t.Errorf("câu ngoài phạm vi: muốn OUT_OF_SCOPE, được %s → %q", r.Outcome, r.Answer)
	}
	if r := hoi("nhan-vien", "Quy trình đăng ký làm thêm giờ thế nào?"); r.Outcome != contracts.OutcomeInsufficientEvidence || strings.Contains(r.Answer, "trưởng nhóm") {
		t.Errorf("chỉ có bản nháp: muốn INSUFFICIENT_EVIDENCE, không trích bản nháp; được %s → %q", r.Outcome, r.Answer)
	}
}
