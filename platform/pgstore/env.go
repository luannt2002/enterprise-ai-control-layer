package pgstore

import "os"

// legacyEnvPrefix là tiền tố biến môi trường DSN dùng từ sáng tới tối 21/10/2026, trước D-RM-02. Chỉ EnvDSN đọc nó, và chỉ khi
// biến EAIS_ cùng tên KHÔNG được đặt (cùng luật với biến model, models.modelEnv).
const legacyEnvPrefix = "EACL_PG_"

// EnvDSN trả DSN vai quản trị (EAIS_PG_ADMIN_DSN, cho Migrate) và DSN vai ứng dụng (EAIS_PG_DSN, cho Open). Biến EAIS_ đã đặt —
// kể cả đặt rỗng — luôn thắng; không đặt thì đọc biến cùng tên của tiền tố tương thích legacyEnvPrefix.
func EnvDSN() (admin, app string) {
	return env("ADMIN_DSN"), env("DSN")
}

func env(name string) string {
	if v, ok := os.LookupEnv("EAIS_PG_" + name); ok {
		return v
	}
	return os.Getenv(legacyEnvPrefix + name)
}
