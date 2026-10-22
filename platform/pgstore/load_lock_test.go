package pgstore

import "testing"

// Khoá nạp theo tenant: tất định, khác nhau giữa các tenant (tenant khác nhau không chờ nhau) và không trùng khoá của Migrate.
func TestLoadLockKey(t *testing.T) {
	a, b := loadLockKey("abc-infra"), loadLockKey("def-build")
	if a == b {
		t.Error("hai tenant khác nhau cùng một khoá nạp: nạp của tenant này sẽ chờ tenant kia")
	}
	if loadLockKey("abc-infra") != a {
		t.Error("khoá nạp không tất định: hai tiến trình sẽ không xếp hàng với nhau")
	}
	if a == migrateLockKey || b == migrateLockKey {
		t.Error("khoá nạp trùng khoá của Migrate")
	}
}
