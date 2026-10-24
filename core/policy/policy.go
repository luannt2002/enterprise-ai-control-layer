// Package policy là ranh giới quyền của lõi: vai nghiệp vụ → capability + phạm vi dữ liệu (policy.yaml của pack). Quyền ở đây là
// quyền của PHẦN MỀM, không phải "vai" trong prompt.
//
// Gói chỉ biết core/contracts: định tuyến hỏi (intelligence/router), lệnh ghi (execution/commands) và xác thực (cmd/eais) dùng CHUNG
// một cài đặt, nên sửa chặng định tuyến không thể lặng lẽ đổi quyền của lệnh ghi (F-031).
package policy

import (
	"fmt"
	"slices"
	"strings"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// RoleDef là một vai nghiệp vụ trong pack (policy.yaml).
type RoleDef struct {
	Capabilities []string `yaml:"capabilities"`
	// Scope: "all" = xem mọi thực thể; "attribute" = chỉ thực thể có khoá nằm trong thuộc
	// tính entity_scope của principal. Không có giá trị mặc định nào mở toang.
	Scope string `yaml:"scope"`
}

// Policy là chính sách vai → capability + phạm vi dữ liệu.
type Policy struct {
	Roles map[string]RoleDef `yaml:"roles"`
}

// Validate kiểm chính sách.
func (p Policy) Validate() error {
	var errs []string
	if len(p.Roles) == 0 {
		errs = append(errs, "policy không có vai nào")
	}
	for name, r := range p.Roles {
		if r.Scope != "all" && r.Scope != "attribute" {
			errs = append(errs, fmt.Sprintf("vai %s: scope phải là all hoặc attribute, đang là %q", name, r.Scope))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("policy không hợp lệ:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// Allows báo principal có capability không (qua bất kỳ vai nào của họ). Đóng mặc định:
// capability rỗng không cấp quyền cho ai.
func (p Policy) Allows(pr contracts.Principal, capability string) bool {
	if capability == "" {
		return false
	}
	for _, role := range pr.Roles {
		for _, c := range p.Roles[role].Capabilities {
			if c == capability {
				return true
			}
		}
	}
	return false
}

// Holding trả principal chỉ còn các vai có capability: quyền và phạm vi / ACL phải đến từ CÙNG một vai (Q11), nên vai thiếu capability
// không cho vai khác mượn ACL hay phạm vi của nó. capability rỗng (ý định không đòi capability — router cũng không kiểm) thì giữ nguyên.
func (p Policy) Holding(pr contracts.Principal, capability string) contracts.Principal {
	if capability == "" {
		return pr
	}
	out := pr
	out.Roles = nil
	for _, role := range pr.Roles {
		if slices.Contains(p.Roles[role].Capabilities, capability) {
			out.Roles = append(out.Roles, role)
		}
	}
	return out
}

// InScope báo principal có được xem một thực thể không. lineage là khoá của thực thể và mọi
// tổ tiên của nó: được giao dự án PRJ-A thì xem được máy thuộc PRJ-A.
func (p Policy) InScope(pr contracts.Principal, lineage []string) bool {
	for _, role := range pr.Roles {
		if p.Roles[role].Scope == "all" {
			return true
		}
	}
	for _, k := range strings.Split(pr.Attributes["entity_scope"], ",") {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		for _, key := range lineage {
			if key == k {
				return true
			}
		}
	}
	return false
}
