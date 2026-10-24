package eais_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"testing"
	"unicode"
)

// Cổng tên hãng (F-015; CLAUDE.md §30): mã chỉ biết GIAO THỨC — tên hãng, endpoint, tên model chỉ ở .env. Quét mọi tệp git đang theo
// dõi có đuôi .go .sh .py .yaml .yml .json, trừ docs/ và testdata/ (dữ liệu đóng băng), so TỪNG TỪ (chữ thường) với danh sách
// sha256 dưới đây — chính cổng không chứa tên hãng nào. Từ là dãy chữ / số; thêm các mảnh CamelCase và chữ ↔ số của nó ("xyzClient",
// "xyz3"), nên tên dính trong định danh hay mã model vẫn bị bắt; tên nằm giữa một từ dài hơn thì không.
//
// Thêm một tên: printf '%s' <tên viết thường> | sha256sum. Tên trùng tên tệp hợp đồng agent ở gốc repo và thư mục công cụ của nó
// KHÔNG nằm trong danh sách (mọi nhắc tệp đó sẽ đỏ oan); mã model của dòng đó vẫn bị bắt qua tên phiên bản.
var tenHangSHA256 = []string{
	"053ea4804ef1bb33d4a3d6fb024a614b6d257cebc2bc7cd915da9c9522f37ffc",
	"0b79cf2de0ee6b8ee227d9ebfe69b91aa92b1d2509641c4b743518447dbc4c68",
	"2a62945e88351cb2380621f1fef5a480a05ff933fe6c6b2fafac3ea06dee8291",
	"3669ffb921cca6ee1edfe0f1e620fc2379771e9a8bd92659904540e984f3209e",
	"45d780f82c9247fc1142f8b60c12b6c6cd3d9f23dd836cfa6e84c498d2f83292",
	"4d6f3311104439e576b628d5bf82b00dbb7ea0affd3d14d17b727eec01372a5c",
	"531e67a59846756e7b8c8ba5b7e33ae8991351b0c656f95171409fbda8115de8",
	"5a5110ebe1544b31e6d4207a419bc1deb148735f89ae984ff0a2ab809b51f1df",
	"5d72436256ada53828b51895a94bb8489e9f1ac4fe937a8024ef1594e7045ff6",
	"665e05102ac943a0667937bcf276cde5beb523de08f59c03c2e4a487215a1cf5",
	"67f2d22514622d1be30c14ee9f3cb104503a159a33a656ca91a1953aa9616429",
	"6f7ac1823da81d2e52d1a1549ee69c85bbf8bb56d06682849e7c09da2785ce3b",
	"72e8088a05b38ae3cffbc126c1c2095046c7700edab31ce35867ce8ee3ac8932",
	"76e3c7bfe641ea125c0c2e1c5f89349e17b352ed128d528de2443794e7acf870",
	"79f43c3b9c23fbdf293bdba80dab38ba194fc1da79187091b3d3760dad840bf8",
	"7d3194f79e645c42e4396dda38be04766810ec6a00d00aced3ffc2a0a1f1a9ef",
	"8bdc3c22e340202bfd1c2dd177012ba9ebc208a7437740eb8a835a225f41bcf2",
	"920510199770f4d65cb8aaa2cd12bdb2b8c37f5b3907c50a734a0e3409da2823",
	"b5d641b432cca81e57b5bb5b73c53e64c50d44dce93595a85d1d8885f48548e8",
	"c70eca6b0f88f44d81a41311647e50fda1ac454ec04ffd442b0eb4743a993131",
	"c9ad8f2cc1294afa0ef22fc2c019ff7243cdd272b2147ddca3f34c5036b05768",
	"d203a277f19a8f7d78fe8e9a621534b2cbd8ef1f0d809908ffc481e778858872",
	"e12ce8285efc67c6d93d3a122e2589ed95089bcbb775ba5634d94e2b8385db07",
	"fb49d6d5dd712290f262eafa7238e32e6e645612e6fc8680e1ee4150e057cbf3",
	"fc5a1047f5919892fcdf8aa79ea5d6bb6531b5c176939ef0110906cb225941c1",
}

// tenHangChoPhep: tệp được tạm giữ tên hãng, kèm lý do. Mục không còn tên hãng nào là mục thừa (đỏ) — để tên không quay lại lặng lẽ.
var tenHangChoPhep = map[string]string{
	"cmd/eais/unit_test.go": "ngoài lát L4: cổng 'không có đường gọi model thật' liệt kê chuỗi cấm bằng tên; đổi sang băm rồi bỏ mục",
	"scripts/du-toan-g7.py": "ngoài lát L4: dự toán G7 ghi tên + giá model; chuyển sang EAIS_G7_MODEL / EAIS_G7_PRICES rồi bỏ mục",
}

var duoiQuet = regexp.MustCompile(`\.(go|sh|py|ya?ml|json)$`)

// tuCuaChu trả mọi từ (chữ thường) của s: dãy chữ / số, cộng các mảnh tách ở chỗ chữ thường → chữ hoa và chữ ↔ số.
func tuCuaChu(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		rs := []rune(w)
		out[strings.ToLower(w)] = true
		start := 0
		for i := 1; i <= len(rs); i++ {
			if i == len(rs) || (unicode.IsLower(rs[i-1]) && unicode.IsUpper(rs[i])) || unicode.IsDigit(rs[i-1]) != unicode.IsDigit(rs[i]) {
				out[strings.ToLower(string(rs[start:i]))] = true
				start = i
			}
		}
	}
	return out
}

// tenHangTrong trả các từ của đường dẫn rel và nội dung b có băm nằm trong cam, đã sắp xếp.
func tenHangTrong(rel string, b []byte, cam map[string]bool) []string {
	var out []string
	for w := range tuCuaChu(rel + "\n" + string(b)) {
		h := sha256.Sum256([]byte(w))
		if cam[hex.EncodeToString(h[:])] {
			out = append(out, w)
		}
	}
	slices.Sort(out)
	return out
}

func tapBam(t *testing.T, list []string) map[string]bool {
	t.Helper()
	cam := map[string]bool{}
	for _, h := range list {
		if len(h) != 64 || strings.Trim(h, "0123456789abcdef") != "" || cam[h] {
			t.Fatalf("băm sai dạng hoặc trùng: %q", h)
		}
		cam[h] = true
	}
	return cam
}

func TestKhongTenHangTrongMa(t *testing.T) {
	cam := tapBam(t, tenHangSHA256)
	out, err := exec.Command("git", "ls-files", "-z").Output()
	if err != nil {
		t.Fatalf("git ls-files: %v (cổng cần cây git)", err)
	}
	quet, gap := 0, map[string]bool{}
	for _, rel := range strings.Split(string(out), "\x00") {
		if !duoiQuet.MatchString(rel) || strings.HasPrefix(rel, "docs/") || strings.HasPrefix(rel, "testdata/") || strings.Contains(rel, "/testdata/") {
			continue
		}
		b, err := os.ReadFile(rel)
		if err != nil {
			t.Fatal(err)
		}
		quet++
		ten := tenHangTrong(rel, b, cam)
		if len(ten) == 0 {
			continue
		}
		if _, ok := tenHangChoPhep[rel]; ok {
			gap[rel] = true
			continue
		}
		t.Errorf("%s: tên hãng %q trong mã — chỉ được ở .env (mã chỉ biết giao thức)", rel, ten)
	}
	if quet < 100 { // rỗng không phải xanh
		t.Fatalf("chỉ quét %d tệp", quet)
	}
	for rel, why := range tenHangChoPhep {
		if !gap[rel] {
			t.Errorf("%s: mục cho phép thừa (không còn tên hãng) — bỏ mục: %s", rel, why)
		}
	}
}

// Ca âm cho cách tách từ, với một "tên hãng" giả (băm tính lúc chạy): bắt được ở mọi cách viết thường gặp trong mã; không bắt khi tên
// chỉ là một phần của từ dài hơn. Giới hạn đã biết (ca cuối của nhóm không bắt): tên có chữ hoa ở giữa ("HangXyz") dính trong một định
// danh dài hơn bị tách thành hai mảnh.
func TestTenHangCachTachTu(t *testing.T) {
	h := sha256.Sum256([]byte("hangxyz"))
	cam := tapBam(t, []string{hex.EncodeToString(h[:])})
	for _, s := range []string{`provider := "hangxyz"`, "HANGXYZ_API_KEY", "https://api.hangxyz.com", "hangxyzClient", "newHangxyzClient",
		"hangxyz-pro-5-5", "hangxyz5", "// Hangxyz", "x.HangXyz"} {
		if got := tenHangTrong("a.go", []byte(s), cam); !slices.Equal(got, []string{"hangxyz"}) {
			t.Errorf("%q: được %q, muốn bắt", s, got)
		}
	}
	for _, s := range []string{"hangxyzabc", "abchangxyz", "hang xyz", "newHangXyzClient"} {
		if got := tenHangTrong("a.go", []byte(s), cam); len(got) != 0 {
			t.Errorf("%q: bắt oan %q", s, got)
		}
	}
	if got := tenHangTrong("cmd/hangxyz/x.go", nil, cam); len(got) != 1 {
		t.Errorf("tên hãng trong đường dẫn phải bị bắt, được %q", got)
	}
}
