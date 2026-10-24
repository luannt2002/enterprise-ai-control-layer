package rag

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Vết của chặng truy xuất (ROADMAP-RAG §6.7): đường truy xuất, số bản tài liệu / đoạn người hỏi THẤY, số bản thấy được mà không làm căn
// cứ được ở ngày hỏi, lý do "im" nêu đúng luật đã chặn. Tài liệu ngoài vai / phạm vi không được đếm ở đâu: người hỏi đọc được vết của
// chính mình, đếm chúng là cho biết có tài liệu họ không được xem ("không có" = "không được xem", §6.2).
func TestVetTruyXuat(t *testing.T) {
	hoi := func(ix *MemoryIndex, pr contracts.Principal, q string) ([]contracts.Evidence, contracts.RetrievalReport) {
		t.Helper()
		var rep contracts.RetrievalReport
		ctx := contracts.WithKnowledgeQuery(context.Background(), contracts.KnowledgeQuery{Topic: q, Report: &rep})
		ev, err := ix.Retrieve(ctx, pr, q, 3)
		if n := (*contracts.KnowledgeNotice)(nil); err != nil && !errors.As(err, &n) {
			t.Fatalf("%q: %v", q, err)
		}
		return ev, rep
	}
	thay := func(ix *MemoryIndex, pr contracts.Principal) int { // số đoạn người hỏi thấy, đếm độc lập với mã đang kiểm
		n := 0
		for i := range ix.chunks {
			if ix.visible(ix.chunks[i].doc, pr) {
				n++
			}
		}
		return n
	}
	ks := ksLai("")
	ix := NewMemoryIndex("t", khoLai())
	ix.Now = func() time.Time { return time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC) }

	// Trúng: NT v1 hết hiệu lực + NT v3 bản nháp = 2 bản không làm căn cứ được; BT (phạm vi khác), MAT, QC v2 (vai khác) không đếm.
	ev, rep := hoi(ix, ks, "Vữa trộn tại chỗ theo mác nào?")
	if len(ev) == 0 || rep != (contracts.RetrievalReport{Path: "từ khoá", Docs: 6, Chunks: thay(ix, ks), NotServable: 2}) {
		t.Errorf("trúng: %v %+v", ids(ev), rep)
	}
	// Chủ đề ngoài kho: im theo luật chủ đề.
	if ev, rep := hoi(ix, ks, "Thủ tục xin giấy phép xây dựng là gì?"); len(ev) != 0 || !strings.Contains(rep.Silence, "chủ đề") {
		t.Errorf("chủ đề ngoài kho: %v %+v", ids(ev), rep)
	}
	// Câu chỉ tài liệu ngoài vai trả lời được: vết y như câu ngoài kho — không số đếm, không chữ nào của tài liệu bị cấm.
	ev, rep = hoi(ix, ks, "Hồ sơ lưu két sắt ở đâu?")
	if len(ev) != 0 || rep.Docs != 6 || !strings.Contains(rep.Silence, "chủ đề") {
		t.Errorf("tài liệu ngoài vai: %v %+v", ids(ev), rep)
	}
	for _, cam := range []string{"MAT", "két", "mật", "giam-doc"} {
		if strings.Contains(rep.Path+rep.Silence, cam) {
			t.Errorf("vết lộ tài liệu ngoài vai (%q): %+v", cam, rep)
		}
	}
	// Bằng nhau tuyệt đối với kho KHÔNG có tài liệu ngoài vai: tài liệu bị cấm không đổi một số / một chữ nào của vết.
	var khongMat []Document
	for _, d := range khoLai() {
		if d.ID != "MAT" {
			khongMat = append(khongMat, d)
		}
	}
	ix2 := NewMemoryIndex("t", khongMat)
	ix2.Now = ix.Now
	for _, q := range []string{"Hồ sơ lưu két sắt ở đâu?", "Vữa trộn tại chỗ theo mác nào?", "Ai lập biên bản bàn giao hồ sơ?"} {
		_, a := hoi(ix, ks, q)
		_, b := hoi(ix2, ks, q)
		if a != b {
			t.Errorf("%q: vết đổi theo tài liệu ngoài vai:\n  có MAT   %+v\n  không MAT %+v", q, a, b)
		}
	}
	// Đoạn khớp mà xếp sau hạng k: đếm để vết nêu "bị loại vì ngân sách" (§6.7) — cả đường từ khoá lẫn đường lai.
	sauK := func(ix *MemoryIndex, q string) ([]contracts.Evidence, contracts.RetrievalReport) {
		t.Helper()
		var rep contracts.RetrievalReport
		ev, err := ix.Retrieve(contracts.WithKnowledgeQuery(context.Background(), contracts.KnowledgeQuery{Report: &rep}), ks, q, 1)
		if err != nil {
			t.Fatalf("%q: %v", q, err)
		}
		return ev, rep
	}
	if ev, rep := sauK(ix, "Biên bản kiểm tra vữa trộn theo mác?"); len(ev) != 1 || rep.Beyond < 1 {
		t.Errorf("từ khoá, k = 1: muốn 1 đoạn + ≥ 1 đoạn xếp sau, được %v %+v", ids(ev), rep)
	}
	gd := contracts.Principal{ID: "g", TenantID: "t", Kind: contracts.PrincipalUser, Roles: []string{"giam-doc"}}
	if ev, rep := hoi(ix, gd, "Hồ sơ lưu két sắt ở đâu?"); len(ev) == 0 || rep.Docs != 2 || rep.Silence != "" {
		t.Errorf("giám đốc thấy MAT, QC v2: %v %+v", ids(ev), rep)
	}
	// Không đoạn nào đạt ngưỡng của đường từ khoá.
	ix.MinScore = 100
	if ev, rep := hoi(ix, ks, "Vữa trộn tại chỗ theo mác nào?"); len(ev) != 0 || !strings.Contains(rep.Silence, "ngưỡng từ khoá") {
		t.Errorf("dưới ngưỡng từ khoá: %v %+v", ids(ev), rep)
	}

	// Đường lai: im theo điểm xếp lại / cosine, nêu điểm đầu và ngưỡng.
	lai := chiMucLai(t, khoLai(), &nhungGia{khaiNiem: khaiNiemThu})
	lai.Semantic.Reranker, lai.Semantic.MinScore = xepGia{uuTien: "tư vấn giám sát"}, 0.95
	ev, rep = hoi(lai, ks, "Ai lập văn bản bàn giao?")
	if len(ev) != 0 || rep.Path != "lai, xếp lại" || !strings.Contains(rep.Silence, "0.900") || !strings.Contains(rep.Silence, "0.950") {
		t.Errorf("lai, điểm xếp lại dưới ngưỡng: %v %+v", ids(ev), rep)
	}
	lai.Semantic.Reranker, lai.Semantic.MinScore = nil, 0.5
	if ev, rep := hoi(lai, ks, "Thủ tục xin giấy phép là gì?"); len(ev) != 0 || rep.Path != "lai, cosine" || !strings.Contains(rep.Silence, "cosine") {
		t.Errorf("lai, cosine dưới ngưỡng: %v %+v", ids(ev), rep)
	}
	if ev, rep := hoi(lai, ks, "Ai lập văn bản bàn giao?"); len(ev) == 0 || rep.Silence != "" || rep.Docs != 6 {
		t.Errorf("lai, trúng: %v %+v", ids(ev), rep)
	}
	lai.Semantic.MinScore = 0
	ev, khongNguong := sauK(lai, "Biên bản kiểm tra vữa trộn theo mác?")
	if len(ev) != 1 || khongNguong.Beyond < 1 {
		t.Errorf("lai, k = 1, ngưỡng 0: muốn 1 đoạn + ≥ 1 đoạn xếp sau, được %v %+v", ids(ev), khongNguong)
	}
	lai.Semantic.MinScore = 0.5 // chỉ đếm đoạn đạt ngưỡng (soát độc lập 24/10): ngưỡng cao hơn ⇒ ít đoạn "xếp sau" hơn
	if _, coNguong := sauK(lai, "Biên bản kiểm tra vữa trộn theo mác?"); coNguong.Beyond >= khongNguong.Beyond {
		t.Errorf("lai: đoạn dưới ngưỡng không được đếm là đoạn khớp xếp sau: ngưỡng 0 → %d, ngưỡng 0.5 → %d", khongNguong.Beyond, coNguong.Beyond)
	}
}
