package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// viet dựng một cây mã giả từ bản đồ đường dẫn → nội dung.
func viet(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for p, body := range files {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func chay(args ...string) (int, string) {
	var out bytes.Buffer
	code := run(args, &out, &out)
	return code, out.String()
}

const goc = `package p

import "testing"

var ky = []int{1, 2}

func TestGiu(t *testing.T) {
	if "{" != "{" { // ngoặc trong chuỗi không được làm lệch ranh giới hàm
		t.Fatal("a thắng")
	}
}

func TestDao(t *testing.T) {
	if got := 1; got != 1 {
		t.Fatal("a thắng")
	}
}

func BenchmarkB(b *testing.B) {}

func TestMotDong(t *testing.T) { t.Log("x") }
`

func TestBaoDoiThanKhiGiuTen(t *testing.T) {
	// cây mới: TestGiu chỉ đổi comment, định dạng và dời xuống dưới; TestMotDong tách thành nhiều dòng (dấu chấm phẩy tự chèn); TestDao
	// đảo kỳ vọng mà giữ tên; bảng ky đổi; BenchmarkB giữ nguyên
	moi := `package p

import "testing"

var ky = []int{1, 3}

func TestDao(t *testing.T) {
	if got := 1; got != 2 {
		t.Fatal("hoà ⇒ hỏi lại")
	}
}

func BenchmarkB(b *testing.B) {}

func TestMotDong(t *testing.T) {
	t.Log("x")
}

// TestGiu: comment mới
func TestGiu(t *testing.T) {
	if "{" !=
		"{" { // comment khác
		t.Fatal("a thắng")
	}
}
`
	a := viet(t, map[string]string{"x/p_test.go": goc, "x/p.go": "package p\n\nfunc lo() {}\n", "x/testdata/q_test.go": "không phải Go"})
	b := viet(t, map[string]string{"x/p_test.go": moi, "x/p.go": "package p\n\nfunc lo() { _ = 1 }\n", "x/testdata/q_test.go": "vẫn không phải Go"})

	code, out := chay(a, b)
	if code != 1 || !strings.Contains(out, "ĐỔI THÂN (test): x TestDao") || !strings.Contains(out, "ĐỔI (khai báo khác): x ky") {
		t.Fatalf("đảo kỳ vọng giữ tên: thoát %d, muốn 1 và nêu x TestDao, x ky\n%s", code, out)
	}
	for _, khong := range []string{"x TestGiu", "x TestMotDong", "x BenchmarkB", "x lo"} {
		if strings.Contains(out, khong) {
			t.Errorf("%s không đổi thân (chỉ comment / định dạng / chỗ, hoặc không phải tệp _test.go) mà bị báo:\n%s", khong, out)
		}
	}

	// khai có chủ ý đủ cả hai thì xanh; thiếu một thì đỏ; mục khai thừa thì đỏ; mục không lý do là lỗi dùng
	cy := filepath.Join(t.TempDir(), "doi.txt")
	os.WriteFile(cy, []byte("# đổi có chủ ý\nTestDao hợp đồng mới: hoà ⇒ hỏi lại\nky bảng kỳ vọng theo hợp đồng mới\n"), 0o644)
	if code, out := chay(a, b, cy); code != 0 || !strings.Contains(out, "đổi có chủ ý: x TestDao — hợp đồng mới") {
		t.Errorf("khai đủ: thoát %d, muốn 0\n%s", code, out)
	}
	os.WriteFile(cy, []byte("TestDao hợp đồng mới\n"), 0o644)
	if code, out := chay(a, b, cy); code != 1 || !strings.Contains(out, "x ky") {
		t.Errorf("thiếu khai ky: thoát %d, muốn 1\n%s", code, out)
	}
	os.WriteFile(cy, []byte("TestDao hợp đồng mới\nky bảng\nTestGiu không đổi mà khai\n"), 0o644)
	if code, out := chay(a, b, cy); code != 1 || !strings.Contains(out, "THỪA: TestGiu") {
		t.Errorf("mục khai thừa: thoát %d, muốn 1 và nêu THỪA\n%s", code, out)
	}
	os.WriteFile(cy, []byte("TestDao\n"), 0o644)
	if code, _ := chay(a, b, cy); code != 2 {
		t.Errorf("mục không lý do: thoát %d, muốn 2", code)
	}

	// đối chứng: hai cây giống hệt thì xanh; cây không có tệp _test.go nào là rỗng, không phải xanh
	if code, out := chay(a, a); code != 0 || !strings.Contains(out, "đổi thân 0") {
		t.Errorf("hai cây giống nhau: thoát %d, muốn 0\n%s", code, out)
	}
	if code, _ := chay(a, viet(t, map[string]string{"x/p.go": "package p\n"})); code != 2 {
		t.Errorf("cây không có test: thoát %d, muốn 2", code)
	}
}
