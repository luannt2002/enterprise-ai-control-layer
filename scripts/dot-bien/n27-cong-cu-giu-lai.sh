#!/usr/bin/env bash
# Danh mục đột biến cho công cụ bộ giữ lại (scripts/bang-cong-h1.py, scripts/dong-bang-giu-lai.py), bắt bằng gói
# test Go scripts/kiemcongcu (chạy python3 trên dữ liệu giả). Mỗi dòng: m <tệp> <chuỗi gốc> <chuỗi đột biến> <nhãn>.
# Không có dòng cho phép so băm hai tệp: đã bỏ vì tương đương với kiểm dòng đầu (hai tệp giống hệt luôn có một tệp
# sai dòng đầu). scripts/chay-giu-lai.sh không có test tự động (cần repo git + origin giả); các đường của nó đã thử tay, ghi ở commit.
cd "$(dirname "$0")/../.." || exit 1
m() { scripts/mutate.sh "$1" "$2" "$3" "$4"; echo "rc=$?"; }
export PKGS="./scripts/kiemcongcu/"
B=scripts/bang-cong-h1.py
m $B 'ok = {cid: mem["cases"][cid][0] and pg["cases"][cid][0] for cid in ids}' 'ok = {cid: mem["cases"][cid][0] for cid in ids}' 'BC1 chỉ đọc đường bộ nhớ'
m $B 'replay_ok = all(path["replay"] == (len(ids), 0) for path in (mem, pg))' 'replay_ok = all(path["replay"][1] == 0 for path in (mem, pg))' 'BC2 phát lại không cần đủ số ca'
m $B '    t = scan_text(answer)' '    t = answer' 'BC3 quét canary trên chuỗi %q chưa giải mã'
m $B '    return unicodedata.normalize("NFKC", unquote_go(answer))' '    return unquote_go(answer)' 'BC4 bỏ NFKC'
m $B 're.compile(r"(?<![\d.,])7[.,\s]?731(?!\d|[.,]\d)")' 're.compile(r"7[.,\s]?731")' 'BC5 số canary 7731 không có biên'
m $B '|def-build|dự án z|' '|def-build|' 'BC6 bỏ canary tên "dự án z"'
m $B '    if problems:' '    if False:' 'BC7 không chặn đầu vào hỏng'
m $B '    if not pg["head"].startswith("ĐƯỜNG DỮ LIỆU: PostgreSQL"):' '    if False:' 'BC9 không kiểm đường PG'
m $B 'all(p_all.get(c, (0, 0))[1] >= 5 for c in CLASSES)' 'True' 'BC10 G6 bỏ điều kiện mỗi lớp ≥ 5'
m $B 'replay_ok and indep == len(ids)' 'replay_ok' 'BC11 G6 bỏ điều kiện 100% độc lập'
m $B 'g6 = len(ids) >= 100 and' 'g6 = len(ids) >= 0 and' 'BC12 G6 bỏ điều kiện ≥ 100 ca'
m $B 'g5 = full(p_all, ["permission", "unsupported_answer"]) and not leaked' 'g5 = full(p_all, ["permission", "unsupported_answer"])' 'BC13 G5 bỏ canary'
m $B '        kept = ids - hinted' '        kept = ids' 'BC14 độ nhạy 2 không bỏ ca gợi ý'
m $B '        if not ID_RE.match(cid):' '        if False:' 'BC15 nhận dòng gợi ý sai dạng'
D=scripts/dong-bang-giu-lai.py
m $D '    if (d / "holdout-manifest.json").exists() and not check:' '    if False:' 'DB1 cho ghi đè bộ đã đóng băng'
m $D '    if b"\r" in raw_bytes:' '    if False:' 'DB2 nhận CRLF'
m $D '    if not raw.endswith("\n"):' '    if False:' 'DB3 nhận kho thiếu xuống dòng cuối'
m $D '            if not (d / k).exists() or sha((d / k).read_bytes()) != v:' '            if False:' 'DB4 --kiem không so tệp trên đĩa'
m $D '            sys.exit(f"{path.name}:{n}: {cid!r} không có trong kho.yaml")' '            pass' 'DB5 nhận id loại không có trong kho'
m $D '            sys.exit(f"{path.name}:{n}: {cid} thiếu lý do")' '            pass' 'DB6 nhận dòng loại thiếu lý do'
