package commands

// A29 (c) ở mức DỮ LIỆU THÔ (HOP-DONG §4.4, §7.5, §10): vết của hai lượt PERMISSION_DENIED, một cho máy không tồn tại và một cho máy ngoài phạm vi, KHÔNG
// được phân biệt được: cùng kết quả (trừ trace_id), cùng chín bước với cùng trạng thái, quyết định, lý do, đầu ra và nhãn lỗi; chỉ phần đầu vào (khoá máy và
// command_id người gọi đã gõ) khác nhau. Lý do thật chỉ nằm ở reason_detail của audit. Đường lưu vết thật (traces.Record, eais trace, GET /v1/trace/{id}) là
// phần của K3: khi K3 gộp, ca A29 ở cmd/eais kiểm lại điều này trên vết đã lưu. Đột biến M24 nhắm test này.

import (
	"reflect"
	"strings"
	"testing"
)

func TestA29CVetPermissionDeniedKhongTachKhongTonTaiVaNgoaiPhamVi(t *testing.T) {
	w := newWorld(t)
	pm := user("p-3", "project_manager", "PRJ-A")
	notFound := w.run(pm, req(1, "EX-99", day1))   // máy không tồn tại
	outOfScope := w.run(pm, req(2, "EX-09", day1)) // máy có thật nhưng thuộc dự án PRJ-B
	wantFail(t, notFound, StatusFailed, ReasonPermissionDenied)
	wantFail(t, outOfScope, StatusFailed, ReasonPermissionDenied)

	a, b := notFound.Result, outOfScope.Result
	a.TraceID, b.TraceID = "", ""
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("kết quả hai lượt phải y hệt (trừ trace_id): %+v với %+v", a, b)
	}
	if notFound.Entity != "" || outOfScope.Entity != "" {
		t.Fatalf("vết không được nêu khoá máy nào là tồn tại: %q, %q", notFound.Entity, outOfScope.Entity)
	}
	if len(notFound.Steps) != len(outOfScope.Steps) {
		t.Fatalf("số bước khác nhau: %d và %d", len(notFound.Steps), len(outOfScope.Steps))
	}
	for i := range notFound.Steps {
		x, y := notFound.Steps[i], outOfScope.Steps[i]
		x.Input, y.Input = nil, nil // chỉ phần đã gõ (khoá máy, command_id) được phép khác
		if !reflect.DeepEqual(x, y) {
			t.Errorf("bước %s khác nhau ngoài đầu vào: %+v với %+v", x.Code, x, y)
		}
		for _, s := range append(append([]string{x.Decision, x.Reason, x.Error}, x.Output...), y.Reason) {
			for _, real := range []string{"ngoài phạm vi", "không có trong danh mục", "không tồn tại"} {
				if strings.Contains(strings.ToLower(s), real) {
					t.Errorf("bước %s lộ lý do thật %q trong %q (chỉ audit được giữ lý do thật)", x.Code, real, s)
				}
			}
		}
	}
}
