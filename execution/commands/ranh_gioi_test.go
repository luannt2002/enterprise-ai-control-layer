package commands

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Cổng ranh giới (F-031): lệnh ghi không phụ thuộc chặng hỏi intelligence/ — kể cả GIÁN TIẾP — nên sửa định tuyến không đổi được quyền
// của lệnh; core/policy chỉ biết lõi (không data/, không intelligence/). Đi đồ thị import của mã không-test trong module bằng go/parser,
// tương đương `go list -deps` mà không gọi công cụ go trong test.
func TestRanhGioiImport(t *testing.T) {
	for _, c := range []struct {
		dir    string
		forbid []string
	}{
		{".", []string{"/intelligence/"}},
		{"../../core/policy", []string{"/intelligence/", "/data/"}},
	} {
		deps := moduleDeps(t, c.dir)
		if !slices.Contains(deps, modulePath+"core/contracts") { // đồ thị rỗng (đường dẫn sai) thì cổng xanh oan
			t.Fatalf("%s: không thấy core/contracts trong phụ thuộc %v", c.dir, deps)
		}
		for _, d := range deps {
			for _, f := range c.forbid {
				if strings.Contains(d, f) {
					t.Errorf("%s phụ thuộc %s (cấm %s)", c.dir, d, f)
				}
			}
		}
	}
}

const modulePath = "github.com/luannt2002/enterprise-ai-control-layer/"

// moduleDeps trả mọi gói của module mà mã không-test trong dir import, trực tiếp hay gián tiếp.
func moduleDeps(t *testing.T, dir string) []string {
	t.Helper()
	seen := map[string]bool{}
	var walk func(d string)
	walk = func(d string) {
		files, err := filepath.Glob(filepath.Join(d, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			tree, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatal(err)
			}
			for _, im := range tree.Imports {
				p, _ := strconv.Unquote(im.Path.Value)
				if !strings.HasPrefix(p, modulePath) || seen[p] {
					continue
				}
				seen[p] = true
				walk(filepath.Join("../..", strings.TrimPrefix(p, modulePath)))
			}
		}
	}
	walk(dir)
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	slices.Sort(out)
	return out
}
