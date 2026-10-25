// Lệnh so-than: so THÂN test giữa hai cây mã (F-034). scripts/cong/so_ten.py chỉ so (gói, tên test) + trạng thái, nên đảo kỳ vọng mà
// giữ tên (ddf4037) vẫn ra "0 đổi". Ở đây mọi khai báo mức gói của tệp *_test.go — hàm Test* / Benchmark* / Fuzz*, và cả bảng kỳ vọng,
// hàm trợ giúp — được tách bằng go/parser (không vỡ vì ngoặc trong chuỗi) rồi so theo DÃY TOKEN (go/scanner, bỏ comment và dấu chấm
// phẩy tự chèn): đổi comment, định dạng hay chỗ đặt không tính; đổi một chữ trong thân thì tính.
//
// Dùng (từ gốc repo; cây gốc dựng bằng git worktree add --detach <thư mục> <mốc>):
//
//	go run ./scripts/cong/so-than <cây gốc> <cây mới> [tệp đổi có chủ ý]
//
// Tệp đổi có chủ ý: mỗi dòng "<tên> <lý do>" — tên là tên hàm test / khai báo (khớp ở mọi gói); dòng trống và dòng bắt đầu bằng # bỏ
// qua. Mã thoát: 0 mọi khai báo đổi đều có trong tệp · 1 có khai báo đổi mà không khai, hoặc mục khai không đổi (thừa) · 2 lỗi dùng,
// mục không lý do, cây không có test, hay không đọc / không dịch được tệp.
package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

// khai là một khai báo mức gói của tệp _test.go: thân dạng dãy token, và có phải hàm test hay không.
type khai struct {
	than string
	test bool
}

func run(args []string, out, errw io.Writer) int {
	if len(args) < 2 || len(args) > 3 {
		fmt.Fprintln(errw, "dùng: so-than <cây gốc> <cây mới> [tệp đổi có chủ ý]")
		return 2
	}
	var cay [2]map[string]khai
	for i, root := range args[:2] {
		m, err := docCay(root)
		if err != nil {
			fmt.Fprintln(errw, err)
			return 2
		}
		if len(m) == 0 {
			fmt.Fprintf(errw, "%s: không có khai báo nào trong tệp _test.go — rỗng không phải xanh\n", root)
			return 2
		}
		cay[i] = m
	}
	coY := map[string]string{}
	if len(args) == 3 {
		var err error
		if coY, err = docCoY(args[2]); err != nil {
			fmt.Fprintln(errw, err)
			return 2
		}
	}
	a, b := cay[0], cay[1]
	var doi, mat, moi []string
	for k, x := range a {
		if y, ok := b[k]; !ok {
			mat = append(mat, k)
		} else if x.than != y.than {
			doi = append(doi, k)
		}
	}
	for k := range b {
		if _, ok := a[k]; !ok {
			moi = append(moi, k)
		}
	}
	sort.Strings(doi)
	sort.Strings(mat)
	sort.Strings(moi)
	soTest := func(m map[string]khai) (n int) {
		for _, x := range m {
			if x.test {
				n++
			}
		}
		return n
	}
	fmt.Fprintf(out, "gốc %d khai báo (%d test) · mới %d khai báo (%d test) · đổi thân %d · mất %d · mới %d\n",
		len(a), soTest(a), len(b), soTest(b), len(doi), len(mat), len(moi))
	khongKhai, dung := 0, map[string]bool{}
	for _, k := range doi {
		ten := k[strings.LastIndexByte(k, ' ')+1:]
		if ly, ok := coY[ten]; ok {
			dung[ten] = true
			fmt.Fprintf(out, "  đổi có chủ ý: %s — %s\n", k, ly)
			continue
		}
		khongKhai++
		if b[k].test {
			fmt.Fprintf(out, "  ĐỔI THÂN (test): %s\n", k)
		} else {
			fmt.Fprintf(out, "  ĐỔI (khai báo khác): %s\n", k)
		}
	}
	for _, k := range mat {
		fmt.Fprintf(out, "  mất: %s\n", k)
	}
	for _, k := range moi {
		fmt.Fprintf(out, "  mới: %s\n", k)
	}
	var thua []string
	for ten := range coY {
		if !dung[ten] {
			thua = append(thua, ten)
		}
	}
	sort.Strings(thua)
	for _, ten := range thua {
		fmt.Fprintf(out, "  THỪA: %s khai đổi có chủ ý mà không đổi thân — bỏ mục\n", ten)
	}
	if khongKhai > 0 || len(thua) > 0 {
		fmt.Fprintf(out, "ĐỎ: %d khai báo đổi thân không khai, %d mục khai thừa\n", khongKhai, len(thua))
		return 1
	}
	fmt.Fprintln(out, "XANH: không khai báo test nào đổi thân ngoài tệp đổi có chủ ý")
	return 0
}

// docCay đọc mọi tệp _test.go dưới root (bỏ thư mục ẩn và testdata/, như công cụ go) thành bản đồ "<thư mục gói> <tên>" → khai báo.
// Hai khai báo cùng khoá (gói p và p_test cùng thư mục, nhiều hàm init) được nối thân: đổi ở bất kỳ bản nào vẫn hiện.
func docCay(root string) (map[string]khai, error) {
	out := map[string]khai{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && (strings.HasPrefix(d.Name(), ".") || d.Name() == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		dir, _ := filepath.Rel(root, filepath.Dir(path))
		them := func(ten string, test bool, n ast.Node) {
			if ten == "_" {
				return
			}
			k := filepath.ToSlash(dir) + " " + ten
			x := out[k]
			x.than += dayToken(src[fset.Position(n.Pos()).Offset:fset.Position(n.End()).Offset])
			x.test = x.test || test
			out[k] = x
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				ten := d.Name.Name
				if d.Recv != nil && len(d.Recv.List) > 0 {
					ten = tenKieu(d.Recv.List[0].Type) + "." + ten
				}
				them(ten, d.Recv == nil && laTest(ten), d)
			case *ast.GenDecl:
				for _, s := range d.Specs {
					switch s := s.(type) {
					case *ast.ValueSpec:
						for _, id := range s.Names {
							them(id.Name, false, s)
						}
					case *ast.TypeSpec:
						them(s.Name.Name, false, s)
					}
				}
			}
		}
		return nil
	})
	return out, err
}

// laTest theo luật của go test: tiền tố Test / Benchmark / Fuzz, chữ ngay sau (nếu có) không phải chữ thường.
func laTest(ten string) bool {
	for _, p := range []string{"Test", "Benchmark", "Fuzz"} {
		if rest, ok := strings.CutPrefix(ten, p); ok {
			r, _ := utf8.DecodeRuneInString(rest)
			return rest == "" || !unicode.IsLower(r)
		}
	}
	return false
}

// tenKieu trả tên kiểu nhận của phương thức (bỏ * và tham số kiểu).
func tenKieu(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.StarExpr:
		return tenKieu(x.X)
	case *ast.IndexExpr:
		return tenKieu(x.X)
	case *ast.IndexListExpr:
		return tenKieu(x.X)
	case *ast.Ident:
		return x.Name
	}
	return fmt.Sprintf("%T", e)
}

// dayToken: dãy token của src, bỏ comment và dấu chấm phẩy tự chèn ở cuối dòng — đổi định dạng hay comment không đổi dãy.
func dayToken(src []byte) string {
	fset := token.NewFileSet()
	var s scanner.Scanner
	s.Init(fset.AddFile("", -1, len(src)), src, nil, 0)
	var sb strings.Builder
	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.SEMICOLON && lit == "\n" {
			continue
		}
		if lit == "" {
			lit = tok.String()
		}
		sb.WriteString(lit)
		sb.WriteByte(' ')
	}
	return sb.String()
}

// docCoY đọc tệp đổi có chủ ý: "<tên> <lý do>" mỗi dòng.
func docCoY(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]string{}
	sc := bufio.NewScanner(f)
	for so := 1; sc.Scan(); so++ {
		dong := strings.TrimSpace(sc.Text())
		if dong == "" || strings.HasPrefix(dong, "#") {
			continue
		}
		ten, ly, _ := strings.Cut(dong, " ")
		if strings.TrimSpace(ly) == "" {
			return nil, fmt.Errorf("%s:%d: cần '<tên> <lý do>', gặp %q", path, so, dong)
		}
		out[ten] = strings.TrimSpace(ly)
	}
	return out, sc.Err()
}
