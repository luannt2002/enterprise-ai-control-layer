package commands

// Ca phát triển (tầng 1) của HOP-DONG §10 chạy lại trên kho PG THẬT: cùng pipeline, cùng lớp bọc probe và rào F7 của K1
// (fake_test.go), chỉ thay kho giả bằng PGStore trên PostgreSQL. Cái kho giả chỉ giả lập được ("chặn rồi báo 23505"), PG thật mới
// chứng minh được: ba ca đua A02, A02b, A04b ở đây do chỉ mục duy nhất và khoá chính của PG phân xử giữa các goroutine THẬT.
// Mọi số đếm lấy bằng kết nối quản trị riêng của harness (pgEnv), không do pipeline tự báo.

import (
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// reclassified đếm các lượt mà C6 đi qua đường phân loại lại sau vi phạm khoá duy nhất (§6.4): bằng chứng rằng PG THẬT đã báo 23505 cho
// lượt đó. Không có nó, một ca đua có thể xanh nhờ lịch chạy mà không hề chạm tới chốt chống trùng của cơ sở dữ liệu.
func reclassified(outs []Outcome) int {
	n := 0
	for _, o := range outs {
		if len(o.Steps) > stWrite && slices.ContainsFunc(o.Steps[stWrite].Output, func(s string) bool { return strings.HasPrefix(s, "phân loại lại") }) {
			n++
		}
	}
	return n
}

func TestPGA01GuiDoiTuanTu(t *testing.T) {
	w := newPGWorld(t)
	p, r := w.user("p-1", "director", ""), req(1, "EX-03", day1)

	o1 := w.run(p, r)
	tk := wantDone(t, o1, false)
	assertSteps(t, o1, "")
	if tk.Diff != "68" || tk.Left.Value != "3850" || tk.Right.Value != "3782" || tk.Left.Unit != "L" || tk.Right.Unit != "L" {
		t.Fatalf("giá trị phiếu sai: %+v", tk)
	}
	o2 := w.run(p, r)
	tk2 := wantDone(t, o2, true)
	if tk2.ID != tk.ID || tk2.Diff != "68" {
		t.Fatalf("phát lại phải trả cùng phiếu: %+v và %+v", tk, tk2)
	}
	assertSteps(t, o2, "")
	w.check(w.tenant, want{tickets: 1, ledger: 1, executed: 1, replayed: 1})
	w.counters(1, 1)
}

// A02 trên PG: N goroutine thật, cùng khoá, cùng nội dung, ép đua bằng F7. N-1 goroutine thua ở chỉ mục duy nhất của phiếu (cùng
// khoá nghiệp vụ) sau khi chờ bên thắng commit; pipeline đọc lại thấy sổ lệnh cùng nội dung và trả DONE phát lại, không ai nhận
// lỗi vi phạm khoá.
func TestPGA02GuiDoiDongThoiF7(t *testing.T) {
	w := newPGWorld(t)
	const n = 6
	var prs []contracts.Principal
	var reqs []Request
	for range n {
		prs, reqs = append(prs, w.user("p-1", "director", "")), append(reqs, req(2, "EX-03", day1))
	}
	outs := w.race(prs, reqs)
	ids, fresh := map[string]bool{}, 0
	for _, o := range outs {
		tk := wantDone(t, o, o.Result.Replayed)
		ids[tk.ID] = true
		if !o.Result.Replayed {
			fresh++
		}
	}
	if len(ids) != 1 || fresh != 1 {
		t.Fatalf("muốn đúng 1 mã phiếu và đúng 1 lần replayed=false, được %d mã, %d lần mới", len(ids), fresh)
	}
	if got := reclassified(outs); got != n-1 {
		t.Fatalf("muốn %d lượt thua đi qua đường phân loại lại sau 23505 của PG, được %d (ca đua không chạm chốt chống trùng)", n-1, got)
	}
	w.check(w.tenant, want{tickets: 1, ledger: 1, executed: 1, replayed: n - 1})
	w.counters(n, n-1)
}

// A02b trên PG: cùng command_id, ba nội dung khác nhau (ba máy). Ba phiếu khác khoá nghiệp vụ nên cả ba INSERT phiếu đều qua; ba
// INSERT sổ lệnh đụng khoá chính, hai lần thua chờ bên thắng commit rồi bị 23505 và huỷ CẢ phiếu của mình.
func TestPGA02bDuaKhacNoiDungF7(t *testing.T) {
	w := newPGWorld(t)
	prs := []contracts.Principal{w.user("p-1", "director", ""), w.user("p-1", "director", ""), w.user("p-1", "director", "")}
	reqs := []Request{req(3, "EX-03", day1), req(3, "EX-04", day1), req(3, "EX-05", day1)}
	outs := w.race(prs, reqs)
	done, reused := 0, 0
	for _, o := range outs {
		switch o.Result.ReasonCode {
		case "":
			wantDone(t, o, false)
			done++
		case ReasonIdempotencyKeyReused:
			wantFail(t, o, StatusFailed, ReasonIdempotencyKeyReused)
			reused++
		default:
			t.Fatalf("kết quả ngoài {DONE, IDEMPOTENCY_KEY_REUSED}: %+v (cause=%v)", o.Result, o.Cause)
		}
	}
	if done != 1 || reused != 2 {
		t.Fatalf("muốn DONE=1 IDEMPOTENCY_KEY_REUSED=2, được %d và %d", done, reused)
	}
	if got := reclassified(outs); got != 2 {
		t.Fatalf("muốn 2 lượt thua đi qua đường phân loại lại sau 23505 của khoá chính sổ lệnh, được %d", got)
	}
	w.check(w.tenant, want{tickets: 1, ledger: 1, executed: 1, refused: 2}) // phiếu của hai bên thua đã bị huỷ, không mồ côi
}

func TestPGA03CungKhoaKhacNoiDung(t *testing.T) {
	for name, second := range map[string]Request{"khác máy": req(4, "EX-04", day1), "khác ngày": req(4, "EX-03", day2)} {
		t.Run(name, func(t *testing.T) {
			w := newPGWorld(t)
			p := w.user("p-1", "director", "")
			first := wantDone(t, w.run(p, req(4, "EX-03", day1)), false)
			o := w.run(p, second)
			wantFail(t, o, StatusFailed, ReasonIdempotencyKeyReused)
			assertSteps(t, o, "C4b")
			if ids := w.env.ticketIDs(w.tenant); len(ids) != 1 || ids[0] != first.ID {
				t.Fatalf("phiếu cũ phải nguyên vẹn: %v", ids)
			}
			w.check(w.tenant, want{tickets: 1, ledger: 1, executed: 1, refused: 1})
		})
	}
}

func TestPGA04KhacKhoaCungViecTuanTu(t *testing.T) {
	w := newPGWorld(t)
	p := w.user("p-1", "director", "")
	first := wantDone(t, w.run(p, req(5, "EX-03", day1)), false)
	o := w.run(p, req(6, "EX-03", day1))
	wantFail(t, o, StatusFailed, ReasonDuplicateOpenTicket)
	if tk := o.Result.Ticket; tk == nil || tk.ID != first.ID || tk.Status != TicketOpen {
		t.Fatalf("phải trả mã và trạng thái phiếu cũ: %+v", tk)
	}
	assertSteps(t, o, "C5") // từ chối đến từ R5 ở C5, không phải từ chỉ mục duy nhất ở C6
	w.check(w.tenant, want{tickets: 1, ledger: 1, executed: 1, refused: 1})
	w.counters(1, 1)
	if n := w.env.auditRows(w.tenant, AuditRefused, ReasonDuplicateOpenTicket, "p-1"); n != 1 {
		t.Fatalf("audit REFUSED DUPLICATE_OPEN_TICKET = %d, muốn 1", n)
	}
}

// A04b trên PG: N goroutine thật, KHÁC khoá lệnh, cùng (máy, chỉ số, ngày). Chỉ mục duy nhất từng phần của phiếu OPEN phân xử:
// đúng một bên thắng, các bên còn lại nhận DUPLICATE_OPEN_TICKET kèm mã phiếu của bên thắng.
func TestPGA04bDuaKhacKhoaF7(t *testing.T) {
	w := newPGWorld(t)
	const n = 6
	var prs []contracts.Principal
	var reqs []Request
	for i := range n {
		prs, reqs = append(prs, w.user("p-1", "director", "")), append(reqs, req(10+i, "EX-03", day1))
	}
	outs := w.race(prs, reqs)
	var winner string
	done, dup := 0, 0
	for _, o := range outs {
		if o.Result.Status == StatusDone {
			winner = wantDone(t, o, false).ID
			done++
		}
	}
	for _, o := range outs {
		if o.Result.Status == StatusDone {
			continue
		}
		wantFail(t, o, StatusFailed, ReasonDuplicateOpenTicket)
		if o.Result.Ticket == nil || o.Result.Ticket.ID != winner {
			t.Fatalf("bên thua phải nhận mã phiếu của lần thắng %s: %+v (cause=%v)", winner, o.Result.Ticket, o.Cause)
		}
		dup++
	}
	if done != 1 || dup != n-1 {
		t.Fatalf("muốn DONE=1 DUPLICATE_OPEN_TICKET=%d, được %d và %d", n-1, done, dup)
	}
	if got := reclassified(outs); got != n-1 {
		t.Fatalf("muốn %d lượt thua đi qua đường phân loại lại sau 23505 của chỉ mục duy nhất từng phần, được %d", n-1, got)
	}
	w.check(w.tenant, want{tickets: 1, ledger: 1, executed: 1, refused: n - 1})
}

func TestPGA05NguoiGoiKhacCungKhoa(t *testing.T) {
	w := newPGWorld(t)
	first := wantDone(t, w.run(w.user("p-1", "director", ""), req(7, "EX-03", day1)), false)
	o := w.run(w.user("p-2", "accountant", ""), req(7, "EX-03", day1)) // cùng khoá, cùng nội dung, người khác
	wantFail(t, o, StatusFailed, ReasonDuplicateOpenTicket)
	if o.Result.Replayed || o.Result.Ticket == nil || o.Result.Ticket.ID != first.ID {
		t.Fatalf("không được phát lại kết quả của người kia: %+v", o.Result)
	}
	if n := w.env.auditRows(w.tenant, AuditExecuted, "", "p-1"); n != 1 {
		t.Fatalf("EXECUTED của p-1 = %d, muốn 1", n)
	}
	if n := w.env.auditRows(w.tenant, AuditRefused, ReasonDuplicateOpenTicket, "p-2"); n != 1 {
		t.Fatalf("REFUSED của p-2 = %d, muốn 1", n)
	}
	w.check(w.tenant, want{tickets: 1, ledger: 1, executed: 1, refused: 1})
}

// A06 trên PG: hai tenant, cùng khoá lệnh, cùng người gọi: mỗi bên một phiếu của mình; RLS không cho bên này thấy dòng bên kia.
func TestPGA06HaiTenantCungKhoa(t *testing.T) {
	w := newPGWorld(t)
	tb, pb := w.other()
	pa := w.user("p-1", "director", "")
	pbUser := contracts.Principal{ID: "p-1", TenantID: tb, Kind: contracts.PrincipalUser, Roles: []string{"director"}}
	r := req(8, "EX-03", day1)
	ta := wantDone(t, w.run(pa, r), false)
	tbk := wantDone(t, pb.Run(ctxOf(tb), pbUser, r), false)
	if ta.ID == tbk.ID {
		t.Fatalf("hai tenant phải có hai phiếu khác nhau")
	}
	w.check(w.tenant, want{tickets: 1, ledger: 1, executed: 1})
	w.check(tb, want{tickets: 1, ledger: 1, executed: 1})
	st, err := w.env.pg.ReadBack(ctxOf(tb), "p-1", r.CommandID, ta.ID) // ngữ cảnh của B hỏi mã phiếu của A
	if err != nil || len(st.Tickets) != 0 {
		t.Fatalf("tenant B không được thấy phiếu của tenant A: %+v, %v", st.Tickets, err)
	}
}

// A19 trên PG: F1 trước khi mở giao dịch (probe).
func TestPGA19F1TruocKhiMoGiaoDich(t *testing.T) {
	w := newPGWorld(t)
	w.probe.f1 = 1
	p, r := w.user("p-1", "director", ""), req(70, "EX-03", day1)
	o := w.run(p, r)
	wantFail(t, o, StatusFailed, ReasonStoreUnavailable)
	assertSteps(t, o, "C6")
	w.check(w.tenant, want{})
	w.counters(1, 0)
	wantDone(t, w.run(p, r), false)
	w.check(w.tenant, want{tickets: 1, ledger: 1, executed: 1})
}

// A20 trên PG: F2 qua núm của kho (export_test.go). INSERT audit giữa giao dịch hỏng SAU khi phiếu và sổ lệnh đã được INSERT trong
// cùng giao dịch THẬT: rollback phải huỷ cả ba, không còn phiếu hay dòng sổ lệnh mồ côi.
func TestPGA20F2LoiInsertAuditHuyCaBaDong(t *testing.T) {
	w := newPGWorld(t)
	w.env.pg.failAuditInsertTimes(1)
	p, r := w.user("p-1", "director", ""), req(71, "EX-03", day1)
	o := w.run(p, r)
	wantFail(t, o, StatusFailed, ReasonStoreUnavailable)
	assertSteps(t, o, "C6")
	w.check(w.tenant, want{}) // 0 phiếu, 0 sổ lệnh, 0 EXECUTED
	if n := w.env.countWhere("discrepancy_tickets", w.tenant, ""); n != 0 {
		t.Fatalf("phiếu mồ côi sau rollback: %d", n)
	}
	wantDone(t, w.run(p, r), false) // núm chỉ hỏng một lần: gửi lại thì ghi bình thường
	w.check(w.tenant, want{tickets: 1, ledger: 1, executed: 1})
}

// A21 trên PG: timeout sau khi ghi. (a) F3: commit THẬT rồi báo COMMIT_UNKNOWN; (b) F4: dựng pipeline mới.
func TestPGA21TimeoutSauKhiGhi(t *testing.T) {
	t.Run("F3 mất ack", func(t *testing.T) {
		w := newPGWorld(t)
		w.probe.f3 = 1
		p, r := w.user("p-1", "director", ""), req(72, "EX-03", day1)
		o := w.run(p, r)
		wantFail(t, o, StatusUnknown, ReasonCommitUnknown)
		assertSteps(t, o, "C6")
		w.check(w.tenant, want{tickets: 1, ledger: 1, executed: 1})
		stored := w.onlyTicketID()
		again := wantDone(t, w.run(p, r), true)
		if again.ID != stored {
			t.Fatalf("gửi lại phải phát lại đúng phiếu đã commit: %s và %s", again.ID, stored)
		}
		w.check(w.tenant, want{tickets: 1, ledger: 1, executed: 1, replayed: 1})
	})
	t.Run("F4 mất phản hồi, dựng pipeline mới", func(t *testing.T) {
		w := newPGWorld(t)
		p, r := w.user("p-1", "director", ""), req(73, "EX-03", day1)
		_ = w.run(p, r)
		w.restart()
		first := w.onlyTicketID()
		again := wantDone(t, w.run(p, r), true)
		if again.ID != first {
			t.Fatalf("sau khởi động lại vẫn phải phát lại đúng phiếu: %s và %s", again.ID, first)
		}
		w.check(w.tenant, want{tickets: 1, ledger: 1, executed: 1, replayed: 1})
	})
}

// A22 trên PG: COMMIT đến muộn. Hai thứ tự; cả hai đều chỉ còn đúng một phiếu. Thứ tự "đến sau" cho PG thật chặn lần ghi muộn bằng
// chính chỉ mục duy nhất.
func TestPGA22CommitDenMuon(t *testing.T) {
	t.Run("commit đến trước lần gửi lại", func(t *testing.T) {
		w := newPGWorld(t)
		w.probe.f5 = 1
		p, r := w.user("p-1", "director", ""), req(74, "EX-03", day1)
		wantFail(t, w.run(p, r), StatusUnknown, ReasonCommitUnknown)
		w.check(w.tenant, want{})
		w.probe.releaseDelayed()
		w.check(w.tenant, want{tickets: 1, ledger: 1, executed: 1})
		wantDone(t, w.run(p, r), true)
		w.check(w.tenant, want{tickets: 1, ledger: 1, executed: 1, replayed: 1})
	})
	t.Run("commit đến sau lần gửi lại", func(t *testing.T) {
		w := newPGWorld(t)
		w.probe.f5 = 1
		p, r := w.user("p-1", "director", ""), req(75, "EX-03", day1)
		wantFail(t, w.run(p, r), StatusUnknown, ReasonCommitUnknown)
		wantDone(t, w.run(p, r), false)
		w.probe.releaseDelayed() // lần ghi muộn đụng chỉ mục duy nhất của PG và bị chặn
		w.check(w.tenant, want{tickets: 1, ledger: 1, executed: 1})
	})
}

// A23 trên PG: đọc lại hỏng thì không bao giờ DONE; lệch trường hoặc không thấy thì có audit VERIFY_FAILED.
func TestPGA23DocLaiHong(t *testing.T) {
	t.Run("F6 lỗi khi đọc lại", func(t *testing.T) {
		w := newPGWorld(t)
		p := w.user("p-1", "director", "")
		w.probe.f6fail = 1
		r := req(76, "EX-03", day1)
		o := w.run(p, r)
		wantFail(t, o, StatusUnknown, ReasonReadbackUnavailable)
		assertSteps(t, o, "C7")
		w.check(w.tenant, want{tickets: 1, ledger: 1, executed: 1})
		wantDone(t, w.run(p, r), true)
	})
	for _, mode := range []string{"missing", "diff"} {
		t.Run("F6 đọc lại lệch: "+mode, func(t *testing.T) {
			w := newPGWorld(t)
			w.probe.f6stale, w.probe.staleMode = 1, mode
			o := w.run(w.user("p-1", "director", ""), req(77, "EX-03", day1))
			wantFail(t, o, StatusUnknown, ReasonVerifyMismatch)
			assertSteps(t, o, "C7")
			w.check(w.tenant, want{tickets: 1, ledger: 1, executed: 1, verifyFailed: 1})
		})
	}
	// C2 trên PG: dòng EXECUTED đọc lại từ DB thật phải khớp mọi cột đọc được (ca DONE ở trên chứng minh không lệch giả).
	for _, mode := range []string{"audit-fp", "audit-trace", "audit-roles", "audit-reason"} {
		t.Run("F6 audit EXECUTED lệch: "+mode, func(t *testing.T) {
			w := newPGWorld(t)
			w.probe.f6stale, w.probe.staleMode = 1, mode
			o := w.run(w.user("p-1", "director", ""), req(79, "EX-03", day1))
			wantFail(t, o, StatusUnknown, ReasonVerifyMismatch)
			w.check(w.tenant, want{tickets: 1, ledger: 1, executed: 1, verifyFailed: 1})
		})
	}
	for _, mode := range []string{"missing", "subject"} {
		t.Run("phát lại cũng đọc lại: "+mode, func(t *testing.T) {
			w := newPGWorld(t)
			p, r := w.user("p-1", "director", ""), req(78, "EX-03", day1)
			wantDone(t, w.run(p, r), false)
			w.probe.f6stale, w.probe.staleMode = 1, mode
			wantFail(t, w.run(p, r), StatusUnknown, ReasonVerifyMismatch)
			w.check(w.tenant, want{tickets: 1, ledger: 1, executed: 1, verifyFailed: 1})
		})
	}
}

// Bão đua trên PG: nhiều goroutine thật, ba kiểu xung đột TRỘN lẫn trong một lượt thả rào (cùng khoá cùng nội dung; cùng khoá khác
// nội dung; khác khoá cùng việc), lặp nhiều vòng trên các (máy, ngày) khác nhau. Không đoán ai thắng; chỉ kiểm bất biến mà mọi lịch
// chạy phải giữ: không bao giờ hai phiếu mở cùng khoá nghiệp vụ, không bao giờ hai dòng sổ lệnh cùng khoá, không INTERNAL_ERROR hay
// STORE_UNAVAILABLE (hạn chót đủ dài), mọi DONE trỏ tới phiếu có thật, và sổ lệnh khớp phiếu: dòng sổ lệnh nào cũng có phiếu OPEN.
func TestPGBaoDuaGiuBatBien(t *testing.T) {
	const rounds = 4
	for round := range rounds {
		w := newPGWorld(t)
		key, day := []string{"EX-03", "EX-04", "EX-05", "EX-08"}[round], day1
		// 8 lời gọi, mọi lời gọi đều tới được C6 (rào F7 đợi đủ 8): 3 bản y hệt (cmd 100) + 2 bản cùng cmd 100 khác máy (đều có số
		// liệu, chênh lệch khác 0) + 3 khoá lệnh khác (cmd 101..103) cùng việc với bản y hệt.
		reqs := []Request{
			req(100, key, day), req(100, key, day), req(100, key, day),
			req(100, "EX-10", day), req(100, "EX-09", day),
			req(101, key, day), req(102, key, day), req(103, key, day),
		}
		var prs []contracts.Principal
		for range reqs {
			prs = append(prs, w.user("p-1", "director", ""))
		}
		outs := w.race(prs, reqs)
		doneIDs := map[string]bool{}
		for i, o := range outs {
			switch o.Result.ReasonCode {
			case "":
				doneIDs[o.Result.Ticket.ID] = true
			case ReasonIdempotencyKeyReused, ReasonDuplicateOpenTicket:
			default:
				t.Fatalf("vòng %d lời gọi %d: kết cục ngoài dự kiến %s/%s (cause=%v)", round, i, o.Result.Status, o.Result.ReasonCode, o.Cause)
			}
		}
		ids := w.env.ticketIDs(w.tenant)
		if n := w.env.openTickets(w.tenant); n != len(ids) || n > 3 {
			t.Fatalf("vòng %d: %d phiếu OPEN, %d dòng phiếu", round, n, len(ids))
		}
		// bất biến cốt lõi: không hai phiếu OPEN cùng khoá nghiệp vụ, không hai dòng sổ lệnh cùng khoá
		var dupTickets, dupLedger int
		if err := w.env.admin.QueryRow(t.Context(), `
			SELECT (SELECT count(*) FROM (SELECT 1 FROM discrepancy_tickets WHERE tenant_id = $1 AND status = 'OPEN'
			          GROUP BY subject_key, metric, day HAVING count(*) > 1) x),
			       (SELECT count(*) FROM (SELECT 1 FROM commands WHERE tenant_id = $1 GROUP BY caller_id, command_id HAVING count(*) > 1) y)`,
			string(w.tenant)).Scan(&dupTickets, &dupLedger); err != nil {
			t.Fatal(err)
		}
		if dupTickets != 0 || dupLedger != 0 {
			t.Fatalf("vòng %d: phiếu OPEN trùng khoá nghiệp vụ = %d, dòng sổ lệnh trùng khoá = %d", round, dupTickets, dupLedger)
		}
		for id := range doneIDs {
			if n := w.env.countWhere("discrepancy_tickets", w.tenant, "id = $2", id); n != 1 {
				t.Fatalf("vòng %d: DONE trỏ tới phiếu %s không có ở PG", round, id)
			}
		}
		orphan := w.env.countWhere("commands", w.tenant, "ticket_id NOT IN (SELECT id FROM discrepancy_tickets WHERE tenant_id = $1)")
		if orphan != 0 {
			t.Fatalf("vòng %d: %d dòng sổ lệnh không có phiếu", round, orphan)
		}
		if w.env.ledgerRows(w.tenant) != len(ids) {
			t.Fatalf("vòng %d: %d dòng sổ lệnh cho %d phiếu: mỗi lệnh đã ghi phải có đúng một phiếu", round, w.env.ledgerRows(w.tenant), len(ids))
		}
	}
}

// Ba kho cùng lúc dùng chung một Store PG (một pool) nhưng khác tenant: ghi song song không chặn nhau và không lẫn dòng.
func TestPGNhieuTenantGhiSongSong(t *testing.T) {
	env := newPGEnv(t)
	const tenants = 4
	ws := make([]*pgWorld, tenants)
	for i := range ws {
		ws[i] = env.newWorld()
	}
	var wg sync.WaitGroup
	results := make([]Outcome, tenants)
	for i := range ws {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = ws[i].run(ws[i].user("p-1", "director", ""), req(200, "EX-03", day1)) // cùng khoá lệnh, khác tenant
		}()
	}
	wg.Wait()
	for i, o := range results {
		if o.Result.Status != StatusDone {
			t.Fatalf("tenant %d: %+v (cause=%v)", i, o.Result, o.Cause)
		}
		ws[i].check(ws[i].tenant, want{tickets: 1, ledger: 1, executed: 1})
	}
}
