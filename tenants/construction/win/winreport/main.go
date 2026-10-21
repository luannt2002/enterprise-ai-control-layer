// winreport in báo cáo WIN MVP trên một bộ dữ liệu do bộ sinh xuất ra (docs/04-core-mvp.md §6).
//
//	go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out /tmp/win -profile functional -seed 42
//	go run ./tenants/construction/win/winreport -data /tmp/win
//
// Thoát 0 chỉ khi G1 khớp hết, mọi ca đạt và cổng đếm G6 đạt.
package main

import (
	"context"
	"flag"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/evals"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
	"github.com/luannt2002/enterprise-ai-control-layer/tenants/construction/win/winrun"
)

func main() {
	cfg := winrun.Config{}
	flag.StringVar(&cfg.Data, "data", "", "thư mục dữ liệu do bộ sinh xuất ra")
	flag.StringVar(&cfg.Pack, "pack", "tenants/construction/pack", "pack của tenant")
	flag.StringVar(&cfg.World, "world", "tenants/construction/win/world.yaml", "world.yaml")
	flag.StringVar(&cfg.CasesGlob, "cases", "tenants/construction/win/cases-*.yaml", "các tệp ca WIN")
	flag.StringVar(&cfg.Tenant, "tenant", "abc-infra", "tenant được chấm")
	verbose := flag.Bool("v", false, "in cả ca đạt")
	usePG := flag.Bool("pg", false, "đọc dữ liệu qua PostgreSQL (EAIS_PG_ADMIN_DSN + EAIS_PG_DSN, xem deploy/docker-compose.yml)")
	flag.Parse()
	if cfg.Data == "" {
		fmt.Fprintln(os.Stderr, "thiếu -data")
		os.Exit(2)
	}
	path := "bộ nhớ"
	if *usePG {
		admin, app := pgstore.EnvDSN()
		if admin == "" || app == "" {
			fmt.Fprintln(os.Stderr, "-pg cần EAIS_PG_ADMIN_DSN và EAIS_PG_DSN")
			os.Exit(2)
		}
		ctx := context.Background()
		if err := pgstore.Migrate(ctx, admin); err != nil {
			fmt.Fprintln(os.Stderr, "LỖI migrate:", err)
			os.Exit(2)
		}
		pg, err := pgstore.Open(ctx, app)
		if err != nil {
			fmt.Fprintln(os.Stderr, "LỖI mở PG:", err)
			os.Exit(2)
		}
		defer pg.Close()
		cfg.PG, path = pg, "PostgreSQL (RLS theo tenant của Principal)"
	}
	t0 := time.Now()
	r, err := winrun.Run(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "LỖI:", err)
		os.Exit(2)
	}
	fmt.Printf("ĐƯỜNG DỮ LIỆU: %s · tổng thời gian chạy %v\n", path, time.Since(t0).Round(time.Millisecond))
	fmt.Printf("NẠP %s: %d quan sát, %d cách ly (%v)\n", cfg.Tenant, r.Observations, r.Quarantined, r.IngestTime)
	fmt.Printf("G1: %d/%d bản ghi khớp manifest\n", r.G1Total-len(r.G1Problems), r.G1Total)
	fmt.Printf("G1 metamorphic: %v biểu diễn theo bản ghi, %d lệch\n", r.Variants, len(r.VariantProblems))
	for _, p := range r.VariantProblems {
		fmt.Println("  ✗", p)
	}
	for i, p := range r.G1Problems {
		if i == 40 {
			fmt.Printf("  … và %d lỗi nữa\n", len(r.G1Problems)-40)
			break
		}
		fmt.Println("  ✗", p)
	}
	fmt.Printf("\nBỘ CA WIN (%d ca)\n%s", len(r.Cases), r.Report.Summary())
	fmt.Println("\nCA ĐỎ:")
	ok := len(r.G1Problems) == 0 && len(r.VariantProblems) == 0
	for _, c := range r.Report.Results {
		if c.Pass && !*verbose {
			continue
		}
		ok = ok && c.Pass
		mark := "ĐỎ "
		if c.Pass {
			mark = "ĐẠT"
		}
		fmt.Printf("  %s %s [%s] %s\n      → %q\n", mark, c.ID, c.FailureClass, strings.Join(c.Problems, "; "), c.Answer)
	}
	if counts := evals.WinGateCounts(r.Cases); len(counts) > 0 {
		fmt.Println("\nCỔNG ĐẾM G6 chưa đạt:", strings.Join(counts, "; "))
		ok = false
	}
	fmt.Printf("\nG6 PHÁT LẠI: %d câu đã giao có bản ghi quyết định, %d lệch khi phát lại\n", r.Decisions, len(r.ReplayProblems))
	for _, p := range r.ReplayProblems {
		fmt.Println("  ✗", p)
	}
	ok = ok && len(r.ReplayProblems) == 0 && r.Decisions > 0
	fmt.Println("phiên bản dữ liệu (sha256 manifest):", r.DataVersion)
	for _, name := range slices.Sorted(maps.Keys(r.CaseHashes)) {
		fmt.Printf("bộ ca %s sha256 %s\n", name, r.CaseHashes[name])
	}
	if !ok {
		os.Exit(1)
	}
}
