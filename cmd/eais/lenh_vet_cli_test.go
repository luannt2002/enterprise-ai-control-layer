package main

// U9 (HOP-DONG §15): `eais trace` đọc được lượt lệnh từ kho vết thật, và `eais trace diff` có câu trả lời rõ ràng cho lượt lệnh. FirstDifferentStep chỉ so các
// trường quyết định của luồng hỏi và gọi tên chặng S2…S9; đưa hai lượt lệnh cho nó thì nó chỉ nhầm sang chặng của luồng hỏi (ví dụ outcome khác thành "S6"
// trong khi chặng gây ra là C4). Nên diff từ chối lượt lệnh bằng lời nói rõ và chỉ sang `eais trace`, nơi BƯỚC LỆCH ĐẦU TIÊN chỉ đúng chặng.

import (
	"strings"
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/execution/commands"
)

func TestTraceDiffLuotLenhTuChoiRoRang(t *testing.T) {
	st, admin := openPG(t)
	e := newEnv(t, st)
	dropTicketRows(t, admin, e.tenantA, e.tenantB)
	done := wantResult(t, e.cmd(tokDirector, cmdReq(1, "EX-03", dayAgo)), 200, commands.StatusDone, "", "", commands.RetryNone).TraceID
	denied := wantResult(t, e.cmd(tokDriver, cmdReq(2, "EX-03", dayAgo)), 403, commands.StatusFailed, commands.ReasonPermissionDenied, "policy", commands.RetryNone).TraceID
	ask1 := decodeAs[contracts.AskResponse](t, e.ask(tokDirector, qSQL)).TraceID
	ask2 := decodeAs[contracts.AskResponse](t, e.ask(tokDirector, qSQL)).TraceID

	// hai lượt lệnh, một lệnh một hỏi (cả hai thứ tự): mã 2 (dùng sai), không in "khác" hay "khớp" giả
	for _, p := range [][2]string{{done, denied}, {done, done}, {done, ask1}, {ask1, denied}} {
		rc, out, errOut := runCLI("trace", "diff", "-tenant", e.tenantA, p[0], p[1])
		if rc != 2 || out != "" || !strings.Contains(errOut, "lượt lệnh") || !strings.Contains(errOut, "eais trace -tenant") {
			t.Errorf("diff %s %s: muốn mã 2, stdout rỗng, stderr nói lượt lệnh và chỉ sang `eais trace`; được rc=%d stdout=%q stderr=%q", p[0], p[1], rc, out, errOut)
		}
	}
	// lượt hỏi giữ nguyên: hai lần hỏi cùng một câu thì KHỚP, mã 0
	if rc, out, errOut := runCLI("trace", "diff", "-tenant", e.tenantA, ask1, ask2); rc != 0 || !strings.Contains(out, "KHỚP") || errOut != "" {
		t.Errorf("diff hai lượt hỏi phải như cũ: rc=%d stdout=%q stderr=%q", rc, out, errOut)
	}
	// còn `eais trace` thì chỉ đúng chặng của từng lượt lệnh
	for id, want := range map[string]string{done: "Không có bước lệch.", denied: "BƯỚC LỆCH ĐẦU TIÊN: C4 permission"} {
		if rc, out, _ := runCLI("trace", "-tenant", e.tenantA, id); rc != 0 || !strings.Contains(out, want) {
			t.Errorf("eais trace %s: rc=%d, thiếu %q", id, rc, want)
		}
	}
}
