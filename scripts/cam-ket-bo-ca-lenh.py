#!/usr/bin/env python3
"""Đóng băng bộ ca giữ lại của một hợp đồng việc kiểu LỆNH (entry_type COMMAND, khuôn docs/CONVENTIONS.md §5) bằng CAM KẾT
BĂM (ADR-0006 §10): ghi holdout-manifest.json gồm sha256 của tệp ca, số đếm theo lớp và băm bộ đồ nghề của người soạn, KHÔNG
chép nội dung ca. Bản cho bộ ca WIN là scripts/cam-ket-giu-lai.py; tệp này tách riêng để không đổi hành vi của bản đó.

    python3 -I scripts/cam-ket-bo-ca-lenh.py <tệp ca> <thư mục ra> --bo-do-nghe <thư mục>

Tệp ca: YAML {suite: lenh, cases: [...]}, mỗi ca đúng 23 khoá của §5, entry_type COMMAND, lớp ca (A01…) ở khoá scenario.
Chỉ in sha256 và số đếm, không in input hay kỳ vọng (phiên viết mã không được đọc chúng). Từ chối khi thư mục ra đã có
manifest (cam kết không ghi đè), tệp hỏng, ca thiếu / thừa khoá, entry_type khác COMMAND, scenario rỗng, id trùng.
Thoát 0 khi ghi xong, 2 khi bị từ chối.
"""
import argparse
import datetime
import hashlib
import json
import subprocess
import sys
from collections import Counter
from pathlib import Path

import yaml

KEYS = [
    "id", "tenant", "domain", "business_process", "scenario", "entry_type", "category", "input", "preconditions",
    "sources", "expected_route", "expected_tools", "expected_context", "expected_output_properties", "expected_state",
    "must_not_do", "business_rules", "permissions", "failure_injection", "expected_fallback", "metrics", "severity",
    "acceptance_criteria",
]


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


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("cases")
    ap.add_argument("out")
    ap.add_argument("--bo-do-nghe", required=True, help="thư mục bộ đồ nghề người soạn được đọc (ghi băm từng tệp)")
    a = ap.parse_args()
    src, out, kit = Path(a.cases), Path(a.out), Path(a.bo_do_nghe)
    manifest = out / "holdout-manifest.json"
    if manifest.exists():
        refuse(f"{manifest} đã có — cam kết không ghi đè")
    if not src.is_file():
        refuse(f"không có tệp ca {src}")
    if not kit.is_dir():
        refuse(f"không có bộ đồ nghề {kit}")
    raw = src.read_bytes()
    try:
        doc = yaml.safe_load(raw)
    except yaml.YAMLError as e:
        refuse(f"{src.name} không đọc được YAML ({type(e).__name__})")
    if not isinstance(doc, dict) or doc.get("suite") != "lenh" or not isinstance(doc.get("cases"), list) or not doc["cases"]:
        refuse(f"{src.name} không phải bộ ca lệnh có ca")
    per_class, per_category, ids = Counter(), Counter(), Counter()
    for n, c in enumerate(doc["cases"], 1):
        if not isinstance(c, dict):
            refuse(f"ca thứ {n} không phải ánh xạ")
        missing, extra = [k for k in KEYS if k not in c], sorted(k for k in c if k not in KEYS)
        if missing or extra:
            refuse(f"ca thứ {n}: thiếu khoá {missing}, thừa khoá {extra}")
        if c["entry_type"] != "COMMAND":
            refuse(f"ca thứ {n}: entry_type phải là COMMAND")
        if not str(c["scenario"] or "").strip():
            refuse(f"ca thứ {n}: scenario (lớp ca) rỗng")
        ids[str(c["id"])] += 1
        per_class[str(c["scenario"])] += 1
        per_category[str(c["category"])] += 1
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
        "files_sha256": {src.name: sha(raw)},
        "counts": {
            "total": sum(per_class.values()),
            "per_class": dict(sorted(per_class.items())),
            "per_category": dict(sorted(per_category.items())),
        },
        "author_kit": {"tree_sha256": tree, "files_sha256": files},
    }
    out.mkdir(parents=True, exist_ok=True)
    manifest.write_text(json.dumps(m, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"sha256 {src.name}: {m['files_sha256'][src.name]}")
    for k, v in m["counts"]["per_class"].items():
        print(f"lớp {k:<10} {v} ca")
    print(f"TỔNG {m['counts']['total']} ca")
    print(f"băm bộ đồ nghề: {tree} ({len(files)} tệp)")
    print(f"đã ghi {manifest}")


if __name__ == "__main__":
    main()
