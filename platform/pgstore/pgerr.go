package pgstore

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// classifyPGError gắn nhãn "lỗi do ai" cho lỗi của một giao dịch PostgreSQL, để bên gọi biết thử lại có ích không
// (contracts.Retryable, contracts.HTTPStatus):
//   - tạm thời ⇒ nhãn hạ tầng (thử lại, 503): lỗi không đến từ máy chủ PG (mất kết nối, I/O, ngữ cảnh hết hạn) và các lớp SQLSTATE
//     08 (kết nối), 40 (tuần tự hoá, deadlock), 53 (thiếu tài nguyên), 57 (huỷ do hết giờ, máy chủ đang tắt), 58 (lỗi hệ thống),
//     cộng 55P03 (không lấy được khoá);
//   - lớp 22 (dữ liệu sai dạng) và 23 (vi phạm ràng buộc) ⇒ lỗi dữ liệu (409): thử lại y nguyên vẫn hỏng;
//   - mọi mã khác, gồm 42501 (thiếu quyền, RLS chặn ghi) ⇒ lỗi của mã (500).
//
// Lỗi đã có nhãn (fn trong giao dịch tự gắn) giữ nguyên vì contracts.Label giữ nhãn gần nhất; lỗi nil trả nil. Giới hạn đã biết: lỗi không đến từ máy chủ nhưng do mã (ví dụ quét sai
// kiểu cột) vẫn mang nhãn hạ tầng như trước.
func classifyPGError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return contracts.Label(contracts.Infra("postgres"), err)
	}
	class := pgErr.Code
	if len(class) >= 2 {
		class = class[:2]
	}
	switch {
	case class == "08" || class == "40" || class == "53" || class == "57" || class == "58" || pgErr.Code == "55P03":
		return contracts.Label(contracts.Infra("postgres"), err)
	case class == "22" || class == "23":
		return contracts.Label(contracts.FaultData, err)
	default:
		return contracts.Label(contracts.FaultCode, err)
	}
}
