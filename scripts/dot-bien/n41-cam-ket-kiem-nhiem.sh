#!/usr/bin/env bash
# Danh mục đột biến cho công cụ H3 (ADR-0006 §10): scripts/cam-ket-giu-lai.py (cam kết băm) và scripts/kiem-nhiem-soan.py
# (kiểm toán nhiễm người soạn), bắt bằng gói test Go scripts/kiemcongcu. Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>.
# Kỳ vọng: mọi dòng "BỊ BẮT".
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./scripts/kiemcongcu/"
C=scripts/cam-ket-giu-lai.py
m $C '    if manifest.exists():' '    if False:' 'CK1 ghi đè cam kết đã có'
m $C '    if dup:' '    if False:' 'CK2 nhận id trùng'
m $C '"files_sha256": {src.name: sha(raw)},' '"files_sha256": {src.name: sha(raw + b"x")},' 'CK3 băm sai nội dung tệp ca'
m $C '            with_value += 1' '            with_value += 0' 'CK4 không đếm ca có kỳ vọng số liệu'
m $C 'doc.get("suite") != "win" or ' '' 'CK5 nhận tệp không phải bộ WIN'
m $C 'files = {str(p.relative_to(kit)): sha(p.read_bytes())' 'files = {str(p.relative_to(kit)): sha(b"")' 'CK6 băm bộ đồ nghề rỗng'
K=scripts/kiem-nhiem-soan.py
m $K '                bad.append((name, "<thư mục làm việc>"))
                continue' '                continue' 'KN1 bỏ qua Grep / Glob không có đường dẫn'
m $K '            if RECURSIVE.search(command):' '            if False:' 'KN2 lệnh đệ quy từ thư mục cha chỉ là cảnh báo'
m $K '        if name == "Read":' '        if name == "Readx":' 'KN3 không xét lệnh Read'
m $K '        if any(p == a or p.startswith(a + os.sep) for a in allowed):
            return' '        if True:
            return' 'KN4 mọi đường dẫn đều được phép'
m $K '    return 1 if total_bad else 0' '    return 0' 'KN5 có vi phạm vẫn thoát 0'
m $K '            ws = words(str(inp.get("command", "")))' '            ws = words(str(inp.get("command", "")))
            print(inp.get("command", ""))' 'KN6 in nội dung lệnh'
