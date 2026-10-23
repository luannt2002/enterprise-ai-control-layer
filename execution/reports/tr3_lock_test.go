package reports

// Khoá tuần tự hoá TR3 (HOP-DONG §5.6 mục 2; LC74; đột biến DB26). Hai lớp kiểm:
//   - xác định, không cần PG: bọc giao dịch của kho bộ nhớ để GHI LẠI thứ tự khoá và lần đọc nhóm; mỗi lần đọc nhóm phải đứng sau khoá của ô đó, và các khoá của một giao dịch tăng dần;
//   - đua thật (kho PG): hai giao dịch cùng nạp hai bản nghi trùng của hai tệp khác nhau; điểm dừng giữa lúc đọc nhóm và lúc ghi làm đua xảy ra CHẮC CHẮN nếu khoá vắng, không dựa vào lịch chạy.

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

type lockRecorder struct {
	txn
	log *[]string
}

func (l *lockRecorder) lock(k int64) error {
	*l.log = append(*l.log, fmt.Sprintf("lock:%d", k))
	return l.txn.lock(k)
}

func (l *lockRecorder) slotRecords(entity, day string) ([]*Record, error) {
	*l.log = append(*l.log, "read:"+entity+"\x1f"+day)
	return l.txn.slotRecords(entity, day)
}

// Mỗi lần đọc nhóm đứng sau khoá đúng ô của nó; các khoá của một giao dịch (một lần thử) tăng dần. Chạy trên cả bộ sinh chuỗi sự kiện, gồm sửa tin đổi ô (hai khoá một giao dịch).
func TestKhoaTuanTu_MoiLanDocNhomCoKhoaTruoc_KhoaTangDan(t *testing.T) {
	m := NewMemory(testOptions())
	var attempts [][]string
	reads, multi := 0, 0
	for s := 0; s < 400; s++ {
		ctx, tenant := newCtx("lockord")
		m.wrapTx = func(x txn) txn {
			attempts = append(attempts, nil)
			return &lockRecorder{txn: x, log: &attempts[len(attempts)-1]}
		}
		g := newGenRun(9100+int64(s), m, ctx)
		for i, steps := 0, g.steps(); i < steps; i++ {
			_, _, _ = g.step(t)
		}
		for ai, log := range attempts {
			held := map[string]bool{}
			var prev int64
			nLocks := 0
			for _, e := range log {
				var k int64
				if _, err := fmt.Sscanf(e, "lock:%d", &k); err == nil {
					if nLocks > 0 && k <= prev {
						t.Fatalf("hạt giống %d, lần thử %d: khoá %d sau khoá %d: không tăng dần (nguy cơ chết khoá)", 9100+s, ai, k, prev)
					}
					prev, nLocks = k, nLocks+1
					held[fmt.Sprint(k)] = true
					continue
				}
				var ent, day string
				for i := range e {
					if e[i] == 0x1f {
						ent, day = e[len("read:"):i], e[i+1:]
						break
					}
				}
				reads++
				if !held[fmt.Sprint(slotLockKey(tenant, slotKey{Entity: ent, Day: day}))] {
					t.Fatalf("hạt giống %d, lần thử %d: đọc nhóm (%s, %s) khi chưa giữ khoá của ô: đua giữa hai tiến trình (§5.6 mục 2)", 9100+s, ai, ent, day)
				}
			}
			if nLocks >= 2 {
				multi++
			}
		}
		attempts = nil
	}
	t.Logf("%d lần đọc nhóm đều có khoá trước; %d lần thử giữ từ hai khoá trở lên, tất cả tăng dần", reads, multi)
	if reads < 1000 || multi < 20 {
		t.Fatalf("phép kiểm chạm quá ít: %d lần đọc nhóm, %d lần thử nhiều khoá (muốn ≥ 1000, ≥ 20)", reads, multi)
	}
}

// Một thao tác phải chạy lại khi cần khoá nhỏ hơn khoá đã giữ, và vẫn cho kết quả đúng (sửa tin đổi ô sang ô có khoá nhỏ hơn).
func TestKhoaTuanTu_SuaTinDoiOSangOCoKhoaNhoHon_ChayLaiVanDung(t *testing.T) {
	m := NewMemory(testOptions())
	ctx, tenant := newCtx("relock")
	mustDeliver(t, ctx, m, "b", 1, mk("a", 0))
	days := []string{"2026-10-19", "2026-10-20", "2026-10-21", "2026-10-22", "2026-10-23", "2026-10-24"}
	big, small := days[0], days[0]
	for _, d := range days {
		if slotLockKey(tenant, slotKey{"E1", d}) > slotLockKey(tenant, slotKey{"E1", big}) {
			big = d
		}
		if slotLockKey(tenant, slotKey{"E1", d}) < slotLockKey(tenant, slotKey{"E1", small}) {
			small = d
		}
	}
	mustDeliver(t, ctx, m, "b", 2, mk("x", 5, withDay(big)))
	attempts := 0
	m.wrapTx = func(x txn) txn { attempts++; return x }
	target := mk("x", 5, withDay(small), withRaw("sửa x"))
	if small == big {
		t.Fatal("không dựng được hai ô khác khoá")
	}
	mustAmend(t, ctx, m, "x", target)
	if attempts < 2 {
		t.Fatalf("sửa tin đổi từ ô khoá lớn sang ô khoá nhỏ phải chạy lại (lấy khoá theo thứ tự): %d lần thử", attempts)
	}
	if got := mustRecord(t, ctx, m, "x"); got.OccurredOn != small || got.Version != 2 {
		t.Fatalf("sau sửa tin: %+v", got)
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// LC74: hai tiến trình nạp ĐỒNG THỜI hai tệp khác nhau, mỗi tệp một tin; hai tin cùng (máy, ngày, giá trị, đơn vị), cách nhau dưới W; lặp 100 lần, đảo bên khởi động trước.

func TestLC74_HaiTienTrinhDongThoi_DungMotBanAcceptedMotBanHeld(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		for round := 0; round < 100; round++ {
			ctx, _ := newCtx("lc74")
			a := mk("A", 0, withSender("S1"), withRaw("tệp 1"))
			b := mk("B", 30, withSender("S2"), withRaw("tệp 2"))
			first, second := a, b
			if round%2 == 1 {
				first, second = b, a
			}
			raceDeliver(t, ctx, env.st, first, second)
			wantRec(t, ctx, env.st, "A", DispAccepted, "")
			wantHeldPD(t, ctx, env.st, "B", "A")
			wantTotal(t, ctx, env.st, "1000")
			if bad := oracle(mustRecords(t, ctx, env.st), 60*time.Minute); len(bad) > 0 {
				t.Fatalf("vòng %d: %v", round, bad)
			}
		}
	})
}

// raceDeliver cho hai goroutine nạp hai đơn vị, mỗi bên một lô riêng, bắt đầu cùng lúc.
func raceDeliver(t *testing.T, ctx context.Context, st Store, items ...Item) {
	t.Helper()
	start := make(chan struct{})
	errs := make([]error, len(items))
	var wg sync.WaitGroup
	for i, it := range items {
		wg.Add(1)
		go func(i int, it Item) {
			defer wg.Done()
			<-start
			_, errs[i] = st.Deliver(ctx, "lo-"+it.Key, 1, it)
		}(i, it)
	}
	close(start)
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("nạp %s: %v", items[i].Key, err)
		}
	}
}

type barrierSetter interface{ setAfterSlotRead(f func()) }

func (m *Memory) setAfterSlotRead(f func()) { m.afterSlotRead = f }

// DB26: điểm dừng giữa lúc đọc nhóm và lúc ghi làm đua xảy ra CHẮC CHẮN nếu khoá vắng: mỗi bên, sau khi đọc nhóm, đợi bên kia cũng đã đọc (tối đa 400 ms). Có khoá thì bên thứ hai không tới
// được lần đọc nhóm trước khi bên thứ nhất ghi xong (điểm dừng hết hạn), nên một bản ACCEPTED và một bản HELD; bỏ khoá thì cả hai cùng đọc nhóm rỗng và cùng ACCEPTED (cộng đôi).
func TestDB26_DiemDungGiuaDocNhomVaGhi_KhoaChanDua(t *testing.T) {
	forEachStore(t, func(t *testing.T, env storeEnv) {
		bs, ok := env.st.(barrierSetter)
		if !ok {
			t.Skip("kho không có điểm dừng")
		}
		for round := 0; round < 6; round++ {
			var mu sync.Mutex
			arrived, release := 0, make(chan struct{})
			bs.setAfterSlotRead(func() {
				mu.Lock()
				arrived++
				if arrived == 2 {
					close(release)
				}
				mu.Unlock()
				select {
				case <-release:
				case <-time.After(400 * time.Millisecond):
				}
			})
			ctx, _ := newCtx("db26")
			raceDeliver(t, ctx, env.st, mk("A", 0, withSender("S1"), withRaw("tệp 1")), mk("B", 30, withSender("S2"), withRaw("tệp 2")))
			bs.setAfterSlotRead(nil)
			wantRec(t, ctx, env.st, "A", DispAccepted, "")
			wantHeldPD(t, ctx, env.st, "B", "A")
			wantTotal(t, ctx, env.st, "1000")
		}
	})
}
