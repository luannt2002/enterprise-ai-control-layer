package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/ingestion"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
	"github.com/luannt2002/enterprise-ai-control-layer/execution/commands"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// store là phần của kho mà ứng dụng dùng; *pgstore.Store hiện thực. Giao diện hẹp để test chèn được lỗi (ghi vết thất bại, panic
// trong luồng hỏi) mà không cần giết PG; hành vi thật (RLS, ghi lại không nhân đôi) do test có PG kiểm trên chính *pgstore.Store.
type store interface {
	entityresolution.Directory
	sourceoftruth.Store
	pipeline.KnowledgeRetriever
	SaveDecision(ctx context.Context, rec traces.Record) (bool, error)
	Decision(ctx context.Context, traceID string) (traces.Record, error)
	SaveUsage(ctx context.Context, u pgstore.UsageRecord) (bool, error)
	Usage(ctx context.Context, traceID string) ([]pgstore.UsageRecord, error)
	SaveQuarantine(ctx context.Context, loadID string, recs []pgstore.QuarantineRecord) (int, error)
	Quarantine(ctx context.Context, loadID string) ([]pgstore.QuarantineRecord, error)
	Load(ctx context.Context, entities []entityresolution.Entity, facts []sourceoftruth.Fact, docs []rag.Document) error
	Ping(ctx context.Context) error
	Ready(ctx context.Context) error
}

var _ store = (*pgstore.Store)(nil)

// config là đầu vào dựng app. Không có giá trị mặc định mang tên tenant: đường dẫn pack / dữ liệu do người chạy (hoặc script) đưa.
type config struct {
	Pack  string // thư mục pack của MỘT tenant
	Data  string // thư mục gốc dữ liệu: <Data>/<tenant>/… (đầu ra của bộ sinh, hoặc dữ liệu nguồn của tenant)
	World string // world.yaml: nguồn principal dev và đồng hồ "world" (chỉ cần khi serve)
	Now   string // "" = đồng hồ thật của tenant · "world" = ngày tham chiếu của world.yaml + 10 giờ (như đường đo WIN) · RFC3339
}

// app gom những gì dùng chung giữa CLI và HTTP: runtime của pack, bản đồ nguồn, kho, luồng hỏi.
type app struct {
	rt       *tenant.Runtime
	sm       ingestion.SourceMap
	st       store
	dataRoot string
	clock    func() time.Time // nil = đồng hồ của tenant
	model    *meteredModel
	log      *slog.Logger
	// afterRead chỉ để test: chạy sau khi bộ nạp đã đọc hết nguồn, trước khi băm lại thư mục (chèn thay đổi dữ liệu giữa chừng).
	afterRead func()
	// ask là luồng hỏi đang phục vụ. Deps của Ask là giá trị (DataVersion nằm trong đó) nên nạp lại dữ liệu không sửa tại chỗ — sẽ là
	// data race với các lượt đang chạy — mà dựng Ask mới rồi thế con trỏ; lượt đang chạy dùng nốt bản cũ.
	ask atomic.Pointer[pipeline.Ask]
	// tools là tools.yaml của pack, đã nạp và kiểm chéo với policy và danh mục chỉ số (HOP-DONG §4.3): cấu hình hỏng thì newApp lỗi nên không có
	// app nào chạy. cmds là pipeline lệnh theo tên tool, mỗi tool một route lệnh (server.handler); rỗng khi kho không có kho phiếu (bản dựng thử
	// bằng kho bộ nhớ), khi đó không có route lệnh nào.
	tools *commands.Tools
	cmds  map[string]*commands.Pipeline
}

// newApp tải pack, bản đồ nguồn, đồng hồ; tính phiên bản dữ liệu hiện có trên đĩa và dựng luồng hỏi trên kho st. Chưa chạm kho.
func newApp(cfg config, st store, log *slog.Logger) (*app, error) {
	rt, err := tenant.Load(cfg.Pack)
	if err != nil {
		return nil, err
	}
	if !validTenantID(rt.Meta.Tenant) {
		return nil, fmt.Errorf("pack %s: tenant %q không hợp lệ (chữ thường, số, gạch ngang / dưới)", cfg.Pack, rt.Meta.Tenant)
	}
	sm, err := ingestion.LoadSources(filepath.Join(cfg.Pack, "sources.yaml"), rt)
	if err != nil {
		return nil, err
	}
	a := &app{rt: rt, sm: sm, st: st, dataRoot: cfg.Data, model: &meteredModel{inner: &models.Extractive{}, name: "extractive"}, log: log}
	if a.clock, err = parseClock(cfg, rt); err != nil {
		return nil, err
	}
	if err := a.loadCommands(cfg.Pack); err != nil {
		return nil, err
	}
	dv, err := dataVersion(a.tenantDir())
	if err != nil {
		return nil, err
	}
	a.setAsk(dv)
	return a, nil
}

// loadCommands nạp tools.yaml của pack ngay sau tenant.Load (core/tenant không biết tệp này: luồng hỏi và công cụ đo giữ nguyên hành vi nạp) và
// dựng pipeline lệnh cho từng tool. Cấu hình hỏng thì trả lỗi gom đủ thông tin: ứng dụng không khởi động (HOP-DONG §4.3, A12). Số liệu, danh mục
// thực thể và đồng hồ là của chính app (luồng hỏi cũng đọc chúng từ đó); kho phiếu là của kho (commandStoreOf).
func (a *app) loadCommands(packDir string) error {
	tools, err := commands.LoadTools(packDir, a.rt.Policy, a.rt.Metrics)
	if err != nil {
		return err
	}
	a.tools = tools
	ts := commandStoreOf(a.st)
	if ts == nil {
		return nil
	}
	a.cmds = map[string]*commands.Pipeline{}
	for _, name := range tools.Names() {
		tool, _ := tools.Get(name)
		p, err := commands.New(commands.Deps{
			Tenant: a.tenantID(), Location: a.rt.Location, Tool: tool, Policy: a.rt.Policy, Catalog: a.rt.Metrics,
			Facts: a.st, Directory: a.st, Store: ts, PackVersion: a.rt.PackVersion, CodeVersion: buildRevision(), Now: a.now,
		})
		if err != nil {
			return fmt.Errorf("lệnh %s: %w", name, err)
		}
		a.cmds[name] = p
	}
	return nil
}

// commandStoreOf trả kho phiếu của st, hoặc nil khi st không có. *pgstore.Store mở được giao dịch đã đặt tenant (commands.TxBeginner) nên được bọc
// bằng kho PG của phiếu; một kho tự cài commands.Store (bản dựng thử) thì dùng thẳng. Giao diện store của app không đổi: kho bộ nhớ có sẵn của
// test vẫn là một store và đơn giản là không có lệnh.
func commandStoreOf(st store) commands.Store {
	switch s := any(st).(type) {
	case commands.Store:
		return s
	case commands.TxBeginner:
		return commands.NewPGStore(s)
	}
	return nil
}

// tenantID là tenant của pack mà app này phục vụ: MỘT tenant mỗi tiến trình.
func (a *app) tenantID() string { return a.rt.Meta.Tenant }

func (a *app) tenantDir() string { return filepath.Join(a.dataRoot, a.tenantID()) }

// setAsk dựng luồng hỏi trên kho cho phiên bản dữ liệu dv. Model luôn là model tất định (extractive): ứng dụng này KHÔNG có đường
// nào tới model trả tiền (docs/PROMPT-BUILD-EAIS.md §9.3; test TestKhongCoDuongGoiModelThat giữ điều đó).
func (a *app) setAsk(dv string) {
	a.ask.Store(&pipeline.Ask{RT: a.rt, Deps: pipeline.Deps{
		Directory: a.st, Facts: a.st, Knowledge: a.st, Model: a.model, DataVersion: dv, Now: a.clock,
	}})
}

// dataVersionNow là phiên bản dữ liệu mà luồng hỏi đang gắn vào mọi bản ghi quyết định.
func (a *app) dataVersionNow() string { return a.ask.Load().Deps.DataVersion }

// now là đồng hồ của lượt (dùng cho bản ghi lỗi, nơi Handle không trả Decision).
func (a *app) now() time.Time {
	if a.clock != nil {
		return a.clock().In(a.rt.Location)
	}
	return a.rt.Now()
}

func (a *app) versions() traces.Versions {
	return traces.Versions{Pack: a.rt.PackVersion, Code: buildRevision(), Data: a.dataVersionNow()}
}

// parseClock đọc cờ -now. "world" cho đúng đồng hồ của đường đo WIN (ngày tham chiếu của world.yaml lúc 10:00 giờ tenant) để câu
// "hôm qua" rơi vào dữ liệu seed; rỗng là đồng hồ thật.
func parseClock(cfg config, rt *tenant.Runtime) (func() time.Time, error) {
	switch cfg.Now {
	case "":
		return nil, nil
	case "world":
		w, err := loadWorld(cfg.World)
		if err != nil {
			return nil, err
		}
		ref, err := time.ParseInLocation("2006-01-02", w.ReferenceDate, rt.Location)
		if err != nil {
			return nil, fmt.Errorf("world %s: reference_date %q: %w", cfg.World, w.ReferenceDate, err)
		}
		t := ref.Add(10 * time.Hour)
		return func() time.Time { return t }, nil
	}
	t, err := time.Parse(time.RFC3339, cfg.Now)
	if err != nil {
		return nil, fmt.Errorf("-now %q: muốn rỗng, \"world\" hoặc RFC3339: %w", cfg.Now, err)
	}
	return func() time.Time { return t }, nil
}

// tenantSlug là dạng hợp lệ của tenant khi dùng làm tên thư mục: không có "/", "..", khoảng trắng.
var tenantSlug = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}$`)

func validTenantID(id string) bool { return tenantSlug.MatchString(id) }

// dataVersion là băm nội dung MỌI tệp dưới dir: sha256 của các dòng "<đường dẫn tương đối>\x00<sha256 nội dung>" xếp tăng dần. Thêm,
// sửa, xoá hay đổi tên một tệp đều đổi nó; cùng nội dung ở thư mục khác cho cùng giá trị. Thư mục rỗng hoặc không có là lỗi: không
// nạp "không gì" rồi báo là một phiên bản.
func dataVersion(dir string) (string, error) {
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return "", contracts.Label(contracts.Infra("data_dir"), fmt.Errorf("thư mục dữ liệu %s không có hoặc không phải thư mục", dir))
	}
	var lines []string
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		lines = append(lines, filepath.ToSlash(rel)+"\x00"+hex.EncodeToString(h.Sum(nil)))
		return nil
	})
	if err != nil {
		return "", contracts.Label(contracts.Infra("data_dir"), err)
	}
	if len(lines) == 0 {
		return "", contracts.Label(contracts.Infra("data_dir"), fmt.Errorf("thư mục dữ liệu %s rỗng", dir))
	}
	sort.Strings(lines)
	h := sha256.New()
	for _, l := range lines {
		io.WriteString(h, l+"\n")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// loadIDOf là mã lần nạp: băm của (nội dung dữ liệu, phiên bản pack, phiên bản mã). Cùng ba thứ ⇒ cùng mã ⇒ nạp lại không nhân đôi
// bản ghi cách ly (SaveQuarantine bỏ qua dòng y nguyên). Đổi dữ liệu hoặc pack ⇒ mã mới, bản ghi cách ly của lần nạp cũ giữ nguyên
// làm lịch sử. Phiên bản mã nằm trong mã vì cách ly do bộ nạp quyết: mã nạp đổi cách diễn đạt lý do cho cùng khoá thì là một lần
// nạp mới, không phải xung đột "cùng mã mà khác nội dung" khoá cứng người vận hành (xem platform/pgstore/vet.go).
func loadIDOf(dataVersion, packVersion, codeVersion string) string {
	sum := sha256.Sum256([]byte(dataVersion + "\n" + packVersion + "\n" + codeVersion))
	return "load-" + hex.EncodeToString(sum[:8])
}

// ingestReport là kết quả một lần nạp.
type ingestReport struct {
	LoadID        string `json:"load_id"`
	DataVersion   string `json:"data_version"`
	PackVersion   string `json:"pack_version"`
	Observations  int    `json:"observations"`
	Quarantined   int    `json:"quarantined"`
	NewQuarantine int    `json:"quarantine_new"` // bản ghi cách ly GHI MỚI; 0 khi nạp lại cùng mã
	ElapsedMs     int64  `json:"elapsed_ms"`
}

// ingest nạp thư mục dữ liệu <dataRoot>/<tenant> của tenant mà pack này phục vụ: nguồn → quan sát + cách ly; lưu bản ghi cách ly
// (kèm lý do) rồi THAY toàn bộ dữ liệu của tenant trong kho. Lưu cách ly TRƯỚC khi thay dữ liệu: nếu thứ hai hỏng thì không có dữ
// liệu sống nào thiếu bằng chứng cách ly của nó, và chạy lại là idempotent. Dữ liệu đổi trong lúc nạp (băm trước ≠ băm sau) là lỗi —
// phiên bản ghi vào vết phải là của đúng thứ đã nạp.
func (a *app) ingest(ctx context.Context) (ingestReport, error) {
	t0 := time.Now()
	tenantID, dir := a.tenantID(), a.tenantDir()
	before, err := dataVersion(dir)
	if err != nil {
		return ingestReport{}, err
	}
	res, err := ingestion.Ingest(ctx, dir, a.rt, a.sm, ingestion.PdfToText{})
	if err != nil {
		return ingestReport{}, err
	}
	directory, err := ingestion.LoadDirectory(dir, a.sm)
	if err != nil {
		return ingestReport{}, err
	}
	docs, err := rag.LoadMarkdown(filepath.Join(dir, "docs"), tenantID)
	if err != nil {
		return ingestReport{}, err
	}
	if a.afterRead != nil {
		a.afterRead()
	}
	after, err := dataVersion(dir)
	if err != nil {
		return ingestReport{}, err
	}
	if after != before {
		return ingestReport{}, contracts.Errorf(contracts.FaultData, "dữ liệu nguồn đổi trong lúc nạp; chạy lại")
	}
	rep := ingestReport{LoadID: loadIDOf(before, a.rt.PackVersion, buildRevision()), DataVersion: before, PackVersion: a.rt.PackVersion,
		Observations: len(res.Observations), Quarantined: len(res.Quarantined)}
	recs := make([]pgstore.QuarantineRecord, 0, len(res.Quarantined))
	for _, q := range res.Quarantined {
		recs = append(recs, pgstore.QuarantineRecord{Key: q.Key, Source: q.Source, Reason: q.Reason, Detail: q.Detail})
	}
	tctx := contracts.WithTenant(ctx, contracts.TenantID(tenantID))
	if rep.NewQuarantine, err = a.st.SaveQuarantine(tctx, rep.LoadID, recs); err != nil {
		return rep, err
	}
	if err := a.st.Load(tctx, directory, res.Facts(), docs); err != nil {
		return rep, err
	}
	a.setAsk(before)
	rep.ElapsedMs = time.Since(t0).Milliseconds()
	return rep, nil
}

// meteredModel bọc gateway để đếm lời gọi và token THEO LƯỢT (không theo tiến trình): bộ đếm nằm trong ngữ cảnh của lượt, nên các
// lượt song song không lẫn vào nhau. Usage của lượt ghi vào kho vết sau khi lượt chạy xong.
type meteredModel struct {
	inner models.Gateway
	name  string
}

type usageKey struct{}

// usageAcc cộng dồn lời gọi model của MỘT lượt.
type usageAcc struct {
	mu     sync.Mutex
	calls  int
	tokens contracts.Usage
}

func (m *meteredModel) Generate(ctx context.Context, req models.Request) (contracts.Draft, error) {
	d, err := m.inner.Generate(ctx, req)
	if acc, ok := ctx.Value(usageKey{}).(*usageAcc); ok {
		acc.mu.Lock()
		acc.calls++ // lời gọi lỗi cũng tính: tiền (nếu có) đã tiêu
		acc.tokens.InputTokens += d.Usage.InputTokens
		acc.tokens.OutputTokens += d.Usage.OutputTokens
		acc.tokens.CacheRead += d.Usage.CacheRead
		acc.tokens.CacheWrite += d.Usage.CacheWrite
		acc.mu.Unlock()
	}
	return d, err
}

// record là bản ghi usage của lượt traceID (calls = 0 khi route không cần model — ghi đúng như vậy).
func (a *usageAcc) record(traceID, model string) pgstore.UsageRecord {
	a.mu.Lock()
	defer a.mu.Unlock()
	return pgstore.UsageRecord{TraceID: traceID, Model: model, Calls: a.calls, Tokens: a.tokens}
}
