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
//     cộng các mã tạm thời lẻ ở transientCodes;
//   - lớp 22 (dữ liệu sai dạng) và 23 (vi phạm ràng buộc) ⇒ lỗi dữ liệu (409): thử lại y nguyên vẫn hỏng;
//   - mọi mã khác, gồm 42501 (thiếu quyền, RLS chặn ghi) ⇒ lỗi của mã (500).
//
// Lỗi đã có nhãn (fn trong giao dịch tự gắn) giữ nguyên vì contracts.Label giữ nhãn gần nhất; lỗi nil trả nil. Giới hạn đã biết: lỗi không đến từ máy chủ nhưng do mã (ví dụ quét sai
// kiểu cột) vẫn mang nhãn hạ tầng như trước.
// transientCodes là các mã tạm thời nằm NGOÀI các lớp hạ tầng: 55P03 không lấy được khoá; 25006 ghi vào máy chủ chỉ đọc (gặp lúc chuyển
// sang máy chủ chính mới); 25P03 phiên rỗi trong giao dịch quá hạn và 25P04 giao dịch quá hạn (PG 17) — cùng bản chất với 57014 huỷ do
// hết giờ. Các mã khác của lớp 25 (giao dịch đã hỏng, lồng giao dịch…) là lỗi của mã.
var transientCodes = map[string]bool{"55P03": true, "25006": true, "25P03": true, "25P04": true}

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
	case class == "08" || class == "40" || class == "53" || class == "57" || class == "58" || transientCodes[pgErr.Code]:
		return contracts.Label(contracts.Infra("postgres"), err)
	case class == "22" || class == "23":
		return contracts.Label(contracts.FaultData, err)
	default:
		return contracts.Label(contracts.FaultCode, err)
	}
}
