// Package db giữ migration SQL của EAIS, nhúng vào binary để test và công cụ áp cùng một bản.
package db

import "embed"

// Migrations là các tệp migrations/NNNN_*.sql, áp theo thứ tự tên tệp.
//
//go:embed migrations/*.sql
var Migrations embed.FS
