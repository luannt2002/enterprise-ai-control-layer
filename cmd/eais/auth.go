package main

import (
	"fmt"
	"net"
	"os"
	"slices"
	"strings"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/policy"
	"go.yaml.in/yaml/v3"
)

// worldFile là phần của world.yaml mà ứng dụng cần: ngày tham chiếu (đồng hồ "world") và principal của từng tenant (token dev). Các
// trường còn lại của world.yaml (thực thể, bản ghi nguồn…) là của bộ sinh; yaml không strict nên bỏ qua chúng.
type worldFile struct {
	ReferenceDate string `yaml:"reference_date"`
	Tenants       []struct {
		ID         string `yaml:"id"`
		Principals []struct {
			ID          string `yaml:"id"`
			Role        string `yaml:"role"`
			EntityScope string `yaml:"entity_scope"`
		} `yaml:"principals"`
	} `yaml:"tenants"`
}

func loadWorld(path string) (worldFile, error) {
	var w worldFile
	b, err := os.ReadFile(path)
	if err != nil {
		return w, fmt.Errorf("world: %w", err)
	}
	if err := yaml.Unmarshal(b, &w); err != nil {
		return w, fmt.Errorf("world %s: %w", path, err)
	}
	return w, nil
}

// devTokenPrefix mở đầu mọi token dev. Token dev là THÔNG TIN CÔNG KHAI suy ra từ world.yaml (không phải bí mật): chỉ dùng được khi
// chạy với -dev, và -dev chỉ nghe loopback.
const devTokenPrefix = "dev-"

// devPrincipals dựng bảng token dev → principal từ world.yaml: token = "dev-<tenant>-<principal>". Tra theo chuỗi nguyên văn (không
// tách chuỗi) nên không mơ hồ; hai (tenant, principal) cho cùng một token là lỗi cấu hình, từ chối thay vì chọn một bên. Principal
// mang đúng tenant, vai và phạm vi thực thể khai trong world.yaml — không có gì đến từ yêu cầu.
func devPrincipals(w worldFile) (map[string]contracts.Principal, error) {
	out := map[string]contracts.Principal{}
	for _, tn := range w.Tenants {
		for _, p := range tn.Principals {
			if tn.ID == "" || p.ID == "" || p.Role == "" {
				return nil, fmt.Errorf("world: principal thiếu tenant, id hoặc vai (tenant %q, id %q)", tn.ID, p.ID)
			}
			pr := contracts.Principal{ID: p.ID, TenantID: contracts.TenantID(tn.ID), Kind: contracts.PrincipalUser, Roles: []string{p.Role}}
			if p.EntityScope != "" {
				pr.Attributes = map[string]string{"entity_scope": p.EntityScope}
			}
			tok := devTokenPrefix + tn.ID + "-" + p.ID
			if _, dup := out[tok]; dup {
				return nil, fmt.Errorf("world: hai principal cho cùng token dev %q", tok)
			}
			out[tok] = pr
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("world: không có principal nào để cấp token dev")
	}
	return out, nil
}

// bearerToken lấy token từ tiêu đề Authorization ("Bearer <token>", tên scheme không phân biệt hoa thường — RFC 7235). Thiếu, sai
// scheme hay token rỗng thì ok = false.
func bearerToken(header string) (string, bool) {
	const scheme = "bearer "
	if len(header) <= len(scheme) || !strings.EqualFold(header[:len(scheme)], scheme) {
		return "", false
	}
	tok := strings.TrimSpace(header[len(scheme):])
	if tok == "" || strings.ContainsAny(tok, " \t") {
		return "", false
	}
	return tok, true
}

// clonePrincipal sao principal để một lượt không chung slice / map với bảng token (Decision giữ tham chiếu tới chúng).
func clonePrincipal(p contracts.Principal) contracts.Principal {
	p.Roles = slices.Clone(p.Roles)
	if p.Attributes != nil {
		m := make(map[string]string, len(p.Attributes))
		for k, v := range p.Attributes {
			m[k] = v
		}
		p.Attributes = m
	}
	return p
}

// fullAccess báo principal có TOÀN QUYỀN dữ liệu của tenant theo policy của pack: có một vai phạm vi "all" mà nắm MỌI capability
// pack định nghĩa (hợp capability của mọi vai). Không nêu tên vai hay capability nào — pack khác cho kết quả khác. Hai nơi dùng nó:
//   - nạp lại dữ liệu (THAY toàn bộ dữ liệu của tenant, nên chỉ vai nắm trọn quyền đọc mới được);
//   - đọc vết của người khác trong tenant (vết mang câu hỏi, câu trả lời và bằng chứng; người đã đọc được mọi dữ liệu của tenant thì
//     đọc vết không lộ thêm gì). Người còn lại chỉ đọc được vết của chính mình.
func fullAccess(pol policy.Policy, pr contracts.Principal) bool {
	all := map[string]bool{}
	for _, rd := range pol.Roles {
		for _, c := range rd.Capabilities {
			all[c] = true
		}
	}
	for _, role := range pr.Roles {
		rd, ok := pol.Roles[role]
		if !ok || rd.Scope != "all" {
			continue
		}
		have := map[string]bool{}
		for _, c := range rd.Capabilities {
			have[c] = true
		}
		if len(have) == len(all) { // vai nắm đủ mọi capability
			return true
		}
	}
	return false
}

// checkListenAddr từ chối địa chỉ nghe không phải loopback: ở chế độ -dev, token là thông tin công khai nên cổng KHÔNG được mở ra
// ngoài máy. Chỉ nhận IP loopback tường minh hoặc "localhost"; ":8080" hay "0.0.0.0:8080" (mọi giao diện) bị từ chối.
func checkListenAddr(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("địa chỉ nghe %q: %w", addr, err)
	}
	if host == "localhost" {
		return nil
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return nil
	}
	return fmt.Errorf("địa chỉ nghe %q không phải loopback: chế độ -dev dùng token công khai nên chỉ được nghe 127.0.0.1 / ::1 / localhost", addr)
}

// loopbackHost báo giá trị tiêu đề Host có trỏ về loopback không (chống DNS rebinding: trang web ngoài trỏ tên của nó về 127.0.0.1
// rồi gọi server dev từ trình duyệt của người dùng sẽ mang Host lạ).
func loopbackHost(hostHeader string) bool {
	host := hostHeader
	if h, _, err := net.SplitHostPort(hostHeader); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
