// Test đầu–cuối cho CỔNG PHÁN QUYẾT của câu xác nhận (hợp đồng vận hành ca 21/10 §8, ADR-0009): "Đúng." / "Không." chỉ được nói khi
// claim của người hỏi đã giải đủ chiều — thực thể, nghĩa chỉ số, kỳ đọc được, đúng MỘT con số có đơn vị cùng chiều đo, chiều khẳng định,
// quan hệ bằng — và MỌI phần của mệnh đề hỏi đều được giải thích. Còn dư bất cứ gì thì không phán quyết: trả số liệu, hỏi lại hoặc báo
// thiếu bằng chứng. Pack THẬT, danh mục + fact giả lập; hôm nay = 19/10/2026, EX-03 được cấp 3.850 L.
package construction_test

import (
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// verdictOf trả phán quyết mở đầu câu trả lời: "Đúng.", "Không." hoặc "" (không phán quyết).
func verdictOf(answer string) string {
	for _, v := range []string{"Đúng.", "Không."} {
		if strings.HasPrefix(answer, v) {
			return v
		}
	}
	return ""
}

// noVerdictOK kiểm câu không phán quyết vẫn là một kết quả an toàn, có ích: trả số liệu có thẩm quyền, hỏi lại, hoặc báo thiếu bằng chứng
// — không phải lỗi hệ thống.
func noVerdictOK(t *testing.T, q string, r contracts.AskResponse) {
	t.Helper()
	switch r.Outcome {
	case contracts.OutcomeResolved, contracts.OutcomeClarificationRequired, contracts.OutcomeInsufficientEvidence:
	default:
		t.Errorf("%q: không phán quyết thì phải trả số / hỏi lại / thiếu bằng chứng, được [%s/%s] %q", q, r.Route, r.Outcome, r.Answer)
	}
}

// 15 câu thăm dò của các người soát đêm 20→21/10 (báo cáo sáng 21/10 mục 8): END 15ed135 phán quyết SAI cả 15. Mỗi câu chỉ được
// không phán quyết, hoặc phán quyết ĐÚNG khi câu có một phán quyết đúng duy nhất.
func TestCauThamDoXacNhanKhongPhanQuyetSai(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct{ q, right string }{
		{"Định mức 5000 L, vậy EX-03 hôm nay được cấp con số 3850 đúng không?", "Đúng."},
		{"EX-03 hôm nay được cấp con số 4000 đúng không, kho xuất 3850 L?", "Không."},
		{"EX-03 hôm nay được cấp 3850 L, tại bể 7 đúng không?", ""},             // "bể 7" chưa kiểm được
		{"EX-03 hôm nay đã được cấp dầu theo chứng từ 912 đúng không?", ""},     // "chứng từ 912" chưa kiểm được
		{"EX-03 hôm nay kho 2 được cấp dầu đúng không?", ""},                    // "kho 2" chưa kiểm được
		{"Dự án A Q3 được cấp 4600 L đúng không?", ""},                          // kỳ "Q3" chưa đọc được
		{"EX-03 tuần44 được cấp 3850 L đúng không?", ""},                        // kỳ "tuần44" chưa đọc được
		{"Định mức 5000 L, vậy EX-03 hôm nay được cấp 3850 kg đúng không?", ""}, // kg khác chiều đo
		{"EX-03 hôm nay không được cấp 3850 L đúng không?", ""},                 // phủ định: mẫu câu "không phải {claimed}" không diễn đạt được
		{"EX-03 hôm nay được cấp trên 3000 L đúng không?", "Đúng."},             // 3.850 > 3.000
		{"Cho hỏi 15-9 máy xúc EX-03 được cấp 3,5 m3 phải không?", ""},          // kỳ "15-9" chưa đọc được
		{"EX-03 15th Sep được cấp 3850 L đúng không?", ""},                      // kỳ "15th Sep" chưa đọc được
		{"EX-03 ngày 12 ca 2 được cấp 3850 L đúng không?", ""},                  // dữ liệu theo ngày, không theo ca
		{"EX-03 hôm nay ca 2 được cấp 3850 L đúng không?", ""},                  // như trên
		{"EX-03 sáng nay được cấp 3850 L đúng không?", ""},                      // dữ liệu theo ngày, không theo buổi
	} {
		r := ask(t, a, user("director", ""), c.q).Response
		switch v := verdictOf(r.Answer); {
		case v == "":
			noVerdictOK(t, c.q, r)
		case v != c.right:
			t.Errorf("%q: phán quyết SAI %q (đúng: %q hoặc không phán quyết): %q", c.q, v, c.right, r.Answer)
		}
	}
}

// Tính chất (metamorphic): chen vào một câu xác nhận đang được phán quyết đúng một cụm KHÔNG được giải thích — số khác, nhãn, mốc dưới
// ngày, phủ định, so sánh, nguồn / nghĩa khác, từ lạ — thì phán quyết phải MẤT, không bao giờ đổi chiều hay giữ nguyên như chưa có gì.
func TestCongPhanQuyetCumLaLamMatPhanQuyet(t *testing.T) {
	a := newAsk(t)
	bases := []struct{ head, tail, verdict string }{ // câu = head + " " + chèn + " " + tail
		{"EX-03 hôm nay", "được cấp 3850 L đúng không?", "Đúng."},
		{"EX-03 hôm nay", "được cấp 4000 L đúng không?", "Không."},
		{"EX-03 hôm nay", "được cấp 4k dầu đúng không?", "Không."},
	}
	inserts := []string{
		"tại bể 7", "theo chứng từ 912", "kho 2", "ca 2", "buổi sáng", "đợt 3", "lần 2",
		"không", "chưa", "theo ERP", "đội báo", "cảm biến đo", "theo hoá đơn",
		"trên", "dưới", "hơn", "ít nhất", "tổng cộng", "trung bình", "mỗi ngày",
		"xyzzy", "riêng phần dầu xe bồn", "Q3", "tuần44", "15th",
	}
	for _, b := range bases {
		base := b.head + " " + b.tail
		if v := verdictOf(ask(t, a, user("director", ""), base).Response.Answer); v != b.verdict {
			t.Fatalf("câu gốc %q: muốn %q, được %q — tính chất không kiểm được", base, b.verdict, v)
		}
		for _, ins := range inserts {
			q := b.head + " " + ins + " " + b.tail
			r := ask(t, a, user("director", ""), q).Response
			if v := verdictOf(r.Answer); v != "" {
				t.Errorf("%q: chen %q mà vẫn phán quyết %q: %q", q, ins, v, r.Answer)
				continue
			}
			noVerdictOK(t, q, r)
		}
	}
}

// Đối chứng của tính chất trên: từ trung tính (lịch sự, trợ từ, trợ động từ) không mang nghĩa claim nên GIỮ phán quyết — cổng đóng khi
// thiếu nhưng không được đóng oan những cách nói thường gặp.
func TestCongPhanQuyetTuTrungTinhGiuPhanQuyet(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct{ q, verdict string }{
		{"Cho hỏi EX-03 hôm nay được cấp 3850 L đúng không?", "Đúng."},
		{"EX-03 hôm nay đã được cấp 4000 L đúng không ạ?", "Không."},
		{"Hôm nay máy xúc EX-03 được cấp 3.850 lít dầu phải không?", "Đúng."},
		{"Vậy EX-03 hôm nay có được cấp 4000 L đúng không?", "Không."},
		{"EX-03 hôm nay được cấp 3850 L, đúng không?", "Đúng."},
	} {
		r := ask(t, a, user("director", ""), c.q).Response
		if v := verdictOf(r.Answer); v != c.verdict || r.Outcome != contracts.OutcomeResolved {
			t.Errorf("%q: muốn RESOLVED %q, được [%s/%s] %q", c.q, c.verdict, r.Route, r.Outcome, r.Answer)
		}
	}
}

// Nguồn và ngữ cảnh (ADR-0009): con số được so với nguồn có thẩm quyền cao nhất ⇒ câu nêu CHÍNH nguồn đó vẫn được phán quyết; nêu nguồn
// thấp hơn ("ERP") là claim về con số của nguồn ấy ⇒ không phán quyết. Mệnh đề ngữ cảnh nêu sự kiện của chỉ số / nguồn khác vẫn cho phán
// quyết; ngữ cảnh có phần lạ (phủ định, nhãn, số trần) thì không.
func TestCongPhanQuyetNguonVaNguCanh(t *testing.T) {
	a := newAsk(t)
	for _, c := range []struct{ q, verdict string }{
		{"Phiếu xuất kho ghi 3.850 L dầu cho EX-03 hôm nay đúng không?", "Đúng."},
		{"EX-03 ngày 19/10 phiếu xuất kho 4.000 L dầu đúng không?", "Không."},
		{"EX-03 hôm nay kho xuất ERP 3.900 L dầu đúng không?", ""},
		{"Hôm nay EX-03: phiếu kho 3.850 L, ERP 3.900 L, đội báo 4.000 L, vậy EX-03 được cấp 4.000 L dầu đúng không?", "Không."},
		{"Không tính ERP, EX-03 hôm nay được cấp 3850 L đúng không?", ""},
		{"Hôm nay ca 2 báo 2k, vậy EX-03 được cấp 3850 L đúng không?", ""},
		{"Phiếu kho 912, vậy EX-03 hôm nay được cấp 3850 L đúng không?", ""}, // số trần ở ngữ cảnh có thể là nhãn
	} {
		r := ask(t, a, user("director", ""), c.q).Response
		if v := verdictOf(r.Answer); v != c.verdict || r.Outcome != contracts.OutcomeResolved {
			t.Errorf("%q: muốn RESOLVED %q, được [%s/%s] %q", c.q, c.verdict, r.Route, r.Outcome, r.Answer)
		}
	}
}

// Câu không phán quyết vẫn mang đúng số có thẩm quyền và mẩu "chưa kết luận" — mẩu không có chữ "Đúng." / "Không." (bộ chấm đọc chuỗi đó
// là phán quyết) và không chép chữ của người hỏi.
func TestCongPhanQuyetMauKhongPhanQuyet(t *testing.T) {
	a := newAsk(t)
	q := "EX-03 hôm nay được cấp 3850 L tại bể A đúng không?"
	r := ask(t, a, user("director", ""), q).Response
	if r.Outcome != contracts.OutcomeResolved || !strings.Contains(r.Answer, "3.850 L") || !strings.Contains(r.Answer, "Chưa kết luận") ||
		strings.Contains(r.Answer, "Đúng.") || strings.Contains(r.Answer, "Không.") || strings.Contains(r.Answer, "bể A") {
		t.Errorf("%q: muốn trả số 3.850 L + mẩu chưa kết luận, không phán quyết, không chép chữ người hỏi — được [%s] %q", q, r.Outcome, r.Answer)
	}
}

// Các chiều khác của claim: chỉ số chỉ theo quy ước của pack (từ khoá chung "tổng"), không nêu mốc (kỳ mặc định của pack), con số xấp xỉ
// ("khoảng") — đều không phán quyết nhưng vẫn trả số liệu.
func TestCongPhanQuyetChieuChuaGiai(t *testing.T) {
	a := newAsk(t)
	for _, q := range []string{
		"EX-03 hôm nay tổng 3850 L dầu đúng không?",
		"EX-03 được cấp 3850 L dầu đúng không?",
		"EX-03 hôm nay được cấp khoảng 4000 L đúng không?",
	} {
		r := ask(t, a, user("director", ""), q).Response
		if v := verdictOf(r.Answer); v != "" || r.Outcome != contracts.OutcomeResolved {
			t.Errorf("%q: muốn RESOLVED không phán quyết, được [%s/%s] %q", q, r.Route, r.Outcome, r.Answer)
		}
	}
}

// Soát đối kháng độc lập (vai D, 21/10): các câu cổng từng hứa đóng mà chưa đóng — chủ thể (A, B), đơn vị / định dạng số (C), câu không
// dấu (D), "tuần" trước một ngày (E), nguồn (F: chạy trên bộ fact chỉ nguồn thấp), gạch dài làm mất dấu âm (G), từ khoá của ý định không
// phải số liệu ở mệnh đề ngữ cảnh (H), thuật ngữ / mã / tiền / số trần có dấu chấm trong phạm vi hỏi. Mọi câu phải KHÔNG phán quyết
// (vẫn trả số liệu); claim đã giải đủ chiều thì VẪN phán quyết (đối chứng dương).
func TestCongPhanQuyetSoatDoiKhang(t *testing.T) {
	std, low := newAsk(t), newAskFacts(t, "seed/facts-nguon-thap.yaml")
	for _, c := range []struct {
		q   string
		low bool
	}{
		{q: "Dự án A hôm nay được cấp 5050 L, vậy EX-03 được cấp 3850 L đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L, vậy dự án A được cấp 5050 L đúng không?"},
		{q: "Dự án A hôm nay được cấp 3850 L cho EX-03 đúng không?"},
		{q: "Cầu A (EX-03) hôm nay được cấp 3850 L đúng không?"},
		{q: "Hôm nay dự án A được cấp 5050 L. EX-03 được cấp 5050 L đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L đúng không, dự án A 3850 L đúng không?"},
		{q: "EX-03 của dự án B hôm nay được cấp 3850 L đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L cho dự án B đúng không?"},
		{q: "Lu 02 hôm nay được cấp 1,2 đúng không?"},
		{q: "Lu 02 hôm nay được cấp 1,200 khối đúng không?"},
		{q: "EX-03 hôm nay được cấp 3.850 m3 đúng không?"},
		{q: "EX-03 hom nay duoc cap co 4k dung khong?"},
		{q: "EX-03 hom nay duoc cap do 4000 L dung khong?"},
		{q: "EX-03 hom nay co the duoc cap 3850 L dung khong?"},
		{q: "Du an A dau thang 9 duoc cap 4600 L dung khong?"},
		{q: "EX-03 dau nam nay duoc cap 7350 L dung khong?"},
		{q: "Dự án A tuần 15/9 được cấp 4600 L đúng không?"},
		{q: "EX-03 hôm nay phiếu xuất kho ghi 3900 L đúng không?", low: true},
		{q: "EX-03 hôm nay phiếu kho ghi 3900 L đúng không?", low: true},
		{q: "EX-03 hôm nay báo cáo ngày ghi 4000 L đúng không?", low: true},
		{q: "EX-03 hôm nay được cấp –3850 L đúng không?"},
		{q: "EX-03 hôm nay được cấp —3850 L đúng không?"},
		{q: "Làm gì có, EX-03 hôm nay được cấp 3850 L đúng không?"},
		{q: "Khi lệch, EX-03 hôm nay được cấp 3850 L đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L khối lượng đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L PX-0913 đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L 25.000 đồng đúng không?"},
		{q: "EX-03 hôm nay được cấp 3.850 đúng không?"},
	} {
		a := std
		if c.low {
			a = low
		}
		r := ask(t, a, user("director", ""), c.q).Response
		if v := verdictOf(r.Answer); v != "" || r.Outcome != contracts.OutcomeResolved {
			t.Errorf("%q: muốn trả số, KHÔNG phán quyết — được [%s] %q", c.q, r.Outcome, r.Answer)
		}
	}
	for _, c := range []struct {
		q, want string
		low     bool
	}{
		{q: "Máy 03 của dự án A hôm nay được cấp 3850 L đúng không?", want: "Đúng."},
		{q: "EX-03 của dự án A hôm nay được cấp 3850 L đúng không?", want: "Đúng."},
		{q: "Dự án A hôm nay được cấp 3850 L đúng không?", want: "Không."},
		{q: "EX-03 hôm nay phiếu xuất kho ghi 3850 L đúng không?", want: "Đúng."},
		{q: "EX-03 hôm nay ERP ghi được cấp 3900 L đúng không?", want: "Đúng.", low: true}, // hôm nay chỉ ERP có số
		{q: "Lu rung 02 hôm nay được cấp 1,2 m3 đúng không?", want: "Đúng."},
		{q: "Đội báo 2k, vậy EX-03 hôm nay được cấp 3850 L đúng không?", want: "Đúng."},
		{q: "EX-03 hôm nay được cấp 3850 L có đúng không?", want: "Đúng."}, // "có đúng" và "đúng không" chồng nhau: MỘT cụm xác nhận
	} {
		a := std
		if c.low {
			a = low
		}
		r := ask(t, a, user("director", ""), c.q).Response
		if v := verdictOf(r.Answer); v != c.want {
			t.Errorf("%q: muốn %q — được [%s] %q", c.q, c.want, r.Outcome, r.Answer)
		}
	}
}

// Soát đối kháng vòng 2 (vai D, trên T2.1): một cụm xác nhận mà HAI phát biểu cùng con số ("EX-03 … 3850 L, dự án A 3850 L, đúng
// không?" — dự án A thật là 5.050 L), mệnh đề hỏi chỉ nêu cấp cha, lượng có dấu chấm / phẩy mà đơn vị chỉ suy từ thuật ngữ ("dầu 1,2")
// ⇒ KHÔNG phán quyết. Đối chứng: số nhắc lại liền kề trong ngoặc, "dầu 3850", cấp cha nhiều tầng, từ chỉ lượng ở ngữ cảnh ⇒ VẪN phán
// quyết; hai kỳ đọc được ⇒ không phán quyết.
func TestCongPhanQuyetSoatDoiKhangVong2(t *testing.T) {
	std := newAsk(t)
	tiers := newAskWith(t, "seed/entities-bon-tang.yaml", "seed/facts-nguon-thap.yaml")
	for _, q := range []string{
		"EX-03 hôm nay được cấp 3850 L, dự án A 3850 L, đúng không?",
		"EX-03 hôm nay được cấp 3850 L; dự án A cấp 3850 L; đúng không?",
		"EX-03 hôm nay được cấp 3850 L (dự án A 3850 L) đúng không?",
		"EX-03 hôm nay được cấp 3850 L, 3850 L, đúng không?",
		"EX-03 hôm nay được cấp 3850 L, dự án A đúng không?",
		"Lu 02 hôm nay được cấp dầu 1,2 đúng không?",
		"EX-03 hôm nay tháng 10 được cấp 3850 L đúng không?",
	} {
		r := ask(t, std, user("director", ""), q).Response
		if v := verdictOf(r.Answer); v != "" || r.Outcome != contracts.OutcomeResolved {
			t.Errorf("%q: muốn trả số, KHÔNG phán quyết — được [%s] %q", q, r.Outcome, r.Answer)
		}
	}
	for _, c := range []struct {
		q, want string
		tiers   bool
	}{
		{q: "EX-03 hôm nay được cấp 3850 L (3850 lít) đúng không?", want: "Đúng."},
		{q: "EX-03 hôm nay được cấp dầu 3850 đúng không?", want: "Đúng."},
		{q: "EX-03 hôm nay được cấp 3.850 L đúng không?", want: "Đúng."},
		{q: "Bao nhiêu định mức, vậy EX-03 hôm nay được cấp 3850 L đúng không?", want: "Đúng."},
		{q: "EX-03 của công ty ABC hôm nay được cấp 3900 L đúng không?", want: "Đúng.", tiers: true}, // công ty là cấp cha 3 tầng
	} {
		a := std
		if c.tiers {
			a = tiers
		}
		r := ask(t, a, user("director", ""), c.q).Response
		if v := verdictOf(r.Answer); v != c.want {
			t.Errorf("%q: muốn %q — được [%s] %q", c.q, c.want, r.Outcome, r.Answer)
		}
	}
}

// Soát lần ba (21/10, sau H3), hai người soát độc lập:
//   - dòng vá (ii) của T2.2 thu hẹp kiểm chủ thể từ CÂU về mệnh đề chứa cụm xác nhận ⇒ câu hai vế mà vế sau nói về dự án A bị phán
//     quyết theo EX-03 — SAI chiều (dự án A thật 5.050 L), T2.1 không phán quyết ⇒ kiểm chủ thể ở CẢ hai phạm vi;
//   - dòng vá (iv) làm mất phán quyết oan khi lượng có chữ ("3,85k", "1,2 nghìn"): chữ đã xoá mơ hồ phần nghìn / thập phân.
func TestCongPhanQuyetSoatLan3(t *testing.T) {
	std := newAsk(t)
	tiers := newAskWith(t, "seed/entities-bon-tang.yaml", "seed/facts-nguon-thap.yaml")
	for _, c := range []struct {
		q     string
		tiers bool
	}{
		{q: "EX-03 hôm nay được cấp 3850 L. Dự án A được cấp 5050 L, đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L. Dự án A được cấp 3850 L, đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L. Dự án A 5050 L, đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L! Dự án A 3850 L, đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L\nDự án A được cấp 3850 L, đúng không?"},
		{q: "EX-03 hôm nay được cấp bao nhiêu dầu? Dự án A được cấp 3850 L, đúng không?"},
		{q: "EX-03 hôm nay được cấp 3900 L. Công ty ABC 3900 L, đúng không?", tiers: true},
		// E3 (ADR-0009 R3-3, R3-4): sau con số xác nhận chỉ được nhắc lại CHÍNH chủ thể — dạng gương (tên đứng SAU lần nhắc lại) và
		// tên cấp cha sau con số với dấu ngăn bất kỳ
		{q: "EX-03 hôm nay được cấp 3850 L (3850 L của dự án A) đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L (3850 L cho dự án A) đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L 3850 L của dự án A đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L: dự án A đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L, dự án A, đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L - dự án A đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L đúng không, dự án A?"},
		{q: "EX-03 hôm nay được cấp 3900 L (3900 L của công ty ABC) đúng không?", tiers: true},
		// mệnh đề hỏi đứng TRƯỚC con số, chỉ nêu cấp cha (chủ thể ở trước nó): E3 không thấy tên này — phép kiểm ở mệnh đề hỏi bắt
		{q: "EX-03, dự án A đúng không, hôm nay được cấp 3850 L?"},
		{q: "EX-03 hôm nay, dự án A đúng không, được cấp 3850 L?"},
		// cấp cha đứng SAU con số trong cùng mệnh đề (5050 L là tổng của dự án A): không được phán quyết theo máy
		{q: "EX-03 hôm nay được cấp 5050 L của dự án A đúng không?"},
		{q: "EX-03 hôm nay được cấp 5050 L (dự án A) đúng không?"},
	} {
		a := std
		if c.tiers {
			a = tiers
		}
		r := ask(t, a, user("director", ""), c.q).Response
		if v := verdictOf(r.Answer); v != "" || r.Outcome != contracts.OutcomeResolved {
			t.Errorf("%q: chủ thể của câu hỏi chưa giải, muốn trả số, KHÔNG phán quyết — được [%s] %q", c.q, r.Outcome, r.Answer)
		}
	}
	for _, c := range []struct{ q, want string }{
		{q: "Dự án A hôm nay được cấp 5050 L đúng không?", want: "Đúng."}, // đối chứng: dự án A tự nó vẫn phán quyết được
		{q: "EX-03 hôm nay được cấp dầu 3,85k đúng không?", want: "Đúng."},
		{q: "Lu 02 hôm nay được cấp 1,2 nghìn dầu đúng không?", want: "Đúng."},
		// ghim nhánh của hai dòng vá mà đột biến từng sống (người soát góc B): ngoặc đóng, ba lần nhắc liền kề, xuống dòng, dấu chấm
		{q: "EX-03 hôm nay (được cấp 3850 L) 3850 L đúng không?", want: "Đúng."},
		{q: "EX-03 hôm nay được cấp 3850 L (3850 lít) (3850 L) đúng không?", want: "Đúng."},
		{q: "EX-03 hôm nay được cấp 3850 L\n3850 L đúng không?", want: ""},
		{q: "EX-03 hôm nay được cấp dầu 3.850 đúng không?", want: ""},
		// hậu tố nhân chỉ xoá mơ hồ khi đứng CUỐI con số: "3,85K" rõ; "1,2k5", "3,5k35" (chữ số sau hậu tố) vẫn mơ hồ (soát T2.3)
		{q: "EX-03 hôm nay được cấp dầu 3,85K đúng không?", want: "Đúng."},
		{q: "Lu 02 hôm nay được cấp dầu 1,2k5 đúng không?", want: ""},
		{q: "EX-03 hôm nay được cấp dầu 3,5k35 đúng không?", want: ""},
		// đối chứng dương của E3: nhắc lại CHÍNH chủ thể sau con số (tên đầy đủ, bí danh có chủ thể trong ứng viên) vẫn phán quyết
		{q: "EX-03 hôm nay được cấp 3850 L (máy xúc EX-03) đúng không?", want: "Đúng."},
		{q: "EX-03 hôm nay được cấp 3850 L máy 03 đúng không?", want: "Đúng."},
	} {
		r := ask(t, std, user("director", ""), c.q).Response
		if v := verdictOf(r.Answer); v != c.want {
			t.Errorf("%q: muốn %q — được [%s] %q", c.q, c.want, r.Outcome, r.Answer)
		}
	}
}

// ADR-0009 Vòng 5: đóng R4-2, R4-3 (S1), R4-4 (S2), R5-1 (S1: ngữ cảnh nêu nguồn khác, chỉ số khác hay từ chỉ lượng mà không kèm số
// của riêng nó; gồm R2-4) và R5-2 (S1: số bằng chữ "<hàng> <một chữ số>", phản hồi audit ROADMAP K5). Kỳ vọng suy từ luật của ADR, không
// chép đầu ra của bản sửa; ca R4 theo bảng của bản soát mã độc lập 21/10, ca D theo bản đột biến 22/10. Năm dạng chính sách mặc định
// (bảng của ADR-0009 Vòng 5) được ghim theo hành vi đã chọn: chủ dự án quyết khác thì đổi luật cùng các dòng "mặc định Vòng 5".
func TestCongPhanQuyetVong5(t *testing.T) {
	std := newAsk(t)
	tiers := newAskWith(t, "seed/entities-bon-tang.yaml", "seed/facts-nguon-thap.yaml")
	for _, c := range []struct {
		q     string
		tiers bool
	}{
		// R4-2: tên cấp cha đứng TRƯỚC con số, ở mệnh đề khác hay sau dấu ngăn lạ: con số là của cấp cha
		{q: "EX-03 hôm nay được cấp 3850 L. Dự án A, 5050 L đúng không?"},
		{q: "EX-03: dự án A hôm nay được cấp 3850 L đúng không?"},
		{q: "EX-03 - dự án A hôm nay được cấp 3850 L đúng không?"},
		{q: "EX-03 (dự án A hôm nay được cấp 3850 L đúng không?"},
		{q: "EX-03 [dự án A hôm nay được cấp 3850 L đúng không?"},
		{q: "EX-03\ndự án A hôm nay được cấp 3850 L đúng không?"},
		{q: "EX-03,dự án A hôm nay được cấp 3850 L đúng không?"},
		{q: "EX-03 hôm nay được cấp 3900 L. Công ty ABC, 5000 L đúng không?", tiers: true},
		{q: "EX-03: chương trình quốc lộ hôm nay được cấp 3900 L đúng không?", tiers: true},
		// R4-3: danh từ chung "dự án", "công trình" đứng một mình là tên chưa giải
		{q: "EX-03 hôm nay được cấp 3850 L. Dự án, 5050 L đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L. Công trình, 5050 L đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L. Dự án, 3850 L đúng không?"},
		// R4-4: bí danh mơ hồ "máy 03" tách câu khỏi lần nhắc rõ của chủ thể
		{q: "EX-03 hôm nay được cấp 3850 L. Máy 03 được cấp 3000 L, đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L. Máy 03, 3000 L đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L. Máy 03 3850 L, đúng không?"},
		// R5-1: ngữ cảnh nêu nguồn khác / chỉ số khác / từ chỉ lượng mà không kèm số của riêng nó: đó là nguồn / chỉ số của chính claim
		// (ERP ghi 3.900 L; đội báo 4.000 L)
		{q: "EX-03 hôm nay được cấp 3850 L. ERP, 3900 L đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L. ERP, 3850 L đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L. Định mức, 5000 L đúng không?"},
		{q: "ERP, EX-03 hôm nay được cấp 3900 L đúng không?"},
		{q: "Đội báo, EX-03 hôm nay được cấp 3850 L đúng không?"},
		{q: "EX-03 hôm nay đội báo, được cấp 3850 L đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L. Đội báo, 4000 L được cấp đúng không?"},
		{q: "Định mức, hôm nay được cấp 3850 L cho EX-03 đúng không?"},
		{q: "Hôm nay EX-03 được cấp bao nhiêu dầu? Bảng tổng hợp, 3850 L đúng không?"},
		// đề mục có mốc / từ trung tính vẫn là đề mục: "ngày" đứng trước mốc số, "hôm nay", "cho hỏi", "ghi"
		{q: "Ngày 19/10, ERP, EX-03 được cấp 3900 L đúng không?"},
		{q: "ERP ngày 19/10, EX-03 được cấp 3900 L đúng không?"},
		{q: "Hôm nay, ERP, EX-03 được cấp 3900 L đúng không?"},
		{q: "ERP ghi, EX-03 hôm nay được cấp 3900 L đúng không?"},
		{q: "Cho hỏi, ERP, EX-03 hôm nay được cấp 3900 L đúng không?"},
		// R5-4: đề mục có thêm thuật ngữ / từ khoá của chính ý định / tên đúng nguồn đã dùng vẫn là đề mục (ERP ghi 3.900 L, đội báo 4.000 L)
		{q: "Dầu ERP, EX-03 hôm nay được cấp 3900 L đúng không?"},
		{q: "Ngày 19/10, dầu ERP, EX-03 được cấp 3900 L đúng không?"},
		{q: "Dầu đội báo, EX-03 hôm nay được cấp 3850 L đúng không?"},
		{q: "Kho xuất, ERP, EX-03 hôm nay được cấp 3900 L đúng không?"},
		{q: "Phiếu xuất kho, ERP, EX-03 hôm nay được cấp 3850 L đúng không?"},
		// R5-5: "phiếu kho" là nguồn khác khi số trả lời lấy từ ERP (bộ nguồn thấp: phiếu kho không có bản ghi)
		{q: "Phiếu kho, EX-03 hôm nay được cấp 3900 L đúng không?", tiers: true},
		// ... kể cả khi "phiếu kho" đứng ở ngữ cảnh sau tên chủ thể: nó vẫn là chủ gần nhất của con số (soát vòng 4, V5S4-01)
		{q: "EX-03 hôm nay phiếu kho, được cấp 3900 L đúng không?", tiers: true},
		// số bằng chữ kết thúc bằng "<hàng> <một chữ số>": nói thường là hàng kế dưới (1.200, 3.800), bộ đọc cho 1.002, 3.008
		{q: "Lu 02 hôm nay được cấp một nghìn hai lít dầu đúng không?"},
		{q: "EX-03 hôm nay được cấp ba nghìn tám lít dầu đúng không?"},
		{q: "EX-03 hôm nay được cấp ba nghìn tám rưỡi lít dầu đúng không?"},
		// R4-4: bí danh mơ hồ sau con số, ngăn bằng dấu câu (luật bí danh: không có bằng chứng dương)
		{q: "EX-03 ngày 19/10 được cấp 3850 L, máy 03 đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L; máy 03 đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L - máy 03 đúng không?"},
		{q: "EX-03 hôm nay được cấp 3850 L: máy 03 đúng không?"},
		{q: "EX-03, (máy 03) hôm nay được cấp 3850 L đúng không?"},
		{q: "Máy 03 hôm nay được cấp 3850 L, EX-03 đúng không?"},
		// định ngữ chỉ nối bằng ĐÚNG chữ "của", mọi mắt xích của chuỗi
		{q: "EX-03 hôm nay được cấp của dự án A 3850 L đúng không?"},
		{q: "EX-03: dự án A của chương trình quốc lộ hôm nay được cấp 3900 L đúng không?", tiers: true},
		// mặc định Vòng 5 (fail-closed): viết liền, đồng vị phẩy, bí danh chen giữa không ngắt
		{q: "EX-03 dự án A hôm nay được cấp 3850 L đúng không?"},
		{q: "EX-03, dự án A, hôm nay được cấp 3850 L, đúng không?"},
		{q: "EX-03 hôm nay Máy 03 được cấp 3850 L đúng không?"},
	} {
		a := std
		if c.tiers {
			a = tiers
		}
		r := ask(t, a, user("director", ""), c.q).Response
		if v := verdictOf(r.Answer); v != "" || r.Outcome != contracts.OutcomeResolved {
			t.Errorf("%q: claim chưa giải, muốn trả số, KHÔNG phán quyết — được [%s] %q", c.q, r.Outcome, r.Answer)
		}
	}
	for _, c := range []struct {
		q, want string
		tiers   bool
	}{
		// định ngữ "của" (cả chuỗi nhiều tầng) gắn cấp cha vào chủ thể: vẫn phán quyết, đúng chiều
		{q: "EX-03 của dự án A hôm nay được cấp 3850 L đúng không?", want: "Đúng."},
		{q: "EX-03 của dự án A hôm nay được cấp 5050 L đúng không?", want: "Không."},
		{q: "EX-03 của dự án A của chương trình quốc lộ hôm nay được cấp 3900 L đúng không?", want: "Đúng.", tiers: true},
		// ngày viết bằng số không phải ngắt mệnh đề: bí danh sau số vẫn nhắc lại chính chủ thể (đối chứng đã ghim ở Vòng 4, R4-6)
		{q: "EX-03 ngày 19/10 được cấp 3850 L máy 03 đúng không?", want: "Đúng."},
		{q: "EX-03 ngày 19/10/2026 được cấp 3850 L máy 03 đúng không?", want: "Đúng."},
		{q: "EX-03 ngày 19.10 được cấp 3850 L máy 03 đúng không?", want: "Đúng."},
		{q: "EX-03 của dự án A ngày 19/10 được cấp 3850 L đúng không?", want: "Đúng."},
		// ngữ cảnh nêu lượng của chỉ số / nguồn khác, chủ thể đứng ngay trước con số: vẫn phán quyết
		{q: "Định mức 5000 L, vậy EX-03 của dự án A hôm nay được cấp 3850 L đúng không?", want: "Đúng."},
		{q: "ERP ghi 3900 L, vậy EX-03 hôm nay được cấp 3850 L đúng không?", want: "Đúng."},
		{q: "Đội báo 4000 L, vậy EX-03 hôm nay được cấp 3850 L đúng không?", want: "Đúng."},
		// cụm ở ngữ cảnh đứng ngay trước con số mà KHÔNG phải nguồn / chỉ số khác: từ hỏi của ý định yếu ("bao nhiêu"), từ khoá của
		// chính ý định ("được cấp", "phiếu kho")
		{q: "EX-03 hôm nay được cấp bao nhiêu? 3850 L đúng không?", want: "Đúng."},
		{q: "EX-03 hôm nay được cấp 3850 L, 3850 L đúng không?", want: "Đúng."},
		{q: "EX-03 hôm nay được cấp dầu. Được cấp, 3850 L đúng không?", want: "Đúng."},
		{q: "EX-03 hôm nay được cấp 3850 L. Phiếu kho, 3850 L đúng không?", want: "Đúng."},
		// tên ĐÚNG nguồn đã tạo con số trả lời (nhãn "phiếu xuất kho", không phải từ khoá): vẫn phán quyết, cả hai chiều
		{q: "Hôm nay EX-03 được cấp bao nhiêu dầu? Phiếu xuất kho, 3850 L đúng không?", want: "Đúng."},
		{q: "EX-03 hôm nay được cấp 3850 L. Phiếu xuất kho, 3850 L đúng không?", want: "Đúng."},
		{q: "EX-03 hôm nay được cấp 3850 L. Phiếu xuất kho, 3900 L đúng không?", want: "Không."},
		{q: "Phiếu xuất kho, EX-03 hôm nay được cấp 3850 L đúng không?", want: "Đúng."},
		{q: "Phiếu xuất kho ngày 19/10, EX-03 được cấp 3850 L đúng không?", want: "Đúng."},
		// đề mục chỉ gồm chữ của chính claim (thuật ngữ, từ khoá chính ý định, đúng nguồn đã dùng): không phải nguồn / chỉ số khác
		{q: "Dầu, EX-03 hôm nay được cấp 3850 L đúng không?", want: "Đúng."},
		{q: "Kho xuất, EX-03 hôm nay được cấp 3850 L đúng không?", want: "Đúng."},
		{q: "Phiếu xuất kho, dầu, EX-03 hôm nay được cấp 3850 L đúng không?", want: "Đúng."},
		// đề mục chỉ có mốc / từ trung tính không phải nguồn / chỉ số khác
		{q: "Hôm nay, EX-03 được cấp 3850 L đúng không?", want: "Đúng."},
		{q: "Cho hỏi, EX-03 hôm nay được cấp 3850 L đúng không?", want: "Đúng."},
		// số bằng chữ mơ hồ ở NGỮ CẢNH chỉ được che, không đem so; bí danh mơ hồ ở ngữ cảnh không phải kiểm
		{q: "Đội báo hai nghìn tám lít, vậy EX-03 hôm nay được cấp 3850 L đúng không?", want: "Đúng."},
		{q: "Máy 03 được cấp 3000 L. EX-03 hôm nay được cấp 3850 L, đúng không?", want: "Đúng."},
		{q: "Máy 03 được cấp 3000 L. EX-03 hôm nay được cấp 4000 L, đúng không?", want: "Không."},
		// "của" không phân biệt hoa thường; lần nhắc rõ GẦN NHẤT quyết bí danh trong ngoặc
		{q: "EX-03 CỦA dự án A hôm nay được cấp 3850 L đúng không?", want: "Đúng."},
		{q: "EX-03 hôm nay được cấp 3850 L, EX-03 (máy 03) đúng không?", want: "Đúng."},
		// mặc định Vòng 5: định ngữ trong ngoặc trần, bí danh rồi lần nhắc rõ trong ngoặc
		{q: "EX-03 (dự án A) hôm nay được cấp 3850 L đúng không?", want: "Đúng."},
		{q: "Máy 03 (EX-03) hôm nay được cấp 3850 L đúng không?", want: "Đúng."},
		// cụm ở ngữ cảnh đã có số của riêng nó ("đội báo 3k"): con số xác nhận đứng sau không phải của nó
		{q: "EX-03 hôm nay đội báo 3k, vậy được cấp 3850 L đúng không?", want: "Đúng."},
		{q: "EX-03 hôm nay ERP ghi 3900 L, vậy được cấp 3850 L đúng không?", want: "Đúng."},
		// bí danh gỡ bằng cấp cha, không có lần nhắc rõ nào của chủ thể
		{q: "Dự án A máy 03 hôm nay được cấp 3850 L đúng không?", want: "Đúng."},
		// số bằng chữ viết rõ hàng ("trăm", "linh", "rưỡi") vẫn phán quyết, cả hai chiều
		{q: "Lu 02 hôm nay được cấp một nghìn hai trăm lít dầu đúng không?", want: "Đúng."},
		{q: "Lu 02 hôm nay được cấp một nghìn không trăm linh hai lít dầu đúng không?", want: "Không."},
		{q: "EX-03 hôm nay được cấp ba nghìn tám trăm năm mươi lít dầu đúng không?", want: "Đúng."},
		{q: "EX-03 hôm nay được cấp hai nghìn rưỡi lít dầu đúng không?", want: "Không."},
	} {
		a := std
		if c.tiers {
			a = tiers
		}
		r := ask(t, a, user("director", ""), c.q).Response
		if v := verdictOf(r.Answer); v != c.want {
			t.Errorf("%q: muốn %q — được [%s] %q", c.q, c.want, r.Outcome, r.Answer)
		}
	}
}
