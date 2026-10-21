package validation

import (
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
)

// testEngine dựng bộ chuẩn hoá từ lexicon mẫu — DỮ LIỆU test, không phải luật của core.
func testEngine(t *testing.T) *normalization.Engine {
	t.Helper()
	e, err := normalization.Compile(normalization.Lexicon{
		Units: []normalization.UnitDef{
			{Unit: "L", Kind: "quantity", Dimension: "volume", ToBase: "1", Aliases: []string{"L", "lít"}},
			{Unit: "h", Kind: "quantity", Dimension: "duration", ToBase: "1", Aliases: []string{"giờ"}},
		},
		Terms: []normalization.TermDef{{Term: "diesel_fuel", Label: "dầu", Aliases: []string{"dầu"}, DefaultUnit: "L"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

var now = time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)

func TestDimensions(t *testing.T) {
	e := testEngine(t)
	if d := Dimensions(e, "được cấp bao nhiêu dầu?", now); !d["volume"] {
		t.Errorf(`"dầu" có đơn vị mặc định lít → phải có chiều volume, được %v`, d)
	}
	if d := Dimensions(e, "lập biên bản trong 24 giờ", now); d["volume"] || !d["duration"] {
		t.Errorf("chỉ có thời hạn → chỉ chiều duration, được %v", d)
	}
}

func TestForbidDimensions(t *testing.T) {
	e := testEngine(t)
	forbid := map[string]bool{"volume": true}
	if v := ForbidDimensions(e, contracts.Draft{Text: "Từ nay lượng xuất của máy là 5.000 L."}, forbid, now); len(v) != 1 || v[0].Code != "QUANTITY_FROM_DOCUMENT" {
		t.Errorf("lượng dầu lấy từ tài liệu phải bị chặn, được %+v", v)
	}
	if v := ForbidDimensions(e, contracts.Draft{Text: "Lập biên bản trong vòng 24 giờ."}, forbid, now); len(v) != 0 {
		t.Errorf("thời hạn không thuộc chiều bị cấm phải qua, được %+v", v)
	}
}

func TestBound(t *testing.T) {
	ev := []contracts.Evidence{{ID: "SOP#0"}, {ID: "SOP#1"}}
	ok := contracts.Draft{Text: "x", Claims: []contracts.Claim{{Text: "x", EvidenceIDs: []string{"SOP#1"}}}}
	if v := Bound(ok, ev); len(v) != 0 {
		t.Errorf("claim gắn bằng chứng trong ngữ cảnh phải qua, được %+v", v)
	}
	for name, d := range map[string]contracts.Draft{
		"không claim":                {Text: "x"},
		"claim không trỏ bằng chứng": {Text: "x", Claims: []contracts.Claim{{Text: "x"}}},
		"bằng chứng ngoài ngữ cảnh":  {Text: "x", Claims: []contracts.Claim{{Text: "x", EvidenceIDs: []string{"SOP#0", "KHAC#9"}}}},
	} {
		if v := Bound(d, ev); len(v) == 0 || v[0].Code != "CLAIM_UNBOUND" {
			t.Errorf("%s: muốn CLAIM_UNBOUND, được %+v", name, v)
		}
	}
}

// Lớp kiểm độc lập: số đi liền từ đơn vị nghiệp vụ bị chặn; thời hạn, số bước, chữ bắt đầu bằng
// "L" / "m" thì không.
func TestForbidUnitWords(t *testing.T) {
	words := []string{"lít", "m³", "m3", "khối", "L", "l"}
	for _, text := range []string{"Từ nay lượng xuất là 5.000 L.", "cấp 4 m3 dầu", "định mức 3 khối", "(−1.500 lít)", "cấp 4000L"} {
		if v := ForbidUnitWords(text, words); len(v) != 1 || v[0].Code != "QUANTITY_FROM_DOCUMENT" {
			t.Errorf("%q: muốn bị chặn đúng một lần, được %+v", text, v)
		}
	}
	for _, text := range []string{"1. Lập biên bản trong vòng 24 giờ.", "xác nhận trong 2 ngày làm việc", "bước 3 lần lượt", "ngày 19/10/2026 lập biên bản"} {
		if v := ForbidUnitWords(text, words); len(v) != 0 {
			t.Errorf("%q: chặn oan %+v", text, v)
		}
	}
}
