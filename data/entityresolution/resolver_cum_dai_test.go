package entityresolution

import (
	"context"
	"testing"
)

// Cụm dài nhất thắng (resolver.go: "máy xúc 03" thắng "máy 03" thắng "03"): khi một bí danh là TIỀN TỐ của bí danh dài hơn
// thì phải nhận bí danh dài. Đột biến đảo thứ tự khớp (n36 G2b) từng sống vì danh mục test không có cặp tiền tố như vậy.
func TestResolveCumDaiNhatThang(t *testing.T) {
	d := MemoryDirectory{
		{ID: "a", Type: "equipment", Key: "EX-03", Name: "Máy xúc EX-03", Aliases: []Alias{{Text: "máy 03"}}},
		{ID: "b", Type: "equipment", Key: "DZ-03", Name: "Máy ủi DZ-03", Aliases: []Alias{{Text: "máy 03"}, {Text: "máy 03 ủi"}}},
	}
	res, err := Resolve(context.Background(), d, in("máy 03 ủi hôm qua lấy bao nhiêu dầu"))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Entities) != 1 || res.Entities[0].Key != "DZ-03" || len(res.Ambiguous) != 0 {
		t.Errorf("muốn nhận cụm dài \"máy 03 ủi\" = DZ-03, không mơ hồ; được thực thể %+v, mơ hồ %+v", res.Entities, res.Ambiguous)
	}
}
