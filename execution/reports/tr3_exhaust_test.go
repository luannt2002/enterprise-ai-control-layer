package reports

// R6-01 và R6-02, duyệt HẾT (không chỉ ngẫu nhiên): không gian nhỏ gồm ba bản a, b, c cùng (thực thể, ngày, giá trị, đơn vị), gửi cách nhau 30 phút (mỗi cặp liền kề ≤ W = 60), một bản
// đính chính q chờ APPLY, và các sự kiện KEEP_BOTH, MARK_DUPLICATE (mang duplicate_of hiện tại), thu hồi, sửa giá trị, sửa thành HELD vì lý do khác, APPLY. Tìm kiếm theo chiều sâu có giới hạn,
// bỏ trạng thái đã thăm ở độ sâu không lớn hơn; sau MỌI sự kiện của MỌI trạng thái tới được: phép kiểm tính chất (oracle), không máy nhả, lệnh bị từ chối không đổi bản ghi, tổng khớp tính lại.
// Chuỗi qua bản đang REJECTED (R5-01) mà bộ sinh ngẫu nhiên hiếm khi dựng nằm trong không gian này (chuỗi 7 bước: nạp ba bản, MARK_DUPLICATE c, MARK_DUPLICATE b, thu hồi b, thu hồi a).

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func (m *Memory) cloneForTest() *Memory {
	c := &Memory{opt: m.opt, data: map[string]*memTenant{}}
	for id, t := range m.data {
		nt := &memTenant{recs: map[string]*Record{}, versions: map[string][]versionRow{}, fps: map[string][]string{}, history: map[string][]HistoryEntry{}, ledger: append([]DeliveryEntry(nil), t.ledger...)}
		for k, r := range t.recs {
			cr := r.clone()
			nt.recs[k] = &cr
		}
		for k, v := range t.versions {
			nt.versions[k] = append([]versionRow(nil), v...)
		}
		for k, v := range t.fps {
			nt.fps[k] = append([]string(nil), v...)
		}
		for k, v := range t.history {
			nt.history[k] = append([]HistoryEntry(nil), v...)
		}
		c.data[id] = nt
	}
	return c
}

func exStateKey(rs []Record) string {
	var b strings.Builder
	for _, r := range rs {
		fmt.Fprintf(&b, "%s|%s|%s|%s|%s|%d|%s|%s|%s|%s|%s|%v|%s;", r.Key, r.Disposition, r.Reason, r.DuplicateOf, r.Decision, r.Version, r.Entity, r.OccurredOn, r.Value.Number, r.Kind, r.SupersededBy, r.ReviewedPairs, r.SentAt.Format("15:04"))
	}
	return b.String()
}

type exEvent struct {
	desc  string
	allow string // khoá được phép chuyển sang ACCEPTED trong sự kiện này ("" = không ai)
	run   func(st *Memory, ctx context.Context) error
}

func exEvents(rs []Record) []exEvent {
	by := map[string]Record{}
	for _, r := range rs {
		by[r.Key] = r
	}
	var out []exEvent
	mins := map[string]int{"a": 0, "b": 30, "c": 60}
	for _, u := range []string{"a", "b", "c"} {
		u := u
		r, has := by[u]
		if !has {
			out = append(out, exEvent{"deliver " + u, u, func(st *Memory, ctx context.Context) error {
				_, err := st.Deliver(ctx, "lo-"+u, 1, mk(u, mins[u]))
				return err
			}})
			continue
		}
		if r.Disposition == DispHeld && r.Reason == ReasonPossibleDuplicate {
			seen := r.DuplicateOf
			out = append(out,
				exEvent{"markDup " + u, "", func(st *Memory, ctx context.Context) error {
					_, err := st.Decide(ctx, Command{Key: u, Decision: DecMarkDuplicate, Actor: "pm", Why: "t", SeenDuplicateOf: &seen})
					return err
				}},
				exEvent{"keepBoth " + u, u, func(st *Memory, ctx context.Context) error {
					_, err := st.Decide(ctx, Command{Key: u, Decision: DecKeepBoth, Actor: "pm", Why: "t", SeenDuplicateOf: &seen})
					return err
				}})
		}
		if r.Disposition != DispRetracted && r.Disposition != DispSuperseded {
			ver := r.Version + 1
			other := "2000"
			if r.Value.Number == "2000" {
				other = "1000"
			}
			out = append(out,
				exEvent{"retract " + u, "", func(st *Memory, ctx context.Context) error { return st.Retract(ctx, u, "ingest", "t") }},
				exEvent{"amendValue " + u, u, func(st *Memory, ctx context.Context) error {
					return st.Amend(ctx, u, mk(u, mins[u], withValue(other), withRaw(fmt.Sprintf("sửa %s v%d", u, ver))), "ingest", "t")
				}},
				exEvent{"amendHeld " + u, "", func(st *Memory, ctx context.Context) error {
					return st.Amend(ctx, u, mk(u, mins[u], heldFor(ReasonAmbiguousEntity), withRaw(fmt.Sprintf("sửa %s v%d", u, ver))), "ingest", "t")
				}})
		}
	}
	q, hasQ := by["q"]
	if !hasQ {
		out = append(out, exEvent{"deliver q", "q", func(st *Memory, ctx context.Context) error {
			_, err := st.Deliver(ctx, "lo-q", 1, mk("q", 200, heldFor(ReasonCorrectionUnconfirmed), withKind(KindCorrection), withValue("800")))
			return err
		}})
	} else if q.Disposition == DispHeld && q.Reason == ReasonCorrectionUnconfirmed {
		for _, u := range []string{"a", "b", "c"} {
			u := u
			if r, ok := by[u]; ok && (r.Disposition == DispAccepted || r.Disposition == DispHeld || r.Disposition == DispRejected) {
				out = append(out, exEvent{"apply q→" + u, "q", func(st *Memory, ctx context.Context) error {
					_, err := st.Decide(ctx, Command{Key: "q", Decision: DecApply, Actor: "pm", Why: "t", Target: u})
					return err
				}})
			}
		}
	}
	return out
}

func TestR601_DuyetHetKhongGianBaBan(t *testing.T) {
	if testing.Short() {
		t.Skip("duyệt hết không gian trạng thái bỏ qua ở -short")
	}
	maxDepth := 9 // mặc định: 82.400 trạng thái, ~3 giây; EAIS_DUYET_DO_SAU=11 duyệt 392.084 trạng thái, ~18 giây
	if v := os.Getenv("EAIS_DUYET_DO_SAU"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 14 {
			maxDepth = n
		}
	}
	w := 60 * time.Minute
	ctx, _ := newCtx("duyet")
	root := NewMemory(testOptions())
	visited := map[string]int{}
	states, edges, returned := 0, 0, 0
	var walk func(st *Memory, depth int, trail []string)
	walk = func(st *Memory, depth int, trail []string) {
		before, _ := st.Records(ctx)
		key := exStateKey(before)
		if d, ok := visited[key]; ok && d <= depth {
			return
		}
		if _, ok := visited[key]; !ok {
			states++
		}
		visited[key] = depth
		if depth == maxDepth {
			return
		}
		for _, ev := range exEvents(before) {
			c := st.cloneForTest()
			err := ev.run(c, ctx)
			edges++
			after, _ := c.Records(ctx)
			allow := map[string]bool{}
			if ev.allow != "" {
				allow[ev.allow] = true
			}
			where := fmt.Sprintf("chuỗi %s ; %s", strings.Join(trail, " ; "), ev.desc)
			checkStep(t, where, ev.desc, before, after, allow, err, w)
			if err != nil {
				continue
			}
			for _, r := range after {
				if r.Disposition == DispHeld && r.Decision == DecMarkDuplicate {
					returned++
					break
				}
			}
			walk(c, depth+1, append(append([]string(nil), trail...), ev.desc))
		}
	}
	walk(root, 0, nil)
	t.Logf("duyệt hết tới độ sâu %d: %d trạng thái, %d cạnh (sự kiện) đều giữ bất biến; %d cạnh dẫn tới trạng thái có bản bị bác quay lại HELD", maxDepth, states, edges, returned)
	if states < 500 || returned < 50 {
		t.Fatalf("không gian quá nhỏ để chứng minh gì: %d trạng thái, %d cạnh có bản quay lại HELD", states, returned)
	}
}
