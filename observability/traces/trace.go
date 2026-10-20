// Package traces đo thời gian từng chặng của một lượt xử lý.
//
// Bất biến mà package này bảo vệ: **tổng các chặng phải xấp xỉ tổng lượt**. Chênh lệch
// lớn nghĩa là có việc xảy ra ngoài mọi đồng hồ — một chặng đắt có thể báo 0 ms trong
// khi công việc thật nằm ở khoảng trống giữa hai chặng. Unaccounted() phơi con số đó ra.
package traces

import (
	"sync"
	"time"
)

// Span là thời gian của một chặng.
type Span struct {
	Name string `json:"name"`
	Ms   int64  `json:"ms"`
}

// Trace ghi thời gian các chặng của MỘT lượt. An toàn khi nhiều goroutine cùng ghi
// (các chặng song song như truy vấn nhiều nguồn).
type Trace struct {
	mu    sync.Mutex
	spans []Span
	start time.Time
	now   func() time.Time
}

// New tạo trace bắt đầu từ bây giờ. now nil thì dùng time.Now (test truyền đồng hồ giả).
func New(now func() time.Time) *Trace {
	if now == nil {
		now = time.Now
	}
	return &Trace{start: now(), now: now}
}

// Measure chạy fn và ghi thời gian của nó.
//
// Bọc bằng closure chứ không phải cặp Begin/End: cặp đó cho phép quên gọi End, và một
// chặng quên đóng trông y hệt một chặng 0 ms.
func Measure[T any](t *Trace, name string, fn func() (T, error)) (T, error) {
	s := t.now()
	v, err := fn()
	t.add(name, t.now().Sub(s))
	return v, err
}

// Measure2 đo một chặng trả về giá trị + cờ "có hay không". Tồn tại vì Go không cho
// generic biến thiên số giá trị trả về; thiếu nó thì lời gọi kiểu Get(...) (T, bool)
// nằm NGOÀI mọi đồng hồ.
func Measure2[T any](t *Trace, name string, fn func() (T, bool)) (T, bool) {
	s := t.now()
	v, ok := fn()
	t.add(name, t.now().Sub(s))
	return v, ok
}

// Do đo một chặng không trả giá trị.
func (t *Trace) Do(name string, fn func() error) error {
	s := t.now()
	err := fn()
	t.add(name, t.now().Sub(s))
	return err
}

func (t *Trace) add(name string, d time.Duration) {
	t.mu.Lock()
	t.spans = append(t.spans, Span{Name: name, Ms: d.Milliseconds()})
	t.mu.Unlock()
}

// Spans trả bản sao các chặng đã ghi.
func (t *Trace) Spans() []Span {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]Span(nil), t.spans...)
}

// TotalMs là tổng thời gian từ lúc tạo trace.
func (t *Trace) TotalMs() int64 { return t.now().Sub(t.start).Milliseconds() }

// Unaccounted là phần thời gian KHÔNG nằm trong chặng nào. Gần 0 nghĩa là các chặng đã
// kể hết câu chuyện; lớn nghĩa là đồng hồ đang nói dối, đừng tin kết luận rút từ chặng.
//
// Chặng song song làm tổng chặng vượt tổng lượt, khi đó trả 0 thay vì số âm.
func (t *Trace) Unaccounted() int64 {
	var sum int64
	for _, s := range t.Spans() {
		sum += s.Ms
	}
	if u := t.TotalMs() - sum; u > 0 {
		return u
	}
	return 0
}
