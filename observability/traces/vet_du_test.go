package traces_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// R2-03. Các test dưới đây chạy lượt hỏi THẬT rồi đọc vết ở dạng người soát đọc (`eais trace`): các bước S2, S3, S4, S5, S7 và cổng phán
// quyết ở S9 phải mang đầu vào quyết định, phương án đã xét và lý do bị loại do chính luồng hỏi ghi tại chỗ quyết định — không còn ô
// "chưa ghi". Mỗi ca chỉ khẳng định thứ mà ca đó dạy; chữ cụ thể lấy từ đúng dữ liệu của lượt (luật sinh đoạn, khoá thực thể, tên ý định…).

// text gộp mọi chữ của một bước mà người đọc thấy: vào, ra, lý do, phương án bị loại.
func text(s traces.Step) string {
	return strings.Join(slices.Concat(s.Input, s.Output, []string{s.Reason}, s.Rejected), "\n")
}

func steps(t *testing.T, pr contracts.Principal, q string) []traces.Step {
	t.Helper()
	return traces.NewRecord(realTurn(t, realAsk(t), pr, q)).Steps
}

func wantAll(t *testing.T, label, got string, frags ...string) {
	t.Helper()
	for _, f := range frags {
		if !strings.Contains(got, f) {
			t.Errorf("%s: thiếu %q trong:\n%s", label, f, got)
		}
	}
}

// Không còn ô "chưa ghi" ở các bước đã được xuất, trên mọi kiểu lượt: số liệu, hỏi lại, từ chối, tri thức, hỗn hợp, xác nhận.
func TestVetDuKhongConODeTrongOCacBuocDaXuat(t *testing.T) {
	director := person("director", "")
	for _, c := range []struct {
		name string
		pr   contracts.Principal
		q    string
	}{
		{"số liệu có xung đột nguồn", director, "EX-03 hôm nay được cấp bao nhiêu lít dầu?"},
		{"hỏi lại vì bí danh mơ hồ", director, "Máy 03 hôm nay được cấp bao nhiêu dầu?"},
		{"từ chối vì ngoài phạm vi", person("site_engineer", "PRJ-B"), "EX-03 hôm nay được cấp bao nhiêu dầu?"},
		{"ngoài phạm vi: không ý định", director, "Tối nay kèo nào thơm?"},
		{"tri thức", director, "Quy trình xử lý chênh lệch nhiên liệu là gì?"},
		{"hỗn hợp số liệu và tri thức", director, "EX-03 hôm nay được cấp bao nhiêu dầu, và theo quy trình thì khi lệch phải làm gì?"},
		{"xác nhận đúng", director, "EX-03 hôm nay được cấp 3850 L đúng không?"},
		{"xác nhận bị giữ phán quyết", director, "EX-03 được cấp 3850 L đúng không?"},
		{"không có dữ liệu", director, "Máy 07 hôm nay được cấp bao nhiêu dầu?"},
	} {
		t.Run(c.name, func(t *testing.T) {
			for _, s := range steps(t, c.pr, c.q) {
				switch s.Stage {
				case "S2", "S3", "S4", "S5", "S7", "S9":
				default:
					continue
				}
				if s.Status == traces.StatusSkipped {
					continue
				}
				if traces.IsNotRecorded(s.Reason) || slices.ContainsFunc(s.Rejected, traces.IsNotRecorded) {
					// S9 kiểm trượt còn một chỗ trống có chủ đích (bản nháp bị loại) — ca nào kiểm trượt thì không nằm trong danh sách này
					t.Errorf("%s %s còn chỗ trống: lý do %q, loại %q", s.Stage, s.Name, s.Reason, s.Rejected)
				}
			}
		})
	}
}

// S2: luật sinh từng đoạn (Span.Rule) và cách đọc khác của chỗ mơ hồ / ngày hỏng.
func TestVetDuS2LuatSinhDoanVaChoMoHo(t *testing.T) {
	director := person("director", "")
	s2 := stepOf(t, steps(t, director, "EX-03 hôm nay được cấp 4k dầu đúng không?"), "S2")
	wantAll(t, "S2 luật sinh đoạn", s2.Reason, "quantity “4k”", "unit_from_term", "date “hôm nay”", "date:relative")

	// viết tắt nhiều nghĩa đứng trước số trần: bộ chuẩn hoá không tự quyết, hai cách đọc phải nằm trong phương án bị loại
	amb := stepOf(t, steps(t, director, "EX-03 hôm nay DA: 4.000 đúng không?"), "S2")
	wantAll(t, "S2 chỗ mơ hồ", strings.Join(amb.Rejected, "\n"), `“DA”`, "diesel_fuel", "project")

	// ngày không có trên lịch: không sinh đoạn ngày, và vết nói rõ nó bị bỏ chứ không rơi về ngày mặc định
	bad := stepOf(t, steps(t, director, "EX-03 ngày 31/02 được cấp bao nhiêu dầu?"), "S2")
	wantAll(t, "S2 ngày hỏng", strings.Join(bad.Rejected, "\n"), "31/02", "không đọc được")
}

// S3: thực thể rõ, cụm mơ hồ, và tập ứng viên TRƯỚC / SAU lọc phạm vi — lọc ở router (gỡ mơ hồ) và lọc ở câu hỏi lại.
func TestVetDuS3UngVienTruocSauLocPhamVi(t *testing.T) {
	// kỹ sư chỉ được xem PRJ-A: router gỡ "máy 03" về EX-03 bằng phạm vi; DZ-03 (PRJ-B) bị loại và vết nói đúng lý do
	eng := stepOf(t, steps(t, person("site_engineer", "PRJ-A"), "Máy 03 hôm nay được cấp bao nhiêu dầu?"), "S3")
	wantAll(t, "S3 gỡ bằng phạm vi", eng.Reason, "may 3", "DZ-03", "EX-03", "phạm vi dữ liệu của người hỏi", "chọn EX-03")
	wantAll(t, "S3 ứng viên bị loại", strings.Join(eng.Rejected, "\n"), "1 ứng viên bị loại khi gỡ mơ hồ: ngoài phạm vi dữ liệu của người hỏi")

	// giám đốc xem được cả hai: lọc phạm vi không gỡ được, hỏi lại — cả hai ứng viên còn lại và vết nói đúng điều đó
	dir := stepOf(t, steps(t, person("director", ""), "Máy 03 hôm nay được cấp bao nhiêu dầu?"), "S3")
	wantAll(t, "S3 giữ nguyên mơ hồ", dir.Reason, "may 3", "DZ-03", "EX-03", "giữ nguyên mơ hồ")

	// thực thể rõ kèm khoá so khớp và nguyên văn trong câu
	clear := stepOf(t, steps(t, person("director", ""), "EX-03 hôm nay được cấp bao nhiêu dầu?"), "S3")
	wantAll(t, "S3 thực thể rõ", clear.Reason, `“EX-03”`, "ex 3", "Máy xúc EX-03")
}

// S4: ý định khớp đủ / khớp một phần (thiếu thuật ngữ nào) / ý định chung, và NGUỒN của từng ứng viên khi hỏi lại.
func TestVetDuS4YDinhDaXet(t *testing.T) {
	director := person("director", "")

	ok := stepOf(t, steps(t, director, "EX-03 hôm nay được cấp bao nhiêu dầu?"), "S4")
	wantAll(t, "S4 khớp đủ", ok.Reason, "fuel_issued", "được cấp", "khớp đủ", "ưu tiên 20", "chọn fuel_issued: ưu tiên cao nhất (20)")
	wantAll(t, "S4 ý định yếu bị loại", strings.Join(ok.Rejected, "\n"), "fuel_generic", "ưu tiên 10")

	// từ khoá khớp mà câu không có vật tư: khớp một phần, vết nói thiếu thuật ngữ nào
	part := stepOf(t, steps(t, director, "Hôm nay nhận bao nhiêu?"), "S4")
	wantAll(t, "S4 khớp một phần", strings.Join(part.Rejected, "\n"), "fuel_issued: khớp một phần — từ khoá nhận khớp nhưng câu thiếu thuật ngữ diesel_fuel")

	// ý định chung (D-OP-15 a2): không từ khoá riêng nào khớp mà câu vẫn đủ thực thể + thuật ngữ + cụm xác nhận
	gen := stepOf(t, steps(t, director, "Dầu xuất hôm nay của EX-03 là 3850 L đúng không?"), "S4")
	wantAll(t, "S4 ý định chung", gen.Reason, "ý định chung", "fuel_generic")

	// hai máy cùng loại tranh một khe, một trong hai viết bằng bí danh mơ hồ: ứng viên của bí danh đến từ S3, thực thể rõ do S4 gộp thêm
	mix := stepOf(t, steps(t, director, "RL-02 và máy 03 hôm nay được cấp bao nhiêu dầu?"), "S4")
	// mỗi ứng viên nói ĐÚNG nơi sinh ra nó: RL-02 là thực thể rõ do S4 gộp thêm; DZ-03 và EX-03 là ứng viên của cụm mơ hồ do S3 thấy
	wantAll(t, "S4 nguồn ứng viên", mix.Reason, "ứng viên hỏi lại RL-02 ← do S4 gộp thêm:", "ứng viên hỏi lại DZ-03 ← S3:", "ứng viên hỏi lại EX-03 ← S3:", `cụm mơ hồ “may 3”`)
	if strings.Contains(mix.Reason, "ứng viên hỏi lại DZ-03 ← do S4") || strings.Contains(mix.Reason, "ứng viên hỏi lại RL-02 ← S3") {
		t.Errorf("nguồn ứng viên bị đảo: %s", mix.Reason)
	}

	// chỉ một cụm mơ hồ: mọi ứng viên đến từ S3
	only := stepOf(t, steps(t, director, "Máy 03 hôm nay được cấp bao nhiêu dầu?"), "S4")
	if strings.Contains(only.Reason, "S4 gộp") {
		t.Errorf("không có thực thể nào do S4 gộp thêm: %s", only.Reason)
	}
	wantAll(t, "S4 mọi ứng viên từ S3", only.Reason, "DZ-03", "EX-03", "S3")
}

// S5: các bậc kiểm router đã đi qua theo thứ tự, route đã loại và vì sao.
func TestVetDuS5CacBacDaXet(t *testing.T) {
	ok := stepOf(t, steps(t, person("director", ""), "EX-03 hôm nay được cấp bao nhiêu dầu?"), "S5")
	wantAll(t, "S5 đã kiểm", ok.Reason, "đã kiểm theo thứ tự", "capability của ý định: qua", "phạm vi dữ liệu: qua", "thiếu khe bắt buộc: qua", "route khai của ý định — fuel_issued → SQL_QUERY → chọn SQL_QUERY")
	wantAll(t, "S5 route đã loại", strings.Join(ok.Rejected, "\n"), "REJECT (capability của ý định): không áp dụng — metrics.read: có", "REJECT (phạm vi dữ liệu): không áp dụng — EX-03", "CLARIFICATION", "HYBRID_RETRIEVAL")

	// bị chặn ở bậc phạm vi thực thể: route của ý định (SQL_QUERY) nằm trong phương án bị loại, kèm lý do chặn
	deny := stepOf(t, steps(t, person("site_engineer", "PRJ-B"), "EX-03 hôm nay được cấp bao nhiêu dầu?"), "S5")
	wantAll(t, "S5 bị chặn", deny.Reason, "ngoài phạm vi dữ liệu của người hỏi")
	wantAll(t, "S5 route của ý định bị loại", strings.Join(deny.Rejected, "\n"), "SQL_QUERY", "fuel_issued", "ngoài phạm vi")

	// câu xác nhận nêu hai con số: luồng hỏi ghi đè route của router — vết phải kể cả route ban đầu lẫn lý do ghi đè
	over := stepOf(t, steps(t, person("director", ""), "EX-03 hôm nay được cấp 3850 L đúng không, 4000 L thì sao?"), "S5")
	wantAll(t, "S5 ghi đè", over.Reason, "ghi đè", "nhiều con số khác nhau")
	wantAll(t, "S5 route ban đầu", strings.Join(over.Rejected, "\n"), "SQL_QUERY (route của router): bị ghi đè")
	s3 := stepOf(t, steps(t, person("director", ""), "EX-03 hôm nay được cấp 3850 L đúng không, 4000 L thì sao?"), "S3")
	wantAll(t, "S3 không kể ứng viên", s3.Reason, "không kể ứng viên thực thể trong câu hỏi lại")
}

// S7: bằng chứng được dùng và bị loại (thẩm quyền; quyền đọc tài liệu).
func TestVetDuS7BangChungDungVaBiLoai(t *testing.T) {
	s7 := stepOf(t, steps(t, person("director", ""), "EX-03 hôm nay được cấp bao nhiêu lít dầu?"), "S7")
	wantAll(t, "S7 dùng", s7.Reason, "nguồn warehouse_slip (thẩm quyền 1): 1 bằng chứng — tạo nên con số chính",
		"nguồn erp (thẩm quyền 2): 1 bằng chứng — không tạo nên con số chính", "phép tính của con số chính (sum(fact:", "quyền: phạm vi dữ liệu đã kiểm ở S5")
	wantAll(t, "S7 loại", strings.Join(s7.Rejected, "\n"), "nguồn erp = 3900 L không được chọn làm con số chính", "thẩm quyền thấp hơn nguồn được chọn")

	// tri thức: số đoạn truy xuất, đoạn được dẫn, và nói rõ quyền đọc do chặng truy xuất kiểm
	rag := stepOf(t, steps(t, person("director", ""), "Quy trình xử lý chênh lệch nhiên liệu là gì?"), "S7")
	wantAll(t, "S7 tri thức", rag.Reason, "SOP-FUEL-01", "đã truy xuất 1 đoạn", "quyền: ACL tài liệu", "dẫn nguồn: 1 đoạn được claim của bản nháp dẫn")

	// câu xác nhận kèm số của người hỏi và các nghĩa khác cùng kỳ: bằng chứng đi kèm được gọi đúng tên, không lẫn với số liệu
	cf := stepOf(t, steps(t, person("director", ""), "EX-03 hôm nay được cấp 3850 L đúng không?"), "S7")
	wantAll(t, "S7 câu xác nhận", cf.Reason, "question", "người hỏi", "chỉ số khác")
}

// S9: cổng phán quyết của câu xác nhận — phán quyết hay không, và vì sao.
func TestVetDuS9CongPhanQuyet(t *testing.T) {
	director := person("director", "")
	yes := stepOf(t, steps(t, director, "EX-03 hôm nay được cấp 3850 L đúng không?"), "S9")
	wantAll(t, "S9 Đúng", yes.Reason, "cổng phán quyết", "được phán quyết", "→ \"Đúng.\" (confirm_yes)")
	wantAll(t, "S9 Đúng: phán quyết bị loại", strings.Join(yes.Rejected, "\n"), "(confirm_no): bị loại", "confirm_withheld")
	no := stepOf(t, steps(t, director, "EX-03 hôm nay được cấp 4000 L đúng không?"), "S9")
	wantAll(t, "S9 Không", no.Reason, "cổng phán quyết", "được phán quyết", "→ \"Không.\" (confirm_no)")
	wantAll(t, "S9 Không: phán quyết bị loại", strings.Join(no.Rejected, "\n"), "(confirm_yes): bị loại", "confirm_withheld")

	// giữ phán quyết: chiều chưa giải được gọi đúng tên, kèm nghĩa; phán quyết bị loại nằm trong phương án bị loại
	held := stepOf(t, steps(t, director, "EX-03 được cấp 3850 L đúng không?"), "S9")
	wantAll(t, "S9 giữ phán quyết", held.Reason, "cổng phán quyết", "không phán quyết", "period", "mốc thời gian")
	wantAll(t, "S9 phán quyết bị loại", strings.Join(held.Rejected, "\n"), "Đúng.", "Không.")
	resid := stepOf(t, steps(t, director, "EX-03 hôm nay được cấp ~4000 L đúng không?"), "S9")
	wantAll(t, "S9 phần dư", resid.Reason, "không phán quyết", "residual")

	// có cụm xác nhận mà không có con số nào để xác nhận, hay chỉ số là phép tính: cổng không áp dụng và nói đúng vì sao
	noNum := stepOf(t, steps(t, director, "EX-03 hôm nay được cấp bao nhiêu lít dầu, đúng không?"), "S9")
	wantAll(t, "S9 không có con số", noNum.Reason, "không áp dụng — câu có cụm xác nhận nhưng không có đúng một con số")
	calc := stepOf(t, steps(t, director, "EX-03 hôm nay chênh lệch dầu bao nhiêu, đúng không?"), "S9")
	wantAll(t, "S9 chỉ số dẫn xuất", calc.Reason, "không áp dụng — issue_vs_measured là chỉ số dẫn xuất")

	// không phải câu xác nhận: nói rõ cổng không áp dụng, không im lặng
	plain := stepOf(t, steps(t, director, "EX-03 hôm nay được cấp bao nhiêu lít dầu?"), "S9")
	wantAll(t, "S9 không áp dụng", plain.Reason, "cổng phán quyết", "không áp dụng")
}
