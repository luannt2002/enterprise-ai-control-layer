package validation

import (
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
)

func litres(number string) contracts.ContextPackage {
	return contracts.ContextPackage{Evidence: []contracts.Evidence{{ID: "calc:issued_fuel",
		Values: []contracts.Value{{Kind: contracts.ValueQuantity, Number: number, Unit: "L"}}}}}
}

// S2-3: ignore là chuỗi bỏ đi trước khi đếm số. Cắt "0 L" khỏi giữa "3.850 L" để lại "3.85" — một số
// chưa từng được kiểm — nên câu "Không. … 3.850 L, không phải 0 L." ra SYSTEM_FAILURE oan. Chỉ lần
// xuất hiện TRỌN VẸN (không dính vào một số hay một từ lớn hơn) mới được bỏ đi.
func TestFidelityIgnoreKhongCatGiuaMotSo(t *testing.T) {
	e := testEngine(t)
	for _, c := range []struct{ name, text, claimed, truth string }{
		{"0 L nằm cuối 3.850 L", "Không. Dầu kho xuất của máy xúc hôm nay là 3.850 L, không phải 0 L.", "0 L", "3850"},
		{"50 L nằm cuối 3.850 L", "Không. Dầu kho xuất của máy xúc hôm nay là 3.850 L, không phải 50 L.", "50 L", "3850"},
		{"850 L nằm sau dấu nghìn", "Không. Dầu kho xuất của máy xúc hôm nay là 3.850 L, không phải 850 L.", "850 L", "3850"},
		{"0 L nằm cuối 4.000 L", "Không. Dầu đội báo của máy xúc hôm nay là 4.000 L, không phải 0 L.", "0 L", "4000"},
		{"200 L nằm cuối 1.200 L", "Không. Dầu kho xuất của lu rung hôm nay là 1.200 L, không phải 200 L.", "200 L", "1200"},
		{"5 L nằm sau dấu phẩy thập phân", "Không. Dầu hoá đơn hôm nay là 0,5 L, không phải 5 L.", "5 L", "0.5"},
	} {
		if v := Fidelity(e, contracts.Draft{Text: c.text}, litres(c.truth), []string{c.claimed}, now); len(v) != 0 {
			t.Errorf("%s: số của người hỏi %q là phần ignore, số có thẩm quyền %s phải còn nguyên: %+v", c.name, c.claimed, c.truth, v)
		}
		// Không ignore thì số người hỏi không có bằng chứng → phải bị bắt (chứng tỏ ignore có tác dụng).
		if v := Fidelity(e, contracts.Draft{Text: c.text}, litres(c.truth), nil, now); len(v) == 0 {
			t.Errorf("%s: bỏ ignore mà số %q không có bằng chứng vẫn qua", c.name, c.claimed)
		}
	}
}

// Chiều ngược của S2-3 (fail-open): cắt giữa số làm MỘT SỐ KHÔNG CÓ BẰNG CHỨNG thành số có bằng
// chứng. "30 L" bị cắt "0 L" còn "3" — bằng chứng có 3 L nên số 30 L lọt qua.
func TestFidelityIgnoreKhongLamSoLaThanhSoCoBangChung(t *testing.T) {
	e := testEngine(t)
	v := Fidelity(e, contracts.Draft{Text: "Chênh 30 L, người hỏi nói 0 L."}, litres("3"), []string{"0 L"}, now)
	if len(v) != 1 || v[0].Code != "number_without_evidence" {
		t.Errorf("30 L không có bằng chứng phải bị bắt đúng một lần, được %+v", v)
	}
}

// Ignore vẫn phải bỏ được các lần xuất hiện trọn vẹn (nhiều lần, sát dấu câu, tên có chữ số).
func TestFidelityIgnoreBoLanTronVen(t *testing.T) {
	e := testEngine(t)
	text := "Máy xúc EX-03 (EX-03): 3.850 L; người hỏi 0 L, 0 L."
	if v := Fidelity(e, contracts.Draft{Text: text}, litres("3850"), []string{"Máy xúc EX-03", "EX-03", "0 L"}, now); len(v) != 0 {
		t.Errorf("tên thực thể và số của người hỏi đều là ignore, không được chặn: %+v", v)
	}
	// "0 L" dính chữ ("0 Lập") không phải số của người hỏi: không được bỏ đi.
	if v := Fidelity(e, contracts.Draft{Text: "3.850 L. Mục 7 Lập biên bản."}, litres("3850"), []string{"7 L"}, now); len(v) != 1 {
		t.Errorf("\"7 Lập\" không phải \"7 L\": số 7 phải còn bị kiểm, được %+v", v)
	}
}

// Dấu âm ngay trước chữ số làm "50 L" thành một phần của "-50 L": không được cắt, nên số âm vẫn bị kiểm.
func TestFidelityIgnoreKhongCatDauAm(t *testing.T) {
	e := testEngine(t)
	v := Fidelity(e, contracts.Draft{Text: "Chênh -50 L, người hỏi 50 L."}, litres("50"), []string{"50 L"}, now)
	if len(v) != 1 || v[0].Code != "number_without_evidence" || !strings.Contains(v[0].Detail, "-50") {
		t.Errorf("-50 L không có bằng chứng (bằng chứng chỉ có +50 L) phải bị bắt đúng một lần, được %+v", v)
	}
}

// Chiều xuôi của dấu nghìn: "3" đứng một mình thì bỏ, còn "3" đầu "3.850" (dấu chấm còn chữ số nối tiếp) thì giữ.
func TestFidelityIgnoreKhongCatDauSoCoDauNghin(t *testing.T) {
	e := testEngine(t)
	if v := Fidelity(e, contracts.Draft{Text: "Dầu hôm nay 3.850 L; mục 3."}, litres("3850"), []string{"3"}, now); len(v) != 0 {
		t.Errorf("\"3\" đứng một mình là ignore, \"3.850\" phải còn nguyên: %+v", v)
	}
}
