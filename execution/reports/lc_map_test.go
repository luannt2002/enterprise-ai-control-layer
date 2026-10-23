package reports

// Bảng "lớp ca → test" (HOP-DONG §10) của các bước 2, 3a, 3b và kho PG. Cổng cấu trúc: mỗi lớp ca thuộc đơn vị này phải có ít nhất một test mang mã LCnn trong tên. Nếu ai xoá hoặc
// đổi tên test làm một lớp ca mất chỗ đo thì test này đỏ. Lớp ca của bước 4 đến 9 (cờ loại từ chữ, thời gian, quyền người gửi, phục hồi đầy đủ, báo cáo xuất, nhánh form) CHƯA có ở đây.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// lcOfUnit: các lớp ca mà đơn vị này (kho bản ghi, sổ lần giao, TR1, TR2, TR3, quyết định của người, bản rời nhóm, kho PG) chịu trách nhiệm đo.
var lcOfUnit = []string{
	"LC01", "LC02", "LC03", "LC04", "LC05", "LC06", "LC07", "LC08", // bảo toàn, cách ly lưu bền (phần kho)
	"LC15", "LC16", // APPLY và REJECT đính chính (cơ chế quyết định; nhận dạng đính chính từ chữ là bước 4)
	"LC21", "LC22", "LC23", "LC24", "LC25", "LC26", "LC27", "LC28", // trùng: TR1, TR2, TR3
	"LC53", "LC63", "LC64", "LC65", "LC66", "LC67", "LC70", // phục hồi phần kho: khởi động lại, đảo thứ tự, đồng thời, lỗi giữa giao dịch
	"LC74", "LC75", "LC77", "LC78", "LC79", "LC80", // TR3: đồng thời, đến muộn, quyết định không miễn TR3, vân tay không có sent_at
	"LC81", "LC83", "LC84", // bản rời nhóm (R6-01), màn hình cũ, bản được giữ rời nhóm sau quyết định
}

var lcToken = regexp.MustCompile(`LC\d{2}`)

func testNamesByLC(t *testing.T) map[string][]string {
	t.Helper()
	files, err := filepath.Glob("*_test.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("không đọc được tệp test: %v", err)
	}
	out := map[string][]string{}
	fset := token.NewFileSet()
	for _, f := range files {
		af, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for _, d := range af.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Recv != nil || !strings.HasPrefix(fd.Name.Name, "Test") {
				continue
			}
			seen := map[string]bool{}
			for _, lc := range lcToken.FindAllString(fd.Name.Name, -1) {
				if !seen[lc] {
					seen[lc] = true
					out[lc] = append(out[lc], fd.Name.Name)
				}
			}
		}
	}
	return out
}

func TestLCMap_MoiLopCaCuaDonViCoItNhatMotTest(t *testing.T) {
	byLC := testNamesByLC(t)
	for _, lc := range lcOfUnit {
		if len(byLC[lc]) == 0 {
			t.Errorf("lớp ca %s không có test nào mang mã trong tên: lớp ca mất chỗ đo", lc)
		}
	}
	var keys []string
	for k := range byLC {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Logf("%s → %s", k, strings.Join(byLC[k], ", "))
	}
	// mọi mã LC trong tên test phải thuộc §10 (LC01 đến LC84): chặn gõ nhầm mã
	for _, k := range keys {
		var n int
		for _, c := range k[2:] {
			n = n*10 + int(c-'0')
		}
		if n < 1 || n > 84 {
			t.Errorf("mã %s trong tên test %v nằm ngoài LC01 đến LC84 của §10", k, byLC[k])
		}
	}
}
