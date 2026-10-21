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
