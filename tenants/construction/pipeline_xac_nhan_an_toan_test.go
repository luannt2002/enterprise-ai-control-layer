package construction_test

import (
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
)

// Câu xác nhận có HAI con số trở lên (lượng ngữ cảnh, dải, lựa chọn) không được phán quyết theo một con số do hệ tự chọn: hỏi lại,
// như trước bản sửa A (1276b5f). Luật A-02 ("số trần cạnh lượng viết rõ đơn vị là nhãn") đổi các câu này thành "Không." SAI; bản sửa R
// vá bằng danh sách đóng và hai người soát độc lập vẫn tìm ra biến thể (R-A06, R-A07, B-02, B-03). Hôm nay = 19/10/2026; EX-03 được cấp
// 3.850 L, đội báo 4.000 L.
func TestCauXacNhanNhieuConSoKhongPhanQuyetSai(t *testing.T) {
	a := newAsk(t)
	for _, q := range []string{
		"Định mức 5.000 L nên EX-03 hôm nay được cấp 3850 đúng không?",
		"Bồn chứa 10 m3 nên EX-03 hôm nay được cấp 3850 đúng không?",
		"Định mức 5.000 lít mỗi ngày nên EX-03 hôm nay được cấp 3850 đúng không?",
		"EX-03 hôm nay được cấp 3800 ~ 3900 L đúng không?",
		"EX-03 hôm nay được cấp 3850 chứ ko phải 4000 L đúng không?",
		"EX-03 đội báo 3850 L hoặc 4000 hôm nay đúng không?",
		"EX-03 được cấp 4000 L hay 3850 hôm nay đúng không?",
	} {
		r := ask(t, a, user("director", ""), q).Response
		if strings.HasPrefix(r.Answer, "Không.") || r.Outcome != contracts.OutcomeClarificationRequired {
			t.Errorf("%q: muốn hỏi lại (không phán quyết thay người hỏi), được [%s/%s] %q", q, r.Route, r.Outcome, r.Answer)
		}
	}
}

// Câu xác nhận nêu một MỐC THỜI GIAN chưa đọc được (số đứng sau từ chỉ kỳ, kể cả khi ngay sau đó là danh từ đếm / nhãn hay chen
// "thứ" / "số") thì hỏi lại, không phán quyết cho hôm nay hay cho kỳ khác (R-A01, R-A02, R-A03, R-A04, B-01).
func TestCauXacNhanMocChuaDocDuocThiHoiLai(t *testing.T) {
	a := newAsk(t)
	for _, q := range []string{
		"Quý 3 dự án A được cấp 4600 L đúng không?",
		"Tuần 41 máy xúc EX-03 được cấp 3850 L đúng không?",
		"EX-03 tuần thứ 41 được cấp 3850 L đúng không?",
		"EX-03 ngày số 12 được cấp 3850 L đúng không?",
		"EX-03 tháng 9 2025 được cấp 3500 L đúng không?",
		"Hôm 15 máy xúc 03 đã nhận 3850 L đúng không?",
		"EX-03 15 thg 9 được cấp 3850 L đúng không?",
		// "N-M" sau từ nhãn: số hiệu hay ngày dd-mm? Không phân định được ⇒ hỏi lại (soát 37e8a08, F6: bản miễn của R-A14 làm "phiếu 15-9"
		// rơi im lặng về hôm nay). Hỏi lại thừa ở "phiếu 12-10" là giá đã chấp nhận.
		"RL-02 phiếu 20-9 được cấp 1100 L đúng không?",
		"EX-03 phiếu 15-9 được cấp 3500 L đúng không?",
		"EX-03 đợt 12-10 được cấp 3850 L đúng không?",
		"EX-03 hôm nay theo phiếu 12-10 được cấp 3850 L đúng không?",
	} {
		r := ask(t, a, user("director", ""), q).Response
		if r.Outcome != contracts.OutcomeClarificationRequired {
			t.Errorf("%q: muốn hỏi lại mốc thời gian, được [%s/%s] %q", q, r.Route, r.Outcome, r.Answer)
		}
	}
}

// Đối chứng: câu xác nhận rõ nghĩa một vế vẫn được phán quyết, kể cả khi có số hiệu sau từ nhãn ("phiếu 912", "ca 2", "phiếu 12-45" — "N-M"
// không có dạng ngày thì phần "-M" thuộc số hiệu).
func TestCauXacNhanMotVeVanPhanQuyet(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct{ q, prefix string }{
		{"EX-03 hôm nay được cấp 3850 L đúng không?", "Đúng."},
		{"EX-03 hôm nay được cấp 4000 L đúng không?", "Không."},
		{"EX-03 hôm nay được cấp 3850 đúng không?", "Đúng."},
		{"EX-03 hôm nay theo phiếu 912 được cấp 3850 L đúng không?", "Đúng."},
		{"EX-03 hôm nay ca 2 được cấp 3850 L đúng không?", "Đúng."},
		{"EX-03 hôm nay theo phiếu 12-45 được cấp 3850 L đúng không?", "Đúng."},
	} {
		xacNhan(t, ask(t, a, user("director", ""), c.q).Response, c.q, c.prefix)
	}
}

// Số trần đứng SAU một lượng và nối với nó bằng từ nối dùng chung đơn vị của lượng đó (joinClaims): "3,85 m3 hay 3,85" là MỘT con số
// (3.850 L) nên vẫn phán quyết; số đứng trước một danh từ ("3,85 bể") thì không nối nên là con số thứ hai — hỏi lại. Sau khi gỡ luật A-02
// (37e8a08), nối chỉ còn đổi được KẾT QUẢ ở những câu này (hai vế trùng giá trị sau khi dùng chung đơn vị); đột biến R02c, R02d, R02p,
// R02q của n35 sống trên 37e8a08 vì thiếu đúng các ca này.
func TestCauXacNhanSoNoiSauLuongDungChungDonVi(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct{ q, prefix string }{
		{"EX-03 hôm nay được cấp 3,85 m3 hay 3,85 đúng không?", "Đúng."},
		{"EX-03 hôm nay được cấp 3,85 m3 (hoặc 3,85) đúng không?", "Đúng."},
	} {
		xacNhan(t, ask(t, a, user("director", ""), c.q).Response, c.q, c.prefix)
	}
	q := "EX-03 hôm nay được cấp 3,85 m3 hay 3,85 bể đúng không?"
	if r := ask(t, a, user("director", ""), q).Response; r.Outcome != contracts.OutcomeClarificationRequired {
		t.Errorf("%q: muốn hỏi lại (3,85 bể là số đếm, không nối vào m3), được [%s/%s] %q", q, r.Route, r.Outcome, r.Answer)
	}
}
