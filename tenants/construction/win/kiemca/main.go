// kiemca kiểm ĐỊNH DẠNG các tệp ca WIN mà KHÔNG chạy EACL và KHÔNG in nội dung ca (câu hỏi, kỳ vọng): dùng cho bộ giữ
// lại trước khi chạy, vì người viết mã không được đọc kỳ vọng của bộ giữ lại (ADR-0006 §3). In số ca theo lớp hỏng, số ca
// có kỳ vọng số liệu, và lỗi định dạng (mã ca + tên trường, như evals.ValidateWinSuite). Thoát 0 khi không lỗi, 1 khi có
// lỗi định dạng, 2 khi không đọc được tệp.
//
//	go run ./tenants/construction/win/kiemca -world tenants/construction/win/world.yaml -tenant abc-infra tệp-ca.yaml…
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/luannt2002/enterprise-ai-systems/evals"
	"go.yaml.in/yaml/v3"
)

func main() {
	world := flag.String("world", "tenants/construction/win/world.yaml", "world.yaml (danh sách principal theo tenant)")
	tenant := flag.String("tenant", "abc-infra", "tenant của bộ ca")
	flag.Parse()
	os.Exit(run(os.Stdout, *world, *tenant, flag.Args()))
}

// worldPrincipals là phần của world.yaml mà việc kiểm định dạng cần: id principal của từng tenant.
type worldPrincipals struct {
	Tenants []struct {
		ID         string `yaml:"id"`
		Principals []struct {
			ID string `yaml:"id"`
		} `yaml:"principals"`
	} `yaml:"tenants"`
}

func run(out io.Writer, worldPath, tenant string, files []string) int {
	if len(files) == 0 {
		fmt.Fprintln(out, "LỖI: không có tệp ca nào")
		return 2
	}
	b, err := os.ReadFile(worldPath)
	if err != nil {
		fmt.Fprintln(out, "LỖI: đọc world.yaml:", err)
		return 2
	}
	var w worldPrincipals
	if err := yaml.Unmarshal(b, &w); err != nil {
		fmt.Fprintln(out, "LỖI: world.yaml hỏng:", err)
		return 2
	}
	known := map[string]bool{}
	for _, t := range w.Tenants {
		if t.ID == tenant {
			for _, p := range t.Principals {
				known[p.ID] = true
			}
		}
	}
	if len(known) == 0 {
		fmt.Fprintf(out, "LỖI: world.yaml không có principal nào của tenant %s\n", tenant)
		return 2
	}
	perClass, seen := map[string]int{}, map[string]string{}
	total, withValue, problems := 0, 0, 0
	for _, f := range files {
		s, err := evals.LoadWinSuite(f)
		if err != nil {
			fmt.Fprintf(out, "LỖI: %s không đọc được (%v)\n", f, err)
			return 2
		}
		for _, p := range evals.ValidateWinSuite(s, tenant, known) {
			fmt.Fprintf(out, "%s: %s\n", f, p)
			problems++
		}
		for _, c := range s.Cases {
			total++
			perClass[c.FailureClass]++
			if c.Expect.Value != nil {
				withValue++
			}
			if prev, ok := seen[c.ID]; ok && prev != f {
				fmt.Fprintf(out, "%s: id %s đã có ở %s\n", f, c.ID, prev)
				problems++
			}
			seen[c.ID] = f
		}
	}
	classes := make([]string, 0, len(perClass))
	for k := range perClass {
		classes = append(classes, k)
	}
	sort.Strings(classes)
	for _, k := range classes {
		fmt.Fprintf(out, "lớp %-18s %d ca\n", k, perClass[k])
	}
	fmt.Fprintf(out, "TỔNG %d ca · có kỳ vọng số liệu %d · lỗi định dạng %d\n", total, withValue, problems)
	if problems > 0 {
		return 1
	}
	return 0
}
