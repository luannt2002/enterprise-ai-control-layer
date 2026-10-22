package traces

import (
	"fmt"
	"strings"
	"text/tabwriter"
)

// Dấu đánh dấu trong bảng vết (FormatTable). Một bước có thể mang nhiều dấu.
const (
	MarkDeviant    = "LỆCH"            // bước chạy nhưng kết quả lệch đường thẳng (Status deviant)
	MarkError      = "LỖI"             // bước báo lỗi (Status error hoặc có Fault)
	MarkGap        = "TRỐNG"           // lý do / phương án bị loại chưa được luồng hỏi xuất ra (NotRecorded)
	MarkBadVerdict = "PHÁN QUYẾT HỎNG" // S9 báo kiểm trượt (verdict "false|…")
)

// FormatTable dựng bảng vết từng bước của MỘT lượt cho người đọc (CLI `eais trace`, GET /v1/trace/{id}?format=table): phần đầu
// (ai hỏi, câu gì, phiên bản, kết quả, thời gian, lỗi), bảng bước S0…S10, chi tiết từng bước (vào / ra / lý do / phương án bị
// loại / bằng chứng), và dòng chốt chỉ BƯỚC LỆCH ĐẦU TIÊN.
//
// Đánh dấu (xem hằng Mark*): LỆCH, LỖI, TRỐNG (chỗ luồng hỏi chưa xuất ra — ghi rõ là trống, không bịa), PHÁN QUYẾT HỎNG. Hàm thuần:
// cùng bản ghi cho cùng chuỗi, không đọc đồng hồ hay môi trường.
func FormatTable(r Record) string {
	var b strings.Builder
	d := r.Decision
	fmt.Fprintf(&b, "VẾT %s\n", d.TraceID)
	fmt.Fprintf(&b, "  tenant %s · người hỏi %s (%s, vai %s) · lúc %s\n", d.Tenant, d.Principal, d.Kind, strings.Join(d.Roles, ","), d.Now.Format("2006-01-02T15:04:05Z07:00"))
	fmt.Fprintf(&b, "  câu hỏi: %s\n", d.Question)
	fmt.Fprintf(&b, "  phiên bản: pack %s · mã %s · dữ liệu %s\n", orDash(d.PackVersion), orDash(d.CodeVersion), orDash(d.DataVersion))
	fmt.Fprintf(&b, "  kết quả: route %s · outcome %s · verdict %s\n", orDash(d.Route), orDash(d.Outcome), orDash(d.Verdict))
	fmt.Fprintf(&b, "  thời gian: tổng %d ms · ngoài mọi chặng %d ms\n", r.TotalMs, r.UnaccountedMs)
	if r.Fault != "" {
		fmt.Fprintf(&b, "  %s: %s: %s\n", MarkError, r.Fault, r.Error)
	}
	b.WriteString("\n")

	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "#\tBƯỚC\tTÊN\tTRẠNG THÁI\tMS\tQUYẾT ĐỊNH\tĐÁNH DẤU")
	first := -1
	for i, s := range r.Steps {
		marks := stepMarks(r, s)
		if first < 0 && (s.Status == StatusDeviant || s.Status == StatusError || s.Fault != "") {
			first = i
		}
		ms := "-"
		if s.Timed {
			ms = fmt.Sprintf("%d", s.Ms)
		}
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n", s.Seq, s.Stage, s.Name, s.Status, ms, s.Decision, strings.Join(marks, " · "))
	}
	tw.Flush()

	b.WriteString("\nCHI TIẾT\n")
	for _, s := range r.Steps {
		fmt.Fprintf(&b, "%s %s [%s]\n", s.Stage, s.Name, s.Status)
		detail := func(label string, vals []string) {
			for i, v := range vals {
				prefix := "    " + label + ":"
				if i > 0 {
					prefix = "    " + strings.Repeat(" ", len([]rune(label))+1)
				}
				fmt.Fprintf(&b, "%s %s\n", prefix, v)
			}
		}
		detail("vào", s.Input)
		detail("ra", s.Output)
		if s.Reason != "" {
			detail("lý do", []string{s.Reason})
		}
		detail("loại", s.Rejected)
		for _, e := range s.Evidence {
			fmt.Fprintf(&b, "    bằng chứng: %s (nguồn %s, %s)\n", e.ID, e.Source.SourceType, e.Source.Name)
		}
		if s.Fault != "" {
			fmt.Fprintf(&b, "    lỗi: %s: %s\n", s.Fault, s.Error)
		}
	}

	if first >= 0 {
		s := r.Steps[first]
		fmt.Fprintf(&b, "\nBƯỚC LỆCH ĐẦU TIÊN: %s %s — %s\n", s.Stage, s.Name, s.Decision)
	} else {
		b.WriteString("\nKhông có bước lệch.\n")
	}
	return b.String()
}

// stepMarks trả các dấu của một bước, theo thứ tự cố định: LỖI, LỆCH, PHÁN QUYẾT HỎNG, TRỐNG.
func stepMarks(r Record, s Step) []string {
	var marks []string
	if s.Status == StatusError || s.Fault != "" {
		marks = append(marks, MarkError)
	}
	if s.Status == StatusDeviant {
		marks = append(marks, MarkDeviant)
	}
	if s.Stage == StageValidate && strings.HasPrefix(r.Decision.Verdict, "false|") {
		marks = append(marks, MarkBadVerdict)
	}
	gap := IsNotRecorded(s.Reason)
	for _, x := range s.Rejected {
		gap = gap || IsNotRecorded(x)
	}
	if gap {
		marks = append(marks, MarkGap)
	}
	return marks
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
