package tenant

import (
	"strings"
	"testing"
)

// Pack chỉ-tài-liệu (không khai chỉ số nào) không phải khai mẫu câu của luồng SỐ LIỆU (no_data, needs_review, approx_note,
// conflict_note): không nhánh nào của nó nói các câu đó (lát L6 báo, F-029). Pack có chỉ số thì vẫn bắt buộc đủ.
func TestLoadPackChiTaiLieuKhongCanMauSoLieu(t *testing.T) {
	tmpl := "templates:\n  out_of_scope: \"Ngoài phạm vi.\"\n  clarify_intent: \"Bạn muốn hỏi: {options}?\"\n  clarify_missing: \"Thiếu {missing}.\"\n" +
		"  forbidden: \"Không có quyền.\"\n  fallback_unverified: \"Chưa kiểm được.\"\n  knowledge_answer: \" Theo {title}: {procedure}\"\n" +
		"  no_knowledge: \"Chưa có tài liệu.\"\n  hybrid_no_knowledge: \" Chưa có tài liệu cho phần quy trình.\"\n" + mauPhienBan
	docs := map[string]string{
		"metrics.yaml":   "metrics: []\n",
		"intents.yaml":   "unmatched_route: REJECT\nunmatched_template: out_of_scope\nintents:\n" + ragIntent,
		"templates.yaml": tmpl,
	}
	if _, err := Load(writePack(t, fullPack(docs))); err != nil {
		t.Fatalf("pack chỉ-tài-liệu không khai mẫu câu số liệu phải nạp được: %v", err)
	}
	docs["metrics.yaml"] = validMetrics
	if _, err := Load(writePack(t, fullPack(docs))); err == nil || !strings.Contains(err.Error(), "no_data") {
		t.Errorf("pack có chỉ số mà thiếu mẫu no_data: Load phải lỗi nêu tên mẫu, được %v", err)
	}
}
