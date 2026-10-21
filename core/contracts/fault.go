package contracts

import (
	"errors"
	"net/http"
	"strings"
)

// Nhãn "lỗi do ai" (docs/CONVENTIONS.md §4.3).
//
// Một lỗi ra khỏi EAIS mà không có nhãn thì người trực không biết nên sửa mã, hỏi nhà
// cung cấp, chờ hạ tầng hay báo bên gọi gửi lại. Mọi lỗi nên mang đúng một nhãn.
const (
	FaultInput  = "input"  // bên gọi gửi sai — sửa request, đừng thử lại y nguyên
	FaultPolicy = "policy" // bị chính sách chặn: quyền, phạm vi
	FaultData   = "data"   // dữ liệu tenant thiếu/hỏng/xung đột không tự giải được
	FaultCode   = "code"   // lỗi của chính EAIS — thử lại không giúp gì
)

// Loại lỗi của nhà cung cấp/hạ tầng — nửa sau của nhãn "third_party:<provider>:<kind>"
// hoặc "infra:<component>". Không phải enum đóng: mã HTTP không có nghĩa riêng thì
// chính nó là loại ("http_401"), vì đoán nghĩa cho từng mã 4xx là tự bịa một vốn từ.
const (
	KindRateLimit   = "rate_limit"   // hết hạn mức: chờ rồi thử lại
	KindUnavailable = "unavailable"  // 5xx: sập tạm
	KindUnreachable = "unreachable"  // không tới được: mạng/DNS
	KindTimeout     = "timeout"      // quá hạn chờ
	KindBadResponse = "bad_response" // tới được, nhưng nội dung không dùng được
)

const (
	thirdPartyPrefix = "third_party:"
	infraPrefix      = "infra:"
)

// ThirdParty dựng nhãn lỗi của một nhà cung cấp ngoài.
func ThirdParty(provider, kind string) string { return thirdPartyPrefix + provider + ":" + kind }

// Infra dựng nhãn lỗi của một thành phần hạ tầng (postgres, parser…).
func Infra(component string) string { return infraPrefix + component }

// transient là các loại mà THỬ LẠI SAU còn có ích.
var transient = map[string]bool{
	KindRateLimit: true, KindUnavailable: true, KindUnreachable: true,
	KindTimeout: true, KindBadResponse: true,
}

// Fault gắn nhãn cho một lỗi mà vẫn giữ nguyên lỗi gốc để ghi log.
type Fault struct {
	Label string
	Err   error
}

func (f *Fault) Error() string { return f.Err.Error() }
func (f *Fault) Unwrap() error { return f.Err }

// Label gắn nhãn cho err. Lỗi nil trả nil; lỗi đã có nhãn thì giữ nhãn gần nhất —
// tầng trong biết rõ hơn tầng ngoài ai là người gây lỗi.
func Label(label string, err error) error {
	if err == nil {
		return nil
	}
	if FaultOf(err) != "" {
		return err
	}
	return &Fault{Label: label, Err: err}
}

// Errorf tạo lỗi mới có nhãn.
func Errorf(label, msg string) error { return &Fault{Label: label, Err: errors.New(msg)} }

// FaultOf trả nhãn gần nhất trong chuỗi lỗi; "" khi chưa ai gắn.
func FaultOf(err error) string {
	var f *Fault
	if errors.As(err, &f) {
		return f.Label
	}
	return ""
}

// Retryable trả lời đúng câu hỏi mà bên gọi cần: thử lại sau có ích không?
func Retryable(err error) bool {
	label := FaultOf(err)
	if strings.HasPrefix(label, infraPrefix) {
		return true
	}
	rest, ok := strings.CutPrefix(label, thirdPartyPrefix)
	if !ok {
		return false
	}
	_, kind, ok := strings.Cut(rest, ":")
	return ok && transient[kind]
}

// HTTPStatus quy nhãn lỗi thành mã HTTP cho biên HTTP. Lỗi chưa có nhãn là lỗi của
// mã (500): chưa ai phân loại được nghĩa là chưa ai lường trước được nó.
func HTTPStatus(err error) int {
	label := FaultOf(err)
	switch {
	case label == FaultInput:
		return http.StatusBadRequest
	case label == FaultPolicy:
		return http.StatusForbidden
	case label == FaultData:
		return http.StatusConflict
	case Retryable(err):
		return http.StatusServiceUnavailable
	case strings.HasPrefix(label, thirdPartyPrefix):
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}
