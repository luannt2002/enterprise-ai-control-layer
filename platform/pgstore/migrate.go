package pgstore

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
	"github.com/luannt2002/enterprise-ai-systems/db"
)

// Migrate áp các migration chưa áp, theo thứ tự tên tệp, mỗi tệp một giao dịch, ghi vào
// schema_migrations. Chạy bằng vai QUẢN TRỊ (tạo vai eais_app, bật RLS), không bằng eais_app.
func Migrate(ctx context.Context, adminDSN string) error {
	conn, err := pgx.Connect(ctx, adminDSN)
	if err != nil {
		return contracts.Label(contracts.Infra("postgres"), err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return contracts.Label(contracts.Infra("postgres"), err)
	}
	names, err := fs.Glob(db.Migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)
	for _, n := range names {
		version := path.Base(n)
		var done bool
		if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&done); err != nil {
			return contracts.Label(contracts.Infra("postgres"), err)
		}
		if done {
			continue
		}
		sql, err := db.Migrations.ReadFile(n)
		if err != nil {
			return err
		}
		err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, string(sql)); err != nil {
				return fmt.Errorf("migration %s: %w", version, err)
			}
			_, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version)
			return err
		})
		if err != nil {
			return contracts.Label(contracts.Infra("postgres"), err)
		}
	}
	return nil
}
