package commands

import (
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// Bảng mã lý do (HOP-DONG §2.4) và mã HTTP suy từ nhãn fault (§2.6.2): mỗi dòng của bảng là một ca.
func TestBangMaLyDo(t *testing.T) {
	cases := []struct {
		code   ReasonCode
		status Status
		fault  string
		retry  Retry
		http   int
	}{
		{ReasonInvalidRequest, StatusFailed, "input", RetryFixRequest, http.StatusBadRequest},
		{ReasonIdempotencyKeyReused, StatusFailed, "input", RetryFixRequest, http.StatusBadRequest},
		{ReasonPermissionDenied, StatusFailed, "policy", RetryNone, http.StatusForbidden},
		{ReasonInsufficientEvidence, StatusFailed, "data", RetryLater, http.StatusConflict},
		{ReasonHumanReviewRequired, StatusFailed, "data", RetryLater, http.StatusConflict},
		{ReasonNoDiscrepancy, StatusFailed, "data", RetryNone, http.StatusConflict},
		{ReasonDuplicateOpenTicket, StatusFailed, "data", RetryNone, http.StatusConflict},
		{ReasonStoreUnavailable, StatusFailed, "infra:postgres", RetrySameKey, http.StatusServiceUnavailable},
		{ReasonCommitUnknown, StatusUnknown, "infra:postgres", RetrySameKey, http.StatusServiceUnavailable},
		{ReasonReadbackUnavailable, StatusUnknown, "infra:postgres", RetrySameKey, http.StatusServiceUnavailable},
		{ReasonVerifyMismatch, StatusUnknown, "code", RetryNone, http.StatusInternalServerError},
		{ReasonInternalError, StatusFailed, "code", RetryNone, http.StatusInternalServerError},
	}
	for _, c := range cases {
		r := resultFor(c.code)
		if r.Status != c.status || r.ReasonCode != c.code || r.Fault != c.fault || r.Retry != c.retry {
			t.Errorf("%s: được status=%s fault=%q retry=%s, muốn status=%s fault=%q retry=%s", c.code, r.Status, r.Fault, r.Retry, c.status, c.fault, c.retry)
		}
		if got := r.HTTPStatus(); got != c.http {
			t.Errorf("%s: mã HTTP %d, muốn %d", c.code, got, c.http)
		}
		if r.Detail == "" {
			t.Errorf("%s: thiếu detail (câu cố định theo mã lý do)", c.code)
		}
	}
	// INTERNAL_ERROR: FAILED nếu lỗi trước BEGIN, UNKNOWN nếu sau BEGIN mà không xác nhận rollback.
	if u := internalResult(StatusUnknown); u.Status != StatusUnknown || u.ReasonCode != ReasonInternalError || u.Fault != "code" || u.Detail == "" {
		t.Errorf("INTERNAL_ERROR sau BEGIN: %+v", u)
	}
	if internalResult(StatusFailed).Detail == internalResult(StatusUnknown).Detail {
		t.Errorf("FAILED nói chắc chắn chưa ghi, UNKNOWN nói chưa biết: hai câu không được trùng")
	}
}

// DONE luôn là 200 (kể cả phát lại); qua HTTP chỉ còn lớp lỗi (phần trước dấu hai chấm).
func TestDoneVaLopLoi(t *testing.T) {
	if got := (Result{Status: StatusDone}).HTTPStatus(); got != http.StatusOK {
		t.Errorf("DONE phải 200, được %d", got)
	}
	for fault, class := range map[string]string{"infra:postgres": "infra", "input": "input", "policy": "policy", "data": "data", "code": "code", "": ""} {
		if got := (Result{Fault: fault}).FaultClass(); got != class {
			t.Errorf("FaultClass(%q) = %q, muốn %q", fault, got, class)
		}
	}
}

// detail là câu cố định, viết cho bên gọi: không tên thành phần, trình điều khiển, đường dẫn hay SQL (§2.3).
func TestDetailKhongLoChiTietNoiBo(t *testing.T) {
	banned := []string{"postgres", "pgx", "sql", "infra", "driver", "panic", "/home", ".go", "tenant_id"}
	for code := range reasons {
		for _, r := range []Result{resultFor(code), internalResult(StatusUnknown)} {
			low := strings.ToLower(r.Detail)
			for _, b := range banned {
				if strings.Contains(low, b) {
					t.Errorf("%s: detail %q lộ %q", code, r.Detail, b)
				}
			}
		}
	}
}

// A11 (phần mã): yêu cầu chỉ có đúng bốn trường; không tenant, người gọi, vai, phạm vi, số, trạng thái, mã phiếu, chữ tự do.
// Cổng kiến trúc TestRequestTypesHaveNoTenantField quét cùng cây này và phải xanh (chạy ở gốc repo).
func TestRequestChiCoBonTruong(t *testing.T) {
	rt := reflect.TypeOf(Request{})
	var tags []string
	for i := range rt.NumField() {
		tags = append(tags, strings.Split(rt.Field(i).Tag.Get("json"), ",")[0])
	}
	slices.Sort(tags)
	if want := []string{"command_id", "day", "metric", "subject_key"}; !slices.Equal(tags, want) {
		t.Fatalf("trường của Request = %v, muốn đúng %v", tags, want)
	}
}
