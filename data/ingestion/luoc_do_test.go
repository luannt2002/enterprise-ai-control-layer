package ingestion_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/ingestion"
)

// luocDoKhac là ánh xạ nguồn với lược đồ danh mục KHÁC pack tham chiếu: khách hàng → đơn hàng, đơn trỏ khách qua trường "customer";
// chữ tự do nhận dạng đơn hàng. Chỉ số / loại nguồn mượn của pack tham chiếu để kiểm riêng phần lược đồ (F-004).
const luocDoKhac = `entities_file: danh-muc.json
entity_lists:
  - {list: customers, type: customer}
  - {list: orders, type: order, parent_field: customer}
free_text_entity: {type: order, label: đơn hàng}
sources:
  - type: warehouse_slip
    metric: issued_fuel
    files: [{path: xuat.csv, format: csv}]
    fields: {record_id: record_id, entity: ma, date: ngay, quantity: luong}
    default_unit: L
    entity_namespace: kho
  - type: chat_message
    metric: reported_fuel
    files: [{path: tin.jsonl, format: jsonl}]
    fields: {record_id: record_id, text: text, datetime: sent_at, sender: sender}
    default_unit: L
    free_text: true
    carry_entity: true
`

func napLuocDo(t *testing.T, sources string) (string, ingestion.SourceMap, *tenant.Runtime) {
	t.Helper()
	rt, err := tenant.Load(packDir)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	write(t, dir, "sources.yaml", sources)
	write(t, dir, "danh-muc.json", `{"customers":[{"id":"KH-1","name":"Công ty Một","aliases":["khách một"]}],
"orders":[{"id":"DH-1","name":"Đơn DH-1","customer":"KH-1","aliases":["đơn cát"],"source_ids":{"kho":"SO-1"}},
 {"id":"DH-2","name":"Đơn DH-2","customer":"KH-1","aliases":["đơn đá"],"source_ids":{"kho":"SO-2"}}],
"ghi_chu":"khoá không khai thì bỏ qua"}`)
	write(t, dir, "xuat.csv", "record_id,ma,ngay,luong\nR-1,SO-1,2026-10-19,120\nR-2,SO-9,2026-10-19,5\n")
	write(t, dir, "tin.jsonl", strings.Join([]string{
		`{"record_id":"T-1","sent_at":"2026-10-19T08:00:00+07:00","sender":"Lan","text":"đơn đá 300 lít"}`,
		`{"record_id":"T-2","sent_at":"2026-10-19T09:00:00+07:00","sender":"Lan","text":"thêm 50 lít"}`,
		`{"record_id":"T-3","sent_at":"2026-10-19T09:30:00+07:00","sender":"Minh","text":"khách một thêm 70 lít"}`,
	}, "\n")+"\n")
	sm, err := ingestion.LoadSources(filepath.Join(dir, "sources.yaml"), rt)
	if err != nil {
		t.Fatal(err)
	}
	return dir, sm, rt
}

// F-004: loại thực thể, quan hệ cha và loại mà chữ tự do nhận dạng do pack khai — danh mục khách → đơn (khác dự án → thiết bị) nạp được,
// nguồn có cấu trúc tra mã, chữ tự do nhận dạng đơn hàng, lý do cách ly và ghi chú kế thừa gọi đúng tên loại của pack.
func TestLuocDoDanhMucDoPackKhai(t *testing.T) {
	dir, sm, rt := napLuocDo(t, luocDoKhac)
	entities, err := ingestion.LoadDirectory(dir, sm)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entities {
		var al []string
		for _, a := range e.Aliases {
			al = append(al, a.Text+"@"+a.Source)
		}
		got = append(got, e.ID+"|"+e.Type+"|"+e.Parent+"|"+strings.Join(al, ","))
	}
	want := []string{"KH-1|customer||khách một@master", "DH-1|order|KH-1|đơn cát@master,SO-1@kho", "DH-2|order|KH-1|đơn đá@master,SO-2@kho"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("danh mục:\n%s\nmuốn:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	res, err := ingestion.Ingest(context.Background(), dir, rt, sm, nil)
	if err != nil {
		t.Fatal(err)
	}
	obs := map[string]string{}
	notes := map[string]string{}
	for _, o := range res.Observations {
		obs[o.Key] = o.Entity + " " + o.Value.Number
		notes[o.Key] = o.Note
	}
	for k, w := range map[string]string{"xuat.csv#R-1": "DH-1 120", "tin.jsonl#T-1": "DH-2 300", "tin.jsonl#T-2": "DH-2 50"} {
		if obs[k] != w {
			t.Errorf("%s = %q, muốn %q", k, obs[k], w)
		}
	}
	if !strings.HasPrefix(notes["tin.jsonl#T-2"], "đơn hàng kế thừa") {
		t.Errorf("ghi chú kế thừa phải gọi tên loại của pack: %q", notes["tin.jsonl#T-2"])
	}
	q := map[string]string{}
	for _, x := range res.Quarantined {
		q[x.Key] = x.Reason + ": " + x.Detail
	}
	if !strings.HasPrefix(q["xuat.csv#R-2"], ingestion.ReasonUnknownEntity) {
		t.Errorf("mã SO-9 không có trong danh mục phải cách ly unknown_entity: %q", q["xuat.csv#R-2"])
	}
	if q["tin.jsonl#T-3"] != ingestion.ReasonUnknownEntity+": tin không nhắc đơn hàng nào" {
		t.Errorf("tin không nhắc thực thể: lý do phải gọi tên loại của pack, được %q", q["tin.jsonl#T-3"])
	}
}

// Lược đồ danh mục thiếu hay sai là lỗi cấu hình: chặn lúc nạp sources.yaml, không để mọi bản ghi rơi vào cách ly unknown_entity.
func TestLuocDoDanhMucHongBiChan(t *testing.T) {
	rt, err := tenant.Load(packDir)
	if err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct{ from, to, want string }{
		"thiếu entity_lists":       {"entity_lists:\n  - {list: customers, type: customer}\n  - {list: orders, type: order, parent_field: customer}\n", "", "entity_lists"},
		"mảng thiếu loại":          {"{list: customers, type: customer}", "{list: customers}", "type"},
		"mảng khai hai lần":        {"{list: orders, type: order, parent_field: customer}", "{list: customers, type: order}", "customers"},
		"chữ tự do thiếu loại":     {"free_text_entity: {type: order, label: đơn hàng}\n", "", "free_text_entity"},
		"chữ tự do loại chưa khai": {"free_text_entity: {type: order,", "free_text_entity: {type: invoice,", "invoice"},
		"chữ tự do thiếu tên gọi":  {"free_text_entity: {type: order, label: đơn hàng}", "free_text_entity: {type: order}", "label"},
	} {
		p := filepath.Join(t.TempDir(), "sources.yaml")
		if err := os.WriteFile(p, []byte(strings.Replace(luocDoKhac, c.from, c.to, 1)), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := ingestion.LoadSources(p, rt); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: LoadSources phải lỗi nhắc %q, được %v", name, c.want, err)
		}
	}
}
