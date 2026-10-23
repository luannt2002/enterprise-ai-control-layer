package ragcases

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/evals"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
)

// Phép so ở mức bằng chứng (KE-HOACH U8): bản rẻ của luận đề gốc (docs/04-core-mvp.md §1, "hệ tất định có RAG bên
// trong đáng tin hơn hệ lấy RAG làm trung tâm"). Cùng câu hỏi, cùng tài liệu; ba cách chọn thứ đưa cho model. Không
// gọi model — chỉ đo model SẼ THẤY gì:
//   - eais: chỉ mục như chạy thật (phiên bản / trạng thái / hiệu lực / phạm vi / thông báo);
//   - rag: "lấy RAG làm trung tâm" — cùng bộ tìm từ khoá, có tenant và ACL theo vai, nhưng không biết phiên bản, hiệu
//     lực, trạng thái, phạm vi (như một chatbot RAG thường);
//   - nap-thang: nạp thẳng mọi tài liệu người hỏi đọc được theo vai, không chọn đoạn.
//
// "Không làm căn cứ được" = bản hết hiệu lực / chưa duyệt / bị giữ / đã thay thế vào ngày hỏi, bản đang xung đột, hoặc
// ngoài phạm vi của người hỏi. Bộ ca còn nhỏ (5 ca): đây là bằng chứng minh hoạ có lệnh chạy lại, không phải số thống kê.
func TestSoSanhBaHe(t *testing.T) {
	cases, err := evals.Load(casesYAML)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]evals.Case{}
	for _, c := range cases {
		byID[fmt.Sprint(c["id"])] = c
	}
	files, _ := filepath.Glob(filepath.Join(caDir, "*", "ky-vong.yaml"))
	if len(files) < minCa {
		t.Fatalf("quét ra %d ca, cần ít nhất %d", len(files), minCa)
	}
	tong := map[string][2]int{} // hệ → {số ca đạt, số tài liệu không làm căn cứ được đã lọt}
	var bang []string
	for _, f := range files {
		ca := napCa(t, filepath.Dir(f), f, byID)
		dungDuoc := canCu(ca)
		for _, he := range []string{"eais", "rag", "nap-thang"} {
			ev, err := chon(he, ca)
			got, _, err := phanLoai(ev, err)
			if err != nil {
				t.Fatalf("%s / %s: %v", ca.kv.Ca, he, err)
			}
			var lot []string
			for _, k := range nguon(ev) {
				if !dungDuoc[k] {
					lot = append(lot, k)
				}
			}
			dat := got == ca.kv.KetQua && len(lot) == 0
			v := tong[he]
			if dat {
				v[0]++
			}
			v[1] += len(lot)
			tong[he] = v
			bang = append(bang, fmt.Sprintf("%-10s %-9s kết quả %-14s đạt %-5v lọt %v", ca.kv.Ca, he, got, dat, lot))
			if he == "eais" && len(lot) > 0 {
				t.Errorf("%s: EAIS đưa tài liệu không làm căn cứ được cho model: %v", ca.kv.Ca, lot)
			}
		}
	}
	t.Logf("\n%s", strings.Join(bang, "\n"))
	for _, he := range []string{"eais", "rag", "nap-thang"} {
		t.Logf("TỔNG %-9s đạt %d/%d ca · %d lượt tài liệu không làm căn cứ được lọt tới model", he, tong[he][0], len(files), tong[he][1])
	}
	// Phép so phải phân biệt được: bộ ca mà hai hệ đối chứng cũng sạch thì phép so vô nghĩa.
	if tong["rag"][1] == 0 || tong["nap-thang"][1] == 0 {
		t.Errorf("hệ đối chứng không lọt tài liệu nào: bộ ca chưa đủ sức phân biệt (%v)", tong)
	}
}

// chon trả thứ hệ he đưa cho model.
func chon(he string, ca caDaNap) ([]contracts.Evidence, error) {
	tenant := string(ca.pr.TenantID)
	switch he {
	case "eais":
		ix := rag.NewMemoryIndex(tenant, ca.docs)
		ix.Now = func() time.Time { return ca.day }
		return ix.Retrieve(context.Background(), ca.pr, ca.kv.Hoi, ca.kv.K)
	case "rag":
		return rag.NewMemoryIndex(tenant, phangHoa(ca.docs)).Retrieve(context.Background(), ca.pr, ca.kv.Hoi, ca.kv.K)
	}
	var ev []contracts.Evidence // nạp thẳng: mọi tài liệu cùng tenant mà vai được đọc
	for _, d := range phangHoa(ca.docs) {
		if d.Tenant == tenant && slices.ContainsFunc(ca.pr.Roles, func(r string) bool { return slices.Contains(d.ACL, r) }) {
			ev = append(ev, contracts.Evidence{ID: d.ID, Kind: contracts.EvidencePassage, Text: d.Body,
				Source: contracts.SourceRef{SourceType: "document", Name: d.Title, DocumentID: d.ID}})
		}
	}
	return ev, nil
}

// phangHoa bỏ mọi hiểu biết về phiên bản: mỗi bản thành một tài liệu thường (mã "<id>@v<bản>" để còn truy ngược).
func phangHoa(docs []rag.Document) []rag.Document {
	out := make([]rag.Document, len(docs))
	for i, d := range docs {
		out[i] = rag.Document{ID: khoa(d), Tenant: d.Tenant, Title: d.Title, Body: d.Body, ACL: d.ACL}
	}
	return out
}

func khoa(d rag.Document) string {
	if d.Version == 0 {
		return d.ID
	}
	return fmt.Sprintf("%s@v%d", d.ID, d.Version)
}

// nguon trả các tài liệu (theo khoá "<id>@v<bản>") có mặt trong bằng chứng.
func nguon(ev []contracts.Evidence) []string {
	var out []string
	for _, e := range ev {
		k := e.Source.DocumentID
		if e.Source.DocumentVersion > 0 {
			k = fmt.Sprintf("%s@v%d", k, e.Source.DocumentVersion)
		}
		if !slices.Contains(out, k) {
			out = append(out, k)
		}
	}
	return out
}

// canCu tính độc lập với gói rag: bản nào làm căn cứ được cho người hỏi vào ngày hỏi — cùng tenant, vai trong ACL, phạm
// vi khớp (nếu khai), đã duyệt hoặc không khai, đang hiệu lực, và là bản DUY NHẤT như thế của mã đó.
func canCu(ca caDaNap) map[string]bool {
	today := ca.day.Format("2006-01-02")
	scope := strings.Split(ca.pr.Attributes["entity_scope"], ",")
	ok := map[string]bool{}
	dem := map[string]int{}
	for _, d := range ca.docs {
		thay := string(ca.pr.TenantID) == d.Tenant &&
			slices.ContainsFunc(ca.pr.Roles, func(r string) bool { return slices.Contains(d.ACL, r) }) &&
			(len(d.Scope) == 0 || slices.ContainsFunc(scope, func(s string) bool { return slices.Contains(d.Scope, strings.TrimSpace(s)) }))
		hieuLuc := (d.Status == "" || d.Status == rag.StatusApproved) &&
			(d.EffectiveFrom == "" || today >= d.EffectiveFrom) && (d.EffectiveTo == "" || today < d.EffectiveTo)
		if thay && hieuLuc {
			ok[khoa(d)] = true
			dem[d.ID]++
		}
	}
	for _, d := range ca.docs {
		if dem[d.ID] > 1 {
			ok[khoa(d)] = false // hai bản cùng hiệu lực: không bản nào làm căn cứ được
		}
	}
	return ok
}

// Phép so ba hệ trên bộ S7 phát triển (KE-HOACH U8a mở rộng): 5 ca ở trên chỉ xoay quanh phiên bản; bộ S7 có thêm quyền, phạm
// vi, bản nháp, câu không có đáp án. Cùng câu, cùng kho, cùng bộ đếm của S7 (quyền đọc tính lại độc lập, không dùng gói rag).
//   - eais: lượt hỏi thật (ngày hỏi, luật phạm vi của policy, chủ đề — qua ngữ cảnh);
//   - rag: RAG thường — cùng bộ tìm từ khoá, tenant + ACL theo vai, không biết phiên bản / hiệu lực / trạng thái / phạm vi / chủ đề;
//   - nap-thang: mọi tài liệu vai được đọc, nguyên văn (không có "top-3": chỉ đếm thứ lọt tới model).
func TestSoSanhBaHeTrenS7(t *testing.T) {
	bo, docs, rt := napS7(t)
	phang := phangHoa(docs)
	goc := func(ev []contracts.Evidence) []contracts.Evidence { // "<mã>@v<bản>" → mã + bản, để bộ đếm tra đúng bản trong kho
		for i := range ev {
			if id, v, ok := strings.Cut(ev[i].Source.DocumentID, "@v"); ok {
				ev[i].Source.DocumentID = id
				ev[i].Source.DocumentVersion, _ = strconv.Atoi(v)
			}
		}
		return ev
	}
	ixRag := rag.NewMemoryIndex("abc-infra", phang)
	he := map[string]s7Truy{
		"eais": rag.NewMemoryIndex("abc-infra", docs).Retrieve,
		"rag": func(_ context.Context, pr contracts.Principal, q string, k int) ([]contracts.Evidence, error) {
			ev, err := ixRag.Retrieve(context.Background(), pr, q, k) // không ngữ cảnh của lượt: RAG thường không có
			return goc(ev), err
		},
		"nap-thang": func(_ context.Context, pr contracts.Principal, _ string, _ int) ([]contracts.Evidence, error) {
			var ev []contracts.Evidence
			for _, d := range phang {
				if slices.ContainsFunc(pr.Roles, func(r string) bool { return slices.Contains(d.ACL, r) }) {
					ev = append(ev, contracts.Evidence{ID: d.ID + "#toan-van", Kind: contracts.EvidencePassage, Text: d.Body,
						Source: contracts.SourceRef{SourceType: "document", Name: d.Title, DocumentID: d.ID}})
				}
			}
			return goc(ev), nil
		},
	}
	kq := map[string]s7KetQua{}
	for _, ten := range []string{"eais", "rag", "nap-thang"} {
		kq[ten] = doS7(t, bo, docs, "abc-infra", rt, he[ten])
		x := kq[ten]
		top3 := fmt.Sprintf("%d/%d", x.trung, x.traLoi)
		if ten == "nap-thang" {
			top3 = "—"
		}
		t.Logf("%-9s top-3 %-6s · đoạn ngoài quyền %2d · đoạn không làm căn cứ (nháp / hết hiệu lực / xung đột) %2d · câu không có đáp án mà trả đoạn %d/%d",
			ten, top3, len(x.ngoaiQuyen), len(x.khongCanCu), x.traNham, len(bo.Cau)-x.traLoi)
	}
	if e := kq["eais"]; len(e.ngoaiQuyen)+len(e.khongCanCu) != 0 {
		t.Errorf("EAIS đưa tới model đoạn ngoài quyền / không làm căn cứ: %v %v", e.ngoaiQuyen, e.khongCanCu)
	}
	for _, ten := range []string{"rag", "nap-thang"} { // phép so phải phân biệt được
		if x := kq[ten]; len(x.ngoaiQuyen)+len(x.khongCanCu) == 0 {
			t.Errorf("hệ đối chứng %s không lọt gì: bộ S7 không đủ sức phân biệt", ten)
		}
	}
}
