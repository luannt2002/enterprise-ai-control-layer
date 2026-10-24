package ingestion_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/ingestion"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// muonGhiChu: tin kế thừa thực thể của tin trước mang ghi chú suy luận; tin nêu thực thể và dòng phiếu có cấu trúc thì không.
var muonGhiChu = map[string]string{
	"tin.jsonl#T-1": "",
	"tin.jsonl#T-2": "đơn hàng kế thừa từ tin trước của Lan cùng ngày",
	"xuat.csv#R-1":  "",
}

// F-008 (phần còn lại): ghi chú suy luận đã áp khi nạp (thực thể kế thừa từ tin trước, carry_entity) đi theo fact — trước đây chỉ có ở
// Observation của ingestion, mất khi thành fact, nên không biết con số nào gắn thực thể bằng suy luận.
func TestFactGiuGhiChuSuyLuan(t *testing.T) {
	dir, sm, rt := napLuocDo(t, luocDoKhac)
	res, err := ingestion.Ingest(context.Background(), dir, rt, sm, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range res.Facts() {
		got[f.ID] = f.Source.Note
	}
	for k, w := range muonGhiChu {
		if n, ok := got[k]; !ok || n != w {
			t.Errorf("%s: ghi chú %q (có fact: %v), muốn %q", k, n, ok, w)
		}
	}
}

// Cùng dữ liệu qua PostgreSQL (cột facts.note, migration 0011): nạp rồi đọc lại — ghi chú đọc lại đủ, fact không suy luận đọc lại rỗng.
func TestFactGiuGhiChuSuyLuanQuaPG(t *testing.T) {
	admin, app := pgstore.EnvDSN()
	if admin == "" || app == "" {
		t.Skip("chưa có EAIS_PG_ADMIN_DSN / EAIS_PG_DSN — xem deploy/docker-compose.yml")
	}
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	s, err := pgstore.Open(ctx, app)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	dir, sm, rt := napLuocDo(t, luocDoKhac)
	res, err := ingestion.Ingest(ctx, dir, rt, sm, nil)
	if err != nil {
		t.Fatal(err)
	}
	entities, err := ingestion.LoadDirectory(dir, sm)
	if err != nil {
		t.Fatal(err)
	}
	tctx := contracts.WithTenant(ctx, contracts.TenantID(fmt.Sprintf("ghi-chu-%d", time.Now().UnixNano())))
	if err := s.Load(tctx, entities, res.Facts(), nil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Load(tctx, nil, nil, nil) })
	day := time.Date(2026, 10, 19, 0, 0, 0, 0, rt.Location)
	got := map[string]string{}
	for _, m := range []string{"reported_fuel", "issued_fuel"} {
		fs, err := s.Facts(tctx, m, nil, day, day)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range fs {
			got[f.ID] = f.Source.Note
		}
	}
	for k, w := range muonGhiChu {
		if n, ok := got[k]; !ok || n != w {
			t.Errorf("%s: đọc lại từ PG ghi chú %q (có fact: %v), muốn %q", k, n, ok, w)
		}
	}
}
