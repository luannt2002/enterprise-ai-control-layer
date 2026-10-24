package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
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

// loadLedger là phần TUỲ CHỌN của kho: sổ nạp (migration 0006). *pgstore.Store ghi phiên bản của lần nạp CÙNG giao dịch thay dữ liệu và đọc lại dòng
// mới nhất của tenant; kho bộ nhớ của test không có sổ nên app dùng phiên bản nó tự nhớ (loadedVersion). Tách khỏi store như commands.Store: giao
// diện store của app không đổi, kho nào không có sổ đơn giản không có phiên bản bền.
type loadLedger interface {
	LoadWithVersion(ctx context.Context, info pgstore.LoadInfo, entities []entityresolution.Entity, facts []sourceoftruth.Fact, docs []rag.Document) error
	LatestLoad(ctx context.Context) (pgstore.LoadRecord, error)
}

var _ loadLedger = (*pgstore.Store)(nil)

func ledgerOf(st store) loadLedger {
	l, _ := any(st).(loadLedger)
	return l
}

// dataVersionNone là phiên bản mà vết khai khi PG không có lần nạp nào được ghi nhận cho tenant (chưa từng nạp, hoặc dữ liệu được nạp bằng đường không
// kèm phiên bản): nói rõ là chưa biết, không bịa một băm từ thư mục trên đĩa.
const dataVersionNone = "(chưa ghi nhận lần nạp)"

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
	// ask là luồng hỏi nền. Deps của Ask là giá trị (DataVersion nằm trong đó) nên nạp lại dữ liệu không sửa tại chỗ — sẽ là data race với các
	// lượt đang chạy — mà dựng Ask mới rồi thế con trỏ; lượt đang chạy dùng nốt bản cũ. DataVersion của nó là phiên bản app biết lúc dựng / nạp:
	// với kho có sổ nạp, MỖI lượt dùng askFor(phiên bản đọc từ sổ) chứ không dùng thẳng nó (xem loadedVersion).
	ask atomic.Pointer[pipeline.Ask]
	// tools là tools.yaml của pack, đã nạp và kiểm chéo với policy và danh mục chỉ số (HOP-DONG §4.3): cấu hình hỏng thì newApp lỗi nên không có
	// app nào chạy. cmds là pipeline lệnh theo tên tool, mỗi tool một route lệnh (server.handler); rỗng khi kho không có kho phiếu (bản dựng thử
	// bằng kho bộ nhớ), khi đó không có route lệnh nào.
	tools *commands.Tools
	cmds  map[string]*commands.Pipeline
}

// newApp tải pack, bản đồ nguồn, đồng hồ; kiểm thư mục dữ liệu dùng được rồi dựng luồng hỏi trên kho st. Kho có sổ nạp: phiên bản nền của luồng hỏi
// là dòng mới nhất của sổ (đọc một lần lúc khởi động); kho không có sổ: băm thư mục trên đĩa (dữ liệu nạp thẳng từ thư mục đó).
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
	dv, err := dataVersion(a.tenantDir()) // cũng là kiểm thư mục dữ liệu: có, là thư mục, không rỗng — serve không khởi động trên thư mục hỏng
	if err != nil {
		return nil, err
	}
	if ledgerOf(st) != nil { // phiên bản nền là của PG, KHÔNG phải của thư mục trên đĩa (thư mục có thể chưa từng được nạp: R2-01)
		ctx, cancel := context.WithTimeout(context.Background(), saveTimeout) // cùng hạn với một lần lưu vết: một truy vấn nhỏ
		defer cancel()
		if dv, err = a.loadedVersion(ctx, contracts.TenantID(a.tenantID())); err != nil {
			return nil, fmt.Errorf("không đọc được phiên bản dữ liệu từ sổ nạp của kho: %w", err)
		}
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
	if len(tools.Names()) == 0 { // route lệnh sẽ im lặng thành 404: nói ra lúc khởi động (soát K3, K3S-02; chú thích của LoadTools đòi vậy)
		a.log.Warn("pack không có tool lệnh nào (tools.yaml vắng, đổi tên hay rỗng): không có route /v1/commands/<tool>", "pack", packDir)
	} else {
		a.log.Info("đã nạp tool lệnh của pack", "pack", packDir, "tools", strings.Join(tools.Names(), ","))
	}
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

// dataVersionNow là phiên bản dữ liệu nền mà app đang nhớ (lúc dựng, hoặc lúc nạp xong gần nhất của chính app này). Vết KHÔNG lấy từ đây khi kho có
// sổ nạp: một lần nạp ở tiến trình khác (eais ingest) đổi PG mà app này không biết — xem loadedVersion.
func (a *app) dataVersionNow() string { return a.ask.Load().Deps.DataVersion }

// loadedVersion là phiên bản dữ liệu mà MỘT lượt phải ghi vào vết (R2-01). Kho có sổ nạp: phiên bản của lần nạp đã COMMIT mới nhất của tenant, đọc từ PG
// mỗi lần gọi (một truy vấn nhỏ) — nên nó đúng cả khi thư mục trên đĩa đổi, khi server trỏ nhầm thư mục, và khi PG vừa được nạp bằng CLI; chưa có lần nạp
// nào được ghi nhận thì dataVersionNone. Kho không có sổ (bộ nhớ của test): phiên bản app tự nhớ, đúng vì dữ liệu nạp thẳng từ thư mục đó. Lượt đọc phiên
// bản TRƯỚC khi chạy: lần nạp commit giữa chừng lượt vẫn có thể làm lượt đọc dữ liệu mới nhưng khai phiên bản cũ (giới hạn đã biết ở README).
func (a *app) loadedVersion(ctx context.Context, tenant contracts.TenantID) (string, error) {
	l := ledgerOf(a.st)
	if l == nil {
		return a.dataVersionNow(), nil
	}
	rec, err := l.LatestLoad(contracts.WithTenant(ctx, tenant))
	switch {
	case errors.Is(err, pgstore.ErrNotFound), err == nil && rec.DataVersion == "":
		return dataVersionNone, nil
	case err != nil:
		return "", err
	}
	return rec.DataVersion, nil
}

// askFor là luồng hỏi cho MỘT lượt: bản chép của luồng nền với phiên bản dữ liệu dv của lượt đó (Deps là giá trị, không sửa tại chỗ).
func (a *app) askFor(dv string) *pipeline.Ask {
	k := *a.ask.Load()
	k.Deps.DataVersion = dv
	return &k
}

// now là đồng hồ của lượt (dùng cho bản ghi lỗi, nơi Handle không trả Decision).
func (a *app) now() time.Time {
	if a.clock != nil {
		return a.clock().In(a.rt.Location)
	}
	return a.rt.Now()
}

// versions là ba phiên bản gắn vào bản ghi của lượt lỗi và lượt lệnh; dv là phiên bản dữ liệu của lượt đó (loadedVersion), rỗng khi không đọc được.
func (a *app) versions(dv string) traces.Versions {
	return traces.Versions{Pack: a.rt.PackVersion, Code: buildRevision(), Data: dv}
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
	// Held: số bản tài liệu cổng chất lượng giữ lại (chờ soát, KHÔNG được công bố ở lần nạp này); Holds: "mã@vN: lý do" của từng bản (F-018).
	Held  int      `json:"held"`
	Holds []string `json:"holds,omitempty"`
	// KeptPublished: "mã@vN: lý do của cổng" — bản cổng muốn giữ lại nhưng kho ĐANG công bố đúng nội dung đó, nên giữ nguyên công bố
	// (pgstore Load, rag.KeepPublished): cảnh báo cho người nạp, không phải lỗi.
	KeptPublished []string `json:"kept_published,omitempty"`
	ElapsedMs     int64    `json:"elapsed_ms"`
}

// ingest nạp thư mục dữ liệu <dataRoot>/<tenant> của tenant mà pack này phục vụ: nguồn → quan sát + cách ly; lưu bản ghi cách ly
// (kèm lý do) rồi THAY toàn bộ dữ liệu của tenant trong kho. Lưu cách ly TRƯỚC khi thay dữ liệu: nếu thứ hai hỏng thì không có dữ
// liệu sống nào thiếu bằng chứng cách ly của nó, và chạy lại là idempotent. Dữ liệu đổi trong lúc nạp (băm trước ≠ băm sau) là lỗi —
// phiên bản ghi vào vết phải là của đúng thứ đã nạp, và với kho có sổ nạp nó được ghi cùng giao dịch thay dữ liệu (loadInto).
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
	gate := make([]string, len(docs)) // lý do giữ của cổng TRƯỚC khi nạp: kho PG trả lại đã duyệt bản đang công bố đúng nội dung đó
	for i, d := range docs {
		gate[i] = d.Hold
	}
	if err := a.loadInto(tctx, pgstore.LoadInfo{LoadID: rep.LoadID, DataVersion: before}, directory, res.Facts(), docs); err != nil {
		return rep, err
	}
	// Bản bị cổng chất lượng giữ lại vẫn được nạp (chờ soát, không phục vụ): lần nạp thành công nhưng phải nói ra, không thì tài liệu
	// biến mất khỏi câu trả lời mà không ai biết (F-018).
	for i, d := range docs {
		switch at := fmt.Sprintf("%s@v%d: ", d.ID, d.Version); {
		case d.Hold != "":
			rep.Holds = append(rep.Holds, at+d.Hold)
		case gate[i] != "":
			rep.KeptPublished = append(rep.KeptPublished, at+gate[i])
		}
	}
	if rep.Held = len(rep.Holds); rep.Held > 0 {
		a.log.Warn("cổng chất lượng giữ lại tài liệu: các bản này KHÔNG được công bố", "tenant", tenantID, "load_id", rep.LoadID,
			"held", rep.Held, "holds", strings.Join(rep.Holds, " | "))
	}
	if len(rep.KeptPublished) > 0 {
		a.log.Warn("cổng chất lượng không đạt nhưng kho đang công bố đúng nội dung này: giữ nguyên công bố", "tenant", tenantID,
			"load_id", rep.LoadID, "kept_published", strings.Join(rep.KeptPublished, " | "))
	}
	a.setAsk(before)
	rep.ElapsedMs = time.Since(t0).Milliseconds()
	return rep, nil
}

// loadInto thay toàn bộ dữ liệu của tenant trong kho. Kho có sổ nạp ghi phiên bản của lần nạp CÙNG giao dịch với dữ liệu (dữ liệu và phiên bản cùng commit hoặc
// cùng huỷ); kho không có sổ chỉ thay dữ liệu.
func (a *app) loadInto(ctx context.Context, info pgstore.LoadInfo, entities []entityresolution.Entity, facts []sourceoftruth.Fact, docs []rag.Document) error {
	if l := ledgerOf(a.st); l != nil {
		return l.LoadWithVersion(ctx, info, entities, facts, docs)
	}
	return a.st.Load(ctx, entities, facts, docs)
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
