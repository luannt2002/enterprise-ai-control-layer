package contracts

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestLabelGiuNhanGanNhat(t *testing.T) {
	inner := Label(ThirdParty("embedder", KindRateLimit), errors.New("429"))
	// Tầng ngoài gắn nhãn chung chung thì KHÔNG được đè nhãn cụ thể của tầng trong.
	outer := Label(FaultCode, fmt.Errorf("gather: %w", inner))
	if got := FaultOf(outer); got != "third_party:embedder:rate_limit" {
		t.Fatalf("FaultOf = %q, muốn nhãn của tầng trong", got)
	}
}

func TestLabelNilTraNil(t *testing.T) {
	if Label(FaultCode, nil) != nil {
		t.Fatal("Label(nil) phải trả nil, không được biến thành lỗi")
	}
}

func TestRetryableVaHTTPStatus(t *testing.T) {
	cases := []struct {
		name      string
		err       error
		retryable bool
		status    int
	}{
		{"input", Errorf(FaultInput, "thiếu message_id"), false, http.StatusBadRequest},
		{"policy", Errorf(FaultPolicy, "không có quyền"), false, http.StatusForbidden},
		{"data", Errorf(FaultData, "xung đột nguồn"), false, http.StatusConflict},
		{"code", Errorf(FaultCode, "bug"), false, http.StatusInternalServerError},
		{"chưa nhãn", errors.New("lạ"), false, http.StatusInternalServerError},
		{"infra", Errorf(Infra("postgres"), "conn refused"), true, http.StatusServiceUnavailable},
		{"nhà cung cấp tạm", Errorf(ThirdParty("model", KindUnavailable), "503"), true, http.StatusServiceUnavailable},
		{"nhà cung cấp từ chối", Errorf(ThirdParty("model", "http_401"), "401"), false, http.StatusBadGateway},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Retryable(c.err); got != c.retryable {
				t.Errorf("Retryable = %v, muốn %v", got, c.retryable)
			}
			if got := HTTPStatus(c.err); got != c.status {
				t.Errorf("HTTPStatus = %d, muốn %d", got, c.status)
			}
		})
	}
}

func TestRouteKindDuMuoiHaiVaNeedsModel(t *testing.T) {
	if len(AllRoutes) != 12 {
		t.Fatalf("có %d route, quy ước là 12", len(AllRoutes))
	}
	seen := map[RouteKind]bool{}
	for _, r := range AllRoutes {
		if seen[r] {
			t.Fatalf("route trùng: %s", r)
		}
		seen[r] = true
		if !r.Valid() {
			t.Fatalf("%s không Valid", r)
		}
	}
	if RouteKind("SOMETHING").Valid() {
		t.Fatal("route lạ không được Valid")
	}
	// Route trả lời bằng mẫu câu của tenant thì KHÔNG gọi model.
	for _, r := range []RouteKind{RouteSQLQuery, RouteCalculation, RouteClarification, RouteReject, RouteHumanEscalation} {
		if r.NeedsModel() {
			t.Errorf("%s không được cần model", r)
		}
	}
	for _, r := range []RouteKind{RouteRAG, RouteHybrid, RouteDirectLLM} {
		if !r.NeedsModel() {
			t.Errorf("%s phải cần model", r)
		}
	}
}
