package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

// traceOpen mở kho và đọc lại cờ -tenant chung của các lệnh đọc vết. Lệnh CLI chạy bằng quyền của người vận hành có DSN (không có
// principal) nên tenant do người vận hành nói rõ bằng cờ — RLS vẫn giới hạn mọi truy vấn trong đúng tenant đó.
func traceOpen(ctx context.Context, name string, args []string, stderr io.Writer, nargs int) (st *pgstore.Store, tctx context.Context, rest []string, tenantID string, ok bool) {
	fs := newFlags(name, stderr)
	fs.StringVar(&tenantID, "tenant", "", "tenant của vết / lần nạp (bắt buộc)")
	if err := fs.Parse(args); err != nil {
		return nil, nil, nil, "", false
	}
	if tenantID == "" || fs.NArg() != nargs {
		fmt.Fprintf(stderr, "%s: cần -tenant và đúng %d đối số\n", name, nargs)
		return nil, nil, nil, "", false
	}
	if !validTenantID(tenantID) {
		fmt.Fprintf(stderr, "%s: -tenant %q không hợp lệ\n", name, tenantID)
		return nil, nil, nil, "", false
	}
	if st, ok = openStore(ctx, stderr); !ok {
		return nil, nil, nil, "", false
	}
	return st, contracts.WithTenant(ctx, contracts.TenantID(tenantID)), fs.Args(), tenantID, true
}

// cmdTrace in bảng vết của một lượt (cùng bảng với GET /v1/trace/{id}?format=table).
func cmdTrace(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	st, tctx, rest, tenantID, ok := traceOpen(ctx, "trace", args, stderr, 1)
	if !ok {
		return 2
	}
	defer st.Close()
	rec, err := st.Decision(tctx, rest[0])
	if errors.Is(err, pgstore.ErrNotFound) {
		fmt.Fprintf(stderr, "không có vết %q trong tenant %s\n", rest[0], tenantID)
		return 1
	}
	if err != nil {
		fmt.Fprintln(stderr, "đọc vết lỗi:", traces.Sanitize(err.Error()))
		return 1
	}
	usage, err := st.Usage(tctx, rest[0])
	if err != nil {
		fmt.Fprintln(stderr, "đọc usage lỗi:", traces.Sanitize(err.Error()))
		return 1
	}
	fmt.Fprint(stdout, traces.FormatTable(rec), formatUsage(usage))
	return 0
}

// cmdTraceDiff chỉ bước ĐẦU TIÊN mà hai lượt cho kết quả khác nhau (traces.FirstDifferentStep). Mã thoát như diff(1): 0 khớp, 1 khác.
func cmdTraceDiff(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	st, tctx, rest, tenantID, ok := traceOpen(ctx, "trace diff", args, stderr, 2)
	if !ok {
		return 2
	}
	defer st.Close()
	var recs [2]traces.Record
	for i, id := range rest {
		rec, err := st.Decision(tctx, id)
		if errors.Is(err, pgstore.ErrNotFound) {
			fmt.Fprintf(stderr, "không có vết %q trong tenant %s\n", id, tenantID)
			return 1
		}
		if err != nil {
			fmt.Fprintln(stderr, "đọc vết lỗi:", traces.Sanitize(err.Error()))
			return 1
		}
		recs[i] = rec
	}
	a, b := recs[0], recs[1]
	if isCommandRecord(a) || isCommandRecord(b) {
		fmt.Fprint(stderr, commandDiffRefusal)
		return 2
	}
	fmt.Fprint(stdout, diffReport(rest[0], rest[1], a, b))
	if _, differ := traces.FirstDifferentStep(a, b); differ {
		return 1
	}
	return 0
}

// commandDiffRefusal là lời từ chối so lượt lệnh (HOP-DONG §15 U9 (3)): FirstDifferentStep chỉ so các trường quyết định của luồng hỏi và gọi tên chặng S2…S9, nên với
// lượt lệnh nó chỉ nhầm sang chặng của luồng hỏi (đo 23/10: lượt DONE so với lượt PERMISSION_DENIED ra "S3 (resolve)" trong khi chặng gây ra là C4). Từ chối bằng lời
// rõ ràng tốt hơn một câu trả lời trông có nghĩa mà sai; so hai lượt lệnh có nghĩa gì là việc của Phase sau.
const commandDiffRefusal = "trace diff: không so được lượt lệnh (C0…C7): phép so chỉ gọi tên chặng của luồng hỏi (S2…S9) và sẽ chỉ nhầm chặng cho lượt lệnh.\n" +
	"Xem từng lượt bằng `eais trace -tenant T <trace_id>`: dòng BƯỚC LỆCH ĐẦU TIÊN chỉ đúng chặng dừng của lượt đó (HOP-DONG §15 U9).\n"

// isCommandRecord báo bản ghi vết có phải của một lượt lệnh không: mọi lượt lệnh có chín bước mang mã C…, lượt hỏi chỉ có S… hoặc S?.
func isCommandRecord(r traces.Record) bool {
	return slices.ContainsFunc(r.Steps, func(s traces.Step) bool { return strings.HasPrefix(s.Stage, "C") })
}

// diffReport dựng báo cáo so hai vết: câu hỏi và phiên bản của từng bên (so hai MỐC cho cùng câu mới có nghĩa), rồi bước đầu tiên khác.
func diffReport(idA, idB string, a, b traces.Record) string {
	var s strings.Builder
	for _, x := range []struct {
		label, id string
		r         traces.Record
	}{{"A", idA, a}, {"B", idB, b}} {
		d := x.r.Decision
		fmt.Fprintf(&s, "%s: %s · câu %q · pack %s · mã %s · dữ liệu %s\n", x.label, x.id, d.Question, orDash(d.PackVersion), orDash(d.CodeVersion), orDash(d.DataVersion))
	}
	if a.Decision.Question != b.Decision.Question {
		s.WriteString("CẢNH BÁO: hai lượt khác CÂU HỎI — so sánh chỉ có nghĩa khi cùng câu (hai mốc mã / dữ liệu / pack).\n")
	}
	d, differ := traces.FirstDifferentStep(a, b)
	if !differ {
		s.WriteString("KHỚP: hai lượt cho cùng kết quả ở mọi bước đã ghi.\n")
		return s.String()
	}
	fmt.Fprintf(&s, "BƯỚC ĐẦU TIÊN KHÁC: %s (%s)\n  A: %s\n  B: %s\n", d.Stage, d.Name, orDash(d.A), orDash(d.B))
	return s.String()
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// cmdQuarantine xuất báo cáo cách ly của một lần nạp: từng bản ghi (khoá, nguồn, lý do, chi tiết) và đếm theo lý do.
func cmdQuarantine(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	st, tctx, rest, tenantID, ok := traceOpen(ctx, "quarantine", args, stderr, 1)
	if !ok {
		return 2
	}
	defer st.Close()
	recs, err := st.Quarantine(tctx, rest[0])
	if err != nil {
		fmt.Fprintln(stderr, "đọc bản ghi cách ly lỗi:", traces.Sanitize(err.Error()))
		return 1
	}
	if len(recs) == 0 {
		fmt.Fprintf(stderr, "không có bản ghi cách ly nào cho lần nạp %q trong tenant %s (mã sai, hoặc lần nạp không có bản ghi nào bị cách ly)\n", rest[0], tenantID)
		return 1
	}
	fmt.Fprint(stdout, quarantineReport(tenantID, rest[0], recs))
	return 0
}

func quarantineReport(tenantID, loadID string, recs []pgstore.QuarantineRecord) string {
	var s strings.Builder
	fmt.Fprintf(&s, "CÁCH LY lần nạp %s · tenant %s · %d bản ghi\n\n", loadID, tenantID, len(recs))
	byReason := map[string]int{}
	for _, r := range recs {
		byReason[r.Reason]++
		fmt.Fprintf(&s, "%s\t%s\t%s\t%s\n", r.Key, r.Source, r.Reason, r.Detail)
	}
	s.WriteString("\nTHEO LÝ DO\n")
	reasons := make([]string, 0, len(byReason))
	for r := range byReason {
		reasons = append(reasons, r)
	}
	sort.Strings(reasons)
	for _, r := range reasons {
		fmt.Fprintf(&s, "  %s: %d\n", r, byReason[r])
	}
	return s.String()
}
