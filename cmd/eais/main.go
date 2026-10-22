// Lệnh eais là ứng dụng EAIS: MỘT binary, các lệnh con. Nó chỉ lắp ráp những gì có sẵn (luồng hỏi pipeline.Ask, bộ nạp ingestion,
// kho pgstore, vết traces) thành thứ chạy được và quan sát được; không thêm luật nào vào câu trả lời. Mặc định KHÔNG gọi model: luồng
// hỏi dùng model extractive tất định, và không có cờ hay đường nào tới model trả tiền (docs/PROMPT-BUILD-EAIS.md §9.3).
//
//	eais migrate                                          áp migration (vai quản trị: EAIS_PG_ADMIN_DSN)
//	eais ingest   -pack P -data D                         nạp dữ liệu nguồn của tenant của pack P (vai ứng dụng: EAIS_PG_DSN)
//	eais serve    -dev -pack P -world W -data D           chạy HTTP (chỉ chế độ dev, chỉ loopback)
//	eais trace    -tenant T <trace_id>                    in bảng vết S0→S10 của một lượt
//	eais trace diff -tenant T <id1> <id2>                 chỉ bước ĐẦU TIÊN mà hai lượt cho kết quả khác nhau
//	eais quarantine -tenant T <mã lần nạp>                báo cáo bản ghi cách ly của một lần nạp
//
// Cờ đứng TRƯỚC đối số (quy ước gói flag). Mã thoát: 0 xong · 1 không tìm thấy / hai vết khác nhau / lỗi chạy · 2 dùng sai hoặc cấu hình sai.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

const usageText = `eais — ứng dụng EAIS (Enterprise AI Systems)

  eais migrate                                 áp migration (cần EAIS_PG_ADMIN_DSN)
  eais ingest -pack P -data D                  nạp dữ liệu nguồn của tenant của pack (cần EAIS_PG_DSN)
  eais serve -dev -pack P -world W -data D     chạy HTTP; chỉ -dev, chỉ loopback [-addr 127.0.0.1:8080] [-now world|RFC3339]
  eais trace -tenant T <trace_id>              in bảng vết S0→S10 của một lượt
  eais trace diff -tenant T <id1> <id2>        bước đầu tiên hai lượt cho kết quả khác nhau
  eais quarantine -tenant T <mã lần nạp>       báo cáo bản ghi cách ly của một lần nạp

Cờ đứng trước đối số. DSN: EAIS_PG_ADMIN_DSN (vai quản trị, chỉ cho migrate), EAIS_PG_DSN (vai ứng dụng eais_app, cho mọi lệnh khác).
`

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usageText)
		return 2
	}
	ctx := context.Background()
	switch args[0] {
	case "migrate":
		return cmdMigrate(ctx, args[1:], stdout, stderr)
	case "ingest":
		return cmdIngest(ctx, args[1:], stdout, stderr)
	case "serve":
		return cmdServe(args[1:], stdout, stderr)
	case "trace":
		if len(args) > 1 && args[1] == "diff" {
			return cmdTraceDiff(ctx, args[2:], stdout, stderr)
		}
		return cmdTrace(ctx, args[1:], stdout, stderr)
	case "quarantine":
		return cmdQuarantine(ctx, args[1:], stdout, stderr)
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usageText)
		return 0
	}
	fmt.Fprintf(stderr, "lệnh lạ %q\n\n%s", args[0], usageText)
	return 2
}

func newFlags(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

// openStore mở kho bằng vai ứng dụng (EAIS_PG_DSN) và kiểm kho sẵn sàng (đã migrate, đủ quyền).
func openStore(ctx context.Context, stderr io.Writer) (*pgstore.Store, bool) {
	_, dsn := pgstore.EnvDSN()
	if dsn == "" {
		fmt.Fprintln(stderr, "thiếu EAIS_PG_DSN (vai ứng dụng eais_app; xem deploy/docker-compose.yml)")
		return nil, false
	}
	st, err := pgstore.Open(ctx, dsn)
	if err != nil {
		fmt.Fprintln(stderr, "không mở được kho:", traces.Sanitize(err.Error()))
		return nil, false
	}
	if err := st.Ready(ctx); err != nil {
		st.Close()
		fmt.Fprintln(stderr, traces.Sanitize(err.Error()))
		return nil, false
	}
	return st, true
}

func cmdMigrate(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if err := newFlags("migrate", stderr).Parse(args); err != nil {
		return 2
	}
	admin, _ := pgstore.EnvDSN()
	if admin == "" {
		fmt.Fprintln(stderr, "thiếu EAIS_PG_ADMIN_DSN (vai quản trị; xem deploy/docker-compose.yml)")
		return 2
	}
	if err := pgstore.Migrate(ctx, admin); err != nil {
		fmt.Fprintln(stderr, "migrate lỗi:", traces.Sanitize(err.Error()))
		return 1
	}
	fmt.Fprintln(stdout, "migration đã áp (các tệp đã áp trước đó được bỏ qua)")
	return 0
}

func cmdIngest(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := newFlags("ingest", stderr)
	var cfg config
	fs.StringVar(&cfg.Pack, "pack", "", "thư mục pack của tenant (bắt buộc)")
	fs.StringVar(&cfg.Data, "data", "", "thư mục gốc dữ liệu, chứa <tenant>/ (bắt buộc)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if cfg.Pack == "" || cfg.Data == "" || fs.NArg() != 0 {
		fmt.Fprintln(stderr, "ingest: cần -pack và -data, không có đối số thừa")
		return 2
	}
	st, ok := openStore(ctx, stderr)
	if !ok {
		return 1
	}
	defer st.Close()
	a, err := newApp(cfg, st, slog.New(slog.NewTextHandler(stderr, nil)))
	if err != nil {
		fmt.Fprintln(stderr, "ingest:", traces.Sanitize(err.Error()))
		return 1
	}
	rep, err := a.ingest(ctx)
	if err != nil {
		fault, msg := traces.SanitizeError(err)
		fmt.Fprintf(stderr, "ingest lỗi (%s): %s\n", fault, msg)
		if rep.LoadID != "" {
			fmt.Fprintf(stderr, "(bản ghi cách ly của lần nạp %s có thể đã lưu; chạy lại là an toàn — cùng mã, không nhân đôi)\n", rep.LoadID)
		}
		return 1
	}
	fmt.Fprintf(stdout, "NẠP %s: %d quan sát, %d cách ly (%d ghi mới, %d đã có từ lần nạp cùng mã)\n", a.tenantID(), rep.Observations, rep.Quarantined,
		rep.NewQuarantine, rep.Quarantined-rep.NewQuarantine)
	fmt.Fprintf(stdout, "mã lần nạp: %s\nphiên bản dữ liệu: %s\nphiên bản pack: %s\nthời gian: %d ms\nxem cách ly: eais quarantine -tenant %s %s\n",
		rep.LoadID, rep.DataVersion, rep.PackVersion, rep.ElapsedMs, a.tenantID(), rep.LoadID)
	return 0
}

// cmdServe chạy HTTP. Chưa có cơ chế xác thực cho môi trường thật, nên serve CHỈ chạy với -dev: token dev suy từ world.yaml, địa chỉ
// nghe phải là loopback (kiểm hai lần: theo chuỗi đưa vào, và theo địa chỉ hệ điều hành thực sự gắn), Host phải là loopback.
func cmdServe(args []string, stdout, stderr io.Writer) int {
	fs := newFlags("serve", stderr)
	var cfg config
	addr := fs.String("addr", "127.0.0.1:8080", "địa chỉ nghe; chế độ -dev chỉ nhận loopback")
	dev := fs.Bool("dev", false, "bật chế độ DEV: token dev công khai suy từ world.yaml (CHỈ DEV; bắt buộc hiện nay)")
	fs.StringVar(&cfg.Pack, "pack", "", "thư mục pack của tenant (bắt buộc)")
	fs.StringVar(&cfg.World, "world", "", "world.yaml: nguồn principal dev (bắt buộc)")
	fs.StringVar(&cfg.Data, "data", "", "thư mục gốc dữ liệu, chứa <tenant>/ (bắt buộc)")
	fs.StringVar(&cfg.Now, "now", "", `đồng hồ: rỗng = thật · "world" = ngày tham chiếu của world.yaml lúc 10:00 · hoặc RFC3339`)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if !*dev {
		fmt.Fprintln(stderr, "serve: chưa có cơ chế xác thực cho môi trường thật (đang chờ quyết định) — chỉ chạy được với -dev, và -dev chỉ nghe loopback")
		return 2
	}
	if err := checkListenAddr(*addr); err != nil {
		fmt.Fprintln(stderr, "serve:", err)
		return 2
	}
	if cfg.Pack == "" || cfg.World == "" || cfg.Data == "" || fs.NArg() != 0 {
		fmt.Fprintln(stderr, "serve: cần -pack, -world và -data, không có đối số thừa")
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log := slog.New(slog.NewTextHandler(stderr, nil))
	w, err := loadWorld(cfg.World)
	if err == nil {
		var principals map[string]contracts.Principal
		if principals, err = devPrincipals(w); err == nil {
			return serveDev(ctx, cfg, *addr, principals, log, stdout, stderr)
		}
	}
	fmt.Fprintln(stderr, "serve:", traces.Sanitize(err.Error()))
	return 2
}

func serveDev(ctx context.Context, cfg config, addr string, principals map[string]contracts.Principal, log *slog.Logger, stdout, stderr io.Writer) int {
	st, ok := openStore(ctx, stderr)
	if !ok {
		return 1
	}
	defer st.Close()
	a, err := newApp(cfg, st, log)
	if err != nil {
		fmt.Fprintln(stderr, "serve:", traces.Sanitize(err.Error()))
		return 1
	}
	srv := newServer(a, principals, log)
	srv.hostGuard = true
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintln(stderr, "serve:", err)
		return 1
	}
	if tcp, ok := ln.Addr().(*net.TCPAddr); !ok || !tcp.IP.IsLoopback() { // địa chỉ thực sự gắn, không chỉ chuỗi đưa vào
		ln.Close()
		fmt.Fprintf(stderr, "serve: địa chỉ thực sự gắn %s không phải loopback — từ chối\n", ln.Addr())
		return 2
	}
	log.Warn("CHẾ ĐỘ DEV — CHỈ DEV: token dev là thông tin công khai (dev-<tenant>-<principal>), chỉ nghe loopback, không dùng ngoài máy dev",
		"tenant", a.tenantID(), "principals", len(principals), "model", "extractive (không gọi model trả tiền)")
	fmt.Fprintf(stdout, "đang nghe http://%s\n", ln.Addr())
	hs := &http.Server{Handler: srv.handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
		MaxHeaderBytes: 16 << 10, ErrorLog: slog.NewLogLogger(log.Handler(), slog.LevelWarn)}
	done := make(chan error, 1)
	go func() { done <- hs.Serve(ln) }()
	select {
	case err := <-done:
		fmt.Fprintln(stderr, "serve:", err)
		return 1
	case <-ctx.Done():
	}
	shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := hs.Shutdown(shutCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(stderr, "serve: tắt chưa gọn:", err)
	}
	c := &srv.cnt
	log.Info("tổng kết", "unauthenticated", c.Unauthenticated.Load(), "trace_save_failed", c.TraceSaveFailed.Load(), "usage_save_failed", c.UsageSaveFailed.Load(),
		"ingest_denied", c.IngestDenied.Load(), "host_rejected", c.HostRejected.Load())
	return 0
}
