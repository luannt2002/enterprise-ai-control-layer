# So kết quả go test -json theo TÊN (gói + test): mất, đổi trạng thái, mới. Rỗng không phải xanh: số tên phải > 0.
# "đổi" ở đây là đổi TRẠNG THÁI; đổi thân / kỳ vọng mà giữ tên chỉ hiện ở scripts/cong/so-than (F-034; cong.sh chạy khi đặt CONG_MOC).
import json, sys
def load(p):
    out = {}
    for l in open(p):
        try:
            e = json.loads(l)
        except ValueError:
            continue
        if e.get("Test") and e.get("Action") in ("pass", "fail", "skip"):
            out[(e["Package"], e["Test"])] = e["Action"]
    return out
a, b = load(sys.argv[1]), load(sys.argv[2])
assert len(a) > 0 and len(b) > 0, "tệp rỗng"
mat = sorted(k for k in a if k not in b)
doi = sorted((k, a[k], b[k]) for k in a if k in b and a[k] != b[k])
moi = sorted(k for k in b if k not in a)
from collections import Counter
print("gốc", len(a), dict(Counter(a.values())), "· mới", len(b), dict(Counter(b.values())))
print("mất", len(mat), mat[:10]); print("đổi", len(doi), doi[:10]); print("mới", len(moi), [k[1] for k in moi][:20])
print("fail trong bản mới:", sorted(k for k, v in b.items() if v == "fail")[:20])
