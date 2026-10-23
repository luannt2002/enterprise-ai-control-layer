#!/usr/bin/env bash
# Danh mục đột biến M1–M26 của cổng X8 cho khối M4 (HOP-DONG §11.2): lệnh open_discrepancy_ticket, gói execution/commands, migration 0004.
# Mỗi dòng: m <mã> <nhãn> <tệp> <chuỗi gốc> <chuỗi đột biến> <mẫu "phải thấy">... — một đột biến, và kỳ vọng nó làm ca ở cột "Ca phải đỏ" của §11.2 đỏ
# ĐÚNG dòng "Phải thấy" của §10 (§11.1, quy tắc đột biến). Mỗi mẫu là biểu thức chính quy ERE khớp TÊN test con đỏ. Bộ chạy ca (bochay_test.go) đặt mỗi
# khẳng định của một ca thành một test con tên theo khoá được khẳng định, nên mẫu `A04_[^ ]*/write_unit_calls` nghĩa là "ca lớp A04 đỏ ở dòng write_unit_calls".
# Đột biến chỉ "ĐÚNG CHỖ" khi MỖI mẫu khớp ít nhất một test đỏ; đỏ mà thiếu một mẫu, hay đỏ sai chỗ, là "khác"; không test nào đỏ là "SỐNG SÓT".
#
# Chạy (PG RIÊNG của bạn, hai biến đặt rõ, không có giá trị mặc định; đừng trỏ vào database đang phục vụ việc khác — test tạo database tạm
# trong cùng cụm rồi xoá):
#   EAIS_PG_ADMIN_DSN=postgres://<quản trị>@<máy>:<cổng>/<database riêng>?sslmode=disable \
#   EAIS_PG_DSN=postgres://<vai ứng dụng>@<máy>:<cổng>/<database riêng>?sslmode=disable \
#   taskset -c <nhân> bash scripts/dot-bien/m4-phieu-chenh-lech.sh
# Biến tuỳ chọn: PKGS (mặc định ./execution/commands/), MUTATE_SH (mặc định scripts/mutate.sh), DOT_BIEN_CHI="M1 M21" (chỉ chạy các mã đó).
# Mã thoát 0 chỉ khi không đột biến nào sống sót hay "khác" (như dot-bien-vet.sh); dòng "chờ K3" và "tương đương" có lý do và không tính.
#
# Đột biến nào cần cửa HTTP hay đường lưu vết của K3 (M9 ở cửa HTTP, M24 phần vết đã lưu, M25): dòng được viết sẵn ở cuối tệp, tắt bằng dấu #, ghi rõ CHỜ K3
# (nhánh m4/k3-route, chưa gộp), đã thử trên cây thử gộp. Dòng M5 và M13 có ghi chú giới hạn kèm số đo (xem chỗ đó).
set -u
cd "$(dirname "$0")/../.." || exit 1
: "${EAIS_PG_ADMIN_DSN:?đặt EAIS_PG_ADMIN_DSN (PG riêng của bạn)}"
: "${EAIS_PG_DSN:?đặt EAIS_PG_DSN (PG riêng của bạn)}"
export EAIS_PG_ADMIN_DSN EAIS_PG_DSN
export PKGS=${PKGS:-./execution/commands/}
MUTATE=${MUTATE_SH:-scripts/mutate.sh}
CHI=${DOT_BIEN_CHI:-}

# Nền xanh: đột biến chỉ đo được khi CÙNG lệnh test xanh trên cây chưa đột biến (nền đỏ thì mọi đột biến đều "bị bắt" giả). mutate.sh bản mới tự
# chạy nền; kiểm lại ở đây để danh mục cũng đúng với mutate.sh bản cũ.
if ! go test $PKGS -count=1 >/dev/null 2>&1; then
	echo "NỀN ĐỎ: go test $PKGS đã đỏ trên cây chưa đột biến — sửa nền rồi mới đo" >&2
	exit 5
fi

bat=0 song=0 khac=0 cho=0
ANY='TestCaPhatTrien(BoNho|PG)'
PG='TestCaPhatTrienPG'

m() {
	local ma=$1 nhan=$2 tep=$3 goc=$4 dot=$5
	shift 5
	if [ -n "$CHI" ] && ! grep -qw -- "$ma" <<<"$CHI"; then return 0; fi
	local muon out rc thieu="" p ten
	muon=$(IFS='|'; printf '%s' "$*")
	out=$(MUTATE_TEST_MUON="$muon" "$MUTATE" "$tep" "$goc" "$dot" "$ma $nhan"); rc=$?
	printf '%s\nrc=%s\n' "$out" "$rc"
	case $rc in
	0)
		ten=$(sed -n 's/^.*BỊ BẮT: //p' <<<"$out" | tr ' ' '\n')
		for p in "$@"; do
			grep -E -q -- "$p" <<<"$ten" || thieu="$thieu [$p]"
		done
		if [ -z "$thieu" ]; then
			echo "[$ma] ĐÚNG CHỖ"
			bat=$((bat + 1))
		else
			echo "[$ma] BỊ BẮT NHƯNG THIẾU MẪU:$thieu"
			khac=$((khac + 1))
		fi
		;;
	3) song=$((song + 1)) ;;
	*) khac=$((khac + 1)) ;;
	esac
}

# mpy: như m nhưng chuỗi gốc và chuỗi đột biến do một đoạn Python sinh từ chính tệp (cho đột biến dời nhiều đoạn mã liền nhau). Đoạn Python đọc tệp ở
# argv[1], in chuỗi gốc rồi chuỗi đột biến, mỗi chuỗi kết thúc bằng NUL.
mpy() {
	local ma=$1 nhan=$2 tep=$3 py=$4
	shift 4
	if [ -n "$CHI" ] && ! grep -qw -- "$ma" <<<"$CHI"; then return 0; fi
	local -a pair
	mapfile -d '' -t pair < <(python3 -I -c "$py" "$tep")
	if [ "${#pair[@]}" -ne 2 ]; then
		echo "[$ma] KHÔNG SINH ĐƯỢC ĐỘT BIẾN từ $tep (mã đã đổi, sửa đoạn Python)"
		khac=$((khac + 1))
		return 0
	fi
	m "$ma" "$nhan" "$tep" "${pair[0]}" "${pair[1]}" "$@"
}

# mm: như m nhưng chạy trên một database VỪA DỰNG LẠI, cho đột biến migration mà test dùng database dùng chung (migration chỉ áp MỘT lần mỗi database: trên database đã
# migrate bằng bản gốc thì bản đột biến không được áp và đột biến sống giả). Các test tạo database tạm của chính chúng (bộ chạy ca, TestPGTicketMigration0004ChayLai)
# không cần mm. Tên database = tên trong EAIS_PG_ADMIN_DSN + _dotbien, từ chối nếu trùng tên database trong một trong hai DSN. Với mutate.sh bản mới (có --nen):
# chạy nền MỘT lần trên database đó rồi dựng lại, để mutate.sh không áp migration GỐC trước đột biến.
db_of() { printf '%s' "$1" | sed -E 's#^[a-z]+://[^/]*/([^/?]*).*#\1#'; }
dsn_db() { printf '%s' "$1" | sed -E 's#^([a-z]+://[^/]*)/[^/?]*#\1/'"$2"'#'; }
DB_MM="$(db_of "$EAIS_PG_ADMIN_DSN")_dotbien"
for d in "$EAIS_PG_ADMIN_DSN" "$EAIS_PG_DSN"; do
	if [ "$(db_of "$d")" = "$DB_MM" ]; then
		echo "tên database đột biến $DB_MM trùng database trong DSN: script sẽ DROP chính nó" >&2
		exit 1
	fi
done
nen_mm=""
fresh_mm() { psql -q -X "$EAIS_PG_ADMIN_DSN" -c "DROP DATABASE IF EXISTS $DB_MM WITH (FORCE)" -c "CREATE DATABASE $DB_MM"; }
mm() {
	local ma=$1
	if [ -n "$CHI" ] && ! grep -qw -- "$ma" <<<"$CHI"; then return 0; fi
	fresh_mm || { echo "[$ma] không dựng được database $DB_MM"; khac=$((khac + 1)); return 0; }
	local a d
	a=$(dsn_db "$EAIS_PG_ADMIN_DSN" "$DB_MM") d=$(dsn_db "$EAIS_PG_DSN" "$DB_MM")
	if [ -z "$nen_mm" ] && grep -q -- '--nen' "$MUTATE"; then
		nen_mm=$(EAIS_PG_ADMIN_DSN=$a EAIS_PG_DSN=$d "$MUTATE" --nen)
		echo "$nen_mm"
		fresh_mm || { echo "[$ma] không dựng được database $DB_MM"; khac=$((khac + 1)); return 0; }
	fi
	EAIS_PG_ADMIN_DSN=$a EAIS_PG_DSN=$d MUTATE_NEN_CO_SAN=1 m "$@"
}

P=execution/commands/pipeline.go
S=execution/commands/pgticket.go
M=db/migrations/0004_phieu_chenh_lech.sql

# ---- Tầng lệnh: sổ lệnh, fingerprint, chỉ mục nghiệp vụ (X1) ----
m M1 'bỏ bước tra sổ lệnh (C4b): lần tra không bao giờ thấy dòng nào' $P \
	'led, found, err := r.p.d.Store.FindLedger(ctx, r.caller.ID, r.req.CommandID)' \
	'led, found, err := LedgerRecord{}, false, error(nil)' \
	"A01_[^ ]*/results" "A21_[^ ]*/results"
m M2 'bỏ khoá chính (tenant, người gọi, command_id) của sổ lệnh' $M \
	'  PRIMARY KEY (tenant_id, caller_id, command_id),' '  CHECK (true),' \
	"$PG/A02b_[^ ]*/(results|tickets_open|ledger_rows)"
m M3 'bỏ so fingerprint ở C4b' $P \
	'case led.Fingerprint != r.fp:' 'case false:' \
	"A03_[^ ]*/results"
m M4 'bỏ chỉ mục duy nhất nghiệp vụ (chỉ còn chỉ mục thường)' $M \
	'CREATE UNIQUE INDEX IF NOT EXISTS discrepancy_tickets_open_key' 'CREATE INDEX IF NOT EXISTS discrepancy_tickets_open_key' \
	"$PG/A04b_[^ ]*/(results|tickets_open)"
# M5 theo đúng chữ của §11.2: đặt tra sổ lệnh SAU luật "không trùng". Ở mã hiện tại (K1v2, e1ce950) R5 tra LẠI sổ lệnh khi thấy phiếu OPEN (cửa sổ C4b–R5, §6.3 hàng 3), nên
# đổi thứ tự C4b và C5 cho cùng kết quả, cùng bộ đếm, cùng vết ở MỌI ca khai được bằng khuôn §12.1: A01 và A21 (cột "Ca phải đỏ" của §11.2) KHÔNG đỏ dưới M5 [ĐO 23/10: 0 test
# của bộ ca phát triển đỏ]. Đột biến vẫn bị bắt, nhưng bởi test của K1 về thứ tự nhãn lỗi hạ tầng (TestStepFaultMangNhanThatCuaThanhPhanHong: lỗi đọc số liệu hay danh mục
# phải được gắn vào đúng chặng), nên dòng này khẳng định test đó, không khẳng định A01 / A21. §11.2 cần sửa cột M5 (đề xuất: chỉ A03 qua M5b). M5b nhắm phần giữ lại của ý đồ M5.
m M5 'đặt tra sổ lệnh (C4b) SAU luật R5 trong chuỗi chặng (A01, A21 KHÔNG đỏ: lần tra lại của R5 che)' $P \
	'r.enter(stLedger).ledger(ctx) && r.enter(stRules).rules(ctx)' 'r.enter(stRules).rules(ctx) && r.enter(stLedger).ledger(ctx)' \
	"TestStepFaultMangNhanThatCuaThanhPhanHong"
m M5b 'C4b không tra gì (chỉ còn lần tra lại của R5 sau khi thấy phiếu mở)' $P \
	'func (r *run) ledger(ctx context.Context) bool { return r.consultLedger(ctx, false) }' \
	'func (r *run) ledger(ctx context.Context) bool { return true }' \
	"A03_[^ ]*/write_unit_calls"

# ---- Quyền (X2) ----
m M6 'bỏ kiểm tickets.create' $P \
	'if !d.Policy.Allows(r.pr, d.Tool.Capability) {' 'if false && !d.Policy.Allows(r.pr, d.Tool.Capability) {' \
	"A07_[^ ]*/results"
m M7 'bỏ kiểm phạm vi thực thể' $P \
	'if !d.Policy.InScope(r.pr, lineage) {' 'if false && !d.Policy.InScope(r.pr, lineage) {' \
	"A08_[^ ]*/results"
m M8 'bỏ kiểm metrics.read lúc chạy (ca phát triển A09)' $P \
	'if !d.Policy.Allows(r.pr, CapMetricsRead) {' 'if false && !d.Policy.Allows(r.pr, CapMetricsRead) {' \
	"A09_[^ ]*/results"
m M8b 'bỏ luật nạp pack số 2 (§4.3): vai có tickets.create mà thiếu metrics.read (A12)' execution/commands/tools.go \
	'if slices.Contains(caps, t.Capability) && !slices.Contains(caps, CapMetricsRead) {' \
	'if false && slices.Contains(caps, t.Capability) && !slices.Contains(caps, CapMetricsRead) {' \
	"TestLoadToolsLuatNap"
m M9a 'C1 không so tenant của Principal với tenant của runtime' $P \
	'if r.pr.ID == "" || string(r.pr.TenantID) != r.p.d.Tenant {' 'if r.pr.ID == "" {' \
	"A10_[^ ]*/results"
m M9b 'bỏ đặt GUC tenant ở BeginTenantTx (RLS không còn tenant)' platform/pgstore/tenanttx.go \
	'"SELECT set_config('"'"'eais.tenant_id'"'"', $1, true)", string(tenant)); err != nil {' \
	'"SELECT $1::text", string(tenant)); err != nil {' \
	"$PG/A06_[^ ]*/(results|tickets_open|ledger_rows|other_tenants)"
m M10 'trả mã khác nhau cho "không tồn tại" và "ngoài phạm vi"' $P \
	$'\t\treturn deny("khoá thực thể không có trong danh mục")' \
	$'\t\tr.refuse(stPermission, resultFor(ReasonInsufficientEvidence), Step{Input: in, Output: []string{"denied"}, Decision: "từ chối", Reason: "không có quyền thực hiện yêu cầu này"}, "khoá thực thể không có trong danh mục")\n\t\treturn false' \
	"A08_[^ ]*/(results|distinct_details)"

# ---- Ghi, đọc lại, audit (X3, X4) ----
mpy M11 'ghi audit EXECUTED NGOÀI giao dịch, sau commit' $S '
import sys
s = open(sys.argv[1], encoding="utf-8").read()
a = s.index("\tsent = true //")
end = "\treturn nil\n}\n"
b = s.index(end, s.index("func (s *PGStore) insertUnit")) + len(end)
frm = s[a:b]
audit = "\tif err := insertAudit(ctx, tx, u.Audit); err != nil {\n\t\treturn fmt.Errorf(\"INSERT audit: %w\", err)\n\t}\n"
assert audit in frm
to = frm.replace(audit, "")
post = ("\tif err := classifyCommit(tx.Commit(ctx)); err != nil {\n\t\treturn err\n\t}\n"
        "\tatx, err := s.db.BeginTenantTx(ctx)\n\tif err != nil {\n\t\treturn fmt.Errorf(\"%w: %w\", ErrCommitUnknown, err)\n\t}\n"
        "\tdefer func() { _ = atx.Rollback(ctx) }()\n"
        "\tif err := insertAudit(ctx, atx, u.Audit); err != nil {\n\t\treturn fmt.Errorf(\"%w: %w\", ErrCommitUnknown, err)\n\t}\n\treturn atx.Commit(ctx)\n}\n")
assert "\treturn classifyCommit(tx.Commit(ctx))\n}\n" in to
to = to.replace("\treturn classifyCommit(tx.Commit(ctx))\n}\n", post)
sys.stdout.write(frm + "\0" + to + "\0")
' "$PG/A20_[^ ]*/(results|tickets_open|ledger_rows|audit)"
m M12 'bỏ đọc lại sau ghi: DONE ngay từ bản chụp ý định' $P \
	$'\tcase r.committed:\n\t\tr.verify(ctx, r.unit.Ticket.ID, false)' \
	$'\tcase r.committed:\n\t\tr.res = doneResult(r.unit.Ticket, false)\n\t\tr.setStep(stVerify, Step{Status: StepOK, Decision: string(StatusDone), Reason: "đột biến: không đọc lại"})' \
	"A23_[^ ]*/results"
# M13 theo đúng chữ của §11.2 ("gộp lỗi COMMIT vào FAILED như inTenant"). Phân loại lỗi COMMIT nằm ở hai chỗ: classifyCommit (pgticket.go) và nhánh ErrCommitUnknown của
# write (pipeline.go). Các điểm chèn lỗi F3, F5 trả ErrCommitUnknown THẲNG từ lớp bọc nên không đi qua classifyCommit; lỗi COMMIT thật của trình điều khiển do
# TestU1RanhGioiCommit (pgticket_u1_test.go) canh. Dòng này nhắm nhánh của pipeline (A21, A22); M13b nhắm classifyCommit bằng chính test U1.
m M13 'COMMIT_UNKNOWN thành FAILED STORE_UNAVAILABLE ở pipeline (gộp lỗi COMMIT vào FAILED)' $P \
	'r.fail(stWrite, resultFor(ReasonCommitUnknown), err, Step{' 'r.fail(stWrite, resultFor(ReasonStoreUnavailable), err, Step{' \
	"A21_[^ ]*/results" "A22_[^ ]*/results"
m M13b 'classifyCommit gộp mọi lỗi COMMIT còn lại vào ErrBeforeCommit (chắc chắn chưa ghi)' $S \
	'	return fmt.Errorf("%w: %w", ErrCommitUnknown, err)
}

// notSentYet' \
	'	return fmt.Errorf("%w: %w", ErrBeforeCommit, err)
}

// notSentYet' \
	"TestU1RanhGioiCommit"

# ---- Dữ liệu và luật (X5) ----
m M14 'bỏ cờ approx khi lập phiếu' $P \
	'Approx: res.Approx,' 'Approx: false,' \
	"A17_[^ ]*/ticket"
m M19 'chọn một số khi hai nguồn ngang thẩm quyền lệch (bỏ nhánh NeedsReview)' $P \
	'case res.NeedsReview:' 'case false && res.NeedsReview:' \
	"A16_[^ ]*/results"
m M20 'bỏ phân loại lại sau vi phạm khoá duy nhất (§6.4): trả lỗi mã' $P \
	'r.reclassify(ctx, err, in)' \
	'r.fail(stWrite, internalResult(StatusFailed), err, Step{Input: in, Decision: "lỗi mã", Reason: "đột biến"}, "đột biến")' \
	"A02_[^ ]*/results" "A02b_[^ ]*/results" "A04b_[^ ]*/results"
m M21 'bỏ kiểm R5 ở C5 (còn chỉ mục duy nhất và §6.4): lần thứ hai vẫn gọi đơn vị ghi phiếu' $P \
	$'ref, found, err := d.Store.FindOpenTicket(ctx, r.ent.Key, r.req.Metric, r.req.Day)\n\tif err != nil {\n\t\tr.infra(stRules, err, in, "không tìm được phiếu đang mở")' \
	$'ref, found, err := TicketRef{}, false, error(nil)\n\tif err != nil {\n\t\tr.infra(stRules, err, in, "không tìm được phiếu đang mở")' \
	"A04_[^ ]*/write_unit_calls"

# ---- Biên (X6) ----
mm M15 'cấp UPDATE, DELETE cho eais_app trên ba bảng (A26: quyền SQL, ngoài khuôn §12.1)' $M \
	"EXECUTE format('GRANT SELECT, INSERT ON %I TO eais_app', t);" "EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON %I TO eais_app', t);" \
	"TestPGTicketQuyenChiThemVaDoc"
m M16 'thêm khoá ngoại từ phiếu tới entities (Load xoá và nạp lại danh mục)' $M \
	"  CHECK (status = 'OPEN')," $'  CHECK (status = \'OPEN\'),\n  FOREIGN KEY (tenant_id, entity_id) REFERENCES entities (tenant_id, id),' \
	"$PG/A18_[^ ]*/chay"
m M17 'nối gói lệnh tới gateway model (A24: cổng import)' $P \
	$'\t"github.com/google/uuid"\n' $'\t"github.com/google/uuid"\n\t_ "github.com/luannt2002/enterprise-ai-control-layer/intelligence/models"\n' \
	"TestA24VaRangBuocImport"
m M18 'thêm câu "đã tạo phiếu" vào mẫu câu của luồng hỏi (A25)' tenants/construction/pack/templates.yaml \
	'derived_total: "{metric_label} của {entity_name} {period}: {value} {unit}."' \
	'derived_total: "{metric_label} của {entity_name} {period}: {value} {unit}. Đã tạo phiếu kiểm tra."' \
	"TestA25CauLenhVaoLuongHoiKhongNoiDaTaoPhieu"

# ---- Lưu trữ (X1, X6) ----
mm M22 'bỏ khoá ngoại từ sổ lệnh tới phiếu (A26)' $M \
	'  FOREIGN KEY (tenant_id, ticket_id) REFERENCES discrepancy_tickets (tenant_id, id),' '  CHECK (true),' \
	"TestPGTicketSoLenhMoCoiVaKhoaNgoai"
mpy M23 'đảo thứ tự ghi: sổ lệnh trước phiếu' $S '
import sys
s = open(sys.argv[1], encoding="utf-8").read()
a = s.index("\tif _, err := tx.Exec(ctx, sqlInsertTicket,")
m = s.index("\tl := u.Ledger\n")
end = "\t\treturn fmt.Errorf(\"INSERT sổ lệnh: %w\", err)\n\t}\n"
b = s.index(end, m) + len(end)
sys.stdout.write(s[a:b] + "\0" + s[m:b] + s[a:m] + "\0")
' "$PG/A01_[^ ]*/results"

# ---- Vết ở mức dữ liệu thô (X9, phần chạy được trước K3) ----
m M24 'bước C4 của lượt PERMISSION_DENIED ghi lý do thật (tách "không tồn tại" với "ngoài phạm vi")' $P \
	'Decision: "từ chối", Reason: "không có quyền thực hiện yêu cầu này",' 'Decision: "từ chối", Reason: "không có quyền: " + real,' \
	"TestA29CVetPermissionDeniedKhongTachKhongTonTaiVaNgoaiPhamVi"

# ---- Người gọi ----
m M26 'tra sổ lệnh chỉ theo (tenant, command_id), bỏ người gọi' $S \
	'WHERE caller_id = $1 AND command_id = $2' 'WHERE command_id = $2 AND length($1::text) >= 0' \
	"$PG/A05_[^ ]*/results"

# ---------------------------------------------------------------------------------------------------------------------------------------------------
# CHỜ K3 (nhánh m4/k3-route, chưa gộp vào main): cần cửa HTTP và đường lưu vết của cmd/eais. Ba dòng dưới được soạn theo m4/k3-route@1a56f9b và ĐÃ THỬ ngày 23/10 trên một cây
# THỬ GỘP (K4 + K3, không commit, PKGS=./cmd/eais/, mutate.sh bản mới): cả ba "BỊ BẮT" đúng test của K3 (mẫu ghi ở cuối mỗi dòng). Chúng vẫn tắt bằng dấu # vì main chưa có route lệnh:
# bỏ dấu # khi K3 đã gộp, đặt PKGS=./cmd/eais/ khi chạy, và đối chiếu lại chuỗi gốc (mutate.sh báo "xuất hiện 0 lần" nếu mã đã đổi).
#
# M9 ở cửa HTTP (A11, A28): thân yêu cầu mang trường tenant / principal / vai được nhận thay vì bị từ chối (bỏ giải mã chặt của decodeCommand).
# m M9h 'cửa HTTP không còn từ chối trường lạ (tenant, principal, role) trong thân' cmd/eais/server.go \
#	$'\tdec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxCommandBody))\n\tdec.DisallowUnknownFields()' $'\tdec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxCommandBody))' \
#	'TestLenhHTTPThanSai' 'TestLenhHTTPTenTruongLaDuocCat'
# M24 (phần vết đã lưu, A29 (c)): chi tiết đã nằm ở m M24 phía trên (dữ liệu thô); phần qua eais trace và GET /v1/trace/{id} do K3 kiểm trong TestLenhHTTPVet.
# M25: không lưu vết cho lượt không phải DONE, hoặc để lỗi lưu vết đổi kết quả đã ghi (A29 (b), (e)):
# m M25a 'lỗi lưu vết đổi kết quả đã ghi (đổi mã HTTP)' cmd/eais/server.go \
#	$'\t\t} else {\n\t\t\tres.TraceID = ""\n\t\t}' $'\t\t} else {\n\t\t\tres.TraceID = ""\n\t\t\tstatus = http.StatusServiceUnavailable\n\t\t}' \
#	'TestLenhHTTPMatVetKhongDoiKetQua'
# m M25b 'chỉ lưu vết lượt DONE' cmd/eais/server.go \
#	'if s.persistTrace(r.Context(), pr, rec) == nil {' 'if out.Result.Status != commands.StatusDone || s.persistTrace(r.Context(), pr, rec) == nil {' \
#	'TestLenhHTTPVet|TestLenhHTTPKetCuc'
cho=$((cho + 3)) # M9 (cửa HTTP), M25a, M25b

echo "TỔNG: đúng chỗ $bat · sống sót $song · khác $khac · chờ K3 $cho"
[ "$song" -eq 0 ] && [ "$khac" -eq 0 ]
