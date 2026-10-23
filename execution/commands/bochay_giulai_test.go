//go:build giulai

package commands

// Cổng vào của bộ chạy chung cho bộ ca GIỮ LẠI độc lập (HOP-DONG §12, §12.1). Chỉ bật bằng build tag giulai, nên KHÔNG nằm trong
// `go test ./...` thường (không sinh test bị bỏ qua): điều phối chạy nó MỘT lần ở cổng ra, sau khi đóng băng và băm bộ ca.
//
// Cách chạy (PG riêng của lần chạy; test tạo database tạm trong cụm đó rồi xoá khi xong):
//
//	EAIS_PG_ADMIN_DSN=postgres://<quản trị>@<máy>:<cổng>/<database riêng>?sslmode=disable \
//	EAIS_PG_DSN=postgres://<vai ứng dụng>@<máy>:<cổng>/<database riêng>?sslmode=disable \
//	EAIS_GIULAI_CA=<đường tệp ca>            EAIS_GIULAI_SHA256_CA=<băm kỳ vọng của tệp ca> \
//	EAIS_GIULAI_THEGIOI=<đường tệp thế giới> EAIS_GIULAI_SHA256_THEGIOI=<băm kỳ vọng của tệp thế giới> \
//	go test -tags giulai -count=1 -run '^TestBoChayGiuLai$' -v ./execution/commands/
//
// Biến tuỳ chọn: EAIS_GIULAI_PACK (mặc định ../../tenants/construction/pack), EAIS_GIULAI_OVERLAY (thư mục thay policy.yaml / tools.yaml của
// pack, chỉ dùng khi pack chưa có tools.yaml; có dùng thì báo cáo ghi băm từng tệp overlay), EAIS_GIULAI_NEN (bonho | pg | ca-hai, mặc định
// ca-hai), EAIS_GIULAI_BAO_CAO (đường tệp JSON ghi báo cáo, thêm hậu tố -<nền>.json).
//
// Băm kỳ vọng là BẮT BUỘC: thiếu, hay khác băm thật, thì từ chối chạy. Báo cáo ghi băm tệp ca và tệp thế giới đã chạy, kết quả theo ca,
// theo lớp (khoá scenario) và theo cổng X1–X9 (không gộp một con số).

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBoChayGiuLai(t *testing.T) {
	need := func(name string) string {
		v := os.Getenv(name)
		if v == "" {
			t.Fatalf("thiếu biến môi trường %s (xem chú thích đầu tệp)", name)
		}
		return v
	}
	cfg := chayCfg{
		Cases:        need("EAIS_GIULAI_CA"),
		World:        need("EAIS_GIULAI_THEGIOI"),
		WantCasesSHA: need("EAIS_GIULAI_SHA256_CA"),
		WantWorldSHA: need("EAIS_GIULAI_SHA256_THEGIOI"),
		Pack:         os.Getenv("EAIS_GIULAI_PACK"),
		Overlay:      os.Getenv("EAIS_GIULAI_OVERLAY"),
		Seed:         42,
	}
	if cfg.Pack == "" {
		cfg.Pack = filepath.Join("..", "..", "tenants", "construction", "pack")
	}
	sum, err := fileSHA(cfg.Cases)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkWantSHA("tệp ca", sum, cfg.WantCasesSHA); err != nil {
		t.Fatal(err) // từ chối chạy khi băm lệch
	}
	cs, err := loadCaseSet(cfg.Cases)
	if err != nil {
		t.Fatal(err)
	}
	wd, err := worldFor(cfg) // kiểm băm tệp thế giới, rồi bộ sinh và đường nạp thật dựng dữ liệu hạt giống
	if err != nil {
		t.Fatal(err)
	}
	nen := os.Getenv("EAIS_GIULAI_NEN")
	if nen == "" {
		nen = "ca-hai"
	}
	var reports []*chayReport
	if nen == "bonho" || nen == "ca-hai" {
		t.Run("bo_nho", func(t *testing.T) {
			reports = append(reports, runCaseSet(t, cs, wd, newMemBackend(wd)))
		})
	}
	if nen == "pg" || nen == "ca-hai" {
		t.Run("pg", func(t *testing.T) {
			admin, app := os.Getenv("EAIS_PG_ADMIN_DSN"), os.Getenv("EAIS_PG_DSN")
			if admin == "" || app == "" {
				t.Fatal("nền pg cần EAIS_PG_ADMIN_DSN và EAIS_PG_DSN (PG riêng của lần chạy): thiếu thì ca ĐỎ, không bỏ qua")
			}
			e := newScratchEnv(t, "giulai")
			be := newPGBackend(e, wd)
			for _, tn := range wd.order {
				if err := be.loadData(tn); err != nil {
					t.Fatalf("nạp dữ liệu hạt giống của %s vào PG: %v", tn, err)
				}
			}
			reports = append(reports, runCaseSet(t, cs, wd, be))
		})
	}
	if nen != "bonho" && nen != "pg" && nen != "ca-hai" {
		t.Fatalf("EAIS_GIULAI_NEN %q không hợp lệ (bonho | pg | ca-hai)", nen)
	}
	for _, r := range reports {
		t.Log("\n" + r.render())
	}
	if p := os.Getenv("EAIS_GIULAI_BAO_CAO"); p != "" {
		for _, r := range reports {
			writeReport(t, p+"-"+r.Backend+".json", []*chayReport{r})
		}
	}
}
