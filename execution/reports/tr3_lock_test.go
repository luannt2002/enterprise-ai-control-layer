package reports

// Khoá tuần tự hoá TR3 (HOP-DONG §5.6 mục 2; LC74; đột biến DB26). Ba lớp kiểm:
//   - xác định, không cần PG: bọc giao dịch của kho bộ nhớ để GHI LẠI thứ tự khoá và lần đọc nhóm; mỗi lần đọc nhóm phải đứng sau khoá của ô đó, và các khoá của một giao dịch tăng dần;
//   - đua thật (kho PG): hai giao dịch cùng nạp hai bản nghi trùng của hai tệp khác nhau; điểm dừng giữa lúc đọc nhóm và lúc ghi làm đua xảy ra CHẮC CHẮN nếu khoá vắng, không dựa vào lịch chạy;
//   - tải hỗn hợp (kho PG, R-01): nạp × quyết định × thu hồi × sửa tin đồng thời trên cùng tenant, đòi 0 lần chết khoá (40P01).

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
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
			_, errs[i] = st.Deliver(ctx, fmt.Sprintf("lo-%s-%d", it.Key, i), 1, it) // mã lô cấp mới cho mỗi lần gọi (§3.8), kể cả khi hai bên giao cùng một khoá
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

// APPLY đính chính mà bản gốc nằm ở ô có khoá NHỎ hơn ô của bản đính chính: khoá thứ hai phá thứ tự tăng dần nên giao dịch phải chạy lại với cả hai ô lấy sẵn (đường getLocked của bản gốc, khác đường settle).
func TestKhoaTuanTu_ApplyBanGocOOKhacCoKhoaNhoHon_ChayLaiVanDung(t *testing.T) {
	m := NewMemory(testOptions())
	ctx, tenant := newCtx("relockap")
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
	mustDeliver(t, ctx, m, "b", 1, mk("goc", 0, withDay(small), withValue("3000")))
	mustDeliver(t, ctx, m, "b", 2, mk("dc", 5, heldFor(ReasonCorrectionUnconfirmed), withKind(KindCorrection), withDay(big), withValue("2000")))
	attempts := 0
	m.wrapTx = func(x txn) txn { attempts++; return x }
	got := mustDecide(t, ctx, m, Command{Key: "dc", Decision: DecApply, Actor: "pm-1", Why: "báo nhầm", Target: "goc"})
	if attempts < 2 {
		t.Fatalf("APPLY có bản gốc ở ô khoá nhỏ hơn phải chạy lại (lấy khoá theo thứ tự): %d lần thử", attempts)
	}
	if got.Disposition != DispAccepted || mustRecord(t, ctx, m, "goc").Disposition != DispSuperseded {
		t.Fatalf("sau APPLY: đính chính %+v, bản gốc %+v", got, mustRecord(t, ctx, m, "goc"))
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// R-01 (soát 23/10): nạp × quyết định × thu hồi × sửa tin ĐỒNG THỜI trên CÙNG tenant của kho PG. Chuỗi chờ vòng cần một giao dịch giữ khoá hàng trong lúc chờ khoá tư vấn
// của một ô (ô đích của sửa tin / CHOOSE / CORRECT / APPLY, hay ô mới của bản ghi), còn giao dịch giữ khoá ô đó chờ chính hàng ấy (UPDATE, hay khoá ngoại của dòng sổ
// DUPLICATE trỏ bản ghi). Đòi 0 lần 40P01, 0 lỗi ngoài lệnh bị từ chối (nhãn input), và sau mỗi vòng: điểm bất động TR3, số dòng sổ = số lần giao thành công (BB1).
// Vòng chẵn "nóng": mọi bản có ô nằm cùng MỘT ô, bản không có ô (thực thể mơ hồ, cách ly) được quyết định hay sửa vào ô đó.
func TestPG_R01_ThaoTacHonHopDongThoi_KhongChetKhoa(t *testing.T) {
	st, _ := pgSetup(t)
	pg := NewPG(st, testOptions())
	const workers, perWorker, rounds = 6, 25, 98 // 14.700 thao tác (người soát đo trên 14.608); 6 kết nối: container PG 1 GiB dùng chung, đừng tăng
	ops, deadlocks := 0, 0
	var others []string
	for round := 0; round < rounds; round++ {
		ctx, _ := newCtx("r01")
		hot := round%2 == 0
		var mu sync.Mutex
		var wg sync.WaitGroup
		delivered := 0
		start := make(chan struct{})
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				rng := rand.New(rand.NewSource(int64(round*workers + w)))
				<-start
				for i := 0; i < perWorker; i++ {
					isDeliver, err := r01Op(ctx, pg, rng, hot, fmt.Sprintf("r%d-w%d-%d", round, w, i))
					mu.Lock()
					ops++
					switch {
					case err == nil:
						if isDeliver {
							delivered++
						}
					case contracts.FaultOf(err) == contracts.FaultInput: // lệnh bị từ chối: bình thường dưới tải ngẫu nhiên
					case sqlState(err) == "40P01":
						deadlocks++
					default:
						others = append(others, err.Error())
					}
					mu.Unlock()
				}
			}(w)
		}
		close(start)
		wg.Wait()
		if bad := oracle(mustRecords(t, ctx, pg), 60*time.Minute); len(bad) > 0 {
			t.Fatalf("vòng %d: vi phạm bất biến sau tải đồng thời:\n  %s", round, strings.Join(bad, "\n  "))
		}
		if led, err := pg.Ledger(ctx, ""); err != nil || len(led) != delivered {
			t.Fatalf("vòng %d: sổ có %d dòng (lỗi %v), muốn %d = số lần giao thành công (BB1)", round, len(led), err, delivered)
		}
	}
	t.Logf("%d thao tác, %d lỗi 40P01, %d lỗi khác", ops, deadlocks, len(others))
	if deadlocks > 0 || len(others) > 0 {
		if len(others) > 3 {
			others = others[:3]
		}
		t.Fatalf("chết khoá %d / %d thao tác; lỗi khác ngoài lệnh bị từ chối: %q", deadlocks, ops, others)
	}
}

// r01Op chạy MỘT thao tác ngẫu nhiên (giao 45%, quyết định 30%, thu hồi 10%, sửa tin 15%). Giao vào sáu khoá k0..k5 (TR1, TR2, id_conflict xảy ra thường); ba loại còn
// lại nhắm một bản ghi bất kỳ đang có. Trả true nếu là lần giao.
func r01Op(ctx context.Context, st Store, rng *rand.Rand, hot bool, batch string) (bool, error) {
	if rng.Intn(100) < 45 {
		_, err := st.Deliver(ctx, batch, 1, r01Item(rng, fmt.Sprintf("k%d", rng.Intn(6)), hot))
		return true, err
	}
	rs, err := st.Records(ctx)
	if err != nil || len(rs) == 0 {
		return false, err
	}
	r := rs[rng.Intn(len(rs))]
	switch p := rng.Intn(55); {
	case p < 30:
		_, err = st.Decide(ctx, r01Command(rng, r, hot))
	case p < 40:
		err = st.Retract(ctx, r.Key, "ingest", "thử tải")
	default:
		it := r01Item(rng, r.Key, hot)
		it.Raw = "sửa " + batch
		err = st.Amend(ctx, r.Key, it, "ingest", "thử tải")
	}
	return false, err
}

func r01Day(rng *rand.Rand, hot bool) string {
	if hot {
		return "2026-10-19"
	}
	return []string{"2026-10-19", "2026-10-20", "2026-10-21"}[rng.Intn(3)]
}

// r01Item: một người gửi chung và bốn nguyên văn để TR2 trỏ chéo giữa các khoá; một phần HELD không có ô hay HELD đính chính, để CHOOSE / CORRECT / APPLY và sửa tin chuyển
// bản ghi giữa các ô.
func r01Item(rng *rand.Rand, key string, hot bool) Item {
	mods := []func(*Item){withSender("S1"), withRaw(fmt.Sprintf("tin %d", rng.Intn(4))), withDay(r01Day(rng, hot))}
	if !hot && rng.Intn(2) == 0 {
		mods = append(mods, withEntity("E2"))
	}
	switch p := rng.Intn(100); {
	case p < 60:
	case p < 75:
		mods = append(mods, heldFor(ReasonAmbiguousEntity), withEntity(""))
	case p < 85:
		mods = append(mods, heldFor(ReasonCorrectionUnconfirmed), withKind(KindCorrection))
	case p < 93:
		mods = append(mods, heldFor(ReasonNumberReadingUncertain), withValue(""))
	default:
		mods = append(mods, quarantinedFor(ReasonUnparseable), withEntity(""), withDay(""), withValue(""))
	}
	return mk(key, []int{0, 20, 40, 55, 70}[rng.Intn(5)], mods...)
}

// r01Command chọn quyết định theo trạng thái đọc được (có thể đã cũ: lệnh bị từ chối vẫn là một giao dịch ghi audit dưới khoá).
func r01Command(rng *rand.Rand, r Record, hot bool) Command {
	cmd := Command{Key: r.Key, Actor: "pm-1", Why: "thử tải"}
	switch r.Reason {
	case ReasonPossibleDuplicate:
		seen := r.DuplicateOf
		cmd.Decision, cmd.SeenDuplicateOf = DecKeepBoth, &seen
		if rng.Intn(2) == 0 {
			cmd.Decision = DecMarkDuplicate
		}
	case ReasonAmbiguousEntity:
		cmd.Decision, cmd.Entity = DecChoose, "E1"
		if !hot && rng.Intn(2) == 0 {
			cmd.Entity = "E2"
		}
	case ReasonCorrectionUnconfirmed:
		cmd.Decision, cmd.Target = DecApply, fmt.Sprintf("k%d", rng.Intn(6))
	default:
		cmd.Decision, cmd.Entity, cmd.OccurredOn, cmd.Value = DecCorrect, "E1", r01Day(rng, hot), &Quantity{Number: "1000", Unit: "L"}
	}
	return cmd
}
