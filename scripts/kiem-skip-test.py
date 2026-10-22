#!/usr/bin/env python3
"""Cổng "không test nào bị SKIP hay FAIL" cho CI có PG.

Đọc luồng `go test -json` (từ tệp hoặc stdin), in danh sách test bị bỏ qua kèm lý do, và thoát 1 nếu có test nào bị SKIP mà
không nằm trong danh sách cho phép. Lý do: ở CI đã có PG và mọi công cụ, một test SKIP nghĩa là một thứ chưa được kiểm mà
báo cáo vẫn xanh — "bỏ qua" không phải "đạt".

Cổng cũng đỏ khi luồng có test hoặc gói FAIL. Đây là lớp thứ hai: lớp thứ nhất là mã thoát của `go test` (bước CI đặt pipefail để
`| tee` không nuốt nó), nhưng luồng JSON vẫn kể lại test hỏng kể cả khi ai đó làm mất lớp thứ nhất — cổng không được báo "XANH"
cho một luồng mà chính nó cho thấy có test hỏng. Gói không dịch được chỉ có sự kiện fail mức gói (không có Test) nên được đếm riêng.

Dùng:
    go test -count=1 -json ./... > test.json            # thoát khác 0 nếu có test đỏ
    python3 -I scripts/kiem-skip-test.py test.json      # thoát 1 nếu có test SKIP hoặc FAIL
    python3 -I scripts/kiem-skip-test.py --cho-phep scripts/skip-cho-phep.txt test.json

Danh sách cho phép (tuỳ chọn): mỗi dòng "<gói> <TênTest> # lý do"; dòng trống và dòng bắt đầu bằng # bị bỏ qua. Mục không còn
khớp test nào bị SKIP cũng làm cổng đỏ (danh sách thừa che lỗi sau này). Chỉ test có tên (Test != "") được tính là test bị bỏ qua;
gói không có tệp test (Action skip ở mức gói) không phải test bị bỏ qua. Danh sách cho phép chỉ áp cho SKIP, không bao giờ cho FAIL.

Mã thoát: 0 sạch · 1 có SKIP ngoài danh sách, mục cho phép thừa, hoặc có test / gói FAIL · 2 lỗi dùng / không đọc được đầu vào.
"""
import argparse
import json
import sys
from collections import defaultdict


def doc(path):
    if path in (None, "-"):
        return sys.stdin
    return open(path, encoding="utf-8")


def phan_tich(luong):
    """Trả (số test theo kết cục, danh sách SKIP [(gói, test, lý do)], test FAIL [(gói, test)], gói FAIL, số dòng không phải JSON)."""
    ket_cuc = defaultdict(int)
    skip = []
    test_hong = []
    goi_hong = []
    dau_ra = defaultdict(list)  # (gói, test) → các dòng output, để lấy lý do
    khong_json = 0
    for dong in luong:
        dong = dong.strip()
        if not dong:
            continue
        try:
            ev = json.loads(dong)
        except json.JSONDecodeError:
            khong_json += 1
            continue
        goi, ten, hanh_dong = ev.get("Package", ""), ev.get("Test", ""), ev.get("Action", "")
        if not ten:
            if hanh_dong == "fail" and goi and goi not in goi_hong:  # gói hỏng: test FAIL, hoặc không dịch được (không có sự kiện mức test)
                goi_hong.append(goi)
            continue
        if hanh_dong == "output":
            dau_ra[(goi, ten)].append(ev.get("Output", "").rstrip("\n"))
        elif hanh_dong in ("pass", "fail", "skip"):
            ket_cuc[hanh_dong] += 1
            if hanh_dong == "fail":
                test_hong.append((goi, ten))
            if hanh_dong == "skip":
                ly_do = ""
                for o in dau_ra[(goi, ten)]:
                    o = o.strip()
                    if ".go:" in o:  # "file_test.go:34: lý do"
                        ly_do = o.split(".go:", 1)[1].split(":", 1)[-1].strip()
                if not ly_do:
                    ly_do = "(không có lý do trong đầu ra)"
                skip.append((goi, ten, ly_do))
    return ket_cuc, skip, test_hong, goi_hong, khong_json


def doc_cho_phep(path):
    out = {}
    with open(path, encoding="utf-8") as f:
        for so, dong in enumerate(f, 1):
            dong = dong.strip()
            if not dong or dong.startswith("#"):
                continue
            phan = dong.split("#", 1)[0].split()
            if len(phan) != 2:
                sys.exit(f"{path}:{so}: cần '<gói> <TênTest> # lý do', gặp {dong!r}")
            out[(phan[0], phan[1])] = dong
    return out


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("dau_vao", nargs="?", help="tệp đầu ra của go test -json (mặc định stdin)")
    ap.add_argument("--cho-phep", help="tệp danh sách SKIP được phép")
    args = ap.parse_args()
    try:
        with doc(args.dau_vao) as f:
            ket_cuc, skip, test_hong, goi_hong, khong_json = phan_tich(f)
    except OSError as e:
        print(f"không đọc được đầu vào: {e}", file=sys.stderr)
        return 2
    cho_phep = doc_cho_phep(args.cho_phep) if args.cho_phep else {}
    tong = sum(ket_cuc.values())
    print(f"test: {tong} (pass {ket_cuc['pass']}, fail {ket_cuc['fail']}, skip {ket_cuc['skip']}); dòng không phải JSON: {khong_json}")
    if tong == 0:
        print("ĐỎ: không có test nào trong đầu vào — luồng go test -json rỗng hoặc hỏng, không thể báo 'không SKIP'", file=sys.stderr)
        return 2
    ngoai = [s for s in skip if (s[0], s[1]) not in cho_phep]
    thua = [k for k in cho_phep if k not in {(s[0], s[1]) for s in skip}]
    for goi, ten, ly_do in skip:
        nhan = "được phép" if (goi, ten) in cho_phep else "SKIP"
        print(f"  {nhan}: {goi} {ten}: {ly_do}")
    for goi, ten in thua:
        print(f"  THỪA: {goi} {ten} có trong danh sách cho phép mà không bị SKIP — bỏ mục", file=sys.stderr)
    for goi, ten in test_hong:
        print(f"  FAIL: {goi} {ten}")
    tt = {g for g, _ in test_hong}
    for goi in goi_hong:
        if goi not in tt:  # gói hỏng mà không có test nào FAIL: không dịch được, hoặc chết giữa chừng
            print(f"  FAIL (gói): {goi} — hỏng mà không có test nào báo fail (không dịch được, hoặc tiến trình test chết)")
    if ngoai or thua or test_hong or goi_hong:
        print(f"ĐỎ: {len(ngoai)} test SKIP ngoài danh sách cho phép, {len(thua)} mục cho phép thừa, {len(test_hong)} test FAIL, "
              f"{len(goi_hong)} gói FAIL", file=sys.stderr)
        return 1
    print("XANH: không test nào bị SKIP ngoài danh sách cho phép, không test hay gói nào FAIL")
    return 0


if __name__ == "__main__":
    sys.exit(main())
