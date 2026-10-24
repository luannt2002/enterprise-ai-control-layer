package ingestion_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/ingestion"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// nguonGoc là người gửi | thời điểm gửi (RFC3339, UTC) của một fact — rỗng / "0" khi nguồn không có.
func nguonGoc(f sourceoftruth.Fact) string {
	at := "0"
	if !f.Source.ObservedAt.IsZero() {
		at = f.Source.ObservedAt.UTC().Format(time.RFC3339)
	}
	return f.Source.ReportedBy + "|" + at
}

// muonNguonGoc: tin nhắn mang người gửi + giờ gửi, kể cả tin kế thừa thực thể; dòng phiếu có cấu trúc không có cả hai.
var muonNguonGoc = map[string]string{
	"tin.jsonl#T-1": "Lan|2026-10-19T01:00:00Z",
	"tin.jsonl#T-2": "Lan|2026-10-19T02:00:00Z",
	"xuat.csv#R-1":  "|0",
}

// F-008: Observation → Fact giữ người gửi và thời điểm gửi (cột reported_by, observed_at của facts) — trước đây mất im lặng, nên không
// truy được ai báo, và tổng "last" (xếp theo ObservedAt) không xác định với dữ liệu nạp.
func TestFactGiuNguoiGuiVaGioGui(t *testing.T) {
	dir, sm, rt := napLuocDo(t, luocDoKhac)
	res, err := ingestion.Ingest(context.Background(), dir, rt, sm, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range res.Facts() {
		got[f.ID] = nguonGoc(f)
	}
	for k, w := range muonNguonGoc {
		if got[k] != w {
			t.Errorf("%s: nguồn gốc %q, muốn %q", k, got[k], w)
		}
	}
}

// Cùng dữ liệu qua PostgreSQL: nạp (kèm một dòng phiếu lặp y nguyên — F-009: phải vào cách ly, không làm hỏng lần nạp) rồi đọc lại fact
// — người gửi và giờ gửi đọc lại đủ; cách ly lưu được lý do duplicate.
func TestFactGiuNguonGocQuaPG(t *testing.T) {
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
	b, err := os.ReadFile(filepath.Join(dir, "xuat.csv"))
	if err != nil {
		t.Fatal(err)
	}
	write(t, dir, "xuat.csv", string(b)+"R-1,SO-1,2026-10-19,120\n")
	res, err := ingestion.Ingest(ctx, dir, rt, sm, nil)
	if err != nil {
		t.Fatal(err)
	}
	entities, err := ingestion.LoadDirectory(dir, sm)
	if err != nil {
		t.Fatal(err)
	}
	tn := fmt.Sprintf("nguon-goc-%d", time.Now().UnixNano())
	tctx := contracts.WithTenant(ctx, contracts.TenantID(tn))
	var recs []pgstore.QuarantineRecord
	for _, q := range res.Quarantined {
		recs = append(recs, pgstore.QuarantineRecord{Key: q.Key, Source: q.Source, Reason: q.Reason, Detail: q.Detail})
	}
	if _, err := s.SaveQuarantine(tctx, "nap-1", recs); err != nil {
		t.Fatal(err)
	}
	if err := s.Load(tctx, entities, res.Facts(), nil); err != nil {
		t.Fatalf("nạp có bản ghi trùng khoá phải thành công (bản trùng vào cách ly): %v", err)
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
			got[f.ID] = nguonGoc(f)
		}
	}
	for k, w := range muonNguonGoc {
		if got[k] != w {
			t.Errorf("%s: đọc lại từ PG nguồn gốc %q, muốn %q", k, got[k], w)
		}
	}
	q, err := s.Quarantine(tctx, "nap-1")
	if err != nil {
		t.Fatal(err)
	}
	var dup []string
	for _, r := range q {
		if r.Reason == ingestion.ReasonDuplicate {
			dup = append(dup, r.Key)
		}
	}
	if strings.Join(dup, ",") != "xuat.csv#R-1" {
		t.Errorf("cách ly lý do duplicate = %v, muốn [xuat.csv#R-1]", dup)
	}
}
