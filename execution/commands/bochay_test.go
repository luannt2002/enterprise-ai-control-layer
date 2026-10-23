package commands

// Bộ chạy CHUNG cho ca lệnh khai theo khuôn HOP-DONG §12.1 (MỘT bộ chạy cho mọi ca, KHÔNG rẽ nhánh theo mã ca).
//
// Đọc tệp ca YAML {suite: lenh, cases: [...]}, gọi evals.Validate, tự kiểm hình dạng theo §12.1 (ca sai hình dạng là ca ĐỎ, không bị
// bỏ qua), dựng thế giới từ tệp thế giới bằng bộ sinh tenants/construction/win/gen rồi nạp qua đường nạp thật (data/ingestion) vào
// bộ nhớ hoặc vào PostgreSQL (database riêng của lần chạy), dựng mỗi runtime của ca, chạy các lời gọi tuần tự đúng thứ tự khai (hoặc
// song song qua rào F7), đếm bằng kết nối RIÊNG của harness (vai quản trị với PG), rồi so với expected_state và
// expected_output_properties. Báo cáo theo ca, theo lớp (khoá scenario) và theo cổng X1–X9; không gộp một con số.
//
// Mỗi khẳng định của một ca là một test con (tên = khoá được khẳng định, ví dụ write_unit_calls), nên một đột biến làm ca đỏ ở ĐÚNG
// dòng "Phải thấy" của HOP-DONG §10 hiện ra trong tên test đỏ và scripts/mutate.sh (MUTATE_TEST_MUON) nhắm được từng dòng.
//
// Bộ chạy nằm ở tệp không có build tag để ca phát triển (testdata/chay) chạy trong `go test ./...` thường; cổng vào của bộ ca giữ lại
// (đọc đường tệp từ biến môi trường) ở bochay_giulai_test.go, chỉ bật bằng build tag giulai.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/ingestion"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
	"github.com/luannt2002/enterprise-ai-control-layer/evals"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/router"
	"github.com/luannt2002/enterprise-ai-control-layer/tenants/construction/win/gen"
)

// ---------------------------------------------------------------------------
// Cấu hình và thế giới
// ---------------------------------------------------------------------------

// chayCfg là đầu vào của một lần chạy. Pack và tệp thế giới do bên gọi chỉ ra; không có giá trị mặc định cho tệp ca và tệp thế giới.
type chayCfg struct {
	Cases string // tệp ca YAML
	World string // tệp thế giới (định dạng world.yaml): bộ sinh dựng dữ liệu hạt giống từ đây
	Pack  string // thư mục pack (tenant.Load, sources.yaml, policy.yaml, tools.yaml)
	// Overlay là thư mục tuỳ chọn thay hoặc thêm tệp của pack: policy.yaml (thay chính sách), tools.yaml (thay tool),
	// policy-them.yaml (vai THÊM sau khi đã kiểm luật nạp §4.3, chỉ cho ca phát triển như A09).
	Overlay string
	Seed    uint64
	// WantCasesSHA, WantWorldSHA: băm kỳ vọng do điều phối cấp; khác băm thật thì từ chối chạy. Rỗng thì không kiểm.
	WantCasesSHA string
	WantWorldSHA string
}

// tenantData là dữ liệu hạt giống đã nạp của một tenant (qua bộ sinh rồi đường nạp thật).
type tenantData struct {
	id    string
	dir   entityresolution.MemoryDirectory
	facts sourceoftruth.MemoryStore
}

// worlds là mọi thứ dùng chung của một lần chạy: pack, chính sách, tool, đồng hồ cố định và dữ liệu hạt giống từng tenant.
type worlds struct {
	loc         *time.Location
	now         time.Time // ngày tham chiếu lúc 10:00 giờ tenant, cố định suốt ca (§12.1)
	policy      router.Policy
	tool        Tool
	catalog     *sourceoftruth.Catalog
	packVersion string
	tenants     map[string]*tenantData
	order       []string
	files       map[string]string // tên → sha256 các tệp đầu vào đã dùng
	tmp         string
}

func fileSHA(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// checkWantSHA từ chối chạy khi băm thật khác băm kỳ vọng (§12.1: bộ ca đã đóng băng).
func checkWantSHA(what, got, want string) error {
	if want != "" && !strings.EqualFold(got, want) {
		return fmt.Errorf("từ chối chạy: băm %s là %s, kỳ vọng %s", what, got, want)
	}
	return nil
}

// buildWorlds dựng thế giới: bộ sinh render tệp thế giới ra thư mục dữ liệu, đường nạp thật đọc lại thành quan sát và danh mục.
func buildWorlds(cfg chayCfg, tmp string) (*worlds, error) {
	ctx := context.Background()
	wd := &worlds{tenants: map[string]*tenantData{}, files: map[string]string{}, tmp: tmp}
	for name, p := range map[string]string{"ca": cfg.Cases, "the_gioi": cfg.World} {
		sum, err := fileSHA(p)
		if err != nil {
			return nil, err
		}
		wd.files[name] = sum
	}
	if err := checkWantSHA("tệp ca", wd.files["ca"], cfg.WantCasesSHA); err != nil {
		return nil, err
	}
	if err := checkWantSHA("tệp thế giới", wd.files["the_gioi"], cfg.WantWorldSHA); err != nil {
		return nil, err
	}
	rt, err := tenant.Load(cfg.Pack)
	if err != nil {
		return nil, fmt.Errorf("nạp pack: %w", err)
	}
	wd.loc, wd.catalog, wd.packVersion = rt.Location, rt.Metrics, rt.PackVersion

	policy := rt.Policy
	toolsDir := cfg.Pack
	if cfg.Overlay != "" {
		if p := filepath.Join(cfg.Overlay, "policy.yaml"); exists(p) {
			policy = router.Policy{}
			if err := decodeStrict(p, &policy); err != nil {
				return nil, fmt.Errorf("overlay policy.yaml: %w", err)
			}
			wd.files["overlay_policy"], _ = fileSHA(p)
		}
		if p := filepath.Join(cfg.Overlay, "tools.yaml"); exists(p) {
			toolsDir = cfg.Overlay
			wd.files["overlay_tools"], _ = fileSHA(p)
		}
	}
	tools, err := LoadTools(toolsDir, policy, rt.Metrics) // luật nạp §4.3 kiểm trên chính sách THẬT, trước khi thêm vai chỉ cho ca phát triển
	if err != nil {
		return nil, fmt.Errorf("nạp tools.yaml: %w", err)
	}
	tl, ok := tools.Get(CommandOpenDiscrepancyTicket)
	if !ok {
		return nil, fmt.Errorf("pack %s chưa có tool %s (tools.yaml)", toolsDir, CommandOpenDiscrepancyTicket)
	}
	wd.tool = tl
	if cfg.Overlay != "" {
		if p := filepath.Join(cfg.Overlay, "policy-them.yaml"); exists(p) {
			var extra router.Policy
			if err := decodeStrict(p, &extra); err != nil {
				return nil, fmt.Errorf("overlay policy-them.yaml: %w", err)
			}
			merged := router.Policy{Roles: map[string]router.RoleDef{}}
			for k, v := range policy.Roles {
				merged.Roles[k] = v
			}
			for k, v := range extra.Roles {
				merged.Roles[k] = v
			}
			policy = merged
			wd.files["overlay_policy_them"], _ = fileSHA(p)
		}
	}
	wd.policy = policy

	var head struct {
		ReferenceDate string `yaml:"reference_date"`
		Tenants       []struct {
			ID string `yaml:"id"`
		} `yaml:"tenants"`
	}
	b, err := os.ReadFile(cfg.World)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(b, &head); err != nil {
		return nil, fmt.Errorf("đọc tệp thế giới: %w", err)
	}
	ref, err := time.ParseInLocation(time.DateOnly, head.ReferenceDate, rt.Location)
	if err != nil {
		return nil, fmt.Errorf("reference_date %q của tệp thế giới: %w", head.ReferenceDate, err)
	}
	wd.now = ref.Add(10 * time.Hour)

	out := filepath.Join(tmp, "du-lieu")
	if err := gen.Generate(cfg.World, out, "functional", cfg.Seed); err != nil {
		return nil, fmt.Errorf("bộ sinh không đọc được tệp thế giới (lỗi của bộ chạy, không sửa tệp đóng băng): %w", err)
	}
	sm, err := ingestion.LoadSources(filepath.Join(cfg.Pack, "sources.yaml"), rt)
	if err != nil {
		return nil, fmt.Errorf("nạp sources.yaml: %w", err)
	}
	for _, tn := range head.Tenants {
		dir := filepath.Join(out, tn.ID)
		res, err := ingestion.Ingest(ctx, dir, rt, sm, ingestion.PdfToText{})
		if err != nil {
			return nil, fmt.Errorf("nạp dữ liệu của tenant %s: %w", tn.ID, err)
		}
		d, err := ingestion.LoadDirectory(dir, sm)
		if err != nil {
			return nil, fmt.Errorf("danh mục của tenant %s: %w", tn.ID, err)
		}
		wd.tenants[tn.ID] = &tenantData{id: tn.ID, dir: d, facts: res.Facts()}
		wd.order = append(wd.order, tn.ID)
	}
	return wd, nil
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

// ---------------------------------------------------------------------------
// Ca: đọc, kiểm hình dạng
// ---------------------------------------------------------------------------

type chayCaller struct{ ID, Role, Scope, Tenant string }

type chayCall struct {
	Label   string
	Caller  chayCaller
	Runtime string
	Req     Request
	Reload  bool
}

type auditWant struct {
	Outcome string
	Count   int
	Reason  string
	Caller  string
}

type stateWant struct {
	TicketsOpen, LedgerRows, WriteUnitCalls, AuditWrites, CrossRows *int
	Audit                                                           []auditWant
	Other                                                           map[string]*stateWant
}

type operandWant struct{ Metric, Value, Unit *string }

type ticketWant struct {
	Status, SubjectKey, Metric, Day, Diff *string
	Approx                                *bool
	Left, Right                           *operandWant
}

type outWant struct {
	Results                                                    map[string]int
	ReplayedFalse, ReplayedTrue, DistinctTickets, DistinctDets *int
	InvalidFields                                              []string
	HasInvalidFields                                           bool
	Ticket                                                     *ticketWant
}

type chayCase struct {
	ID, Scenario, Tenant string
	Calls                []chayCall
	Point                string
	Parallel             int
	State                *stateWant
	Out                  *outWant
	Errs                 []string // lỗi hình dạng: ca ĐỎ, không chạy
}

type caseSet struct {
	Path     string
	SHA      string
	Tenant   string
	Cases    []chayCase
	Problems []string // lỗi của evals.Validate (cả tệp)
}

// normAny chuẩn hoá mọi bản đồ lồng nhau về map[string]any (§12.1): yaml.v3 giải mã bản đồ lồng trong evals.Case thành chính kiểu evals.Case,
// và khoá không phải chuỗi (true, 12) thành map[any]any.
func normAny(v any) any {
	switch x := v.(type) {
	case evals.Case:
		return normAny(map[string]any(x))
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, e := range x {
			out[k] = normAny(e)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(x))
		for k, e := range x {
			out[fmt.Sprint(k)] = normAny(e)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = normAny(e)
		}
		return out
	}
	return v
}

func asInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case uint64:
		return int(x), true
	}
	return 0, false
}

// loadCaseSet đọc tệp ca, gọi evals.Validate trên danh sách `cases` (evals.Load chỉ đọc danh sách trần, không đọc dạng bọc) rồi
// kiểm hình dạng từng ca theo §12.1.
func loadCaseSet(path string) (*caseSet, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(b)
	cs := &caseSet{Path: path, SHA: hex.EncodeToString(sum[:])}
	var doc struct {
		Suite string       `yaml:"suite"`
		Cases []evals.Case `yaml:"cases"`
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if doc.Suite != "lenh" {
		return nil, fmt.Errorf("%s: suite %q, muốn lenh", path, doc.Suite)
	}
	if len(doc.Cases) == 0 {
		return nil, fmt.Errorf("%s: không có ca nào", path)
	}
	if s, ok := doc.Cases[0]["tenant"].(string); ok {
		cs.Tenant = s
	}
	cs.Problems = evals.Validate(doc.Cases, cs.Tenant)
	for _, c := range doc.Cases {
		cs.Cases = append(cs.Cases, parseChayCase(c))
	}
	return cs, nil
}

func checkKeys(m map[string]any, where string, allowed ...string) []string {
	var bad []string
	for k := range m {
		if !slices.Contains(allowed, k) {
			bad = append(bad, fmt.Sprintf("%s: khoá lạ %q (hợp lệ: %s)", where, k, strings.Join(allowed, ", ")))
		}
	}
	sort.Strings(bad)
	return bad
}

func parseChayCase(c evals.Case) chayCase {
	str := func(k string) string { s, _ := c[k].(string); return s }
	pc := chayCase{ID: str("id"), Scenario: str("scenario"), Tenant: str("tenant")}
	bad := func(f string, a ...any) { pc.Errs = append(pc.Errs, fmt.Sprintf(f, a...)) }
	if str("entry_type") != "COMMAND" || str("expected_route") != "TOOL" {
		bad("ca lệnh phải có entry_type: COMMAND và expected_route: TOOL")
	}
	if pc.Scenario == "" {
		bad("thiếu scenario (mã lớp ca ở §10, ví dụ A01)")
	}
	pc.Errs = append(pc.Errs, parseInput(&pc, normAny(c["input"]), normAny(c["permissions"]))...)
	if fi := normAny(c["failure_injection"]); fi != nil {
		m, ok := fi.(map[string]any)
		if !ok {
			bad("failure_injection phải là null hoặc {point: <tên điểm>}")
		} else {
			pc.Errs = append(pc.Errs, checkKeys(m, "failure_injection", "point", "parallel")...)
			pt, _ := m["point"].(string)
			switch {
			case !validFaultPoint(pt):
				bad("failure_injection.point %q không phải một trong chín tên: %s", pt, strings.Join(faultPoints, ", "))
			case pt == fpRaceBarrier:
				n, ok := asInt(m["parallel"])
				if !ok || n < 2 {
					bad("race_barrier cần parallel: N (số nguyên >= 2)")
				}
				pc.Point, pc.Parallel = pt, n
			default:
				if _, has := m["parallel"]; has {
					bad("parallel chỉ dùng với race_barrier")
				}
				pc.Point = pt
			}
		}
	}
	if pc.Point == fpRaceBarrier {
		if pc.Parallel != len(pc.Calls) {
			bad("parallel (%d) khác số lời gọi sau khi mở repeat (%d)", pc.Parallel, len(pc.Calls))
		}
		for _, cl := range pc.Calls {
			if cl.Reload {
				bad("reload_before không dùng với race_barrier (mục %s)", cl.Label)
			}
		}
	}
	if pc.Point == fpReplyLost && len(pc.Calls) < 2 {
		bad("ca reply_lost cần ít nhất hai lời gọi (§12.1)")
	}
	pc.State = parseState(normAny(c["expected_state"]), "expected_state", true, &pc.Errs)
	pc.Out = parseOut(normAny(c["expected_output_properties"]), &pc.Errs)
	return pc
}

// parseInput điền Calls từ input và permissions; trả các lỗi hình dạng.
func parseInput(pc *chayCase, in, perm any) []string {
	var errs []string
	bad := func(f string, a ...any) { errs = append(errs, fmt.Sprintf(f, a...)) }
	field := func(m map[string]any, k, where string) string {
		v, has := m[k]
		if !has || v == nil {
			return ""
		}
		s, ok := v.(string)
		if !ok {
			bad("%s.%s phải là chuỗi (viết có nháy: ngày, số, on/off/yes/no dễ bị YAML hiểu thành kiểu khác), được %T", where, k, v)
		}
		return s
	}
	switch x := in.(type) {
	case map[string]any: // một lời gọi
		errs = append(errs, checkKeys(x, "input", "command_id", "subject_key", "metric", "day")...)
		pm, ok := perm.(map[string]any)
		if !ok {
			bad("permissions phải là {principal: <vai>, entity_scope: <chuỗi>} khi input là một bản đồ")
			return errs
		}
		errs = append(errs, checkKeys(pm, "permissions", "principal", "entity_scope")...)
		role := field(pm, "principal", "permissions")
		if role == "" {
			bad("permissions.principal (tên vai) là bắt buộc")
		}
		pc.Calls = []chayCall{{
			Label:   "input",
			Caller:  chayCaller{ID: "p-" + role, Role: role, Scope: field(pm, "entity_scope", "permissions"), Tenant: pc.Tenant},
			Runtime: pc.Tenant,
			Req: Request{CommandID: field(x, "command_id", "input"), SubjectKey: field(x, "subject_key", "input"),
				Metric: field(x, "metric", "input"), Day: field(x, "day", "input")},
		}}
	case []any: // nhiều lời gọi
		if perm != nil {
			bad("permissions phải là null khi input là danh sách (người gọi nằm ở từng mục)")
		}
		labels := map[string]bool{}
		callers := map[string]chayCaller{}
		for i, it := range x {
			im, ok := it.(map[string]any)
			if !ok {
				bad("input[%d] phải là bản đồ", i)
				continue
			}
			where := fmt.Sprintf("input[%d]", i)
			errs = append(errs, checkKeys(im, where, "id", "principal", "runtime", "command_id", "subject_key", "metric", "day", "repeat", "reload_before")...)
			label := field(im, "id", where)
			if label == "" || labels[label] {
				bad("%s: id mục rỗng hoặc trùng (%q)", where, label)
			}
			labels[label] = true
			pm, ok := im["principal"].(map[string]any)
			if !ok {
				bad("%s: thiếu principal {id, role, entity_scope, tenant}", where)
				continue
			}
			errs = append(errs, checkKeys(pm, where+".principal", "id", "role", "entity_scope", "tenant")...)
			rt := field(im, "runtime", where)
			if rt == "" {
				rt = pc.Tenant
			}
			cl := chayCaller{ID: field(pm, "id", where+".principal"), Role: field(pm, "role", where+".principal"),
				Scope: field(pm, "entity_scope", where+".principal"), Tenant: field(pm, "tenant", where+".principal")}
			if cl.ID == "" || cl.Role == "" {
				bad("%s.principal: id và role là bắt buộc", where)
			}
			if cl.Tenant == "" {
				cl.Tenant = rt
			}
			if prev, seen := callers[cl.ID]; seen && (prev.Role != cl.Role || prev.Scope != cl.Scope) {
				bad("%s: người gọi %q xuất hiện với vai hoặc phạm vi khác nhau", where, cl.ID)
			}
			callers[cl.ID] = cl
			n := 1
			if v, has := im["repeat"]; has {
				k, ok := asInt(v)
				if !ok || k < 1 {
					bad("%s.repeat phải là số nguyên >= 1", where)
				} else {
					n = k
				}
			}
			reload := false
			if v, has := im["reload_before"]; has {
				b, ok := v.(bool)
				if !ok {
					bad("%s.reload_before phải là true / false", where)
				}
				reload = b
			}
			req := Request{CommandID: field(im, "command_id", where), SubjectKey: field(im, "subject_key", where),
				Metric: field(im, "metric", where), Day: field(im, "day", where)}
			for k := 0; k < n; k++ {
				lb := label
				if n > 1 {
					lb = fmt.Sprintf("%s#%d", label, k+1)
				}
				pc.Calls = append(pc.Calls, chayCall{Label: lb, Caller: cl, Runtime: rt, Req: req, Reload: reload && k == 0})
			}
		}
		if len(x) == 0 {
			bad("input là danh sách rỗng")
		}
	default:
		bad("input phải là bản đồ (một lời gọi) hoặc danh sách (nhiều lời gọi), được %T", in)
	}
	return errs
}

var stateKeys = []string{"tickets_open", "ledger_rows", "audit", "write_unit_calls", "audit_writes", "other_tenants", "cross_tenant_rows"}

func parseState(v any, where string, top bool, errs *[]string) *stateWant {
	bad := func(f string, a ...any) { *errs = append(*errs, fmt.Sprintf(f, a...)) }
	if v == nil {
		return nil
	}
	if l, ok := v.([]any); ok && len(l) == 0 { // khuôn của CONVENTIONS §5 ghi [] khi không có kỳ vọng
		return nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		bad("%s phải là bản đồ khoá đếm hoặc null, được %T", where, v)
		return nil
	}
	allowed := stateKeys
	if !top {
		allowed = slices.DeleteFunc(slices.Clone(stateKeys), func(k string) bool { return k == "other_tenants" || k == "cross_tenant_rows" })
	}
	*errs = append(*errs, checkKeys(m, where, allowed...)...)
	w := &stateWant{}
	num := func(k string) *int {
		x, has := m[k]
		if !has {
			return nil
		}
		n, ok := asInt(x)
		if !ok || n < 0 {
			bad("%s.%s phải là số nguyên >= 0", where, k)
			return nil
		}
		return &n
	}
	w.TicketsOpen, w.LedgerRows, w.WriteUnitCalls, w.AuditWrites, w.CrossRows = num("tickets_open"), num("ledger_rows"), num("write_unit_calls"), num("audit_writes"), num("cross_tenant_rows")
	if a, has := m["audit"]; has {
		list, ok := a.([]any)
		if !ok {
			bad("%s.audit phải là danh sách {outcome, count, reason_code, caller}", where)
		}
		for i, it := range list {
			im, ok := it.(map[string]any)
			if !ok {
				bad("%s.audit[%d] phải là bản đồ", where, i)
				continue
			}
			*errs = append(*errs, checkKeys(im, fmt.Sprintf("%s.audit[%d]", where, i), "outcome", "count", "reason_code", "caller")...)
			aw := auditWant{}
			aw.Outcome, _ = im["outcome"].(string)
			aw.Reason, _ = im["reason_code"].(string)
			aw.Caller, _ = im["caller"].(string)
			n, okN := asInt(im["count"])
			if !slices.Contains([]string{"EXECUTED", "REPLAYED", "REFUSED", "VERIFY_FAILED", "any"}, aw.Outcome) {
				bad("%s.audit[%d].outcome %q không hợp lệ", where, i, aw.Outcome)
			}
			if !okN || n < 0 {
				bad("%s.audit[%d].count bắt buộc, là số nguyên >= 0", where, i)
			}
			aw.Count = n
			w.Audit = append(w.Audit, aw)
		}
	}
	if o, has := m["other_tenants"]; has {
		om, ok := o.(map[string]any)
		if !ok {
			bad("%s.other_tenants phải là bản đồ tenant → khoá đếm", where)
		}
		w.Other = map[string]*stateWant{}
		for tn, sub := range om {
			w.Other[tn] = parseState(sub, where+".other_tenants."+tn, false, errs)
		}
	}
	return w
}

func parseOut(v any, errs *[]string) *outWant {
	bad := func(f string, a ...any) { *errs = append(*errs, fmt.Sprintf(f, a...)) }
	if v == nil {
		return nil
	}
	if l, ok := v.([]any); ok && len(l) == 0 {
		return nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		bad("expected_output_properties phải là bản đồ khoá kết quả hoặc null, được %T", v)
		return nil
	}
	*errs = append(*errs, checkKeys(m, "expected_output_properties", "results", "replayed_false", "replayed_true", "distinct_ticket_ids", "distinct_details", "invalid_fields", "ticket")...)
	w := &outWant{}
	num := func(k string) *int {
		x, has := m[k]
		if !has {
			return nil
		}
		n, ok := asInt(x)
		if !ok || n < 0 {
			bad("expected_output_properties.%s phải là số nguyên >= 0", k)
			return nil
		}
		return &n
	}
	w.ReplayedFalse, w.ReplayedTrue, w.DistinctTickets, w.DistinctDets = num("replayed_false"), num("replayed_true"), num("distinct_ticket_ids"), num("distinct_details")
	if r, has := m["results"]; has {
		rm, ok := r.(map[string]any)
		if !ok {
			bad("expected_output_properties.results phải là bản đồ loại → số nguyên")
		}
		w.Results = map[string]int{}
		for k, x := range rm {
			n, ok := asInt(x)
			if !ok || n < 0 {
				bad("expected_output_properties.results.%s phải là số nguyên cụ thể >= 0 (không biểu thức)", k)
			}
			w.Results[k] = n
		}
	}
	if f, has := m["invalid_fields"]; has {
		list, ok := f.([]any)
		if !ok {
			bad("expected_output_properties.invalid_fields phải là danh sách chuỗi")
		}
		w.HasInvalidFields = true
		for _, e := range list {
			s, ok := e.(string)
			if !ok {
				bad("expected_output_properties.invalid_fields chỉ chứa chuỗi")
			}
			w.InvalidFields = append(w.InvalidFields, s)
		}
	}
	if t, has := m["ticket"]; has {
		tm, ok := t.(map[string]any)
		if !ok {
			bad("expected_output_properties.ticket phải là bản đồ theo dạng §2.5")
		} else {
			w.Ticket = parseTicketWant(tm, errs)
		}
	}
	return w
}

func parseTicketWant(m map[string]any, errs *[]string) *ticketWant {
	bad := func(f string, a ...any) { *errs = append(*errs, fmt.Sprintf(f, a...)) }
	*errs = append(*errs, checkKeys(m, "ticket", "status", "subject_key", "metric", "day", "diff", "approx", "left", "right")...)
	s := func(k string) *string {
		x, has := m[k]
		if !has {
			return nil
		}
		v, ok := x.(string)
		if !ok {
			bad("ticket.%s phải là chuỗi (số thập phân viết thành chuỗi)", k)
			return nil
		}
		return &v
	}
	w := &ticketWant{Status: s("status"), SubjectKey: s("subject_key"), Metric: s("metric"), Day: s("day"), Diff: s("diff")}
	if x, has := m["approx"]; has {
		b, ok := x.(bool)
		if !ok {
			bad("ticket.approx phải là true / false")
		}
		w.Approx = &b
	}
	op := func(k string) *operandWant {
		x, has := m[k]
		if !has {
			return nil
		}
		om, ok := x.(map[string]any)
		if !ok {
			bad("ticket.%s phải là {metric, value, unit}", k)
			return nil
		}
		*errs = append(*errs, checkKeys(om, "ticket."+k, "metric", "value", "unit")...)
		get := func(f string) *string {
			y, has := om[f]
			if !has {
				return nil
			}
			v, ok := y.(string)
			if !ok {
				bad("ticket.%s.%s phải là chuỗi", k, f)
				return nil
			}
			return &v
		}
		return &operandWant{Metric: get("metric"), Value: get("value"), Unit: get("unit")}
	}
	w.Left, w.Right = op("left"), op("right")
	return w
}

// ---------------------------------------------------------------------------
// Nền lưu trữ: bộ nhớ và PostgreSQL
// ---------------------------------------------------------------------------

// chayBackend là nơi ca chạy và nơi harness đếm (kết nối riêng, ngoài đường ứng dụng).
type chayBackend interface {
	label() string
	deps(tn string) (Store, sourceoftruth.Store, entityresolution.Directory, error)
	reset(tenants []string) error
	arm(ctl *faultCtl)
	disarm()
	loadData(tn string) error
	tickets(tn string) (int, error) // phiếu OPEN
	ledger(tn string) (int, error)
	audits(tn, outcome, reason, caller string) (int, error) // outcome rỗng = mọi kết cục
	ticketExists(tn, id string) (bool, error)
	crossRows(tn string) (int, error)
}

type memBackend struct {
	m  *memStore
	wd *worlds
}

func newMemBackend(wd *worlds) *memBackend { return &memBackend{m: newMemStore(), wd: wd} }

func (b *memBackend) label() string { return "bo_nho" }

func (b *memBackend) deps(tn string) (Store, sourceoftruth.Store, entityresolution.Directory, error) {
	td, ok := b.wd.tenants[tn]
	if !ok {
		return nil, nil, nil, fmt.Errorf("tenant %q không có trong tệp thế giới", tn)
	}
	return b.m, td.facts, td.dir, nil
}

func (b *memBackend) reset(tenants []string) error {
	b.m.mu.Lock()
	defer b.m.mu.Unlock()
	for _, tn := range tenants {
		delete(b.m.rows, contracts.TenantID(tn))
	}
	b.m.failAuditInTx = 0
	return nil
}

func (b *memBackend) arm(ctl *faultCtl) {
	if ctl.point == fpInTxAuditInsert { // F2 của kho giả: bắn ở INSERT audit, sau hai INSERT đầu
		b.m.mu.Lock()
		b.m.failAuditInTx = 1
		b.m.mu.Unlock()
	}
}

func (b *memBackend) disarm() { b.m.mu.Lock(); b.m.failAuditInTx = 0; b.m.mu.Unlock() }

func (b *memBackend) loadData(tn string) error { // dữ liệu hạt giống của bộ nhớ bất biến: nạp lại đúng nó là không đổi
	if _, ok := b.wd.tenants[tn]; !ok {
		return fmt.Errorf("tenant %q không có trong tệp thế giới", tn)
	}
	return nil
}

func (b *memBackend) tickets(tn string) (int, error) {
	return b.m.countOpen(contracts.TenantID(tn)), nil
}
func (b *memBackend) ledger(tn string) (int, error) {
	return b.m.countLedger(contracts.TenantID(tn)), nil
}

func (b *memBackend) audits(tn, outcome, reason, caller string) (int, error) {
	return len(b.m.audits(contracts.TenantID(tn), AuditOutcome(outcome), ReasonCode(reason), caller)), nil
}

func (b *memBackend) ticketExists(tn, id string) (bool, error) {
	for _, t := range b.m.ticketList(contracts.TenantID(tn)) {
		if t.ID == id {
			return true, nil
		}
	}
	return false, nil
}

// kho giả tách tenant theo khoá bản đồ nên không có dòng của tenant khác để đọc: 0 theo cấu trúc (cách ly thật chỉ kiểm được ở PG).
func (b *memBackend) crossRows(string) (int, error) { return 0, nil }

type pgBackend struct {
	e  *pgEnv
	wd *worlds
}

func newPGBackend(e *pgEnv, wd *worlds) *pgBackend { return &pgBackend{e: e, wd: wd} }

func (b *pgBackend) label() string { return "pg" }

func (b *pgBackend) deps(tn string) (Store, sourceoftruth.Store, entityresolution.Directory, error) {
	if _, ok := b.wd.tenants[tn]; !ok {
		return nil, nil, nil, fmt.Errorf("tenant %q không có trong tệp thế giới", tn)
	}
	return b.e.pg, b.e.store, b.e.store, nil
}

func (b *pgBackend) reset(tenants []string) error {
	ctx := context.Background()
	for _, table := range []string{"commands", "audit_log", "discrepancy_tickets"} { // sổ lệnh có khoá ngoại tới phiếu: xoá trước
		if _, err := b.e.admin.Exec(ctx, "DELETE FROM "+table+" WHERE tenant_id = ANY($1)", tenants); err != nil {
			return fmt.Errorf("dọn %s: %w", table, err)
		}
	}
	b.e.pg.beforeAuditInsert = nil
	return nil
}

func (b *pgBackend) arm(ctl *faultCtl) {
	if ctl.point == fpInTxAuditInsert { // F2: ngay trước INSERT audit EXECUTED, sau INSERT phiếu và sổ lệnh trong cùng giao dịch
		b.e.pg.beforeAuditInsert = func() error {
			if ctl.reach(fpInTxAuditInsert) {
				return errors.New("F2 in_tx_audit_insert: INSERT audit bị từ chối giữa giao dịch")
			}
			return nil
		}
	}
}

func (b *pgBackend) disarm() { b.e.pg.beforeAuditInsert = nil }

func (b *pgBackend) loadData(tn string) error {
	td, ok := b.wd.tenants[tn]
	if !ok {
		return fmt.Errorf("tenant %q không có trong tệp thế giới", tn)
	}
	return b.e.store.Load(ctxOf(contracts.TenantID(tn)), td.dir, td.facts, nil)
}

func (b *pgBackend) scalar(q string, args ...any) (int, error) {
	var n int
	err := b.e.admin.QueryRow(context.Background(), q, args...).Scan(&n)
	return n, err
}

func (b *pgBackend) tickets(tn string) (int, error) {
	return b.scalar("SELECT count(*) FROM discrepancy_tickets WHERE tenant_id = $1 AND status = 'OPEN'", tn)
}

func (b *pgBackend) ledger(tn string) (int, error) {
	return b.scalar("SELECT count(*) FROM commands WHERE tenant_id = $1", tn)
}

func (b *pgBackend) audits(tn, outcome, reason, caller string) (int, error) {
	return b.scalar(`SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND ($2::text = '' OR outcome = $2)
		AND ($3::text = '' OR reason_code = $3) AND ($4::text = '' OR caller_id = $4)`, tn, outcome, reason, caller)
}

func (b *pgBackend) ticketExists(tn, id string) (bool, error) {
	n, err := b.scalar("SELECT count(*) FROM discrepancy_tickets WHERE tenant_id = $1 AND id = $2", tn, id)
	return n > 0, err
}

// crossRows đọc ba bảng bằng VAI ỨNG DỤNG đã đặt GUC tenant của runtime và đếm dòng thuộc tenant KHÁC (RLS phải giấu hết: 0).
func (b *pgBackend) crossRows(tn string) (int, error) {
	ctx := ctxOf(contracts.TenantID(tn))
	tx, err := b.e.store.BeginTenantTx(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	total := 0
	for _, table := range []string{"discrepancy_tickets", "commands", "audit_log"} {
		var n int
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE tenant_id <> current_setting('eais.tenant_id')").Scan(&n); err != nil {
			return 0, err
		}
		total += n
	}
	return total, nil
}

// ---------------------------------------------------------------------------
// Chạy một ca
// ---------------------------------------------------------------------------

type callResult struct {
	Label    string
	Runtime  string
	Received bool // false: kết quả bị F4 bỏ (bên gọi không nhận gì)
	Out      Outcome
}

type rtState struct {
	tenant string
	fs     *faultStore
	pipe   *Pipeline
}

// caseRun là một lần chạy một ca: các runtime, điểm chèn lỗi và kết quả nhận được.
type caseRun struct {
	pc  *chayCase
	wd  *worlds
	be  chayBackend
	ctl *faultCtl
	rts map[string]*rtState
	res []callResult
}

func (r *caseRun) tenantsInvolved() []string {
	set := map[string]bool{r.pc.Tenant: true}
	for _, c := range r.pc.Calls {
		set[c.Runtime], set[c.Caller.Tenant] = true, true
	}
	if r.pc.State != nil {
		for tn := range r.pc.State.Other {
			set[tn] = true
		}
	}
	out := make([]string, 0, len(set))
	for tn := range set {
		out = append(out, tn)
	}
	sort.Strings(out)
	return out
}

func (r *caseRun) mkRuntime(tn string) error {
	st, facts, dir, err := r.be.deps(tn)
	if err != nil {
		return err
	}
	rt := r.rts[tn]
	if rt == nil {
		rt = &rtState{tenant: tn, fs: &faultStore{inner: st, ctl: r.ctl}}
		r.rts[tn] = rt
	}
	p, err := New(Deps{
		Tenant: tn, Location: r.wd.loc, Tool: r.wd.tool, Policy: r.wd.policy, Catalog: r.wd.catalog, Facts: facts, Directory: dir,
		Store: rt.fs, PackVersion: r.wd.packVersion, CodeVersion: "chay-ca", Now: func() time.Time { return r.wd.now },
	})
	if err != nil {
		return err
	}
	rt.pipe = p
	return nil
}

// restart dựng pipeline MỚI trên cùng kho và cùng lớp bọc (bộ đếm tích luỹ): giả lập khởi động lại tiến trình (F4).
func (r *caseRun) restart() error {
	for tn := range r.rts {
		if err := r.mkRuntime(tn); err != nil {
			return err
		}
	}
	return nil
}

func principalOf(c chayCaller) contracts.Principal {
	p := contracts.Principal{ID: c.ID, TenantID: contracts.TenantID(c.Tenant), Kind: contracts.PrincipalUser, Roles: []string{c.Role}}
	if c.Scope != "" {
		p.Attributes = map[string]string{"entity_scope": c.Scope}
	}
	return p
}

// execute chạy ca: trạng thái sạch, dựng runtime, chạy các lời gọi. Lỗi trả về là lỗi của ca (ca ĐỎ).
func (r *caseRun) execute() error {
	ctl, err := newFaultCtl(r.pc.Point, r.pc.Parallel, raoWait)
	if err != nil {
		return err
	}
	r.ctl, r.rts = ctl, map[string]*rtState{}
	if err := r.be.reset(r.tenantsInvolved()); err != nil { // mỗi ca bắt đầu từ trạng thái sạch
		return err
	}
	r.be.arm(ctl)
	defer r.be.disarm()
	for _, c := range r.pc.Calls {
		if r.rts[c.Runtime] == nil {
			if err := r.mkRuntime(c.Runtime); err != nil {
				return err
			}
		}
	}
	ctx := context.Background()
	if r.pc.Point == fpRaceBarrier {
		out := make([]callResult, len(r.pc.Calls))
		var wg sync.WaitGroup
		for i, c := range r.pc.Calls {
			wg.Add(1)
			go func() {
				defer wg.Done()
				o := r.rts[c.Runtime].pipe.Run(ctx, principalOf(c.Caller), c.Req)
				out[i] = callResult{Label: c.Label, Runtime: c.Runtime, Received: true, Out: o}
			}()
		}
		wg.Wait()
		r.res = out
		if ctl.rao.timedOut.Load() {
			return fmt.Errorf("F7: không đủ %d lần tới rào trong hạn %s (ca ĐỎ, không được bỏ qua)", r.pc.Parallel, raoWait)
		}
		return nil
	}
	for _, c := range r.pc.Calls { // tuần tự đúng thứ tự khai: lời gọi k+1 bắt đầu sau khi lời gọi k đã trả kết quả (hoặc đã bị F4 bỏ)
		if c.Reload {
			if err := r.be.loadData(c.Runtime); err != nil {
				return fmt.Errorf("reload_before ở mục %s: Load lỗi: %w", c.Label, err)
			}
		}
		o := r.rts[c.Runtime].pipe.Run(ctx, principalOf(c.Caller), c.Req)
		res := callResult{Label: c.Label, Runtime: c.Runtime, Received: true, Out: o}
		if ctl.reach(fpTraceSaveFail) { // F8: kho vết hỏng SAU C7; kết quả đã ghi không đổi, chỉ mất trace_id (HOP-DONG §7.5, "Mất vết")
			res.Out.Result.TraceID = ""
			ctl.mu.Lock()
			ctl.traceSaveFailures++
			ctl.mu.Unlock()
		}
		ctl.releaseHeld()           // F5: COMMIT bị giữ được thả ngay sau khi lời gọi đã trả kết quả, trước lời gọi kế tiếp
		if ctl.reach(fpReplyLost) { // F4: bên gọi không nhận gì; dựng pipeline mới trên cùng cơ sở dữ liệu
			res.Received = false
			ctl.mu.Lock()
			ctl.restarts++
			ctl.mu.Unlock()
			if err := r.restart(); err != nil {
				return err
			}
		}
		r.res = append(r.res, res)
	}
	return nil
}

// ---------------------------------------------------------------------------
// So kỳ vọng
// ---------------------------------------------------------------------------

type chayCheck struct {
	Name string
	OK   bool
	Msg  string
}

func (r *caseRun) kind(o Outcome) string {
	if o.Result.Status == StatusDone {
		return "DONE"
	}
	return string(o.Result.ReasonCode)
}

func eqInt(name string, got int, want *int, checks *[]chayCheck) {
	if want == nil {
		return
	}
	*checks = append(*checks, chayCheck{Name: name, OK: got == *want, Msg: fmt.Sprintf("%s: được %d, muốn %d", name, got, *want)})
}

func (r *caseRun) evalState(prefix string, tn string, w *stateWant, checks *[]chayCheck) {
	add := func(name string, ok bool, f string, a ...any) {
		*checks = append(*checks, chayCheck{Name: prefix + name, OK: ok, Msg: prefix + fmt.Sprintf(f, a...)})
	}
	count := func(name string, f func() (int, error), want *int) {
		if want == nil {
			return
		}
		n, err := f()
		if err != nil {
			add(name, false, "%s: không đếm được: %v", name, err)
			return
		}
		add(name, n == *want, "%s: được %d, muốn %d", name, n, *want)
	}
	count("tickets_open", func() (int, error) { return r.be.tickets(tn) }, w.TicketsOpen)
	count("ledger_rows", func() (int, error) { return r.be.ledger(tn) }, w.LedgerRows)
	for _, a := range w.Audit {
		name := "audit." + a.Outcome
		if a.Reason != "" {
			name += "." + a.Reason
		}
		if a.Caller != "" {
			name += "." + a.Caller
		}
		out := a.Outcome
		if out == "any" {
			out = ""
		}
		n, err := r.be.audits(tn, out, a.Reason, a.Caller)
		if err != nil {
			add(name, false, "%s: không đếm được: %v", name, err)
			continue
		}
		add(name, n == a.Count, "%s: được %d, muốn %d", name, n, a.Count)
	}
	var wu, aw int
	if rt := r.rts[tn]; rt != nil {
		wu, aw = rt.fs.counters()
	}
	if w.WriteUnitCalls != nil {
		add("write_unit_calls", wu == *w.WriteUnitCalls, "write_unit_calls: được %d, muốn %d", wu, *w.WriteUnitCalls)
	}
	if w.AuditWrites != nil {
		add("audit_writes", aw == *w.AuditWrites, "audit_writes: được %d, muốn %d", aw, *w.AuditWrites)
	}
}

func (r *caseRun) evaluate() []chayCheck {
	var checks []chayCheck
	if w := r.pc.State; w != nil {
		r.evalState("", r.pc.Tenant, w, &checks)
		names := make([]string, 0, len(w.Other))
		for tn := range w.Other {
			names = append(names, tn)
		}
		sort.Strings(names)
		for _, tn := range names {
			if sub := w.Other[tn]; sub != nil {
				r.evalState("other_tenants."+tn+".", tn, sub, &checks)
			}
		}
		if w.CrossRows != nil {
			total, err := 0, error(nil)
			for tn := range r.rts {
				n, e := r.be.crossRows(tn)
				total, err = total+n, errors.Join(err, e)
			}
			ok := err == nil && total == *w.CrossRows
			checks = append(checks, chayCheck{Name: "cross_tenant_rows", OK: ok, Msg: fmt.Sprintf("cross_tenant_rows: được %d, muốn %d (lỗi %v)", total, *w.CrossRows, err)})
		}
	}
	if w := r.pc.Out; w != nil {
		r.evalOut(w, &checks)
	}
	return checks
}

func (r *caseRun) evalOut(w *outWant, checks *[]chayCheck) {
	var got []callResult
	for _, c := range r.res {
		if c.Received {
			got = append(got, c)
		}
	}
	add := func(name string, ok bool, f string, a ...any) {
		*checks = append(*checks, chayCheck{Name: name, OK: ok, Msg: fmt.Sprintf(f, a...)})
	}
	if w.Results != nil {
		have, total := map[string]int{}, 0
		for k, n := range w.Results {
			have[k] = 0
			total += n
		}
		for _, c := range got {
			have[r.kind(c.Out)]++
		}
		ok := total == len(got)
		for k, n := range w.Results {
			ok = ok && have[k] == n
		}
		for k, n := range have {
			if _, listed := w.Results[k]; !listed && n > 0 {
				ok = false // loại không nêu là 0
			}
		}
		add("results", ok, "results: được %v, muốn %v (tổng muốn %d, nhận được %d kết quả)", have, w.Results, total, len(got))
	}
	repl := func(name string, want *int, replayed bool) {
		if want == nil {
			return
		}
		n := 0
		for _, c := range got {
			if c.Out.Result.Status == StatusDone && c.Out.Result.Replayed == replayed {
				n++
			}
		}
		add(name, n == *want, "%s: được %d, muốn %d", name, n, *want)
	}
	repl("replayed_false", w.ReplayedFalse, false)
	repl("replayed_true", w.ReplayedTrue, true)
	if w.DistinctTickets != nil {
		ids := map[string]bool{}
		ok, why := true, ""
		for _, c := range got {
			if t := c.Out.Result.Ticket; t != nil {
				ids[t.ID] = true
				if found, err := r.be.ticketExists(c.Runtime, t.ID); err != nil || !found {
					ok, why = false, fmt.Sprintf("; mã %s không phải dòng của bảng phiếu tenant %s (lỗi %v)", t.ID, c.Runtime, err)
				}
			}
		}
		add("distinct_ticket_ids", ok && len(ids) == *w.DistinctTickets, "distinct_ticket_ids: được %d, muốn %d%s", len(ids), *w.DistinctTickets, why)
	}
	if w.DistinctDets != nil {
		ds := map[string]bool{}
		for _, c := range got {
			ds[c.Out.Result.Detail] = true
		}
		add("distinct_details", len(ds) == *w.DistinctDets, "distinct_details: được %d, muốn %d", len(ds), *w.DistinctDets)
	}
	if w.HasInvalidFields {
		set := map[string]bool{}
		for _, c := range got {
			for _, f := range c.Out.Result.InvalidFields {
				set[f] = true
			}
		}
		have := make([]string, 0, len(set))
		for f := range set {
			have = append(have, f)
		}
		sort.Strings(have)
		want := slices.Clone(w.InvalidFields)
		sort.Strings(want)
		want = slices.Compact(want)
		add("invalid_fields", slices.Equal(have, want), "invalid_fields: được %v, muốn %v", have, want)
	}
	if t := w.Ticket; t != nil {
		ok, why := true, ""
		done := 0
		for _, c := range got {
			if c.Out.Result.Status != StatusDone {
				continue
			}
			done++
			if bad := diffTicketWant(t, c.Out.Result.Ticket); bad != "" {
				ok, why = false, fmt.Sprintf("; mục %s: %s", c.Label, bad)
			}
		}
		if done == 0 {
			ok, why = false, "; không có kết quả DONE nào để so"
		}
		add("ticket", ok, "ticket: không khớp%s", why)
	}
}

func diffTicketWant(w *ticketWant, t *Ticket) string {
	if t == nil {
		return "kết quả DONE không có ticket"
	}
	var bad []string
	eq := func(name string, want *string, got string) {
		if want != nil && *want != got {
			bad = append(bad, fmt.Sprintf("%s được %q muốn %q", name, got, *want))
		}
	}
	eq("status", w.Status, t.Status)
	eq("subject_key", w.SubjectKey, t.SubjectKey)
	eq("metric", w.Metric, t.Metric)
	eq("day", w.Day, t.Day)
	eq("diff", w.Diff, t.Diff)
	if w.Approx != nil && (t.Approx == nil || *t.Approx != *w.Approx) {
		bad = append(bad, fmt.Sprintf("approx được %v muốn %v", t.Approx != nil && *t.Approx, *w.Approx))
	}
	for _, s := range []struct {
		name string
		want *operandWant
		got  *Operand
	}{{"left", w.Left, t.Left}, {"right", w.Right, t.Right}} {
		if s.want == nil {
			continue
		}
		if s.got == nil {
			bad = append(bad, s.name+" vắng")
			continue
		}
		eq(s.name+".metric", s.want.Metric, s.got.Metric)
		eq(s.name+".value", s.want.Value, s.got.Value)
		eq(s.name+".unit", s.want.Unit, s.got.Unit)
	}
	return strings.Join(bad, "; ")
}

// ---------------------------------------------------------------------------
// Báo cáo theo ca, theo lớp, theo cổng
// ---------------------------------------------------------------------------

type caseOutcome struct {
	ID       string   `json:"id"`
	Scenario string   `json:"scenario"`
	OK       bool     `json:"ok"`
	Failed   []string `json:"failed,omitempty"`
}

type gateDef struct {
	ID      string
	Name    string
	Classes []string
}

// chayGates là bảng cổng X1–X9 của HOP-DONG §11.1. X8 (đột biến) do danh mục đột biến, không do bộ chạy.
var chayGates = []gateDef{
	{"X1", "gửi đôi", []string{"A01", "A02", "A02b", "A03", "A04", "A04b", "A05", "A06"}},
	{"X2", "thiếu quyền", []string{"A07", "A08", "A09", "A10", "A11", "A12"}},
	{"X3", "timeout sau khi ghi", []string{"A21", "A22"}},
	{"X4", "nguyên tử và hỏng giữa chừng", []string{"A19", "A20", "A23"}},
	{"X5", "dữ liệu và luật", []string{"A13", "A14", "A15", "A16", "A17", "A18"}},
	{"X6", "biên", []string{"A24", "A25", "A26"}},
	{"X7", "không đổi đường hỏi", []string{"A27"}},
	{"X8", "đột biến", nil},
	{"X9", "cửa vào và vết", []string{"A28", "A29"}},
}

// ngoaiKhuon là các lớp mà khuôn §12.1 chưa khai được (§12, U10) hoặc chỉ là ca phát triển: không tính là THIẾU khi vắng.
var ngoaiKhuon = map[string]bool{"A09": true, "A11": true, "A12": true, "A24": true, "A25": true, "A26": true, "A27": true, "A28": true, "A29": true}

type classTally struct {
	Cases int `json:"cases"`
	OK    int `json:"ok"`
}

type gateReport struct {
	ID      string                `json:"id"`
	Name    string                `json:"name"`
	Status  string                `json:"status"`
	Classes map[string]classTally `json:"classes"`
	Absent  []string              `json:"absent,omitempty"`
}

type chayReport struct {
	Backend string                `json:"backend"`
	Files   map[string]string     `json:"sha256"`
	Cases   []caseOutcome         `json:"cases"`
	Classes map[string]classTally `json:"by_class"`
	Gates   []gateReport          `json:"by_gate"`
}

func newReport(backend string, files map[string]string) *chayReport {
	return &chayReport{Backend: backend, Files: files, Classes: map[string]classTally{}}
}

func (rp *chayReport) add(c caseOutcome) {
	rp.Cases = append(rp.Cases, c)
	t := rp.Classes[c.Scenario]
	t.Cases++
	if c.OK {
		t.OK++
	}
	rp.Classes[c.Scenario] = t
}

// finish tính trạng thái từng cổng: ĐỎ nếu có ca đỏ ở một lớp của cổng; THIẾU nếu vắng một lớp mà khuôn khai được; ĐẠT còn lại.
func (rp *chayReport) finish() {
	rp.Gates = nil
	for _, g := range chayGates {
		gr := gateReport{ID: g.ID, Name: g.Name, Classes: map[string]classTally{}}
		red, present := false, 0
		var missing []string
		for _, c := range g.Classes {
			t, ok := rp.Classes[c]
			if !ok {
				gr.Absent = append(gr.Absent, c)
				if !ngoaiKhuon[c] {
					missing = append(missing, c)
				}
				continue
			}
			present++
			gr.Classes[c] = t
			red = red || t.OK != t.Cases
		}
		switch {
		case g.ID == "X8":
			gr.Status = "NGOÀI BỘ CHẠY (danh mục đột biến scripts/dot-bien/)"
		case red:
			gr.Status = "ĐỎ"
		case present == 0:
			gr.Status = "KHÔNG CÓ CA (ngoài khuôn §12.1 hoặc vắng)"
		case len(missing) > 0:
			gr.Status = "THIẾU CA: " + strings.Join(missing, ", ")
		case len(gr.Absent) > 0:
			gr.Status = "ĐẠT phần khai được bằng khuôn (ngoài khuôn: " + strings.Join(gr.Absent, ", ") + ")"
		default:
			gr.Status = "ĐẠT"
		}
		rp.Gates = append(rp.Gates, gr)
	}
}

func (rp *chayReport) render() string {
	rp.finish()
	var b strings.Builder
	fmt.Fprintf(&b, "BỘ CHẠY CA LỆNH (HOP-DONG §12.1) — nền %s\n", rp.Backend)
	keys := make([]string, 0, len(rp.Files))
	for k := range rp.Files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "  sha256 %-20s %s\n", k, rp.Files[k])
	}
	fmt.Fprintf(&b, "THEO LỚP (ca đạt/ca):")
	classes := make([]string, 0, len(rp.Classes))
	for c := range rp.Classes {
		classes = append(classes, c)
	}
	sort.Strings(classes)
	for _, c := range classes {
		fmt.Fprintf(&b, " %s %d/%d", c, rp.Classes[c].OK, rp.Classes[c].Cases)
	}
	b.WriteString("\nTHEO CỔNG (không gộp một con số):\n")
	for _, g := range rp.Gates {
		fmt.Fprintf(&b, "  %s %-30s %s\n", g.ID, g.Name, g.Status)
		cl := make([]string, 0, len(g.Classes))
		for c := range g.Classes {
			cl = append(cl, c)
		}
		sort.Strings(cl)
		for _, c := range cl {
			fmt.Fprintf(&b, "      %s %d/%d\n", c, g.Classes[c].OK, g.Classes[c].Cases)
		}
	}
	red := 0
	for _, c := range rp.Cases {
		if !c.OK {
			red++
			fmt.Fprintf(&b, "CA ĐỎ %s (%s): %s\n", c.ID, c.Scenario, strings.Join(c.Failed, " | "))
		}
	}
	fmt.Fprintf(&b, "TỔNG: %d ca, %d đỏ\n", len(rp.Cases), red)
	return b.String()
}

// ---------------------------------------------------------------------------
// Điểm vào dùng chung
// ---------------------------------------------------------------------------

// chayTempDir là thư mục tạm của bộ sinh, dùng chung cả tiến trình test và dọn ở TestMain.
var (
	chayOnce sync.Once
	chayDir  string
	chayErr  error
)

func chayTemp() (string, error) {
	chayOnce.Do(func() { chayDir, chayErr = os.MkdirTemp("", "eais-chay-ca-") })
	return chayDir, chayErr
}

func TestMain(m *testing.M) {
	code := m.Run()
	if chayDir != "" {
		_ = os.RemoveAll(chayDir)
	}
	os.Exit(code)
}

var (
	worldMu    sync.Mutex
	worldCache = map[string]*worlds{}
)

// worldFor dựng thế giới một lần cho mỗi bộ đầu vào trong một tiến trình test (bộ sinh và đường nạp tốn vài giây).
func worldFor(cfg chayCfg) (*worlds, error) {
	worldMu.Lock()
	defer worldMu.Unlock()
	key := strings.Join([]string{cfg.Cases, cfg.World, cfg.Pack, cfg.Overlay, cfg.WantCasesSHA, cfg.WantWorldSHA}, "|")
	if wd, ok := worldCache[key]; ok {
		return wd, nil
	}
	tmp, err := chayTemp()
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp(tmp, "w-")
	if err != nil {
		return nil, err
	}
	wd, err := buildWorlds(cfg, dir)
	if err != nil {
		return nil, err
	}
	worldCache[key] = wd
	return wd, nil
}

// shapeError là lỗi hình dạng của một ca (§12.1): ca ĐỎ, không chạy.
type shapeError struct{ msg string }

func (e *shapeError) Error() string { return e.msg }

// execCase chạy một ca trên một nền và trả các khẳng định đã so. Ca sai hình dạng trả *shapeError (không chạy).
func execCase(pc *chayCase, wd *worlds, be chayBackend) ([]chayCheck, error) {
	if len(pc.Errs) > 0 {
		return nil, &shapeError{"ca sai hình dạng (ĐỎ, không chạy): " + strings.Join(pc.Errs, "; ")}
	}
	r := &caseRun{pc: pc, wd: wd, be: be}
	if err := r.execute(); err != nil {
		return nil, err
	}
	return r.evaluate(), nil
}

// runCaseSet chạy mọi ca của tệp trên MỘT nền, mỗi ca một test con và mỗi khẳng định một test con của ca. Trả báo cáo.
func runCaseSet(t *testing.T, cs *caseSet, wd *worlds, be chayBackend) *chayReport {
	t.Helper()
	rp := newReport(be.label(), wd.files)
	t.Run("hinh_dang_tep", func(t *testing.T) {
		for _, p := range cs.Problems {
			t.Errorf("evals.Validate: %s", p)
		}
	})
	for i := range cs.Cases {
		pc := &cs.Cases[i]
		co := caseOutcome{ID: pc.ID, Scenario: pc.Scenario, OK: true}
		t.Run(pc.Scenario+"_"+pc.ID, func(t *testing.T) {
			fail := func(name, msg string) {
				co.OK = false
				co.Failed = append(co.Failed, msg)
				t.Run(name, func(t *testing.T) { t.Error(msg) })
			}
			checks, err := execCase(pc, wd, be)
			var se *shapeError
			switch {
			case errors.As(err, &se):
				fail("hinh_dang", err.Error())
				return
			case err != nil:
				fail("chay", "ca không chạy được: "+err.Error())
				return
			}
			for _, ck := range checks {
				ok := ck.OK
				t.Run(ck.Name, func(t *testing.T) {
					if !ok {
						t.Error(ck.Msg)
					}
				})
				if !ok {
					co.OK = false
					co.Failed = append(co.Failed, ck.Msg)
				}
			}
		})
		rp.add(co)
	}
	return rp
}

// writeReport ghi báo cáo JSON ra path (nếu path không rỗng).
func writeReport(t *testing.T, path string, reports []*chayReport) {
	t.Helper()
	if path == "" {
		return
	}
	for _, r := range reports {
		r.finish()
	}
	b, err := json.MarshalIndent(reports, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		t.Fatalf("ghi báo cáo %s: %v", path, err)
	}
}
