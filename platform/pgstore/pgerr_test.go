package pgstore

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Lỗi 5b của soát độc lập 23/10: mọi lỗi PG từng mang nhãn hạ tầng nên bị thử lại 3 lần rồi trả 503 "thử lại sau", kể cả vi phạm
// ràng buộc, RLS chặn ghi hay thiếu quyền — những lỗi mà thử lại y nguyên không bao giờ thành công. Chỉ lỗi tạm thời được thử lại.
func TestPhanLoaiLoiPG(t *testing.T) {
	for _, c := range []struct {
		name      string
		err       error
		retryable bool
		status    int
	}{
		{"mất kết nối (không phải lỗi của PG)", errors.New("read tcp: connection reset by peer"), true, http.StatusServiceUnavailable},
		{"ngữ cảnh hết hạn", context.DeadlineExceeded, true, http.StatusServiceUnavailable},
		{"08006 kết nối hỏng", &pgconn.PgError{Code: "08006"}, true, http.StatusServiceUnavailable},
		{"40001 tuần tự hoá", &pgconn.PgError{Code: "40001"}, true, http.StatusServiceUnavailable},
		{"40P01 deadlock", &pgconn.PgError{Code: "40P01"}, true, http.StatusServiceUnavailable},
		{"53300 quá nhiều kết nối", &pgconn.PgError{Code: "53300"}, true, http.StatusServiceUnavailable},
		{"55P03 không lấy được khoá", &pgconn.PgError{Code: "55P03"}, true, http.StatusServiceUnavailable},
		{"57014 huỷ do hết giờ", &pgconn.PgError{Code: "57014"}, true, http.StatusServiceUnavailable},
		{"57P01 máy chủ đang tắt", &pgconn.PgError{Code: "57P01"}, true, http.StatusServiceUnavailable},
		// lớp 25 chỉ ba mã tạm thời (soát độc lập 5b-2): ghi vào máy chủ chỉ đọc lúc chuyển máy chủ chính, phiên / giao dịch bị cắt do hết giờ
		{"25006 giao dịch chỉ đọc (lúc chuyển máy chủ chính)", &pgconn.PgError{Code: "25006"}, true, http.StatusServiceUnavailable},
		{"25P03 phiên rỗi trong giao dịch quá hạn", &pgconn.PgError{Code: "25P03"}, true, http.StatusServiceUnavailable},
		{"25P04 giao dịch quá hạn (PG 17)", &pgconn.PgError{Code: "25P04"}, true, http.StatusServiceUnavailable},
		{"23505 trùng khoá duy nhất", &pgconn.PgError{Code: "23505"}, false, http.StatusConflict},
		{"23503 khoá ngoại", &pgconn.PgError{Code: "23503"}, false, http.StatusConflict},
		{"22P02 dữ liệu sai dạng", &pgconn.PgError{Code: "22P02"}, false, http.StatusConflict},
		{"42501 thiếu quyền / RLS chặn ghi", &pgconn.PgError{Code: "42501"}, false, http.StatusInternalServerError},
		{"42P01 bảng không có", &pgconn.PgError{Code: "42P01"}, false, http.StatusInternalServerError},
		{"XX000 lỗi nội bộ", &pgconn.PgError{Code: "XX000"}, false, http.StatusInternalServerError},
		// lỗi tất định không mang SQLSTATE (soát độc lập 5b-1): thử lại y nguyên vẫn hỏng
		{"không có dòng", pgx.ErrNoRows, false, http.StatusInternalServerError},
		{"không có dòng, bọc", fmt.Errorf("đọc phiếu: %w", pgx.ErrNoRows), false, http.StatusInternalServerError},
		{"commit thành rollback (fn nuốt lỗi)", pgx.ErrTxCommitRollback, false, http.StatusInternalServerError},
		{"quét sai kiểu cột", pgx.ScanArgError{ColumnIndex: 0, Err: errors.New("cannot scan text into *int")}, false, http.StatusInternalServerError},
	} {
		got := classifyPGError(c.err)
		if contracts.Retryable(got) != c.retryable || contracts.HTTPStatus(got) != c.status {
			t.Errorf("%s: nhãn %q, thử lại %v, HTTP %d; muốn thử lại %v, HTTP %d", c.name, contracts.FaultOf(got),
				contracts.Retryable(got), contracts.HTTPStatus(got), c.retryable, c.status)
		}
		if !errors.Is(got, c.err) {
			t.Errorf("%s: mất lỗi gốc", c.name)
		}
	}
	// lỗi đã có nhãn (fn trong giao dịch tự gắn, vd khoá ngoại của usage là lỗi của bên gọi) giữ nguyên nhãn
	in := contracts.Errorf(contracts.FaultInput, "usage khi chưa có bản ghi quyết định")
	if got := classifyPGError(in); contracts.FaultOf(got) != contracts.FaultInput {
		t.Errorf("lỗi đã có nhãn bị đổi nhãn thành %q", contracts.FaultOf(got))
	}
	if classifyPGError(nil) != nil {
		t.Error("lỗi nil phải trả nil")
	}
}

// Cùng phép phân loại, đi qua inTenant với lỗi THẬT của PostgreSQL (không ghi gì: chỉ SELECT và đặt thời hạn trong giao dịch).
func TestLoiPGThatQuaInTenant(t *testing.T) {
	admin, app := EnvDSN()
	if admin == "" || app == "" {
		t.Skip("chưa có EAIS_PG_ADMIN_DSN / EAIS_PG_DSN — xem deploy/docker-compose.yml")
	}
	ctx := context.Background()
	if err := Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	s, err := Open(ctx, app)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(s.Close)
	for _, c := range []struct {
		name, sql string
		retryable bool
		status    int
	}{
		{"chia cho 0 (22012)", "SELECT 1/0", false, http.StatusConflict},
		{"bảng không có (42P01)", "SELECT * FROM bang_khong_ton_tai_5b", false, http.StatusInternalServerError},
		{"thiếu quyền đọc bảng hệ thống (42501)", "SELECT rolpassword FROM pg_authid LIMIT 1", false, http.StatusInternalServerError},
		{"hết giờ câu lệnh (57014)", "SET LOCAL statement_timeout = '20ms'; SELECT pg_sleep(1)", true, http.StatusServiceUnavailable},
	} {
		err := s.inTenant(ctx, "abc-infra", func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, c.sql)
			return err
		})
		if err == nil {
			t.Fatalf("%s: không lỗi", c.name)
		}
		if contracts.Retryable(err) != c.retryable || contracts.HTTPStatus(err) != c.status {
			t.Errorf("%s: nhãn %q, thử lại %v, HTTP %d; muốn thử lại %v, HTTP %d (lỗi: %v)", c.name, contracts.FaultOf(err),
				contracts.Retryable(err), contracts.HTTPStatus(err), c.retryable, c.status, err)
		}
	}
	// lỗi tất định do pgx sinh ở phía khách (không SQLSTATE): không có dòng, quét sai kiểu
	for _, c := range []struct {
		name string
		scan func(tx pgx.Tx) error
	}{
		{"không có dòng", func(tx pgx.Tx) error { var n int; return tx.QueryRow(ctx, "SELECT 1 WHERE false").Scan(&n) }},
		{"quét chữ vào số", func(tx pgx.Tx) error { var n int; return tx.QueryRow(ctx, "SELECT 'abc'::text").Scan(&n) }},
	} {
		err := s.inTenant(ctx, "abc-infra", c.scan)
		if err == nil || contracts.Retryable(err) || contracts.HTTPStatus(err) != http.StatusInternalServerError {
			t.Errorf("%s: nhãn %q, thử lại %v, HTTP %d; muốn lỗi của mã, không thử lại, HTTP 500 (lỗi: %v)", c.name,
				contracts.FaultOf(err), contracts.Retryable(err), contracts.HTTPStatus(err), err)
		}
	}
}
