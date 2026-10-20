package evals

import (
	"strings"
	"testing"
)

// Bộ ca của hai tenant tham chiếu phải là dữ liệu đúng hợp đồng CONVENTIONS §5.
func TestBoCaTenantDungHopDong(t *testing.T) {
	for path, tenant := range map[string]string{
		"../tenants/construction/evals/cases.yaml": "abc-infra",
		"../tenants/sports/evals/cases.yaml":       "xyz-sports",
	} {
		cases, err := Load(path)
		if err != nil {
			t.Errorf("%v", err)
			continue
		}
		if len(cases) == 0 {
			t.Errorf("%s: không có ca nào", path)
		}
		for _, p := range Validate(cases, tenant) {
			t.Errorf("%s: %s", path, p)
		}
		t.Logf("%s: %d ca, theo category %v", path, len(cases), Categories(cases))
	}
}

// Cổng kiểm phải thật sự chặn: mỗi kiểu hỏng sinh đúng lỗi của nó.
func TestKiemCaChanCaHong(t *testing.T) {
	good := func(id string) Case {
		c := Case{}
		for _, k := range Keys {
			c[k] = nil
		}
		c["id"], c["tenant"], c["entry_type"], c["category"], c["severity"] = id, "abc-infra", "QUERY", "happy_path", "S1"
		c["input"], c["expected_route"] = "dầu 4k", "SQL_QUERY"
		return c
	}
	if p := Validate([]Case{good("CON-T-001")}, "abc-infra"); len(p) != 0 {
		t.Fatalf("ca đúng bị chặn oan: %v", p)
	}
	missing := good("CON-T-002")
	delete(missing, "must_not_do")
	unknown := good("CON-T-003")
	unknown["expected_answer"] = "4.000 L"
	route := good("CON-T-004")
	route["expected_route"] = "RAG_PLUS"
	tenant := good("CON-T-005")
	tenant["tenant"] = "xyz-sports"
	noInput := good("CON-T-006")
	noInput["input"] = nil
	problems := strings.Join(Validate([]Case{good("CON-T-001"), good("CON-T-001"), missing, unknown, route, tenant, noInput}, "abc-infra"), "\n")
	for _, want := range []string{"id trùng", "thiếu khoá must_not_do", "khoá lạ expected_answer", "expected_route lạ", "tenant \"xyz-sports\"", "không có input"} {
		if !strings.Contains(problems, want) {
			t.Errorf("không bắt được %q; lỗi thu được:\n%s", want, problems)
		}
	}
}
