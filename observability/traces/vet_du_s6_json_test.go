package traces_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// V8AS-03 (soát vòng 1 của V8a): ba đột biến của người soát sống sót vì chưa test nào khoá hai điều dưới đây.

// E22. S6 của lượt KHÔNG truy vấn nguồn (từ chối, hỏi lại — router kết thúc trước chặng thực thi; chuyển người vì nguồn ngang thẩm quyền thì đã truy vấn nên có span) giữ chữ "không truy vấn nguồn". Chữ "cần truy xuất nhưng lượt thoát sớm trước truy xuất (lý do ở S8)" chỉ dành cho route
// cần model (RAG, HYBRID) mà không có span truy xuất — nói nó ở route chưa bao giờ truy xuất là kể một lỗi không có. VS8-41 chỉ khoá chiều ngược lại (route RAG thoát sớm phải nói câu đó).
func TestVetS6ChuCuaLuotKhongTruyVanNguon(t *testing.T) {
	a := realAsk(t)
	for _, c := range []struct {
		name  string
		pr    contracts.Principal
		q     string
		route contracts.RouteKind
	}{
		{"từ chối vì ngoài phạm vi", person("site_engineer", "PRJ-B"), "EX-03 hôm nay được cấp bao nhiêu dầu?", contracts.RouteReject},
		{"hỏi lại vì bí danh mơ hồ", person("director", ""), "Máy 03 hôm nay được cấp bao nhiêu dầu?", contracts.RouteClarification},
	} {
		t.Run(c.name, func(t *testing.T) {
			turn := realTurn(t, a, c.pr, c.q)
			if turn.Plan.Route != c.route {
				t.Fatalf("tiền đề: route %s, muốn %s", turn.Plan.Route, c.route)
			}
			s6 := stepOf(t, traces.NewRecord(turn).Steps, "S6")
			want := "không có span execute / retrieve: route " + turn.Decision.Route + " không truy vấn nguồn"
			if s6.Status != traces.StatusSkipped || s6.Reason != want || strings.Contains(s6.Reason, "cần truy xuất") {
				t.Errorf("S6 của route không truy vấn nguồn:\n  muốn: %s\n  được: %s (%s)", want, s6.Reason, s6.Status)
			}
		})
	}
}

// E16a / E16b. Phần giải thích của S8 (Why.Generate) và S10 (Why.Respond) đi qua JSON của Decision — bản ghi mà tiến trình khác đọc lại — nguyên vẹn: đổi thẻ json của một trong hai
// thành "-" làm nó mất mà không test nào đỏ (các ô cũ có test vòng JSON riêng; hai ô mới của V8a thì chưa).
func TestVetWhyVongJSONGiuS8VaS10(t *testing.T) {
	turn := realTurn(t, realAsk(t), person("director", ""), "Quy trình xử lý chênh lệch nhiên liệu là gì?")
	w := turn.Decision.Why
	if len(w.Generate.Reasons) == 0 || len(w.Respond.Reasons) == 0 {
		t.Fatalf("tiền đề: lượt thật phải mang giải thích ở S8 và S10: %+v", w)
	}
	b, err := json.Marshal(turn.Decision)
	if err != nil {
		t.Fatal(err)
	}
	var back traces.Decision
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back.Why.Generate, w.Generate) {
		t.Errorf("Why.Generate (S8) mất hay đổi qua vòng JSON:\n  trước: %+v\n  sau:   %+v", w.Generate, back.Why.Generate)
	}
	if !reflect.DeepEqual(back.Why.Respond, w.Respond) {
		t.Errorf("Why.Respond (S10) mất hay đổi qua vòng JSON:\n  trước: %+v\n  sau:   %+v", w.Respond, back.Why.Respond)
	}
	if !reflect.DeepEqual(back.Why, w) {
		t.Errorf("Why đổi qua vòng JSON:\n  trước: %+v\n  sau:   %+v", w, back.Why)
	}
	var raw struct {
		Why map[string]json.RawMessage `json:"why"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"generate", "respond"} {
		if _, ok := raw.Why[key]; !ok {
			t.Errorf("JSON của Decision.Why thiếu khoá %q: %s", key, string(b))
		}
	}
}
