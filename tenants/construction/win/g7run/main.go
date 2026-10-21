// g7run chạy cổng G7 VALUE của WIN MVP (docs/04-core-mvp.md §6; luật D5 sửa đổi ở
// docs/adr/0005-g7-hop-dong-va-sua-d5.md): CÙNG dữ liệu tenant A và CÙNG bộ ca, chạy EAIS và các
// baseline mỗi hệ k lượt bằng CÙNG một model, rồi so EAIS với từng baseline. Chấm rò rỉ canary và
// claim không căn cứ bằng bộ đếm độc lập (evals.G7Counter), không bằng bộ chuẩn hoá của EAIS.
//
//	go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out /tmp/win -profile functional -seed 42
//	go run ./tenants/construction/win/g7run -data /tmp/win -cases <bộ ca> -manifest <hồ sơ ca> -out <thư mục bằng chứng>
//
// Hệ (-systems, ADR-0005 §2): baseline cổng B2+ ("b2plus") và B1 ("b1", cần -manifest có b1_window
// từng ca); chỉ báo cáo B2 ("b2") và B2″ ("b2pp"). G7 chỉ ĐẠT khi đạt với CẢ hai baseline cổng.
//
// Không có model thật thì G7 là CHẶN (docs/04 §6): provider rỗng hoặc "extractive" vẫn chạy hết
// ống nối bằng model giả tất định để thấy mọi mắt xích nối đúng, nhưng kết quả KHÔNG phải bằng chứng.
// Model thật cấu hình bằng biến môi trường (models.ConfigFromEnv: EAIS_MODEL_PROVIDER, EAIS_MODEL,
// EAIS_MODEL_API_KEY, EAIS_MODEL_BASE_URL, EAIS_MODEL_THINKING, EAIS_MODEL_EFFORT…); cờ -provider ghi
// đè EAIS_MODEL_PROVIDER. Thiếu model hay khoá là lỗi cấu hình, thoát ngay. Mọi hệ dùng CHUNG một
// gateway; không gửi temperature (ADR-0005 §1). -smoke N chạy N ca đầu một lượt mỗi hệ để kiểm
// provider nhận tham số, trước khi chi tiền cho lượt chạy thật.
//
// Cách chấm từng ca (-grade): mặc định "content" chấm lại câu trả lời của MỌI hệ chỉ theo NỘI DUNG
// (evals.G7ContentReport). "full" dùng nguyên kết quả RunWin, tức chấm cả route / outcome / model_calls /
// ứng viên / conflict / trích dẫn — những nhãn mà baseline không có nên trượt theo thiết kế, làm D5
// không bao giờ đỏ; chỉ dùng để đối chiếu.
//
// Mã thoát (soát C2, F-07):
//   - 0 = ĐẠT: model thật qua endpoint chính thức, bộ chấm đã đóng băng, chấm nội dung, tham số đúng hợp đồng §4;
//     với -smoke: mọi lời gọi tới endpoint chính thức trả 200;
//   - 1 = KHÔNG ĐẠT: luật trượt khi đủ mọi điều kiện trên; với -smoke: có câu lỗi / HTTP khác 200 / hệ cổng bị chặn
//     (kể cả chạy khô — trượt là trượt);
//   - 2 = CHẶN hoặc lỗi: model giả, chạy khô mà đạt, bộ chấm chưa đóng băng, -grade full, tham số lệch hợp đồng, lượt
//     thật có hệ cổng bị chặn (dừng trước lời gọi model), cấu hình / tệp / thư mục -out sai.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/ingestion"
	"github.com/luannt2002/enterprise-ai-control-layer/evals"
	"github.com/luannt2002/enterprise-ai-control-layer/evals/baseline"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
	"go.yaml.in/yaml/v3"
	"golang.org/x/text/unicode/norm"
)

// blockedLine là dòng đầu khi chạy bằng model giả: G7 vẫn CHẶN, kết quả chỉ để thử ống nối.
const blockedLine = "G7: CHẶN — chưa có model thật; đây là chạy thử ống nối bằng model giả tất định, KHÔNG phải bằng chứng G7"

// scorerFrozen: bộ chấm nội dung đã đóng băng theo hợp đồng chưa. ADR-0007: v1 đã bị phá, v2 còn ĐỀ XUẤT ⇒ false.
// CHỈ bật khi bộ chấm v2 đủ điều kiện đóng băng của ADR-0007 (hiện thực, hiệu chỉnh trên pilot thật, ngưỡng ngoài mẫu).
const scorerFrozen = false

// Model và tham số duy nhất hợp đồng G7 §4 cho ra kết luận (D-G7-2).
const (
	contractModel    = "claude-sonnet-5-5"
	contractThinking = "between_tools"
	contractEffort   = "medium"
	// contractBudgetUSD là trần chi tiêu của TOÀN BỘ G7 (hợp đồng §9, D-G7-3).
	contractBudgetUSD = 15.0
	// contractMaxTokens, contractMaxRetries: hợp đồng §4 (max_tokens 1024; thử lại 2 lần).
	contractMaxTokens  = 1024
	contractMaxRetries = 2
	// contractSmoke là cỡ thử khói bắt buộc của hợp đồng ("g7run -smoke 3"); COST-PLAN dự phòng đúng 3 ca.
	contractSmoke = 3
)

// Dòng kết luận khi luật D5 có thể đã tính xong nhưng KHÔNG được rút kết luận G7.
const (
	unfrozenLine  = "G7: CHẶN — bộ chấm nội dung chưa đóng băng (ADR-0007; bộ chấm v1 đã bị phá); KHÔNG phải bằng chứng G7"
	fullGradeLine = "G7: CHẶN — -grade full chấm cả nhãn kiến trúc của EAIS, chỉ để chẩn đoán; KHÔNG phải tiêu chí G7"
	paramsLine    = "G7: CHẶN — model / tham số khác hợp đồng §4 (" + contractModel + ", thinking " + contractThinking + ", effort " + contractEffort + "); KHÔNG phải bằng chứng G7"
)

// verdictInput gom các điều kiện của kết luận G7.
type verdictInput struct {
	pass, fake, dry, frozen, contentGrade, contractParams bool
}

// verdict trả dòng kết luận cuối và mã thoát (C-06, soát lượt 2). G7 chỉ có thể ĐẠT (mã 0) khi đủ mọi điều kiện;
// thiếu một điều kiện thì CHẶN (mã 2) dù luật D5 tính ra ĐẠT; luật D5 trượt ở cấu hình hợp lệ thì mã 1.
func verdict(rule string, in verdictInput) (string, int) {
	switch {
	case in.fake:
		return "phán quyết theo luật: " + rule + "\n" + blockedLine, 2
	case in.dry:
		return "phán quyết theo luật: " + rule + "\n" + dryLine, 2
	case !in.frozen:
		return "phán quyết theo luật: " + rule + "\n" + unfrozenLine, 2
	case !in.contentGrade:
		return "phán quyết theo luật: " + rule + "\n" + fullGradeLine, 2
	case !in.contractParams:
		return "phán quyết theo luật: " + rule + "\n" + paramsLine, 2
	case in.pass:
		return "G7: " + rule, 0
	}
	return "G7: " + rule, 1
}

// officialEndpoint là endpoint duy nhất hợp đồng G7 công nhận (§4), so theo nghĩa (isOfficialEndpoint). Endpoint khác
// (máy chủ giả, proxy) là CHẠY KHÔ: không phải bằng chứng G7, kể cả khi model và tham số y hệt lượt thật.
const officialEndpoint = "https://api.anthropic.com"

// isOfficialEndpoint so endpoint với officialEndpoint theo NGHĨA, không theo chuỗi: scheme và tên máy không phân biệt hoa
// thường, bỏ dấu chấm cuối tên máy, cổng 443 là cổng mặc định của https, path bất kỳ. Cách viết khác của chính máy chủ
// chính thức vẫn tốn tiền thật nên không được rơi vào nhánh chạy khô (soát C3, C3-04).
func isOfficialEndpoint(raw string) bool {
	u, err := url.Parse(raw)
	o, _ := url.Parse(officialEndpoint)
	if err != nil || u.Scheme != o.Scheme || (u.Port() != "" && u.Port() != "443") {
		return false
	}
	return strings.EqualFold(strings.TrimSuffix(u.Hostname(), "."), o.Hostname())
}

const dryLine = "G7: CHẠY KHÔ — endpoint không phải API chính thức của nhà cung cấp; KHÔNG phải bằng chứng G7"

// quantityUnits là đơn vị lượng nhiên liệu của tenant: số mang đơn vị này trong TÀI LIỆU không được
// coi là có căn cứ (docs/04 §6 G5). Viết tay ở đây, không lấy từ lexicon của pack: bộ đếm độc lập.
var quantityUnits = []string{"L", "lít", "lit", "m3", "m³", "khối"}

// Hai cách chấm từng ca của G7 (xem đầu tệp).
const (
	gradeContent = "content"
	gradeFull    = "full"
)

type config struct {
	data, pack, world, cases, tenant, provider, grade string
	systems, manifest, out, contract                  string
	k, topK, smoke                                    int
	pilot                                             bool
	budget                                            float64
}

func main() {
	var cfg config
	flag.StringVar(&cfg.data, "data", "", "thư mục dữ liệu do bộ sinh xuất ra")
	flag.StringVar(&cfg.pack, "pack", "tenants/construction/pack", "pack của tenant")
	flag.StringVar(&cfg.world, "world", "tenants/construction/win/world.yaml", "world.yaml")
	flag.StringVar(&cfg.cases, "cases", "tenants/construction/win/cases-*.yaml", "các tệp ca WIN")
	flag.StringVar(&cfg.tenant, "tenant", "abc-infra", "tenant được chấm")
	flag.IntVar(&cfg.k, "k", evals.G7MinRuns, "số lượt mỗi hệ")
	flag.IntVar(&cfg.topK, "topk", baseline.DefaultTopK, "số đoạn baseline RAG đưa cho model mỗi câu hỏi")
	flag.StringVar(&cfg.grade, "grade", gradeContent, `cách chấm từng ca: "content" (theo nội dung câu trả lời, mặc định) hoặc "full" (nguyên kết quả RunWin)`)
	flag.StringVar(&cfg.provider, "provider", "", `provider model, ghi đè EAIS_MODEL_PROVIDER; rỗng hoặc "extractive" = model giả tất định, G7 CHẶN`)
	flag.StringVar(&cfg.systems, "systems", sysB2Plus+","+sysB1, `baseline so với EAIS, phân cách dấu phẩy: "b2plus", "b1" (cổng); "b2", "b2pp", "b2oracle" (chỉ báo cáo)`)
	flag.StringVar(&cfg.manifest, "manifest", "", "hồ sơ ca có b1_window của từng ca (bắt buộc cho B1)")
	flag.StringVar(&cfg.out, "out", "", "thư mục ghi bằng chứng thô: manifest.json, raw-runs.jsonl, scored.jsonl, aggregate.json, failures.jsonl, cost.json, ket-qua.md")
	flag.StringVar(&cfg.contract, "contract", "docs/g7/HOP-DONG-G7.md", "hợp đồng G7 — băm vào manifest")
	flag.BoolVar(&cfg.pilot, "pilot", false, "lượt pilot (ADR-0006 §6): cần -k 5; in k của lượt chính theo luật đăng ký trước")
	flag.Float64Var(&cfg.budget, "budget", 15, "trần chi tiêu USD cho lần chạy (hợp đồng G7 §9); chạm trần thì dừng gọi model, lần chạy VÔ HIỆU")
	flag.IntVar(&cfg.smoke, "smoke", 0, "thử khói: chạy N ca đầu MỘT lượt mỗi hệ để kiểm provider nhận tham số, rồi thoát")
	flag.Parse()
	if cfg.data == "" || cfg.k < 1 || cfg.topK < 1 || cfg.smoke < 0 || (cfg.pilot && cfg.k != 5) || (cfg.grade != gradeContent && cfg.grade != gradeFull) {
		fmt.Fprintln(os.Stderr, `thiếu -data, hoặc -k < 1, hoặc -topk < 1, hoặc -smoke < 0, hoặc -pilot mà -k khác 5, hoặc -grade không phải "content" / "full"`)
		os.Exit(2)
	}
	os.Exit(run(cfg))
}

// stopBeforeModel: lượt CÓ THỂ TỐN TIỀN (không model giả, endpoint không phải loopback — kể cả proxy) mà có hệ cổng bị
// chặn thì không gọi model — kết luận chắc chắn là KHÔNG ĐẠT, gọi tiếp chỉ tốn tiền (C-04, soát lượt 2; proxy: soát C3,
// C3-04). free = endpoint loopback (máy chủ giả).
func stopBeforeModel(blocked []string, fake, free bool) bool {
	return len(blocked) > 0 && !fake && !free
}

// contractDeviations liệt kê chỗ cấu hình HIỆU LỰC lệch hợp đồng G7 §3–§4: model, thinking, effort, max_tokens,
// temperature, số lần thử lại, top-k của baseline RAG. Rỗng = đúng hợp đồng. Dùng chung cho kiểm trước lượt thật và
// cho verdict() (soát C2, F-02).
func contractDeviations(cfg config, gw models.Gateway) []string {
	if bg, ok := gw.(*budgetGateway); ok {
		gw = bg.inner
	}
	g, ok := gw.(*models.Messages)
	if !ok {
		return []string{"không phải client model thật"}
	}
	var out []string
	if g.Model != contractModel || g.Thinking != contractThinking || g.Effort != contractEffort {
		out = append(out, fmt.Sprintf("model %q / thinking %q / effort %q, hợp đồng %q / %q / %q", g.Model, g.Thinking, g.Effort, contractModel, contractThinking, contractEffort))
	}
	if g.MaxTokens != contractMaxTokens {
		out = append(out, fmt.Sprintf("max_tokens %d, hợp đồng %d", g.MaxTokens, contractMaxTokens))
	}
	if g.Temperature != nil {
		out = append(out, "có gửi temperature, hợp đồng: không gửi")
	}
	if g.MaxRetries != contractMaxRetries {
		out = append(out, fmt.Sprintf("thử lại %d lần, hợp đồng %d", g.MaxRetries, contractMaxRetries))
	}
	if cfg.topK != baseline.DefaultTopK {
		out = append(out, fmt.Sprintf("top-k %d, hợp đồng %d", cfg.topK, baseline.DefaultTopK))
	}
	return out
}

// preflightReal kiểm lượt chạy THẬT (model thật qua endpoint chính thức) TRƯỚC lời gọi model đầu tiên: điều gì chắc
// chắn làm lượt đó không thành bằng chứng thì từ chối ngay, không tiêu tiền rồi mới in CHẶN (soát C2, F-02, F-03).
// Thử khói chỉ cần đúng tham số (rẻ, để chứng minh nhà cung cấp nhận chúng); pilot được miễn điều kiện bộ chấm đã
// đóng băng (pilot là để hiệu chỉnh bộ chấm, không phải bằng chứng G7).
func preflightReal(cfg config, gw models.Gateway) []string {
	reasons := contractDeviations(cfg, gw)
	if cfg.smoke > 0 {
		// Thử khói rẻ nhưng vẫn tốn tiền: cỡ không vượt hợp đồng, và bằng chứng của nó phải ghi được (soát C3, C3-06).
		if cfg.smoke > contractSmoke {
			reasons = append(reasons, fmt.Sprintf("-smoke %d: hợp đồng thử khói %d ca", cfg.smoke, contractSmoke))
		}
		if cfg.out != "" {
			if err := probeWritable(cfg.out); err != nil {
				reasons = append(reasons, "thư mục -out không ghi được: "+err.Error())
			}
		}
		return reasons
	}
	if cfg.grade != gradeContent {
		reasons = append(reasons, "-grade khác content: không phải tiêu chí G7")
	}
	if !cfg.pilot {
		if !scorerFrozen {
			reasons = append(reasons, "bộ chấm nội dung chưa đóng băng (ADR-0007): lượt chính không thể ra kết luận G7")
		}
		if cfg.k != 3 && cfg.k != 5 {
			reasons = append(reasons, fmt.Sprintf("-k %d: hợp đồng §5 chỉ 3 hoặc 5", cfg.k))
		}
		// gatePass đòi mọi hệ cổng: thiếu một hệ trong -systems thì lượt chắc chắn KHÔNG ĐẠT (mã 1, hợp đồng coi là FAIL)
		// vì quên cờ chứ không vì EAIS (soát C3, C3-07). Pilot được chạy một phần hệ.
		if systems, err := parseSystems(cfg.systems); err == nil {
			for _, g := range gateSystems {
				if !slices.Contains(systems, g) {
					reasons = append(reasons, "lượt chính thiếu hệ cổng "+g+" trong -systems: gatePass đòi mọi hệ cổng nên lượt này chắc chắn KHÔNG ĐẠT")
				}
			}
		}
	}
	if cfg.out == "" {
		reasons = append(reasons, "thiếu -out: lượt thật phải lưu bằng chứng thô (hợp đồng §11)")
	} else if err := probeWritable(cfg.out); err != nil {
		reasons = append(reasons, "thư mục -out không ghi được: "+err.Error())
	}
	return reasons
}

// probeWritable tạo (nếu chưa có) thư mục rồi thử ghi và xoá một tệp tạm trong đó.
func probeWritable(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".kiem-ghi-*")
	if err != nil {
		return err
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		return err
	}
	return os.Remove(name)
}

func run(cfg config) int {
	started := time.Now()
	// -budget ≤ 0 từng tắt trần (C-07): trần là bắt buộc và không vượt trần của hợp đồng. Viết dạng phủ định của
	// điều kiện ĐÚNG để NaN (mọi phép so với NaN đều sai) cũng bị từ chối (soát C2, F-06).
	if !(cfg.budget > 0 && cfg.budget <= contractBudgetUSD) {
		fmt.Fprintf(os.Stderr, "LỖI: -budget %.2f: phải > 0 và ≤ %.0f USD (hợp đồng G7 §9)\n", cfg.budget, contractBudgetUSD)
		return 2
	}
	if cfg.out != "" {
		if err := checkOutDir(cfg.out); err != nil {
			fmt.Fprintln(os.Stderr, "LỖI:", err)
			return 2
		}
	}
	systems, err := parseSystems(cfg.systems)
	if err != nil {
		fmt.Fprintln(os.Stderr, "LỖI:", err)
		return 2
	}
	mc := models.ConfigFromEnv()
	if cfg.provider != "" {
		mc.Provider = cfg.provider
	}
	gw, err := models.New(mc) // thiếu model / khoá / tham số sai cho provider thật: lỗi cấu hình, thoát ngay
	if err != nil {
		fmt.Fprintln(os.Stderr, "LỖI:", err)
		return 2
	}
	_, fake := gw.(*models.Extractive)
	if fake {
		fmt.Println(blockedLine)
	}
	sink := newEvidenceSink(nil)
	dry, costly := false, false
	if m, ok := gw.(*models.Messages); ok {
		m.Trace = sink.trace
		// dry: không phải endpoint chính thức ⇒ không phải bằng chứng G7. costly: không phải loopback ⇒ lời gọi CÓ THỂ tốn
		// tiền (cả cách viết khác của endpoint chính thức lẫn proxy), nên phải giữ được trần ngân sách và qua kiểm trước
		// lời gọi model (soát C3, C3-04).
		dry, costly = !isOfficialEndpoint(m.BaseURL), !models.LoopbackEndpoint(m.BaseURL)
		price, priced := priceTable[m.Model]
		if !priced && costly {
			fmt.Fprintf(os.Stderr, "LỖI: model %q không có trong bảng giá: không giữ được trần ngân sách\n", m.Model)
			return 2
		}
		sink.price, sink.capUSD = price, cfg.budget
		gw = &budgetGateway{inner: m, sink: sink}
	}
	if dry {
		fmt.Println(dryLine)
	}
	if !fake && costly {
		if reasons := preflightReal(cfg, gw); len(reasons) > 0 {
			fmt.Fprintln(os.Stderr, "LỖI: lượt chạy thật bị từ chối TRƯỚC lời gọi model:", strings.Join(reasons, "; "))
			return 2
		}
	}
	s, err := build(cfg, gw, systems)
	if err != nil {
		fmt.Fprintln(os.Stderr, "LỖI:", err)
		return 2
	}
	if ea, ok := s.askers[sysEAIS].(eaisAsker); ok { // bản ghi quyết định của EAIS vào bằng chứng thô (C-11)
		ea.onDecision = sink.decision
		s.askers[sysEAIS] = ea
	}
	// Tệp nào phải băm vào bằng chứng mà không đọc được thì dừng TRƯỚC lời gọi model, không ghi băm rỗng (C-07).
	for _, f := range append(slices.Clone(s.files), filepath.Join(cfg.data, "manifest.json")) {
		if _, err := os.ReadFile(f); f != "" && err != nil {
			fmt.Fprintln(os.Stderr, "LỖI: không đọc được tệp phải băm vào bằng chứng:", err)
			return 2
		}
	}
	if stopBeforeModel(s.blocked, fake, !costly) {
		fmt.Fprintln(os.Stderr, "LỖI: hệ cổng không chạy được, không gọi model:", strings.Join(s.blocked, "; "))
		return 2
	}
	sink.setCases(s.cases)
	ctx := context.Background()
	if cfg.smoke > 0 {
		return smoke(ctx, cfg, mc, gw, s, sink, fake, dry, started)
	}

	full := map[string][]evals.Report{}
	for i := 0; i < cfg.k; i++ {
		for _, sys := range s.order {
			rec, err := newRecorder(s.askers[sys], sys, i+1, sink)
			if err != nil {
				fmt.Fprintln(os.Stderr, "LỖI:", err)
				return 2
			}
			full[sys] = append(full[sys], evals.RunWin(ctx, rec, s.cases, s.principals))
		}
	}
	graded := map[string][]evals.Report{}
	counts := map[string]evals.G7Counts{}
	for _, sys := range s.order {
		for _, r := range full[sys] {
			if cfg.grade == gradeContent {
				r = evals.G7ContentReport(s.cases, r)
			}
			graded[sys] = append(graded[sys], r)
		}
		counts[sys] = s.counter.Count(s.cases, graded[sys])
	}

	var out strings.Builder
	say := func(f string, a ...any) { fmt.Fprintf(&out, f, a...) }
	provider := mc.Provider
	if provider == "" {
		provider = "extractive (mặc định)"
	} else if mc.Model != "" {
		provider += " · model " + mc.Model
	}
	var labels []string
	for _, sys := range s.order {
		labels = append(labels, sysLabel[sys])
	}
	say("tenant %s · %d ca · k = %d · provider %s · hệ: %s · baseline RAG top-%d trên %d đoạn quan sát + %d tài liệu\n",
		cfg.tenant, len(s.cases), cfg.k, provider, strings.Join(labels, ", "), cfg.topK, s.paragraphs, s.documents)
	say("canary suy ra từ world.yaml (chữ của tenant khác, không có ở tenant được chấm): %s\n", strings.Join(s.canaries, " "))
	if cfg.grade == gradeFull {
		say("chấm từng ca: theo TOÀN BỘ kỳ vọng của ca (RunWin) — baseline không có route / outcome / model_calls / ứng viên / conflict / trích dẫn của EAIS nên trượt theo thiết kế\n")
	} else {
		say("chấm từng ca: theo NỘI DUNG câu trả lời, cùng tiêu chí cho mọi hệ (evals.G7ContentReport)\n")
		var ref []string
		for _, sys := range s.order {
			ref = append(ref, fmt.Sprintf("%s %d/%d", sysLabel[sys], passed(full[sys][0]), len(s.cases)))
		}
		say("tham khảo, theo TOÀN BỘ kỳ vọng của ca (RunWin, lượt 1): %s\n", strings.Join(ref, " · "))
	}
	for _, b := range s.blocked {
		say("HỆ CỔNG KHÔNG CHẠY: %s\n", b)
	}
	unstable := unstablePaths(sink.recs)
	if len(unstable) > 0 {
		say("LƯỢT VÔ HIỆU (ADR-0005 §5): đường tất định của EAIS khác nhau giữa các lượt ở %d ca: %s\n", len(unstable), strings.Join(unstable, ", "))
	}
	invalid := invalidRuns(sink.recs)
	if spent, hit := sink.spent(); hit {
		invalid = append(invalid, fmt.Sprintf("chạm trần ngân sách %.2f USD (đã tiêu %.4f USD)", cfg.budget, spent))
	}
	for _, x := range invalid {
		say("LƯỢT VÔ HIỆU (lỗi gọi model): %s\n", x)
	}
	verdicts := map[string]evals.G7Verdict{}
	for _, b := range s.order[1:] {
		v, err := evals.CompareG7(graded[sysEAIS], graded[b], counts[sysEAIS])
		if err != nil {
			fmt.Fprintln(os.Stderr, "LỖI:", err)
			return 2
		}
		verdicts[b] = v
		role := "chỉ báo cáo"
		if slices.Contains(gateSystems, b) {
			role = "baseline cổng"
		}
		say("\n=== EAIS so với %s (%s) ===\n%s", sysLabel[b], role, v.Format())
		say("Bộ đếm độc lập, %s (chỉ báo cáo): %d câu · rò rỉ canary %d · số không căn cứ %d\n", sysLabel[b], counts[b].Answers, counts[b].Leaks, counts[b].Unsupported)
	}
	pass := gatePass(s.order, s.blocked, unstable, invalid, verdicts)
	flaky := flakyCases(passCounts(graded), cfg.k)
	for _, sys := range s.order {
		say("ca chập chờn của %s (đạt ở 0 < n < %d lượt): %d %s\n", sysLabel[sys], cfg.k, len(flaky[sys]), strings.Join(flaky[sys], " "))
	}
	if cfg.pilot {
		say("PILOT (ADR-0006 §6): k của lượt chính = %d (hệ cổng có ≥ 2 ca chập chờn thì 5, không thì 3). Pilot KHÔNG phải bằng chứng G7.\n", pilotK(flaky))
	}
	say("\nBộ đếm độc lập, EAIS: %d câu · rò rỉ canary %d · số không căn cứ %d\n", counts[sysEAIS].Answers, counts[sysEAIS].Leaks, counts[sysEAIS].Unsupported)
	for _, e := range counts[sysEAIS].Examples {
		say("  - %s\n", e)
	}
	rule := "KHÔNG ĐẠT — xem luật, hệ cổng thiếu hoặc lượt vô hiệu ở trên"
	if pass {
		rule = "ĐẠT với mọi baseline cổng (" + strings.Join(gateSystems, ", ") + ")"
	}
	line, code := verdict(rule, verdictInput{pass: pass, fake: fake, dry: dry, frozen: scorerFrozen, contentGrade: cfg.grade == gradeContent,
		contractParams: len(contractDeviations(cfg, gw)) == 0})
	say("\n%s\n", line)
	fmt.Print(out.String())

	if cfg.out != "" {
		if err := saveRun(cfg, mc, gw, fake, dry, pass, line, code, s, sink, full, graded, verdicts, counts, unstable, invalid, started, out.String()); err != nil {
			fmt.Fprintln(os.Stderr, "LỖI ghi bằng chứng:", err)
			return 2
		}
		fmt.Println("bằng chứng thô:", cfg.out)
	}
	return code
}

// gatePass là kết luận của G7 (ADR-0005 §5–6): ĐẠT chỉ khi MỌI baseline cổng đều chạy được và đều
// đạt, không hệ cổng nào bị chặn, không lượt nào vô hiệu (đường tất định không ổn định, lỗi gọi model,
// chạm trần ngân sách). Baseline chỉ-báo-cáo không ảnh hưởng.
func gatePass(order, blocked, unstable, invalid []string, verdicts map[string]evals.G7Verdict) bool {
	if len(blocked) > 0 || len(unstable) > 0 || len(invalid) > 0 {
		return false
	}
	for _, g := range gateSystems {
		v, ok := verdicts[g]
		if !slices.Contains(order, g) || !ok || !v.Pass {
			return false
		}
	}
	return true
}

// saveRun gom bằng chứng của một lần chạy và ghi ra cfg.out.
func saveRun(cfg config, mc models.Config, gw models.Gateway, fake, dry, pass bool, line string, code int, s *setup, sink *evidenceSink, full, graded map[string][]evals.Report,
	verdicts map[string]evals.G7Verdict, counts map[string]evals.G7Counts, unstable, invalid []string, started time.Time, summary string) error {
	m := newManifest(cfg, mc, gw, fake, s.order, len(s.cases), s.files)
	m.StartedAt, m.FinishedAt = started.Format(time.RFC3339), time.Now().Format(time.RFC3339)
	m.Blocked, m.Unstable, m.Invalid = s.blocked, unstable, invalid
	m.PackVersion, m.DryRun = s.packVersion, dry
	m.SpentUSD, m.BudgetHit = sink.spent()
	modelErr := map[string]bool{}
	for _, r := range sink.recs {
		if r.ModelError {
			modelErr[fmt.Sprintf("%s|%d|%s", r.System, r.Run, r.CaseID)] = true
		}
	}
	var scored []scoredRow
	var failures []failureRow
	for _, sys := range s.order {
		for i := range graded[sys] {
			for j, g := range graded[sys][i].Results {
				f := full[sys][i].Results[j]
				scored = append(scored, scoredRow{System: sys, Run: i + 1, CaseID: g.ID, Class: g.FailureClass,
					Pass: g.Pass, Problems: g.Problems, FullPass: f.Pass, FullProblems: f.Problems})
				if !g.Pass {
					layer := "CHƯA PHÂN LOẠI"
					if modelErr[fmt.Sprintf("%s|%d|%s", sys, i+1, g.ID)] {
						layer = "PROVIDER" // máy gán: ca trượt vì lỗi gọi model, không vì câu trả lời
					}
					failures = append(failures, failureRow{System: sys, Run: i + 1, CaseID: g.ID, Class: g.FailureClass,
						Problems: g.Problems, FirstWrongLayer: layer})
				}
			}
		}
	}
	pc := passCounts(graded)
	agg := map[string]any{"verdicts": verdicts, "counts": counts, "eais_unstable_cases": unstable, "invalid_runs": invalid, "blocked": s.blocked,
		"fake_model": fake, "dry_run": dry, "pass_count": pc, "flaky_cases": flakyCases(pc, cfg.k), "k": cfg.k, "pilot": cfg.pilot,
		// gate_pass là kết luận G7 thật (mã 0 của verdict); rule_pass chỉ là luật D5 trên các hệ cổng, chưa xét điều kiện
		// bằng chứng (model giả, chạy khô, bộ chấm chưa đóng băng…). Soát C2, F-01.
		"gate_pass": code == 0, "rule_pass": pass, "conclusion": line, "exit_code": code}
	if cfg.pilot {
		agg["pilot_main_k"] = pilotK(flakyCases(pc, cfg.k))
	}
	return writeEvidence(cfg.out, m, sink.recs, scored, agg, failures, costs(sink.recs, mc.Model, s.order), summary)
}

// smoke chạy N ca đầu MỘT lượt mỗi hệ: mọi câu hỏi phải không lỗi, và với model thật mọi lời gọi
// model phải kết thúc bằng HTTP 200 — tức provider NHẬN đúng bộ tham số sẽ đóng băng. Hệ cổng bị chặn
// thì KHÔNG ĐẠT (mã 1). Chạy khô mà đạt thì mã 2: chưa chứng minh nhà cung cấp thật nhận tham số.
// Giới hạn: lấy N ca ĐẦU nên đường gọi model của EAIS có thể chưa được thử (dòng EAIS ghi "lời gọi
// model 0"); mọi hệ dùng chung một gateway nên bộ tham số vẫn được thử qua baseline.
func smoke(ctx context.Context, cfg config, mc models.Config, gw models.Gateway, s *setup, sink *evidenceSink, fake, dry bool, started time.Time) int {
	cases := s.cases[:min(cfg.smoke, len(s.cases))]
	ok := true
	var out strings.Builder
	for _, sys := range s.order {
		rec, err := newRecorder(s.askers[sys], sys, 0, sink)
		if err != nil {
			fmt.Fprintln(os.Stderr, "LỖI:", err)
			return 2
		}
		before := len(sink.recs)
		evals.RunWin(ctx, rec, cases, s.principals)
		for _, r := range sink.recs[before:] {
			var st []string
			in, outTok := 0, 0
			for _, c := range r.Calls {
				st = append(st, fmt.Sprint(c.Status))
				in, outTok = in+c.InputTokens, outTok+c.OutputTokens
			}
			bad := r.Error != ""
			if !fake && r.ModelCalls > 0 && (len(r.Calls) == 0 || r.Calls[len(r.Calls)-1].Status != 200) {
				bad = true
			}
			ok = ok && !bad
			fmt.Fprintf(&out, "%-5s %s · lời gọi model %d · HTTP [%s] · token vào %d ra %d · lỗi %q\n", sysLabel[sys], r.CaseID, r.ModelCalls, strings.Join(st, " "), in, outTok, r.Error)
		}
	}
	for _, b := range s.blocked {
		fmt.Fprintf(&out, "HỆ CỔNG KHÔNG CHẠY: %s\n", b)
	}
	ok = ok && len(s.blocked) == 0 // hệ cổng chưa được thử thì thử khói chưa chứng minh được nó (C-13)
	res := "THỬ KHÓI: ĐẠT"
	if !ok {
		res = "THỬ KHÓI: KHÔNG ĐẠT"
	}
	fmt.Fprintln(&out, res)
	if dry {
		fmt.Fprintln(&out, "(chạy khô: endpoint không chính thức — chưa chứng minh nhà cung cấp thật nhận bộ tham số)")
	}
	fmt.Print(out.String())
	if cfg.out != "" {
		m := newManifest(cfg, mc, gw, fake, s.order, len(cases), s.files)
		m.StartedAt, m.FinishedAt, m.Blocked = started.Format(time.RFC3339), time.Now().Format(time.RFC3339), s.blocked
		m.PackVersion, m.DryRun = s.packVersion, dry
		m.SpentUSD, m.BudgetHit = sink.spent()
		if err := writeEvidence(cfg.out, m, sink.recs, nil, map[string]any{"smoke": res}, nil, costs(sink.recs, mc.Model, s.order), out.String()); err != nil {
			fmt.Fprintln(os.Stderr, "LỖI ghi bằng chứng:", err)
			return 2
		}
	}
	// Mã thoát như run(): 0 chỉ khi nhà cung cấp THẬT nhận bộ tham số; chạy khô đạt là 2 (C-13).
	switch {
	case fake:
		fmt.Println(blockedLine)
		return 2
	case !ok:
		return 1
	case dry:
		return 2
	}
	return 0
}

// passed đếm số ca đạt trong một Report.
func passed(r evals.Report) int {
	n := 0
	for _, x := range r.Results {
		if x.Pass {
			n++
		}
	}
	return n
}

// setup là mọi thứ một lượt G7 cần, dựng một lần.
type setup struct {
	askers                map[string]evals.Asker
	order                 []string // EAIS trước, rồi các baseline chạy được, theo thứ tự -systems
	blocked               []string // hệ cổng không chạy được và lý do
	principals            map[string]contracts.Principal
	cases                 []evals.WinCase
	files                 []string // tệp đầu vào cần băm vào manifest
	counter               *evals.G7Counter
	canaries              []string
	paragraphs, documents int
	packVersion           string
}

// build dựng EAIS (như winrun.Run), các baseline và bộ đếm độc lập trên CÙNG một bộ dữ liệu và CÙNG
// một gateway model.
func build(cfg config, gw models.Gateway, systems []string) (*setup, error) {
	ctx := context.Background()
	rt, err := tenant.Load(cfg.pack)
	if err != nil {
		return nil, err
	}
	sm, err := ingestion.LoadSources(filepath.Join(cfg.pack, "sources.yaml"), rt)
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(cfg.data, cfg.tenant)
	res, err := ingestion.Ingest(ctx, dir, rt, sm, ingestion.PdfToText{})
	if err != nil {
		return nil, err
	}
	w, err := loadWorld(cfg.world)
	if err != nil {
		return nil, err
	}
	ref, err := time.ParseInLocation("2006-01-02", w.ReferenceDate, rt.Location)
	if err != nil {
		return nil, err
	}
	principals := map[string]contracts.Principal{}
	for _, tn := range w.Tenants {
		if tn.ID != cfg.tenant {
			continue
		}
		for _, p := range tn.Principals {
			pr := contracts.Principal{ID: p.ID, TenantID: contracts.TenantID(cfg.tenant), Kind: contracts.PrincipalUser, Roles: []string{p.Role}}
			if p.EntityScope != "" {
				pr.Attributes = map[string]string{"entity_scope": p.EntityScope}
			}
			principals[p.ID] = pr
		}
	}
	if len(principals) == 0 {
		return nil, fmt.Errorf("world.yaml không có principal nào của tenant %s", cfg.tenant)
	}
	directory, err := ingestion.LoadDirectory(dir, sm)
	if err != nil {
		return nil, err
	}
	docs, err := rag.LoadMarkdown(filepath.Join(dir, "docs"), cfg.tenant)
	if err != nil {
		return nil, err
	}
	files, _ := filepath.Glob(cfg.cases)
	sort.Strings(files)
	if len(files) == 0 {
		return nil, fmt.Errorf("không có tệp ca nào khớp %q", cfg.cases)
	}
	var cases []evals.WinCase
	known := map[string]bool{}
	for id := range principals {
		known[id] = true
	}
	for _, f := range files {
		suite, err := evals.LoadWinSuite(f)
		if err != nil {
			return nil, err
		}
		// Kiểm định dạng bộ ca (lớp hỏng, route, outcome, id…) TRƯỚC lời gọi model đầu tiên: trước đây lớp lạ chỉ lộ ở
		// CompareG7 sau khi đã gọi model đủ k lượt (soát C2, F-03).
		if probs := evals.ValidateWinSuite(suite, cfg.tenant, known); len(probs) > 0 {
			return nil, fmt.Errorf("%s: %s", f, strings.Join(probs[:min(len(probs), 5)], "; "))
		}
		cases = append(cases, suite.Cases...)
	}
	// C-03 (soát lượt 2): ca có principal không thuộc tenant được chấm, hoặc id trùng, không được âm thầm bỏ qua —
	// trước đây ca đó không được hỏi, vắng khỏi bằng chứng thô, và ĐẠT nếu kỳ vọng là từ chối.
	seen := map[string]bool{}
	for _, c := range cases {
		if _, ok := principals[c.Principal]; !ok {
			return nil, fmt.Errorf("ca %s: principal %q không thuộc tenant %s trong world.yaml", c.ID, c.Principal, cfg.tenant)
		}
		if seen[c.ID] {
			return nil, fmt.Errorf("id ca %s xuất hiện hai lần trong %q", c.ID, cfg.cases)
		}
		seen[c.ID] = true
	}

	// EAIS: gateway bọc một bộ đếm lời gọi (cổng G4 đo bằng gateway, không bằng trace của hệ).
	counting := &countingGateway{inner: gw}
	now := ref.Add(10 * time.Hour)
	askers := map[string]evals.Asker{sysEAIS: eaisAsker{a: &pipeline.Ask{RT: rt, Deps: pipeline.Deps{
		Directory: directory, Facts: res.Facts(), Knowledge: rag.NewMemoryIndex(cfg.tenant, docs), Model: counting,
		// bản ghi quyết định gắn đúng bộ dữ liệu đã phục vụ, như winrun (soát C2, F-10)
		DataVersion: fileSHA256(filepath.Join(cfg.data, "manifest.json")),
		Now:         func() time.Time { return now },
	}}, gw: counting}}
	packFiles, _ := filepath.Glob(filepath.Join(cfg.pack, "*.yaml"))
	sort.Strings(packFiles)
	st := &setup{askers: askers, order: []string{sysEAIS}, principals: principals, cases: cases, documents: len(docs), packVersion: rt.PackVersion,
		files: append(append(append([]string(nil), files...), packFiles...), cfg.world, cfg.manifest, cfg.contract)}

	// Baseline RAG: mỗi quan sát của tenant một đoạn văn bản + cùng các tài liệu SOP, cùng gateway.
	names := map[string]string{}
	byID := map[string]entityresolution.Entity{}
	for _, e := range directory {
		names[e.ID] = e.Name
		byID[e.ID] = e
	}
	paras := func(plus bool) []string {
		out := make([]string, len(res.Observations))
		for i, o := range res.Observations {
			if plus {
				out[i] = paragraphPlus(rt, byID, o)
			} else {
				out[i] = paragraph(rt, names, o)
			}
		}
		return out
	}
	st.paragraphs = len(res.Observations)
	for _, sys := range systems[1:] {
		switch sys {
		case sysB2, sysB2Plus, sysB2PP:
			b := baseline.New(cfg.tenant, paras(sys != sysB2), docs, gw)
			b.TopK = cfg.topK
			switch sys {
			case sysB2Plus:
				b.Preamble = todayLine(ref)
			case sysB2PP:
				rules, err := domainRules(cfg.pack)
				if err != nil {
					return nil, err
				}
				b.Preamble = todayLine(ref) + "\n" + rules
			}
			askers[sys] = b
		case sysB1:
			if cfg.manifest == "" {
				st.blocked = append(st.blocked, "B1: thiếu -manifest (b1_window từng ca, do người soạn ca gán)")
				continue
			}
			windows, err := loadWindows(cfg.manifest, rt.Location)
			if err != nil {
				return nil, err
			}
			rows, err := readRawRows(dir, rt.Location, ingestion.PdfToText{})
			if err != nil {
				return nil, err
			}
			fixed, err := fixedEvidence(dir, docs)
			if err != nil {
				return nil, err
			}
			d, err := newDirectLLM(gw, rows, fixed, cases, windows, todayLine(ref))
			if err != nil {
				return nil, err
			}
			askers[sys] = d
		case sysB2Oracle:
			if cfg.manifest == "" {
				return nil, fmt.Errorf("B2-oracle cần -manifest có truth.record_ids của từng ca")
			}
			truths, err := loadTruths(cfg.manifest)
			if err != nil {
				return nil, err
			}
			recs, err := loadGenRecords(filepath.Join(cfg.data, "manifest.json"))
			if err != nil {
				return nil, err
			}
			plus := paras(true)
			inner := baseline.New(cfg.tenant, plus, docs, gw)
			inner.TopK, inner.Preamble = cfg.topK, todayLine(ref)
			o, err := newOracleRAG(inner, gw, cases, oracleKeys(truths, recs, cfg.tenant), res.Observations, plus, todayLine(ref))
			if err != nil {
				return nil, err
			}
			askers[sys] = o
		}
		st.order = append(st.order, sys)
	}

	// Bộ đếm độc lập: số được phép lấy từ world.yaml (không từ TOÀN BỘ dữ liệu nền sinh ra: vài nghìn
	// giá trị nền phủ gần hết dải số thì "được phép" thành vô nghĩa); số nền chỉ được phép ở đúng ca
	// trỏ tới nó (PerCase, cần -manifest có truth); canary là chữ chỉ có ở tenant khác.
	ownRecords, ownDocs := w.texts(cfg.tenant)
	var others []string
	for _, tn := range w.Tenants {
		if tn.ID != cfg.tenant {
			r, d := w.texts(tn.ID)
			others = append(append(others, r...), d...)
		}
	}
	own := append(append([]string(nil), ownRecords...), ownDocs...)
	canaries := evals.DeriveCanaries(own, others)
	if len(canaries) == 0 {
		return nil, fmt.Errorf("world.yaml không có canary nào của tenant khác: bộ đếm rò rỉ sẽ đếm được 0 vì mù, không vì sạch")
	}
	canaries = append(canaries, nameCanaries(w, cfg.tenant, own)...)
	var perCase map[string][]string
	if cfg.manifest != "" {
		if perCase, err = perCaseNumbers(cfg.manifest, filepath.Join(cfg.data, "manifest.json"), cfg.tenant); err != nil {
			return nil, err
		}
	}
	st.counter = evals.NewG7Counter(evals.G7CounterConfig{
		Canaries: canaries, Records: ownRecords, Documents: ownDocs, QuantityUnits: quantityUnits, PerCase: perCase,
	})
	st.canaries = canaries
	return st, nil
}

// paragraph viết một quan sát thành MỘT đoạn văn bản một dòng cho baseline: ngày, tên thực thể (danh
// mục thiết bị, ví dụ "Máy xúc EX-03"), tên chỉ số, nguyên văn lượng như nguồn ghi ("3850 L",
// "máy 03 sáng 2k"), tên nguồn.
//
// Baseline nhận quan sát ĐÃ QUA bộ nạp của EAIS (thực thể và ngày đã chuẩn hoá, bản ghi bẩn đã bị cách
// ly) — mạnh hơn một RAG đọc thẳng tệp nguồn, nên G7 khắt khe hơn thực tế. Đổi sang chữ thô là quyết
// định của chủ dự án. Tên thực thể có mặt vì đo được ích lợi: xem baseline.DefaultTopK.
func paragraph(rt *tenant.Runtime, names map[string]string, o ingestion.Observation) string {
	label := o.Metric
	if d, ok := rt.Metrics.Metric(o.Metric); ok && d.Label != "" {
		label = d.Label
	}
	name := names[o.Entity]
	if name == "" {
		name = o.Entity
	}
	return fmt.Sprintf("Ngày %s, %s: %s %s (%s).", o.On.Format("2006-01-02"), name, label, o.Raw, rt.Metrics.SourceLabel(o.Metric, o.Source))
}

// countingGateway đếm số lời gọi Generate của một hệ, bọc gateway chung.
type countingGateway struct {
	inner models.Gateway
	n     atomic.Int64
}

func (g *countingGateway) Generate(ctx context.Context, req models.Request) (contracts.Draft, error) {
	g.n.Add(1)
	return g.inner.Generate(ctx, req)
}

// eaisAsker đưa luồng hỏi của EAIS vào runner; số lời gọi model đo ở gateway.
type eaisAsker struct {
	a  *pipeline.Ask
	gw *countingGateway
	// onDecision nhận bản ghi quyết định của từng câu (run() nối vào evidenceSink.decision; C-11).
	onDecision func(traces.Decision)
}

func (e eaisAsker) ModelCalls() int  { return int(e.gw.n.Load()) }
func (e eaisAsker) ResetModelCalls() { e.gw.n.Store(0) }

func (e eaisAsker) Ask(ctx context.Context, pr contracts.Principal, q string) (contracts.AskResponse, error) {
	out, err := e.a.Handle(ctx, pr, contracts.AskRequest{ConversationID: "g7", MessageID: "g7", Text: q})
	if err == nil && e.onDecision != nil {
		e.onDecision(out.Decision)
	}
	return out.Response, err
}

// world là phần của world.yaml mà G7 cần (trường khác bỏ qua).
type world struct {
	ReferenceDate string `yaml:"reference_date"`
	Tenants       []struct {
		ID       string `yaml:"id"`
		Projects []struct {
			ID      string   `yaml:"id"`
			Name    string   `yaml:"name"`
			Aliases []string `yaml:"aliases"`
		} `yaml:"projects"`
		Equipment []struct {
			ID        string            `yaml:"id"`
			Name      string            `yaml:"name"`
			Aliases   []string          `yaml:"aliases"`
			SourceIDs map[string]string `yaml:"source_ids"`
		} `yaml:"equipment"`
		Principals []struct {
			ID          string `yaml:"id"`
			Role        string `yaml:"role"`
			EntityScope string `yaml:"entity_scope"`
		} `yaml:"principals"`
	} `yaml:"tenants"`
	Records []struct {
		Tenant string `yaml:"tenant"`
		Entity string `yaml:"entity"`
		On     string `yaml:"on"`
		Source string `yaml:"source"`
		ValueL string `yaml:"value_l"`
		Raw    string `yaml:"raw"`
		By     string `yaml:"by"`
		Ref    string `yaml:"ref"`
		Expect string `yaml:"expect"`
	} `yaml:"records"`
	Documents []struct {
		Tenant string `yaml:"tenant"`
		Title  string `yaml:"title"`
		Body   string `yaml:"body"`
	} `yaml:"documents"`
}

func loadWorld(path string) (world, error) {
	var w world
	b, err := os.ReadFile(path)
	if err != nil {
		return w, err
	}
	if err := yaml.Unmarshal(b, &w); err != nil {
		return w, fmt.Errorf("%s: %w", path, err)
	}
	return w, nil
}

// nameCanaries trả canary TÊN (D-G7-9 mục 12): mã tenant, mã / tên / bí danh dự án và thiết bị của mọi
// tenant KHÁC, giữ nguyên chữ ("Cầu Z", "PRJ-Z", "def-build"); DeriveCanaries chỉ bắt token có chữ số.
// Bỏ chuỗi ngắn hơn 4 ký tự (khớp nhầm quá dễ) và chuỗi có trong chữ của tenant được chấm (own): tên dùng
// chung như "EX-03" (tenant B cố ý dùng cùng khoá) thì nhắc tới không phải rò rỉ.
func nameCanaries(w world, tenantID string, own []string) []string {
	ownText := strings.ToLower(norm.NFC.String(strings.Join(own, "\n")))
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(norm.NFC.String(s))
		k := strings.ToLower(s)
		if utf8.RuneCountInString(s) < 4 || seen[k] || strings.Contains(ownText, k) {
			return
		}
		seen[k] = true
		out = append(out, s)
	}
	for _, t := range w.Tenants {
		if t.ID == tenantID {
			continue
		}
		add(t.ID)
		for _, p := range t.Projects {
			add(p.ID)
			add(p.Name)
			for _, a := range p.Aliases {
				add(a)
			}
		}
		for _, e := range t.Equipment {
			add(e.ID)
			add(e.Name)
			for _, a := range e.Aliases {
				add(a)
			}
		}
	}
	return out
}

// texts trả chữ của MỘT tenant trong world để bộ đếm tự tách số: records = danh mục (dự án, thiết bị,
// mã theo nguồn) + bản ghi; docs = tài liệu. Bản ghi `expect: quarantine` (bẩn có chủ ý) không phải
// fact nên số của nó không được coi là có căn cứ; id bản ghi của world (R-001…) không bao giờ hiện
// trong câu trả lời nên cũng không vào. Ngày tham chiếu của world có mặt ở mọi tenant.
func (w world) texts(tenantID string) (records, docs []string) {
	records = append(records, w.ReferenceDate)
	for _, t := range w.Tenants {
		if t.ID != tenantID {
			continue
		}
		for _, p := range t.Projects {
			records = append(records, p.ID, p.Name, strings.Join(p.Aliases, " "))
		}
		for _, e := range t.Equipment {
			records = append(records, e.ID, e.Name, strings.Join(e.Aliases, " "))
			for _, code := range e.SourceIDs {
				records = append(records, code)
			}
		}
	}
	for _, r := range w.Records {
		if r.Tenant == tenantID && r.Expect != "quarantine" {
			records = append(records, r.Entity, r.On, r.Source, r.ValueL, r.Raw, r.By, r.Ref)
		}
	}
	for _, d := range w.Documents {
		if d.Tenant == tenantID {
			docs = append(docs, d.Title+"\n"+d.Body)
		}
	}
	return records, docs
}
