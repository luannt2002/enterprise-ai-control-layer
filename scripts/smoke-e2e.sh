#!/usr/bin/env bash
# Smoke E2E của ứng dụng eais bằng curl, cho người chạy tay và cho scripts/chay-thu.sh / CI.
#
# Dùng:  scripts/smoke-e2e.sh http://127.0.0.1:8080
# Cần server đang chạy ở chế độ dev (eais serve -dev ... -now world) trên dữ liệu seed 42, và `curl`, `python3`. Token dev là thông tin
# công khai suy từ world.yaml: dev-<tenant>-<principal>. Đổi bằng biến môi trường nếu world khác:
#   EAIS_SMOKE_DIRECTOR   (dev-abc-infra-u-director)     principal toàn quyền của tenant mà server phục vụ
#   EAIS_SMOKE_ENGINEER   (dev-abc-infra-u-eng-a)        principal không toàn quyền cùng tenant
#   EAIS_SMOKE_OTHER      (dev-def-build-u-b-director)   principal của tenant KHÁC
# Mã thoát: 0 mọi kiểm tra đạt · 1 có kiểm tra hỏng · 2 dùng sai. Mỗi dòng in PASS / FAIL kèm tên kiểm tra.
set -u
BASE=${1:-}
[ -n "$BASE" ] || { echo "dùng: $0 http://127.0.0.1:8080" >&2; exit 2; }
DIRECTOR=${EAIS_SMOKE_DIRECTOR:-dev-abc-infra-u-director}
ENGINEER=${EAIS_SMOKE_ENGINEER:-dev-abc-infra-u-eng-a}
OTHER=${EAIS_SMOKE_OTHER:-dev-def-build-u-b-director}
QUESTION='Hôm qua máy 03 được cấp bao nhiêu dầu?'
tmp=$(mktemp -d) || exit 2
trap 'rm -rf "$tmp"' EXIT
pass=0 fail=0 CODE='' BODY=''

# req <phương thức> <đường dẫn> <token|""> [thân JSON]  →  đặt CODE và BODY
req() {
	local args=(-sS -m 30 -o "$tmp/body" -w '%{http_code}' -X "$1" "$BASE$2")
	[ -n "$3" ] && args+=(-H "Authorization: Bearer $3")
	[ $# -ge 4 ] && args+=(-H 'Content-Type: application/json' --data-binary "$4")
	CODE=$(curl "${args[@]}" 2>"$tmp/err") || { CODE=000; cat "$tmp/err" >&2; }
	BODY=$(cat "$tmp/body")
}
# jget <khoá cấp một>  →  in giá trị từ BODY (rỗng nếu không có / không phải JSON)
jget() { python3 -I -c 'import json,sys
try:
    v = json.load(sys.stdin).get(sys.argv[1], "")
except Exception:
    v = ""
print(v if not isinstance(v, (dict, list)) else json.dumps(v, ensure_ascii=False))' "$1" <<<"$BODY"; }
# check <tên> <đúng|sai>: đúng = lệnh test đã thành công
check() {
	if [ "$2" = ok ]; then pass=$((pass + 1)); echo "PASS $1"; else fail=$((fail + 1)); echo "FAIL $1  [HTTP $CODE] ${BODY:0:300}"; fi
}
is() { [ "$1" = "$2" ] && echo ok || echo no; }
has() { case $1 in *"$2"*) echo ok ;; *) echo no ;; esac; }

req GET /healthz ""
check "healthz 200" "$(is "$CODE" 200)"

body=$(python3 -I -c 'import json,sys; print(json.dumps({"text": sys.argv[1], "conversation_id": "smoke", "message_id": "m1"}, ensure_ascii=False))' "$QUESTION")
req POST /v1/ask "$DIRECTOR" "$body"
check "ask hợp lệ: 200" "$(is "$CODE" 200)"
trace=$(jget trace_id)
check "ask hợp lệ: có trace_id" "$([ -n "$trace" ] && echo ok || echo no)"
check "ask hợp lệ: câu trả lời mang số có thẩm quyền 3.850" "$(has "$(jget answer)" 3.850)"

req GET "/v1/trace/$trace" "$DIRECTOR"
check "đọc lại vết: 200" "$(is "$CODE" 200)"
stages=$(python3 -I -c 'import json,sys; print(",".join(s["stage"] for s in json.load(sys.stdin)["record"]["steps"]))' <<<"$BODY" 2>/dev/null)
check "đọc lại vết: đủ các bước S0…S10" "$(is "$stages" S0,S1,S2,S3,S4,S5,S6,S7,S8,S9,S10)"
req GET "/v1/trace/$trace?format=table" "$DIRECTOR"
check "bảng vết (format=table) có dòng chốt bước lệch" "$([ "$CODE" = 200 ] && has "$BODY" 'BƯỚC LỆCH ĐẦU TIÊN' || echo no)"

req POST /v1/ask "" "$body"
check "thiếu token: 401" "$(is "$CODE" 401)"
req POST /v1/ask "token-la-khong-ton-tai" "$body"
check "token lạ: 401" "$(is "$CODE" 401)"

req POST /v1/ask "$DIRECTOR" '{"text":"x","conversation_id":"c","message_id":"m","tenant":"def-build"}'
check "thân có trường tenant: 400" "$(is "$CODE" 400)"
check "thân có trường tenant: thông điệp nêu rõ" "$(has "$(jget error)" tenant)"

req POST /v1/ask "$DIRECTOR" '{"text":"   ","conversation_id":"c","message_id":"m"}'
check "lượt lỗi (text rỗng): 400" "$(is "$CODE" 400)"
etrace=$(jget trace_id)
req GET "/v1/trace/$etrace" "$DIRECTOR"
check "lượt lỗi cũng có vết đọc lại được (fault input)" "$([ "$CODE" = 200 ] && python3 -I -c 'import json,sys; sys.exit(0 if json.load(sys.stdin)["record"]["fault"]=="input" else 1)' <<<"$BODY" 2>/dev/null && echo ok || echo no)"

req GET "/v1/trace/$trace" "$OTHER"
check "principal tenant khác đọc vết của tenant này: 404" "$(is "$CODE" 404)"
req POST /v1/ask "$OTHER" "$body"
check "principal tenant khác hỏi runtime này: 403" "$(is "$CODE" 403)"
otrace=$(jget trace_id)
req GET "/v1/trace/$otrace" "$OTHER"
check "principal tenant khác đọc được vết của CHÍNH nó (lượt bị chặn cũng có vết): 200" "$(is "$CODE" 200)"
req GET "/v1/trace/$otrace" "$DIRECTOR"
check "principal tenant này không đọc được vết của tenant khác: 404" "$(is "$CODE" 404)"

req POST /v1/ingest "$ENGINEER" '{}'
check "vai không toàn quyền nạp lại: 403" "$(is "$CODE" 403)"
req POST /v1/ingest "$DIRECTOR" '{"path":"/etc"}'
check "nạp lại không nhận đường dẫn từ thân: 400" "$(is "$CODE" 400)"
req POST /v1/ingest "$DIRECTOR" '{}'
check "nạp lại bằng vai toàn quyền: 200" "$(is "$CODE" 200)"
check "nạp lại cùng dữ liệu: 0 bản ghi cách ly ghi mới (không nhân đôi)" "$(is "$(jget quarantine_new)" 0)"

echo "smoke E2E: $pass đạt, $fail hỏng"
[ "$fail" -eq 0 ]
