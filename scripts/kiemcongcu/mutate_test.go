package kiemcongcu

import (
	"crypto/sha256"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Lỗi 5c của soát độc lập 23/10: scripts/mutate.sh từng đo đột biến mà không chạy nền, nên trên nền đã đỏ mọi đột biến đều "bị bắt"
// giả. Các ca dưới chạy mutate.sh thật trên một module Go tạm (không chạm repo): bắt, sống, nền đỏ, đỏ sai chỗ, chưa có nền.

// mod dựng module tạm: a.go (F trả 1, G trả 2), a_test.go (TestF, TestG); thêm "đỏ sẵn" thì có một test luôn đỏ.
func mod(t *testing.T, doSan bool) string {
	t.Helper()
	d := t.TempDir()
	write(t, filepath.Join(d, "go.mod"), "module tamdotbien\n\ngo 1.26\n")
	write(t, filepath.Join(d, "a.go"), "package a\n\nfunc F() int { return 1 }\n\nfunc G() int { return 2 }\n")
	write(t, filepath.Join(d, "a_test.go"), "package a\n\nimport \"testing\"\n\nfunc TestF(t *testing.T) {\n\tif F() != 1 {\n\t\tt.Fatal(\"F\")\n\t}\n}\n\nfunc TestG(t *testing.T) {\n\tif G() != 2 {\n\t\tt.Fatal(\"G\")\n\t}\n}\n")
	if doSan {
		write(t, filepath.Join(d, "b_test.go"), "package a\n\nimport \"testing\"\n\nfunc TestDoSan(t *testing.T) { t.Fatal(\"đỏ sẵn\") }\n")
	}
	return d
}

// mutate chạy scripts/mutate.sh trong thư mục d với env thêm; trả đầu ra và mã thoát.
func mutate(t *testing.T, d string, env []string, args ...string) (string, int) {
	t.Helper()
	script, err := filepath.Abs(filepath.Join("..", "mutate.sh"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", append([]string{script}, args...)...)
	cmd.Dir = d
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	var ee *exec.ExitError
	switch {
	case err == nil:
		return string(out), 0
	case errors.As(err, &ee):
		return string(out), ee.ExitCode()
	default:
		t.Fatalf("không chạy được bash: %v", err)
		return "", -1
	}
}

func sum(t *testing.T, p string) [32]byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(b)
}

func TestMutateNenXanhVaKetCuc(t *testing.T) {
	d := mod(t, false)
	env := []string{"MUTATE_NEN_DIR=" + t.TempDir(), "PKGS=./..."}
	a := filepath.Join(d, "a.go")
	before := sum(t, a)
	if out, rc := mutate(t, d, env, "a.go", "return 1", "return 3", "M-bat"); rc != 0 || !strings.Contains(out, "BỊ BẮT") {
		t.Errorf("đột biến đổi F phải bị bắt (rc 0): rc %d, %q", rc, out)
	}
	if out, rc := mutate(t, d, env, "a.go", "func G() int { return 2 }", "func G() int { return 1 + 1 }", "M-tuong-duong"); rc != 3 {
		t.Errorf("đột biến tương đương phải sống (rc 3): rc %d, %q", rc, out)
	}
	if out, rc := mutate(t, d, append(env, "MUTATE_TEST_MUON=^TestG$"), "a.go", "return 1", "return 3", "M-sai-cho"); rc != 6 || !strings.Contains(out, "ĐỎ SAI CHỖ") {
		t.Errorf("đỏ ở TestF mà nhắm TestG phải là đỏ sai chỗ (rc 6): rc %d, %q", rc, out)
	}
	if out, rc := mutate(t, d, append(env, "MUTATE_TEST_MUON=^TestF$"), "a.go", "return 1", "return 3", "M-dung-cho"); rc != 0 {
		t.Errorf("đỏ đúng test nhắm tới phải bị bắt (rc 0): rc %d, %q", rc, out)
	}
	if sum(t, a) != before {
		t.Error("a.go không được khôi phục nguyên trạng")
	}
}

func TestMutateNenDoThiKhongDo(t *testing.T) {
	d := mod(t, true)
	a := filepath.Join(d, "a.go")
	before := sum(t, a)
	out, rc := mutate(t, d, []string{"MUTATE_NEN_DIR=" + t.TempDir(), "PKGS=./..."}, "a.go", "return 1", "return 3", "M-nen-do")
	if rc != 5 || !strings.Contains(out, "NỀN ĐỎ") {
		t.Errorf("nền đã đỏ thì không đo đột biến (rc 5): rc %d, %q", rc, out)
	}
	if strings.Contains(out, "BỊ BẮT") {
		t.Errorf("nền đỏ mà vẫn báo bị bắt: %q", out)
	}
	if sum(t, a) != before {
		t.Error("a.go bị đổi dù không đo")
	}
}

func TestMutateNenCoSan(t *testing.T) {
	d := mod(t, false)
	env := []string{"MUTATE_NEN_DIR=" + t.TempDir(), "PKGS=./...", "MUTATE_NEN_CO_SAN=1"}
	if out, rc := mutate(t, d, env, "a.go", "return 1", "return 3", "M-chua-co"); rc != 5 || !strings.Contains(out, "CHƯA CÓ NỀN") {
		t.Errorf("MUTATE_NEN_CO_SAN=1 mà chưa chạy --nen thì không đo (rc 5): rc %d, %q", rc, out)
	}
	if out, rc := mutate(t, d, env, "--nen"); rc != 0 || !strings.Contains(out, "XANH") {
		t.Fatalf("--nen trên nền xanh: rc %d, %q", rc, out)
	}
	if out, rc := mutate(t, d, env, "a.go", "return 1", "return 3", "M-co-san"); rc != 0 {
		t.Errorf("có nền đã lưu thì đo bình thường (rc 0): rc %d, %q", rc, out)
	}
}
