package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// Các test của cổng CI (scripts/kiem-skip-test.py và .github/workflows/ci.yml). Cổng mà không có test chứng minh nó đỏ được thì chỉ là hy
// vọng: ở đây cả hai đường (xanh, đỏ) đều được chạy thật.

func runSkipGate(t *testing.T, input string, args ...string) (int, string, string) {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		t.Fatalf("cần python3 để kiểm cổng SKIP: %v", err)
	}
	in := filepath.Join(t.TempDir(), "test.json")
	if err := os.WriteFile(in, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("python3", append([]string{"-I", repoPath("scripts/kiem-skip-test.py")}, append(args, in)...)...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return code, out.String(), errb.String()
}

const (
	evPass = `{"Action":"pass","Package":"p","Test":"TestB","Elapsed":0}` + "\n"
	evSkip = `{"Action":"run","Package":"p","Test":"TestA"}` + "\n" +
		`{"Action":"output","Package":"p","Test":"TestA","Output":"    a_test.go:12: chưa có EAIS_PG_DSN\n"}` + "\n" +
		`{"Action":"skip","Package":"p","Test":"TestA","Elapsed":0}` + "\n"
	evPkgSkip = `{"Action":"skip","Package":"p/khongcotest"}` + "\n" // gói không có tệp test: KHÔNG phải test bị bỏ qua
)

func TestKiemSkipScript(t *testing.T) {
	// xanh: chỉ pass, kể cả khi có gói không có test
	if code, out, errb := runSkipGate(t, evPass+evPkgSkip); code != 0 || !strings.Contains(out, "XANH") {
		t.Errorf("chỉ pass: %d %s%s, muốn 0", code, out, errb)
	}
	// ĐỎ: một test SKIP — thoát 1 và nêu đúng test, gói, lý do
	code, out, errb := runSkipGate(t, evPass+evSkip)
	if code != 1 || !strings.Contains(out, "SKIP: p TestA: chưa có EAIS_PG_DSN") || !strings.Contains(errb, "1 test SKIP ngoài danh sách") {
		t.Errorf("có test SKIP: %d\n%s%s\nmuốn 1 và nêu TestA kèm lý do", code, out, errb)
	}
	// danh sách cho phép: khớp thì xanh, mục không còn khớp thì đỏ (danh sách thừa che lỗi sau này)
	allow := filepath.Join(t.TempDir(), "cho-phep.txt")
	os.WriteFile(allow, []byte("# lý do ghi ở đây\np TestA # cần PG của máy khác\n"), 0o644)
	if code, out, errb := runSkipGate(t, evPass+evSkip, "--cho-phep", allow); code != 0 || !strings.Contains(out, "được phép: p TestA") {
		t.Errorf("SKIP có trong danh sách cho phép: %d %s%s, muốn 0", code, out, errb)
	}
	if code, _, errb := runSkipGate(t, evPass, "--cho-phep", allow); code != 1 || !strings.Contains(errb, "THỪA: p TestA") {
		t.Errorf("mục cho phép không còn khớp: %d %s, muốn 1 (THỪA)", code, errb)
	}
	// đầu vào rỗng / hỏng không được báo "không SKIP": thoát 2
	if code, _, errb := runSkipGate(t, ""); code != 2 || !strings.Contains(errb, "không có test nào") {
		t.Errorf("đầu vào rỗng: %d %s, muốn 2", code, errb)
	}
	if code, _, _ := runSkipGate(t, "không phải JSON\n"+evPass); code != 0 {
		t.Errorf("dòng rác lẫn trong luồng không được làm đỏ cổng khi vẫn có test pass: %d", code)
	}
}

// Workflow CI nhất quán với phần còn lại của repo: cùng ảnh PG ghim digest với deploy/docker-compose.yml, đủ các bước cổng, action ghim
// theo tag đầy đủ hoặc SHA, quyền chỉ đọc, không bí mật, DSN không mang mật khẩu.
func TestWorkflowCI(t *testing.T) {
	raw, err := os.ReadFile(repoPath(".github/workflows/ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var wf struct {
		Permissions map[string]string `yaml:"permissions"`
		Jobs        map[string]struct {
			Services map[string]struct {
				Image string `yaml:"image"`
			} `yaml:"services"`
			Env   map[string]string `yaml:"env"`
			Steps []struct {
				Name string `yaml:"name"`
				Uses string `yaml:"uses"`
				Run  string `yaml:"run"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(raw, &wf); err != nil {
		t.Fatalf("ci.yml không phải YAML hợp lệ: %v", err)
	}
	job, ok := wf.Jobs["kiem"]
	if !ok {
		t.Fatal("ci.yml thiếu job kiem")
	}
	craw, err := os.ReadFile(repoPath("deploy/docker-compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var compose struct {
		Services map[string]struct {
			Image string `yaml:"image"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(craw, &compose); err != nil {
		t.Fatal(err)
	}
	want := compose.Services["pg"].Image
	if !regexp.MustCompile(`^postgres:\d+@sha256:[0-9a-f]{64}$`).MatchString(want) {
		t.Fatalf("ảnh PG của compose %q không ghim digest", want)
	}
	if got := job.Services["postgres"].Image; got != want {
		t.Errorf("ảnh PG của CI %q ≠ của compose %q (hai nơi phải ghim cùng một digest)", got, want)
	}
	if wf.Permissions["contents"] != "read" || len(wf.Permissions) != 1 {
		t.Errorf("permissions = %v, muốn chỉ contents: read", wf.Permissions)
	}
	pinned := regexp.MustCompile(`^[\w.-]+/[\w.-]+@(v\d+\.\d+\.\d+|[0-9a-f]{40})$`)
	var runs []string
	for _, s := range job.Steps {
		if s.Uses != "" && !pinned.MatchString(s.Uses) {
			t.Errorf("action %q không ghim theo tag đầy đủ vN.M.P hoặc SHA", s.Uses)
		}
		runs = append(runs, s.Run)
	}
	all := strings.Join(runs, "\n")
	for _, must := range []string{"go vet ./...", "gofmt -l", "go test -count=1 -json ./...", "scripts/kiem-skip-test.py", "TestDanhTinh", "scripts/chay-thu.sh", "poppler-utils"} {
		if !strings.Contains(all, must) {
			t.Errorf("ci.yml thiếu bước chạy %q", must)
		}
	}
	for k, v := range job.Env {
		if strings.HasPrefix(k, "EAIS_PG_") && !regexp.MustCompile(`^postgres://[^:@/]+@127\.0\.0\.1:\d+/`).MatchString(v) {
			t.Errorf("%s = %q: DSN phải dạng postgres://<vai>@127.0.0.1:<cổng>/… không mật khẩu", k, v)
		}
	}
	if strings.Contains(string(raw), "secrets.") {
		t.Error("ci.yml tham chiếu secrets.* — CI này không được cần bí mật nào")
	}
}
