package ragcases

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/evals"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
	"go.yaml.in/yaml/v3"
)

const (
	caDir     = "../../tenants/construction/knowledge/ca"
	casesYAML = "../../tenants/construction/evals/cases.yaml"
	minCa     = 5 // phép quét ra ít hơn số này là hỏng, không phải xanh
)

// kyVong là tệp ky-vong.yaml của một ca.
type kyVong struct {
	Ca       string `yaml:"ca"`
	Nguon    string `yaml:"nguon"` // dữ liệu mẫu lấy từ đâu trong ca, phần nào giả lập
	Hoi      string `yaml:"hoi"`   // phải trùng nguyên văn input của ca
	NgayHoi  string `yaml:"ngay_hoi"`
	NguoiHoi struct {
		Tenant string   `yaml:"tenant"` // rỗng = tenant của ca
		Vai    []string `yaml:"vai"`
		PhamVi []string `yaml:"pham_vi"`
	} `yaml:"nguoi_hoi"`
	K      int    `yaml:"k"`
	KetQua string `yaml:"ket_qua"` // bang_chung | xung_dot | khong_hieu_luc | khong_co
	Kiem   []kiem `yaml:"kiem"`
}

// kiem là MỘT phép kiểm. do_biet_truoc khác rỗng: phép kiểm đang đỏ có chủ đích (lý do + nấc sẽ sửa) —
// nó phải còn đỏ; xanh lên thì báo để gỡ cờ.
type kiem struct {
	DoanDauChua      string   `yaml:"doan_dau_chua"`
	KhongDoanNaoChua string   `yaml:"khong_doan_nao_chua"`
	ChiBan           []string `yaml:"chi_ban"` // mọi đoạn phải thuộc các "<id>@v<bản>" này
	BanTrongThongBao []int    `yaml:"ban_trong_thong_bao"`
	DoBietTruoc      string   `yaml:"do_biet_truoc"`
}

func (k kiem) String() string {
	switch {
	case k.DoanDauChua != "":
		return fmt.Sprintf("đoạn đầu chứa %q", k.DoanDauChua)
	case k.KhongDoanNaoChua != "":
		return fmt.Sprintf("không đoạn nào chứa %q", k.KhongDoanNaoChua)
	case len(k.ChiBan) > 0:
		return fmt.Sprintf("chỉ bản %v", k.ChiBan)
	default:
		return fmt.Sprintf("thông báo nêu bản %v", k.BanTrongThongBao)
	}
}

// dat báo phép kiểm đạt không, kèm mô tả cái thấy được.
func (k kiem) dat(ev []contracts.Evidence, n *rag.VersionNotice) (bool, string) {
	switch {
	case k.DoanDauChua != "":
		if len(ev) == 0 {
			return false, "không có đoạn nào"
		}
		return strings.Contains(ev[0].Text, k.DoanDauChua), ev[0].Text
	case k.KhongDoanNaoChua != "":
		for _, e := range ev {
			if strings.Contains(e.Text, k.KhongDoanNaoChua) {
				return false, e.ID + ": " + e.Text
			}
		}
		return true, ""
	case len(k.ChiBan) > 0:
		for _, e := range ev {
			if ban, _, _ := strings.Cut(e.ID, "#"); !slices.Contains(k.ChiBan, ban) {
				return false, e.ID
			}
		}
		return len(ev) > 0, fmt.Sprint(len(ev), " đoạn")
	default:
		if n == nil {
			return false, "không có thông báo"
		}
		var got []int
		for _, v := range n.Versions {
			got = append(got, v.Version)
		}
		return slices.Equal(got, k.BanTrongThongBao), fmt.Sprint(got)
	}
}

func TestCaTriThuc(t *testing.T) {
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
		t.Fatalf("quét ra %d ca ở %s, cần ít nhất %d: phép quét rỗng làm mọi kiểm tra sau xanh giả", len(files), caDir, minCa)
	}
	for _, f := range files {
		dir := filepath.Dir(f)
		t.Run(filepath.Base(dir), func(t *testing.T) { chayCa(t, dir, f, byID) })
	}
}

// caDaNap là một ca đã đọc xong: kỳ vọng, tài liệu mẫu, người hỏi, ngày hỏi.
type caDaNap struct {
	kv   kyVong
	docs []rag.Document
	pr   contracts.Principal
	day  time.Time
}

// napCa đọc và kiểm một ca: tệp kỳ vọng hợp lệ, câu hỏi trùng nguyên văn cases.yaml, mỗi phép kiểm đúng một loại.
func napCa(t *testing.T, dir, f string, byID map[string]evals.Case) caDaNap {
	t.Helper()
	b, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	var kv kyVong
	dec := yaml.NewDecoder(strings.NewReader(string(b)))
	dec.KnownFields(true)
	if err := dec.Decode(&kv); err != nil {
		t.Fatalf("ky-vong.yaml: %v", err)
	}
	c, ok := byID[kv.Ca]
	if !ok || kv.Ca != filepath.Base(dir) {
		t.Fatalf("ca %q không có trong cases.yaml hoặc khác tên thư mục", kv.Ca)
	}
	if in := fmt.Sprint(c["input"]); kv.Hoi != in {
		t.Fatalf("câu hỏi lệch cases.yaml:\n  ky-vong: %q\n  ca:      %q", kv.Hoi, in)
	}
	for i, k := range kv.Kiem {
		if n := countSet(k); n != 1 {
			t.Fatalf("kiem[%d] phải có đúng một phép kiểm, có %d", i, n)
		}
	}
	tenant := fmt.Sprint(c["tenant"])
	docs, err := rag.LoadMarkdown(dir, tenant)
	if err != nil {
		t.Fatal(err)
	}
	pr := contracts.Principal{ID: "nguoi-hoi", TenantID: contracts.TenantID(tenant), Kind: contracts.PrincipalUser, Roles: kv.NguoiHoi.Vai,
		Attributes: map[string]string{"entity_scope": strings.Join(kv.NguoiHoi.PhamVi, ",")}}
	if kv.NguoiHoi.Tenant != "" {
		pr.TenantID = contracts.TenantID(kv.NguoiHoi.Tenant)
	}
	day, err := time.Parse("2006-01-02", kv.NgayHoi)
	if err != nil {
		t.Fatalf("ngay_hoi: %v", err)
	}
	return caDaNap{kv, docs, pr, day}
}

// phanLoai đổi kết quả truy xuất thành nhãn ket_qua của tệp kỳ vọng.
func phanLoai(ev []contracts.Evidence, err error) (string, *rag.VersionNotice, error) {
	var n *rag.VersionNotice
	switch {
	case errors.As(err, &n) && n.Kind == rag.NoticeConflict:
		return "xung_dot", n, nil
	case errors.As(err, &n) && n.Kind == rag.NoticeNoEffective:
		return "khong_hieu_luc", n, nil
	case err != nil:
		return "", nil, err
	case len(ev) > 0:
		return "bang_chung", nil, nil
	}
	return "khong_co", nil, nil
}

func chayCa(t *testing.T, dir, f string, byID map[string]evals.Case) {
	ca := napCa(t, dir, f, byID)
	ix := rag.NewMemoryIndex(string(ca.pr.TenantID), ca.docs)
	ix.Now = func() time.Time { return ca.day }
	ev, err := ix.Retrieve(context.Background(), ca.pr, ca.kv.Hoi, ca.kv.K)
	got, n, err := phanLoai(ev, err)
	if err != nil {
		t.Fatalf("lỗi truy xuất: %v", err)
	}
	if got != ca.kv.KetQua {
		t.Fatalf("kết quả %s, muốn %s (đoạn %v)", got, ca.kv.KetQua, ids(ev))
	}
	for _, k := range ca.kv.Kiem {
		ok, seen := k.dat(ev, n)
		switch {
		case k.DoBietTruoc == "" && !ok:
			t.Errorf("trượt: %s — thấy %s", k, seen)
		case k.DoBietTruoc != "" && ok:
			t.Errorf("đỏ biết trước nay đã xanh, gỡ do_biet_truoc: %s", k)
		case k.DoBietTruoc != "":
			t.Logf("đỏ biết trước: %s — thấy %s — %s", k, seen, k.DoBietTruoc)
		}
	}
}

func countSet(k kiem) int {
	n := 0
	for _, set := range []bool{k.DoanDauChua != "", k.KhongDoanNaoChua != "", len(k.ChiBan) > 0, len(k.BanTrongThongBao) > 0} {
		if set {
			n++
		}
	}
	return n
}

func ids(ev []contracts.Evidence) []string {
	var out []string
	for _, e := range ev {
		out = append(out, e.ID)
	}
	return out
}
