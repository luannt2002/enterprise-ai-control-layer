#!/usr/bin/env python3
"""Đóng băng bộ giữ lại theo ADR-0006 §4–5, MÁY MÓC, không đọc nội dung ca.

    python3 -I scripts/dong-bang-giu-lai.py [thư mục]          # mặc định docs/g7/holdout
    python3 -I scripts/dong-bang-giu-lai.py --kiem [thư mục]   # tính lại, so với bản đã đóng băng, không ghi

Đã có holdout-manifest.json thì từ chối ghi (đóng băng lại phải là việc có chủ đích: xoá + commit lý do).
--kiem tính lại việc chia từ kho.yaml / kho-manifest.json / loai-truoc-dong-bang.txt và so từng tệp, từng
ca (tập, frozen_hash) với bản đã đóng băng: in KHỚP (thoát 0) hoặc LỆCH kèm lý do (thoát 1). Bỏ qua frozen_at.

Đầu vào: kho.yaml (ca, định dạng evals/win.go) + kho-manifest.json (hồ sơ ca, mảng) + (nếu có)
loai-truoc-dong-bang.txt.
Đầu ra (cùng thư mục): pilot.yaml, h1.yaml, h2.yaml, holdout-manifest.json; exposed.yaml nếu có ca bị loại.

Loại trước khi chia (ADR-0006 §5): ca đã LỘ cho người viết mã EACL trước khi đóng băng không còn là bộ
giữ lại. loai-truoc-dong-bang.txt mỗi dòng "<id ca> <lý do>" (dòng trống và dòng "#" bỏ qua). Ca bị loại
không vào pilot / h1 / h2, mà vào exposed.yaml (chỉ báo cáo), set = "exposed" kèm excluded_reason trong
manifest. Việc chia dưới đây chạy trên các ca CÒN LẠI. Id không có trong kho hay trùng là lỗi.

Luật chia: trong từng lớp hỏng, xếp ca theo sha256("eacl-holdout-2026-10-20|" + case_id) tăng dần
(chuỗi hex); 2 ca đầu → PILOT, 13 ca kế → H1, còn lại → H2. Lớp không đủ ca thì PILOT lấy trước,
rồi tới H1.

Mỗi ca được cắt theo KHỐI VĂN BẢN (từ dòng "  - id: WIN-…" tới trước dòng "  - id:" kế tiếp), nên tệp ra
giữ nguyên từng byte của ca. frozen_hash của ca = sha256 khối đó. Phần đầu tệp (suite, tenant,
reference_date, "cases:") chép nguyên.

Công cụ CHỈ in số đếm và băm, không in nội dung ca: người viết mã của EACL không được đọc kỳ vọng
của bộ giữ lại trước khi chạy (ADR-0006 §3).
"""
import hashlib
import json
import re
import sys
from datetime import datetime, timezone, timedelta
from pathlib import Path

SALT = "eacl-holdout-2026-10-20|"
QUOTA = (("pilot", 2), ("h1", 13))  # phần còn lại → h2
ID_LINE = re.compile(r"^  - id: (WIN-[IA]-\d{3})\s*$")
CLASS_LINE = re.compile(r"^    failure_class: (\S+)\s*$")


def sha(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def split_blocks(text: str):
    lines = text.splitlines(keepends=True)
    head, blocks, cur = [], [], None
    for ln in lines:
        m = ID_LINE.match(ln.rstrip("\n"))
        if m:
            cur = {"id": m.group(1), "lines": [ln]}
            blocks.append(cur)
        elif cur is None:
            head.append(ln)
        else:
            cur["lines"].append(ln)
    if not head or head[-1].strip() != "cases:":
        sys.exit("kho.yaml: phần đầu phải kết thúc bằng dòng 'cases:' ngay trước ca đầu tiên")
    for b in blocks:
        b["text"] = "".join(b["lines"])
        cls = [CLASS_LINE.match(x.rstrip("\n")) for x in b["lines"]]
        cls = [c.group(1) for c in cls if c]
        if len(cls) != 1:
            sys.exit(f"{b['id']}: cần đúng một dòng failure_class, có {len(cls)}")
        b["class"] = cls[0]
    return "".join(head), blocks


def assign(blocks):
    by_class = {}
    for b in blocks:
        by_class.setdefault(b["class"], []).append(b)
    out = {}
    for cls, bs in by_class.items():
        order = sorted(bs, key=lambda b: sha((SALT + b["id"]).encode()))
        i = 0
        for name, n in QUOTA:
            for b in order[i:i + n]:
                out[b["id"]] = name
            i += n
        for b in order[i:]:
            out[b["id"]] = "h2"
    return out


def read_exclusions(path, ids):
    out = {}
    if not path.exists():
        return out
    for n, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        cid, _, reason = line.partition(" ")
        if cid not in ids:
            sys.exit(f"{path.name}:{n}: {cid!r} không có trong kho.yaml")
        if cid in out:
            sys.exit(f"{path.name}:{n}: {cid} trùng")
        if not reason.strip():
            sys.exit(f"{path.name}:{n}: {cid} thiếu lý do")
        out[cid] = reason.strip()
    return out


def main():
    args = sys.argv[1:]
    check = "--kiem" in args
    args = [a for a in args if a != "--kiem"]
    d = Path(args[0] if args else "docs/g7/holdout")
    kho, man, excl_path = d / "kho.yaml", d / "kho-manifest.json", d / "loai-truoc-dong-bang.txt"
    if (d / "holdout-manifest.json").exists() and not check:
        sys.exit(f"{d}/holdout-manifest.json đã có: bộ đã đóng băng, không ghi đè. Muốn kiểm thì dùng --kiem.")
    raw_bytes = kho.read_bytes()  # đọc byte thô: read_text ở chế độ văn bản tự đổi CRLF thành LF nên không thấy CR
    if b"\r" in raw_bytes:
        sys.exit("kho.yaml có ký tự CR (CRLF): đổi về LF trước khi đóng băng")
    raw = raw_bytes.decode("utf-8")
    if not raw.endswith("\n"):
        sys.exit("kho.yaml không kết thúc bằng xuống dòng: khối ca cuối sẽ dính vào ca kế ở tệp ra")
    head, blocks = split_blocks(raw)
    ids = [b["id"] for b in blocks]
    if len(set(ids)) != len(ids):
        sys.exit("kho.yaml: id ca trùng")
    excluded = read_exclusions(excl_path, set(ids))
    entries = json.loads(man.read_text(encoding="utf-8"))
    if isinstance(entries, dict):
        entries = entries.get("cases", [])
    by_id = {}
    for e in entries:
        cid = e.get("case_id")
        if not cid or cid in by_id:
            sys.exit(f"kho-manifest.json: mục thiếu hoặc trùng case_id: {cid!r}")
        by_id[cid] = e
    if set(by_id) != set(ids):
        sys.exit(f"manifest ↔ ca lệch: thiếu {sorted(set(ids) - set(by_id))[:5]}, thừa {sorted(set(by_id) - set(ids))[:5]}")

    sets = assign([b for b in blocks if b["id"] not in excluded])
    for cid in excluded:
        sets[cid] = "exposed"
    names = ("pilot", "h1", "h2") + (("exposed",) if excluded else ())
    files = {}
    counts = {}
    texts = {}
    for name in names:
        part = [b for b in sorted(blocks, key=lambda b: b["id"]) if sets[b["id"]] == name]
        text = head + "".join(b["text"] for b in part)
        texts[f"{name}.yaml"] = text
        files[f"{name}.yaml"] = sha(text.encode("utf-8"))
        counts[name] = {}
        for b in part:
            counts[name][b["class"]] = counts[name].get(b["class"], 0) + 1

    cases = []
    for b in sorted(blocks, key=lambda b: b["id"]):
        e = dict(by_id[b["id"]])
        e["set"] = sets[b["id"]]
        if b["id"] in excluded:
            e["excluded_reason"] = excluded[b["id"]]
        e["frozen_hash"] = sha(b["text"].encode("utf-8"))
        cases.append(e)
    if check:
        old = json.loads((d / "holdout-manifest.json").read_text(encoding="utf-8"))
        diffs = []
        if old.get("files_sha256") != files:
            diffs.append(f"băm tệp: đã đóng băng {old.get('files_sha256')} ≠ tính lại {files}")
        for k, v in files.items():
            if not (d / k).exists() or sha((d / k).read_bytes()) != v:
                diffs.append(f"{k} trên đĩa khác bản tính lại")
        old_cases = {c["case_id"]: (c.get("set"), c.get("frozen_hash")) for c in old.get("cases", [])}
        new_cases = {c["case_id"]: (c["set"], c["frozen_hash"]) for c in cases}
        if old_cases != new_cases:
            lech = sorted(k for k in set(old_cases) | set(new_cases) if old_cases.get(k) != new_cases.get(k))
            diffs.append(f"{len(lech)} ca lệch tập / frozen_hash, ví dụ {lech[:5]}")
        if diffs:
            for x in diffs:
                print("LỆCH:", x)
            sys.exit(1)
        print(f"KHỚP: {len(cases)} ca; " + "; ".join(f"{n} {sum(counts[n].values())}" for n in names))
        return
    for k, text in texts.items():
        (d / k).write_text(text, encoding="utf-8")
    out = {
        "version": 1,
        "rule": "ADR-0006 §4–5: ca trong loai-truoc-dong-bang.txt → exposed; trong từng lớp của phần còn lại, xếp theo sha256(salt + case_id); 2 → pilot, 13 → h1, còn lại → h2",
        "salt": SALT,
        "frozen_at": datetime.now(timezone(timedelta(hours=7))).isoformat(timespec="seconds"),
        "source_files_sha256": {"kho.yaml": sha(kho.read_bytes()), "kho-manifest.json": sha(man.read_bytes())}
        | ({excl_path.name: sha(excl_path.read_bytes())} if excl_path.exists() else {}),
        "files_sha256": files,
        "counts": counts,
        "cases": cases,
    }
    (d / "holdout-manifest.json").write_text(json.dumps(out, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    print(f"ca: {len(blocks)} · loại trước khi chia: {len(excluded)}")
    for name in names:
        print(f"{name}: {sum(counts[name].values())} ca · " + " ".join(f"{k}={v}" for k, v in sorted(counts[name].items())))
    for k, v in files.items():
        print(f"sha256 {k} {v}")
    print(f"sha256 holdout-manifest.json {sha((d / 'holdout-manifest.json').read_bytes())}")


if __name__ == "__main__":
    main()
