package construction_test

import (
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
)

// V8AS-02 (soát vòng 1 của V8a, hẹp). Đoạn đầu bản nháp ở S8 là chữ tài liệu, và nó từng bị cắt ở 80 ký tự TRƯỚC khi bộ che phạm vi (C9) chạy: bộ che chỉ khớp nguyên
// token, nên khoá hay tên thực thể ngoài phạm vi nằm vắt qua ký tự thứ 80 chỉ còn mảnh đầu ("Máy ủi DZ", "DZ-0") — không còn khớp — và lọt vào vết của chính người hỏi
// hạn chế. Phải che TRƯỚC rồi mới cắt: không mảnh nào của tên / khoá còn lại, dù chỗ cắt rơi vào đâu.

// Tài liệu (kỹ sư PRJ-A được đọc) nhắc máy DZ-03 của PRJ-B; kỹ sư hỏi "máy 03" nên DZ-03 là ứng viên NGOÀI phạm vi của lượt. Tên đứng sau pad ký tự đệm, quét pad để chỗ
// cắt 80 lần lượt rơi trước tên, vào từng chỗ của tên và sau tên. Bản nháp không có số hay đơn vị nên S9 nhận nó: đoạn đầu mới hiện ở S8.
func TestVetS8KhoaVatQuaChoCatKhongLot(t *testing.T) {
	const (
		name = "Máy ủi DZ-03"
		q    = "Quy trình xử lý chênh lệch nhiên liệu của máy 03 là gì?"
		tail = " lập biên bản đối soát nhiên liệu trong vòng hai ngày."
	)
	nameRunes := len([]rune(name))
	// mảnh của tên và của khoá mà không chỗ nào khác của dòng S8 chứa ("Má" không có trong "extractive", "diễn đạt", phần đuôi hay chữ đệm)
	fragments := []string{"Má", "ủi", "DZ", "Z-0", "-03"}
	straddled := 0
	for pad := 56; pad <= 84; pad++ {
		a := newAsk(t)
		a.Deps.Knowledge = rag.NewMemoryIndex("abc-infra", []rag.Document{{ID: "SOP-FUEL-03", Tenant: "abc-infra", Title: "Quy trình xử lý chênh lệch nhiên liệu của máy 03",
			ACL: []string{"director", "site_engineer"}, Body: strings.Repeat("x", pad) + " " + name + tail}})
		a.Deps.Model = &models.Extractive{}
		eng, dir := ask(t, a, user("site_engineer", "PRJ-A"), q), ask(t, a, user("director", ""), q)
		if eng.Plan.Route != contracts.RouteRAG || dir.Plan.Route != contracts.RouteRAG || eng.Response.Outcome != contracts.OutcomeResolved {
			t.Fatalf("pad %d, tiền đề: route RAG và S9 phải nhận bản nháp (%s / %s, outcome %s)", pad, eng.Plan.Route, dir.Plan.Route, eng.Response.Outcome)
		}
		engS8, dirS8 := stepOfRecord(t, recordOfTurn(eng), "S8").Reason, stepOfRecord(t, recordOfTurn(dir), "S8").Reason
		for _, f := range fragments {
			if strings.Contains(engS8, f) {
				t.Errorf("pad %d: mảnh %q của tên / khoá ngoài phạm vi lọt vào đoạn đầu S8 của kỹ sư PRJ-A:\n  %s", pad, f, engS8)
			}
		}
		if !strings.Contains(engS8, "đầu “") || !strings.HasSuffix(engS8, "”") {
			t.Errorf("pad %d: S8 của kỹ sư phải còn đoạn đầu trong cặp “…”: %s", pad, engS8)
		}
		// đối chứng: người toàn quyền thấy chữ tài liệu như cũ — tên còn nguyên khi nó nằm trọn trong 80 ký tự đầu, bị cắt dở khi vắt qua ký tự thứ 80
		if pad+1+nameRunes <= 80 && !strings.Contains(dirS8, strings.Repeat("x", pad)+" "+name) {
			t.Errorf("pad %d: người toàn quyền phải thấy tên trong đoạn đầu: %s", pad, dirS8)
		}
		if pad+1 < 80 && pad+1+nameRunes > 80 {
			straddled++
			if !strings.Contains(dirS8, "…”") {
				t.Errorf("pad %d: đoạn đầu của người toàn quyền phải bị cắt ở 80 ký tự: %s", pad, dirS8)
			}
		}
	}
	if straddled < 8 {
		t.Fatalf("tiền đề: bộ quét chỉ có %d chỗ cắt rơi vào giữa tên, muốn ≥ 8", straddled)
	}
}
