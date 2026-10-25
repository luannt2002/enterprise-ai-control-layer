package meta

import (
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// Cổng hướng import (F-029): phần chung của việc nạp pack không import gói nào của repo — nhất là data/ và intelligence/ (động cơ
// QUERY). Thêm một import nội bộ vào đây là kéo động cơ vào mọi pack, kể cả pack chỉ-tài-liệu.
func TestMetaKhongImportGoiNoiBo(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), e.Name(), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		n++
		for _, im := range f.Imports {
			if p, _ := strconv.Unquote(im.Path.Value); strings.HasPrefix(p, "github.com/luannt2002/enterprise-ai-control-layer/") {
				t.Errorf("%s import %s: phần meta của pack không được phụ thuộc gói nội bộ", e.Name(), p)
			}
		}
	}
	if n == 0 {
		t.Fatal("không thấy tệp .go nào: cổng rỗng không phải xanh")
	}
}
