#!/usr/bin/env bash
# Danh mục đột biến "L5b — phần còn lại của L5" (báo cáo audit vNext 24/10/2026: F-030 danh từ nhãn / đếm của pack, F-005 lời giải thích
# gọi kiểu cha của pack, F-008 ghi chú suy luận của fact).
# Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>. Kỳ vọng: mọi dòng in "BỊ BẮT" (rc=0). Đọc rc từng dòng (danh mục này không
# đếm kết cục). Dòng cần PG (nhãn có [PG]) chỉ đo được khi đặt EAIS_PG_ADMIN_DSN / EAIS_PG_DSN; thiếu thì test PG bị bỏ qua và đột
# biến "sống sót" giả.
# Mã đổi thì chuỗi gốc có thể không còn khớp (rc=2): sửa chuỗi theo mã mới rồi chạy lại — đừng xoá dòng cho xanh.
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
Q=data/normalization/qualifier.go
L=data/normalization/lexicon.go
IN=intelligence/intent/intent.go
P=core/tenant/pack.go
A=core/pipeline/ask.go
X=tenants/construction/pack/lexicon.yaml
export PKGS="./data/normalization/"
m $Q ' || slices.ContainsFunc(e.labels, is))' ')' 'L5b-1 danh từ nhãn của pack bị bỏ qua (F-030)'
m $Q ' || slices.ContainsFunc(e.counts, counted)' '' 'L5b-2 danh từ đếm của pack bị bỏ qua (F-030)'
m $Q '!(slices.ContainsFunc(e.unitLabels, is) && digitBefore(raw, ws))' '!(false && digitBefore(raw, ws))' 'L5b-3 nhãn là đơn vị đứng sau chữ số vẫn là nhãn ("3850 L 19") (F-030)'
m $L 'strings.EqualFold(u.alias, NFC(w))' 'true' 'L5b-4 mọi danh từ nhãn của pack bị coi là đơn vị (F-030)'
m $L 'fail("label_nouns có danh từ rỗng")' '_ = w' 'L5b-5 label_nouns rỗng được nhận (F-030)'
m $L 'fail("count_nouns có danh từ rỗng")' '_ = w' 'L5b-6 count_nouns rỗng được nhận (F-030)'
export PKGS="./data/normalization/ ./intelligence/intent/ ./core/pipeline/ ./core/tenant/ ./tenants/construction/"
m $IN '!labels(in.Raw, sp.Start, sp.End)' '!new(normalization.Engine).LabelContext(in.Raw, sp.Start, sp.End)' 'L5b-7 hasFigure bỏ qua Options.Labels (F-030)'
m $IN '		labels = new(normalization.Engine).LabelContext // pack' '		labels = func(string, int, int) bool { return false } // pack' 'L5b-8 Labels không khai thì mất cả từ chức năng (F-030)'
m $P '				Labels: rt.Normalizer.LabelContext, // danh từ nhãn / đếm của lexicon pack (F-030)' '' 'L5b-9 pack.go không nối Labels (F-030)'
m $A 'a.RT.Normalizer.LabelContext(in.Raw, sp.Start, sp.End)' 'new(normalization.Engine).LabelContext(in.Raw, sp.Start, sp.End)' 'L5b-10 câu xác nhận bỏ danh từ nhãn / đếm của pack (F-030)'
m $A 'a.RT.Normalizer.DashDateAt(in.Raw, in.Spans, i)' 'new(normalization.Engine).DashDateAt(in.Raw, in.Spans, i)' 'L5b-11 ngày dd-mm bỏ danh từ đếm của pack ("10-12 xe" thành ngày) (F-030)'
m $X 'count_nouns: [máy, xe, chuyến, thiết bị, dự án]' 'count_nouns: []' 'L5b-12 pack cầu đường không khai danh từ đếm (F-030)'
m $X 'label_nouns: [khối, lít, l]' 'label_nouns: []' 'L5b-13 pack cầu đường không khai danh từ nhãn (F-030)'
m $X 'label_nouns: [khối, lít, l]' 'label_nouns: [khối, lít]' 'L5b-14 pack cầu đường bỏ nhãn "L" (F-030)'
export PKGS="./intelligence/intent/ ./observability/traces/ ./core/tenant/"
m $IN '	if s.opt.NarrowParentLabel != "" {' '	if false {' 'L5b-15 lời giải thích khe thiếu bỏ kiểu cha do pack khai (F-005)'
m $IN '		by = s.opt.NarrowParentLabel + " đã nêu hay " + by' '		by = "dự án đã nêu hay " + by' 'L5b-16 lời giải thích khe thiếu viết cứng "dự án" (F-005)'
m $P '				NarrowParentLabel: inf.NarrowParent.Label,' '' 'L5b-17 pack.go không nối tên gọi kiểu cha (F-005)'
export PKGS="./data/ingestion/"
m data/ingestion/bridge.go ', ReportedBy: o.Sender, Note: o.Note},' ', ReportedBy: o.Sender},' 'L5b-18 fact mất ghi chú suy luận (F-008)'
m platform/pgstore/load.go '		notes[i] = pgtype.Text{String: f.Source.Note, Valid: f.Source.Note != ""}' '		notes[i] = pgtype.Text{}' 'L5b-19 [PG] Load không ghi ghi chú suy luận (F-008)'
m platform/pgstore/pgstore.go 'observed_at, reported_by, coalesce(note, '"''"')' 'observed_at, reported_by, '"''"'' 'L5b-20 [PG] Facts không đọc ghi chú suy luận (F-008)'
