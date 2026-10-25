package ingestion_test

import (
	"testing"
)

// F-009: bản ghi trùng khoá (cùng tệp, cùng record_id — kể cả cùng rỗng) không được làm hỏng cả lần nạp (khoá chính facts va chạm 23505):
// bản đầu nạp, bản sau vào cách ly lý do duplicate; mọi bản ghi vẫn hoặc là quan sát hoặc là cách ly, và id fact không trùng.
func TestBanGhiTrungKhoaVaoCachLy(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "entities.json", `{"projects":[{"id":"PRJ-A","name":"Cầu A"}],
"equipment":[{"id":"EX-03","name":"Máy xúc EX-03","project":"PRJ-A","source_ids":{"warehouse_slip":"EX-03"}}]}`)
	write(t, dir, "warehouse.csv", "record_id,equipment_code,issue_date,quantity,unit\n"+
		"R-1,EX-03,2026-10-19,100,L\n"+
		"R-1,EX-03,2026-10-19,100,L\n"+ // lặp y nguyên một dòng
		",EX-03,2026-10-19,200,L\n"+
		",EX-03,2026-10-19,300,L\n"+ // hai dòng cùng không có record_id
		"R-2,EX-03,2026-10-19,50,L\n")
	res := ingest(t, dir)
	obs := map[string]string{}
	for _, o := range res.Observations {
		obs[o.Key] = o.Value.Number
	}
	if len(obs) != 3 || obs["warehouse.csv#R-1"] != "100" || obs["warehouse.csv#"] != "200" || obs["warehouse.csv#R-2"] != "50" {
		t.Errorf("quan sát = %v, muốn bản ĐẦU của mỗi khoá: R-1=100, (rỗng)=200, R-2=50", obs)
	}
	var q []string
	for _, x := range res.Quarantined {
		q = append(q, x.Key+" "+x.Reason)
	}
	if len(q) != 2 || q[0] != "warehouse.csv# duplicate" || q[1] != "warehouse.csv#R-1 duplicate" {
		t.Errorf("cách ly = %q, muốn hai bản sau vào cách ly lý do duplicate", q)
	}
	ids := map[string]bool{}
	for _, f := range res.Facts() {
		if ids[f.ID] {
			t.Errorf("id fact trùng %s: khoá chính facts (tenant, id) sẽ va chạm", f.ID)
		}
		ids[f.ID] = true
	}
}
