package pipeline

import (
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Soát C8–C11 C10-S3: chủ thể truy vấn (firstEntitySlot) và kỳ (periodBounds) lấy theo TÊN khe đã sắp xếp, không theo thứ tự lặp của map —
// cùng ý định, cùng khe thì cùng chủ thể và cùng kỳ ở mọi lần chạy (ADR-0002 D18). Ghim bằng nhiều lần chạy: thứ tự map của Go đổi giữa các lần lặp.
func TestKheChuTheVaKyTheoThuTuCoDinh(t *testing.T) {
	ent := func(key string) contracts.Slot {
		return contracts.Slot{Entity: &contracts.EntityRef{ID: key, Key: key, Type: "equipment"}}
	}
	day := func(text string) contracts.Slot {
		return contracts.Slot{Value: &contracts.Value{Kind: contracts.ValueDate, Text: text}}
	}
	it := contracts.Intent{Slots: map[string]contracts.Slot{
		"z_may": ent("EX-ZZ"), "a_may": ent("EX-AA"), "m_may": ent("EX-MM"),
		"z_ky": day("2026-10-03"), "b_ky": day("2026-10-01"), "q_ky": day("2026-10-02"),
	}}
	now := time.Date(2026, 10, 19, 10, 0, 0, 0, time.FixedZone("ICT", 7*3600))
	for i := range 200 {
		if got := firstEntitySlot(it).Entity.Key; got != "EX-AA" {
			t.Fatalf("lần %d: chủ thể %s, muốn EX-AA (khe a_may đứng đầu theo tên)", i, got)
		}
		from, _, err := periodBounds(it, now)
		if err != nil || from.Format("2006-01-02") != "2026-10-01" {
			t.Fatalf("lần %d: kỳ bắt đầu %v (lỗi %v), muốn 2026-10-01 (khe b_ky đứng đầu theo tên)", i, from, err)
		}
	}
}
