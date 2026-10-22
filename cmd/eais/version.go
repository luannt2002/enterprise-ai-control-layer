package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"sync"
)

// buildVersion do trình liên kết điền: go build -ldflags "-X main.buildVersion=<mã>" (scripts/chay-thu.sh nhúng mã git của cây đang dựng,
// kèm "+dirty" khi cây có sửa chưa commit). Giữ nó là biến chuỗi KHÔNG khởi tạo: -X chỉ có tác dụng với biến chuỗi chưa khởi tạo hoặc khởi
// tạo bằng hằng chuỗi, và lặng lẽ bỏ qua mọi thứ khác — đổi tên / kiểu biến này mà không đổi script thì phiên bản mã quay về nguồn khác mà
// không lỗi nào báo (TestBuildVersionLaBienLienKet canh; chay-thu.sh cũng đối chiếu với phiên bản mà server báo lúc khởi động).
var buildVersion string

// binHashPrefix là tiền tố của phiên bản mã suy từ băm chính binary: phân biệt được với mã git (40 chữ số hex) khi đọc vết.
const binHashPrefix = "bin-sha256:"

// pickCodeVersion chọn phiên bản mã theo thứ tự ưu tiên:
//  1. linked — giá trị nhúng bằng -ldflags (người dựng chủ động chọn nên thắng cả bản sửa VCS);
//  2. bản sửa VCS mà `go build` tự ghi vào binary (vcs.revision, kèm "+dirty" khi vcs.modified);
//  3. phiên bản module thật của binary (vd cài bằng `go install pkg@v1.2.3`), nếu không phải "(devel)";
//  4. băm sha256 của CHÍNH binary, mang tiền tố binHashPrefix. Đây là chỗ trám cho binary dựng trong git worktree liên kết hay cây không có
//     .git: không VCS, không ldflags, mà vẫn phải phân biệt được hai binary khác mã — mã lần nạp đổi theo mã (loadIDOf), và "eais trace
//     diff" hai mốc mã vô nghĩa nếu cả hai bên đều in "(devel)".
//
// Không bao giờ trả "(devel)" hay rỗng; chỉ khi không đọc được cả binary mới trả "(unknown)". bi có thể nil (binary không mang build info).
func pickCodeVersion(linked string, bi *debug.BuildInfo, selfHash func() (string, error)) string {
	if v := strings.TrimSpace(linked); v != "" {
		return v
	}
	if bi != nil {
		rev, dirty := "", false
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				rev = s.Value
			case "vcs.modified":
				dirty = s.Value == "true"
			}
		}
		if rev != "" {
			if dirty {
				rev += "+dirty"
			}
			return rev
		}
		if v := bi.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	if sum, err := selfHash(); err == nil && sum != "" {
		return binHashPrefix + sum
	}
	return "(unknown)"
}

// selfSHA256 là băm sha256 (hex) của chính binary đang chạy (os.Executable).
func selfSHA256() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// codeVersionOnce tính phiên bản mã MỘT lần cho cả tiến trình: băm chính binary phải đọc cả tệp, mà buildRevision được gọi ở mọi lượt hỏi.
var codeVersionOnce = sync.OnceValue(func() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		bi = nil
	}
	return pickCodeVersion(buildVersion, bi, selfSHA256)
})

// buildRevision là phiên bản mã đang chạy (xem pickCodeVersion): nguồn DUY NHẤT cho vết (bản ghi quyết định, bước, lượt lỗi) và mã lần nạp
// (loadIDOf) của ứng dụng này.
func buildRevision() string { return codeVersionOnce() }
