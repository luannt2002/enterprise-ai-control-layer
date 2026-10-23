package pgstore

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
)

// KnowledgeSQL là chiến lược nguồn tri thức "postgres" (rag.SourceKinds): đọc tài liệu từ một database Postgres CÓ SẴN bằng câu truy
// vấn của cấu hình; chuỗi kết nối lấy từ biến môi trường có TÊN ở dsn_env. Truy vấn chạy trong giao dịch READ ONLY. Cột theo TÊN:
// id, title, body bắt buộc (text); acl, scope (text[]), version (int), status, effective_from, effective_to (text, ngày YYYY-MM-DD)
// tuỳ chọn — thiếu cột thì lấy giá trị của cấu hình; cột lạ là lỗi. Cần cột acl hoặc acl ở cấu hình: không ai mặc định đọc được.
//
// Chỉ dùng cho kho tài liệu mà tenant giao cho EAIS đọc. Database của phần mềm khách đang giữ quyền (chế độ A) phải đi qua API của nó:
// đọc thẳng là vòng qua quyền của app (CLAUDE.md §9).
//
// Rào (soát độc lập 23/10): giao thức mở rộng — một truy vấn chỉ một lệnh, kể cả khi chuỗi kết nối chọn giao thức đơn giản (khi đó
// "SELECT …; COMMIT; DELETE …" chạy trọn và vượt READ ONLY); vai superuser bị từ chối (READ ONLY không chặn COPY … TO PROGRAM);
// statement_timeout 60 s; lỗi kết nối không kèm chi tiết của driver (có thể in lại mật khẩu trong chuỗi kết nối). Vai nên chỉ có SELECT.
func KnowledgeSQL(spec rag.SourceSpec, _ string) (rag.Source, error) {
	if spec.DSNEnv == "" || strings.TrimSpace(spec.Query) == "" {
		return nil, fmt.Errorf("kind postgres cần dsn_env (tên biến môi trường) và query")
	}
	if len(spec.TieuDe) > 0 {
		return nil, fmt.Errorf("kind postgres chưa hỗ trợ tieu_de")
	}
	return rag.SourceFunc(func(ctx context.Context, tenant string) ([]rag.Document, error) {
		dsn := os.Getenv(spec.DSNEnv)
		if dsn == "" {
			return nil, fmt.Errorf("biến môi trường %s trống", spec.DSNEnv)
		}
		cfg, err := pgx.ParseConfig(dsn)
		if err != nil {
			return nil, fmt.Errorf("biến %s: chuỗi kết nối không đọc được (chi tiết bị che: có thể chứa mật khẩu)", spec.DSNEnv)
		}
		cfg.DefaultQueryExecMode = pgx.QueryExecModeDescribeExec
		cfg.RuntimeParams["statement_timeout"] = "60000"
		conn, err := pgx.ConnectConfig(ctx, cfg)
		if err != nil {
			return nil, contracts.Label(contracts.Infra("postgres"), fmt.Errorf("không kết nối được database của nguồn (biến %s): %s", spec.DSNEnv, pgCode(err)))
		}
		defer conn.Close(ctx)
		var super, bypass bool
		if err := conn.QueryRow(ctx, "SELECT rolsuper, rolbypassrls FROM pg_catalog.pg_roles WHERE rolname = current_user").Scan(&super, &bypass); err != nil {
			return nil, contracts.Label(contracts.Infra("postgres"), err)
		}
		if super || bypass {
			return nil, contracts.Errorf(contracts.FaultPolicy, "biến "+spec.DSNEnv+": vai superuser / BYPASSRLS bị từ chối — dùng vai chỉ có quyền SELECT trên bảng tài liệu")
		}
		var out []rag.Document
		err = pgx.BeginTxFunc(ctx, conn, pgx.TxOptions{AccessMode: pgx.ReadOnly}, func(tx pgx.Tx) error {
			rows, err := tx.Query(ctx, spec.Query)
			if err != nil {
				return err
			}
			defer rows.Close()
			fields := rows.FieldDescriptions()
			if len(fields) == 0 { // lỗi của server (vd ghi trong giao dịch READ ONLY) chỉ lộ ra khi đọc: nêu đúng nó, không báo "thiếu cột"
				rows.Close()
				if err := rows.Err(); err != nil {
					return err
				}
			}
			cols := make([]string, len(fields))
			for i, f := range fields {
				cols[i] = f.Name
			}
			hasACL := slices.Contains(cols, "acl")
			if !hasACL && len(spec.ACL) == 0 {
				return fmt.Errorf("không có cột acl và cấu hình không khai acl")
			}
			if _, _, err := sqlDest(cols); err != nil { // cột lạ là lỗi cả khi truy vấn không ra hàng nào
				return err
			}
			for _, need := range []string{"id", "title", "body"} {
				if !slices.Contains(cols, need) {
					return fmt.Errorf("truy vấn thiếu cột %s (có: %s)", need, strings.Join(cols, ", "))
				}
			}
			for rows.Next() {
				d := rag.Document{Tenant: tenant, ACL: spec.ACL, Version: spec.Version, Status: spec.Status,
					EffectiveFrom: spec.EffectiveFrom, EffectiveTo: spec.EffectiveTo, Scope: spec.Scope}
				dest, set, err := sqlDest(cols)
				if err != nil {
					return err
				}
				if err := rows.Scan(dest...); err != nil {
					return err
				}
				set(&d)
				if len(d.ACL) == 0 {
					return fmt.Errorf("tài liệu %s: quyền đọc rỗng (cột acl NULL / rỗng và cấu hình không khai acl)", d.ID)
				}
				out = append(out, d)
			}
			return rows.Err()
		})
		if err != nil {
			return nil, contracts.Label(contracts.Infra("postgres"), err)
		}
		return out, nil
	}), nil
}

// pgCode chỉ giữ mã + thông điệp của lỗi do SERVER trả (không chứa chuỗi kết nối); lỗi khác của driver bị che.
func pgCode(err error) string {
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe.Code + " " + pe.Message
	}
	return "lỗi kết nối (chi tiết bị che: có thể chứa chuỗi kết nối)"
}

// sqlColumns: tên cột → hàm tạo đích quét + hàm chép vào tài liệu (cột NULL giữ giá trị của cấu hình).
var sqlColumns = map[string]func() (any, func(*rag.Document)){
	"id":             textCol(func(d *rag.Document, v string) { d.ID = v }),
	"title":          textCol(func(d *rag.Document, v string) { d.Title = v }),
	"body":           textCol(func(d *rag.Document, v string) { d.Body = v }),
	"status":         textCol(func(d *rag.Document, v string) { d.Status = v }),
	"effective_from": textCol(func(d *rag.Document, v string) { d.EffectiveFrom = v }),
	"effective_to":   textCol(func(d *rag.Document, v string) { d.EffectiveTo = v }),
	"acl":            listCol(func(d *rag.Document, v []string) { d.ACL = v }),
	"scope":          listCol(func(d *rag.Document, v []string) { d.Scope = v }),
	"version": func() (any, func(*rag.Document)) {
		var v *int
		return &v, func(d *rag.Document) {
			if v != nil {
				d.Version = *v
			}
		}
	},
}

func textCol(put func(*rag.Document, string)) func() (any, func(*rag.Document)) {
	return func() (any, func(*rag.Document)) {
		var v *string
		return &v, func(d *rag.Document) {
			if v != nil {
				put(d, *v)
			}
		}
	}
}

func listCol(put func(*rag.Document, []string)) func() (any, func(*rag.Document)) {
	return func() (any, func(*rag.Document)) {
		var v []string
		return &v, func(d *rag.Document) {
			if v != nil {
				put(d, v)
			}
		}
	}
}

func sqlDest(cols []string) ([]any, func(*rag.Document), error) {
	dest := make([]any, len(cols))
	sets := make([]func(*rag.Document), len(cols))
	for i, c := range cols {
		mk, ok := sqlColumns[c]
		if !ok {
			return nil, nil, fmt.Errorf("cột %q lạ (nhận: %s)", c, strings.Join(slices.Sorted(maps.Keys(sqlColumns)), ", "))
		}
		dest[i], sets[i] = mk()
	}
	return dest, func(d *rag.Document) {
		for _, s := range sets {
			s(d)
		}
	}, nil
}
