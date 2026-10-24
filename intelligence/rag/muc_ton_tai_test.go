package rag

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Câu hỏi nhắc MỤC ĐÁNH SỐ không có trong tài liệu người hỏi thấy ("Điều 999", "Chương XX") thì im ngay, tất định — điểm xếp lại của
// loại câu này lệch qua lại quanh ngưỡng giữa hai lần chạy (đo 24/10: "Chương XX" 0.212 ↔ 0.179, một lượt đã trả lời về chương không
// có). Tiền tố đánh số học từ chính tiêu đề của kho (từ đầu của ≥ 2 tiêu đề có số / la mã VIẾT HOA theo sau), so GIỮ DẤU; mục "có"
// nếu cụm đó xuất hiện ở BẤT KỲ đâu trong tài liệu người hỏi thấy (kể cả dẫn chiếu trong thân), la mã ↔ số và số 0 đầu coi như nhau.
// Câu nhắc cả mục có lẫn mục không, câu không nhắc mục nào, câu không dấu, kho không có tiêu đề: không đổi. Mục chỉ có ở tài liệu ngoài
// vai = không có (§6.2). Soát độc lập 24/10: các ca chặn oan (mức ↔ mục, Chương 2 ↔ Chương II, "những điều vi phạm"…) là ca dương dưới.
func TestMucKhongTonTai(t *testing.T) {
	docs := []Document{
		{ID: "QC", Tenant: "t", Title: "Quy chế an toàn", ACL: []string{"ks"}, Body: thanQuyDinh + "\nTheo Điều 25 Luật An ninh mạng.\n## Lưu ý\nHồ sơ.\n## Lưu ý\nKhác."},
		{ID: "MAT", Tenant: "t", Title: "Quy chế mật", ACL: []string{"giam-doc"}, Body: "## Chương IX\n#### Điều 99. Hồ sơ mật\n1. Lưu két sắt."},
	}
	ix := NewMemoryIndex("t", docs)
	ix.Now = func() time.Time { return time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC) }
	ks := contracts.Principal{ID: "u", TenantID: "t", Kind: contracts.PrincipalUser, Roles: []string{"ks"}}
	gd := contracts.Principal{ID: "g", TenantID: "t", Kind: contracts.PrincipalUser, Roles: []string{"giam-doc"}}
	hoi := func(pr contracts.Principal, q string) ([]contracts.Evidence, contracts.RetrievalReport) {
		t.Helper()
		var rep contracts.RetrievalReport
		ev, err := ix.Retrieve(contracts.WithKnowledgeQuery(context.Background(), contracts.KnowledgeQuery{Report: &rep}), pr, q, 3)
		if err != nil {
			t.Fatalf("%q: %v", q, err)
		}
		return ev, rep
	}
	for _, q := range []string{"Điều 999 quy định gì?", "Chương XX nói về gì?", "Điều 99 và Chương IX quy định gì?", "Điều 0999 là gì?"} {
		if ev, rep := hoi(ks, q); len(ev) != 0 || !strings.Contains(rep.Silence, "không có trong tài liệu người hỏi thấy") {
			t.Errorf("ks %q: muốn im vì mục không có, được %v %+v", q, ids(ev), rep)
		} else if strings.Contains(rep.Silence, "mật") || strings.Contains(rep.Silence, "MAT") {
			t.Errorf("lý do im lộ tài liệu ngoài vai: %q", rep.Silence)
		}
	}
	for _, q := range []string{"Điều 18 quy định gì?", "Điều 18 và Điều 999 quy định gì?", "Chương II nói về trung tâm dữ liệu thế nào?", "Thiết bị di động là gì?",
		"Chương 2 quy định những gì?", "Điều 018 quy định gì?", "dieu 999 quy dinh gi", "Điều 25 Luật An ninh mạng quy định gì?",
		"Những điều vi phạm nào bị xử lý?", "Điều e ngại nhất là gì?", "chương xx là gì?", "Hồ sơ phải lưu 5 năm không?", "Vi phạm bị phạt mức 5 triệu phải không?"} {
		if _, rep := hoi(ks, q); strings.Contains(rep.Silence, "không có trong tài liệu") {
			t.Errorf("ks %q: mục có (hoặc không nhắc mục) không được im vì luật này: %+v", q, rep)
		}
	}
	if ev, rep := hoi(gd, "Điều 99 quy định gì?"); len(ev) == 0 || rep.Silence != "" {
		t.Errorf("giám đốc thấy Điều 99: phải trả đoạn, được %v %+v", ids(ev), rep)
	}
	// Lý do im mang chữ của người hỏi: bọc Quote để bộ che (WithholdQuoted) che được khi câu mang bí mật.
	if _, rep := hoi(ks, "Điều 999 quy định gì?"); !strings.Contains(rep.Silence, contracts.Quote("điều 999")) {
		t.Errorf("lý do im phải bọc Quote: %q", rep.Silence)
	}
	// "mức" không phải "mục" (so giữ dấu): kho có tiêu đề "Mục 1", "Mục 2".
	muc := NewMemoryIndex("t", []Document{{ID: "MU", Tenant: "t", Title: "Quy chế", ACL: []string{"ks"}, Body: "## Mục 1\nGhi chú.\n## Mục 2\nGhi chú khác."}})
	var repMuc contracts.RetrievalReport
	if _, err := muc.Retrieve(contracts.WithKnowledgeQuery(context.Background(), contracts.KnowledgeQuery{Report: &repMuc}), ks, "Vi phạm bị phạt mức 5 triệu phải không?", 3); err != nil || strings.Contains(repMuc.Silence, "không có trong tài liệu") {
		t.Errorf("mức ≠ mục: không được im, được %v %+v", err, repMuc)
	}
	if _, err := muc.Retrieve(contracts.WithKnowledgeQuery(context.Background(), contracts.KnowledgeQuery{Report: &repMuc}), ks, "Mục 7 ghi gì?", 3); err != nil || !strings.Contains(repMuc.Silence, "không có trong tài liệu") {
		t.Errorf("Mục 7 không có: phải im, được %v %+v", err, repMuc)
	}
	// Soát lại 24/10: mục có thật mà viết đậm / có dấu "/" / nằm trong khoảng "3–5" vẫn là có; chữ cái có dấu ("Mục Ý kiến") và câu viết
	// HOA toàn bộ ("ĐIỀU E NGẠI…") không tạo nhãn chữ.
	tang := NewMemoryIndex("t", []Document{{ID: "TG", Tenant: "t", Title: "Biện pháp thi công", ACL: []string{"ks"},
		Body: "## Tầng 1\nMóng.\n## Tầng 2\nSàn.\nCác tầng 3–5 dùng mác 300.\nThông tư này hướng dẫn **Điều 25** và Điều 7/ Luật Xây dựng.\n## Mục 1\nA.\n## Mục 2\nB.\n## Điều 1\nC.\n## Điều 2\nD."}})
	for _, q := range []string{"Tầng 3 dùng bê tông mác gì?", "Tầng 4 dùng mác gì?", "Điều 25 Luật Xây dựng nói gì?", "Điều 7 quy định gì?",
		"Mục Ý kiến khách hàng ghi gì?", "ĐIỀU E NGẠI NHẤT LÀ GÌ?", "NHỮNG ĐIỀU Ở TRÊN ÁP DỤNG CHO AI?"} {
		var r contracts.RetrievalReport
		if _, err := tang.Retrieve(contracts.WithKnowledgeQuery(context.Background(), contracts.KnowledgeQuery{Report: &r}), ks, q, 3); err != nil || strings.Contains(r.Silence, "không có trong tài liệu") {
			t.Errorf("%q: không được im vì luật mục, được %v %+v", q, err, r)
		}
	}
	var rt contracts.RetrievalReport
	if _, err := tang.Retrieve(contracts.WithKnowledgeQuery(context.Background(), contracts.KnowledgeQuery{Report: &rt}), ks, "Tầng 9 dùng mác gì?", 3); err != nil || !strings.Contains(rt.Silence, "không có trong tài liệu") {
		t.Errorf("Tầng 9 không có: phải im, được %v %+v", err, rt)
	}
	// Chỉ MỘT tiêu đề đánh số: chưa đủ để coi là hệ đánh số (bảo thủ) — luật không chạy.
	mot := NewMemoryIndex("t", []Document{{ID: "M1", Tenant: "t", Title: "Quy chế", ACL: []string{"ks"}, Body: "#### Điều 1. Phạm vi\n1. Áp dụng cho kho."}})
	var rep1 contracts.RetrievalReport
	if _, err := mot.Retrieve(contracts.WithKnowledgeQuery(context.Background(), contracts.KnowledgeQuery{Report: &rep1}), ks, "Điều 2 quy định gì?", 3); err != nil || strings.Contains(rep1.Silence, "không có trong tài liệu") {
		t.Errorf("một tiêu đề đánh số: luật không được chạy, được %v %+v", err, rep1)
	}
	// Kho không có tiêu đề: không học được tiền tố nào, luật không chạy.
	sop := NewMemoryIndex("t", []Document{{ID: "SOP", Tenant: "t", Title: "Quy trình", ACL: []string{"ks"}, Body: "1. Điều 999 là mã nội bộ của bước kiểm tra."}})
	var rep contracts.RetrievalReport
	if _, err := sop.Retrieve(contracts.WithKnowledgeQuery(context.Background(), contracts.KnowledgeQuery{Report: &rep}), ks, "Điều 999 là gì?", 3); err != nil || strings.Contains(rep.Silence, "không có trong tài liệu") {
		t.Errorf("kho không có tiêu đề: luật không được chạy, được %v %+v", err, rep)
	}
}
