package commands

import (
	"fmt"
	"sync/atomic"
)

// Núm F2 (HOP-DONG §10, điểm chèn lỗi in_tx_audit_insert) cho kho PG: làm INSERT audit EXECUTED giữa đơn vị ghi phiếu hỏng n
// lần, SAU khi hai INSERT đầu (phiếu, sổ lệnh) đã chạy trong cùng giao dịch. Lớp bọc probe ở ngoài Store không với tới giữa giao
// dịch, nên F2 cần một móc trong chính kho. Móc chỉ gắn được từ tệp _test.go này: đường chạy thường để trường nil.
func (s *PGStore) failAuditInsertTimes(n int) {
	var left atomic.Int32
	left.Store(int32(n))
	s.beforeAuditInsert = func() error {
		if left.Add(-1) >= 0 {
			return fmt.Errorf("F2: INSERT audit bị từ chối giữa giao dịch")
		}
		return nil
	}
}
