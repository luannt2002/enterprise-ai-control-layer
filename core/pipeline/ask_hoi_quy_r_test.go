package pipeline

import (
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/data/normalization"
)

// Test đơn vị cho các hàm thuần của luồng hỏi sau các HỒI QUY R-01, R-02, R-10 (bản sửa 6f23ff3): đầu–cuối nằm ở tenants/construction/pipeline_hoi_quy_r_test.go.

// R-10 — dấu chấm lửng "..." giữa câu không phải hết câu; một dấu chấm đơn theo sau là khoảng trắng vẫn là hết câu.
func TestSentenceStartDauChamLung(t *testing.T) {
	for _, c := range []struct {
		text string
		at   int // vị trí tính đầu câu cho (cuối chuỗi nếu < 0)
		want int
	}{
		{"Hôm nay EX-03 được cấp 3850 L ... đúng không?", -1, 0},
		{"Hôm nay EX-03 được cấp 3850 L... đúng không?", -1, 0},
		{"Hôm nay EX-03 được cấp 3850 L .... đúng không?", -1, 0},
		{"Hôm nay EX-03 được cấp 3850 L. Đúng không?", -1, len("Hôm nay EX-03 được cấp 3850 L.")},
		{"Phiếu 912. EX-03 được cấp 3.850 L, đúng không?", -1, len("Phiếu 912.")},
		{"A? B! C", -1, len("A? B!")},
	} {
		at := c.at
		if at < 0 {
			at = len(c.text)
		}
		if got := sentenceStart(c.text, at); got != c.want {
			t.Errorf("sentenceStart(%q) = %d, muốn %d", c.text, got, c.want)
		}
	}
	if !ellipsisAt("a ... b", 4) || !ellipsisAt("a ... b", 2) || ellipsisAt("a. b", 1) || ellipsisAt("3.850", 1) {
		t.Error("ellipsisAt: chấm trong '...' là dấu chấm lửng; chấm đơn và chấm nghìn thì không")
	}
}

// R-02 — từ nối hai con số của một lời xác nhận: dải, lựa chọn, liệt kê, phủ định vế; mọi từ khác (nhãn, danh từ, dấu câu) không phải từ nối.
func TestClaimConnector(t *testing.T) {
	for gap, want := range map[string]bool{
		" - ": true, "-": true, " – ": true, " đến ": true, " tới ": true, " hay ": true, " HOẶC ": true, " hay là ": true, " và ": true,
		" chứ không phải ": true, " chứ  không   phải ": true, " không phải ": true, " (hoặc ": true, " den ": true, " chu khong phai ": true, " / ": true, " so với ": true, " thay vì ": true, " hoặc là ": true,
		"": false, " ": false, " cho ": false, ", ": false, " L hay ": false, " đúng không, hay ": false, " phiếu ": false, " ca ": false, ". ": false,
	} {
		if got := claimConnector(gap); got != want {
			t.Errorf("claimConnector(%q) = %v, muốn %v", gap, got, want)
		}
	}
}

// R-02 — mệnh đề hỏi mở đầu bằng từ nối LỰA CHỌN / PHỦ ĐỊNH ("…3850 L, hay 4000 L đúng không?") nối tiếp mệnh đề liền trước; "vậy", "và" mở mệnh đề mới.
func TestContrastLeadingVaPrevClauseStart(t *testing.T) {
	for clause, want := range map[string]bool{
		" hay 4000 L đúng không": true, " hoặc 4000": true, "chứ không phải 4000 L": true, " hay là 4k": true, " không phải 4000": true,
		" vậy tổng báo 4k đúng không": false, " và 4000 L": false, " còn 4000 thì sao": false, " hayy 4000": false, "": false,
		// từ nối chỉ tính khi MỞ ĐẦU mệnh đề, không phải khi nằm giữa nó
		" vậy 4000 hay 3850 L đúng không": false, " vậy 4000 hoặc 3850 L": false, " em nói không phải 4000": false,
		" không phải 4000 L đúng không": true, " chứ không phải": true,
	} {
		if got := contrastLeading(clause); got != want {
			t.Errorf("contrastLeading(%q) = %v, muốn %v", clause, got, want)
		}
	}
	text := "Đội báo 2k, máy chạy 3 ca. EX-03 được cấp 3850 L, hay 4000 L đúng không?"
	lo := len("Đội báo 2k, máy chạy 3 ca. EX-03 được cấp 3850 L,")
	if got := prevClauseStart(text, lo); got != len("Đội báo 2k, máy chạy 3 ca.") {
		t.Errorf("prevClauseStart = %d, muốn đầu mệnh đề \" EX-03 được cấp 3850 L\" (%d)", got, len("Đội báo 2k, máy chạy 3 ca."))
	}
	if got := prevClauseStart("EX-03 được cấp 3850 L, hay 4000 L", len("EX-03 được cấp 3850 L,")); got != 0 {
		t.Errorf("prevClauseStart ở mệnh đề đầu = %d, muốn 0", got)
	}
	// dấu chấm trong số ("3.850") không cắt mệnh đề
	if got := prevClauseStart("EX-03 được cấp 3.850 L, hay 4000 L", len("EX-03 được cấp 3.850 L,")); got != 0 {
		t.Errorf("prevClauseStart qua số có dấu chấm = %d, muốn 0", got)
	}
}

// R-02g — lượng viết kèm MẪU SỐ ("5.000 L/ngày", "5 m3/ca") là tốc độ / định mức, không phải số lượng đã cấp.
func TestRateAfter(t *testing.T) {
	for _, c := range []struct {
		text string
		end  int
		want bool
	}{
		{"5.000 L/ngày nên", len("5.000 L"), true}, {"5 m3/ca", len("5 m3"), true}, {"3850 L đúng", len("3850 L"), false}, {"3850 L/", len("3850 L"), false},
		{"3850 L/3", len("3850 L"), false}, {"3850 L", len("3850 L"), false}, {"3850 L", 100, false}, {"3850 L", -1, false},
	} {
		if got := rateAfter(c.text, c.end); got != c.want {
			t.Errorf("rateAfter(%q, %d) = %v, muốn %v", c.text, c.end, got, c.want)
		}
	}
}

// R-01 — mốc thời gian người hỏi viết bằng chữ chỉ là bối cảnh vô hại khi kỳ ĐÃ ĐỌC nằm trọn trong nó; không có kỳ nào đọc được, kỳ rộng hơn / ngoài mốc,
// hay mốc là khoảng đếm thì không khớp. 19/10/2026 là thứ Hai của tuần ISO 43, quý 4, năm 2026.
func TestQualifierHolds(t *testing.T) {
	d := func(y int, m time.Month, dd int) time.Time { return time.Date(y, m, dd, 0, 0, 0, 0, time.UTC) }
	today := [2]time.Time{d(2026, 10, 19), d(2026, 10, 19)}
	month10 := [2]time.Time{d(2026, 10, 1), d(2026, 10, 31)}
	week43 := [2]time.Time{d(2026, 10, 19), d(2026, 10, 25)}
	crossYearWeek := [2]time.Time{d(2025, 12, 29), d(2026, 1, 4)} // tuần ISO 1 của 2026
	for _, c := range []struct {
		name    string
		q       normalization.Qualifier
		periods [][2]time.Time
		want    bool
	}{
		{"tuần 43 khớp hôm nay", normalization.Qualifier{Kind: normalization.QualifierWeek, Value: 43}, [][2]time.Time{today}, true},
		{"tuần 44 không khớp hôm nay", normalization.Qualifier{Kind: normalization.QualifierWeek, Value: 44}, [][2]time.Time{today}, false},
		{"tuần 43 khớp tuần này", normalization.Qualifier{Kind: normalization.QualifierWeek, Value: 43}, [][2]time.Time{week43}, true},
		{"tuần 43 không ôm trọn cả tháng", normalization.Qualifier{Kind: normalization.QualifierWeek, Value: 43}, [][2]time.Time{month10}, false},
		{"tuần 1 ôm tuần vắt hai năm", normalization.Qualifier{Kind: normalization.QualifierWeek, Value: 1}, [][2]time.Time{crossYearWeek}, true},
		{"tuần 43 không ôm trọn kỳ vắt hai tuần", normalization.Qualifier{Kind: normalization.QualifierWeek, Value: 43}, [][2]time.Time{{d(2026, 10, 19), d(2026, 10, 31)}}, false},
		{"tháng 9 không khớp hôm nay", normalization.Qualifier{Kind: normalization.QualifierMonth, Value: 9}, [][2]time.Time{today}, false},
		{"tháng 9 không ôm trọn kỳ vắt hai tháng", normalization.Qualifier{Kind: normalization.QualifierMonth, Value: 9}, [][2]time.Time{{d(2026, 9, 15), d(2026, 10, 15)}}, false},
		{"quý 3 không ôm trọn kỳ vắt hai quý", normalization.Qualifier{Kind: normalization.QualifierQuarter, Value: 3}, [][2]time.Time{{d(2026, 9, 15), d(2026, 10, 15)}}, false},
		{"năm 2025 không ôm trọn kỳ vắt sang năm sau", normalization.Qualifier{Kind: normalization.QualifierYear, Value: 2025}, [][2]time.Time{{d(2025, 12, 20), d(2026, 1, 5)}}, false},
		{"ngày 1 không ôm trọn cả tháng", normalization.Qualifier{Kind: normalization.QualifierDay, Value: 1}, [][2]time.Time{month10}, false},
		{"quý 4 khớp hôm nay", normalization.Qualifier{Kind: normalization.QualifierQuarter, Value: 4}, [][2]time.Time{today}, true},
		{"quý 4 khớp tháng 10", normalization.Qualifier{Kind: normalization.QualifierQuarter, Value: 4}, [][2]time.Time{month10}, true},
		{"quý 3 không khớp hôm nay", normalization.Qualifier{Kind: normalization.QualifierQuarter, Value: 3}, [][2]time.Time{today}, false},
		{"năm 2026 khớp hôm nay", normalization.Qualifier{Kind: normalization.QualifierYear, Value: 2026}, [][2]time.Time{today}, true},
		{"năm 2025 không khớp hôm nay", normalization.Qualifier{Kind: normalization.QualifierYear, Value: 2025}, [][2]time.Time{today}, false},
		{"ngày 19 khớp hôm nay", normalization.Qualifier{Kind: normalization.QualifierDay, Value: 19}, [][2]time.Time{today}, true},
		{"ngày 12 không khớp hôm nay", normalization.Qualifier{Kind: normalization.QualifierDay, Value: 12}, [][2]time.Time{today}, false},
		{"ngày 19 không ôm trọn cả tháng", normalization.Qualifier{Kind: normalization.QualifierDay, Value: 19}, [][2]time.Time{month10}, false},
		{"tháng 10 khớp hôm nay", normalization.Qualifier{Kind: normalization.QualifierMonth, Value: 10}, [][2]time.Time{today}, true},
		{"không có kỳ nào đọc được", normalization.Qualifier{Kind: normalization.QualifierWeek, Value: 43}, nil, false},
		{"khoảng đếm không kiểm được", normalization.Qualifier{Kind: normalization.QualifierSpan, Value: 7}, [][2]time.Time{today}, false},
		{"phải khớp MỌI kỳ đã đọc", normalization.Qualifier{Kind: normalization.QualifierQuarter, Value: 4}, [][2]time.Time{today, {d(2026, 7, 1), d(2026, 7, 31)}}, false},
	} {
		if got := qualifierHolds(c.q, c.periods); got != c.want {
			t.Errorf("%s: qualifierHolds = %v, muốn %v", c.name, got, c.want)
		}
	}
}
