package rag

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Câu hỏi gõ KHÔNG dấu: điểm xếp lại của nó thấp hơn hẳn câu có dấu cùng ý (đo 24/10 trên bộ phát triển văn bản pháp quy: 8 câu không
// dấu, 6 câu top-3 đã có đáp án nhưng 0 câu qua ngưỡng "biết im"; khôi phục dấu bằng từ vựng của kho → 4 câu qua). Khôi phục CHỈ theo cặp
// từ liền nhau (từ đơn đoán sai nghĩa: "do" → "đo", "ve" → "vẽ" — soát độc lập 24/10) bằng từ vựng của CHÍNH các tài liệu người hỏi
// thấy: từ không đổi giữ nguyên từng byte; mã / email / đường dẫn / từ viết hoa toàn bộ không đụng; cặp không ghép qua dấu câu; câu đã có
// dấu không đổi; chữ chỉ có trong tài liệu ngoài vai không vào câu (không lộ qua câu gửi đi lẫn vết).
func TestKhoiPhucDauCauKhongDau(t *testing.T) {
	ix := NewMemoryIndex("t", khoLai())
	ix.Now = func() time.Time { return time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC) }
	ks := ksLai("")
	gd := contracts.Principal{ID: "g", TenantID: "t", Kind: contracts.PrincipalUser, Roles: []string{"giam-doc"}}
	thay := func(pr contracts.Principal) map[*Document]bool {
		vis := map[*Document]bool{}
		for _, d := range ix.docs {
			vis[d] = ix.visible(d, pr)
		}
		return vis
	}
	for _, c := range []struct {
		pr       contracts.Principal
		q, want  string
		restored bool
	}{
		{ks, "Ai lap bien ban nghiem thu?", "Ai lap biên bản nghiệm thu?", true},                         // "ai lập" không là cặp có trong kho
		{ks, "kiem tra gian giao truoc moi ca the nao", "kiểm tra giàn giáo trước mỗi ca the nao", true}, // "the nao" không có trong kho
		{ks, "Ai lập biên ban nghiem thu?", "Ai lập biên ban nghiem thu?", false},                        // đã có dấu: không đụng
		{ks, "ddieuf 11 la gi", "ddieuf 11 la gi", false},                                                // không từ nào có trong kho
		{ks, "ho so luu ket sat", "ho so luu ket sat", false},                                            // cặp nào cũng chỉ có ở MAT (vai khác)
		{gd, "ho so luu ket sat", "hồ sơ lưu két sắt", true},
	} {
		got, _, ok := ix.restoreDiacritics(c.q, thay(c.pr))
		if got != c.want || ok != c.restored {
			t.Errorf("%v %q: được %q (%v), muốn %q (%v)", c.pr.Roles, c.q, got, ok, c.want, c.restored)
		}
	}
	// Cặp từ quyết định: "bàn" (bàn giao) gặp nhiều hơn "bản" (biên bản) nhưng "bien ban" là cặp "biên bản"; từ đơn không đổi; chữ hoa đầu
	// từ giữ; mã / email / từ viết hoa toàn bộ và mọi ký tự giữa các từ giữ nguyên từng byte; cặp không ghép qua dấu phẩy.
	bg := NewMemoryIndex("t", []Document{{ID: "BG", Tenant: "t", Title: "Bàn giao", ACL: []string{"ks"},
		Body: "1. Bàn giao mặt bằng.\n2. Bàn giao hồ sơ.\n3. Bàn giao thiết bị.\n4. Lập biên bản.\n5. Bán giao dịch.\n6. Kiểm tra giàn. Giáo dục an toàn."}})
	all := map[*Document]bool{bg.docs[0]: true}
	// "bàn giao" 3 lần thắng "bán giao" 1 lần; "giàn. Giáo" cách bằng dấu chấm không phải cặp; từ dính ".xlsx" không đụng.
	for q, want := range map[string]string{"Bien ban do ai lap?": "Biên bản do ai lap?", "ban giao mat bang": "bàn giao mặt bằng", "Ban nao lap?": "Ban nao lap?",
		"Bien ban theo TT-NHNN cho EX-03, gui nguyen.van@x.vn": "Biên bản theo TT-NHNN cho EX-03, gui nguyen.van@x.vn",
		"MAT BANG ban giao": "MAT BANG bàn giao", "gui tep mat bang.xlsx": "gui tep mat bang.xlsx", "gian giao": "gian giao", "bien, ban giao": "bien, bàn giao", "DA-07  ban   giao": "DA-07  bàn   giao"} {
		if got, _, _ := bg.restoreDiacritics(q, all); got != want {
			t.Errorf("%q: được %q, muốn %q", q, got, want)
		}
	}
	// Soát lại 24/10: cặp chồng mà dạng thắng mâu thuẫn từ đã gán thì bỏ cả cặp ("nghiệm thu" rồi "thụ đo" ⇒ không thành "thu đo");
	// hoà điểm không đoán ("có số" 1 : "cơ sở" 1); đường dẫn kiểu "D:\…\" không đụng; câu quá 64 từ không khôi phục.
	mt := NewMemoryIndex("t", []Document{{ID: "MT", Tenant: "t", Title: "Nghiệm thu", ACL: []string{"ks"},
		Body: "1. Nghiệm thu công trình.\n2. Cảm thụ đo lường.\n3. Cảm thụ đo đạc.\n4. Thu độ ẩm.\n5. Có số liệu.\n6. Cơ sở vật chất."}})
	mtVis := map[*Document]bool{mt.docs[0]: true}
	for q, want := range map[string]string{"nghiem thu do ai ky": "nghiệm thu do ai ky", "co so phap ly": "co so phap ly", `D:\Bien ban\nghiem thu`: `D:\Bien ban\nghiem thu`,
		strings.Repeat("nghiem thu ", 33): strings.Repeat("nghiem thu ", 33)} {
		if got, _, _ := mt.restoreDiacritics(q, mtVis); got != want {
			t.Errorf("%.40q: được %.60q, muốn %.60q", q, got, want)
		}
	}
	// Lần gọi đầu đồng thời (từ vựng dựng lười một lần): chạy với -race.
	moi := NewMemoryIndex("t", khoLai())
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); moi.restoreDiacritics("kiem tra gian giao", thay(ks)) }()
	}
	wg.Wait()
	if got, _, _ := ix.restoreDiacritics("ho so luu ket sat", thay(ks)); strings.Contains(got, "két") || strings.Contains(got, "sắt") || strings.Contains(got, "hồ sơ") {
		t.Errorf("chữ chỉ có trong tài liệu ngoài vai lọt vào câu khôi phục: %q", got)
	}

	// Đường lai: reranker chỉ chấm cao khi câu đưa vào CÓ dấu ⇒ câu không dấu chỉ qua ngưỡng nhờ khôi phục; vết ghi câu đã khôi phục.
	lai := chiMucLai(t, khoLai(), &nhungGia{khaiNiem: khaiNiemThu})
	lai.Semantic.Reranker, lai.Semantic.MinScore = xepCanDau{}, 0.5
	var rep contracts.RetrievalReport
	ev, err := lai.Retrieve(contracts.WithKnowledgeQuery(context.Background(), contracts.KnowledgeQuery{Report: &rep}), ks, "Ai lap bien ban nghiem thu?", 3)
	if err != nil || len(ev) == 0 || rep.Restored != "bien → biên, ban → bản, nghiem → nghiệm" || rep.Silence != "" {
		t.Errorf("lai, câu không dấu: %v %v %+v", ids(ev), err, rep)
	}
	rep = contracts.RetrievalReport{}
	if ev, err := lai.Retrieve(contracts.WithKnowledgeQuery(context.Background(), contracts.KnowledgeQuery{Report: &rep}), ks, "Ai lập biên bản nghiệm thu?", 3); err != nil || len(ev) == 0 || rep.Restored != "" {
		t.Errorf("lai, câu có dấu: không khôi phục, được %v %v %+v", ids(ev), err, rep)
	}
}

// xepCanDau là reranker giả: chấm 0,9 cho đoạn có "nghiệm thu" khi câu hỏi đưa vào CÓ cụm đó (có dấu), 0,1 cho mọi thứ khác.
type xepCanDau struct{}

func (xepCanDau) Rerank(_ context.Context, q string, docs []string) ([]float64, error) {
	out := make([]float64, len(docs))
	for i, d := range docs {
		out[i] = 0.1
		if strings.Contains(strings.ToLower(q), "nghiệm thu") && strings.Contains(strings.ToLower(d), "nghiệm thu") {
			out[i] = 0.9
		}
	}
	return out, nil
}

// Chi phí khôi phục dấu theo cỡ kho (mỗi câu không dấu duyệt mọi đoạn người hỏi thấy):
// go test -run '^$' -bench KhoiPhucDau -benchtime 20x ./intelligence/rag/
func BenchmarkKhoiPhucDau(b *testing.B) {
	goc, err := LoadMarkdown("../../tenants/construction/knowledge/s7/kho", "abc-infra")
	if err != nil {
		b.Fatal(err)
	}
	for _, n := range []int{6, 60, 300} { // × 17 tệp ≈ 100 / 1.000 / 5.100 tài liệu (như BenchmarkTruyXuatBoNho)
		var docs []Document
		for i := range n {
			for _, d := range goc {
				d.ID = fmt.Sprintf("%s-%05d", d.ID, i)
				docs = append(docs, d)
			}
		}
		ix := NewMemoryIndex("abc-infra", docs)
		vis := map[*Document]bool{}
		for _, d := range ix.docs {
			vis[d] = true
		}
		b.Run(fmt.Sprintf("%dtai-lieu/dung-tu-vung", len(docs)), func(b *testing.B) { // lần câu không dấu đầu tiên trả
			for range b.N {
				NewMemoryIndex("abc-infra", docs).vocabs()
			}
		})
		b.Run(fmt.Sprintf("%dtai-lieu", len(docs)), func(b *testing.B) {
			for range b.N {
				ix.restoreDiacritics("lech dau giua kho xuat voi may dung thi may tieng phai co bien ban?", vis)
			}
		})
	}
}
