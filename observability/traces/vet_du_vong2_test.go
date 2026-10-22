package traces_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// R2-03 vòng 2 (soát vòng 1: F1 an ninh, F3 kích thước, F4 phạm vi). Mọi test ở đây chạy lượt hỏi THẬT rồi đọc vết ở dạng bền
// (NewRecord → SanitizeRecord), đúng thứ được lưu và được GET /v1/trace/{id} trả về.

// secretCases là các dạng bí mật mà người soát vòng 1 đã thử (và vài dạng khó hơn): mỗi ca là MỘT mảnh chữ chứa bí mật, đặt vào mệnh đề
// hỏi của một câu nhiều mệnh đề, kèm giá trị bí mật không được phép còn lại ở BẤT KỲ chỗ nào của vết.
var secretCases = []struct{ name, piece, value string }{
	{"json api_key nháy kép", `{"api_key": "abc123xyz"}`, "abc123xyz"},
	{"password nháy kép", `password="hunter2pw"`, "hunter2pw"},
	{"json secret không khoảng trắng", `{"secret":"s3cr3tVal"}`, "s3cr3tVal"},
	{"token có dấu hai chấm", `token: "tok3nVal"`, "tok3nVal"},
	{"cookie", `Cookie: sid=cookieVal99`, "cookieVal99"},
	{"authorization bearer", `Authorization: "Bearer bearerVal123456"`, "bearerVal123456"},
	{"bearer đứng riêng", `Bearer bearerAlone987654`, "bearerAlone987654"},
	// dấu chấm câu nằm TRONG bí mật cắt mệnh đề hỏi ngay giữa giá trị: đoạn được trích không còn tên khoá nên mẫu che không nhận ra nó
	{"bí mật chứa dấu chấm câu", `token: "abc. tailSecret77"`, "tailSecret77"},
	{"bí mật chứa dấu phẩy và khoảng trắng", `password="alpha, betaSecret88"`, "betaSecret88"},
	// nháy kiểu chữ in trong bí mật không được phá cặp dấu bao chữ của vết
	{"bí mật chứa nháy kiểu chữ in", `{"api_key": "q”x99leak"}`, "x99leak"},
}

// secretQuestions đặt mảnh bí mật vào hai vị trí của mệnh đề hỏi: SAU các từ khoá, và TRƯỚC chúng (khi đó dấu chấm câu trong bí mật cắt mệnh đề
// hỏi ngay giữa giá trị mà mệnh đề vẫn còn từ khoá ý định, nên đoạn đuôi của bí mật bị trích ra không kèm tên khoá).
func secretQuestions(piece string) map[string]string {
	return map[string]string{
		"sau từ khoá":   "đội báo 4k, báo cáo ghi 4 m3, vậy EX-03 được cấp 4k dầu " + piece + " đúng không?",
		"trước từ khoá": "đội báo 4k, báo cáo ghi 4 m3, vậy EX-03 " + piece + " được cấp 4k dầu đúng không?",
	}
}

// F1 (S1): bí mật có nháy kép nằm trong chữ tự do của người hỏi (mệnh đề hỏi, chữ chưa giải thích, cụm đã nhận dạng) không được lọt vào
// bản ghi bền, JSON lẫn bảng chữ — kể cả khi mẫu che của Sanitize không nhận ra đoạn chữ bị cắt ra khỏi câu.
func TestVetDuBiMatCoNhayKepKhongLot(t *testing.T) {
	for _, c := range secretCases {
		for place, q := range secretQuestions(c.piece) {
			t.Run(c.name+"/"+place, func(t *testing.T) {
				turn := realTurn(t, realAsk(t), person("director", ""), q)
				if why, _ := json.Marshal(turn.Decision.Why); strings.Contains(string(why), c.value) {
					t.Errorf("Handle đã đưa %q vào giải thích TRƯỚC khi bản ghi được làm sạch: %s", c.value, why)
				}
				clean, err := traces.SanitizeRecord(traces.NewRecord(turn))
				if err != nil {
					t.Fatal(err)
				}
				b, _ := json.Marshal(clean)
				if strings.Contains(string(b), c.value) {
					t.Errorf("%q còn trong JSON bản ghi bền: %s", c.value, b)
				}
				if tab := traces.FormatTable(clean); strings.Contains(tab, c.value) {
					t.Errorf("%q còn trong bảng chữ:\n%s", c.value, tab)
				}
			})
		}
	}
}

// Câu KHÔNG chứa bí mật thì mệnh đề hỏi được trích nguyên văn trong cặp “…” (không thoát nháy): người đọc vết thấy đúng mệnh đề đã quyết
// ý định. Đây là mặt dương của F1: bỏ phần trích (hay thoát nháy kiểu %q) thì test đỏ.
func TestVetDuMenhDeHoiDuocTrichNguyenVan(t *testing.T) {
	s4 := stepOf(t, steps(t, person("director", ""), "đội báo 4k, báo cáo ghi 4 m3, vậy EX-03 được cấp 4k dầu đúng không?"), "S4")
	want := "từ khoá lấy trong mệnh đề hỏi “vậy EX-03 được cấp 4k dầu đúng không” (mệnh đề chứa cụm xác nhận quyết định ý định, các mệnh đề trước là ngữ cảnh)"
	if !slices.Contains(strings.Split(s4.Reason, "; "), want) {
		t.Errorf("thiếu dòng %q trong lý do của S4:\n%s", want, s4.Reason)
	}
}

// bigDirectory: n máy cùng bí danh "máy 03", chia đều cho hai dự án — danh mục tổng hợp cho phép đo độ phình theo số ứng viên của bí danh.
func bigDirectory(n int) entityresolution.MemoryDirectory {
	dir := entityresolution.MemoryDirectory{
		{ID: "PRJ-A", Type: "project", Key: "PRJ-A", Name: "Cầu A"},
		{ID: "PRJ-B", Type: "project", Key: "PRJ-B", Name: "Đường B"},
	}
	for i := range n {
		parent := "PRJ-A"
		if i%2 == 1 {
			parent = "PRJ-B"
		}
		id := fmt.Sprintf("MX-%02d", i+1)
		dir = append(dir, entityresolution.Entity{ID: id, Type: "equipment", Key: id, Name: "Máy " + id, Parent: parent,
			Aliases: []entityresolution.Alias{{Text: "máy 03", Source: "seed"}}})
	}
	return dir
}

// whyBytes là tổng byte của mọi dòng giải thích (lý do + phương án bị loại) mà người đọc vết thấy trong các bước.
func whyBytes(steps []traces.Step) (n int) {
	for _, s := range steps {
		n += len(s.Reason)
		for _, r := range s.Rejected {
			n += len(r)
		}
	}
	return n
}

// maxWhyBytes là trần byte của phần giải thích trong MỘT bản ghi, bất kể câu dài bao nhiêu hay bí danh có bao nhiêu ứng viên (trần của
// câu hỏi là 2000 ký tự). Phần giải thích không được phình theo (số lần nhắc × số ứng viên của bí danh).
const maxWhyBytes = 12 << 10

// F3 (S3): câu dài nhắc đi nhắc lại một bí danh có nhiều ứng viên không được làm vết phình: các dòng giống nhau gộp một dòng kèm số lần,
// danh sách dài cắt kiểu "và N ứng viên khác".
func TestVetDuCanTrenKichThuocKhiCauDaiVaBiDanhNhieuUngVien(t *testing.T) {
	q := strings.Repeat("máy 03 ", 150) + "được cấp bao nhiêu dầu?"
	for _, pr := range []contracts.Principal{person("director", ""), person("site_engineer", "PRJ-A")} {
		t.Run(pr.Roles[0], func(t *testing.T) {
			a := realAsk(t)
			a.Deps.Directory = bigDirectory(42)
			turn := realTurn(t, a, pr, q)
			rec := traces.NewRecord(turn)
			t.Logf("phần giải thích của vết: %d byte (trần %d); câu %d ký tự, bí danh 42 ứng viên", whyBytes(rec.Steps), maxWhyBytes, len([]rune(q)))
			if n := whyBytes(rec.Steps); n > maxWhyBytes {
				t.Errorf("phần giải thích của vết dài %d byte, trần %d (câu %d ký tự, bí danh 42 ứng viên)", n, maxWhyBytes, len([]rune(q)))
			}
			s3 := stepOf(t, rec.Steps, "S3")
			if strings.Count(s3.Reason, "khớp 42 thực thể") > 1 {
				t.Errorf("dòng cụm mơ hồ phải gộp một dòng kèm số lần, không in lặp %d lần", strings.Count(s3.Reason, "khớp 42 thực thể"))
			}
			if !strings.Contains(s3.Reason, "ứng viên khác") {
				t.Errorf("danh sách 42 ứng viên phải bị cắt kiểu \"và N ứng viên khác\":\n%.600s", s3.Reason)
			}
		})
	}
}

// F4 (S3): vết do chính người hỏi đọc (traceGet) không kể tên / loại của thực thể ngoài phạm vi và không kể trạng thái phạm vi của từng
// ứng viên; người có phạm vi dữ liệu không hạn chế (toàn quyền dữ liệu) đọc đủ.
func TestVetDuNguoiHoiKhongThayTenVaPhamViCuaUngVienNgoaiPhamVi(t *testing.T) {
	eng := person("site_engineer", "PRJ-A")
	for _, q := range []string{"Máy 03 hôm nay được cấp bao nhiêu dầu?", "DZ-03 hôm nay được cấp bao nhiêu dầu?", "RL-02 và máy 03 hôm nay được cấp bao nhiêu dầu?"} {
		t.Run(q, func(t *testing.T) {
			rec := traces.NewRecord(realTurn(t, realAsk(t), eng, q))
			for _, st := range rec.Steps {
				if st.Stage != "S3" && st.Stage != "S4" {
					continue
				}
				all := text(st)
				for _, leak := range []string{"Máy ủi", "loại equipment", "loại project"} {
					if strings.Contains(all, leak) {
						t.Errorf("%s kể tên / loại của thực thể mà người hỏi (PRJ-A) không có quyền: %q trong:\n%s", st.Stage, leak, all)
					}
				}
				// dòng nào nhắc DZ-03 (máy của PRJ-B, ngoài phạm vi) cũng không được nói gì về phạm vi hay việc nó bị loại
				for _, line := range slices.Concat(strings.Split(st.Reason, "; "), st.Rejected) {
					if strings.Contains(line, "DZ-03") && (strings.Contains(line, "phạm vi") || strings.Contains(line, "bị loại")) {
						t.Errorf("%s kể trạng thái phạm vi của ứng viên DZ-03: %q", st.Stage, line)
					}
				}
			}
		})
	}
	// giám đốc (phạm vi dữ liệu không hạn chế) đọc đủ tên và loại
	dir := stepOf(t, steps(t, person("director", ""), "EX-03 hôm nay được cấp bao nhiêu dầu?"), "S3")
	if !strings.Contains(dir.Reason, "khớp EX-03 (Máy xúc EX-03, loại equipment)") {
		t.Errorf("người có toàn quyền dữ liệu phải đọc đủ tên và loại:\n%s", dir.Reason)
	}
}
