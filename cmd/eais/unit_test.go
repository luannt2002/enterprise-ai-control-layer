package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/router"
	"go.yaml.in/yaml/v3"
)

func TestBearerToken(t *testing.T) {
	for _, c := range []struct {
		h    string
		want string
		ok   bool
	}{
		{"Bearer abc", "abc", true}, {"bearer abc", "abc", true}, {"BEARER abc", "abc", true}, {"Bearer   abc  ", "abc", true},
		{"", "", false}, {"Bearer", "", false}, {"Bearer ", "", false}, {"Bearer    ", "", false}, {"Basic abc", "", false},
		{"Token abc", "", false}, {"abc", "", false}, {"Bearer a b", "", false}, {"Bearerabc", "", false},
	} {
		if got, ok := bearerToken(c.h); got != c.want || ok != c.ok {
			t.Errorf("bearerToken(%q) = %q, %v; muốn %q, %v", c.h, got, ok, c.want, c.ok)
		}
	}
}

// Địa chỉ nghe ở chế độ dev: chỉ loopback. Các dạng "mở ra mọi giao diện" và tên máy lạ đều bị từ chối.
func TestCheckListenAddr(t *testing.T) {
	for addr, ok := range map[string]bool{
		"127.0.0.1:8080": true, "127.0.0.1:0": true, "[::1]:8080": true, "localhost:8080": true, "127.1.2.3:80": true,
		":8080": false, "0.0.0.0:8080": false, "[::]:8080": false, "192.168.1.5:8080": false, "10.0.0.1:80": false,
		"example.com:80": false, "8080": false, "": false, "127.0.0.1": false,
	} {
		if err := checkListenAddr(addr); (err == nil) != ok {
			t.Errorf("checkListenAddr(%q) = %v, muốn ok=%v", addr, err, ok)
		}
	}
}

func TestLoopbackHost(t *testing.T) {
	for h, ok := range map[string]bool{
		"127.0.0.1:8080": true, "127.0.0.1": true, "localhost": true, "localhost:3000": true, "[::1]:8080": true, "[::1]": true, "::1": true,
		"evil.example": false, "evil.example:8080": false, "127.0.0.1.evil.example": false, "10.0.0.1:80": false, "": false, "0.0.0.0": false,
	} {
		if got := loopbackHost(h); got != ok {
			t.Errorf("loopbackHost(%q) = %v, muốn %v", h, got, ok)
		}
	}
}

// Token dev suy từ world.yaml thật: đúng tenant, vai, phạm vi; hai tenant có principal riêng; trùng token bị từ chối.
func TestDevPrincipalsTuWorld(t *testing.T) {
	w, err := loadWorld(repoPath("tenants/construction/win/world.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	ps, err := devPrincipals(w)
	if err != nil {
		t.Fatal(err)
	}
	dir, ok := ps["dev-abc-infra-u-director"]
	if !ok || dir.TenantID != "abc-infra" || dir.ID != "u-director" || len(dir.Roles) != 1 || dir.Roles[0] != "director" || len(dir.Attributes) != 0 {
		t.Errorf("u-director = %+v (có %v)", dir, ok)
	}
	eng := ps["dev-abc-infra-u-eng-a"]
	if eng.Roles[0] != "site_engineer" || eng.Attributes["entity_scope"] != "PRJ-A" {
		t.Errorf("u-eng-a = %+v", eng)
	}
	if b := ps["dev-def-build-u-b-director"]; b.TenantID != "def-build" {
		t.Errorf("u-b-director = %+v", b)
	}
	for tok := range ps {
		if !strings.HasPrefix(tok, devTokenPrefix) {
			t.Errorf("token %q không mang tiền tố dev", tok)
		}
	}
	var dup worldFile
	if err := yaml.Unmarshal([]byte("tenants:\n  - {id: a-b, principals: [{id: c, role: r}]}\n  - {id: a, principals: [{id: b-c, role: r}]}\n"), &dup); err != nil {
		t.Fatal(err)
	}
	if _, err := devPrincipals(dup); err == nil || !strings.Contains(err.Error(), "cùng token dev") {
		t.Errorf("hai principal cùng token phải bị từ chối, được %v", err)
	}
	if _, err := devPrincipals(worldFile{}); err == nil {
		t.Error("world không có principal nào phải là lỗi")
	}
}

// fullAccess theo policy THẬT của pack: chỉ director; không nêu tên vai nào trong mã (xem các ca tổng hợp bên dưới).
func TestFullAccessTheoPolicy(t *testing.T) {
	rt, err := tenant.Load(repoPath("tenants/construction/pack"))
	if err != nil {
		t.Fatal(err)
	}
	p := func(roles ...string) contracts.Principal {
		return contracts.Principal{ID: "u", TenantID: "t", Roles: roles}
	}
	for role, want := range map[string]bool{"director": true, "accountant": false, "project_manager": false, "site_engineer": false, "driver": false, "khong-co-trong-policy": false} {
		if got := fullAccess(rt.Policy, p(role)); got != want {
			t.Errorf("fullAccess(%s) = %v, muốn %v", role, got, want)
		}
	}
	if !fullAccess(rt.Policy, p("driver", "director")) || fullAccess(rt.Policy, p()) {
		t.Error("nhiều vai: có director thì toàn quyền; không vai thì không")
	}
	pol := func(roles map[string]router.RoleDef) router.Policy { return router.Policy{Roles: roles} }
	all := func(scope string, caps ...string) router.RoleDef {
		return router.RoleDef{Capabilities: caps, Scope: scope}
	}
	for name, c := range map[string]struct {
		pol  router.Policy
		role string
		want bool
	}{
		"mỗi vai một capability, không ai nắm hết": {pol(map[string]router.RoleDef{"a": all("all", "x"), "b": all("all", "y")}), "a", false},
		"vai nắm hết, scope all":                   {pol(map[string]router.RoleDef{"a": all("all", "x", "y"), "b": all("all", "y")}), "a", true},
		"vai nắm hết nhưng scope attribute":        {pol(map[string]router.RoleDef{"a": all("attribute", "x", "y")}), "a", false},
		"policy rỗng":                              {pol(nil), "a", false},
		"pack khác, tên vai khác":                  {pol(map[string]router.RoleDef{"owner": all("all", "read", "write")}), "owner", true},
	} {
		if got := fullAccess(c.pol, p(c.role)); got != c.want {
			t.Errorf("%s: fullAccess = %v, muốn %v", name, got, c.want)
		}
	}
}

func TestDataVersion(t *testing.T) {
	mk := func(files map[string]string) string {
		dir := t.TempDir()
		for name, body := range files {
			path := filepath.Join(dir, name)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return dir
	}
	base := map[string]string{"a.csv": "1,2", "docs/x.md": "# x"}
	v0, err := dataVersion(mk(base))
	if err != nil || len(v0) != 64 {
		t.Fatalf("dataVersion = %q, %v", v0, err)
	}
	if v1, _ := dataVersion(mk(base)); v1 != v0 {
		t.Error("cùng nội dung ở thư mục khác phải cùng phiên bản")
	}
	for name, files := range map[string]map[string]string{
		"đổi một byte":   {"a.csv": "1,3", "docs/x.md": "# x"},
		"thêm tệp":       {"a.csv": "1,2", "docs/x.md": "# x", "b": ""},
		"đổi tên tệp":    {"a2.csv": "1,2", "docs/x.md": "# x"},
		"chuyển thư mục": {"a.csv": "1,2", "docs2/x.md": "# x"},
		"bớt tệp":        {"a.csv": "1,2"},
	} {
		if v, err := dataVersion(mk(files)); err != nil || v == v0 {
			t.Errorf("%s: phiên bản %q phải khác %q (lỗi %v)", name, v, v0, err)
		}
	}
	for name, dir := range map[string]string{"rỗng": t.TempDir(), "không có": filepath.Join(t.TempDir(), "khong-co")} {
		if _, err := dataVersion(dir); err == nil || contracts.FaultOf(err) != contracts.Infra("data_dir") {
			t.Errorf("thư mục %s: muốn lỗi nhãn infra:data_dir, được %v", name, err)
		}
	}
}

func TestLoadIDOf(t *testing.T) {
	id := loadIDOf("d", "p", "c")
	if !regexp.MustCompile(`^load-[0-9a-f]{16}$`).MatchString(id) || id != loadIDOf("d", "p", "c") {
		t.Fatalf("mã lần nạp %q sai dạng hoặc không tất định", id)
	}
	for name, other := range map[string]string{"dữ liệu": loadIDOf("d2", "p", "c"), "pack": loadIDOf("d", "p2", "c"), "mã": loadIDOf("d", "p", "c2"),
		"ranh giới": loadIDOf("dp", "", "c")} {
		if other == id {
			t.Errorf("đổi %s mà mã lần nạp không đổi", name)
		}
	}
}

func TestTenantSlug(t *testing.T) {
	for id, ok := range map[string]bool{"abc-infra": true, "e2e-a-123": true, "a": true, "a_b": true, "": false, "../x": false, "a/b": false, "A": false,
		"-a": false, "a b": false, "a\x00": false, strings.Repeat("a", 64): false} {
		if validTenantID(id) != ok {
			t.Errorf("validTenantID(%q) = %v, muốn %v", id, !ok, ok)
		}
	}
}

// CỔNG: ứng dụng này KHÔNG có đường gọi model trả tiền. Mã không-test của cmd/eais chỉ được dùng models.Extractive (và kiểu Gateway /
// Request), không chuỗi nào nhắc biến môi trường hay địa chỉ của nhà cung cấp. Thêm đường gọi model thật vào đây là đổi luật chi tiền
// (docs/PROMPT-BUILD-EAIS.md §9.3): phải đi qua cổng, không lẻn qua một cờ.
func TestKhongCoDuongGoiModelThat(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("không đọc được mã của cmd/eais: %v", err)
	}
	allowed := map[string]bool{"Extractive": true, "Gateway": true, "Request": true}
	forbiddenText := []string{"EAIS_MODEL", "ANTHROPIC", "anthropic.com", "OPENAI", "api.openai"}
	sawExtractive := false
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		tree, err := parser.ParseFile(fset, f, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(tree, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if id, ok := x.X.(*ast.Ident); ok && id.Name == "models" {
					if !allowed[x.Sel.Name] {
						t.Errorf("%s: models.%s — đường tới model ngoài Extractive", fset.Position(x.Pos()), x.Sel.Name)
					}
					sawExtractive = sawExtractive || x.Sel.Name == "Extractive"
				}
			case *ast.BasicLit:
				if x.Kind != token.STRING {
					return true
				}
				s, _ := strconv.Unquote(x.Value)
				for _, bad := range forbiddenText {
					if strings.Contains(strings.ToUpper(s), strings.ToUpper(bad)) {
						t.Errorf("%s: chuỗi nhắc tới model / nhà cung cấp trả tiền: %q", fset.Position(x.Pos()), bad)
					}
				}
			}
			return true
		})
	}
	if !sawExtractive {
		t.Error("cổng vô nghĩa: không thấy models.Extractive trong mã — kiểm lại cách quét")
	}
}

// Dùng sai thì thoát 2 và nói rõ, KHÔNG chạm kho (các kiểm tra tham số đứng trước việc mở kết nối).
func TestRunDungSai(t *testing.T) {
	t.Setenv("EAIS_PG_DSN", "")
	t.Setenv("EAIS_PG_ADMIN_DSN", "")
	for _, c := range []struct {
		args []string
		code int
		msg  string
	}{
		{nil, 2, "ứng dụng EAIS"},
		{[]string{"lenh-la"}, 2, "lệnh lạ"},
		{[]string{"help"}, 0, ""},
		{[]string{"serve"}, 2, "chỉ chạy được với -dev"},
		{[]string{"serve", "-dev", "-addr", "0.0.0.0:8080", "-pack", "p", "-world", "w", "-data", "d"}, 2, "không phải loopback"},
		{[]string{"serve", "-dev", "-addr", ":8080", "-pack", "p", "-world", "w", "-data", "d"}, 2, "không phải loopback"},
		{[]string{"serve", "-dev", "-addr", "127.0.0.1:0"}, 2, "cần -pack, -world và -data"},
		{[]string{"serve", "-dev", "-addr", "127.0.0.1:0", "-pack", "p", "-world", "khong-co.yaml", "-data", "d"}, 2, "world"},
		{[]string{"ingest"}, 2, "cần -pack và -data"},
		{[]string{"migrate"}, 2, "EAIS_PG_ADMIN_DSN"},
		{[]string{"trace"}, 2, "cần -tenant"},
		{[]string{"trace", "-tenant", "t"}, 2, "đúng 1 đối số"},
		{[]string{"trace", "diff", "-tenant", "t", "a"}, 2, "đúng 2 đối số"},
		{[]string{"trace", "-tenant", "../x", "id"}, 2, "không hợp lệ"},
		{[]string{"quarantine", "-tenant", "t"}, 2, "đúng 1 đối số"},
		{[]string{"trace", "-tenant", "t", "id"}, 2, "EAIS_PG_DSN"}, // hợp lệ về tham số, nhưng chưa có DSN: cấu hình sai
	} {
		var out, errb bytes.Buffer
		code := run(c.args, &out, &errb)
		if code != c.code || !strings.Contains(out.String()+errb.String(), c.msg) {
			t.Errorf("run(%v) = %d, muốn %d có %q:\n%s%s", c.args, code, c.code, c.msg, out.String(), errb.String())
		}
	}
}
