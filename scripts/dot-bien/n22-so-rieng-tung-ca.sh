#!/usr/bin/env bash
# Danh mục đột biến cho số được phép RIÊNG từng ca của bộ đếm claim không căn cứ (G7CounterConfig.PerCase,
# g7run perCaseNumbers). Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng "BỊ BẮT".
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./evals/"
m evals/g7.go '			extra = append(extra, c.perCase[r.ID]...)' '			extra = append(extra)' 'PC1 Count bỏ PerCase'
m evals/g7.go '			extra = append(extra, c.perCase[r.ID]...)' '			for _, v := range c.perCase {
				extra = append(extra, v...)
			}' 'PC2 số riêng của một ca dùng cho mọi ca'
export PKGS="./tenants/construction/win/g7run/"
F=tenants/construction/win/g7run/systems.go
m $F 'r.On >= from && r.On <= to' 'r.On >= from && to != ""' 'PC3 bỏ cận trên của kỳ'
m $F '			if r.Tenant != tenantID || r.ValueL == "" {' '			if r.ValueL == "" {' 'PC4 lấy cả bản ghi tenant khác'
m $F 'inPeriod := t.Period != nil && ents[r.Entity] && ' 'inPeriod := t.Period != nil && ' 'PC5 bỏ lọc thực thể'
m $F 'if ids[r.Key] || (r.WorldID != "" && ids[r.WorldID]) || inPeriod {' 'if ids[r.Key] || false || inPeriod {' 'PC6 bỏ khớp record_ids theo world_id'
m $F '				nums = append(nums, o.Value)' '				_ = o' 'PC7 bỏ other_values'
m $F '			nums = append(nums, t.Value)' '			_ = t.Value' 'PC8 bỏ truth.value'
m $F '				return nil, fmt.Errorf("%s: truth.period của %s không ở dạng YYYY-MM-DD", path, e.CaseID)' '				_ = path' 'PC9 nhận kỳ sai dạng'
m $F '		return nil, fmt.Errorf("%s: không có bản ghi nào — bộ đếm sẽ đếm oan mọi số của dữ liệu nền", path)' '		_ = path' 'PC10 nhận manifest bộ sinh rỗng'
m $F '		case seen[k]:' '		case false && seen[k]:' 'PC11 nhận case_id trùng'
m tenants/construction/win/g7run/main.go 'QuantityUnits: quantityUnits, PerCase: perCase,' 'QuantityUnits: quantityUnits, PerCase: map[string][]string{"_": perCase["_"]},' 'PC12 build không nối PerCase'
