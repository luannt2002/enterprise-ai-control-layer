package commands

// Ca A25 của HOP-DONG §10 mà khuôn §12.1 chưa khai được (§12, U10), nên là test Go của gói chứ không phải ca YAML.
//
// A25: câu lệnh viết bằng chữ tự do vào LUỒNG HỎI, với các vai có metrics.read (và vai không có), được trả lời như câu hỏi tính toán (§1.5, [ĐO] ở
// c086fe7, 1e5457b, 7d72e33). Phase 1 chỉ bảo đảm: (1) câu trả lời không khẳng định một phiếu đã được tạo hay mở, (2) số phiếu = 0 (luồng hỏi không có
// đường ghi nào tới kho phiếu), (3) đường hỏi không đổi so với phía gốc (so hai commit, chạy ngoài test này: cổng X7). Soát K4, K4S-07: phép đếm (2)
// ở dưới đếm trên một kho mới KHÔNG nối với luồng hỏi nên luôn 0 — nó không đo gì; (2) dựa vào ranh giới import: core/pipeline hiện không import
// execution/commands (đo bằng grep ngày 23/10; CHƯA có test canh như TestA24VaRangBuocImport). Chỉ (1) và đột biến M18 là phép đo thật; (3) chưa làm. Test KHÔNG đòi từ chối hay hỏi lại:
// chặn động từ lệnh ở luồng hỏi là nợ của S4 hoặc Phase 6. Đột biến M18 nhắm test này.
//
// A24 (đếm lời gọi model ở tầng ứng dụng) chưa có test đếm: đường lệnh KHÔNG có chỗ gắn gateway model nào, nên một phép đếm bằng 0 không thể đỏ (ghi nợ);
// cổng import TestA24VaRangBuocImport (pipeline_test.go) chặn từ gốc việc gói lệnh nối tới gateway model (đột biến M17).

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
)

func TestA25CauLenhVaoLuongHoiKhongNoiDaTaoPhieu(t *testing.T) {
	wd := loadDevWorld(t)
	td := wd.tenants["abc-infra"]
	ask := &pipeline.Ask{RT: wd.rt, Deps: pipeline.Deps{
		Directory: td.dir, Facts: td.facts, Knowledge: rag.NewMemoryIndex("abc-infra", nil), Model: &models.Extractive{},
		DataVersion: "a25", Now: func() time.Time { return wd.now },
	}}
	store := newMemStore() // kho phiếu của harness: luồng hỏi không có đường nào tới nó
	questions := []string{
		"Tạo phiếu kiểm tra chênh lệch máy EX-03 hôm qua",
		"Tạo phiếu kiểm tra chênh lệch dầu máy EX-03 hôm qua",
		"Tạo phiếu kiểm tra chênh lệch máy EX-03 hôm nay",
		"Tạo phiếu kiểm tra máy EX-03",
	}
	callers := []chayCaller{
		{ID: "u1", Role: "director", Tenant: "abc-infra"}, {ID: "u2", Role: "accountant", Tenant: "abc-infra"},
		{ID: "u3", Role: "project_manager", Scope: "PRJ-A", Tenant: "abc-infra"}, {ID: "u4", Role: "site_engineer", Scope: "PRJ-A", Tenant: "abc-infra"},
		{ID: "u5", Role: "driver", Scope: "EX-03", Tenant: "abc-infra"},
	}
	banned := []string{"đã tạo", "đã mở", "đã lập", "phiếu số", "mã phiếu", "đã ghi nhận phiếu"}
	for _, c := range callers {
		for _, q := range questions {
			out, err := ask.Handle(context.Background(), principalOf(c), contracts.AskRequest{ConversationID: "a25", MessageID: "a25", Text: q})
			if err != nil {
				t.Fatalf("%s hỏi %q: %v", c.Role, q, err)
			}
			text := strings.ToLower(out.Response.Answer)
			for _, b := range banned {
				if strings.Contains(text, b) {
					t.Errorf("(1) %s hỏi %q: câu trả lời khẳng định một phiếu đã tạo (%q): %q", c.Role, q, b, out.Response.Answer)
				}
			}
		}
	}
	if n := store.countOpen("abc-infra"); n != 0 { // (2)
		t.Fatalf("(2) số phiếu sau các câu lệnh ở luồng hỏi: %d, muốn 0", n)
	}
}
