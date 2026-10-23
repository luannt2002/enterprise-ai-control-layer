#!/usr/bin/env python3
"""Đóng băng bộ ca giữ lại độc lập của Phase 0a (HOP-DONG 0a §12) bằng CAM KẾT BĂM (ADR-0006 §10): ghi holdout-manifest.json gồm
sha256 của tệp ca và các tệp thế giới giả lập đi kèm, số đếm theo lớp, và băm bộ đồ nghề của người soạn — KHÔNG chép nội dung ca.
Bản cho bộ ca WIN là scripts/cam-ket-giu-lai.py, cho bộ ca lệnh là scripts/cam-ket-bo-ca-lenh.py; tệp này tách riêng để không đổi
hành vi của hai bản đó.

    python3 -I scripts/cam-ket-bo-ca-0a.py <tệp ca> <thư mục ra> --bo-do-nghe <thư mục> --the-gioi <tệp> [--the-gioi <tệp> …]

Tệp ca: YAML {suite: 0a, cases: [...]}. Mỗi ca có id (0A-I-NNN), class (mã lớp LC-xx hoặc "tự nhiên"), input, world (tên một tệp
--the-gioi, hoặc danh sách tên không trùng khi ca dùng nhiều thế giới — vd lượt đi qua hai tenant), rationale; ca có blocked_by (phụ thuộc câu chưa chốt) KHÔNG được có expected, ca không blocked phải có expected (§12).
Chỉ in sha256 và số đếm, không in input hay kỳ vọng (phiên viết mã không được đọc chúng). Từ chối khi thư mục ra đã có manifest
(cam kết không ghi đè), tệp hỏng, thiếu khoá, id sai mẫu / trùng, world không phải tệp thế giới đã khai, kỳ vọng đặt cho ca blocked.
Thoát 0 khi ghi xong, 2 khi bị từ chối.
"""
import argparse
import datetime
import hashlib
import json
import re
import subprocess
import sys
from collections import Counter
from pathlib import Path

import yaml

REQUIRED = ["id", "class", "input", "world", "rationale"]
ID = re.compile(r"^0A-I-\d{3}$")


def refuse(msg):
    print(f"TỪ CHỐI: {msg}", file=sys.stderr)
    sys.exit(2)


def sha(b):
    return hashlib.sha256(b).hexdigest()


def kit_hashes(kit):
    """sha256 từng tệp của bộ đồ nghề và một băm gộp của cả cây — cùng cách tính với scripts/cam-ket-giu-lai.py."""
    files = {str(p.relative_to(kit)): sha(p.read_bytes()) for p in sorted(kit.rglob("*")) if p.is_file()}
    tree = sha("".join(f"{k} {v}\n" for k, v in sorted(files.items())).encode())
    return files, tree


def empty(v):
    return v is None or (isinstance(v, (str, list, dict)) and len(v) == 0)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("cases")
    ap.add_argument("out")
    ap.add_argument("--bo-do-nghe", required=True, help="thư mục bộ đồ nghề người soạn được đọc (ghi băm từng tệp)")
    ap.add_argument("--the-gioi", action="append", default=[], help="tệp thế giới giả lập đi kèm bộ ca (chỉ ghi sha256)")
    a = ap.parse_args()
    src, out, kit = Path(a.cases), Path(a.out), Path(a.bo_do_nghe)
    manifest = out / "holdout-manifest.json"
    if manifest.exists():
        refuse(f"{manifest} đã có — cam kết không ghi đè")
    if not src.is_file():
        refuse(f"không có tệp ca {src}")
    if not kit.is_dir():
        refuse(f"không có bộ đồ nghề {kit}")
    worlds = [Path(w) for w in a.the_gioi]
    if not worlds:
        refuse("chưa khai tệp thế giới nào (--the-gioi)")
    for w in worlds:
        if not w.is_file():
            refuse(f"không có tệp thế giới {w}")
    names = [w.name for w in worlds]
    if len(set(names)) != len(names) or src.name in names:
        refuse("tên tệp thế giới trùng nhau hoặc trùng tệp ca")
    raw = src.read_bytes()
    try:
        doc = yaml.safe_load(raw)
    except yaml.YAMLError as e:
        refuse(f"{src.name} không đọc được YAML ({type(e).__name__})")
    if not isinstance(doc, dict) or str(doc.get("suite")) != "0a" or not isinstance(doc.get("cases"), list) or not doc["cases"]:
        refuse(f"{src.name} không phải bộ ca 0a có ca")
    per_class, ids, used_worlds, keys = Counter(), Counter(), Counter(), Counter()
    blocked = 0
    for n, c in enumerate(doc["cases"], 1):
        if not isinstance(c, dict):
            refuse(f"ca thứ {n} không phải ánh xạ")
        missing = [k for k in REQUIRED if empty(c.get(k))]
        if missing:
            refuse(f"ca thứ {n}: thiếu khoá {missing}")
        cid = str(c["id"])
        if not ID.match(cid):
            refuse(f"ca thứ {n}: id không đúng mẫu 0A-I-NNN")
        wl = [str(w) for w in c["world"]] if isinstance(c["world"], list) else [str(c["world"])]
        if any(w not in names for w in wl) or len(set(wl)) != len(wl):
            refuse(f"ca thứ {n}: world không phải tệp thế giới đã khai (hoặc danh sách có tên trùng)")
        is_blocked = not empty(c.get("blocked_by"))
        if is_blocked and not empty(c.get("expected")):
            refuse(f"ca thứ {n}: ca blocked_by không được đặt kỳ vọng (§12)")
        if not is_blocked and empty(c.get("expected")):
            refuse(f"ca thứ {n}: thiếu expected")
        blocked += is_blocked
        ids[cid] += 1
        per_class[str(c["class"])] += 1
        used_worlds.update(wl)
        keys.update(c.keys())
    dup = sorted(k for k, v in ids.items() if v > 1)
    if dup:
        refuse(f"id trùng: {', '.join(dup)}")
    files, tree = kit_hashes(kit)
    rev = subprocess.run(["git", "rev-parse", "HEAD"], capture_output=True, text=True).stdout.strip()
    m = {
        "version": 1,
        "rule": "ADR-0006 §10: cam kết băm — chỉ commit sha256 + số đếm; nội dung công bố khi chạy",
        "frozen_at": datetime.datetime.now().astimezone().isoformat(timespec="seconds"),
        "revision_at_freeze": rev,
        "content_published": False,
        "files_sha256": {src.name: sha(raw), **{w.name: sha(w.read_bytes()) for w in worlds}},
        "counts": {
            "total": sum(per_class.values()),
            "natural": per_class.get("tự nhiên", 0),
            "blocked": blocked,
            "per_class": dict(sorted(per_class.items())),
            "per_world": dict(sorted(used_worlds.items())),
            "keys": dict(sorted(keys.items())),
        },
        "author_kit": {"tree_sha256": tree, "files_sha256": files},
    }
    out.mkdir(parents=True, exist_ok=True)
    manifest.write_text(json.dumps(m, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    for name, h in m["files_sha256"].items():
        print(f"sha256 {name}: {h}")
    for k, v in m["counts"]["per_class"].items():
        print(f"lớp {k:<10} {v} ca")
    print(f"TỔNG {m['counts']['total']} ca · tự nhiên {m['counts']['natural']} · blocked {blocked}")
    print(f"băm bộ đồ nghề: {tree} ({len(files)} tệp)")
    print(f"đã ghi {manifest}")


if __name__ == "__main__":
    main()
