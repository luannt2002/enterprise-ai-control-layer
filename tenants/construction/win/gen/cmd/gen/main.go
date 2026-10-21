// Lệnh gen sinh bộ dữ liệu WIN từ world.yaml theo CONTRACT.md (cạnh world.yaml).
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/tenants/construction/win/gen"
)

func main() {
	world := flag.String("world", "", "đường dẫn world.yaml (bắt buộc)")
	out := flag.String("out", "", "thư mục đầu ra (bắt buộc)")
	profile := flag.String("profile", "functional", "hồ sơ dữ liệu nền: functional hoặc smoke")
	seed := flag.Uint64("seed", 42, "hạt giống RNG; cùng seed + hồ sơ ra cùng byte")
	flag.Parse()
	if *world == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "thiếu -world hoặc -out")
		flag.Usage()
		os.Exit(2)
	}
	start := time.Now()
	if err := gen.Generate(*world, *out, *profile, *seed); err != nil {
		fmt.Fprintln(os.Stderr, "lỗi:", err)
		os.Exit(1)
	}
	fmt.Printf("đã sinh %s (hồ sơ %s, seed %d) trong %s\n", *out, *profile, *seed, time.Since(start).Round(time.Millisecond))
}
