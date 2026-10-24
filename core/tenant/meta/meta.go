// Package meta là phần CHUNG của việc nạp một pack (F-029): định danh pack (pack.yaml), đọc YAML chặt và băm nội dung pack. Nó không
// phụ thuộc động cơ nào (data/, intelligence/): pack chỉ-tài-liệu hay chỉ-lệnh cũng cần đúng phần này. Phần biên dịch cho động cơ QUERY
// (lexicon, chỉ số, ý định, mẫu câu) ở package tenant.
package meta

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// Meta là phần định danh của pack (pack.yaml).
type Meta struct {
	ID       string `yaml:"id"`       // mã pack, ví dụ "construction"
	Tenant   string `yaml:"tenant"`   // slug tenant, ví dụ "abc-infra"
	Version  string `yaml:"version"`  // phiên bản do người viết pack đặt
	Language string `yaml:"language"` // ngôn ngữ trả lời, ví dụ "vi"
	Timezone string `yaml:"timezone"` // múi giờ IANA: "hôm nay" là ngày của tenant
}

// Read đọc pack.yaml của thư mục dir. Lỗi đọc trả err; nội dung sai (thiếu trường, múi giờ hỏng) trả trong problems để người gọi gom
// cùng lỗi của các phần khác.
func Read(dir string) (m Meta, loc *time.Location, problems []string, err error) {
	if err := DecodeStrict(filepath.Join(dir, "pack.yaml"), &m); err != nil {
		return m, nil, nil, err
	}
	if m.ID == "" || m.Tenant == "" || m.Version == "" {
		problems = append(problems, "pack.yaml thiếu id/tenant/version")
	}
	if m.Language == "" {
		problems = append(problems, "pack.yaml thiếu language")
	}
	loc, lerr := time.LoadLocation(m.Timezone)
	if m.Timezone == "" || lerr != nil {
		problems = append(problems, fmt.Sprintf("pack.yaml: timezone %q không hợp lệ", m.Timezone))
	}
	return m, loc, problems, nil
}

// DecodeStrict đọc một tệp YAML và từ chối khoá lạ: gõ nhầm "default_unti" phải đỏ lúc
// khởi động, không được âm thầm thành giá trị rỗng.
func DecodeStrict(path string, out any) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	defer f.Close()
	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil {
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("%s: tệp rỗng", path)
		}
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// Hash băm mọi tệp .yaml trong thư mục theo thứ tự tên, kèm tên tệp.
func Hash(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	h := sha256.New()
	for _, n := range names {
		b, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			return "", err
		}
		h.Write([]byte(n))
		h.Write([]byte{0})
		h.Write(b)
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:16], nil
}
