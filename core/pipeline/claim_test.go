package pipeline

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/tenant"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
	"github.com/luannt2002/enterprise-ai-control-layer/simulations/fixtures"
)

// verdictGap gọi thẳng (pack + danh mục thật của tenant cầu đường): cổng tự kiểm con số trong câu đúng là con số được xác nhận, không
// dựa vào việc luồng hỏi đã chọn đúng; ký hiệu chưa được giải thích ("~", ">", "/") là phần dư. Chủ thể là thực thể nhắc đầu tiên
// (như router; bí danh mơ hồ thì ghi rõ thực thể router gỡ ra), nguồn dùng mặc định là phiếu xuất kho.
func TestVerdictGap(t *testing.T) {
	rt, err := tenant.Load("../../tenants/construction/pack")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := fixtures.LoadDirectory("../../tenants/construction/seed/entities.yaml")
	if err != nil {
		t.Fatal(err)
	}
	a := &Ask{RT: rt, Deps: Deps{Directory: dir}}
	def, _ := rt.Metrics.Metric("issued_fuel")
	now := time.Date(2026, 10, 19, 10, 0, 0, 0, time.FixedZone("ICT", 7*3600))
	ctx := contracts.WithTenant(context.Background(), "abc-infra")
	for _, c := range []struct {
		q, claimed, want string
		subject          string // trống = thực thể nhắc đầu tiên; bí danh mơ hồ do router gỡ thì ghi rõ
		used             []string
	}{
		{q: "EX-03 hôm nay được cấp 3850 L đúng không?", claimed: "3850"},
		{q: "EX-03 hôm nay được cấp 3850 L đúng không?", claimed: "4000", want: "value"},
		{q: "EX-03 hôm nay được cấp ~4000 L đúng không?", claimed: "4000", want: "residual"},
		{q: "EX-03 hôm nay được cấp > 3000 L đúng không?", claimed: "3000", want: "residual"},
		{q: "EX-03 hôm nay được cấp 3850 L/ngày đúng không?", claimed: "3850", want: "residual"},
		{q: "EX-03 hôm nay được cấp 3850 đúng không?", claimed: "3850", want: "value"},
		{q: "EX-03 được cấp 3850 L đúng không?", claimed: "3850", want: "period"},
		// số trần có dấu chấm / phẩy: mơ hồ phần nghìn / thập phân, không nói đơn vị
		{q: "EX-03 hôm nay được cấp 3.850 đúng không?", claimed: "3850", want: "value"},
		// ngoài miền một lần đo: "3.850 m3" đọc thành 3.850.000 L
		{q: "EX-03 hôm nay được cấp 3.850 m3 đúng không?", claimed: "3850000", want: "value"},
		// gạch dài dính vào số là dấu âm không đọc được, không phải dấu câu
		{q: "EX-03 hôm nay được cấp —3850 L đúng không?", claimed: "3850", want: "residual"},
		// câu không dấu: khớp bỏ dấu làm chữ trùng nhau
		{q: "EX-03 hom nay duoc cap 3850 L dung khong?", claimed: "3850", want: "accent"},
		// hai cụm xác nhận là hai claim
		{q: "EX-03 hôm nay được cấp 3850 L đúng không, 3850 L đúng không?", claimed: "3850", want: "marker"},
		// chủ thể: dự án B không phải cấp cha của EX-03; dự án A (nhắc đầu) chứa EX-03 chứ không phải cha của nó
		{q: "EX-03 của dự án B hôm nay được cấp 3850 L đúng không?", claimed: "3850", want: "subject"},
		{q: "Dự án A hôm nay được cấp 3850 L cho EX-03 đúng không?", claimed: "3850", want: "subject"},
		{q: "Máy 03 của dự án A hôm nay được cấp 3850 L đúng không?", claimed: "3850", subject: "EX-03"}, // "máy 03" mơ hồ, router gỡ nhờ dự án A
		// nguồn: tên nguồn chỉ được giải thích khi số trả lời lấy đúng và chỉ từ nguồn đó
		{q: "EX-03 hôm nay phiếu xuất kho ghi 3850 L đúng không?", claimed: "3850"},
		{q: "EX-03 hôm nay phiếu xuất kho ghi 3900 L đúng không?", claimed: "3900", want: "source", used: []string{"erp"}},
		{q: "EX-03 hôm nay phiếu kho ghi 3900 L đúng không?", claimed: "3900", want: "source", used: []string{"erp"}},
		{q: "EX-03 hôm nay ERP ghi 3900 L đúng không?", claimed: "3900", want: "source"},
		// "tuần" trước một ngày là tuần chứa ngày đó, không phải ngày
		{q: "EX-03 tuần 15/9 được cấp 3500 L đúng không?", claimed: "3500", want: "residual"},
		// T2.2 (vòng 2 + soát lần ba): con số nhắc lại chỉ là cùng claim khi liền kề; kiểm chủ thể ở cả phạm vi câu lẫn mệnh đề hỏi;
		// lượng đơn vị suy từ thuật ngữ mà có dấu chấm / phẩy là mơ hồ — trừ khi nguyên văn có chữ ("3,85k")
		{q: "EX-03 hôm nay được cấp 3850 L, dự án A 3850 L, đúng không?", claimed: "3850", want: "value"},
		{q: "EX-03 hôm nay được cấp 3850 L, dự án A đúng không?", claimed: "3850", want: "subject"},
		{q: "EX-03 hôm nay được cấp 3850 L. Dự án A được cấp 3850 L, đúng không?", claimed: "3850", want: "subject"},
		{q: "EX-03 hôm nay được cấp 3850 L. Dự án A 5050 L, đúng không?", claimed: "5050", want: "subject"},
		{q: "Lu 02 hôm nay được cấp dầu 1,2 đúng không?", claimed: "1.2", want: "value"},
		{q: "EX-03 hôm nay được cấp dầu 3,85k đúng không?", claimed: "3850"},
		{q: "EX-03 hôm nay được cấp dầu 3,85K đúng không?", claimed: "3850"},
		{q: "EX-03 hôm nay được cấp dầu 3,5k35 đúng không?", claimed: "3850", want: "value"},
		// E3: sau con số xác nhận chỉ được nhắc lại chính chủ thể
		{q: "EX-03 hôm nay được cấp 3850 L (3850 L của dự án A) đúng không?", claimed: "3850", want: "subject"},
		{q: "EX-03 hôm nay được cấp 3850 L: dự án A đúng không?", claimed: "3850", want: "subject"},
		// Vòng 5: thứ gần nhất đứng TRƯỚC con số phải gắn vào chủ thể — cấp cha / danh từ chung / bí danh tách câu (R4-2, R4-3, R4-4),
		// tên nguồn ở ngữ cảnh (R-1); số bằng chữ "<hàng> <một chữ số>" chưa đọc chắc (K5); ngày viết số không phải ngắt mệnh đề
		{q: "EX-03 hôm nay được cấp 3850 L. Dự án A, 5050 L đúng không?", claimed: "5050", want: "subject"},
		{q: "EX-03 hôm nay được cấp 3850 L. Dự án, 5050 L đúng không?", claimed: "5050", want: "subject"},
		{q: "EX-03 hôm nay được cấp 3850 L. Máy 03 được cấp 3000 L, đúng không?", claimed: "3000", want: "subject"},
		{q: "EX-03 hôm nay được cấp 3850 L. ERP, 3900 L đúng không?", claimed: "3900", want: "source"},
		{q: "ERP, EX-03 hôm nay được cấp 3900 L đúng không?", claimed: "3900", want: "source"},
		{q: "EX-03 hôm nay đội báo, được cấp 3850 L đúng không?", claimed: "3850", want: "metric"}, // trước bước khoá: "subject" (chủ gần nhất)
		{q: "Dự án hôm nay được cấp 3850 L cho EX-03 đúng không?", claimed: "3850", want: "subject"},
		{q: "EX-03 hôm nay được cấp 3850 L, máy 03 đúng không?", claimed: "3850", want: "subject"},
		{q: "DZ-03 (máy 03) hôm nay được cấp 3850 L đúng không?", claimed: "3850", subject: "DZ-03"}, // "máy 03" mơ hồ, có chủ thể trong ứng viên, trong ngoặc liền sau lần nhắc rõ
		{q: "Lu 02 hôm nay được cấp một nghìn hai lít dầu đúng không?", claimed: "1200", want: "value"},
		{q: "EX-03 hôm nay được cấp ba nghìn tám rưỡi lít dầu đúng không?", claimed: "3850", want: "value"},
		{q: "Lu 02 hôm nay được cấp một nghìn hai lẻ lít dầu đúng không?", claimed: "1200", want: "value"}, // "lẻ" cuối dãy: xấp xỉ
		{q: "Lu 02 hôm nay được cấp mười hai trăm lít dầu đúng không?", claimed: "210", want: "value"},     // "<chục> trăm": nói tắt
		// số trần chỉ có đơn vị nhờ chữ "dầu" ("Dầu ERP 19-10") không phải lượng của riêng cụm; con số của cụm KHÁC đứng sau ("ERP, đội báo
		// 4000 L") không phải của cụm trước (soát khoá vòng 2, phát hiện 2 và 3)
		{q: "Dầu ERP 19-10, EX-03 hôm nay được cấp 3850 L đúng không?", claimed: "3850", want: "source"},
		{q: "ERP, đội báo 4000 L, EX-03 hôm nay được cấp 3850 L đúng không?", claimed: "3850", want: "source"},
		// con số "của riêng" cụm ngữ cảnh chỉ là lượng cùng chiều đo: mốc "tuần 43", giờ "8h" không biến cụm thành sự kiện riêng (F2)
		{q: "ERP 8h, EX-03 hôm nay được cấp 3900 L đúng không?", claimed: "3900", want: "source"},
		{q: "Đội báo ngày 19/10 tuần 43, EX-03 được cấp 4000 L đúng không?", claimed: "4000", want: "metric"},
		// KHOÁ (§9.6, soát vòng 4): cụm nguồn / chỉ số khác ở bất kỳ đâu trong ngữ cảnh, không kèm số riêng ⇒ "source" / "metric"
		{q: "Bao nhiêu ERP, EX-03 hôm nay được cấp 3900 L đúng không?", claimed: "3900", want: "source"},
		{q: "Dầu ERP. EX-03 hôm nay được cấp 3900 L đúng không?", claimed: "3900", want: "source"},
		{q: "Đội báo. EX-03 hôm nay được cấp 3850 L đúng không?", claimed: "3850", want: "metric"},
		{q: "Bao nhiêu định mức, vậy EX-03 hôm nay được cấp 3850 L đúng không?", claimed: "3850", want: "metric"},
		// bước khoá chỉ xét NGỮ CẢNH [0, lo): từ khoá chỉ số khác trong chính mệnh đề hỏi là phần dư, không phải đề mục
		{q: "EX-03 hôm nay đội báo được cấp 3850 L đúng không?", claimed: "3850", want: "residual"},
		// đề mục đầu câu (trước bước khoá do topicQualifier bắt, nay do bước khoá): nhãn "metric" cho chỉ số khác, "source" khi cụm vừa là
		// nguồn vừa là chỉ số; đề mục được có từ trung tính, mốc (kể cả "ngày" / "hôm" đứng trước mốc số), dấu ngăn nhiều byte; số do hai
		// nguồn thì không nguồn nào là "đúng nguồn"
		{q: "Đội báo, EX-03 hôm nay được cấp 3850 L đúng không?", claimed: "3850", want: "metric"},
		{q: "Hoá đơn, EX-03 hôm nay được cấp 3850 L đúng không?", claimed: "3850", want: "source"},
		{q: "ERP ơi, EX-03 hôm nay được cấp 3900 L đúng không?", claimed: "3900", want: "source"},
		{q: "Hôm nay, ERP, EX-03 được cấp 3900 L đúng không?", claimed: "3900", want: "source"},
		{q: "Tháng 10, ERP, EX-03 được cấp 3900 L đúng không?", claimed: "3900", want: "source"},
		{q: "Ngày 19/10, ERP, EX-03 được cấp 3900 L đúng không?", claimed: "3900", want: "source"},
		{q: "Ngày 19/10, đội báo, EX-03 được cấp 3850 L đúng không?", claimed: "3850", want: "metric"},
		{q: "ERP ngày 19/10, EX-03 được cấp 3900 L đúng không?", claimed: "3900", want: "source"},
		{q: "Hôm 19/10, ERP, EX-03 được cấp 3900 L đúng không?", claimed: "3900", want: "source"},
		{q: "“ERP”, EX-03 hôm nay được cấp 3900 L đúng không?", claimed: "3900", want: "source"},
		{q: "Phiếu xuất kho, EX-03 hôm nay được cấp 3850 L đúng không?", claimed: "3850", want: "source", used: []string{"warehouse_slip", "erp"}},
		// R5-4: đề mục vẫn "thuần" khi có thêm chữ mà phép kiểm chính tự giải thích — thuật ngữ của chính ý định ("dầu"), từ khoá của chính
		// ý định ("kho xuất"), tên đúng nguồn đã dùng ("phiếu xuất kho"); đề mục chỉ gồm những chữ đó thì không phải nguồn / chỉ số khác
		{q: "Dầu ERP, EX-03 hôm nay được cấp 3900 L đúng không?", claimed: "3900", want: "source"},
		{q: "Kho xuất, ERP, EX-03 hôm nay được cấp 3900 L đúng không?", claimed: "3900", want: "source"},
		{q: "Phiếu xuất kho, ERP, EX-03 hôm nay được cấp 3850 L đúng không?", claimed: "3850", want: "source"},
		{q: "Dầu, EX-03 hôm nay được cấp 3850 L đúng không?", claimed: "3850"},
		{q: "Kho xuất, EX-03 hôm nay được cấp 3850 L đúng không?", claimed: "3850"},
		// R5-5: bí danh "phiếu kho" (trùng từ khoá của chính ý định) là nguồn KHÁC khi số trả lời lấy từ ERP
		{q: "Phiếu kho, EX-03 hôm nay được cấp 3900 L đúng không?", claimed: "3900", want: "source", used: []string{"erp"}},
		// đề mục chỉ có mốc / từ trung tính vẫn phán quyết; từ khoá ý định yếu ở ngữ cảnh không được giải thích ("Tổng,")
		{q: "Hôm nay, EX-03 được cấp 3850 L đúng không?", claimed: "3850"},
		{q: "Cho hỏi, EX-03 hôm nay được cấp 3850 L đúng không?", claimed: "3850"},
		{q: "Tổng, EX-03 hôm nay được cấp 3850 L đúng không?", claimed: "3850", want: "residual"},
		// cụm nguồn / chỉ số khác ở ngữ cảnh đứng ngay trước con số; số ở câu trước không biến "đội báo" thành sự kiện riêng. Trước bước
		// khoá hai câu này ra "subject" (cụm là chủ gần nhất); nay bước khoá bắt trước với nhãn của cụm
		{q: "EX-03 hôm nay ERP, được cấp 3900 L đúng không?", claimed: "3900", want: "source"},
		{q: "ERP ghi 3900 L. EX-03 hôm nay đội báo, được cấp 3850 L đúng không?", claimed: "3850", want: "metric"},
		{q: "EX-03 ngày 19/10 được cấp 3850 L máy 03 đúng không?", claimed: "3850"},
	} {
		in := rt.Normalizer.Normalize(c.q, normalization.Options{Now: now})
		res, err := entityresolution.Resolve(ctx, dir, in)
		if err != nil {
			t.Fatal(err)
		}
		subject := c.subject
		if subject == "" {
			if len(res.Entities) == 0 {
				t.Fatalf("%q: không nhận ra thực thể nào", c.q)
			}
			subject = res.Entities[0].ID
		}
		subj, err := a.subjectOf(ctx, subject)
		if err != nil {
			t.Fatal(err)
		}
		used := c.used
		if used == nil {
			used = []string{"warehouse_slip"}
		}
		got := a.verdictGap(in, mentionsOf(res), contracts.Intent{Name: "fuel_issued"}, def,
			contracts.Value{Kind: contracts.ValueQuantity, Number: c.claimed, Unit: "L"}, subj, used)
		if got != c.want {
			t.Errorf("verdictGap(%q, %s) = %q, muốn %q", c.q, c.claimed, got, c.want)
		}
	}
}

// ADR-0009 Vòng 5: các vị từ chuỗi của luật bí danh / định ngữ. Ngày, giờ viết bằng số không phải ngắt mệnh đề; dấu câu thật thì là.
// Ghim nghĩa của từng hàm, không ghim chính sách cho các dạng còn chờ quyết ("EX-03 (dự án A)", "máy 03 (EX-03)").
func TestViTuDauCauVong5(t *testing.T) {
	for _, c := range []struct {
		s          string
		plain, brk bool
	}{
		{" hôm nay được cấp 3850 L ", true, false},
		{" ngày 19/10 được cấp 3850 L ", true, false},
		{" ngày 19/10/2026 ", true, false},
		{" ngày 19-10 ", true, false},
		{" lúc 8:30 ", true, false},
		{" 3.850 ", true, false},
		{" 0,5 ", true, false},
		{": dự án A ", false, false},
		{" - ", false, false},
		{" (", false, false},
		{", ", false, true},
		{". ", false, true},
		{"; ", false, true},
		{"\n", false, true},
		{"a, b", false, true},
		{"19/ 10", false, false},
		{"3850,", false, true},
		{",3850", false, true},
		{"a,5", false, true},
		{"3,a", false, true},
		{"3;850", false, false},
		{"? ", false, true},
		{"! ", false, true},
	} {
		if got := plainBetween(c.s); got != c.plain {
			t.Errorf("plainBetween(%q) = %v, muốn %v", c.s, got, c.plain)
		}
		if got := hasBreak(c.s); got != c.brk {
			t.Errorf("hasBreak(%q) = %v, muốn %v", c.s, got, c.brk)
		}
	}
	at := func(raw, sub string) (int, int) {
		i := strings.Index(raw, sub)
		return i, i + len(sub)
	}
	for _, c := range []struct {
		raw, first, second string
		want               bool
	}{
		{"EX-03 (dự án A) hôm nay", "EX-03", "dự án A", true},
		{"EX-03 [dự án A] hôm nay", "EX-03", "dự án A", true},
		{"EX-03 (dự án A hôm nay", "EX-03", "dự án A", false},
		{"EX-03 [dự án A) hôm nay", "EX-03", "dự án A", false},
		{"EX-03 - dự án A hôm nay", "EX-03", "dự án A", false},
		{"EX-03 ( dự án A ) hôm nay", "EX-03", "dự án A", true},
	} {
		_, fe := at(c.raw, c.first)
		ss, se := at(c.raw, c.second)
		if got := bareWrapped(c.raw, c.raw[fe:ss], se); got != c.want {
			t.Errorf("bareWrapped(%q, %q) = %v, muốn %v", c.raw, c.second, got, c.want)
		}
	}
	for _, c := range []struct {
		raw, sub string
		want     bool
	}{
		{"EX-03 (máy 03) hôm nay", "máy 03", true},
		{"EX-03 [ máy 03 ] hôm nay", "máy 03", true},
		{"EX-03 (máy 03] hôm nay", "máy 03", false},
		{"EX-03 máy 03 hôm nay", "máy 03", false},
	} {
		s, e := at(c.raw, c.sub)
		if got := wrapped(c.raw, s, e); got != c.want {
			t.Errorf("wrapped(%q, %q) = %v, muốn %v", c.raw, c.sub, got, c.want)
		}
	}
}

// subjectFacts.gap ở mức đơn vị: phạm vi hỏi [lo, len) nêu tên (cấp cha) mà không nêu chủ thể ⇒ "subject". Ở mức câu, numberOwned + E3
// (Vòng 5) đã bao trùm phép kiểm này nên chỉ test đơn vị bắt được đột biến tắt nó (n43 CG24).
func TestSubjectGapPhamViHoi(t *testing.T) {
	s := subjectFacts{id: "EX-03", ancestors: map[string]bool{"PRJ-A": true}}
	names := []mention{{start: 0, end: 5, ids: []string{"EX-03"}}, {start: 30, end: 37, ids: []string{"PRJ-A"}}}
	if got := s.gap(names, 20); got != "subject" {
		t.Errorf("gap(lo = 20) = %q, muốn \"subject\": phạm vi hỏi chỉ nêu cấp cha", got)
	}
	if got := s.gap(names, 0); got != "" {
		t.Errorf("gap(lo = 0) = %q, muốn \"\": phạm vi hỏi có nêu chủ thể", got)
	}
}
