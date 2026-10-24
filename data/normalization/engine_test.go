package normalization

import (
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"golang.org/x/text/unicode/norm"
)

// testLexicon là lexicon mẫu cho test của bộ máy. Đây là DỮ LIỆU test, không phải luật
// của core: core không biết "dầu" là gì, nó chỉ đọc lexicon được đưa vào.
var testLexicon = Lexicon{
	Units: []UnitDef{
		{Unit: "L", Kind: "quantity", Dimension: "volume", ToBase: "1", Aliases: []string{"L", "l", "lít", "lit", "litre", "liter"}},
		{Unit: "m3", Kind: "quantity", Dimension: "volume", ToBase: "1000", Aliases: []string{"m3", "m³", "khối"}},
		{Unit: "m", Kind: "quantity", Dimension: "length", ToBase: "1", Aliases: []string{"m", "mét", "met"}},
		{Unit: "km", Kind: "quantity", Dimension: "length", ToBase: "1000", Aliases: []string{"km", "cây số"}},
		{Unit: "h", Kind: "quantity", Dimension: "duration", ToBase: "1", Aliases: []string{"h", "giờ", "tiếng"}},
		{Unit: "person", Kind: "quantity", Dimension: "headcount", Aliases: []string{"người", "công nhân"}},
		{Unit: "%", Kind: "quantity", Dimension: "ratio", Aliases: []string{"%", "phần trăm"}},
		{Unit: "VND", Kind: "money", Dimension: "currency", Aliases: []string{"đ", "đồng", "vnđ", "vnd"}},
	},
	Multipliers:   []MultiplierDef{{Factor: "1000", Aliases: []string{"k"}}, {Factor: "1000000", Aliases: []string{"tr", "củ"}}},
	ApproxMarkers: []string{"khoảng", "tầm", "cỡ", "gần", "chừng"},
	Terms: []TermDef{
		{Term: "diesel_fuel", Label: "dầu", Aliases: []string{"dầu", "dầu diesel"}, DefaultUnit: "L"},
		{Term: "work_quantity", Label: "khối lượng", Aliases: []string{"khối lượng"}},
		{Term: "workforce", Label: "nhân công", Aliases: []string{"nhân công"}, DefaultUnit: "person"},
		{Term: "project", Label: "dự án", Aliases: []string{"dự án"}},
		{Term: "price", Label: "giá", Aliases: []string{"giá"}, DefaultUnit: "VND"},
	},
	Abbreviations: []AbbrDef{
		{Abbr: "DA", Meanings: []AbbrMeaning{
			{Term: "diesel_fuel", WhenNextDimension: "volume", Scopes: []string{"fuel_report"}},
			{Term: "project", Default: true},
		}},
		{Abbr: "KL", Meanings: []AbbrMeaning{{Term: "work_quantity"}}},
		{Abbr: "NC", Meanings: []AbbrMeaning{{Term: "workforce"}}},
	},
	Codes: []CodeDef{
		{Type: "tire_size", Pattern: `(?P<w>\d{3})\s*[/ ]\s*(?P<a>\d{2})\s*[rR]\s*(?P<r>\d{2})`, Format: "{w}/{a}R{r}"},
		{Type: "issue_slip", Pattern: `(?i)PX-?(?P<n>\d{3,6})`, Format: "PX-{n}", Upper: true},
	},
	LabelNouns: []string{"khối", "lít", "l"},
	CountNouns: []string{"máy", "xe", "chuyến", "thiết bị", "dự án"},
}

// now là "hôm nay" của mọi vector: 19/10/2026, giờ Việt Nam.
var now = time.Date(2026, 10, 19, 9, 0, 0, 0, time.FixedZone("ICT", 7*3600))

type wantSpan struct {
	kind   contracts.ValueKind
	number string
	unit   string
	text   string
	approx bool
}

func mustEngine(t *testing.T) *Engine {
	t.Helper()
	e, err := Compile(testLexicon)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// TestNormalizeVectors là bộ vector NRM-V: mỗi dòng là một cách người Việt thật sự gõ.
// Mỗi vector khai đủ span mong đợi theo thứ tự; span thừa hay thiếu đều là đỏ.
func TestNormalizeVectors(t *testing.T) {
	e := mustEngine(t)
	q := contracts.ValueQuantity
	cases := []struct {
		id, in, scope string
		want          []wantSpan
		ambiguities   int
	}{
		{"NRM-V-001", "dầu 4k", "", []wantSpan{{kind: contracts.ValueTerm, text: "diesel_fuel"}, {kind: q, number: "4000", unit: "L"}}, 0},
		{"NRM-V-002", "DA 4.000L", "", []wantSpan{{kind: contracts.ValueTerm, text: "diesel_fuel"}, {kind: q, number: "4000", unit: "L"}}, 0},
		// Không đoán: "DA" trước số trần có thể là dầu hoặc dự án.
		{"NRM-V-003", "DA: 4.000", "", []wantSpan{{kind: contracts.ValueNumber, number: "4000"}}, 1},
		{"NRM-V-004", "DA: 4.000", "fuel_report", []wantSpan{{kind: contracts.ValueTerm, text: "diesel_fuel"}, {kind: q, number: "4000", unit: "L"}}, 0},
		{"NRM-V-005", "cấp thêm 4 khối dầu", "", []wantSpan{{kind: q, number: "4", unit: "m3"}, {kind: contracts.ValueTerm, text: "diesel_fuel"}}, 0},
		{"NRM-V-006", "máy 03 lấy 2k sáng + 2k chiều", "", []wantSpan{{kind: contracts.ValueNumber, number: "3"}, {kind: contracts.ValueNumber, number: "2000"}, {kind: contracts.ValueNumber, number: "2000"}}, 0},
		{"NRM-V-007", "4,000 L", "", []wantSpan{{kind: q, number: "4000", unit: "L"}}, 0},
		{"NRM-V-008", "bốn ngàn lít", "", []wantSpan{{kind: q, number: "4000", unit: "L"}}, 0},
		{"NRM-V-009", "khoảng 4k dầu", "", []wantSpan{{kind: q, number: "4000", unit: "L", approx: true}, {kind: contracts.ValueTerm, text: "diesel_fuel"}}, 0},
		{"NRM-V-010", "phiếu kho 3.850L nhưng đội trưởng báo 4k", "", []wantSpan{{kind: q, number: "3850", unit: "L"}, {kind: contracts.ValueNumber, number: "4000"}}, 0},
		{"NRM-V-011", "KL 0.35km", "", []wantSpan{{kind: contracts.ValueTerm, text: "work_quantity"}, {kind: q, number: "0.35", unit: "km"}}, 0},
		{"NRM-V-012", "ba trăm năm chục mét", "", []wantSpan{{kind: q, number: "350", unit: "m"}}, 0},
		{"NRM-V-013", "NC 22", "", []wantSpan{{kind: contracts.ValueTerm, text: "workforce"}, {kind: q, number: "22", unit: "person"}}, 0},
		{"NRM-V-014", "máy xúc chạy 7h", "", []wantSpan{{kind: q, number: "7", unit: "h"}}, 0},
		{"NRM-V-015", "mưa từ 14h", "", []wantSpan{{kind: contracts.ValueTime, text: "14:00"}}, 0},
		{"NRM-V-016", "2h chiều", "", []wantSpan{{kind: contracts.ValueTime, text: "14:00"}}, 0},
		{"NRM-V-017", "giá 1tr6", "", []wantSpan{{kind: contracts.ValueTerm, text: "price"}, {kind: contracts.ValueMoney, number: "1600000", unit: "VND"}}, 0},
		{"NRM-V-018", "1.602.000đ", "", []wantSpan{{kind: contracts.ValueMoney, number: "1602000", unit: "VND"}}, 0},
		{"NRM-V-019", "1tr499", "", []wantSpan{{kind: contracts.ValueNumber, number: "1499000"}}, 0},
		{"NRM-V-020", "500k", "", []wantSpan{{kind: contracts.ValueNumber, number: "500000"}}, 0},
		{"NRM-V-021", "hai triệu rưỡi", "", []wantSpan{{kind: contracts.ValueNumber, number: "2500000"}}, 0},
		{"NRM-V-022", "tháng này", "", []wantSpan{{kind: contracts.ValuePeriod, text: "2026-10-01/2026-10-31"}}, 0},
		{"NRM-V-023", "hôm qua", "", []wantSpan{{kind: contracts.ValueDate, text: "2026-10-18"}}, 0},
		{"NRM-V-024", "16/9", "", []wantSpan{{kind: contracts.ValueDate, text: "2026-09-16"}}, 0},
		{"NRM-V-025", "16/9/2026", "", []wantSpan{{kind: contracts.ValueDate, text: "2026-09-16"}}, 0},
		{"NRM-V-026", "2025-11-28", "", []wantSpan{{kind: contracts.ValueDate, text: "2025-11-28"}}, 0},
		{"NRM-V-027", "205/55R16", "", []wantSpan{{kind: contracts.ValueCode, text: "205/55R16"}}, 0},
		{"NRM-V-028", "205 55 r16", "", []wantSpan{{kind: contracts.ValueCode, text: "205/55R16"}}, 0},
		// "đầu" có dấu khác "dầu": không được hiểu thành nhiên liệu.
		{"NRM-V-029", "đầu tháng 9", "", []wantSpan{{kind: contracts.ValuePeriod, text: "2026-09-01/2026-09-30"}}, 0},
		// Gõ không dấu vẫn khớp thuật ngữ.
		{"NRM-V-030", "dau 4k", "", []wantSpan{{kind: contracts.ValueTerm, text: "diesel_fuel"}, {kind: q, number: "4000", unit: "L"}}, 0},
		// "năm" một mình là "năm (year)", không phải số 5.
		{"NRM-V-031", "năm 2026", "", []wantSpan{{kind: contracts.ValueNumber, number: "2026"}}, 0},
		{"NRM-V-032", "80%", "", []wantSpan{{kind: q, number: "80", unit: "%"}}, 0},
		{"NRM-V-033", norm.NFD.String("dầu 4k"), "", []wantSpan{{kind: contracts.ValueTerm, text: "diesel_fuel"}, {kind: q, number: "4000", unit: "L"}}, 0},
		// Tháng chưa tới năm nay → tháng đó năm trước (báo cáo hỏi về quá khứ).
		{"NRM-V-034", "tháng 11", "", []wantSpan{{kind: contracts.ValuePeriod, text: "2025-11-01/2025-11-30"}}, 0},
		{"NRM-V-035", "máy chạy 7h30", "", []wantSpan{{kind: q, number: "7.5", unit: "h"}}, 0},
		{"NRM-V-036", "14:30", "", []wantSpan{{kind: contracts.ValueTime, text: "14:30"}}, 0},
		{"NRM-V-037", "phiếu PX0912", "", []wantSpan{{kind: contracts.ValueCode, text: "PX-0912"}}, 0},
		{"NRM-V-038", "0,35 km", "", []wantSpan{{kind: q, number: "0.35", unit: "km"}}, 0},
		{"NRM-V-039", "4.5 km", "", []wantSpan{{kind: q, number: "4.5", unit: "km"}}, 0},
		{"NRM-V-040", "mười hai người", "", []wantSpan{{kind: q, number: "12", unit: "person"}}, 0},
		{"NRM-V-041", "một trăm linh năm mét", "", []wantSpan{{kind: q, number: "105", unit: "m"}}, 0},
		{"NRM-V-042", "DA cầu Bến Lức", "", []wantSpan{{kind: contracts.ValueTerm, text: "project"}}, 0},
		// 31/2 không phải ngày: không được sinh span ngày.
		{"NRM-V-043", "31/2", "", []wantSpan{{kind: contracts.ValueNumber, number: "31"}, {kind: contracts.ValueNumber, number: "2"}}, 0},
		{"NRM-V-044", "4000L.", "", []wantSpan{{kind: q, number: "4000", unit: "L"}}, 0},
		{"NRM-V-045", "1.602,5 m", "", []wantSpan{{kind: q, number: "1602.5", unit: "m"}}, 0},
		{"NRM-V-046", "~3.900 lít", "", []wantSpan{{kind: q, number: "3900", unit: "L", approx: true}}, 0},
		// Số âm (hiệu của phép tính) đọc lại đúng dấu; gạch nối giữa hai số không phải dấu âm.
		{"NRM-V-047", "chênh -150 L", "", []wantSpan{{kind: q, number: "-150", unit: "L"}}, 0},
		{"NRM-V-048", "(−1.500 lít)", "", []wantSpan{{kind: q, number: "-1500", unit: "L"}}, 0},
		{"NRM-V-049", "18-20 lít", "", []wantSpan{{kind: contracts.ValueNumber, number: "18"}, {kind: q, number: "20", unit: "L"}}, 0},
	}
	for _, c := range cases {
		t.Run(c.id, func(t *testing.T) {
			got := e.Normalize(c.in, Options{Now: now, Scope: c.scope})
			if len(got.Ambiguities) != c.ambiguities {
				t.Errorf("%q: %d ambiguity, muốn %d: %+v", c.in, len(got.Ambiguities), c.ambiguities, got.Ambiguities)
			}
			if len(got.Spans) != len(c.want) {
				t.Fatalf("%q: %d span, muốn %d:\n%s", c.in, len(got.Spans), len(c.want), dump(got.Spans))
			}
			for k, w := range c.want {
				s := got.Spans[k]
				if s.Kind != w.kind || s.Value.Number != w.number || s.Value.Unit != w.unit || s.Value.Text != w.text || s.Value.Approx != w.approx {
					t.Errorf("%q span %d = %s, muốn %+v", c.in, k, dump([]contracts.Span{s}), w)
				}
			}
		})
	}
}

func dump(spans []contracts.Span) string {
	var b strings.Builder
	for _, s := range spans {
		b.WriteString("  " + string(s.Kind) + " raw=" + s.Raw + " number=" + s.Value.Number + " unit=" + s.Value.Unit +
			" text=" + s.Value.Text + " rule=" + s.Rule + "\n")
	}
	return b.String()
}

func TestCanonicalTextThayDangChuan(t *testing.T) {
	e := mustEngine(t)
	cases := map[string]string{
		"dầu 4k":     "dầu 4000 L",
		"DA 4.000L":  "dầu 4000 L",
		"205 55 r16": "205/55R16",
		"NC 22":      "nhân công 22 person",
	}
	for in, want := range cases {
		if got := e.Normalize(in, Options{Now: now}).Canonical; got != want {
			t.Errorf("Canonical(%q) = %q, muốn %q", in, got, want)
		}
	}
}

func TestDonViSuyLuanCoDoTinCayThap(t *testing.T) {
	e := mustEngine(t)
	got := e.Normalize("dầu 4k", Options{Now: now})
	q := got.SpansOf(contracts.ValueQuantity)
	if len(q) != 1 || q[0].Confidence >= 1 {
		t.Fatalf("đơn vị suy từ thuật ngữ phải có độ tin cậy < 1: %+v", q)
	}
	direct := e.Normalize("4000 lít", Options{Now: now}).SpansOf(contracts.ValueQuantity)
	if len(direct) != 1 || direct[0].Confidence != 1 {
		t.Fatalf("đơn vị viết rõ phải có độ tin cậy 1: %+v", direct)
	}
}

func TestConvertCungChieu(t *testing.T) {
	e := mustEngine(t)
	v, err := e.Convert(contracts.Value{Kind: contracts.ValueQuantity, Number: "4", Unit: "m3"}, "L")
	if err != nil || v.Number != "4000" || v.Unit != "L" {
		t.Fatalf("4 m3 → L = %+v, %v", v, err)
	}
	v, err = e.Convert(contracts.Value{Kind: contracts.ValueQuantity, Number: "350", Unit: "m"}, "km")
	if err != nil || v.Number != "0.35" {
		t.Fatalf("350 m → km = %+v, %v", v, err)
	}
	if _, err := e.Convert(contracts.Value{Kind: contracts.ValueQuantity, Number: "4", Unit: "L"}, "km"); err == nil {
		t.Fatal("đổi thể tích sang độ dài phải lỗi")
	}
}

func TestCompileChanLexiconHong(t *testing.T) {
	cases := map[string]Lexicon{
		"alias trùng giữa hai đơn vị": {Units: []UnitDef{
			{Unit: "L", Kind: "quantity", Dimension: "volume", Aliases: []string{"l"}},
			{Unit: "lb", Kind: "quantity", Dimension: "mass", Aliases: []string{"l"}},
		}},
		"default_unit chưa khai báo": {Terms: []TermDef{{Term: "fuel", Aliases: []string{"dầu"}, DefaultUnit: "L"}}},
		"regex hỏng":                 {Codes: []CodeDef{{Type: "x", Pattern: `(`}}},
		"format dùng nhóm không có":  {Codes: []CodeDef{{Type: "x", Pattern: `\d+`, Format: "{n}"}}},
		"hai nghĩa mặc định": {
			Terms:         []TermDef{{Term: "a", Aliases: []string{"aa"}}, {Term: "b", Aliases: []string{"bb"}}},
			Abbreviations: []AbbrDef{{Abbr: "X", Meanings: []AbbrMeaning{{Term: "a", Default: true}, {Term: "b", Default: true}}}},
		},
		"kind sai": {Units: []UnitDef{{Unit: "L", Kind: "volume", Dimension: "volume", Aliases: []string{"l"}}}},
	}
	for name, lex := range cases {
		if _, err := Compile(lex); err == nil {
			t.Errorf("%s: Compile phải lỗi", name)
		}
	}
	if _, err := Compile(testLexicon); err != nil {
		t.Fatalf("lexicon hợp lệ bị chặn oan: %v", err)
	}
}

// Dấu ước lượng dạng ký hiệu chỉ được nhận khi PACK khai nó ở approx_markers (ngoài "~", nhận như trước): lexicon mẫu không có "≈" thì
// "≈ 2,6 m3" không phải ước lượng; thêm "≈" vào lexicon thì là ước lượng, ở cả ô lẫn chữ tự do.
func TestDauKyHieuChiKhiPackKhai(t *testing.T) {
	lex := testLexicon
	with := lex
	with.ApproxMarkers = append(append([]string{}, lex.ApproxMarkers...), "≈")
	// khai một ký hiệu KHÁC có chứa "≈" (soát xx, XXS-01): so khớp phải là bằng đúng, không phải chuỗi con hay tiền tố
	other := lex
	other.ApproxMarkers = append(append([]string{}, lex.ApproxMarkers...), "≈≈")
	for _, c := range []struct {
		name string
		lex  Lexicon
		want bool
	}{{"không khai", lex, false}, {"có khai", with, true}, {"khai ký hiệu khác", other, false}} {
		e, err := Compile(c.lex)
		if err != nil {
			t.Fatal(err)
		}
		for _, cell := range []bool{true, false} {
			in := e.Normalize("≈ 2,6 m3", Options{Now: now, Cell: cell})
			if len(in.Spans) != 1 || in.Spans[0].Value.Approx != c.want {
				t.Errorf("%s (ô=%v): spans %+v, muốn đúng một lượng có approx = %v", c.name, cell, in.Spans, c.want)
			}
		}
	}
}
