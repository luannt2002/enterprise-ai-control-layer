package ragcases

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
)

// Bộ ánh xạ bộ câu: đường dẫn chấm tới mảng, trường nguồn → vai trò, kỳ vọng chuỗi / danh sách, câu không có đáp án theo trường rỗng,
// theo giá trị hoặc theo trường đúng / sai (cả chuỗi "true"), lọc theo trường; JSON và YAML cùng một bộ đọc. Tên trường gõ sai và lọc
// không giữ câu nào là LỖI — không lặng lẽ biến cả bộ thành "phải từ chối" / "không chấm" / rỗng.
func TestAnhXaBoCau(t *testing.T) {
	dir := t.TempDir()
	js := `{"meta": {"v": 1}, "data": {"questions": [
	  {"qid": "Q1", "question": "Điều 1 quy định gì?", "exp": "phạm vi", "bot": "a", "bay": "false"},
	  {"qid": "Q2", "question": "Thời tiết hôm nay?", "exp": null, "bot": "a", "bay": "true"},
	  {"qid": "Q3", "question": "Hai mốc?", "exp": ["01/01/2021", "24 giờ"], "bot": "a"},
	  {"qid": "Q4", "question": "Bộ khác", "exp": "x", "bot": "b"}]}}`
	if err := os.WriteFile(filepath.Join(dir, "bo.json"), []byte(js), 0o644); err != nil {
		t.Fatal(err)
	}
	spec := boCauSpec{Ten: "t", Tep: "bo.json", DanhSach: "data.questions", Loc: map[string]string{"bot": "a"},
		Truong: truongCa{Ma: "qid", Hoi: "question", Chua: "exp"}, TuChoiKhiRong: "exp"}
	cas, err := docBoCau(dir, spec)
	if err != nil {
		t.Fatal(err)
	}
	if len(cas) != 3 || cas[0].Chua[0] != "phạm vi" || !cas[1].TuChoi || len(cas[2].Chua) != 2 || cas[0].TuChoi {
		t.Fatalf("ánh xạ sai: %+v", cas)
	}
	s := spec
	s.TuChoiKhiRong, s.Truong.TuChoi = "", "bay"
	if cas, err := docBoCau(dir, s); err != nil || cas[0].TuChoi || !cas[1].TuChoi {
		t.Errorf("trường đúng / sai dạng chuỗi: %+v %v", cas, err)
	}
	yml := "- {id: g1, question: đơn giá đào đất, loai: co_dap_an}\n- {id: g2, question: thời tiết, loai: khong_co_dap_an}\n"
	if err := os.WriteFile(filepath.Join(dir, "bo.yaml"), []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	cas, err = docBoCau(dir, boCauSpec{Ten: "y", Tep: "bo.yaml", Truong: truongCa{Ma: "id", Hoi: "question"},
		TuChoiKhi: map[string]string{"loai": "khong_co_dap_an"}})
	if err != nil || len(cas) != 2 || cas[0].TuChoi || !cas[1].TuChoi {
		t.Fatalf("YAML + tu_choi_khi: %+v %v", cas, err)
	}
	for name, mut := range map[string]func(*boCauSpec){
		"danh_sach không trỏ tới mảng": func(s *boCauSpec) { s.DanhSach = "meta" },
		"tên trường kỳ vọng gõ sai":    func(s *boCauSpec) { s.Truong.Chua, s.TuChoiKhiRong = "expected", "expected" },
		"khoá lọc không tồn tại":       func(s *boCauSpec) { s.Loc = map[string]string{"bot_id": "a"} },
		"giá trị lọc lệch":             func(s *boCauSpec) { s.Loc = map[string]string{"bot": "A"} },
	} {
		s := spec
		mut(&s)
		if _, err := docBoCau(dir, s); err == nil {
			t.Errorf("%s: muốn lỗi", name)
		}
	}
	for _, c := range []struct {
		text, want string
		ok         bool
	}{
		{"Đơn giá 1.250.000 đồng", "1250000", true}, {"Trụ sở ở Hà Nội", "ha noi", true}, {"Phạm vi điều chỉnh", "phạm vi", true},
		{"Phí 124 giờ", "24 giờ", false}, {"Nộp phạt", "phát", false}, {"Mã 1 2", "12", false}, {"abc", "xyz", false},
		{"Tối thiểu 6 tháng một lần", "sáu tháng một lần", true}, {"lưu 90 ngày", "chín mươi ngày", true}, {"báo cáo trong 24 giờ", "hai mươi bốn giờ", true},
		{"mật khẩu từ sáu ký tự", "6 ký tự", true}, {"mười lăm ngày", "15 ngày", true}, {"ba mươi mốt ngày", "31 ngày", true},
		// Soát độc lập 23/10: các khớp SAI của bản đầu — chữ thường trùng khoá bỏ dấu ("sau", "từ", "nam"), "năm" (thời hạn / năm lịch)
		// đọc thành 5, bỏ số 0 sau dấu thập phân, số ghép qua dấu câu, dạng bỏ dấu "hai muoi tu" (bỏ hẳn: lỏng hơn là khớp sai).
		{"có hiệu lực sau ngày ký", "6 ngày", false}, {"Sau tháng đầu tiên", "sáu tháng", false}, {"Tỷ lệ phạt 2,05%", "2,5%", false},
		{"0,05 mg/l", "0,5 mg/l", false}, {"Việt Nam năm 2020", "5 năm", false}, {"tháng 05 năm 2024", "5 năm", false},
		{"từ ngày ký", "4 ngay", false}, {"mười năm", "15 năm", false}, {"15 năm", "mười năm", false}, {"mười năm", "10 năm", true}, {"hai mươi năm", "20 năm", true},
		{"mười. Ba bên", "13", false}, {"lưu chín mươi ngày", "90", true}, {"hai muoi tu gio", "24 gio", false}, {"giá 2.050.000 đồng", "2.50.000", false},
		{"dòng một\nsáu tháng", "6 tháng", true}, {"thời hạn\nhai năm", "thời hạn hai năm", false},
		{"trong 5 ngày làm việc", "05 ngày làm việc", true}, {"hiệu lực từ 01/01/2021", "01 tháng 01 năm 2021", true}, {"hiệu lực từ 1/1/2021", "01 tháng 01 năm 2021", true},
		{"hiệu lực từ 01/01/2022", "01 tháng 01 năm 2021", false}, {"Mã 105", "05", false}, {"Tối thiểu 7 tháng", "sáu tháng", false},
	} {
		if khop(c.text, c.want) != c.ok {
			t.Errorf("khop(%q, %q) = %v", c.text, c.want, !c.ok)
		}
	}
}

// Bộ chấm chỉ ra TẦNG SAI ĐẦU TIÊN đúng cho từng kiểu trượt — kể cả trượt không do truy hồi (route khác, thông báo phiên bản, không gọi
// truy hồi); câu không có đáp án chỉ đạt khi hệ từ chối ĐÚNG lý do (bị S9 chặn = model đã bịa; từ chối vì quyền = lý do khác).
func TestDanhGiaTangSai(t *testing.T) {
	co := func(text string) lanTruy { return lanTruy{ev: []contracts.Evidence{{ID: "D#1", Text: text}}, calls: 1} }
	rong := lanTruy{calls: 1}
	out := func(route contracts.RouteKind, o contracts.Outcome, ans string) pipeline.Outcome {
		return pipeline.Outcome{Response: contracts.AskResponse{Outcome: o, Answer: ans}, Plan: contracts.RoutePlan{Route: route}}
	}
	rag := contracts.RouteRAG
	hoi := caChung{Ma: "q", Hoi: "Phạm vi?", Chua: []string{"phạm vi"}}
	bay := caChung{Ma: "b", Hoi: "Thời tiết?", TuChoi: true}
	for name, c := range map[string]struct {
		ca   caChung
		out  pipeline.Outcome
		tr   lanTruy
		dat  bool
		tang string
	}{
		"đạt":                     {hoi, out(rag, contracts.OutcomeResolved, "Phạm vi điều chỉnh là…"), co("Phạm vi điều chỉnh"), true, ""},
		"S4":                      {hoi, out(contracts.RouteReject, contracts.OutcomeOutOfScope, ""), lanTruy{}, false, "S4 ý định / định tuyến"},
		"S5":                      {hoi, out(contracts.RouteReject, contracts.OutcomePermissionDenied, ""), lanTruy{}, false, "S5 quyền"},
		"route số liệu":           {hoi, out(contracts.RouteSQLQuery, contracts.OutcomeInsufficientEvidence, ""), lanTruy{}, false, "S5 route khác tri thức (SQL_QUERY)"},
		"thông báo phiên bản":     {hoi, out(rag, contracts.OutcomeInsufficientEvidence, ""), lanTruy{calls: 1, notice: &contracts.KnowledgeNotice{Kind: contracts.NoticeNoEffectiveVersion, DocumentID: "D"}}, false, "S7 phiên bản (no_effective_version)"},
		"không gọi truy hồi":      {hoi, out(rag, contracts.OutcomeInsufficientEvidence, ""), lanTruy{}, false, "S6 không gọi truy hồi"},
		"truy hồi im":             {hoi, out(rag, contracts.OutcomeInsufficientEvidence, ""), rong, false, "S6 truy hồi: im"},
		"truy hồi trượt":          {hoi, out(rag, contracts.OutcomeInsufficientEvidence, ""), co("chuyện khác"), false, "S6 truy hồi: trượt"},
		"model từ chối":           {hoi, out(rag, contracts.OutcomeInsufficientEvidence, ""), co("Phạm vi điều chỉnh"), false, "S8 model từ chối"},
		"diễn đạt thiếu ý":        {hoi, out(rag, contracts.OutcomeResolved, "Thông tư này…"), co("Phạm vi điều chỉnh"), false, "S8 diễn đạt thiếu ý"},
		"S9":                      {hoi, out(rag, contracts.OutcomeSystemFailure, ""), co("Phạm vi điều chỉnh"), false, "S9 kiểm đầu ra chặn"},
		"bẫy: từ chối đúng":       {bay, out(rag, contracts.OutcomeInsufficientEvidence, ""), rong, true, ""},
		"bẫy: ngoài phạm vi đúng": {bay, out(contracts.RouteReject, contracts.OutcomeOutOfScope, ""), lanTruy{}, true, ""},
		"bẫy: trả lời":            {bay, out(rag, contracts.OutcomeResolved, "Trời nắng"), co("nắng"), false, "S6/S8 trả lời câu không có đáp án"},
		"bẫy: S9 chặn bản bịa":    {bay, out(rag, contracts.OutcomeSystemFailure, ""), co("nắng"), false, "S8 trả lời câu không có đáp án (S9 đã chặn)"},
		"bẫy: từ chối vì quyền":   {bay, out(contracts.RouteReject, contracts.OutcomePermissionDenied, ""), lanTruy{}, false, "từ chối vì lý do khác — S5 quyền"},
	} {
		kq := danhGia("t", c.ca, c.out, nil, c.tr)
		if kq.Dat != c.dat || kq.TangSai != c.tang || kq.KhongCham {
			t.Errorf("%s: đạt %v tầng %q, muốn %v %q", name, kq.Dat, kq.TangSai, c.dat, c.tang)
		}
	}
	if kq := danhGia("t", caChung{Ma: "c", Hoi: "Chào"}, out(contracts.RouteClarification, contracts.OutcomeClarificationRequired, ""), nil, lanTruy{}); !kq.KhongCham ||
		!strings.Contains(kq.tieuDe(), "KHÔNG CHẤM") {
		t.Errorf("ca không kỳ vọng phải KHÔNG CHẤM: %+v", kq)
	}
}
