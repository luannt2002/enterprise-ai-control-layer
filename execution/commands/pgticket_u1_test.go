package commands

// Thí nghiệm U1 (HOP-DONG §15; ADR-0011 "Chứng minh" bước 4). KHÔNG phải mã sản phẩm: đo trên PostgreSQL THẬT xem trình điều khiển
// pgx trả gì ở các điểm quanh COMMIT, rồi khẳng định rằng phân loại của pgticket.go (classifyStatement, classifyCommit, WriteTicket)
// khớp bảng đo. Nếu một bản pgx sau này đổi hình dạng lỗi, ca ở đây đỏ và buộc đo lại thay vì phân loại sai âm thầm.
//
// Mọi bẫy dựng trong MỘT database tạm do chính test tạo (tên bắt đầu bằng tên database đang test, xoá khi xong), không đụng database
// đang dùng chung: trigger gài ở bảng audit_log (BEFORE INSERT để tự kết thúc backend hoặc báo lỗi giữa giao dịch; CONSTRAINT
// TRIGGER DEFERRABLE INITIALLY DEFERRED để mở một cửa sổ chạy ngay TRONG lệnh COMMIT bằng pg_sleep), kích hoạt bởi dấu hiệu nằm ở
// reason_detail của dòng audit EXECUTED. Lệnh pg_terminate_backend chỉ nhắm backend có application_name riêng của ca và thuộc đúng
// database tạm.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

const u1HazardSQL = `
CREATE FUNCTION k2_stmt_hazard() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS $f$
BEGIN
  IF NEW.reason_detail = 'k2:kill-self' THEN
    PERFORM pg_terminate_backend(pg_backend_pid());
    PERFORM pg_sleep(10);
  END IF;
  RETURN NEW;
END $f$;
CREATE TRIGGER k2_stmt_hazard BEFORE INSERT ON audit_log FOR EACH ROW EXECUTE FUNCTION k2_stmt_hazard();

CREATE FUNCTION k2_commit_hazard() RETURNS trigger LANGUAGE plpgsql AS $f$
BEGIN
  IF NEW.reason_detail LIKE 'k2:commit-sleep:%' THEN
    PERFORM pg_sleep(split_part(NEW.reason_detail, ':', 3)::float8);
  ELSIF NEW.reason_detail = 'k2:commit-raise' THEN
    RAISE EXCEPTION 'k2: lỗi giả lập ở COMMIT' USING ERRCODE = 'P0001';
  ELSIF NEW.reason_detail = 'k2:commit-raise-23505' THEN
    RAISE EXCEPTION 'k2: vi phạm duy nhất giả lập ở COMMIT' USING ERRCODE = '23505';
  END IF;
  RETURN NULL;
END $f$;
CREATE CONSTRAINT TRIGGER k2_commit_hazard AFTER INSERT ON audit_log DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION k2_commit_hazard();
`

// u1Row là một dòng của bảng đo.
type u1Row struct {
	id, how, raw, safe, class, truth string
}

type u1Lab struct {
	*pgEnv
	mu   sync.Mutex
	rows []u1Row
}

// newU1Lab dựng database tạm (newScratchEnv) rồi gài bẫy; dọn khi test xong.
func newU1Lab(t *testing.T) *u1Lab {
	t.Helper()
	lab := &u1Lab{pgEnv: newScratchEnv(t, "u1")}
	if _, err := lab.admin.Exec(context.Background(), u1HazardSQL); err != nil {
		t.Fatalf("gài bẫy: %v", err)
	}
	return lab
}

// withParams trả DSN dạng URL với các tham số kết nối bổ sung (application_name, lock_timeout...); tham số lạ với pgx.Connect trần
// (pool_max_conns) chỉ dùng cho pgstore.Open.
func withParams(t *testing.T, dsn string, params map[string]string) string {
	t.Helper()
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// newPG mở một Store riêng (pool 2 kết nối) mang application_name của ca và các tham số phiên (lock_timeout, statement_timeout).
func (l *u1Lab) newPG(app string, session map[string]string) *PGStore {
	l.t.Helper()
	params := map[string]string{"application_name": app, "pool_max_conns": "2"}
	dsn := l.appDSN
	for k, v := range session {
		if k == "@host" { // không phải tham số phiên: đổi địa chỉ tới (cầu TCP của thí nghiệm)
			u, err := url.Parse(dsn)
			if err != nil {
				l.t.Fatal(err)
			}
			u.Host = v
			dsn = u.String()
			continue
		}
		params[k] = v
	}
	st, err := pgstore.Open(context.Background(), withParams(l.t, dsn, params))
	if err != nil {
		l.t.Fatal(err)
	}
	l.t.Cleanup(func() { closeWithin(l.t, "pool của ca "+app, st.Close) })
	return NewPGStore(st)
}

// unit dựng đơn vị ghi phiếu mang dấu hiệu kích hoạt bẫy ở reason_detail của dòng audit EXECUTED.
func (l *u1Lab) unit(n int, key, marker string) WriteUnit {
	u := richUnit(n, "p-1", key)
	u.Audit.ReasonDetail = marker
	return u
}

// begin mở giao dịch đã đặt tenant bằng chính cách mà WriteTicket mở (BeginTenantTx).
func (l *u1Lab) begin(ctx context.Context, st *PGStore) pgx.Tx {
	l.t.Helper()
	tx, err := st.db.BeginTenantTx(ctx)
	if err != nil {
		l.t.Fatal(err)
	}
	return tx
}

// pidOf hỏi backend của giao dịch.
func (l *u1Lab) pidOf(ctx context.Context, tx pgx.Tx) int {
	l.t.Helper()
	var pid int
	if err := tx.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
		l.t.Fatal(err)
	}
	return pid
}

// kill kết thúc đúng backend pid, và chỉ khi nó thuộc database tạm (kết nối quản trị của lab nối vào database tạm).
func (l *u1Lab) kill(pid int) {
	l.t.Helper()
	var ok bool
	if err := l.admin.QueryRow(context.Background(), "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE pid = $1 AND datname = current_database()", pid).Scan(&ok); err != nil || !ok {
		l.t.Fatalf("không kết thúc được backend %d trong database tạm: %v %v", pid, ok, err)
	}
}

// waitGone đợi backend pid biến mất khỏi pg_stat_activity.
func (l *u1Lab) waitGone(pid int) {
	l.t.Helper()
	l.eventually(5*time.Second, "backend vẫn còn", func() bool {
		var n int
		if err := l.admin.QueryRow(context.Background(), "SELECT count(*) FROM pg_stat_activity WHERE pid = $1", pid).Scan(&n); err != nil {
			l.t.Fatal(err)
		}
		return n == 0
	})
}

// waitBackend đợi một backend của ca (theo application_name) thoả điều kiện cond và trả pid của nó.
func (l *u1Lab) waitBackend(app, cond string) int {
	l.t.Helper()
	var pid int
	l.eventually(8*time.Second, "chưa thấy backend của ca ở trạng thái "+cond, func() bool {
		err := l.admin.QueryRow(context.Background(),
			"SELECT pid FROM pg_stat_activity WHERE datname = current_database() AND application_name = $1 AND "+cond+" LIMIT 1", app).Scan(&pid)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			l.t.Fatal(err)
		}
		return err == nil
	})
	return pid
}

func (l *u1Lab) eventually(limit time.Duration, msg string, ok func() bool) {
	l.t.Helper()
	for deadline := time.Now().Add(limit); time.Now().Before(deadline); time.Sleep(15 * time.Millisecond) {
		if ok() {
			return
		}
	}
	l.t.Fatalf("quá %v: %s", limit, msg)
}

// use đổi *testing.T mà các hàm trợ giúp của lab báo lỗi vào: mỗi ca con chạy trong goroutine riêng và chỉ được gọi Fatal của chính nó.
// Các ca con chạy tuần tự nên việc đổi này không đua nhau.
func (l *u1Lab) use(t *testing.T) { t.Helper(); l.pgEnv.t = t }

// ticketsOf đếm phiếu của tenant bằng kết nối quản trị.
func (l *u1Lab) ticketsOf(tn contracts.TenantID) int {
	return l.countWhere("discrepancy_tickets", tn, "")
}

// settle đợi một nhịp ngắn rồi đếm lại: dùng cho ca "không được xuất hiện" để bắt dòng hiện muộn.
func (l *u1Lab) settle(tn contracts.TenantID, d time.Duration) int {
	time.Sleep(d)
	return l.ticketsOf(tn)
}

// chain mô tả chuỗi kiểu lỗi pgx trả về, từ ngoài vào trong, kèm SQLSTATE và mức nghiêm trọng của lỗi PG.
func chain(err error) string {
	var parts []string
	var walk func(error)
	walk = func(e error) {
		if e == nil || e == ErrBeforeCommit || e == ErrCommitUnknown || e == ErrUniqueViolation {
			return // sentinel của Store không phải lỗi pgx: bỏ, để bảng chỉ cho thấy cái pgx trả
		}
		if pe, ok := e.(*pgconn.PgError); ok {
			parts = append(parts, fmt.Sprintf("PgError{%s %s}", pe.SeverityUnlocalized, pe.Code))
			return
		}
		switch x := e.(type) {
		case interface{ Unwrap() []error }:
			for _, c := range x.Unwrap() {
				walk(c)
			}
			return
		case interface{ Unwrap() error }:
			if name := fmt.Sprintf("%T", e); !strings.HasPrefix(name, "*fmt.wrap") { // lớp bọc của fmt.Errorf không phải pgx
				parts = append(parts, name)
			}
			walk(x.Unwrap())
			return
		}
		parts = append(parts, fmt.Sprintf("%T(%v)", e, e))
	}
	walk(err)
	return strings.Join(parts, " > ")
}

// sentinel nói lỗi đã được phân loại thành loại nào của Store.
func sentinel(err error) string {
	switch {
	case err == nil:
		return "nil"
	case errors.Is(err, ErrUniqueViolation):
		return "ErrUniqueViolation"
	case errors.Is(err, ErrBeforeCommit):
		return "ErrBeforeCommit"
	case errors.Is(err, ErrCommitUnknown):
		return "ErrCommitUnknown"
	}
	return "KHÔNG PHÂN LOẠI"
}

func (l *u1Lab) record(id, how string, raw, classified error, truth string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rows = append(l.rows, u1Row{id: id, how: how, raw: chain(raw), safe: strconv.FormatBool(pgconn.SafeToRetry(raw)), class: sentinel(classified), truth: truth})
}

func truthOf(n int) string {
	if n == 0 {
		return "0 phiếu"
	}
	return fmt.Sprintf("%d phiếu", n)
}

// Bảng đo U1. Mỗi ca khẳng định cả hình dạng lỗi pgx (chỗ phân loại dựa vào) lẫn kết cục phân loại lẫn sự thật ở DB.
func TestU1RanhGioiCommit(t *testing.T) {
	lab := newU1Lab(t)
	ctx := context.Background()
	wantClass := func(t *testing.T, id string, got error, want error) {
		t.Helper()
		if !errors.Is(got, want) {
			t.Errorf("%s: muốn %v, được %s (%v)", id, want, sentinel(got), chain(got))
		}
		for _, other := range []error{ErrBeforeCommit, ErrCommitUnknown, ErrUniqueViolation} {
			if other != want && errors.Is(got, other) {
				t.Errorf("%s: không được vừa là %v vừa là %v", id, want, other)
			}
		}
	}
	asPg := func(err error) *pgconn.PgError {
		var pe *pgconn.PgError
		if errors.As(err, &pe) {
			return pe
		}
		return nil
	}

	// (a) backend bị kết thúc TRƯỚC COMMIT -------------------------------------------------------------------------------
	t.Run("a1 backend tự kết thúc ở INSERT audit", func(t *testing.T) {
		lab.use(t)
		st, tn := lab.newPG("k2_u1_a1", nil), lab.tenant("a1")
		err := st.WriteTicket(ctxOf(tn), lab.unit(1, "EX-03", "k2:kill-self"))
		wantClass(t, "a1", err, ErrBeforeCommit)
		lab.record("a1", "trigger BEFORE INSERT audit gọi pg_terminate_backend(pg_backend_pid())", err, err, truthOf(lab.settle(tn, 100*time.Millisecond)))
		if lab.ticketsOf(tn) != 0 {
			t.Errorf("a1: backend chết trước COMMIT mà còn phiếu")
		}
	})
	t.Run("a1b backend bị kết thúc khi đang rảnh giữa hai câu lệnh", func(t *testing.T) {
		lab.use(t)
		st, tn := lab.newPG("k2_u1_a1b", nil), lab.tenant("a1b")
		ctx := ctxOf(tn)
		tx := lab.begin(ctx, st)
		pid := lab.pidOf(ctx, tx)
		lab.kill(pid)
		lab.waitGone(pid)
		raw := st.insertUnit(ctx, tx, lab.unit(2, "EX-03", ""))
		if raw == nil {
			t.Fatalf("a1b: INSERT trên kết nối đã chết phải lỗi")
		}
		_ = tx.Rollback(ctx)
		got := classifyStatement(raw)
		wantClass(t, "a1b", got, ErrBeforeCommit)
		lab.record("a1b", "admin pg_terminate_backend khi giao dịch rảnh; câu INSERT kế tiếp", raw, got, truthOf(lab.ticketsOf(tn)))
	})
	t.Run("a2 backend bị kết thúc giữa INSERT cuối và COMMIT", func(t *testing.T) {
		lab.use(t)
		st, tn := lab.newPG("k2_u1_a2", nil), lab.tenant("a2")
		ctx := ctxOf(tn)
		tx := lab.begin(ctx, st)
		if err := st.insertUnit(ctx, tx, lab.unit(3, "EX-03", "")); err != nil {
			t.Fatal(err)
		}
		pid := lab.pidOf(ctx, tx)
		lab.kill(pid)
		lab.waitGone(pid)
		raw := tx.Commit(ctx)
		if raw == nil {
			t.Fatalf("a2: COMMIT trên kết nối đã chết phải lỗi")
		}
		got := classifyCommit(raw)
		lab.record("a2", "admin pg_terminate_backend sau INSERT cuối, rồi tx.Commit", raw, got, truthOf(lab.ticketsOf(tn)))
		// Đo: lời FATAL 57P01 của máy chủ đã nằm sẵn trong bộ đệm nhận và COMMIT vẫn "gửi được" vào ổ cắm, nên pgx trả PgError FATAL,
		// KHÔNG SafeToRetry. Từ phía bên gọi không phân biệt được với "COMMIT đã tới rồi mới bị kết thúc" ⇒ không biết.
		if pe := asPg(raw); pe == nil || pe.SeverityUnlocalized != "FATAL" || pe.Code != "57P01" || pgconn.SafeToRetry(raw) {
			t.Errorf("a2: muốn PgError FATAL 57P01, không SafeToRetry; được %v (SafeToRetry=%v)", chain(raw), pgconn.SafeToRetry(raw))
		}
		wantClass(t, "a2", got, ErrCommitUnknown)
		if lab.ticketsOf(tn) != 0 {
			t.Errorf("a2: backend chết trước COMMIT mà còn phiếu")
		}
	})

	// (b) backend bị kết thúc ĐANG COMMIT -----------------------------------------------------------------------------------
	t.Run("b backend bị kết thúc đang chạy COMMIT", func(t *testing.T) {
		lab.use(t)
		st, tn := lab.newPG("k2_u1_b", nil), lab.tenant("b")
		done := make(chan error, 1)
		go func() { done <- st.WriteTicket(ctxOf(tn), lab.unit(4, "EX-03", "k2:commit-sleep:10")) }()
		pid := lab.waitBackend("k2_u1_b", "state = 'active' AND query = 'commit'")
		lab.kill(pid)
		var err error
		select {
		case err = <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("b: WriteTicket không trả sau khi backend bị kết thúc")
		}
		wantClass(t, "b", err, ErrCommitUnknown)
		lab.record("b", "trigger hoãn pg_sleep(10) trong COMMIT; admin pg_terminate_backend giữa lúc ngủ", err, err, truthOf(lab.settle(tn, 300*time.Millisecond)))
		if lab.ticketsOf(tn) != 0 {
			t.Errorf("b: backend chết trong COMMIT trước khi xong mà còn phiếu")
		}
	})

	// (c) ngữ cảnh hết hạn trước / trong COMMIT -------------------------------------------------------------------------------
	t.Run("c1 ngữ cảnh bị huỷ ngay trước COMMIT", func(t *testing.T) {
		lab.use(t)
		st, tn := lab.newPG("k2_u1_c1", nil), lab.tenant("c1")
		base := ctxOf(tn)
		ctx, cancel := context.WithCancel(base)
		defer cancel()
		tx := lab.begin(ctx, st)
		if err := st.insertUnit(ctx, tx, lab.unit(5, "EX-03", "")); err != nil {
			t.Fatal(err)
		}
		cancel()
		raw := tx.Commit(ctx)
		if raw == nil || !pgconn.SafeToRetry(raw) {
			t.Fatalf("c1: COMMIT với ngữ cảnh đã huỷ phải lỗi và SafeToRetry (chưa gửi gì), được %v", chain(raw))
		}
		got := classifyCommit(raw)
		wantClass(t, "c1", got, ErrBeforeCommit)
		lab.record("c1", "cancel() rồi tx.Commit (ngữ cảnh đã xong trước khi gửi COMMIT)", raw, got, truthOf(lab.settle(tn, 200*time.Millisecond)))
		if lab.ticketsOf(tn) != 0 {
			t.Errorf("c1: COMMIT chưa gửi mà còn phiếu")
		}
	})
	t.Run("c2 ngữ cảnh hết hạn trong lúc COMMIT còn chạy trigger hoãn", func(t *testing.T) {
		lab.use(t)
		st, tn := lab.newPG("k2_u1_c2", nil), lab.tenant("c2")
		ctx, cancel := context.WithTimeout(ctxOf(tn), 2500*time.Millisecond)
		defer cancel()
		start := time.Now()
		err := st.WriteTicket(ctx, lab.unit(6, "EX-03", "k2:commit-sleep:6"))
		took := time.Since(start)
		wantClass(t, "c2", err, ErrCommitUnknown)
		if !errors.Is(err, context.DeadlineExceeded) || asPg(err) != nil || pgconn.SafeToRetry(err) {
			t.Errorf("c2: muốn errTimeout bọc DeadlineExceeded (không PgError, không SafeToRetry), được %v", chain(err))
		}
		if took < 2*time.Second || took > 5500*time.Millisecond {
			t.Errorf("c2: WriteTicket phải trả quanh hạn chót 2,5 s (cửa sổ COMMIT dài 6 s), mất %v", took)
		}
		// Đo: pgx gửi CancelRequest trước khi đóng kết nối (asyncClose), máy chủ đang ở pg_sleep của trigger hoãn nên bị cắt ngang và
		// huỷ giao dịch. Bên gọi chỉ thấy lỗi hết hạn; sự thật ở DB là CHƯA ghi. Đợi backend biến mất rồi mới đếm.
		lab.eventually(10*time.Second, "c2: backend của ca chưa kết thúc", func() bool {
			var n int
			if err := lab.admin.QueryRow(context.Background(), "SELECT count(*) FROM pg_stat_activity WHERE application_name = 'k2_u1_c2' AND state <> 'idle'").Scan(&n); err != nil {
				t.Fatal(err)
			}
			return n == 0
		})
		lab.record("c2", "ngữ cảnh 2,5 s; trigger hoãn pg_sleep(6) trong COMMIT; pgx gửi CancelRequest", err, err, truthOf(lab.ticketsOf(tn))+" (máy chủ bị cắt ngang và huỷ)")
	})
	// c3, c4: cùng hình dạng lỗi với c2 nhưng sự thật ngược lại. Một cầu TCP đứng giữa bên gọi và PG: nó chuyển tiếp COMMIT cho máy
	// chủ (máy chủ commit XONG) rồi giữ hoặc cắt phản hồi. Đây là cửa sổ "ghi rồi mà báo lỗi" thật, không giả bằng mã.
	for _, c := range []struct {
		id, how string
		mode    proxyMode
		timeout time.Duration
	}{
		{"c3", "cầu TCP giữ phản hồi COMMIT; ngữ cảnh 2 s hết hạn khi chờ", proxyHold, 2 * time.Second},
		{"c4", "cầu TCP cắt kết nối ngay khi phản hồi COMMIT về tới nó", proxyCut, 5 * time.Second},
	} {
		t.Run(c.id+" "+c.how, func(t *testing.T) {
			lab.use(t)
			tn := lab.tenant(c.id)
			proxy := newCommitProxy(t, lab.appDSN, c.mode)
			st := lab.newPG("k2_u1_"+c.id, map[string]string{"@host": proxy.addr()})
			ctx, cancel := context.WithTimeout(ctxOf(tn), c.timeout)
			defer cancel()
			err := st.WriteTicket(ctx, lab.unit(14, "EX-03", ""))
			wantClass(t, c.id, err, ErrCommitUnknown)
			if asPg(err) != nil {
				t.Errorf("%s: muốn lỗi mạng / hết hạn, không phải PgError: %v", c.id, chain(err))
			}
			// Đo: hết hạn khi chờ (c3) cho errTimeout bọc DeadlineExceeded, SafeToRetry = false. Mạng đứt khi chờ (c4) cho "conn closed"
			// mang SafeToRetry() = TRUE dù COMMIT đã tới máy chủ và đã commit (lỗi gốc bị che): bằng chứng SafeToRetry không đủ tin.
			if c.mode == proxyHold && (pgconn.SafeToRetry(err) || !errors.Is(err, context.DeadlineExceeded)) {
				t.Errorf("c3: muốn errTimeout bọc DeadlineExceeded không SafeToRetry, được %v (SafeToRetry=%v)", chain(err), pgconn.SafeToRetry(err))
			}
			if c.mode == proxyCut && (!pgconn.SafeToRetry(err) || !errors.Is(err, pgconn.ErrConnClosed)) {
				t.Errorf("c4: muốn \"conn closed\" mang SafeToRetry=true (hình dạng gây hiểu lầm đã đo), được %v (SafeToRetry=%v)", chain(err), pgconn.SafeToRetry(err))
			}
			if proxy.commits.Load() != 1 {
				t.Fatalf("%s: cầu phải thấy đúng 1 lệnh COMMIT đi qua, thấy %d", c.id, proxy.commits.Load())
			}
			var truth string
			lab.eventually(5*time.Second, c.id+": phiếu không xuất hiện dù COMMIT đã tới máy chủ", func() bool {
				if lab.ticketsOf(tn) == 1 {
					truth = "1 phiếu: ĐÃ GHI dù bên gọi nhận lỗi"
					return true
				}
				return false
			})
			lab.record(c.id, c.how, err, err, truth)
		})
	}

	// (d) lock_timeout / statement_timeout lúc chờ khoá chỉ mục duy nhất ------------------------------------------------------
	// blocker giữ khoá khoá nghiệp vụ bằng một giao dịch chưa commit; bên chờ ghi cùng (máy, chỉ số, ngày).
	blocker := func(t *testing.T, tn contracts.TenantID) pgx.Tx {
		t.Helper()
		tx := lab.begin(ctxOf(tn), lab.pg)
		if err := lab.pg.insertUnit(ctxOf(tn), tx, lab.unit(7, "EX-03", "")); err != nil {
			t.Fatal(err)
		}
		return tx
	}
	t.Run("d1 lock_timeout khi chờ khoá chỉ mục duy nhất", func(t *testing.T) {
		lab.use(t)
		st, tn := lab.newPG("k2_u1_d1", map[string]string{"lock_timeout": "300"}), lab.tenant("d1")
		blk := blocker(t, tn)
		defer blk.Rollback(ctx)
		start := time.Now()
		err := st.WriteTicket(ctxOf(tn), lab.unit(8, "EX-03", ""))
		took := time.Since(start)
		wantClass(t, "d1", err, ErrBeforeCommit)
		if pe := asPg(err); pe == nil || pe.Code != "55P03" {
			t.Errorf("d1: muốn PgError 55P03 (lock_not_available), được %v", chain(err))
		}
		if took < 250*time.Millisecond || took > 3*time.Second {
			t.Errorf("d1: lock_timeout 300 ms mà chờ %v", took)
		}
		lab.record("d1", "lock_timeout=300ms; INSERT phiếu chờ khoá do giao dịch khác giữ", err, err, "0 phiếu của bên chờ (bên giữ khoá chưa commit)")
		if err := blk.Rollback(ctx); err != nil {
			t.Fatal(err)
		}
		if n := lab.settle(tn, 100*time.Millisecond); n != 0 {
			t.Errorf("d1: còn %d phiếu", n)
		}
	})
	t.Run("d2 statement_timeout khi chờ khoá chỉ mục duy nhất", func(t *testing.T) {
		lab.use(t)
		st, tn := lab.newPG("k2_u1_d2", map[string]string{"statement_timeout": "300"}), lab.tenant("d2")
		blk := blocker(t, tn)
		defer blk.Rollback(ctx)
		err := st.WriteTicket(ctxOf(tn), lab.unit(9, "EX-03", ""))
		wantClass(t, "d2", err, ErrBeforeCommit)
		if pe := asPg(err); pe == nil || pe.Code != "57014" {
			t.Errorf("d2: muốn PgError 57014 (query_canceled), được %v", chain(err))
		}
		lab.record("d2", "statement_timeout=300ms; INSERT phiếu chờ khoá", err, err, "0 phiếu của bên chờ")
		if err := blk.Rollback(ctx); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("d3 ngữ cảnh hết hạn khi chờ khoá", func(t *testing.T) {
		lab.use(t)
		st, tn := lab.newPG("k2_u1_d3", nil), lab.tenant("d3")
		blk := blocker(t, tn)
		defer blk.Rollback(ctx)
		cctx, cancel := context.WithTimeout(ctxOf(tn), 400*time.Millisecond)
		defer cancel()
		err := st.WriteTicket(cctx, lab.unit(10, "EX-03", ""))
		wantClass(t, "d3", err, ErrBeforeCommit)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("d3: lỗi phải mở ra được context.DeadlineExceeded: %v", chain(err))
		}
		// nhả khoá: backend mồ côi của bên chờ (kết nối đã đóng) không được tự commit gì
		if err := blk.Rollback(ctx); err != nil {
			t.Fatal(err)
		}
		n := lab.settle(tn, 400*time.Millisecond)
		if n != 0 {
			t.Errorf("d3: backend mồ côi đã ghi %d phiếu sau khi khoá được nhả", n)
		}
		lab.record("d3", "ngữ cảnh 400ms; INSERT phiếu chờ khoá, pgx đóng kết nối", err, err, truthOf(n)+", kể cả 400 ms sau khi bên giữ khoá nhả")
	})
	t.Run("d4 bên giữ khoá commit khi bên chờ đang chờ", func(t *testing.T) {
		lab.use(t)
		st, tn := lab.newPG("k2_u1_d4", nil), lab.tenant("d4")
		blk := blocker(t, tn)
		defer blk.Rollback(ctx)
		done := make(chan error, 1)
		go func() { done <- st.WriteTicket(ctxOf(tn), lab.unit(11, "EX-03", "")) }()
		lab.waitBackend("k2_u1_d4", "wait_event_type = 'Lock'")
		if err := blk.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		var err error
		select {
		case err = <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("d4: WriteTicket không trả sau khi khoá được nhả")
		}
		wantClass(t, "d4", err, ErrUniqueViolation)
		if pe := asPg(err); pe == nil || pe.Code != "23505" {
			t.Errorf("d4: muốn PgError 23505, được %v", chain(err))
		}
		lab.record("d4", "bên giữ khoá COMMIT lúc INSERT của bên chờ đang bị chặn", err, err, truthOf(lab.ticketsOf(tn))+" (của bên giữ khoá)")
		if lab.ticketsOf(tn) != 1 {
			t.Errorf("d4: muốn đúng 1 phiếu (của bên giữ khoá), có %d", lab.ticketsOf(tn))
		}
	})

	// (e) PG trả lỗi cho chính lệnh COMMIT -------------------------------------------------------------------------------------
	for _, c := range []struct {
		id, marker, how string
		want            error
		code            string
	}{
		{"e1", "k2:commit-raise", "trigger hoãn báo lỗi P0001 ở COMMIT", ErrBeforeCommit, "P0001"},
		{"e2", "k2:commit-raise-23505", "trigger hoãn báo 23505 ở COMMIT", ErrUniqueViolation, "23505"},
	} {
		t.Run(c.id+" "+c.how, func(t *testing.T) {
			lab.use(t)
			st, tn := lab.newPG("k2_u1_"+c.id, nil), lab.tenant(c.id)
			err := st.WriteTicket(ctxOf(tn), lab.unit(12, "EX-03", c.marker))
			wantClass(t, c.id, err, c.want)
			if pe := asPg(err); pe == nil || pe.Code != c.code || pe.SeverityUnlocalized != "ERROR" {
				t.Errorf("%s: muốn PgError ERROR %s ở COMMIT, được %v", c.id, c.code, chain(err))
			}
			lab.record(c.id, c.how, err, err, truthOf(lab.settle(tn, 200*time.Millisecond)))
			if lab.ticketsOf(tn) != 0 {
				t.Errorf("%s: PG đã trả lỗi cho COMMIT mà còn phiếu", c.id)
			}
		})
	}
	// f: statement_timeout KHÔNG áp cho công việc của COMMIT (PG tắt đồng hồ statement_timeout trước khi chạy trigger hoãn), nên một COMMIT
	// chậm không bao giờ bị máy chủ tự cắt bằng 57014; chỉ CancelRequest (pgx khi ngữ cảnh hết hạn, pg_cancel_backend) mới cắt được.
	t.Run("f statement_timeout không cắt công việc của COMMIT", func(t *testing.T) {
		lab.use(t)
		st, tn := lab.newPG("k2_u1_f", map[string]string{"statement_timeout": "500"}), lab.tenant("f")
		err := st.WriteTicket(ctxOf(tn), lab.unit(15, "EX-03", "k2:commit-sleep:2"))
		if err != nil {
			t.Errorf("f: statement_timeout=500ms không được cắt COMMIT đang chạy trigger hoãn 2 s; được lỗi %v", chain(err))
		}
		lab.record("f", "statement_timeout=500ms; trigger hoãn pg_sleep(2) trong COMMIT", err, err, truthOf(lab.ticketsOf(tn))+" (COMMIT chạy hết cửa sổ)")
		if lab.ticketsOf(tn) != 1 {
			t.Errorf("f: muốn 1 phiếu, có %d", lab.ticketsOf(tn))
		}
	})

	// (g) ngữ cảnh đã hết hạn trước cả khi mở giao dịch ------------------------------------------------------------------------
	t.Run("g ngữ cảnh đã hết hạn trước khi mở giao dịch", func(t *testing.T) {
		lab.use(t)
		st, tn := lab.newPG("k2_u1_g", nil), lab.tenant("g")
		cctx, cancel := context.WithCancel(ctxOf(tn))
		cancel()
		err := st.WriteTicket(cctx, lab.unit(13, "EX-03", ""))
		wantClass(t, "g", err, ErrBeforeCommit)
		lab.record("g", "ngữ cảnh đã huỷ trước WriteTicket", err, err, truthOf(lab.ticketsOf(tn)))
	})

	t.Logf("\n%s", renderU1(lab.rows))
}

// renderU1 in bảng đo để dán vào báo cáo.
func renderU1(rows []u1Row) string {
	var b strings.Builder
	b.WriteString("BẢNG ĐO U1 (cột: ca | cách dựng | pgx trả (chuỗi kiểu lỗi, trong ra ngoài) | SafeToRetry | phân loại của Store | sự thật ở DB)\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "%-4s | %s | %s | %s | %s | %s\n", r.id, r.how, r.raw, r.safe, r.class, r.truth)
	}
	return b.String()
}

// proxyMode là việc cầu làm với phản hồi của máy chủ sau khi lệnh COMMIT đã đi qua nó.
type proxyMode int

const (
	proxyHold proxyMode = iota // giữ phản hồi (bên gọi chỉ thấy im lặng cho tới khi hết hạn)
	proxyCut                   // nhận phản hồi rồi cắt kết nối với bên gọi mà không chuyển phản hồi
)

// commitProxy là cầu TCP một chiều tới PG, chỉ cho thí nghiệm U1. Nó chuyển tiếp mọi byte nguyên vẹn; riêng khi thấy lệnh COMMIT
// (thông điệp Query "commit" của pgx) đi qua một kết nối thì chuyển COMMIT cho máy chủ rồi giữ hoặc cắt phản hồi theo mode. Máy chủ
// vẫn commit xong, nên bên gọi ở trạng thái đúng như khi mạng đứt giữa COMMIT và câu trả lời: đã ghi nhưng không biết.
type commitProxy struct {
	ln       net.Listener
	upstream string
	mode     proxyMode
	commits  atomic.Int32
	wg       sync.WaitGroup
	logf     func(format string, args ...any) // chỉ để gỡ lỗi thí nghiệm; nil = im lặng
}

var terminateMsg = []byte("X\x00\x00\x00\x04") // thông điệp Terminate của giao thức PG

var commitQuery = []byte("Q\x00\x00\x00\x0bcommit\x00") // 'Q', độ dài 11, "commit", NUL

func newCommitProxy(t *testing.T, dsn string, mode proxyMode) *commitProxy {
	t.Helper()
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p := &commitProxy{ln: ln, upstream: u.Host, mode: mode}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			p.wg.Add(1)
			go func() { defer p.wg.Done(); p.pipe(c) }()
		}
	}()
	t.Cleanup(func() { ln.Close(); p.wg.Wait() })
	return p
}

func (p *commitProxy) addr() string { return p.ln.Addr().String() }

func (p *commitProxy) pipe(client net.Conn) {
	server, err := net.Dial("tcp", p.upstream)
	if err != nil {
		client.Close()
		return
	}
	var armed atomic.Bool // đã thấy COMMIT trên kết nối này
	done := make(chan struct{})
	go func() { // máy chủ → bên gọi
		defer client.Close()
		buf := make([]byte, 32<<10)
		for {
			n, err := server.Read(buf)
			if p.logf != nil {
				p.logf("máy chủ → bên gọi: %d byte, loại %q, armed=%v, lỗi=%v", n, buf[:min(n, 1)], armed.Load(), err)
			}
			if n > 0 {
				if armed.Load() {
					if p.mode == proxyCut {
						if tc, ok := client.(*net.TCPConn); ok {
							_ = tc.SetLinger(0) // đóng bằng RST, như mạng đứt
						}
						return
					}
					<-done // proxyHold: không bao giờ chuyển phản hồi, chờ bên gọi bỏ đi
					return
				}
				if _, werr := client.Write(buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()
	buf := make([]byte, 32<<10) // bên gọi → máy chủ
	for {
		n, err := client.Read(buf)
		if p.logf != nil {
			p.logf("bên gọi → máy chủ: %d byte, loại %q, lỗi=%v", n, buf[:min(n, 1)], err)
		}
		if n > 0 {
			if bytes.Contains(buf[:n], commitQuery) {
				armed.Store(true)
				p.commits.Add(1)
			}
			_, werr := server.Write(buf[:n])
			if werr != nil || (armed.Load() && bytes.Contains(buf[:n], terminateMsg)) {
				break // bên gọi bỏ đi (gửi Terminate sau khi hết hạn): đóng hai đầu ngay, khỏi để pgx chờ EOF tới 15 s
			}
		}
		if err != nil {
			break
		}
	}
	close(done)
	server.Close()
	client.Close()
}
