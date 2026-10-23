// Package tenant là Tenant Runtime: nạp pack của một tenant, kiểm TOÀN BỘ lúc khởi động
// (cấu hình hỏng thì không chạy), và đóng dấu phiên bản pack vào mọi lượt.
//
// Pack là nơi duy nhất chứa khác biệt giữa tenant (docs/01-architecture.md §8). Mỗi
// phần của pack do package sở hữu hành vi của nó định nghĩa kiểu và cách kiểm — package
// này chỉ ghép chúng lại.
package tenant

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/intent"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/router"
	"go.yaml.in/yaml/v3"
)

// metricsFile là metrics.yaml.
type metricsFile struct {
	Metrics []sourceoftruth.MetricDef  `yaml:"metrics"`
	Derived []sourceoftruth.DerivedDef `yaml:"derived"`
}

// intentsFile là intents.yaml: cấu hình router + danh sách ý định.
type intentsFile struct {
	UnmatchedRoute    string `yaml:"unmatched_route"`
	UnmatchedTemplate string `yaml:"unmatched_template"`
	// ConfirmMarkers: cụm biến câu hỏi số liệu thành câu XÁC NHẬN một con số người dùng đưa ra
	// ("4k đúng không?"). Có khai thì phải có mẫu confirmTemplates.
	ConfirmMarkers []string `yaml:"confirm_markers"`
	// ClaimNeutralWords: từ KHÔNG mang nghĩa claim (lịch sự, trợ từ, trợ động từ) mà câu xác nhận được phép chứa mà vẫn phán quyết
	// (ADR-0009). Đây là danh sách CHO PHÉP: thiếu một từ chỉ làm câu mất phán quyết (an toàn), không bao giờ làm phán quyết sai.
	ClaimNeutralWords []string `yaml:"claim_neutral_words"`
	// IgnorePhrases, DefaultTerms: chính sách phân loại (xem intent.Options).
	IgnorePhrases []string `yaml:"ignore_phrases"`
	DefaultTerms  []string `yaml:"default_terms"`
	// QuantityMarkers: cụm cho biết câu hỏi đòi một LƯỢNG ("bao nhiêu"). Câu hỏi lượng nghiệp vụ
	// mà chỉ có tài liệu trả lời thì báo thiếu bằng chứng — tài liệu không phải nguồn của số liệu.
	QuantityMarkers []string `yaml:"quantity_markers"`
	// ComparisonMarkers: cụm so sánh hai nghĩa số liệu (xem intent.Options). Có khai thì phải có
	// mẫu compare_total.
	ComparisonMarkers []string `yaml:"comparison_markers"`
	// DismissMarkers: cụm cho biết mệnh đề bị bỏ qua (xem intent.Options.DismissMarkers).
	DismissMarkers []string `yaml:"dismiss_markers"`
	// PurposeMarkers: từ nối mở đoạn mục đích (xem intent.Options.PurposeMarkers). NegationMarkers: từ phủ định (xem intent.Options.NegationMarkers).
	PurposeMarkers  []string `yaml:"purpose_markers"`
	NegationMarkers []string `yaml:"negation_markers"`
	// ConditionMarkers: từ nối mở mệnh đề điều kiện (xem intent.Options.ConditionMarkers).
	ConditionMarkers []string `yaml:"condition_markers"`
	// SourceVerbs: động từ của nguồn (xem intent.Options.SourceVerbs).
	SourceVerbs []string     `yaml:"source_verbs"`
	Intents     []intent.Def `yaml:"intents"`
	// UnsupportedMeanings: các nghĩa hệ chưa hỗ trợ (tiền, dự báo, báo giá, cảnh báo, lệnh ghi, đánh giá): câu mang nghĩa đó được hỏi lại, nói rõ nghĩa nào
	// chưa hỗ trợ, thay vì trả theo chỉ số còn lại. Có khai thì phải có mẫu unsupported_meaning và mẫu riêng của từng nghĩa.
	UnsupportedMeanings []intent.Unsupported `yaml:"unsupported_meanings"`
}

// confirmTemplates là các mẫu câu cần có khi pack khai confirm_markers.
var confirmTemplates = []string{"confirm_yes", "confirm_no", "others_note", "approx_mark", "confirm_withheld"}

// knowledgeTemplates là các mẫu câu cần có khi pack có ý định route RAG (tài liệu không trả lời
// được: câu hỏi thuần tài liệu, và phần quy trình của câu HYBRID).
var knowledgeTemplates = []string{"no_knowledge", "hybrid_no_knowledge"}

type templatesFile struct {
	Templates map[string]string `yaml:"templates"`
}

// requiredTemplates là các mẫu câu luồng hỏi luôn có thể cần — thiếu một cái là một
// nhánh của luồng không có lời để nói.
var requiredTemplates = []string{
	"clarify_intent", "clarify_missing", "forbidden", "no_data", "needs_review",
	"fallback_unverified", "approx_note", "conflict_note",
}

// AllowedPlaceholders là các biến mẫu câu được phép dùng.
var AllowedPlaceholders = map[string]bool{
	"metric_label": true, "entity_name": true, "entity_key": true, "period": true,
	"value": true, "unit": true, "approx_note": true, "conflict_note": true, "assumption_note": true,
	"missing": true, "candidates": true, "options": true, "reason": true, "unsupported_note": true,
	"claimed": true, "others": true, "others_note": true,
	"conflicts": true, "conflict_more": true, "count": true,
	"title": true, "procedure": true,
	"left_label": true, "right_label": true, "left_value": true, "right_value": true,
}

// Meta là phần định danh của pack (pack.yaml).
type Meta struct {
	ID       string `yaml:"id"`       // mã pack, ví dụ "construction"
	Tenant   string `yaml:"tenant"`   // slug tenant, ví dụ "abc-infra"
	Version  string `yaml:"version"`  // phiên bản do người viết pack đặt
	Language string `yaml:"language"` // ngôn ngữ trả lời, ví dụ "vi"
	Timezone string `yaml:"timezone"` // múi giờ IANA: "hôm nay" là ngày của tenant
}

// Runtime là pack đã kiểm và biên dịch của một tenant. Dùng chung cho mọi lượt, chỉ đọc.
type Runtime struct {
	Meta Meta
	// PackVersion là băm nội dung mọi tệp YAML của pack — thứ thật sự quyết định hành vi.
	// Trường Version do người đặt có thể quên tăng; băm thì không.
	PackVersion string
	Location    *time.Location
	Normalizer  *normalization.Engine
	Metrics     *sourceoftruth.Catalog
	Intents     *intent.Set
	Router      router.Settings
	Policy      router.Policy
	Templates   map[string]string
	// ConfirmMarkers: cụm biến câu hỏi số liệu thành câu xác nhận ("đúng không?").
	ConfirmMarkers []string
	// ClaimNeutral: từ trung tính mà câu xác nhận được chứa mà vẫn phán quyết (ADR-0009).
	ClaimNeutral []string
	// QuantityMarkers: cụm cho biết câu hỏi đòi một lượng ("bao nhiêu").
	QuantityMarkers []string
}

// Now trả thời điểm hiện tại theo múi giờ của tenant.
func (r *Runtime) Now() time.Time { return time.Now().In(r.Location) }

// Load nạp và kiểm pack trong thư mục dir. Trả lỗi gom đủ thông tin để sửa pack.
func Load(dir string) (*Runtime, error) {
	var meta Meta
	if err := decodeStrict(filepath.Join(dir, "pack.yaml"), &meta); err != nil {
		return nil, err
	}
	var problems []string
	if meta.ID == "" || meta.Tenant == "" || meta.Version == "" {
		problems = append(problems, "pack.yaml thiếu id/tenant/version")
	}
	if meta.Language == "" {
		problems = append(problems, "pack.yaml thiếu language")
	}
	loc, err := time.LoadLocation(meta.Timezone)
	if meta.Timezone == "" || err != nil {
		problems = append(problems, fmt.Sprintf("pack.yaml: timezone %q không hợp lệ", meta.Timezone))
	}

	var lex normalization.Lexicon
	if err := decodeStrict(filepath.Join(dir, "lexicon.yaml"), &lex); err != nil {
		problems = append(problems, err.Error())
	}
	engine, err := normalization.Compile(lex)
	if err != nil {
		problems = append(problems, "lexicon.yaml: "+err.Error())
	}

	rt := &Runtime{Meta: meta, Location: loc, Normalizer: engine}
	if engine != nil {
		problems = append(problems, loadQuerySections(dir, rt)...)
	}

	version, err := hashPack(dir)
	if err != nil {
		problems = append(problems, err.Error())
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("pack %s không hợp lệ:\n  - %s", dir, strings.Join(problems, "\n  - "))
	}
	rt.PackVersion = version
	return rt, nil
}

// loadQuerySections nạp các phần phục vụ luồng hỏi: chỉ số, ý định, mẫu câu, chính sách.
func loadQuerySections(dir string, rt *Runtime) []string {
	var problems []string
	unitKnown := func(u string) bool { _, ok := rt.Normalizer.Unit(u); return ok }
	termKnown := func(t string) bool { _, ok := rt.Normalizer.Term(t); return ok }

	var mf metricsFile
	if err := decodeStrict(filepath.Join(dir, "metrics.yaml"), &mf); err != nil {
		problems = append(problems, err.Error())
	} else if cat, err := sourceoftruth.Compile(mf.Metrics, mf.Derived, unitKnown); err != nil {
		problems = append(problems, "metrics.yaml: "+err.Error())
	} else {
		rt.Metrics = cat
	}
	metricKnown := func(m string) bool {
		if rt.Metrics == nil {
			return false
		}
		_, ok1 := rt.Metrics.Metric(m)
		_, ok2 := rt.Metrics.Derived(m)
		return ok1 || ok2
	}

	var inf intentsFile
	if err := decodeStrict(filepath.Join(dir, "intents.yaml"), &inf); err != nil {
		problems = append(problems, err.Error())
	} else {
		if set, err := intent.Compile(inf.Intents, termKnown, metricKnown); err != nil {
			problems = append(problems, "intents.yaml: "+err.Error())
		} else {
			for _, t := range inf.DefaultTerms {
				if !termKnown(t) {
					problems = append(problems, fmt.Sprintf("intents.yaml: default_terms %q chưa khai báo trong lexicon", t))
				}
			}
			for _, u := range inf.UnsupportedMeanings {
				if u.Name == "" || u.Template == "" || len(u.Cues) == 0 {
					problems = append(problems, fmt.Sprintf("intents.yaml: unsupported_meanings %q cần name, template và ít nhất một cụm (cues)", u.Name))
				}
				for _, sig := range u.Signals {
					if err := intent.ValidSignal(sig, func(dim string) bool { return len(rt.Normalizer.UnitsOf(dim)) > 0 }); err != nil {
						problems = append(problems, fmt.Sprintf("intents.yaml: unsupported_meanings %q: %v", u.Name, err))
					}
				}
			}
			set.SetOptions(intent.Options{
				Unsupported: inf.UnsupportedMeanings,
				// dấu xác nhận cũng là cụm bỏ qua: "đúng ko" gõ không dấu không phải "dùng"
				IgnorePhrases:     append(append([]string(nil), inf.IgnorePhrases...), inf.ConfirmMarkers...),
				DefaultTerms:      inf.DefaultTerms,
				FocusMarkers:      inf.ConfirmMarkers,
				QuantityMarkers:   inf.QuantityMarkers,
				ComparisonMarkers: inf.ComparisonMarkers,
				DismissMarkers:    inf.DismissMarkers,
				PurposeMarkers:    inf.PurposeMarkers,
				NegationMarkers:   inf.NegationMarkers,
				ConditionMarkers:  inf.ConditionMarkers,
				SourceVerbs:       inf.SourceVerbs,
				// tên nguồn cấp số liệu (nhãn + bí danh ở metrics.yaml) của một chỉ số; metric rỗng = mọi chỉ số
				SourceNames: func(metric string) []string {
					if rt.Metrics == nil {
						return nil
					}
					ids := rt.Metrics.MetricIDs()
					if metric != "" {
						ids = []string{metric}
					}
					var out []string
					for _, id := range ids {
						if m, ok := rt.Metrics.Metric(id); ok {
							for _, src := range m.Sources {
								if src.Label != "" {
									out = append(out, src.Label)
								}
								out = append(out, src.Aliases...)
							}
						}
					}
					return out
				},
				NeutralWords: inf.ClaimNeutralWords,
				// lượng đổi được sang đơn vị của chỉ số (đọc danh mục chỉ số lúc phân loại; danh mục nạp sau ý định)
				QuantityFits: func(metric string, v contracts.Value) bool {
					def, ok := rt.Metrics.Metric(metric)
					if !ok {
						return false
					}
					if v.Unit == def.Unit {
						return true
					}
					_, err := rt.Normalizer.Convert(v, def.Unit)
					return err == nil
				},
				// số nằm trong miền một lần đo của chỉ số (số trần coi như đã ở đơn vị của chỉ số): tiền nói tắt "4 củ", "4,1 triệu" không còn bị hiểu là lít
				ValueInRange: func(metric string, v contracts.Value) bool {
					def, ok := rt.Metrics.Metric(metric)
					if !ok {
						return false
					}
					if v.Kind == contracts.ValueQuantity && v.Unit != def.Unit {
						c, err := rt.Normalizer.Convert(v, def.Unit)
						if err != nil {
							return false
						}
						v = c
					}
					return def.InRange(v.Number) == nil
				},
				DerivedOperands: func(id string) (string, string, bool) {
					d, ok := rt.Metrics.Derived(id)
					return d.Left, d.Right, ok
				},
				UnitAliases: func(dim string) []string {
					var out []string
					money := false
					for _, u := range rt.Normalizer.UnitsOf(dim) {
						out = append(out, u.Aliases...)
						money = money || u.Kind == "money"
					}
					if money { // "bao nhiêu củ": hậu tố nhân cỡ triệu trở lên là cách nói số tiền
						out = append(out, rt.Normalizer.MultiplierAliases("1000000")...)
					}
					return out
				},
				Measurable: func(t string) bool {
					d, ok := rt.Normalizer.Term(t)
					return ok && d.DefaultUnit != ""
				},
			})
			rt.Intents = set
		}
		if !contracts.RouteKind(inf.UnmatchedRoute).Valid() {
			problems = append(problems, fmt.Sprintf("intents.yaml: unmatched_route %q không thuộc 12 route", inf.UnmatchedRoute))
		}
		rt.Router = router.Settings{UnmatchedRoute: inf.UnmatchedRoute, UnmatchedTemplate: inf.UnmatchedTemplate}
		rt.ConfirmMarkers = inf.ConfirmMarkers
		rt.ClaimNeutral = inf.ClaimNeutralWords
		rt.QuantityMarkers = inf.QuantityMarkers
	}

	if err := decodeStrict(filepath.Join(dir, "policy.yaml"), &rt.Policy); err != nil {
		problems = append(problems, err.Error())
	} else if err := rt.Policy.Validate(); err != nil {
		problems = append(problems, "policy.yaml: "+err.Error())
	}

	var tf templatesFile
	if err := decodeStrict(filepath.Join(dir, "templates.yaml"), &tf); err != nil {
		problems = append(problems, err.Error())
		return problems
	}
	rt.Templates = tf.Templates
	needed := append([]string{}, requiredTemplates...)
	if inf.UnmatchedTemplate != "" {
		needed = append(needed, inf.UnmatchedTemplate)
	}
	if len(inf.ConfirmMarkers) > 0 {
		needed = append(needed, confirmTemplates...)
	}
	rag := false
	for _, d := range inf.Intents {
		if d.Template != "" {
			needed = append(needed, d.Template)
		}
		rag = rag || contracts.RouteKind(d.Route) == contracts.RouteRAG
	}
	if rag {
		needed = append(needed, knowledgeTemplates...)
	}
	if len(inf.ComparisonMarkers) > 0 {
		needed = append(needed, "compare_total")
	}
	if len(inf.UnsupportedMeanings) > 0 {
		needed = append(needed, "unsupported_meaning")
		for _, u := range inf.UnsupportedMeanings {
			needed = append(needed, u.Template)
		}
	}
	for _, name := range needed {
		if _, ok := tf.Templates[name]; !ok {
			problems = append(problems, "templates.yaml: thiếu mẫu câu "+name)
		}
	}
	// Ý định khai câu giả định thì mọi mẫu câu trả lời số liệu của nó phải có chỗ cho câu đó: đổi mẫu mà quên {assumption_note} sẽ nuốt
	// nhãn giả định im lặng (D-OP-15 điều kiện (i)).
	for _, d := range inf.Intents {
		if d.Assumption == "" {
			continue
		}
		for _, name := range []string{d.Template, "no_data", "needs_review"} {
			if text, ok := tf.Templates[name]; ok && !strings.Contains(text, "{assumption_note}") {
				problems = append(problems, fmt.Sprintf("templates.yaml: ý định %s có assumption nhưng mẫu %s thiếu {assumption_note}", d.Name, name))
			}
		}
	}
	for name, text := range tf.Templates {
		for _, p := range placeholderNames(text) {
			if !AllowedPlaceholders[p] {
				problems = append(problems, fmt.Sprintf("templates.yaml: mẫu %s dùng biến lạ {%s}", name, p))
			}
		}
	}
	return problems
}

// placeholderNames trả tên các biến {x} trong một mẫu câu.
func placeholderNames(text string) []string {
	var out []string
	for {
		i := strings.Index(text, "{")
		if i < 0 {
			return out
		}
		j := strings.Index(text[i:], "}")
		if j < 0 {
			return out
		}
		out = append(out, text[i+1:i+j])
		text = text[i+j+1:]
	}
}

// decodeStrict đọc một tệp YAML và từ chối khoá lạ: gõ nhầm "default_unti" phải đỏ lúc
// khởi động, không được âm thầm thành giá trị rỗng.
func decodeStrict(path string, out any) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	defer f.Close()
	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil {
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("%s: tệp rỗng", path)
		}
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// hashPack băm mọi tệp .yaml trong thư mục theo thứ tự tên, kèm tên tệp.
func hashPack(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	h := sha256.New()
	for _, n := range names {
		b, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			return "", err
		}
		h.Write([]byte(n))
		h.Write([]byte{0})
		h.Write(b)
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:16], nil
}
