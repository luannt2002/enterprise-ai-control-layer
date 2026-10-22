package main

import (
	"bytes"
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Test của scripts/chay-thu.sh (một lệnh: PG → migrate → nạp → serve → smoke). Script được chạy THẬT dưới bash, nhưng `docker` và `go` là
// hai chương trình GIẢ đứng đầu PATH: không container nào bị đụng, không gì được dịch. `docker ps` giả có một danh sách container (tên, nhãn
// dự án compose, trạng thái) do test dựng và lọc như docker thật: theo tên (--filter name=^/<tên>$), theo nhãn dự án (--filter
// label=com.docker.compose.project=<dự án>) và theo trạng thái (--filter status=…); không có -a thì không thấy container đã dừng hẳn. Mọi lời gọi
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

// composeCalls trả các lời gọi `docker compose …` (dựng / gỡ PG) mà script đã thực hiện. KHÔNG đếm `docker ps --filter
// label=com.docker.compose.project=…`: dòng đó chứa chữ "compose" nhưng chỉ là phép HỎI, không dựng hay gỡ gì.
func (r chayThuResult) composeCalls() []string {
	var out []string
	for _, c := range r.calls {
		if strings.HasPrefix(c, "docker compose ") {
			out = append(out, c)
		}
	}
	return out
}

// fakeContainer là một container mà docker giả "đang có". project rỗng = container không mang nhãn compose (dựng tay bằng docker run).
// status như cột STATE của docker: running | paused | exited. id ghi dạng 12 ký tự hex như `docker ps -q`.
type fakeContainer struct {
	id, name, project, status string
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

// runChayThu chạy scripts/chay-thu.sh. dockerRunning = docker giả có một container đang chạy tên EAIS_PG_CONTAINER (trong env; không đặt thì tên
// mặc định). Container đó KHÔNG mang nhãn dự án compose (như PG dựng tay) để dấu hiệu "tên container" đứng MỘT MÌNH: nếu nó mang nhãn dự án của
// script thì dấu hiệu "dự án" (R4-01) cũng đòi từ chối và che lỗi của phép kiểm tên (cùng cơ chế R3-01 / R4-02); ca cần container có nhãn dùng
// runChayThuDocker. env là các biến thêm (KEY=VALUE);
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
	var containers []fakeContainer
	if dockerRunning {
		containers = []fakeContainer{{id: "0123456789ab", name: pgContainerName(env), status: "running"}}
	}
	return runChayThuDocker(t, script, containers, env...)
}

// chayThuRun là một lần chạy scripts/chay-thu.sh đã dựng sẵn (docker / go giả, thư mục tạm và TMPDIR riêng) nhưng CHƯA chạy. Tách dựng, chạy và chờ
// để test hai lần chạy ĐỒNG THỜI (R5-01) giữ được lần 1 còn sống trong lúc chạy lần 2.
type chayThuRun struct {
	cmd       *exec.Cmd
	out       *bytes.Buffer
	callsPath string
	exited    chan struct{} // đóng khi tiến trình đã thoát (sau start)
	err       error         // kết quả của cmd.Wait, đọc được sau khi exited đóng
}

// newChayThuRun dựng một lần chạy với docker giả đang có đúng danh sách containers (nil = không có container nào). env là các biến thêm
// (KEY=VALUE, biến đứng sau thắng); riêng PATH trong env là PHẦN ĐUÔI của PATH mà script thấy (sau thư mục docker / go giả), mặc định là PATH của
// test. TMPDIR mặc định là một thư mục RIÊNG của lần chạy: tệp khoá của script (R5-01) không dùng chung với script chạy thật trên máy đang chạy test;
// ca nào cần hai lần chạy thấy cùng tệp khoá thì đặt chung TMPDIR. Lần chạy bị chặn bởi giới hạn thời gian: script treo thì test đỏ, không treo cả bộ.
// `go` giả có thêm FAKE_GO_HOLD=<tệp>: ghi <tệp>.dang-giu rồi dừng ở bước dịch tới khi <tệp> xuất hiện, để lần chạy đó còn sống lúc test chạy lần khác.
func newChayThuRun(t *testing.T, script string, containers []fakeContainer, env ...string) *chayThuRun {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Fatalf("cần bash để chạy scripts/chay-thu.sh: %v", err)
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	tmp := filepath.Join(dir, "tmp")
	for _, d := range []string{bin, tmp} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	callsPath := filepath.Join(dir, "calls.log")
	stub := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// docker giả: ghi lời gọi (kèm hai biến mà compose đọc). `docker ps` chọn trong FAKE_CONTAINERS (mỗi dòng "<mã> <tên> <dự án|-> <trạng thái>")
	// đúng như docker thật: --filter name=^/<tên>$ (khớp đúng tên, nên bộ lọc sai tên / không bao giờ khớp không thấy gì), --filter
	// label=com.docker.compose.project[=<dự án>], --filter status=<trạng thái>; không có -a và không lọc status thì chỉ thấy container đang
	// chạy / tạm dừng / khởi động lại (container đã dừng hẳn thì ẩn). In mã (hay tên khi có --format). Nếu docker giả in mã cho mọi lời gọi thì
	// test không phân biệt được bộ lọc đúng với bộ lọc hỏng (R3-01), và nếu không hiểu status thì thêm `--filter status=running` không làm gì đỏ (R4-02).
	stub("docker", `echo "docker $* [EAIS_PG_CONTAINER=${EAIS_PG_CONTAINER-} EAIS_PG_PORT=${EAIS_PG_PORT-}]" >> "$FAKE_CALLS"
if [ "$1" = ps ]; then
	all='' name='' label='' status='' fmt='' prev=''
	for a in "$@"; do
		f=''
		case $a in
		-a | --all | -aq | -qa) all=1 ;;
		--format=*) fmt=${a#--format=} ;;
		--filter=*) f=${a#--filter=} ;;
		*) if [ "$prev" = --filter ]; then f=$a; elif [ "$prev" = --format ]; then fmt=$a; fi ;;
		esac
		case $f in
		name=*) name=${f#name=} ;;
		label=*) label=${f#label=} ;;
		status=*) status=${f#status=} ;;
		esac
		prev=$a
	done
	printf '%s\n' "${FAKE_CONTAINERS-}" | while read -r id cname cproj cstat; do
		[ -n "$id" ] || continue
		if [ -z "$all" ] && [ -z "$status" ]; then
			case $cstat in running | paused | restarting) ;; *) continue ;; esac
		fi
		[ -z "$status" ] || [ "$status" = "$cstat" ] || continue
		[ -z "$name" ] || [ "$name" = "^/$cname\$" ] || continue
		case $label in
		'') ;;
		com.docker.compose.project=*) [ "${label#com.docker.compose.project=}" = "$cproj" ] || continue ;;
		com.docker.compose.project) [ "$cproj" != - ] || continue ;;
		*) continue ;;
		esac
		if [ -n "$fmt" ]; then echo "$cname"; else echo "$id"; fi
	done
fi
exit 0
`)
	// go giả: ghi từng đối số trong ngoặc vuông ("-X main.x=v" là MỘT đối số, không lẫn với hai).
	stub("go", `{ printf 'go'; for a in "$@"; do printf ' [%s]' "$a"; done; echo; } >> "$FAKE_CALLS"
if [ -n "${FAKE_GO_HOLD-}" ]; then
	: > "$FAKE_GO_HOLD.dang-giu"
	while [ ! -e "$FAKE_GO_HOLD" ]; do sleep 0.05; done
fi
exit 42
`)
	var fake []string
	for _, c := range containers {
		project := c.project
		if project == "" {
			project = "-"
		}
		fake = append(fake, c.id+" "+c.name+" "+project+" "+c.status)
	}
	pathTail, rest := os.Getenv("PATH"), []string(nil)
	for _, e := range env {
		if v, ok := strings.CutPrefix(e, "PATH="); ok {
			pathTail = v
		} else {
			rest = append(rest, e)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, "bash", "--noprofile", "--norc", script)
	cmd.WaitDelay = 5 * time.Second // tiến trình con mồ côi (vd flock đang chờ) không giữ ống đầu ra mãi
	cmd.Env = append([]string{"PATH=" + bin + ":" + pathTail, "HOME=" + os.Getenv("HOME"), "TMPDIR=" + tmp, "FAKE_CALLS=" + callsPath,
		"FAKE_CONTAINERS=" + strings.Join(fake, "\n"),
		"EAIS_WORK=" + filepath.Join(dir, "work"), "EAIS_PG_PORT=" + freePort(t)}, rest...) // biến đứng sau thắng: env của ca đè được cổng mặc định
	out := &bytes.Buffer{}
	cmd.Stdout, cmd.Stderr = out, out
	return &chayThuRun{cmd: cmd, out: out, callsPath: callsPath}
}

// start chạy tiến trình nền; exited đóng khi nó thoát.
func (c *chayThuRun) start(t *testing.T) {
	t.Helper()
	if err := c.cmd.Start(); err != nil {
		t.Fatal(err)
	}
	c.exited = make(chan struct{})
	go func() {
		c.err = c.cmd.Wait()
		close(c.exited)
	}()
}

// wait chờ lần chạy xong (đã start) và trả kết quả.
func (c *chayThuRun) wait(t *testing.T) chayThuResult {
	t.Helper()
	<-c.exited
	code := 0
	if ee, ok := c.err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if c.err != nil {
		t.Fatal(c.err)
	}
	raw, _ := os.ReadFile(c.callsPath)
	var calls []string
	for _, l := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if l != "" {
			calls = append(calls, l)
		}
	}
	return chayThuResult{code: code, out: c.out.String(), calls: calls}
}

// runChayThuDocker chạy script (xong mới trở về) với docker giả đang có đúng danh sách containers (nil = không có container nào).
func runChayThuDocker(t *testing.T, script string, containers []fakeContainer, env ...string) chayThuResult {
	t.Helper()
	c := newChayThuRun(t, script, containers, env...)
	c.start(t)
	return c.wait(t)
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
	if r.called("go ") || len(r.composeCalls()) > 0 {
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
	if r.called("go ") || len(r.composeCalls()) > 0 {
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
			if r.called("go ") || len(r.composeCalls()) > 0 {
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
				if strings.Contains(r.out, "dùng lại, sẽ không gỡ nó") || r.called("go ") || len(r.composeCalls()) > 0 {
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
			if len(r.composeCalls()) > 0 {
				t.Errorf("script gọi compose (dựng hay gỡ) trên PG mà nó không dựng: %q", r.calls)
			}
		})
	}
}

// R4-01: cùng dự án compose, tên container KHÁC và cổng KHÁC (trống). Phép kiểm tên container và phép thăm dò cổng đều báo trống, nhưng
// `docker compose up` cùng dự án thấy service đổi tên container và cổng nên TẠO LẠI container cũ: PG đang chạy bị gỡ, dữ liệu tmpfs mất. Dự án
// mặc định (eais) trùng deploy/docker-compose.yml, nên người chỉ đổi EAIS_PG_CONTAINER và EAIS_PG_PORT mà quên EAIS_COMPOSE_PROJECT sẽ gỡ nhầm
// đúng eais-pg đang chạy. Container giả của ca có tên khác và không giữ cổng nào, nên dấu hiệu "dự án đã có container" là hàng rào DUY NHẤT.
// Container tạm dừng (paused) vẫn giữ dữ liệu trong bộ nhớ nên cũng phải chặn. Mỗi ca còn có một container của dự án khác: không được tính.
func TestChayThuTuChoiCungDuAnKhacTenKhacCong(t *testing.T) {
	cases := []struct {
		name    string
		env     []string // ngoài cổng (trống thật, do runChayThuDocker chọn)
		project string   // dự án mà lời từ chối phải nêu
		peer    fakeContainer
	}{
		{"dự án mặc định, chỉ đổi tên và cổng", []string{"EAIS_PG_CONTAINER=thu-pg"}, "eais", fakeContainer{"0123456789ab", "eais-pg", "eais", "running"}},
		{"dự án đặt tay", []string{"EAIS_PG_CONTAINER=thu-pg-b", "EAIS_COMPOSE_PROJECT=thu-proj"}, "thu-proj", fakeContainer{"0123456789ab", "thu-pg-a", "thu-proj", "running"}},
		{"container của dự án đang tạm dừng", []string{"EAIS_PG_CONTAINER=thu-pg-b", "EAIS_COMPOSE_PROJECT=thu-proj"}, "thu-proj", fakeContainer{"0123456789ab", "thu-pg-a", "thu-proj", "paused"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			other := fakeContainer{"ba98765432ab", "khac-pg", "khac-proj", "running"}
			r := runChayThuDocker(t, repoPath("scripts/chay-thu.sh"), []fakeContainer{other, c.peer}, c.env...)
			if r.code != 2 {
				t.Errorf("dự án %q đã có container %q (%s), tên và cổng của ca khác: script thoát %d, muốn 2 (từ chối)\n%s", c.project, c.peer.name, c.peer.status, r.code, r.out)
			}
			for _, want := range []string{"TỪ CHỐI", "dự án compose " + c.project, c.peer.name, "EAIS_COMPOSE_PROJECT", "EAIS_REUSE_PG=1"} {
				if !strings.Contains(r.out, want) {
					t.Errorf("lời từ chối thiếu %q:\n%s", want, r.out)
				}
			}
			if strings.Contains(r.out, other.name) {
				t.Errorf("lời từ chối nêu cả container của dự án KHÁC %q:\n%s", other.name, r.out)
			}
			if r.called("go ") || len(r.composeCalls()) > 0 {
				t.Errorf("script từ chối mà vẫn làm việc (compose up cùng dự án sẽ TẠO LẠI container đang chạy): %q", r.calls)
			}
		})
	}
}

// Container của dự án compose KHÁC (hay dựng tay, không nhãn) không phải của ta: không từ chối, script dựng PG riêng bằng compose đúng dự án
// của nó rồi đi tiếp tới bước dịch và gỡ đúng cái đó khi EAIS_PG_DOWN=1. Chặn kiểu sửa "từ chối khi có container compose bất kỳ" (lọc nhãn mà bỏ
// giá trị dự án): khi đó không ai chạy được hai bản PG cạnh nhau, đúng điều README hứa.
func TestChayThuDuAnKhacHanThiDiTiep(t *testing.T) {
	port := freePort(t)
	others := []fakeContainer{
		{"0123456789ab", "eais-pg", "eais", "running"},     // PG của dự án eais: thứ KHÔNG được đụng
		{"ba98765432ab", "khac-pg", "khac-proj", "paused"}, // dự án khác, tạm dừng
		{"1111111111ab", "tay-pg", "", "running"},          // dựng tay, không nhãn
	}
	r := runChayThuDocker(t, repoPath("scripts/chay-thu.sh"), others, "EAIS_PG_CONTAINER=thu-pg", "EAIS_PG_PORT="+port, "EAIS_COMPOSE_PROJECT=thu-proj", "EAIS_PG_DOWN=1")
	up := "docker compose -p thu-proj -f deploy/docker-compose.yml up -d --wait [EAIS_PG_CONTAINER=thu-pg EAIS_PG_PORT=" + port + "]"
	down := "docker compose -p thu-proj -f deploy/docker-compose.yml down [EAIS_PG_CONTAINER=thu-pg EAIS_PG_PORT=" + port + "]"
	if got := r.composeCalls(); len(got) != 2 || got[0] != up || got[1] != down {
		t.Errorf("dự án khác hẳn: lời gọi compose = %q, muốn đúng [%q, %q]\n%s", got, up, down, r.out)
	}
	if r.code == 2 || strings.Contains(r.out, "TỪ CHỐI: ") || !r.called("go [build]") { // "TỪ CHỐI: " là đầu lời từ chối; lời nhắc cuối script cũng có chữ "TỪ CHỐI" nên không dùng làm dấu
		t.Errorf("container của dự án khác mà script từ chối / không đi tiếp tới bước dịch: %q\n%s", r.calls, r.out)
	}
}

// Quyết định có chủ ý: container của dự án đã DỪNG hẳn (exited) KHÔNG chặn. PG chạy trên tmpfs nên dữ liệu mất ngay khi container dừng: compose dựng
// lại (hay khởi động lại) nó không làm mất gì, còn từ chối sẽ chặn lần chạy lại hợp lệ sau `docker stop` mà không bảo vệ được gì. `docker ps` mặc
// định (không -a) không liệt kê container đã dừng; test này canh việc script không thêm -a (hay status=exited) vào phép hỏi.
func TestChayThuContainerDungHanCuaDuAnKhongChan(t *testing.T) {
	stopped := []fakeContainer{{"0123456789ab", "thu-pg-a", "thu-proj", "exited"}}
	r := runChayThuDocker(t, repoPath("scripts/chay-thu.sh"), stopped, "EAIS_PG_CONTAINER=thu-pg-b", "EAIS_COMPOSE_PROJECT=thu-proj")
	if r.code == 2 || strings.Contains(r.out, "TỪ CHỐI: ") {
		t.Errorf("container đã dừng hẳn của cùng dự án mà script từ chối (dữ liệu tmpfs đã mất, không còn gì để bảo vệ): thoát %d\n%s", r.code, r.out)
	}
	if len(r.composeCalls()) != 1 || !r.called("go [build]") {
		t.Errorf("container đã dừng hẳn: muốn script dựng PG (1 lời gọi compose up) rồi đi tiếp tới bước dịch; lời gọi %q\n%s", r.calls, r.out)
	}
}

// R4-02: container tên đó đang TẠM DỪNG (paused) và cổng của ca là một cổng TRỐNG. Container tạm dừng vẫn giữ dữ liệu trong bộ nhớ nên vẫn phải
// chặn. Cổng trống và container không mang nhãn dự án nên hàng rào DUY NHẤT là bộ lọc tên của `docker ps`: thêm `--filter status=running` cạnh
// nó (bỏ sót container paused, đúng điều chú thích trong script cấm) làm test đỏ.
func TestChayThuTuChoiContainerTamDungCongTrong(t *testing.T) {
	paused := []fakeContainer{{"0123456789ab", defaultPGContainer, "", "paused"}}
	r := runChayThuDocker(t, repoPath("scripts/chay-thu.sh"), paused)
	if r.code != 2 {
		t.Errorf("container %q đang tạm dừng, cổng trống: script thoát %d, muốn 2 (từ chối)\n%s", defaultPGContainer, r.code, r.out)
	}
	if want := "container " + defaultPGContainer + " ĐÃ chạy từ trước"; !strings.Contains(r.out, "TỪ CHỐI") || !strings.Contains(r.out, want) {
		t.Errorf("lời từ chối phải nêu %q (tín hiệu container, không phải cổng):\n%s", want, r.out)
	}
	if r.called("go ") || len(r.composeCalls()) > 0 {
		t.Errorf("script từ chối mà vẫn làm việc: %q", r.calls)
	}
}

// EAIS_REUSE_PG=1 cũng là lời xác nhận ở dấu hiệu "dự án đã có container": script đi tiếp (tới bước dịch), không dựng PG, và KHÔNG gỡ gì kể cả khi
// đặt EAIS_PG_DOWN=1 (không một lời gọi compose nào, dựng hay gỡ).
func TestChayThuDungLaiCungDuAnKhiCoY(t *testing.T) {
	peer := []fakeContainer{{"0123456789ab", "thu-pg-a", "thu-proj", "running"}}
	r := runChayThuDocker(t, repoPath("scripts/chay-thu.sh"), peer, "EAIS_PG_CONTAINER=thu-pg-b", "EAIS_COMPOSE_PROJECT=thu-proj", "EAIS_REUSE_PG=1", "EAIS_PG_DOWN=1")
	if !r.called("go [build]") {
		t.Errorf("EAIS_REUSE_PG=1 mà script không đi tiếp tới bước dịch: thoát %d, lời gọi %q\n%s", r.code, r.calls, r.out)
	}
	if !strings.Contains(r.out, "dùng lại, sẽ không gỡ nó") {
		t.Errorf("script không nói rõ là dùng lại và sẽ không gỡ:\n%s", r.out)
	}
	if got := r.composeCalls(); len(got) > 0 {
		t.Errorf("script gọi compose (dựng hay gỡ) trên PG mà nó không dựng: %q", got)
	}
}

// Chỗ trống thì script tự dựng PG riêng bằng compose với đúng tên container / cổng / dự án được đặt, đi tiếp, và gỡ đúng cái nó dựng khi
// EAIS_PG_DOWN=1.
func TestChayThuChoTrongThiTuDung(t *testing.T) {
	port := freePort(t)
	r := runChayThu(t, false, "EAIS_PG_CONTAINER=thu-pg", "EAIS_PG_PORT="+port, "EAIS_COMPOSE_PROJECT=thu-proj", "EAIS_PG_DOWN=1")
	up := "docker compose -p thu-proj -f deploy/docker-compose.yml up -d --wait [EAIS_PG_CONTAINER=thu-pg EAIS_PG_PORT=" + port + "]"
	down := "docker compose -p thu-proj -f deploy/docker-compose.yml down [EAIS_PG_CONTAINER=thu-pg EAIS_PG_PORT=" + port + "]"
	got := r.composeCalls()
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

// R5-01: hai lần chạy ĐỒNG THỜI cùng dự án compose. Hàng rào pg_chiem là kiểm-rồi-làm, không có khoá: hai lần chạy bắt đầu cùng lúc cùng thấy dự án
// trống rồi cùng dựng PG, và cleanup gỡ bằng `docker compose down` cấp DỰ ÁN nên lần xong trước (EAIS_PG_DOWN=1) gỡ luôn PG của lần kia giữa lúc nó
// chạy. Docker giả không đổi trạng thái (dự án luôn "trống"), nên pg_chiem không cứu được: chỉ khoá theo dự án mới từ chối lần 2. Lần 1 dừng ở `go`
// giả (bước dịch) tới khi test nhả, nên nó còn sống suốt lúc các lần khác chạy; mọi lần chạy dùng CHUNG TMPDIR (nơi đặt tệp khoá).
func TestChayThuKhoaTheoDuAn(t *testing.T) {
	if _, err := exec.LookPath("flock"); err != nil {
		t.Fatalf("cần flock để kiểm khoá theo dự án của scripts/chay-thu.sh: %v", err)
	}
	script := repoPath("scripts/chay-thu.sh")
	shared := t.TempDir()
	release := filepath.Join(t.TempDir(), "nha")
	envOf := func(project string, extra ...string) []string {
		return append([]string{"TMPDIR=" + shared, "EAIS_COMPOSE_PROJECT=" + project, "EAIS_PG_CONTAINER=" + project + "-pg", "EAIS_PG_DOWN=1"}, extra...)
	}

	first := newChayThuRun(t, script, nil, envOf("thu-proj", "FAKE_GO_HOLD="+release)...)
	first.start(t)
	t.Cleanup(func() {
		os.WriteFile(release, nil, 0o644) // nhả lần 1 nếu test dừng giữa chừng
		<-first.exited
	})
	for deadline := time.Now().Add(20 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		if _, err := os.Stat(release + ".dang-giu"); err == nil {
			break
		}
		select {
		case <-first.exited:
			t.Fatalf("lần 1 thoát trước khi tới bước dịch:\n%s", first.wait(t).out)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("lần 1 không tới được bước dịch sau 20 giây")
		}
	}

	// Lần 2: CÙNG dự án, tên container và cổng khác (cổng trống do harness chọn), cũng EAIS_PG_DOWN=1 — tổ hợp mà nếu lần 2 chạy tiếp thì cleanup của
	// nó gỡ PG của lần 1. Bị từ chối thì chưa được làm gì: không một lời gọi docker / go nào.
	second := runChayThuDocker(t, script, nil, envOf("thu-proj", "EAIS_PG_CONTAINER=thu-pg-b")...)
	if second.code != 2 {
		t.Errorf("lần 1 còn sống, lần 2 cùng dự án: thoát %d, muốn 2 (từ chối)\n%s", second.code, second.out)
	}
	for _, want := range []string{"TỪ CHỐI", "một lần chạy khác của dự án compose thu-proj"} {
		if !strings.Contains(second.out, want) {
			t.Errorf("lời từ chối thiếu %q:\n%s", want, second.out)
		}
	}
	if len(second.calls) != 0 {
		t.Errorf("lần 2 bị từ chối mà vẫn gọi docker / go (cleanup của nó gỡ PG của lần 1): %q", second.calls)
	}

	// Dự án KHÁC thì đi tiếp (khoá theo dự án, không phải khoá toàn cục).
	other := runChayThuDocker(t, script, nil, envOf("thu-proj-khac")...)
	if other.code == 2 || strings.Contains(other.out, "TỪ CHỐI: ") || !other.called("go [build]") {
		t.Errorf("dự án khác mà script từ chối / không đi tiếp tới bước dịch: thoát %d, lời gọi %q\n%s", other.code, other.calls, other.out)
	}

	// Đường PG có sẵn (cả hai DSN) không chạm compose nên không đòi khoá dự án, dù khoá của dự án đang bị giữ.
	viaDSN := runChayThuDocker(t, script, nil, envOf("thu-proj", "EAIS_PG_ADMIN_DSN=postgres://postgres@127.0.0.1:1/x?sslmode=disable", "EAIS_PG_DSN=postgres://eais_app@127.0.0.1:1/x?sslmode=disable")...)
	if viaDSN.code == 2 || viaDSN.called("docker") || !viaDSN.called("go [build]") {
		t.Errorf("đã có DSN: script phải đi tiếp tới bước dịch, không chạm docker, không đòi khoá dự án; thoát %d, lời gọi %q\n%s", viaDSN.code, viaDSN.calls, viaDSN.out)
	}

	// Nhả lần 1: nó không bị lần 2 làm gì (thoát bằng mã của go giả, dựng đúng một lần và gỡ đúng của nó), rồi khoá được nhả: lần chạy sau cùng dự án đi tiếp.
	if err := os.WriteFile(release, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	r1 := first.wait(t)
	if got := r1.composeCalls(); r1.code != 42 || len(got) != 2 || !strings.Contains(got[0], " up -d --wait ") || !strings.Contains(got[1], " down ") {
		t.Errorf("lần 1: thoát %d (muốn 42 của go giả), lời gọi compose %q (muốn đúng một up rồi một down của chính nó)\n%s", r1.code, got, r1.out)
	}
	again := runChayThuDocker(t, script, nil, envOf("thu-proj")...)
	if again.code == 2 || strings.Contains(again.out, "TỪ CHỐI: ") || !again.called("go [build]") {
		t.Errorf("lần 1 đã xong mà khoá dự án vẫn giữ: lần chạy sau thoát %d, lời gọi %q\n%s", again.code, again.calls, again.out)
	}
}

// pathKhongFlock dựng một thư mục PATH chỉ có lối tắt tới vài công cụ mà script cần để đi tới bước dịch (dirname, mkdir, git) và KHÔNG có flock:
// mô phỏng máy không có flock (macOS mặc định) mà vẫn chạy được script thật.
func pathKhongFlock(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, tool := range []string{"dirname", "mkdir", "git"} {
		p, err := exec.LookPath(tool)
		if err != nil {
			continue
		}
		if err := os.Symlink(p, filepath.Join(dir, tool)); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// R5-01: không khoá được (máy không có flock; hay không mở được tệp khoá) thì script in CẢNH BÁO rõ rồi đi tiếp như trước khi có khoá, KHÔNG từ chối.
// Lựa chọn có chủ ý: từ chối sẽ chặn mọi lần chạy đơn lẻ (vốn an toàn) trên máy đó để phòng một lần chạy đồng thời hiếm, và cảnh báo nói thẳng rủi ro còn lại.
func TestChayThuKhongKhoaDuocThiCanhBaoRoiDiTiep(t *testing.T) {
	script := repoPath("scripts/chay-thu.sh")
	cases := []struct {
		name string
		env  []string
	}{
		{"máy không có flock", []string{"PATH=" + pathKhongFlock(t)}},
		{"không mở được tệp khoá", []string{"TMPDIR=" + filepath.Join(t.TempDir(), "khong-co-thu-muc")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := runChayThuDocker(t, script, nil, c.env...)
			if r.code == 2 || strings.Contains(r.out, "TỪ CHỐI: ") || !r.called("go [build]") {
				t.Errorf("không khoá được mà script từ chối / không đi tiếp tới bước dịch: thoát %d, lời gọi %q\n%s", r.code, r.calls, r.out)
			}
			for _, want := range []string{"CẢNH BÁO", "đồng thời"} {
				if !strings.Contains(r.out, want) {
					t.Errorf("thiếu %q trong cảnh báo (không khoá được phải nói rõ rủi ro còn lại):\n%s", want, r.out)
				}
			}
		})
	}
}
