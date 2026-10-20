package decimal

import "testing"

func TestPhepTinhChinhXac(t *testing.T) {
	sum, _ := Add("0.1", "0.2")
	if sum != "0.3" {
		t.Fatalf("0.1 + 0.2 = %s — float64 sẽ ra 0.30000000000000004", sum)
	}
	if d, _ := Sub("3850", "3782"); d != "68" {
		t.Fatalf("3850 − 3782 = %s", d)
	}
	if p, _ := Mul("4", "1000"); p != "4000" {
		t.Fatalf("4 × 1000 = %s", p)
	}
	if q, _ := Quo("350", "1000"); q != "0.35" {
		t.Fatalf("350 / 1000 = %s", q)
	}
	if _, err := Quo("1", "0"); err == nil {
		t.Fatal("chia cho 0 phải lỗi")
	}
	if !Equal("4000", "4000.000") {
		t.Fatal("4000 phải bằng 4000.000")
	}
	if ok, _ := Within("3850", "3900", "50"); !ok {
		t.Fatal("|3850−3900| = 50 phải nằm trong dung sai 50")
	}
	if ok, _ := Within("3850", "4000", "50"); ok {
		t.Fatal("|3850−4000| = 150 không được nằm trong dung sai 50")
	}
	if _, err := Add("abc", "1"); err == nil {
		t.Fatal("số hỏng phải lỗi")
	}
}
