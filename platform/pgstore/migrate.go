package pgstore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/db"
)

// migrateLockKey là khoá tư vấn (pg_advisory_lock) mà Migrate giữ suốt lúc chạy: "EAISMIGR" viết thành số.
const migrateLockKey int64 = 0x454149534d494752

// Migrate áp các migration chưa áp, theo thứ tự tên tệp, mỗi tệp một giao dịch, ghi vào
// schema_migrations. Chạy bằng vai QUẢN TRỊ (tạo vai eais_app, bật RLS), không bằng eais_app.
// Danh tính của migration là tên tệp CỘNG checksum nội dung (F-036, ADR-0017). Trước khi áp gì, Migrate đối chiếu schema_migrations với
// các tệp và dừng hẳn (lỗi, không áp tệp nào) khi: (a) tệp đã áp mà nội dung đổi; (b) tệp chưa áp theo tên nhưng đã áp dưới tên khác —
// cùng checksum, hoặc trùng phần tên sau số (nhận được cả dòng ghi trước khi có checksum). Dòng cũ chưa có checksum được điền khi gặp
// lại tệp cùng tên.
func Migrate(ctx context.Context, adminDSN string) error {
	conn, err := pgx.Connect(ctx, adminDSN)
	if err != nil {
		return contracts.Label(contracts.Infra("postgres"), err)
	}
	defer conn.Close(ctx)
	// Các Migrate chạy CÙNG LÚC (go test chạy song song nhiều gói đều gọi Migrate; nhiều bản app khởi động một lượt) phải xếp hàng:
	// không có khoá thì hai phiên cùng CREATE TABLE trên database trống và một phiên chết với "duplicate key value violates unique
	// constraint pg_type_typname_nsp_index". Khoá mức phiên, nhả khi đóng kết nối.
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", migrateLockKey); err != nil {
		return contracts.Label(contracts.Infra("postgres"), err)
	}
	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return contracts.Label(contracts.Infra("postgres"), err)
	}
	// Cột thêm sau (F-036): bảng do bản cũ dựng thì có dòng checksum NULL, điền ở dưới.
	if _, err := conn.Exec(ctx, "ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS checksum text"); err != nil {
		return contracts.Label(contracts.Infra("postgres"), err)
	}
	rows, err := conn.Query(ctx, "SELECT version, coalesce(checksum, '') FROM schema_migrations ORDER BY version")
	if err != nil {
		return contracts.Label(contracts.Infra("postgres"), err)
	}
	var applied [][2]string // {version, checksum}; checksum "" = dòng ghi trước F-036
	var row [2]string
	if _, err := pgx.ForEachRow(rows, []any{&row[0], &row[1]}, func() error { applied = append(applied, row); return nil }); err != nil {
		return contracts.Label(contracts.Infra("postgres"), err)
	}
	names, err := fs.Glob(db.Migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)
	type migration struct {
		version, sum string
		sql          []byte
	}
	var pending, backfill []migration
	var problems []string
	for _, n := range names {
		sql, err := db.Migrations.ReadFile(n)
		if err != nil {
			return err
		}
		m := migration{path.Base(n), checksum(sql), sql}
		done, recorded, renamed := false, "", []string(nil)
		_, slugM, _ := strings.Cut(m.version, "_")
		for _, a := range applied {
			_, slugA, _ := strings.Cut(a[0], "_")
			switch {
			case a[0] == m.version:
				done, recorded = true, a[1]
			case a[1] == m.sum || slugA == slugM:
				renamed = append(renamed, a[0])
			}
		}
		switch {
		case done && recorded == "":
			backfill = append(backfill, m)
		case done && recorded != m.sum:
			problems = append(problems, fmt.Sprintf("%s đã áp nhưng nội dung tệp đã đổi (checksum đã ghi %.12s…, tệp %.12s…): không sửa migration đã đẩy, viết migration mới",
				m.version, recorded, m.sum))
		case !done && len(renamed) > 0:
			problems = append(problems, fmt.Sprintf("%s chưa áp theo tên nhưng DB đã áp %s (cùng nội dung hoặc cùng tên sau số): migration bị đổi tên / đổi số, không chạy lại — đối chiếu rồi sửa dòng schema_migrations",
				m.version, strings.Join(renamed, ", ")))
		case !done:
			pending = append(pending, m)
		}
	}
	if len(problems) > 0 {
		return contracts.Errorf(contracts.Infra("postgres"), "schema_migrations lệch tệp migration, chưa áp gì (docs/adr/0017): "+strings.Join(problems, "; "))
	}
	for _, m := range backfill {
		if _, err := conn.Exec(ctx, "UPDATE schema_migrations SET checksum = $2 WHERE version = $1 AND checksum IS NULL", m.version, m.sum); err != nil {
			return contracts.Label(contracts.Infra("postgres"), err)
		}
	}
	for _, m := range pending {
		err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, string(m.sql)); err != nil {
				return fmt.Errorf("migration %s: %w", m.version, err)
			}
			_, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version, checksum) VALUES ($1, $2)", m.version, m.sum)
			return err
		})
		if err != nil {
			return contracts.Label(contracts.Infra("postgres"), err)
		}
	}
	return nil
}

// checksum là danh tính nội dung của một migration: sha256 hex. Bỏ \r của CRLF để bản checkout trên Windows (core.autocrlf) không thành
// "nội dung đã đổi".
func checksum(sql []byte) string {
	h := sha256.Sum256(bytes.ReplaceAll(sql, []byte("\r\n"), []byte("\n")))
	return hex.EncodeToString(h[:])
}
