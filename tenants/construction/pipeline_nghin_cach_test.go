package construction_test

import (
	"strings"
	"testing"
)

// Câu xác nhận có lượng viết nhóm nghìn ngăn bằng dấu cách ("3 850 lít", docs/tenants/construction/03-data-map.md dòng 14). Hôm nay =
// 19/10/2026, EX-03 được cấp 3.850 L (fixture seed). Lượng đứng ĐẦU câu là chắc chắn: phán quyết theo ĐÚNG lượng người hỏi nêu — trước đây
// "3 850 lít" thành số "3" + lượng "850 lít" nên câu đúng bị trả "Không … không phải 850 L" (ca 264 của H1). Lượng KHÔNG đứng đầu câu là
// CHƯA PHÂN ĐỊNH (D-OP-9, phương án a'): "được cấp 3 850 lít" là 3.850 L, hay số hiệu 3 rồi 850 L? Hệ không tin một cách đọc — hỏi lại, không
// phán quyết. Giá đã chấp nhận: câu thường gặp "được cấp 3 850 lít đúng không?" bị hỏi lại thay vì phán quyết.
func TestCauXacNhanNhomNghinNganBangDauCach(t *testing.T) {
	a := newAsk(t)
	zwsp := string(rune(0x200B)) // viết bằng rune: chuỗi thoát \u trong lệnh ghi tệp bị công cụ giải mã thành ký tự thật
	for _, c := range []struct {
		text, prefix, mustHave string
	}{
		{"3 850 lít EX-03 hôm nay được cấp đúng không?", "Đúng.", ""},
		{"3 850 L EX-03 hôm nay được cấp đúng không?", "Đúng.", ""},
		{"4 000 lít EX-03 hôm nay được cấp đúng không?", "Không.", "không phải 4.000 L"},
		{"1 490 lít EX-03 hôm nay được cấp đúng không?", "Không.", "không phải 1.490 L"},
		// khoảng trắng đầu câu không làm con số mất vị trí "đứng đầu"
		{"  3 850 lít EX-03 hôm nay được cấp đúng không?", "Đúng.", ""},
		// ký tự định dạng vô hình đầu câu cũng vậy (điều đo thứ 2 của D-OP-9): câu KHÔNG bị hỏi lại vì nhóm nghìn. Cổng phán quyết (claim.go) đóng
		// hơn: ZWSP là chữ chưa giải thích nên câu chỉ trả số liệu, không "Đúng." — fail-closed có từ trước, không thuộc nhóm nghìn.
		{zwsp + "3 850 lít EX-03 hôm nay được cấp đúng không?", "", ""},
	} {
		r := ask(t, a, user("director", ""), c.text).Response
		xacNhan(t, r, c.text, c.prefix)
		if c.mustHave != "" && !strings.Contains(r.Answer, c.mustHave) {
			t.Errorf("%q: câu trả lời %q thiếu %q", c.text, r.Answer, c.mustHave)
		}
	}
}

// Nhóm nghìn không đứng đầu câu ⇒ hỏi lại về CON SỐ cần xác nhận, kể cả khi lượng người hỏi nêu khớp số kho ("Đúng." im lặng cho một cách đọc hệ
// tự chọn) hay khác số kho ("Không … không phải 850 L" cho một cách đọc hệ tự chọn). Câu hỏi lại nhắc đúng chữ người hỏi viết.
func TestCauXacNhanNhomNghinKhongDungDauThiHoiLai(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct{ text, group string }{
		{"EX-03 hôm nay được cấp 3 850 lít dầu đúng không?", "3 850 lít"},
		{"EX-03 hôm nay được cấp 3 850 L đúng không?", "3 850 L"},
		{"EX-03 hôm nay được cấp 4 000 lít dầu đúng không?", "4 000 lít"},
		{"EX-03 hôm nay được cấp bể #2 500 lít đúng không?", "2 500 lít"},
		{"EX-03 hôm nay được cấp bể: 2 500 lít đúng không?", "2 500 lít"},
		{"EX-03 hôm nay được cấp 3850 L (3 850 lít) đúng không?", "3 850 lít"},
		{"EX-03 hôm nay được cấp 3 850 lít, 4 000 lít đúng không?", "4 000 lít"},
		// chữ đứng trước là từ nhãn / từ đếm: danh sách của bộ chuẩn hoá không phân định được, nên cũng chỉ là chữ đứng trước (C3-05: "tổng số
		// 3 850 lít" từng thành số "3" + lượng "850 lít" nên câu bị trả "không phải 850 L" khi chưa có cổng phán quyết)
		{"EX-03 hôm nay tổng số 3 850 lít đúng không?", "3 850 lít"},
		{"EX-03 hôm nay được cấp đợt 1 200 lít đúng không?", "1 200 lít"},
		{"EX-03 hôm nay được cấp phiếu 1 200 lít đúng không?", "1 200 lít"},
		{"EX-03 hôm nay được cấp ca 2 850 lít đúng không?", "2 850 lít"},
	} {
		r := ask(t, a, user("director", ""), c.text).Response
		hoiLaiVe(t, r, c.text, "claimed", "Chưa rõ cách đọc", `"`+c.group+`"`)
		if strings.Contains(r.Answer, "không phải") {
			t.Errorf("%q: câu hỏi lại không được bác một con số: %q", c.text, r.Answer)
		}
	}
}

// Lượng nhóm nghìn chồng MỘT PHẦN lên cụm tên thiết bị ("lu 2" của "lu 2 850 lít") không được loại im lặng: bản thử (a') đêm 21/10 bỏ nó vì
// "nằm trong tên" rồi trả lời như câu không có con số (mất lượng người hỏi nêu, S2). "2" là số hiệu của tên hay phần đầu của lượng — hỏi lại.
func TestCauXacNhanNhomNghinChongTenThietBiThiHoiLai(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct{ text, group string }{
		{"lu 2 850 lít hôm nay được cấp đúng không?", "2 850 lít"},
		{"RL-02 lu 2 850 lít hôm nay được cấp đúng không?", "2 850 lít"},
		// kèm một con số xác nhận khác: không phán quyết theo "1200 L" mà bỏ im lặng lượng chồng tên (phán quyết theo số còn lại, F2 của bản thử cũ)
		{"lu 2 850 lít hôm nay được cấp 1200 L đúng không?", "2 850 lít"},
		{"RL-02 hôm nay được cấp 1200 L, lu 2 850 lít đúng không?", "2 850 lít"},
	} {
		r := ask(t, a, user("director", ""), c.text).Response
		hoiLaiVe(t, r, c.text, "claimed", "Chưa rõ cách đọc", `"`+c.group+`"`)
	}
}

// Cờ chưa phân định chỉ chặn phán quyết khi lượng thuộc PHẠM VI hỏi và cùng chiều đo với chỉ số: lượng khác chiều đo ("12 000 km": bản thử
// fd4d19c không gộp nên thành "không phải 12 L", S1) và lượng ở mệnh đề ngữ cảnh đứng trước cụm xác nhận không làm câu bị hỏi lại.
func TestNhomNghinKhacChieuHoacNguCanhKhongChanPhanQuyet(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct{ text, prefix string }{
		{"EX-03 hôm nay được cấp 3 850 lít, vậy 3850 L đúng không?", "Đúng."},   // "3 850 lít" ở mệnh đề ngữ cảnh
		{"EX-03 hôm nay được cấp 3850 lít, máy chạy 12 000 km đúng không?", ""}, // khác chiều đo: gate chưa giải đủ chiều, không phán quyết
	} {
		r := ask(t, a, user("director", ""), c.text).Response
		xacNhan(t, r, c.text, c.prefix)
		if strings.Contains(r.Answer, "không phải") {
			t.Errorf("%q: không được bác con số: %q", c.text, r.Answer)
		}
	}
}

// Lượng nhóm nghìn chồng một phần lên tên nhưng KHÁC chiều đo ("lu 2 000 km") không phải con số xác nhận của chỉ số dầu: không hỏi lại về nó
// (cổng phán quyết vẫn không phán quyết vì câu có phần chưa giải thích). Hỏi lại về nhóm số cũng không kể máy ứng viên: thực thể đã rõ nhờ
// phạm vi của người hỏi (như câu hỏi lại về mốc thời gian, R-01).
func TestHoiLaiVeNhomNghinKhongOanVaKhongKeUngVien(t *testing.T) {
	a := newAsk(t)
	text := "RL-02 lu 2 000 km hôm nay được cấp 1200 L đúng không?"
	r := ask(t, a, user("director", ""), text).Response
	xacNhan(t, r, text, "")
	if strings.Contains(r.Answer, "Chưa rõ cách đọc") {
		t.Errorf("%q: nhóm khác chiều đo không phải con số xác nhận, không được hỏi lại về nó: %q", text, r.Answer)
	}
	text = "Máy 03 hôm nay được cấp 3 850 lít đúng không?"
	r = ask(t, a, user("site_engineer", "PRJ-A"), text).Response
	hoiLaiVe(t, r, text, "claimed", "Chưa rõ cách đọc", `"3 850 lít"`)
	if got := candidateIDs(r); len(got) != 0 {
		t.Errorf("%q: hỏi lại về nhóm số không được kể máy ứng viên: %v (%q)", text, got, r.Answer)
	}
}
