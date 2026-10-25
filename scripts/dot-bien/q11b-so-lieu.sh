#!/usr/bin/env bash
# Danh mục đột biến "Q11b — đường SỐ LIỆU xét phạm vi bằng CÙNG vai có quyền đọc" (Q-P1-10 / Q11 = A, F-031): router S5 (khe thực thể,
# gỡ mơ hồ theo phạm vi, câu hỏi nghĩa số liệu, vế so sánh), ứng viên câu hỏi lại và che phạm vi của vết ở core/pipeline.
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0) ở test nhắm tới (MUTATE_TEST_MUON). Đọc rc từng
# dòng (danh mục này không đếm kết cục). Không dòng nào cần PG.
# Mã đổi thì chuỗi gốc có thể không còn khớp (rc=2): sửa chuỗi theo mã mới rồi chạy lại — đừng xoá dòng cho xanh.
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./intelligence/router/ ./tenants/construction/"
export MUTATE_TEST_MUON='^(TestRouteHaiVaiPhamViMotVai|TestHaiVaiSoLieuMotVai)$'
R=intelligence/router/router.go
m $R '	it, narrowed = narrow(cls, def, held, pol, st.NarrowParent, lineage)' '	it, narrowed = narrow(cls, def, pr, pol, st.NarrowParent, lineage)' 'Q11b-1 gỡ mơ hồ theo phạm vi của mọi vai'
m $R '		out := !pol.InScope(held, lineage(*s.Entity))' '		out := !pol.InScope(pr, lineage(*s.Entity))' 'Q11b-2 bậc phạm vi khe thực thể xét mọi vai'
m $R 'return pol.InScope(pol.Holding(pr, d.Capability), keys)' 'return pol.InScope(pol.Holding(pr, d.Capability[:0]), keys)' 'Q11b-3 câu hỏi nghĩa số liệu xét phạm vi mọi vai (capability rỗng: Holding giữ mọi vai)'
m $R '		held = pol.Holding(held, cdef.Capability)' '		_ = cdef' 'Q11b-4 câu so sánh bỏ capability vế sau khỏi vai xét phạm vi'
A=core/pipeline/ask.go
m $A '		visible, hidden := a.visibleCandidates(ctx, reader, asked)' '		visible, hidden := a.visibleCandidates(ctx, pr, asked)' 'Q11b-5 ứng viên câu hỏi lại theo phạm vi mọi vai'
m $A '	red := a.newScopeRedactor(ctx, reader, req.Text,' '	red := a.newScopeRedactor(ctx, pr, req.Text,' 'Q11b-6 che vết theo phạm vi mọi vai'
m $A 'ci, a.RT.Policy.InScope(reader, nil))' 'ci, a.RT.Policy.InScope(pr, nil))' 'Q11b-7 dòng S3 kể tên / loại theo phạm vi mọi vai'
