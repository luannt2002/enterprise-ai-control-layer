package tenant

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validMeta = `id: test
tenant: test-tenant
version: 0.1.0
language: vi
timezone: Asia/Ho_Chi_Minh
`

const validLexicon = `units:
  - unit: L
    kind: quantity
    dimension: volume
    aliases: [L, lít]
terms:
  - term: fuel
    label: dầu
    aliases: [dầu]
    default_unit: L
`

const validMetrics = `metrics:
  - id: reported_fuel
    label: dầu báo cáo
    unit: L
    sources:
      - {type: chat_message, authority: 1}
`

const validIntents = `unmatched_route: REJECT
unmatched_template: out_of_scope
intents:
  - name: fuel_total
    route: SQL_QUERY
    keywords: [bao nhiêu]
    terms: [fuel]
    metric_by_term: {fuel: reported_fuel}
    template: metric_total
    capability: metrics.read
`

const validPolicy = `roles:
  director: {capabilities: [metrics.read], scope: all}
`

const validTemplates = `templates:
  metric_total: "{metric_label} {entity_name} {period}: {value} {unit}{approx_note}{conflict_note}"
  out_of_scope: "Ngoài phạm vi."
  clarify_intent: "Bạn muốn hỏi: {options}?"
  clarify_missing: "Thiếu {missing}. {candidates}"
  forbidden: "Không có quyền."
  no_data: "Chưa có dữ liệu."
  needs_review: "Cần người duyệt: {reason}"
  fallback_unverified: "Chưa kiểm được số liệu."
  approx_note: " (có ước lượng)"
  conflict_note: " (có nguồn lệch)"
`

// fullPack trả bộ tệp pack hợp lệ đầy đủ; ghi đè từng tệp để dựng ca hỏng.
func fullPack(over map[string]string) map[string]string {
	files := map[string]string{
		"pack.yaml": validMeta, "lexicon.yaml": validLexicon, "metrics.yaml": validMetrics,
		"intents.yaml": validIntents, "policy.yaml": validPolicy, "templates.yaml": validTemplates,
	}
	for k, v := range over {
		if v == "<xoá>" {
			delete(files, k)
			continue
		}
		files[k] = v
	}
	return files
}

func writePack(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestLoadPackHopLe(t *testing.T) {
	dir := writePack(t, fullPack(nil))
	rt, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if rt.Meta.Tenant != "test-tenant" || rt.Location.String() != "Asia/Ho_Chi_Minh" || len(rt.PackVersion) != 16 {
		t.Fatalf("runtime = %+v", rt)
	}
	got := rt.Normalizer.Normalize("dầu 4000", normalizationOptions(rt))
	if got.Canonical != "dầu 4000 L" {
		t.Fatalf("pack nạp xong phải chuẩn hoá được: %q", got.Canonical)
	}
}

func TestPackVersionDoiKhiNoiDungDoi(t *testing.T) {
	a := writePack(t, fullPack(nil))
	b := writePack(t, fullPack(nil))
	c := writePack(t, fullPack(map[string]string{"lexicon.yaml": strings.Replace(validLexicon, "lít", "lit", 1)}))
	ra, errA := Load(a)
	rb, errB := Load(b)
	rc, errC := Load(c)
	if errA != nil || errB != nil || errC != nil {
		t.Fatal(errA, errB, errC)
	}
	if ra.PackVersion != rb.PackVersion {
		t.Fatal("cùng nội dung phải cùng phiên bản")
	}
	if ra.PackVersion == rc.PackVersion {
		t.Fatal("nội dung đổi một chữ thì phiên bản phải đổi — dù người viết quên tăng version")
	}
}

func TestLoadChanPackHong(t *testing.T) {
	cases := map[string]map[string]string{
		"khoá lạ (gõ nhầm)":                           {"lexicon.yaml": strings.Replace(validLexicon, "default_unit", "default_unti", 1)},
		"thiếu lexicon":                               {"lexicon.yaml": "<xoá>"},
		"lexicon rỗng":                                {"lexicon.yaml": ""},
		"múi giờ sai":                                 {"pack.yaml": strings.Replace(validMeta, "Asia/Ho_Chi_Minh", "Asia/Nowhere", 1)},
		"thiếu tenant":                                {"pack.yaml": strings.Replace(validMeta, "tenant: test-tenant\n", "", 1)},
		"đơn vị mặc định chưa khai báo":               {"lexicon.yaml": strings.Replace(validLexicon, "default_unit: L", "default_unit: m3", 1)},
		"ý định trỏ chỉ số chưa có":                   {"intents.yaml": strings.Replace(validIntents, "fuel: reported_fuel", "fuel: issued_fuel", 1)},
		"route lạ":                                    {"intents.yaml": strings.Replace(validIntents, "route: SQL_QUERY", "route: MAGIC", 1)},
		"ý định dữ liệu quên capability":              {"intents.yaml": strings.Replace(validIntents, "    capability: metrics.read\n", "", 1)},
		"thiếu mẫu câu bắt buộc":                      {"templates.yaml": strings.Replace(validTemplates, "  no_data: \"Chưa có dữ liệu.\"\n", "", 1)},
		"mẫu câu dùng biến lạ":                        {"templates.yaml": strings.Replace(validTemplates, "{value}", "{valeu}", 1)},
		"vai mở toang không khai scope":               {"policy.yaml": "roles:\n  director: {capabilities: [metrics.read]}\n"},
		"thiếu policy":                                {"policy.yaml": "<xoá>"},
		"khai cụm so sánh mà thiếu mẫu compare_total": {"intents.yaml": validIntents + "comparison_markers: [nhiều hơn]\n"},
		// ý định có câu giả định mà mẫu câu trả lời không có chỗ cho nó: nhãn bị nuốt im lặng (D-OP-15 (i))
		"ý định có assumption, mẫu câu thiếu {assumption_note}": {"intents.yaml": assumedIntents},
		"ý định RAG thiếu mẫu câu khi tài liệu không trả lời được": {
			"intents.yaml": validIntents + ragIntent, "templates.yaml": validTemplates + "  knowledge_answer: \" Theo {title}: {procedure}\"\n"},
	}
	for name, over := range cases {
		dir := writePack(t, fullPack(over))
		if _, err := Load(dir); err == nil {
			t.Errorf("%s: Load phải lỗi", name)
		}
	}
}

// assumedIntents là validIntents với câu giả định cho ý định chung.
var assumedIntents = strings.Replace(validIntents, "    capability: metrics.read\n", "    capability: metrics.read\n    assumption: \"Hiểu là dầu báo cáo.\"\n", 1)

// Ý định có câu giả định: đủ {assumption_note} ở mẫu câu của nó, no_data và needs_review thì nạp được.
func TestLoadPackCauGiaDinh(t *testing.T) {
	tmpl := strings.NewReplacer("{conflict_note}\"", "{conflict_note}{assumption_note}\"", "\"Chưa có dữ liệu.\"", "\"Chưa có dữ liệu.{assumption_note}\"",
		"\"Cần người duyệt: {reason}\"", "\"Cần người duyệt: {reason}{assumption_note}\"").Replace(validTemplates)
	if _, err := Load(writePack(t, fullPack(map[string]string{"intents.yaml": assumedIntents, "templates.yaml": tmpl}))); err != nil {
		t.Fatalf("pack có câu giả định và đủ chỗ trong mẫu câu phải nạp được: %v", err)
	}
	for _, name := range []string{"metric_total", "no_data", "needs_review"} {
		broken := strings.Replace(tmpl, name+": ", name+"_x: ", 1) // mẫu đó quay về bản không có {assumption_note}
		broken += "  " + name + ": " + map[string]string{"metric_total": "\"{value}\"", "no_data": "\"Chưa có.\"", "needs_review": "\"Duyệt.\""}[name] + "\n"
		if _, err := Load(writePack(t, fullPack(map[string]string{"intents.yaml": assumedIntents, "templates.yaml": broken}))); err == nil {
			t.Errorf("mẫu %s thiếu {assumption_note}: Load phải lỗi", name)
		}
	}
}

const ragIntent = `  - name: sop_question
    route: RAG
    keywords: [quy trình]
    template: knowledge_answer
    capability: documents.read
`

// mauPhienBan là 5 mẫu câu cho thông báo phiên bản tài liệu (docs/viec/2-tra-quy-trinh/HOP-DONG.md §4).
const mauPhienBan = "  knowledge_version_conflict: \"Tài liệu {title} có nhiều bản cùng hiệu lực ({reason}).\"\n" +
	"  knowledge_no_effective: \"Tài liệu {title} chưa có bản hiệu lực ({reason}).\"\n" +
	"  hybrid_knowledge_version_conflict: \" Phần quy trình: {title} có nhiều bản cùng hiệu lực.\"\n" +
	"  hybrid_knowledge_no_effective: \" Phần quy trình: {title} chưa có bản hiệu lực.\"\n" +
	"  hybrid_knowledge_error: \" Phần quy trình tạm thời chưa tra được.\"\n"

// Pack có ý định RAG: đủ mẫu câu tri thức thì nạp được, và cụm hỏi lượng được nạp vào runtime.
func TestLoadPackTriThuc(t *testing.T) {
	dir := writePack(t, fullPack(map[string]string{
		"intents.yaml": validIntents + ragIntent + "quantity_markers: [bao nhiêu]\n",
		"templates.yaml": validTemplates + "  knowledge_answer: \" Theo {title}: {procedure}\"\n" +
			"  no_knowledge: \"Chưa có tài liệu.\"\n  hybrid_no_knowledge: \" Chưa có tài liệu cho phần quy trình.\"\n" + mauPhienBan,
	}))
	rt, err := Load(dir)
	if err != nil {
		t.Fatalf("pack tri thức hợp lệ phải nạp được: %v", err)
	}
	// Mẫu câu cho tài liệu có phiên bản (nhánh RAG U7a) là bắt buộc: thiếu thì câu trả lời rỗng mà không ai biết.
	thieu := writePack(t, fullPack(map[string]string{
		"intents.yaml": validIntents + ragIntent,
		"templates.yaml": validTemplates + "  knowledge_answer: \" Theo {title}: {procedure}\"\n" +
			"  no_knowledge: \"Chưa có tài liệu.\"\n  hybrid_no_knowledge: \" Chưa có.\"\n" + strings.Replace(mauPhienBan, "  knowledge_version_conflict:", "  x_knowledge_version_conflict:", 1),
	}))
	if _, err := Load(thieu); err == nil || !strings.Contains(err.Error(), "knowledge_version_conflict") {
		t.Errorf("thiếu mẫu knowledge_version_conflict: Load phải lỗi nêu tên mẫu, được %v", err)
	}
	if len(rt.QuantityMarkers) != 1 || rt.QuantityMarkers[0] != "bao nhiêu" {
		t.Errorf("QuantityMarkers = %v, muốn [bao nhiêu]", rt.QuantityMarkers)
	}
}
