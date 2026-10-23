package commands

import (
	"context"
	"sync"
	"testing"
	"time"
)

// auditCtxStore ghi lại ngữ cảnh mà pipeline đưa cho WriteAudit (có huỷ chưa, có hạn chót không) rồi chuyển tiếp cho kho thật.
type auditCtxStore struct {
	Store
	mu          sync.Mutex
	calls       int
	errAtWrite  error
	deadline    time.Time
	hasDeadline bool
}

func (s *auditCtxStore) WriteAudit(ctx context.Context, a Audit) error {
	s.mu.Lock()
	s.calls++
	s.errAtWrite = ctx.Err()
	s.deadline, s.hasDeadline = ctx.Deadline()
	s.mu.Unlock()
	return s.Store.WriteAudit(ctx, a)
}

// Soát K3 (K3S-01) và soát lại M4 (M4R-03): dòng audit riêng (REFUSED / REPLAYED / VERIFY_FAILED) ghi bằng ngữ cảnh KHÔNG huỷ theo bên gọi
// nhưng CÓ hạn riêng trong (0, auditTimeout] — bỏ hạn thì kho không trả lời sẽ treo goroutine của handler vô hạn (WithoutCancel gỡ cả hạn
// chót của bên gọi và của tool). Bên gọi huỷ trước khi chạy: lượt bị từ chối ở C4 vẫn ghi đúng một dòng REFUSED.
func TestAuditRiengKhongHuyTheoBenGoiVaCoHanRieng(t *testing.T) {
	var spy *auditCtxStore
	w := newWorld(t, func(d *Deps) { spy = &auditCtxStore{Store: d.Store}; d.Store = spy })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	o := w.pipe.Run(ctx, user("p-huy", "driver", "EX-03"), req(43, "EX-03", day1))
	if o.Result.ReasonCode != ReasonPermissionDenied {
		t.Fatalf("muốn PERMISSION_DENIED, được %+v", o.Result)
	}
	spy.mu.Lock()
	defer spy.mu.Unlock()
	if spy.calls != 1 {
		t.Fatalf("muốn đúng 1 lần ghi audit, được %d", spy.calls)
	}
	if spy.errAtWrite != nil {
		t.Errorf("ngữ cảnh ghi audit đã huỷ theo bên gọi: %v", spy.errAtWrite)
	}
	if !spy.hasDeadline {
		t.Fatalf("ngữ cảnh ghi audit không có hạn chót: kho không trả lời sẽ treo handler vô hạn")
	}
	if left := spy.deadline.Sub(start); left <= 0 || left > auditTimeout+time.Second {
		t.Errorf("hạn của lần ghi audit cách lúc chạy %v, muốn trong (0, %v]", left, auditTimeout)
	}
}
