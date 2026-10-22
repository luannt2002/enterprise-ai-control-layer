package main

import (
	"bytes"
	"fmt"
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
	evFail    = `{"Action":"run","Package":"p","Test":"TestC"}` + "\n" +
		`{"Action":"output","Package":"p","Test":"TestC","Output":"    c_test.go:7: lệch\n"}` + "\n" +
		`{"Action":"fail","Package":"p","Test":"TestC","Elapsed":0}` + "\n"
	evPkgFail = `{"Action":"fail","Package":"q","Elapsed":0,"FailedBuild":"q"}` + "\n" // gói không dịch được: không có sự kiện mức test nào
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
	// ĐỎ: test FAIL (không SKIP) cũng phải làm cổng đỏ — lớp thứ hai sau pipefail của bước go test: nếu ai đó làm bước đó nuốt mã lỗi,
	// luồng JSON vẫn kể lại test hỏng. Gói không dịch được chỉ có sự kiện mức gói (không có Test) nên cũng phải đỏ.
	code, out, errb = runSkipGate(t, evPass+evFail)
	if code != 1 || !strings.Contains(out, "FAIL: p TestC") || !strings.Contains(errb, "1 test FAIL") {
		t.Errorf("có test FAIL: %d\n%s%s\nmuốn 1 và nêu TestC", code, out, errb)
	}
	code, out, errb = runSkipGate(t, evPass+evPkgFail)
	if code != 1 || !strings.Contains(out, "FAIL (gói): q") || !strings.Contains(errb, "1 gói FAIL") {
		t.Errorf("có gói FAIL (không dịch được): %d\n%s%s\nmuốn 1 và nêu gói q", code, out, errb)
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

type ciStep struct {
	Name string            `yaml:"name"`
	Uses string            `yaml:"uses"`
	Run  string            `yaml:"run"`
	Env  map[string]string `yaml:"env"`
}

type ciWorkflow struct {
	Permissions map[string]string `yaml:"permissions"`
	Jobs        map[string]struct {
		Services map[string]struct {
			Image string `yaml:"image"`
		} `yaml:"services"`
		Env      map[string]string `yaml:"env"`
		Defaults struct {
			Run struct {
				Shell string `yaml:"shell"`
			} `yaml:"run"`
		} `yaml:"defaults"`
		Steps []ciStep `yaml:"steps"`
	} `yaml:"jobs"`
}

func loadCI(t *testing.T) (ciWorkflow, []byte) {
	t.Helper()
	raw, err := os.ReadFile(repoPath(".github/workflows/ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var wf ciWorkflow
	if err := yaml.Unmarshal(raw, &wf); err != nil {
		t.Fatalf("ci.yml không phải YAML hợp lệ: %v", err)
	}
	return wf, raw
}

// Workflow CI nhất quán với phần còn lại của repo: cùng ảnh PG ghim digest với deploy/docker-compose.yml, đủ các bước cổng, action ghim
// theo tag đầy đủ hoặc SHA, quyền chỉ đọc, không bí mật, DSN không mang mật khẩu.
func TestWorkflowCI(t *testing.T) {
	wf, raw := loadCI(t)
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
	// Bước go test dùng `set -o pipefail` — tính năng của bash, không phải của sh — nên workflow phải ghim shell: bash.
	if job.Defaults.Run.Shell != "bash" {
		t.Errorf("defaults.run.shell = %q, muốn bash (bước go test dùng pipefail của bash)", job.Defaults.Run.Shell)
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
	for _, must := range []string{"go vet ./...", "gofmt -l", "go test -count=1 -json ./...", "scripts/kiem-skip-test.py", "TestDanhTinh", "scripts/chay-thu.sh", "poppler-utils", "fonts-dejavu-core"} {
		if !strings.Contains(all, must) {
			t.Errorf("ci.yml thiếu bước chạy %q", must)
		}
	}
	// R2-02: bước smoke nhúng commit đang kiểm vào binary (script đọc EAIS_CODE_VERSION); thiếu thì vết của CI khai mã theo git của runner,
	// không phải theo commit mà workflow đang kiểm.
	smoke := 0
	for _, s := range job.Steps {
		if strings.Contains(s.Run, "scripts/chay-thu.sh") {
			smoke++
			if got := s.Env["EAIS_CODE_VERSION"]; got != "${{ github.sha }}" {
				t.Errorf("bước %q: EAIS_CODE_VERSION = %q, muốn ${{ github.sha }}", s.Name, got)
			}
		}
	}
	if smoke != 1 {
		t.Errorf("ci.yml có %d bước chạy scripts/chay-thu.sh, muốn đúng 1", smoke)
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

// runCIStep chạy ĐÚNG đoạn lệnh của một bước ci.yml dưới bash TRẦN (không -e, không pipefail: runner có thể thêm cờ nhưng đoạn lệnh phải
// tự đủ) trong một thư mục làm việc giả, với ba công cụ giả của các cổng — `go`, `python3` và scripts/chay-thu.sh — cùng thoát bằng exit
// (`go` còn in một dòng JSON như go test -json). Trả mã thoát của đoạn lệnh và nội dung test.json mà bước ghi ra (nếu có).
func runCIStep(t *testing.T, script string, exit int) (int, string) {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	for _, d := range []string{bin, filepath.Join(dir, "scripts")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	stub := func(path, body string) {
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+fmt.Sprintf("exit %d\n", exit)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	stub(filepath.Join(bin, "go"), `printf '%s\n' '{"Action":"output","Package":"p","Test":"TestX","Output":"x"}'`+"\n")
	stub(filepath.Join(bin, "python3"), "")
	stub(filepath.Join(dir, "scripts", "chay-thu.sh"), "")
	return runCIScript(t, dir, script, bin+":/usr/bin:/bin")
}

// runCIScript chạy script dưới bash trần trong dir với PATH cho trước; trả mã thoát và test.json (nếu có).
func runCIScript(t *testing.T, dir, script, path string) (int, string) {
	t.Helper()
	sp := filepath.Join(dir, "step.sh")
	if err := os.WriteFile(sp, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", "--noprofile", "--norc", sp)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + path, "HOME=" + dir, "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null"}
	err := cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	out, _ := os.ReadFile(filepath.Join(dir, "test.json"))
	return code, string(out)
}

// "Test hỏng thì CI đỏ" là tính chất của CHÍNH bước go test trong ci.yml, không phải của cấu hình ngầm của runner: bước đó đi qua một
// đường ống (| tee test.json) nên không có pipefail thì mã thoát cả đường ống là của tee (0) và test hỏng vẫn ra xanh. Ở đây bước được
// chạy thật với một `go` giả hỏng / không hỏng.
func TestCIBuocGoTestDoKhiGoTestHong(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Fatalf("cần bash để chạy bước CI: %v", err)
	}
	wf, _ := loadCI(t)
	var step *ciStep
	for i, s := range wf.Jobs["kiem"].Steps {
		if strings.Contains(s.Run, "go test -count=1 -json ./...") {
			step = &wf.Jobs["kiem"].Steps[i]
		}
	}
	if step == nil {
		t.Fatal("ci.yml không có bước chạy go test -count=1 -json ./...")
	}
	if code, _ := runCIStep(t, step.Run, 1); code == 0 {
		t.Error("go test thoát 1 mà bước go test của CI thoát 0 — test hỏng sẽ ra CI xanh (thiếu pipefail, hoặc có || true)")
	}
	if code, out := runCIStep(t, step.Run, 0); code != 0 || !strings.Contains(out, `"Test":"TestX"`) {
		t.Errorf("go test thoát 0: bước thoát %d, test.json %q — muốn 0 và test.json mang đúng luồng go test", code, out)
	}
	// đối chứng của chính phép thử: bỏ các dòng `set …` thì đường ống nuốt mã lỗi (bước thoát 0) — phép thử này phân biệt được hai trường hợp
	noSet := regexp.MustCompile(`(?m)^\s*set .*$`).ReplaceAllString(step.Run, "")
	if code, _ := runCIStep(t, noSet, 1); code != 0 {
		t.Errorf("đối chứng: bỏ pipefail mà bước vẫn thoát %d — phép thử không phân biệt được bước có / không nuốt mã lỗi", code)
	}
}

// Mọi bước cổng của ci.yml phải ĐỎ khi công cụ của cổng đỏ: `|| true`, một `true` thay lệnh, hay một lệnh bị xoá đều làm CI xanh giả mà
// không cổng nào bắt. Ở đây từng bước (trừ apt, cần sudo thật; và gofmt, kiểm riêng bằng công cụ thật bên dưới) được chạy với các
// công cụ giả cùng thoát 1 — phải đỏ — và cùng thoát 0 — phải xanh (đối chứng: phép thử không phải lúc nào cũng đỏ).
func TestCIMoiBuocCongDoKhiCongHong(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Fatalf("cần bash để chạy bước CI: %v", err)
	}
	wf, _ := loadCI(t)
	gated := 0
	for _, s := range wf.Jobs["kiem"].Steps {
		if s.Run == "" || strings.Contains(s.Run, "apt-get") || strings.Contains(s.Run, "gofmt") {
			continue
		}
		gated++
		if code, _ := runCIStep(t, s.Run, 1); code == 0 {
			t.Errorf("bước %q thoát 0 khi công cụ của cổng (go, python3, scripts/chay-thu.sh) thoát 1 — cổng bị vô hiệu (|| true, true, lệnh bị xoá?)", s.Name)
		}
		if code, _ := runCIStep(t, s.Run, 0); code != 0 {
			t.Errorf("đối chứng: bước %q thoát %d khi mọi công cụ thoát 0", s.Name, code)
		}
	}
	if gated < 5 { // go vet, go test, kiem-skip-test, cổng danh tính, smoke E2E
		t.Errorf("chỉ chạy được %d bước cổng, muốn ít nhất 5", gated)
	}
}

// Bước gofmt chạy bằng git, xargs và gofmt THẬT trên một kho git tạm: có tệp .go chưa format (đã `git add`) thì bước phải đỏ và nêu tệp đó;
// format xong thì xanh.
func TestCIBuocGofmtDoKhiCoTepChuaFormat(t *testing.T) {
	for _, tool := range []string{"bash", "git", "xargs", "gofmt"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Fatalf("cần %s để chạy bước gofmt của CI: %v", tool, err)
		}
	}
	wf, _ := loadCI(t)
	var script string
	for _, s := range wf.Jobs["kiem"].Steps {
		if strings.Contains(s.Run, "gofmt -l") {
			script = s.Run
		}
	}
	if script == "" {
		t.Fatal("ci.yml không có bước gofmt")
	}
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + dir, "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null"}
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	write := func(src string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "x.go"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		git("add", "x.go")
	}
	write("package x\n\nfunc  f( ) {}\n") // hai dấu cách và dấu cách trong ngoặc: gofmt sẽ sửa
	if code, _ := runCIScript(t, dir, script, os.Getenv("PATH")); code == 0 {
		t.Error("có tệp .go chưa format mà bước gofmt của CI thoát 0")
	}
	write("package x\n\nfunc f() {}\n")
	if code, _ := runCIScript(t, dir, script, os.Getenv("PATH")); code != 0 {
		t.Errorf("mọi tệp .go đã format mà bước gofmt thoát %d, muốn 0", code)
	}
}
