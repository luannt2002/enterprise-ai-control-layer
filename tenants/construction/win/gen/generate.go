package gen

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
)

// Generate đọc worldPath và ghi bộ dữ liệu WIN vào outDir theo CONTRACT.md §1–§5:
// <outDir>/manifest.json, một thư mục cho mỗi tenant của world, và variants/.
//
// profile là functional hoặc smoke. Cùng world + profile + seed thì mọi tệp giống hệt byte.
// Dữ liệu nền chỉ sinh cho tenant ĐẦU TIÊN của world (tenant "đủ bộ dữ liệu"); các tenant sau là
// canary cách ly, chỉ mang bản ghi và tài liệu của world. Nền không chạm vào world: thiết bị mã FX
// thuộc các dự án nền riêng PRJ-F01…, và bỏ trống mọi ngày xuất hiện trong world (CONTRACT §5).
func Generate(worldPath, outDir, profile string, seed uint64) error {
	spec, ok := profiles[profile]
	if !ok {
		return fmt.Errorf("hồ sơ %q không hợp lệ: chỉ nhận functional hoặc smoke", profile)
	}
	w, err := loadWorld(worldPath)
	if err != nil {
		return err
	}
	if err := checkBaseNames(w, spec.devices); err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	// Chỉ dọn những chỗ bộ sinh sở hữu, để chạy lại vào thư mục cũ vẫn ra đúng cùng một cây tệp.
	owned := []string{"manifest.json", "variants"}
	for _, t := range w.Tenants {
		owned = append(owned, t.ID)
	}
	for _, p := range owned {
		if err := os.RemoveAll(filepath.Join(outDir, p)); err != nil {
			return err
		}
	}

	worldDates := map[string]bool{}
	for _, r := range w.Records {
		worldDates[r.On] = true
	}
	head := fmt.Sprintf("{\"version\": 1, \"reference_date\": %q, \"profile\": %q, \"seed\": %d,\n \"records\": ", w.ReferenceDate, profile, seed)
	man, err := newJSONArray(filepath.Join(outDir, "manifest.json"), head, "}\n")
	if err != nil {
		return err
	}
	for i, t := range w.Tenants {
		in := tenantInput{def: t, records: w.recordsOf(t.ID), docs: w.docsOf(t.ID), ref: w.ref}
		if i == 0 {
			in.base = &baseSpec{tenant: t.ID, devices: spec.devices, days: spec.days, ref: w.ref, skip: worldDates, rng: rand.New(rand.NewPCG(seed, 0x57494e))}
		}
		if err := writeTenant(filepath.Join(outDir, t.ID), in, man.add); err != nil {
			_ = man.close()
			return fmt.Errorf("tenant %s: %w", t.ID, err)
		}
	}
	if err := man.close(); err != nil {
		return err
	}
	return writeVariants(w, outDir)
}
