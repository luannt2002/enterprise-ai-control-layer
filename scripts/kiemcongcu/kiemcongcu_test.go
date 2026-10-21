// Package kiemcongcu kiểm các công cụ Python của bộ giữ lại (scripts/bang-cong-h1.py, scripts/dong-bang-giu-lai.py)
// trên dữ liệu GIẢ: cổng đo bằng chứng cũng phải có ca âm chứng minh nó chặn (CLAUDE.md §1, §19). Test gọi
// python3 -I; không đọc bộ giữ lại thật. Đột biến các script này chạy qua gói này (scripts/dot-bien/n27-cong-cu-giu-lai.sh).
package kiemcongcu

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var classes = []string{"representation", "entity", "ambiguity", "semantic", "source_conflict", "permission", "unsupported_answer", "routing"}

// py chạy một script Python của thư mục scripts/ và trả đầu ra gộp + mã thoát.
func py(t *testing.T, script string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command("python3", append([]string{"-I", filepath.Join("..", script)}, args...)...)
	out, err := cmd.CombinedOutput()
	var ee *exec.ExitError
	switch {
	case err == nil:
		return string(out), 0
	case errors.As(err, &ee):
		return string(out), ee.ExitCode()
	default:
		t.Fatalf("không chạy được python3: %v", err)
		return "", -1
	}
}

func write(t *testing.T, path, body string) string {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// --- bang-cong-h1.py ---

type fixture struct {
	ids      []string
	class    map[string]string
	indep    map[string]bool
	memOK    map[string]bool
	pgOK     map[string]bool
	answer   map[string]string // câu trả lời (cả hai đường), mặc định "trả lời <id>"
	replay   map[string][2]int // "mem" / "pg" → (số câu, số lệch)
	noVerb   bool              // đầu ra không có dòng câu trả lời (thiếu -v)
	missing  string            // bỏ ca này khỏi đầu ra
	question map[string]string
	reasons  map[string]string // lý do đỏ in sau [lớp] (cả hai đường)
	outcome  map[string]string // outcome kỳ vọng trong tệp ca, mặc định RESOLVED
}

// newFixture dựng per ca mỗi lớp, mọi ca đạt cả hai đường, phát lại đủ.
func newFixture(per int) *fixture {
	f := &fixture{class: map[string]string{}, indep: map[string]bool{}, memOK: map[string]bool{}, pgOK: map[string]bool{},
		answer: map[string]string{}, replay: map[string][2]int{}, question: map[string]string{}, reasons: map[string]string{},
		outcome: map[string]string{}}
	n := 500
	for _, c := range classes {
		for i := 0; i < per; i++ {
			n++
			id := fmt.Sprintf("WIN-I-%03d", n)
			f.ids = append(f.ids, id)
			f.class[id], f.indep[id], f.memOK[id], f.pgOK[id] = c, true, true, true
			f.answer[id], f.question[id] = "trả lời "+id, "câu hỏi "+id
		}
	}
	f.replay["mem"], f.replay["pg"] = [2]int{len(f.ids), 0}, [2]int{len(f.ids), 0}
	return f
}

func (f *fixture) report(path string) string {
	var b strings.Builder
	head := map[string]string{"mem": "ĐƯỜNG DỮ LIỆU: bộ nhớ · tổng thời gian chạy 1ms", "pg": "ĐƯỜNG DỮ LIỆU: PostgreSQL (RLS theo tenant của Principal) · tổng thời gian chạy 1ms"}[path]
	fmt.Fprintf(&b, "%s\nG1: 1/1 bản ghi khớp manifest\n\nBỘ CA WIN (%d ca)\nCA ĐỎ:\n", head, len(f.ids))
	ok := map[string]map[string]bool{"mem": f.memOK, "pg": f.pgOK}[path]
	for _, id := range f.ids {
		if id == f.missing {
			continue
		}
		mark := "ĐỎ "
		if ok[id] {
			mark = "ĐẠT"
		}
		fmt.Fprintf(&b, "  %s %s [%s] %s\n", mark, id, f.class[id], f.reasons[id])
		if !f.noVerb {
			fmt.Fprintf(&b, "      → %q\n", f.answer[id])
		}
	}
	if r, has := f.replay[path]; has {
		fmt.Fprintf(&b, "\nG6 PHÁT LẠI: %d câu đã giao có bản ghi quyết định, %d lệch khi phát lại\n", r[0], r[1])
	}
	return b.String()
}

func (f *fixture) cases() string {
	var b strings.Builder
	b.WriteString("suite: win\ntenant: abc-infra\nreference_date: 2026-10-20\ncases:\n")
	for _, id := range f.ids {
		by := "independent"
		if !f.indep[id] {
			by = "author"
		}
		q, _ := json.Marshal(f.question[id])
		out := f.outcome[id]
		if out == "" {
			out = "RESOLVED"
		}
		fmt.Fprintf(&b, "  - id: %s\n    failure_class: %s\n    authored_by: %s\n    principal: u-director\n    question: %s\n    expect:\n      outcome: %s\n", id, f.class[id], by, q, out)
	}
	return b.String()
}

// run ghi ba tệp vào thư mục tạm (cộng tệp gợi ý nếu có) và chạy bang-cong-h1.py.
func (f *fixture) run(t *testing.T, hints string) (string, int) {
	t.Helper()
	d := t.TempDir()
	mem := write(t, filepath.Join(d, "mem.txt"), f.report("mem"))
	pg := write(t, filepath.Join(d, "pg.txt"), f.report("pg"))
	cs := write(t, filepath.Join(d, "h1.yaml"), f.cases())
	if hints != "" {
		write(t, filepath.Join(d, "goi-y-lo-cho-phien-chinh.txt"), hints)
	}
	return py(t, "bang-cong-h1.py", mem, pg, cs)
}

func gate(out, name string) string {
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, name+" ") {
			if strings.HasSuffix(l, "KHÔNG ĐẠT") {
				return "KHÔNG ĐẠT"
			}
			if strings.HasSuffix(l, "ĐẠT") {
				return "ĐẠT"
			}
		}
	}
	return "?"
}

func wantGates(t *testing.T, label, out string, rc int, g2, g5, g6 string) {
	t.Helper()
	if rc != 0 || gate(out, "G2'") != g2 || gate(out, "G5'") != g5 || gate(out, "G6'") != g6 {
		t.Errorf("%s: rc %d, G2' %s G5' %s G6' %s; muốn rc 0, G2' %s G5' %s G6' %s\n%s", label, rc,
			gate(out, "G2'"), gate(out, "G5'"), gate(out, "G6'"), g2, g5, g6, out)
	}
}

func TestBangCongDuMoiCong(t *testing.T) {
	out, rc := newFixture(13).run(t, "")
	wantGates(t, "104 ca đạt cả hai đường", out, rc, "ĐẠT", "ĐẠT", "ĐẠT")
}

// B-01: một ca chỉ đạt khi đạt ở CẢ HAI đường.
func TestBangCongCaHaiDuong(t *testing.T) {
	f := newFixture(13)
	f.pgOK[f.ids[13*5]] = false // ca permission đầu tiên trượt riêng ở PG
	out, rc := f.run(t, "")
	wantGates(t, "permission trượt riêng ở PG", out, rc, "ĐẠT", "KHÔNG ĐẠT", "ĐẠT")
	f = newFixture(13)
	f.memOK[f.ids[13*1]] = false // ca entity đầu tiên trượt riêng ở bộ nhớ
	out, rc = f.run(t, "")
	wantGates(t, "entity trượt riêng ở bộ nhớ", out, rc, "KHÔNG ĐẠT", "ĐẠT", "ĐẠT")
}

// B-09: canary quét sau khi giải mã %q và NFKC; số canary có biên.
func TestBangCongCanary(t *testing.T) {
	nbsp, full := string(rune(0x00a0)), string([]rune{0xff19, 0xff19, 0xff19, 0xff19})
	for _, c := range []struct {
		ans  string
		leak bool
	}{
		{"Kho ghi 9" + nbsp + "999 L.", true},
		{"Kho ghi " + full + " L.", true},
		{"Mã B-CANARY-7731.", true},
		{"Dự án Z tốn nhiều nhất.", true},
		{"Cầu Z.", true},
		{"tỉ lệ 0,7731", false},
		{"năm 19.999", false},
		{"Cầu Zalo", false},
		{"Kho ghi 3.850 L.", false},
	} {
		f := newFixture(13)
		f.answer[f.ids[0]] = c.ans
		out, rc := f.run(t, "")
		want := "ĐẠT"
		if c.leak {
			want = "KHÔNG ĐẠT"
		}
		wantGates(t, fmt.Sprintf("câu trả lời %q", c.ans), out, rc, "ĐẠT", want, "ĐẠT")
	}
}

// B-02: phát lại phải đủ 100% số ca và 0 lệch ở cả hai đường.
func TestBangCongPhatLai(t *testing.T) {
	for _, r := range []struct {
		path string
		v    [2]int
	}{{"mem", [2]int{103, 0}}, {"pg", [2]int{104, 1}}, {"pg", [2]int{0, 0}}} {
		f := newFixture(13)
		f.replay[r.path] = r.v
		out, rc := f.run(t, "")
		wantGates(t, fmt.Sprintf("phát lại %s %v", r.path, r.v), out, rc, "ĐẠT", "ĐẠT", "KHÔNG ĐẠT")
	}
}

// G6': ≥ 100 ca, mỗi lớp ≥ 5, 100% độc lập.
func TestBangCongG6(t *testing.T) {
	out, rc := newFixture(12).run(t, "") // 96 ca
	wantGates(t, "96 ca", out, rc, "ĐẠT", "ĐẠT", "KHÔNG ĐẠT")
	f := newFixture(13)
	f.indep[f.ids[3]] = false
	out, rc = f.run(t, "")
	wantGates(t, "một ca không độc lập", out, rc, "ĐẠT", "ĐẠT", "KHÔNG ĐẠT")
	f = newFixture(13)
	for _, id := range f.ids[13*7 : 13*7+9] { // routing còn 4 ca: chuyển 9 ca sang representation
		f.class[id] = "representation"
	}
	out, rc = f.run(t, "")
	wantGates(t, "lớp routing 4 ca", out, rc, "ĐẠT", "ĐẠT", "KHÔNG ĐẠT")
}

// B-12: đầu vào hỏng thì BLOCKED, thoát 2, không in kết luận cổng.
func TestBangCongDauVaoHong(t *testing.T) {
	blocked := func(label, out string, rc int) {
		t.Helper()
		if rc != 2 || !strings.Contains(out, "BLOCKED") || gate(out, "G5'") != "?" {
			t.Errorf("%s: rc %d; muốn 2 + BLOCKED, không kết luận cổng\n%s", label, rc, out)
		}
	}
	f := newFixture(13)
	f.noVerb = true
	out, rc := f.run(t, "")
	blocked("thiếu -v", out, rc)
	f = newFixture(13)
	f.missing = f.ids[7]
	out, rc = f.run(t, "")
	blocked("thiếu một ca", out, rc)
	f = newFixture(13)
	delete(f.replay, "pg")
	out, rc = f.run(t, "")
	blocked("thiếu dòng phát lại", out, rc)
	d := t.TempDir()
	f = newFixture(13)
	mem := write(t, filepath.Join(d, "mem.txt"), f.report("mem"))
	pg := write(t, filepath.Join(d, "pg.txt"), f.report("pg"))
	cs := write(t, filepath.Join(d, "h1.yaml"), f.cases())
	out, rc = py(t, "bang-cong-h1.py", mem, mem, cs)
	blocked("cùng một tệp hai lần", out, rc)
	out, rc = py(t, "bang-cong-h1.py", pg, mem, cs)
	blocked("đảo hai đường", out, rc)
	f.answer[f.ids[0]] = "câu trả lời khác"
	mem2 := write(t, filepath.Join(d, "mem2.txt"), f.report("mem"))
	out, rc = py(t, "bang-cong-h1.py", mem, mem2, cs)
	blocked("hai tệp đều là đường bộ nhớ", out, rc)
	out, rc = f.run(t, "WIN-I-1 sai dạng\n")
	blocked("tệp gợi ý sai dạng", out, rc)
}

// Độ nhạy 2: bỏ ca có gợi ý lộ thì tính lại G2' / G5'; độ nhạy 1: đếm ca có ký tự nhìn giống.
func TestBangCongDoNhay(t *testing.T) {
	f := newFixture(13)
	bad := f.ids[13*5]
	f.memOK[bad] = false
	f.question[f.ids[0]] = "cấp 5.000" + string(rune(0x00a0)) + "L?"
	out, rc := f.run(t, "# chú thích\n"+bad+" lộ\n")
	wantGates(t, "ca permission trượt nhưng có gợi ý lộ", out, rc, "ĐẠT", "KHÔNG ĐẠT", "ĐẠT")
	if !strings.Contains(out, "bỏ 1 ca có gợi ý lộ") || !strings.Contains(out, "G5' ĐẠT") {
		t.Errorf("độ nhạy 2 phải bỏ 1 ca và cho G5' ĐẠT:\n%s", out)
	}
	if !strings.Contains(out, "ca có ký tự nhìn giống trong câu hỏi: 1,") {
		t.Errorf("độ nhạy 1 phải đếm 1 ca có NBSP trong câu hỏi:\n%s", out)
	}
}

// --- dong-bang-giu-lai.py ---

// kho dựng kho giả: 3 lớp × 16 ca, manifest khớp.
func kho(t *testing.T, d string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("suite: win\ntenant: abc-infra\nreference_date: 2026-10-20\ncases:\n")
	var man []map[string]any
	n := 600
	for _, c := range []string{"entity", "permission", "routing"} {
		for i := 0; i < 16; i++ {
			n++
			id := fmt.Sprintf("WIN-I-%03d", n)
			fmt.Fprintf(&b, "  - id: %s\n    failure_class: %s\n    authored_by: independent\n    principal: u-director\n    question: \"câu %d\"\n    expect:\n      outcome: RESOLVED\n", id, c, n)
			man = append(man, map[string]any{"case_id": id, "gate": "G2", "b1_window": nil})
		}
	}
	write(t, filepath.Join(d, "kho.yaml"), b.String())
	j, _ := json.Marshal(man)
	write(t, filepath.Join(d, "kho-manifest.json"), string(j))
}

func fileSHA(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func TestDongBang(t *testing.T) {
	d1, d2 := t.TempDir(), t.TempDir()
	kho(t, d1)
	kho(t, d2)
	for _, d := range []string{d1, d2} {
		write(t, filepath.Join(d, "loai-truoc-dong-bang.txt"), "# thử\nWIN-I-601 lộ\n")
		if out, rc := py(t, "dong-bang-giu-lai.py", d); rc != 0 || !strings.Contains(out, "loại trước khi chia: 1") ||
			!strings.Contains(out, "h1: 39 ca") || !strings.Contains(out, "exposed: 1 ca") {
			t.Fatalf("đóng băng: rc %d\n%s", rc, out)
		}
	}
	for _, f := range []string{"pilot.yaml", "h1.yaml", "h2.yaml", "exposed.yaml"} {
		if fileSHA(t, filepath.Join(d1, f)) != fileSHA(t, filepath.Join(d2, f)) {
			t.Errorf("%s khác nhau giữa hai lần đóng băng cùng đầu vào: không tất định", f)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(d1, "h1.yaml")); strings.Contains(string(b), "WIN-I-601") {
		t.Error("ca bị loại vẫn nằm trong h1.yaml")
	}
	if out, rc := py(t, "dong-bang-giu-lai.py", d1); rc == 0 {
		t.Errorf("đóng băng lần hai phải bị từ chối (đã có holdout-manifest.json):\n%s", out)
	}
	if out, rc := py(t, "dong-bang-giu-lai.py", "--kiem", d1); rc != 0 || !strings.Contains(out, "KHỚP") {
		t.Errorf("--kiem trên bộ vừa đóng băng: rc %d\n%s", rc, out)
	}
	h1 := filepath.Join(d1, "h1.yaml")
	b, _ := os.ReadFile(h1)
	write(t, h1, strings.Replace(string(b), "câu", "câu sửa", 1))
	if out, rc := py(t, "dong-bang-giu-lai.py", "--kiem", d1); rc != 1 || !strings.Contains(out, "LỆCH") {
		t.Errorf("--kiem sau khi sửa h1.yaml phải LỆCH, thoát 1: rc %d\n%s", rc, out)
	}
}

func TestDongBangTuChoiDauVao(t *testing.T) {
	for name, edit := range map[string]func(d string){
		"CRLF": func(d string) {
			p := filepath.Join(d, "kho.yaml")
			b, _ := os.ReadFile(p)
			os.WriteFile(p, []byte(strings.ReplaceAll(string(b), "\n", "\r\n")), 0o644)
		},
		"thiếu xuống dòng cuối": func(d string) {
			p := filepath.Join(d, "kho.yaml")
			b, _ := os.ReadFile(p)
			os.WriteFile(p, []byte(strings.TrimRight(string(b), "\n")), 0o644)
		},
		"loại id lạ":       func(d string) { write(t, filepath.Join(d, "loai-truoc-dong-bang.txt"), "WIN-I-999 lạ\n") },
		"loại thiếu lý do": func(d string) { write(t, filepath.Join(d, "loai-truoc-dong-bang.txt"), "WIN-I-601\n") },
	} {
		d := t.TempDir()
		kho(t, d)
		edit(d)
		if out, rc := py(t, "dong-bang-giu-lai.py", d); rc == 0 {
			t.Errorf("%s: phải từ chối\n%s", name, out)
		}
		if _, err := os.Stat(filepath.Join(d, "holdout-manifest.json")); err == nil {
			t.Errorf("%s: bị từ chối mà vẫn ghi holdout-manifest.json", name)
		}
	}
}

// --- cam-ket-giu-lai.py (ADR-0006 §10) ---

// TestCamKet: ghi manifest có sha256 đúng của tệp ca + số đếm theo lớp + băm bộ đồ nghề, KHÔNG chép câu hỏi / kỳ vọng
// vào manifest hay đầu ra, và không ghi đè cam kết đã có.
func TestCamKet(t *testing.T) {
	d, out, kit := t.TempDir(), t.TempDir(), t.TempDir()
	kho(t, d)
	cases := filepath.Join(d, "kho.yaml")
	b, _ := os.ReadFile(cases)
	write(t, cases, strings.Replace(string(b), "outcome: RESOLVED", "outcome: RESOLVED\n      value: {number: \"3850\", unit: L}", 1))
	write(t, filepath.Join(kit, "DINH-DANG.md"), "định dạng\n")
	stdout, rc := py(t, "cam-ket-giu-lai.py", cases, out, "--bo-do-nghe", kit)
	if rc != 0 || !strings.Contains(stdout, "TỔNG 48 ca · có kỳ vọng số liệu 1") || !strings.Contains(stdout, "lớp entity             16 ca") {
		t.Fatalf("cam kết: rc %d\n%s", rc, stdout)
	}
	raw, err := os.ReadFile(filepath.Join(out, "holdout-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Files  map[string]string `json:"files_sha256"`
		Counts struct {
			Total    int            `json:"total"`
			PerClass map[string]int `json:"per_class"`
		} `json:"counts"`
		Kit struct {
			Files map[string]string `json:"files_sha256"`
		} `json:"author_kit"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m.Files["kho.yaml"] != fileSHA(t, cases) || m.Counts.Total != 48 || m.Counts.PerClass["routing"] != 16 {
		t.Errorf("manifest sai: %s", raw)
	}
	if m.Kit.Files["DINH-DANG.md"] != fileSHA(t, filepath.Join(kit, "DINH-DANG.md")) {
		t.Errorf("thiếu băm bộ đồ nghề: %s", raw)
	}
	for _, s := range []string{string(raw), stdout} {
		if strings.Contains(s, "câu 6") || strings.Contains(s, "3850") {
			t.Errorf("lộ nội dung ca: %s", s)
		}
	}
	if stdout, rc := py(t, "cam-ket-giu-lai.py", cases, out); rc != 2 {
		t.Errorf("cam kết lần hai phải bị từ chối (thoát 2): rc %d\n%s", rc, stdout)
	}
}

func TestCamKetTuChoiDauVao(t *testing.T) {
	for name, body := range map[string]string{
		"không phải bộ WIN": "suite: khac\ncases:\n  - id: WIN-I-601\n",
		"không có ca":       "suite: win\ncases: []\n",
		"id trùng":          "suite: win\ncases:\n  - id: WIN-I-601\n    failure_class: entity\n  - id: WIN-I-601\n    failure_class: routing\n",
		"YAML hỏng":         "suite: win\ncases: [\n",
	} {
		d, out := t.TempDir(), t.TempDir()
		p := write(t, filepath.Join(d, "h.yaml"), body)
		if stdout, rc := py(t, "cam-ket-giu-lai.py", p, out); rc != 2 {
			t.Errorf("%s: phải từ chối (thoát 2), rc %d\n%s", name, rc, stdout)
		}
		if _, err := os.Stat(filepath.Join(out, "holdout-manifest.json")); err == nil {
			t.Errorf("%s: bị từ chối mà vẫn ghi manifest", name)
		}
	}
}

// --- kiem-nhiem-soan.py (ADR-0006 §10) ---

// TestKiemNhiem: nhật ký đọc trong thư mục được phép thì sạch; đọc mã, Grep không đường dẫn, grep đệ quy từ gốc repo
// thì vi phạm — và đầu ra KHÔNG in nội dung lệnh (nhật ký chứa chính các ca của bộ giữ lại).
func TestKiemNhiem(t *testing.T) {
	root := t.TempDir()
	kit := filepath.Join(root, "docs", "kit")
	code := filepath.Join(root, "core", "ask.go")
	for _, p := range []string{filepath.Join(kit, "DINH-DANG.md"), code} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		write(t, p, "x\n")
	}
	tool := func(name string, input map[string]string) string {
		b, _ := json.Marshal(map[string]any{"message": map[string]any{"content": []any{map[string]any{"type": "tool_use", "name": name, "input": input}}}})
		return string(b) + "\n"
	}
	clean := write(t, filepath.Join(root, "sach.jsonl"), tool("Read", map[string]string{"file_path": filepath.Join(kit, "DINH-DANG.md")})+
		tool("Bash", map[string]string{"command": "cat > docs/kit/nhap.yaml <<'EOF'\nquestion: \"CAU-HOI-BI-MAT 3850 L/ngày\"\nEOF"}))
	dirty := write(t, filepath.Join(root, "ban.jsonl"), tool("Read", map[string]string{"file_path": code})+
		tool("Grep", map[string]string{"pattern": "x"})+
		tool("Bash", map[string]string{"command": "grep -rn CAU-HOI-BI-MAT " + root}))
	args := func(log string) []string { return []string{"--cho-phep", kit, "--goc", root, log} }
	if out, rc := py(t, "kiem-nhiem-soan.py", args(clean)...); rc != 0 || !strings.Contains(out, "TỔNG vi phạm 0") {
		t.Errorf("nhật ký sạch: rc %d\n%s", rc, out)
	}
	out, rc := py(t, "kiem-nhiem-soan.py", args(dirty)...)
	if rc != 1 || !strings.Contains(out, "TỔNG vi phạm 3") || !strings.Contains(out, code) || !strings.Contains(out, "[Grep] <thư mục làm việc>") ||
		!strings.Contains(out, "đệ quy") {
		t.Errorf("nhật ký bẩn phải có 3 vi phạm (đọc mã, Grep không đường dẫn, grep đệ quy từ gốc): rc %d\n%s", rc, out)
	}
	if strings.Contains(out, "CAU-HOI-BI-MAT") {
		t.Errorf("in nội dung lệnh: %s", out)
	}
}

// Độ nhạy 3 (ADR-0006 §10, đăng ký trước H3): ca xác nhận có kỳ vọng hỏi lại / thiếu bằng chứng mà ĐỎ chỉ vì hệ trả số không
// phán quyết. Chặt: lý do đỏ chỉ có outcome; nới: thêm route kéo theo về CLARIFICATION. Không tính: lý do khác, câu trả lời vẫn có
// phán quyết, câu không phải xác nhận, ca xanh. Câu xác nhận không dấu vẫn được nhận.
func TestBangCongDoNhay3(t *testing.T) {
	f := newFixture(13)
	set := func(i int, q, outcome, reasons, answer string, memRed, pgRed bool) string {
		id := f.ids[i]
		f.question[id], f.outcome[id], f.reasons[id], f.answer[id] = q, outcome, reasons, answer
		f.memOK[id], f.pgOK[id] = !memRed, !pgRed
		return id
	}
	const route = "route = SQL_QUERY, muốn CLARIFICATION; "
	loose := set(0, "EX-03 hôm nay được cấp 3850 L đúng không?", "CLARIFICATION_REQUIRED",
		route+"outcome = RESOLVED, muốn CLARIFICATION_REQUIRED", "Dầu kho xuất 3.850 L. Chưa kết luận", true, true)
	strictUnacc := set(1, "ex-03 hom nay duoc cap 3850 L dung khong?", "INSUFFICIENT_EVIDENCE",
		"outcome = RESOLVED, muốn INSUFFICIENT_EVIDENCE", "Dầu kho xuất 3.850 L.", true, true)
	set(2, "EX-03 hôm nay được cấp 3850 L đúng không?", "CLARIFICATION_REQUIRED",
		"outcome = RESOLVED, muốn CLARIFICATION_REQUIRED; thiếu \"Không.\"", "Dầu kho xuất 3.850 L.", true, true)
	set(3, "EX-03 hôm nay được cấp 3850 L phải không?", "CLARIFICATION_REQUIRED",
		"outcome = RESOLVED, muốn CLARIFICATION_REQUIRED", "Không. Dầu kho xuất 3.850 L.", true, true)
	set(4, "EX-03 hôm nay được cấp bao nhiêu?", "CLARIFICATION_REQUIRED",
		"outcome = RESOLVED, muốn CLARIFICATION_REQUIRED", "Dầu kho xuất 3.850 L.", true, true)
	strictOnePath := set(5, "EX-03 hôm nay được cấp 3850 L đúng chưa?", "CLARIFICATION_REQUIRED",
		"outcome = RESOLVED, muốn CLARIFICATION_REQUIRED", "Dầu kho xuất 3.850 L.", true, false)
	set(6, "EX-03 hôm nay được cấp 3850 L đúng không?", "CLARIFICATION_REQUIRED", "", "Bạn cho biết thêm kỳ.", false, false)
	set(7, "EX-03 hôm nay được cấp 3850 L đúng không?", "RESOLVED", "", "Đúng. Dầu kho xuất 3.850 L.", false, false) // ca xác nhận kỳ vọng phán quyết
	out, rc := f.run(t, "")
	if rc != 0 {
		t.Fatalf("rc = %d\n%s", rc, out)
	}
	strict := []string{strictUnacc, strictOnePath}
	if strictUnacc > strictOnePath {
		strict[0], strict[1] = strictOnePath, strictUnacc
	}
	want := fmt.Sprintf("ca xác nhận 7, kỳ vọng hỏi lại / thiếu bằng chứng 6; ĐỎ chỉ vì trả số không phán quyết — chặt 2: %s %s; nới (kèm route kéo theo) 3: %s %s %s",
		strict[0], strict[1], loose, strict[0], strict[1])
	if !strings.Contains(out, want) {
		t.Errorf("độ nhạy 3 sai, muốn chứa %q\n%s", want, out)
	}
}
