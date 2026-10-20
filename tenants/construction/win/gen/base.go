package gen

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"
)

// profileSpec là kích cỡ dữ liệu nền của một hồ sơ (CONTRACT §5).
type profileSpec struct{ devices, days int }

var profiles = map[string]profileSpec{
	"functional": {devices: 40, days: 30},
	"smoke":      {devices: 400, days: 365},
}

// baseSpec mô tả dữ liệu nền của một tenant: devices thiết bị × days ngày, kết thúc ở ngày ref.
// skip là các ngày xuất hiện trong world: nền bỏ trống những ngày đó để không chạm vào ngày của world.
type baseSpec struct {
	tenant        string
	devices, days int
	ref           time.Time
	skip          map[string]bool
	rng           *rand.Rand
}

// baseProjectCount là số dự án nền riêng PRJ-F01…: thiết bị nền KHÔNG thuộc dự án nào của world, để
// tổng theo dự án / kỳ của world (ví dụ "Cầu A tháng 9") không bị lẫn dữ liệu nền.
const baseProjectCount = 4

// baseProjects dựng các dự án nền; không alias nên không thể trùng id hay bí danh của world.
func baseProjects() []project {
	out := make([]project, baseProjectCount)
	for i := range out {
		out[i] = project{ID: fmt.Sprintf("PRJ-F%02d", i+1), Name: fmt.Sprintf("Dự án nền %02d", i+1), Aliases: []string{}}
	}
	return out
}

// baseEquipment dựng n thiết bị nền FX-001…, chia đều cho các dự án nền. Mã FX không bao giờ trùng
// thiết bị của world nên bản ghi nền không chạm vào (thiết bị, ngày) nào của world.
func baseEquipment(n int) []equipment {
	kinds := []string{"excavator", "dozer", "roller"}
	out := make([]equipment, n)
	for i := range out {
		id := fmt.Sprintf("FX-%03d", i+1)
		out[i] = equipment{
			ID: id, Type: kinds[i%len(kinds)], Name: "Thiết bị nền " + id,
			Project: fmt.Sprintf("PRJ-F%02d", i*baseProjectCount/n+1),
			Aliases: []string{},
			SourceIDs: map[string]string{
				"warehouse_slip": id,
				"erp":            fmt.Sprintf("asset-%d", 9001+i),
				"fuel_sensor":    fmt.Sprintf("FX%03d", i+1),
			},
		}
	}
	return out
}

// checkBaseNames chặn dữ liệu nền trùng id, tên, bí danh hay mã theo nguồn với bất kỳ dự án / thiết bị
// nào của world: trùng thì thực thể nền và thực thể world lẫn nhau khi nhận dạng.
func checkBaseNames(w *world, devices int) error {
	taken := map[string]string{} // chữ thường → nó là gì trong world
	for _, t := range w.Tenants {
		for _, p := range t.Projects {
			taken[strings.ToLower(p.ID)], taken[strings.ToLower(p.Name)] = "id dự án "+p.ID, "tên dự án "+p.ID
			for _, a := range p.Aliases {
				taken[strings.ToLower(a)] = "bí danh dự án " + p.ID
			}
		}
		for _, e := range t.Equipment {
			taken[strings.ToLower(e.ID)], taken[strings.ToLower(e.Name)] = "id thiết bị "+e.ID, "tên thiết bị "+e.ID
			for _, a := range e.Aliases {
				taken[strings.ToLower(a)] = "bí danh thiết bị " + e.ID
			}
			for _, v := range e.SourceIDs {
				taken[strings.ToLower(v)] = "mã nguồn của thiết bị " + e.ID
			}
		}
	}
	var mine []string
	for _, p := range baseProjects() {
		mine = append(mine, p.ID, p.Name)
	}
	for _, e := range baseEquipment(devices) {
		mine = append(mine, e.ID, e.Name)
		for _, v := range e.SourceIDs {
			mine = append(mine, v)
		}
	}
	for _, m := range mine {
		if what, ok := taken[strings.ToLower(m)]; ok {
			return fmt.Errorf("dữ liệu nền dùng %q trùng với %s của world", m, what)
		}
	}
	return nil
}

// Tỉ lệ lỗi bơm vào phiếu kho nền, tính trên 1000 phiếu (THAM SỐ GIẢ LẬP, không phải tỉ lệ doanh
// nghiệp thật): 2% thiếu lượng, 1% mã thiết bị lạ, 0,5% đơn vị kg.
const (
	missingPer1000 = 20
	unknownPer1000 = 10
	kgPer1000      = 5
)

// genBase sinh dữ liệu nền theo ngày (bỏ ngày của world), trong ngày theo thứ tự thiết bị. Mỗi thiết
// bị mỗi ngày có phiếu kho, ERP (khoảng 50% ngày, lệch 0–20 L so với phiếu) và cảm biến. Mọi giá trị
// ngẫu nhiên của một lượt đều được rút MỘT lần, theo thứ tự cố định, nên seed quyết định toàn bộ đầu ra.
func genBase(sp *baseSpec, eqs []equipment, s *streams, emit func(any)) {
	seq := 0
	next := func() string { seq++; return fmt.Sprintf("F-%06d", seq) }
	for d := 0; d < sp.days; d++ {
		date := sp.ref.AddDate(0, 0, d-sp.days+1).Format(dateLayout)
		if sp.skip[date] {
			continue
		}
		for _, e := range eqs {
			liters := 300 + sp.rng.IntN(1701) // lượng nhiên liệu thật của thiết bị trong ngày
			fault := sp.rng.IntN(1000)
			hasERP := sp.rng.IntN(2) == 0
			erpGap := sp.rng.IntN(21)
			sensorGap := sp.rng.IntN(60)
			strange := sp.rng.IntN(1000)

			// Phiếu kho: nguồn người nhập nên là nơi bơm lỗi.
			id, code, qty, unit, reason := next(), e.SourceIDs["warehouse_slip"], strconv.Itoa(liters), "L", ""
			switch {
			case fault < missingPer1000:
				qty, reason = "", "missing_value"
			case fault < missingPer1000+unknownPer1000:
				code, reason = fmt.Sprintf("FX-9%03d", strange), "unknown_entity"
			case fault < missingPer1000+unknownPer1000+kgPer1000:
				unit, reason = "kg", "unit_dimension"
			}
			s.slip.row(id, "PX"+id, code, date, qty, unit)
			emit(baseEntry(sp.tenant, "warehouse.csv", id, e.ID, "issued_fuel", "warehouse_slip", date, qty, reason))

			if hasERP {
				id = next()
				q := strconv.Itoa(liters + erpGap)
				s.erp.add(erpRow{RecordID: id, DocNo: "ERP" + id, AssetID: e.SourceIDs["erp"], PostingDate: date, Material: "DIESEL", Quantity: q, UOM: "L"})
				emit(baseEntry(sp.tenant, "erp.json", id, e.ID, "issued_fuel", "erp", date, q, ""))
			}

			id = next()
			q := strconv.Itoa(liters - sensorGap)
			s.sensor.row(id, e.SourceIDs["fuel_sensor"], date, q)
			emit(baseEntry(sp.tenant, "sensor.csv", id, e.ID, "measured_consumption", "fuel_sensor", date, q, ""))
		}
	}
}
