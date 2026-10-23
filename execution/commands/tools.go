package commands

// Nạp tools.yaml (HOP-DONG §4.3, ADR-0011 D8): cấu hình đi cùng pack, hành vi nằm trong mã (P12). tenant.Load KHÔNG biết tệp này,
// nên luồng hỏi và công cụ đo giữ nguyên hành vi nạp; cmd/eais gọi LoadTools ngay sau tenant.Load lúc khởi động, lỗi thì không
// chạy. Gói này không import core/tenant, nên policy và danh mục chỉ số mà tenant.Load đã nạp được truyền vào để kiểm chéo.

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/luannt2002/enterprise-ai-control-layer/core/decimal"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/router"
)

// Tool là một tool trong tools.yaml.
type Tool struct {
	Name       string   `yaml:"name"`       // tên tool = tên lệnh = tên ở đường dẫn route lệnh
	Capability string   `yaml:"capability"` // capability tạo (ví dụ tickets.create); bắt buộc: quên khai không được thành "ai cũng được"
	Metrics    []string `yaml:"metrics"`    // chỉ số dẫn xuất được phép (R1)
	// MinAbsDiff là ngưỡng tuyệt đối của R4 (đơn vị của chỉ số), chuỗi thập phân; rỗng = "0", tức mọi chênh lệch khác 0.
	MinAbsDiff string `yaml:"min_abs_diff"`
	// Timeout là hạn chót của cả lệnh C0 → C7; vắng (0) = không có hạn riêng của tool, chỉ còn ngữ cảnh của bên gọi. Chưa chọn, chưa đo.
	Timeout time.Duration `yaml:"timeout"`
}

type toolsFile struct {
	Tools []Tool `yaml:"tools"`
}

// installedCommands là các lệnh mà gói này đã cài, tức có mã xử lý đứng sau. tools.yaml chỉ khai cấu hình đi cùng pack, không tạo ra lệnh:
// tên không có ở đây bị từ chối lúc nạp (đóng mặc định, CLAUDE.md §19), để một tool gõ nhầm hay thêm bừa không thành route lệnh không có mã.
var installedCommands = []string{CommandOpenDiscrepancyTicket}

// Tools là tập tool đã nạp và đã kiểm chéo, theo thứ tự khai trong tệp.
type Tools struct{ list []Tool }

// Get trả tool theo tên.
func (t *Tools) Get(name string) (Tool, bool) {
	if t == nil {
		return Tool{}, false
	}
	i := slices.IndexFunc(t.list, func(x Tool) bool { return x.Name == name })
	if i < 0 {
		return Tool{}, false
	}
	return t.list[i], true
}

// Names trả tên mọi tool theo thứ tự khai: mỗi tên là một route lệnh cần đăng ký lúc khởi động.
func (t *Tools) Names() []string {
	if t == nil {
		return nil
	}
	names := make([]string, 0, len(t.list))
	for _, x := range t.list {
		names = append(names, x.Name)
	}
	return names
}

// LoadTools nạp packDir/tools.yaml và kiểm chéo với policy và danh mục chỉ số đã nạp. Pack không có tools.yaml thì không có lệnh
// nào (tập rỗng, không lỗi: đóng mặc định, tenant có thể chỉ hỏi). Vì vậy packDir sai hay gõ nhầm tên tệp cũng ra tập rỗng và tắt lệnh mà không
// báo: nơi gọi (cmd/eais) nên ghi một dòng cảnh báo khi len(Names()) == 0. Cấu hình hỏng thì trả lỗi gom đủ thông tin để sửa và KHÔNG
// trả tập tool nào. Pack nạp lỗi nếu (§4.3):
//  1. một tool không khai capability;
//  2. một vai có capability của tool mà thiếu metrics.read (phiếu không được là đường vòng quyền đọc);
//  3. một chỉ số trong danh sách của tool không phải chỉ số DẪN XUẤT đã khai ở metrics.yaml.
//
// Ngoài ba luật đó: khoá lạ bị từ chối (gõ nhầm không được thành giá trị rỗng), tên tool phải có, không trùng và là lệnh mà gói đã cài
// (installedCommands), danh sách chỉ số không rỗng, min_abs_diff là số không âm, timeout không âm.
func LoadTools(packDir string, policy router.Policy, catalog *sourceoftruth.Catalog) (*Tools, error) {
	if catalog == nil {
		return nil, errors.New("LoadTools: thiếu danh mục chỉ số (metrics.yaml) để kiểm chéo luật 3")
	}
	var tf toolsFile
	if err := decodeStrict(filepath.Join(packDir, "tools.yaml"), &tf); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &Tools{}, nil
		}
		return nil, fmt.Errorf("pack %s: tools.yaml không hợp lệ:\n  - %w", packDir, err)
	}
	var problems []string
	seen := map[string]bool{}
	for i := range tf.Tools {
		t := &tf.Tools[i]
		label := t.Name
		switch {
		case t.Name == "":
			label = fmt.Sprintf("#%d", i+1)
			problems = append(problems, fmt.Sprintf("tool %s: thiếu name", label))
		case seen[t.Name]:
			problems = append(problems, fmt.Sprintf("tool %s: tên trùng", label))
		case !slices.Contains(installedCommands, t.Name):
			problems = append(problems, fmt.Sprintf("tool %s: chưa cài trong gói (đã cài: %s)", label, strings.Join(installedCommands, ", ")))
		}
		seen[t.Name] = true
		if t.MinAbsDiff == "" {
			t.MinAbsDiff = "0"
		}
		problems = append(problems, t.problems(label, policy, catalog)...)
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("pack %s: tools.yaml không hợp lệ:\n  - %s", packDir, strings.Join(problems, "\n  - "))
	}
	return &Tools{list: tf.Tools}, nil
}

// problems trả mọi chỗ sai của một tool: ba luật nạp và các luật phụ.
func (t Tool) problems(label string, policy router.Policy, catalog *sourceoftruth.Catalog) []string {
	var out []string
	bad := func(format string, a ...any) {
		out = append(out, fmt.Sprintf("tool %s: ", label)+fmt.Sprintf(format, a...))
	}
	if t.Capability == "" { // luật 1
		bad("không khai capability (quên khai không được biến thành ai cũng được)")
	} else { // luật 2, theo thứ tự tên vai để lỗi tất định
		for _, role := range slices.Sorted(maps.Keys(policy.Roles)) {
			caps := policy.Roles[role].Capabilities
			if slices.Contains(caps, t.Capability) && !slices.Contains(caps, CapMetricsRead) {
				bad("vai %s có %s mà thiếu %s (phiếu không được là đường vòng quyền đọc)", role, t.Capability, CapMetricsRead)
			}
		}
	}
	if len(t.Metrics) == 0 {
		bad("danh sách metrics rỗng")
	}
	for _, m := range t.Metrics { // luật 3
		if _, derived := catalog.Derived(m); !derived {
			bad("chỉ số %q không phải chỉ số dẫn xuất đã khai ở metrics.yaml", m)
		}
	}
	if msg := minAbsDiffProblem(t.MinAbsDiff); msg != "" {
		bad("min_abs_diff %s", msg)
	}
	if t.Timeout < 0 {
		bad("timeout %s không được âm", t.Timeout)
	}
	return out
}

// minAbsDiffProblem trả lý do một giá trị min_abs_diff không dùng được ("" khi dùng được): phải là số thập phân và không âm, vì ngưỡng âm vô
// hiệu hoá R4 (chênh lệch bằng 0 vẫn tạo phiếu). Dùng chung cho LoadTools (lúc nạp pack) và New (lúc dựng pipeline bằng cấu hình tay).
func minAbsDiffProblem(v string) string {
	n, ok := decimal.Parse(v)
	switch {
	case !ok:
		return fmt.Sprintf("%q không phải số", v)
	case n.Sign() < 0:
		return fmt.Sprintf("%q không được âm", v)
	}
	return ""
}

// decodeStrict đọc một tệp YAML và từ chối khoá lạ: gõ nhầm "capabilty" phải đỏ lúc khởi động, không được âm thầm thành giá trị
// rỗng. Cùng cách làm với tenant.Load, viết lại ở đây vì gói này không import core/tenant. Lỗi gói %w để nhận ra tệp vắng.
func decodeStrict(path string, out any) error {
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
