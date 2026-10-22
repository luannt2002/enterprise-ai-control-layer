package main

import (
	"bytes"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Test của scripts/chay-thu.sh (một lệnh: PG → migrate → nạp → serve → smoke). Script được chạy THẬT dưới bash, nhưng `docker` và `go` là
// hai chương trình GIẢ đứng đầu PATH: không container nào bị đụng, không gì được dịch. `docker ps` giả báo có container đang chạy hay
// không theo ý test và — như docker thật — CHỈ thấy nó khi lời gọi lọc ĐÚNG theo tên container đó (--filter name=^/<tên>$); mọi lời gọi
// docker / go được ghi lại để test khẳng định script ĐÃ và CHƯA làm gì. `go` giả thoát 42 nên script dừng
// ngay ở bước dịch binary — đủ để biết nó đã vượt cổng PG hay chưa mà không dựng gì thật.

// chayThuResult là kết quả một lần chạy scripts/chay-thu.sh với docker / go giả.
type chayThuResult struct {
	code  int
	out   string   // stdout và stderr của script, gộp
	calls []string // mỗi dòng một lời gọi docker / go mà script đã thực hiện, theo thứ tự
}

func (r chayThuResult) called(sub string) bool {
	for _, c := range r.calls {
		if strings.Contains(c, sub) {
			return true
		}
	}
	return false
}

// defaultPGContainer là tên container PG mặc định của script (khi EAIS_PG_CONTAINER không đặt) và của deploy/docker-compose.yml.
const defaultPGContainer = "eais-pg"

// pgContainerName là tên container mà script sẽ hỏi docker: EAIS_PG_CONTAINER trong env (biến đứng sau thắng, như exec.Cmd); không đặt
// hoặc rỗng thì tên mặc định.
func pgContainerName(env []string) string {
	name := ""
	for _, e := range env {
		if v, ok := strings.CutPrefix(e, "EAIS_PG_CONTAINER="); ok {
			name = v
		}
	}
	if name == "" {
		return defaultPGContainer
	}
	return name
}

// runChayThu chạy scripts/chay-thu.sh. dockerRunning = `docker ps` giả báo container tên EAIS_PG_CONTAINER (trong env; không đặt thì tên
// mặc định) đang chạy — và chỉ khi lời gọi lọc đúng theo tên đó. env là các biến thêm (KEY=VALUE);
// KHÔNG kế thừa biến EAIS_* của môi trường thật (kể cả DSN): mỗi ca tự nêu cái nó cần. Cổng PG mặc định của ca là một cổng đang TRỐNG chứ
// không phải 25432: script thăm dò cổng bằng một lần kết nối thật, nên cổng mặc định mà trên máy có PG của việc khác thì (1) ca "container
// đang chạy" bị từ chối nhờ cổng chứ không nhờ container, che lỗi của phép kiểm container, (2) test chạm vào PG của người khác.
func runChayThu(t *testing.T, dockerRunning bool, env ...string) chayThuResult {
	t.Helper()
	return runChayThuScript(t, repoPath("scripts/chay-thu.sh"), dockerRunning, env...)
}

// runChayThuScript như runChayThu nhưng chạy script ở đường dẫn cho trước (test phiên bản mã chạy bản sao trong một kho git tạm).
func runChayThuScript(t *testing.T, script string, dockerRunning bool, env ...string) chayThuResult {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Fatalf("cần bash để chạy scripts/chay-thu.sh: %v", err)
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	callsPath := filepath.Join(dir, "calls.log")
	stub := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// docker giả: ghi lời gọi (kèm hai biến mà compose đọc). `docker ps` chỉ in mã container khi test bảo "đang chạy" (FAKE_RUNNING = tên
	// container) VÀ có một đối số lọc đúng bằng name=^/<tên>$ (sau --filter, hoặc dạng --filter=…). Bộ lọc sai tên, không bao giờ khớp hay
	// không lọc theo tên thì không thấy gì: nếu docker giả in mã cho mọi lời gọi thì test không phân biệt được bộ lọc đúng với bộ lọc hỏng (R3-01).
	stub("docker", `echo "docker $* [EAIS_PG_CONTAINER=${EAIS_PG_CONTAINER-} EAIS_PG_PORT=${EAIS_PG_PORT-}]" >> "$FAKE_CALLS"
if [ "$1" = ps ] && [ -n "${FAKE_RUNNING-}" ]; then
	prev=
	for a in "$@"; do
		if [ "$a" = "--filter=name=^/${FAKE_RUNNING}\$" ] || { [ "$prev" = --filter ] && [ "$a" = "name=^/${FAKE_RUNNING}\$" ]; }; then echo 0123456789ab; fi
		prev=$a
	done
fi
exit 0
`)
	// go giả: ghi từng đối số trong ngoặc vuông ("-X main.x=v" là MỘT đối số, không lẫn với hai).
	stub("go", `{ printf 'go'; for a in "$@"; do printf ' [%s]' "$a"; done; echo; } >> "$FAKE_CALLS"
exit 42
`)
	cmd := exec.Command("bash", "--noprofile", "--norc", script)
	cmd.Env = append([]string{"PATH=" + bin + ":" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"), "FAKE_CALLS=" + callsPath,
		"EAIS_WORK=" + filepath.Join(dir, "work"), "EAIS_PG_PORT=" + freePort(t)}, env...) // biến đứng sau thắng: env của ca đè được cổng mặc định
	if dockerRunning {
		cmd.Env = append(cmd.Env, "FAKE_RUNNING="+pgContainerName(env))
	}
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(callsPath)
	var calls []string
	for _, l := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if l != "" {
			calls = append(calls, l)
		}
	}
	return chayThuResult{code: code, out: out.String(), calls: calls}
}

// listenBusyPort mở một cổng loopback và GIỮ nó (đóng khi test xong): script thấy cổng này "đã có tiến trình nghe".
func listenBusyPort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	return strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
}

// freePort trả một cổng loopback đang trống (mở rồi đóng ngay).
func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	ln.Close()
	return port
}

// R2-04: một lệnh với giá trị mặc định KHÔNG được dùng lại và ghi vào PG đang chạy của người khác (script migrate rồi nạp THAY toàn bộ
// dữ liệu tenant). Thấy container tên đó đang chạy ⇒ từ chối (thoát 2), chưa gọi compose và chưa dịch / chạy gì, kèm hướng dẫn.
func TestChayThuTuChoiContainerDangChay(t *testing.T) {
	r := runChayThu(t, true)
	if r.code != 2 {
		t.Errorf("container đang chạy, không EAIS_REUSE_PG: script thoát %d, muốn 2 (từ chối)\n%s", r.code, r.out)
	}
	for _, want := range []string{"TỪ CHỐI", "EAIS_REUSE_PG=1", "EAIS_PG_CONTAINER", "EAIS_PG_PORT", "EAIS_COMPOSE_PROJECT"} {
		if !strings.Contains(r.out, want) {
			t.Errorf("lời từ chối thiếu %q:\n%s", want, r.out)
		}
	}
	if r.called("go ") || r.called("compose") {
		t.Errorf("script từ chối mà vẫn làm việc (migrate / nạp ghi vào PG của người khác nằm sau các lời gọi này): %q", r.calls)
	}
}

// Cổng đã có tiến trình nghe (PG không phải do docker của script này dựng, hay container tên khác) cũng bị từ chối.
func TestChayThuTuChoiCongBan(t *testing.T) {
	port := listenBusyPort(t)
	r := runChayThu(t, false, "EAIS_PG_PORT="+port)
	if r.code != 2 || !strings.Contains(r.out, "TỪ CHỐI") || !strings.Contains(r.out, "127.0.0.1:"+port) {
		t.Errorf("cổng %s đã bị giữ: script thoát %d, muốn 2 và nêu cổng:\n%s", port, r.code, r.out)
	}
	if r.called("go ") || r.called("compose") {
		t.Errorf("script từ chối mà vẫn làm việc: %q", r.calls)
	}
}

// R3-01: người chạy chỉ đổi CỔNG sang một cổng TRỐNG mà giữ cùng tên container (và cùng dự án compose, tức cùng container do compose quản
// lý). Cổng trống nên phép thăm dò cổng không bắt được: hàng rào DUY NHẤT là bộ lọc tên container của `docker ps`. Hỏng bộ lọc thì bước kế
// tiếp là `docker compose up` cùng dự án với cổng khác, và compose TẠO LẠI container đang chạy (mất dữ liệu tmpfs của PG đó) — nặng hơn cả
// ghi đè. Docker giả chỉ thấy container khi lời gọi lọc đúng name=^/<tên>$ nên bộ lọc sai tên / không bao giờ khớp / không lọc đều làm test đỏ.
func TestChayThuTuChoiCungTenContainerChiDoiCong(t *testing.T) {
	cases := []struct {
		name string
		env  []string // ngoài cổng
	}{
		{"tên mặc định", nil},
		{"tên và dự án đặt tay", []string{"EAIS_PG_CONTAINER=thu-pg", "EAIS_COMPOSE_PROJECT=thu-proj"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			port := freePort(t) // trống thật: chỉ container mới là dấu hiệu "đã chạy"
			env := append(append([]string{}, c.env...), "EAIS_PG_PORT="+port)
			r := runChayThu(t, true, env...)
			if r.code != 2 {
				t.Errorf("container %q đang chạy, cổng %s trống: script thoát %d, muốn 2 (từ chối)\n%s", pgContainerName(env), port, r.code, r.out)
			}
			if want := "container " + pgContainerName(env) + " ĐÃ chạy từ trước"; !strings.Contains(r.out, "TỪ CHỐI") || !strings.Contains(r.out, want) {
				t.Errorf("lời từ chối phải nêu %q (tín hiệu container, không phải cổng):\n%s", want, r.out)
			}
			if r.called("go ") || r.called("compose") {
				t.Errorf("script từ chối mà vẫn làm việc (compose up cùng dự án khác cổng sẽ TẠO LẠI container đang chạy): %q", r.calls)
			}
		})
	}
}

// R3-03: chỉ đúng "1" là lời xác nhận dùng lại PG đang chạy. Giá trị khác (0, true — thứ người ta hay gõ để NÓI "không" hay "có") vẫn bị
// TỪ CHỐI, ở cả hai dấu hiệu (container, cổng). Bắt được thay đổi thành "đặt giá trị bất kỳ là cố ý dùng lại" (vd kiểm khác rỗng) mà
// TestChayThuDungLaiKhiCoY (chỉ thử "1") không thấy.
func TestChayThuReusePGChiNhanDungMot(t *testing.T) {
	where := map[string]struct {
		running bool
		env     []string
	}{
		"container đang chạy": {true, nil},
		"cổng bận":            {false, []string{"EAIS_PG_PORT=" + listenBusyPort(t)}},
	}
	for _, val := range []string{"0", "true"} {
		for name, c := range where {
			t.Run("EAIS_REUSE_PG="+val+" "+name, func(t *testing.T) {
				r := runChayThu(t, c.running, append([]string{"EAIS_REUSE_PG=" + val}, c.env...)...)
				if r.code != 2 || !strings.Contains(r.out, "TỪ CHỐI") {
					t.Errorf("EAIS_REUSE_PG=%s: script thoát %d, muốn 2 (chỉ \"1\" mới là dùng lại)\n%s", val, r.code, r.out)
				}
				if strings.Contains(r.out, "dùng lại, sẽ không gỡ nó") || r.called("go ") || r.called("compose") {
					t.Errorf("EAIS_REUSE_PG=%s mà script vẫn dùng lại / làm việc trên PG đang chạy: %q\n%s", val, r.calls, r.out)
				}
			})
		}
	}
}

// EAIS_REUSE_PG=1 là cách CỐ Ý dùng lại: script đi tiếp (tới bước dịch), không dựng PG nữa (đã có), và KHÔNG BAO GIỜ gỡ cái nó không dựng,
// kể cả khi đặt EAIS_PG_DOWN=1.
func TestChayThuDungLaiKhiCoY(t *testing.T) {
	cases := map[string]struct {
		running bool
		env     []string
	}{
		"container đang chạy": {true, nil},
		"cổng bận":            {false, []string{"EAIS_PG_PORT=" + listenBusyPort(t)}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			r := runChayThu(t, c.running, append([]string{"EAIS_REUSE_PG=1", "EAIS_PG_DOWN=1"}, c.env...)...)
			if !r.called("go [build]") {
				t.Errorf("EAIS_REUSE_PG=1 mà script không đi tiếp tới bước dịch: thoát %d, lời gọi %q\n%s", r.code, r.calls, r.out)
			}
			if !strings.Contains(r.out, "dùng lại, sẽ không gỡ nó") {
				t.Errorf("script không nói rõ là dùng lại và sẽ không gỡ:\n%s", r.out)
			}
			if r.called("compose") {
				t.Errorf("script gọi compose (dựng hay gỡ) trên PG mà nó không dựng: %q", r.calls)
			}
		})
	}
}

// Chỗ trống thì script tự dựng PG riêng bằng compose với đúng tên container / cổng / dự án được đặt, đi tiếp, và gỡ đúng cái nó dựng khi
// EAIS_PG_DOWN=1.
func TestChayThuChoTrongThiTuDung(t *testing.T) {
	port := freePort(t)
	r := runChayThu(t, false, "EAIS_PG_CONTAINER=thu-pg", "EAIS_PG_PORT="+port, "EAIS_COMPOSE_PROJECT=thu-proj", "EAIS_PG_DOWN=1")
	up := "docker compose -p thu-proj -f deploy/docker-compose.yml up -d --wait [EAIS_PG_CONTAINER=thu-pg EAIS_PG_PORT=" + port + "]"
	down := "docker compose -p thu-proj -f deploy/docker-compose.yml down [EAIS_PG_CONTAINER=thu-pg EAIS_PG_PORT=" + port + "]"
	var got []string
	for _, c := range r.calls {
		if strings.Contains(c, "compose") {
			got = append(got, c)
		}
	}
	if len(got) != 2 || got[0] != up || got[1] != down {
		t.Errorf("chỗ trống: lời gọi compose = %q, muốn đúng [%q, %q]\n%s", got, up, down, r.out)
	}
	if !r.called("go [build]") {
		t.Errorf("chỗ trống mà script không đi tiếp tới bước dịch: %q\n%s", r.calls, r.out)
	}
}

// Đã đặt cả hai DSN (CI, PG sẵn có) thì script dùng đúng PG đó và KHÔNG chạm Docker — kể cả khi có container trùng tên đang chạy.
func TestChayThuDSNCoSanKhongChamDocker(t *testing.T) {
	r := runChayThu(t, true, "EAIS_PG_ADMIN_DSN=postgres://postgres@127.0.0.1:1/x?sslmode=disable", "EAIS_PG_DSN=postgres://eais_app@127.0.0.1:1/x?sslmode=disable")
	if r.called("docker") || !r.called("go [build]") {
		t.Errorf("đã có DSN: script phải không gọi docker và đi tiếp tới bước dịch; lời gọi %q\n%s", r.calls, r.out)
	}
}

// buildArgs trả các đối số của lời gọi `go build` (từ dòng log `go [build] [-ldflags] [-X …] …`); nil nếu script chưa tới bước dịch.
func (r chayThuResult) buildArgs() []string {
	for _, c := range r.calls {
		if strings.HasPrefix(c, "go [build]") {
			return strings.Split(strings.Trim(strings.TrimPrefix(c, "go "), "[]"), "] [")
		}
	}
	return nil
}

// linkedVersion trả giá trị mà `go build` nhúng bằng -ldflags "-X main.buildVersion=…" (ok = false nếu không có cờ đó hay sai dạng).
func (r chayThuResult) linkedVersion() (string, bool) {
	args := r.buildArgs()
	for i, a := range args {
		if a == "-ldflags" && i+1 < len(args) {
			return strings.CutPrefix(args[i+1], "-X main.buildVersion=")
		}
	}
	return "", false
}

// gitTmp chạy git trong dir với cấu hình sạch (không đọc cấu hình người dùng), trả stdout đã cắt.
func gitTmp(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@example.invalid", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + dir, "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null"}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// scriptTrongKho chép scripts/chay-thu.sh vào <dir>/scripts/ (script tự cd về gốc kho chứa nó) và trả đường dẫn bản chép.
func scriptTrongKho(t *testing.T, dir string) string {
	t.Helper()
	raw, err := os.ReadFile(repoPath("scripts/chay-thu.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "scripts", "chay-thu.sh")
	if err := os.WriteFile(path, raw, 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// R2-02: script nhúng phiên bản mã vào binary bằng -ldflags "-X main.buildVersion=<mã>". Chạy bản chép của script trong một kho git tạm (có
// kiểm soát: sạch, sửa tệp đã theo dõi, tệp mới chưa theo dõi, không có git) thay vì kho thật, để kết quả không phụ thuộc cây đang làm việc.
// Đã đặt cả hai DSN nên script không chạm docker.
func TestChayThuNhungPhienBanMa(t *testing.T) {
	dsn := []string{"EAIS_PG_ADMIN_DSN=postgres://postgres@127.0.0.1:1/x?sslmode=disable", "EAIS_PG_DSN=postgres://eais_app@127.0.0.1:1/x?sslmode=disable"}
	gitEnv := []string{"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null"}
	mkRepo := func(t *testing.T) (script, dir, head string) {
		t.Helper()
		dir = t.TempDir()
		script = scriptTrongKho(t, dir)
		if err := os.WriteFile(filepath.Join(dir, "da-theo-doi.txt"), []byte("v1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		gitTmp(t, dir, "init", "-q")
		gitTmp(t, dir, "add", "scripts", "da-theo-doi.txt")
		gitTmp(t, dir, "commit", "-q", "-m", "x")
		return script, dir, gitTmp(t, dir, "rev-parse", "HEAD")
	}
	check := func(t *testing.T, r chayThuResult, want string) {
		t.Helper()
		got, ok := r.linkedVersion()
		if !ok || got != want {
			t.Errorf("-ldflags nhúng %q (có cờ: %v), muốn %q; lời gọi go build: %q\n%s", got, ok, want, r.buildArgs(), r.out)
		}
	}

	t.Run("kho sạch: mã git của HEAD", func(t *testing.T) {
		script, _, head := mkRepo(t)
		check(t, runChayThuScript(t, script, false, append(dsn, gitEnv...)...), head)
	})
	t.Run("sửa tệp đã theo dõi: +dirty", func(t *testing.T) {
		script, dir, head := mkRepo(t)
		if err := os.WriteFile(filepath.Join(dir, "da-theo-doi.txt"), []byte("v2\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		check(t, runChayThuScript(t, script, false, append(dsn, gitEnv...)...), head+"+dirty")
	})
	t.Run("tệp mới chưa theo dõi: +dirty", func(t *testing.T) {
		script, dir, head := mkRepo(t)
		if err := os.WriteFile(filepath.Join(dir, "moi.go"), []byte("package x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		check(t, runChayThuScript(t, script, false, append(dsn, gitEnv...)...), head+"+dirty")
	})
	t.Run("EAIS_CODE_VERSION thắng git", func(t *testing.T) {
		script, _, _ := mkRepo(t)
		const pinned = "0123456789abcdef0123456789abcdef01234567"
		check(t, runChayThuScript(t, script, false, append(append(dsn, gitEnv...), "EAIS_CODE_VERSION="+pinned)...), pinned)
	})
	t.Run("không có git: không nhúng gì (ứng dụng dùng băm binary)", func(t *testing.T) {
		dir := t.TempDir()
		script := scriptTrongKho(t, dir)
		// GIT_CEILING_DIRECTORIES: git không đi lên thư mục cha tìm kho (thư mục tạm có thể nằm trong một kho nào đó trên máy dev).
		check(t, runChayThuScript(t, script, false, append(append(dsn, gitEnv...), "GIT_CEILING_DIRECTORIES="+filepath.Dir(dir))...), "")
	})
	t.Run("giá trị lạ bị từ chối trước khi làm gì", func(t *testing.T) {
		script, _, _ := mkRepo(t)
		for _, bad := range []string{"a b", "a;b", "$(id)", "a\"b", "-X x=y"} {
			r := runChayThuScript(t, script, false, append(append(dsn, gitEnv...), "EAIS_CODE_VERSION="+bad)...)
			if r.code != 2 || !strings.Contains(r.out, "EAIS_CODE_VERSION") || r.called("go ") || r.called("docker") {
				t.Errorf("EAIS_CODE_VERSION=%q: thoát %d, lời gọi %q, muốn thoát 2, nêu EAIS_CODE_VERSION, chưa gọi go / docker\n%s", bad, r.code, r.calls, r.out)
			}
		}
	})
	t.Run("kho thật của repo: mã git hoặc rỗng nếu không có git", func(t *testing.T) {
		cmd := exec.Command("git", "rev-parse", "--verify", "-q", "HEAD")
		cmd.Dir = repoPath(".")
		raw, err := cmd.Output()
		r := runChayThu(t, false, dsn...)
		got, ok := r.linkedVersion()
		if err != nil { // cây không có git (tải về dạng tệp nén): script không nhúng gì
			if !ok || got != "" {
				t.Errorf("cây không có git: nhúng %q (có cờ: %v), muốn rỗng", got, ok)
			}
			return
		}
		head := strings.TrimSpace(string(raw))
		if !ok || (got != head && got != head+"+dirty") {
			t.Errorf("nhúng %q (có cờ: %v), muốn %q hoặc %q+dirty", got, ok, head, head)
		}
	})
}
