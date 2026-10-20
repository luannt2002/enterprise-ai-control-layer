package traces

import (
	"errors"
	"testing"
	"time"
)

// fakeClock tiến đúng step mỗi lần được hỏi giờ — đo được mà không phải ngủ.
type fakeClock struct {
	t    time.Time
	step time.Duration
}

func (c *fakeClock) now() time.Time {
	c.t = c.t.Add(c.step)
	return c.t
}

func TestMeasureGhiChangVaKhongNuotLoi(t *testing.T) {
	c := &fakeClock{t: time.Unix(0, 0), step: 10 * time.Millisecond}
	tr := New(c.now)
	_, err := Measure(tr, "normalize", func() (int, error) { return 0, errors.New("hỏng") })
	if err == nil || err.Error() != "hỏng" {
		t.Fatalf("Measure phải trả nguyên lỗi của fn, được %v", err)
	}
	spans := tr.Spans()
	if len(spans) != 1 || spans[0].Name != "normalize" || spans[0].Ms != 10 {
		t.Fatalf("spans = %+v", spans)
	}
}

func TestUnaccountedPhoiKhoangTrong(t *testing.T) {
	c := &fakeClock{t: time.Unix(0, 0), step: 10 * time.Millisecond}
	tr := New(c.now) // start = 10ms
	_ = tr.Do("route", func() error { return nil })
	// Việc xảy ra NGOÀI mọi đồng hồ: đồng hồ chạy thêm 3 bước mà không ai đo.
	c.now()
	c.now()
	c.now()
	// Tổng lượt tính tới lần hỏi giờ kế tiếp; chặng đo được chỉ 10ms.
	if u := tr.Unaccounted(); u < 30 {
		t.Fatalf("Unaccounted = %d, phải >= 30ms cho khoảng trống không ai đo", u)
	}
}

func TestMeasure2ChoGiaTriCoKhong(t *testing.T) {
	tr := New(nil)
	v, ok := Measure2(tr, "cache_get", func() (string, bool) { return "x", true })
	if v != "x" || !ok || len(tr.Spans()) != 1 {
		t.Fatalf("Measure2 = %q %v spans=%v", v, ok, tr.Spans())
	}
}
