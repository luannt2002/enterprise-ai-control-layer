package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// R2-02: phiên bản mã trong vết và trong mã lần nạp không được là "(devel)". Các test dưới đây khoá hàm CHỌN phiên bản cho cả ba nguồn
// (ldflags, VCS do go build ghi, băm chính binary) và thứ tự ưu tiên của chúng.

func TestPickCodeVersion(t *testing.T) {
	hash := func() (string, error) { return "abc123", nil }
	noHash := func() (string, error) { return "", errors.New("không đọc được binary") }
	bi := func(mainVersion string, settings ...debug.BuildSetting) *debug.BuildInfo {
		return &debug.BuildInfo{Main: debug.Module{Version: mainVersion}, Settings: settings}
	}
	rev := func(r string) debug.BuildSetting { return debug.BuildSetting{Key: "vcs.revision", Value: r} }
	modified := func(m string) debug.BuildSetting { return debug.BuildSetting{Key: "vcs.modified", Value: m} }
	const git = "0123456789abcdef0123456789abcdef01234567"

	for _, c := range []struct {
		name   string
		linked string
		bi     *debug.BuildInfo
		self   func() (string, error)
		want   string
	}{
		// nguồn 1: -ldflags
		{"ldflags", "linked-1", nil, hash, "linked-1"},
		{"ldflags thắng bản sửa VCS và phiên bản module", "linked-1", bi("v1.2.3", rev(git), modified("true")), hash, "linked-1"},
		{"ldflags được cắt khoảng trắng và xuống dòng", " linked-1\n", bi("(devel)"), hash, "linked-1"},
		{"ldflags chỉ có khoảng trắng coi như không có", " \n", bi("(devel)"), hash, binHashPrefix + "abc123"},
		// nguồn 2: VCS do go build ghi
		{"VCS sạch", "", bi("(devel)", rev(git), modified("false")), noHash, git},
		{"VCS bẩn", "", bi("(devel)", rev(git), modified("true")), noHash, git + "+dirty"},
		{"VCS thắng phiên bản module", "", bi("v0.0.0-20261022000000-0123456789ab", rev(git)), noHash, git},
		{"VCS có khoá mà bản sửa rỗng thì không tính", "", bi("(devel)", rev(""), modified("true")), hash, binHashPrefix + "abc123"},
		// phiên bản module thật (vd go install pkg@v1.2.3)
		{"phiên bản module thật", "", bi("v1.2.3"), noHash, "v1.2.3"},
		// nguồn 3: băm chính binary — chỗ trám cho git worktree liên kết / cây không có .git
		{"không VCS, không ldflags: băm binary", "", bi("(devel)"), hash, binHashPrefix + "abc123"},
		{"phiên bản module rỗng", "", bi(""), hash, binHashPrefix + "abc123"},
		{"binary không mang build info", "", nil, hash, binHashPrefix + "abc123"},
		// không có nguồn nào
		{"không đọc được cả binary", "", bi("(devel)"), noHash, "(unknown)"},
		{"băm rỗng không được coi là phiên bản", "", bi("(devel)"), func() (string, error) { return "", nil }, "(unknown)"},
	} {
		got := pickCodeVersion(c.linked, c.bi, c.self)
		if got != c.want {
			t.Errorf("%s: pickCodeVersion = %q, muốn %q", c.name, got, c.want)
		}
		if got == "" || got == "(devel)" {
			t.Errorf("%s: trả %q — phiên bản mã không được rỗng hay \"(devel)\" (vết không phân biệt được các mốc mã)", c.name, got)
		}
	}
}

// Băm binary là băm THẬT của tệp đang chạy (không phải băm của đường dẫn hay của một chuỗi cố định), và hai bản khác nhau cho phiên bản khác
// nhau: kiểm trên chính binary của test và trên một tệp tự tạo qua pickCodeVersion.
func TestSelfSHA256(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	want := hex.EncodeToString(sum[:])
	got, err := selfSHA256()
	if err != nil || got != want {
		t.Fatalf("selfSHA256 = %q, %v; băm sha256 của %s = %q", got, err, exe, want)
	}
	if len(got) != 64 {
		t.Errorf("băm %q không phải sha256 hex 64 ký tự", got)
	}
}

// buildRevision của chính binary test: không rỗng, không "(devel)", ổn định giữa các lần gọi (tính một lần), và nếu là dạng băm binary thì phần
// sau tiền tố đúng là băm của binary đang chạy. Test binary không mang VCS nên ở đây đi qua nguồn 3 — chính là ca "git worktree liên kết".
func TestBuildRevisionThat(t *testing.T) {
	v := buildRevision()
	if v == "" || v == "(devel)" || v == "(unknown)" {
		t.Fatalf("buildRevision() = %q", v)
	}
	if v != buildRevision() {
		t.Error("buildRevision không ổn định giữa hai lần gọi")
	}
	if rest, ok := strings.CutPrefix(v, binHashPrefix); ok {
		if want, _ := selfSHA256(); rest != want {
			t.Errorf("buildRevision = %q: phần sau tiền tố ≠ băm của binary đang chạy %q", v, want)
		}
	}
}

// Bộ làm sạch bí mật của vết không được che mất phiên bản mã: một chuỗi hex dài hay có dấu hai chấm / dấu cộng dễ bị nhầm với khoá. Kiểm cả ba
// dạng phiên bản mà ứng dụng có thể ghi.
func TestPhienBanMaKhongBiLamSachChe(t *testing.T) {
	for name, v := range map[string]string{
		"mã git":              "0123456789abcdef0123456789abcdef01234567",
		"mã git bẩn":          "0123456789abcdef0123456789abcdef01234567+dirty",
		"băm binary":          binHashPrefix + strings.Repeat("ab", 32),
		"phiên bản module":    "v0.0.0-20261022000000-0123456789ab+dirty",
		"băm binary của test": buildRevision(),
	} {
		if got := traces.Sanitize(v); got != v {
			t.Errorf("%s: Sanitize(%q) = %q — vết sẽ lưu phiên bản mã đã bị che", name, v, got)
		}
	}
}

// -X của trình liên kết lặng lẽ bỏ qua biến không tồn tại hay không phải chuỗi không khởi tạo; script và mã Go phải cùng nói về MỘT biến.
// Test này đọc tên biến trong scripts/chay-thu.sh rồi đòi mã nguồn của package main khai đúng biến đó, kiểu string, không giá trị khởi tạo.
func TestBuildVersionLaBienLienKet(t *testing.T) {
	raw, err := os.ReadFile(repoPath("scripts/chay-thu.sh"))
	if err != nil {
		t.Fatal(err)
	}
	var code []string // bỏ dòng chú thích: phần đầu script có nhắc tới -X main.buildVersion=… để giải thích
	for _, l := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(l), "#") {
			code = append(code, l)
		}
	}
	m := regexp.MustCompile(`-X main\.(\w+)=`).FindAllStringSubmatch(strings.Join(code, "\n"), -1)
	if len(m) != 1 {
		t.Fatalf("scripts/chay-thu.sh nhúng %d biến bằng -X main.<tên>= (ngoài chú thích), muốn đúng 1", len(m))
	}
	name := m[0][1]
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("không đọc được mã của cmd/eais: %v", err)
	}
	found := false
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		tree, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range tree.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, sp := range gd.Specs {
				vs := sp.(*ast.ValueSpec)
				for _, id := range vs.Names {
					if id.Name != name {
						continue
					}
					if typ, ok := vs.Type.(*ast.Ident); !ok || typ.Name != "string" || len(vs.Values) != 0 {
						t.Errorf("%s: biến %s phải khai là `var %s string`, không khởi tạo (-X chỉ nhận biến chuỗi không khởi tạo hoặc khởi tạo bằng hằng chuỗi; test đòi dạng chặt nhất)", f, name, name)
					}
					found = true
				}
			}
		}
	}
	if !found {
		t.Errorf("scripts/chay-thu.sh nhúng main.%s mà package main không khai biến đó ở mức gói — -X sẽ không làm gì", name)
	}
}
