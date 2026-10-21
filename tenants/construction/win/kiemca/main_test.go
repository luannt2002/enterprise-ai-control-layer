package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const world = "../world.yaml"

func writeSuite(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "ca.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// Bộ ca đúng định dạng: thoát 0, in số ca theo lớp và tổng.
func TestKiemCaDungDinhDang(t *testing.T) {
	p := writeSuite(t, `suite: win
tenant: abc-infra
reference_date: 2026-10-20
cases:
  - id: WIN-I-901
    failure_class: semantic
    authored_by: independent
    principal: u-director
    question: "CAU-HOI-BI-MAT"
    expect:
      route: SQL_QUERY
      outcome: RESOLVED
      value: {number: "1", unit: L}
`)
	var out bytes.Buffer
	if code := run(&out, world, "abc-infra", []string{p}); code != 0 {
		t.Fatalf("mã %d, muốn 0: %s", code, out.String())
	}
	if !strings.Contains(out.String(), "TỔNG 1 ca · có kỳ vọng số liệu 1 · lỗi định dạng 0") {
		t.Errorf("thiếu dòng tổng: %s", out.String())
	}
}

// Lỗi định dạng (lớp lạ, route lạ, principal lạ, id trùng giữa hai tệp): thoát 1, báo theo mã ca và tên trường — KHÔNG in
// câu hỏi hay kỳ vọng (người viết mã không được đọc nội dung bộ giữ lại trước khi chạy).
func TestKiemCaBaoLoiKhongInNoiDung(t *testing.T) {
	bad := `suite: win
tenant: abc-infra
reference_date: 2026-10-20
cases:
  - id: WIN-I-902
    failure_class: khong-co-lop-nay
    authored_by: independent
    principal: u-khong-ton-tai
    question: "CAU-HOI-BI-MAT"
    expect:
      route: TUYEN-LA
      outcome: RESOLVED
      contains: ["KY-VONG-BI-MAT"]
`
	p1, p2 := writeSuite(t, bad), writeSuite(t, bad)
	var out bytes.Buffer
	if code := run(&out, world, "abc-infra", []string{p1, p2}); code != 1 {
		t.Fatalf("mã %d, muốn 1: %s", code, out.String())
	}
	s := out.String()
	for _, want := range []string{"failure_class lạ", "principal", "expect.route lạ", "đã có ở"} {
		if !strings.Contains(s, want) {
			t.Errorf("thiếu lỗi %q: %s", want, s)
		}
	}
	if strings.Contains(s, "CAU-HOI-BI-MAT") || strings.Contains(s, "KY-VONG-BI-MAT") {
		t.Errorf("in ra nội dung ca: %s", s)
	}
}

// Không đọc được tệp: thoát 2.
func TestKiemCaTepHong(t *testing.T) {
	var out bytes.Buffer
	if code := run(&out, world, "abc-infra", []string{filepath.Join(t.TempDir(), "khong-co.yaml")}); code != 2 {
		t.Errorf("mã %d, muốn 2", code)
	}
}
