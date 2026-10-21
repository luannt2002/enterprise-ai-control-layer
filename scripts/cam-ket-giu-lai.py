#!/usr/bin/env python3
"""Đóng băng một bộ giữ lại bằng CAM KẾT BĂM (ADR-0006 §10): ghi holdout-manifest.json gồm sha256 của tệp ca và số
đếm, KHÔNG chép nội dung ca. Commit + push manifest là bằng chứng "đóng băng trước khi chạy"; nội dung ở máy tới lúc
chạy, khi đó scripts/chay-giu-lai.sh so băm với manifest này.

    python3 -I scripts/cam-ket-giu-lai.py <tệp ca> <thư mục ra> [--bo-do-nghe <thư mục>]

Chỉ in sha256 và số đếm, không in câu hỏi / kỳ vọng (phiên viết mã không được đọc chúng). Từ chối khi thư mục ra đã có
manifest (cam kết không ghi đè), tệp ca hỏng / không phải bộ WIN / trùng id. Thoát 0 khi ghi xong, 2 khi bị từ chối.
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


def refuse(msg):
    print(f"TỪ CHỐI: {msg}", file=sys.stderr)
    sys.exit(2)


def sha(b):
    return hashlib.sha256(b).hexdigest()


def kit_hashes(kit):
    """sha256 từng tệp của bộ đồ nghề, theo đường tương đối, và một băm gộp của cả cây (đường + băm, đã sắp xếp)."""
    files = {str(p.relative_to(kit)): sha(p.read_bytes()) for p in sorted(kit.rglob("*")) if p.is_file()}
    tree = sha("".join(f"{k} {v}\n" for k, v in sorted(files.items())).encode())
    return files, tree


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("cases")
    ap.add_argument("out")
    ap.add_argument("--bo-do-nghe", help="thư mục bộ đồ nghề người soạn được đọc (ghi băm từng tệp)")
    a = ap.parse_args()
    src, out = Path(a.cases), Path(a.out)
    manifest = out / "holdout-manifest.json"
    if manifest.exists():
        refuse(f"{manifest} đã có — cam kết không ghi đè")
    if not src.is_file():
        refuse(f"không có tệp ca {src}")
    raw = src.read_bytes()
    try:
        doc = yaml.safe_load(raw)
    except yaml.YAMLError as e:
        refuse(f"{src.name} không đọc được YAML ({type(e).__name__})")
    if not isinstance(doc, dict) or doc.get("suite") != "win" or not isinstance(doc.get("cases"), list) or not doc["cases"]:
        refuse(f"{src.name} không phải bộ ca WIN có ca")
    per_class, with_value, ids = Counter(), 0, Counter()
    for c in doc["cases"]:
        if not isinstance(c, dict):
            refuse(f"{src.name} có ca không phải ánh xạ")
        ids[str(c.get("id"))] += 1
        per_class[str(c.get("failure_class"))] += 1
        if isinstance(c.get("expect"), dict) and c["expect"].get("value") is not None:
            with_value += 1
    dup = sorted(k for k, n in ids.items() if n > 1)
    if dup:
        refuse(f"id trùng: {', '.join(dup)}")
    rev = subprocess.run(["git", "rev-parse", "HEAD"], capture_output=True, text=True).stdout.strip()
    m = {
        "version": 1,
        "rule": "ADR-0006 §10: cam kết băm — chỉ commit sha256 + số đếm; nội dung công bố khi chạy (scripts/chay-giu-lai.sh so băm)",
        "frozen_at": datetime.datetime.now().astimezone().isoformat(timespec="seconds"),
        "revision_at_freeze": rev,
        "content_published": False,
        "files_sha256": {src.name: sha(raw)},
        "counts": {"total": sum(per_class.values()), "with_value": with_value, "per_class": dict(sorted(per_class.items()))},
    }
    if a.bo_do_nghe:
        files, tree = kit_hashes(Path(a.bo_do_nghe))
        m["author_kit"] = {"tree_sha256": tree, "files_sha256": files}
    out.mkdir(parents=True, exist_ok=True)
    manifest.write_text(json.dumps(m, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"sha256 {src.name}: {m['files_sha256'][src.name]}")
    for k, v in m["counts"]["per_class"].items():
        print(f"lớp {k:<18} {v} ca")
    print(f"TỔNG {m['counts']['total']} ca · có kỳ vọng số liệu {with_value}")
    if a.bo_do_nghe:
        print(f"băm bộ đồ nghề: {m['author_kit']['tree_sha256']} ({len(m['author_kit']['files_sha256'])} tệp)")
    print(f"đã ghi {manifest}")


if __name__ == "__main__":
    main()
