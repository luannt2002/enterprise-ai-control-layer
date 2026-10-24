package commands

import "testing"

// Q-P1-10 / Q11 = A: capability của lệnh, metrics.read và phạm vi thực thể phải do CÙNG một vai thoả — người nhiều vai không ghép quyền
// của vai này với phạm vi của vai kia. Mỗi ca: người hai vai bị từ chối ở C4 (không phiếu, một dòng audit REFUSED với đúng lý do thật), còn
// đối chứng — cùng hai vai, một vai tự thoả cả ba — tạo được phiếu.
func TestQ11QuyenVaPhamViCungMotVai(t *testing.T) {
	for _, c := range []struct {
		name  string
		extra []string // capability của vai thêm "rong" (scope all)
		role  string   // vai thứ nhất (scope attribute, PRJ-A)
		key   string   // máy bị từ chối
		real  string   // lý do thật ở audit
		ok    string   // máy mà một vai tự thoả cả ba
	}{
		{"vai có quyền phạm vi hẹp + vai phạm vi rộng chỉ đọc số liệu", []string{"metrics.read"}, "project_manager", "EX-09", "thực thể ngoài phạm vi của người gọi", "EX-03"},
		{"vai có quyền phạm vi hẹp + vai phạm vi rộng không capability", nil, "project_manager", "EX-09", "thực thể ngoài phạm vi của người gọi", "EX-03"},
		{"vai chỉ có tickets.create + vai chỉ có metrics.read", []string{"tickets.create"}, "site_engineer", "EX-03", "thiếu capability metrics.read trên cùng vai có tickets.create", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld(t, func(d *Deps) { d.Policy.Roles["rong"] = routerRole(c.extra...) })
			pr := user("p-hai-vai", c.role, "PRJ-A")
			pr.Roles = append(pr.Roles, "rong")
			o := w.run(pr, req(60, c.key, day1))
			wantFail(t, o, StatusFailed, ReasonPermissionDenied)
			assertSteps(t, o, "C4")
			w.check(tenantA, want{refused: 1})
			if got := w.store.audits(tenantA, AuditRefused, ReasonPermissionDenied, "p-hai-vai"); len(got) != 1 || got[0].ReasonDetail != c.real {
				t.Fatalf("lý do thật ở audit: muốn %q, được %+v", c.real, got)
			}
			if c.ok != "" {
				wantDone(t, w.run(pr, req(61, c.ok, day1)), false)
			}
		})
	}
}
