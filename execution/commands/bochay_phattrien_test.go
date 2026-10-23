package commands

// Cổng vào của ca PHÁT TRIỂN cho bộ chạy chung (bochay_test.go). Ca phát triển (testdata/chay) chạy trong `go test ./...` thường: luôn trên
// bộ nhớ, và trên PostgreSQL (database riêng của lần chạy) khi có EAIS_PG_ADMIN_DSN / EAIS_PG_DSN. Cổng vào của bộ ca giữ lại đọc đường tệp
// từ biến môi trường và chỉ bật bằng build tag giulai: bochay_giulai_test.go.

import (
	"os"
	"testing"
)

func loadDev(t *testing.T) (*caseSet, *worlds) {
	t.Helper()
	cs, err := loadCaseSet(devCfg().Cases)
	if err != nil {
		t.Fatal(err)
	}
	return cs, loadDevWorld(t)
}

func reportPath(label string) string {
	if p := os.Getenv("EAIS_CHAY_BAO_CAO"); p != "" {
		return p + "-" + label + ".json"
	}
	return ""
}

func TestCaPhatTrienBoNho(t *testing.T) {
	cs, wd := loadDev(t)
	rp := runCaseSet(t, cs, wd, newMemBackend(wd))
	t.Log("\n" + rp.render())
	writeReport(t, reportPath(rp.Backend), []*chayReport{rp})
}

func TestCaPhatTrienPG(t *testing.T) {
	cs, wd := loadDev(t)
	e := newScratchEnv(t, "chay") // database riêng của lần chạy; bỏ qua nếu thiếu DSN như mọi test PG của gói
	be := newPGBackend(e, wd)
	for _, tn := range wd.order {
		if err := be.loadData(tn); err != nil {
			t.Fatalf("nạp dữ liệu hạt giống của %s vào PG: %v", tn, err)
		}
	}
	rp := runCaseSet(t, cs, wd, be)
	t.Log("\n" + rp.render())
	writeReport(t, reportPath(rp.Backend), []*chayReport{rp})
}
