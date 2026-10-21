// Cổng kiến trúc: mã DÙNG CHUNG không được rẽ nhánh theo tên tenant (CLAUDE.md §16).
// Khác biệt tenant nằm ở pack (tenants/) hoặc domain module (domain/) — hai chỗ đó được
// mang tên tenant, phần còn lại thì không.
package eacl_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// sharedRoots là các cây mã dùng chung cho mọi tenant.
var sharedRoots = []string{"core", "data", "intelligence", "execution", "reliability", "governance", "observability", "platform", "simulations", "cmd"}

// tenantMarkers là tên/slug tenant tham chiếu. Thêm tenant mới thì thêm vào đây.
var tenantMarkers = []string{"abc-infra", "xyz-sports", "construction", "sports"}

func TestSharedCodeHasNoTenantLiterals(t *testing.T) {
	var hits []string
	for _, root := range sharedRoots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if d == nil { // cây chưa tồn tại (chưa có mã) thì bỏ qua
					return filepath.SkipDir
				}
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			hits = append(hits, tenantLiterals(t, path)...)
			return nil
		})
		if err != nil {
			t.Fatalf("duyệt %s: %v", root, err)
		}
	}
	for _, h := range hits {
		t.Errorf("mã dùng chung mang tên tenant: %s", h)
	}
}

// tenantLiterals trả các chuỗi literal (không tính comment) có chứa tên tenant.
func tenantLiterals(t *testing.T, path string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("đọc %s: %v", path, err)
	}
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		s, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		for _, m := range tenantMarkers {
			if strings.Contains(strings.ToLower(s), m) {
				out = append(out, fset.Position(lit.Pos()).String()+" "+lit.Value)
			}
		}
		return true
	})
	return out
}

// Cổng G5 (docs/04 mục 6): tenant chỉ đến từ Principal đã xác thực, nên KHÔNG kiểu yêu cầu nào
// (tên kết thúc bằng "Request") được mang trường tenant — kể cả nhúng nguyên Principal, hay một
// trường tên khác nhưng thẻ JSON là tenant.
func TestRequestTypesHaveNoTenantField(t *testing.T) {
	var hits []string
	for _, root := range sharedRoots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if d == nil {
					return filepath.SkipDir
				}
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			hits = append(hits, tenantFieldsInRequests(t, path)...)
			return nil
		})
		if err != nil {
			t.Fatalf("duyệt %s: %v", root, err)
		}
	}
	for _, h := range hits {
		t.Errorf("kiểu yêu cầu mang tenant (tenant phải đến từ Principal): %s", h)
	}
}

func tenantFieldsInRequests(t *testing.T, path string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("đọc %s: %v", path, err)
	}
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || !strings.HasSuffix(ts.Name.Name, "Request") {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			return true
		}
		for _, field := range st.Fields.List {
			var names []string
			for _, id := range field.Names {
				names = append(names, id.Name)
			}
			typ := fmt.Sprint(field.Type)
			if len(names) == 0 { // trường nhúng: tên là tên kiểu
				names = append(names, typ)
			}
			tag := ""
			if field.Tag != nil {
				tag = field.Tag.Value
			}
			for _, name := range names {
				if strings.Contains(strings.ToLower(name+" "+tag), "tenant") || strings.Contains(typ, "Principal") {
					out = append(out, fmt.Sprintf("%s %s.%s", fset.Position(field.Pos()), ts.Name.Name, name))
				}
			}
		}
		return true
	})
	return out
}
