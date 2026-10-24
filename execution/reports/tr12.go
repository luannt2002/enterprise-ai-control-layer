package reports

import (
	"fmt"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// dupHit là kết quả của TR1 hoặc TR2: lần giao trùng một bản ghi đã có. Chỉ ghi ở sổ lần giao, không bao giờ thành bản ghi.
type dupHit struct {
	reason Reason
	ref    string
}

// hasContent: một phiên bản BẤT KỲ của khoá (kể cả bản đã bị thay) có cùng nội dung chuẩn hoá (§5.1 TR1).
func (c *opctx) hasContent(key, hash string) (bool, error) {
	vs, err := c.versionsOf(key)
	if err != nil {
		return false, err
	}
	for _, v := range vs {
		if v.ContentHash == hash {
			return true, nil
		}
	}
	return false, nil
}

// tr12 áp TR1 rồi TR2 cho đơn vị thường (§3.8 bước 3, 4). Trả hit != nil khi lần giao là DUPLICATE. Khi cùng khoá mà khác nội dung với mọi
// phiên bản (id_conflict, §2.4), nó ĐỔI it: khoá có hậu tố băm và đề xuất HELD id_conflict, rồi trả nil để đi tiếp như một đơn vị mới.
func (c *opctx) tr12(it *Item, hash, fp string) (*dupHit, error) {
	base, err := c.get(it.Key)
	if err != nil {
		return nil, err
	}
	if base != nil {
		same, err := c.hasContent(base.Key, hash)
		if err != nil {
			return nil, err
		}
		if same {
			return &dupHit{reason: ReasonDuplicateKey, ref: base.Key}, nil
		}
		return c.idConflict(it, base.Key, hash)
	}
	other, err := c.tx.findByFingerprint(fp)
	if err != nil {
		return nil, err
	}
	if other != "" {
		return &dupHit{reason: ReasonDuplicateFingerprint, ref: other}, nil
	}
	return nil, nil
}

// idConflict cấp khoá có hậu tố băm cho bản đến sau (§2.4): "<khoá>~<8 ký tự đầu của băm nội dung chuẩn hoá>". Giao lại đúng nội dung đó
// thì gặp chính khoá này và thành DUPLICATE. Hậu tố dài dần (8, 16, 32, 64) nếu khoá đã bị một nội dung khác chiếm: không bao giờ hai nội
// dung khác nhau chung khoá.
func (c *opctx) idConflict(it *Item, baseKey, hash string) (*dupHit, error) {
	for n := 8; n <= len(hash); n *= 2 {
		cand := it.Key + "~" + hash[:n]
		r, err := c.get(cand)
		if err != nil {
			return nil, err
		}
		if r == nil {
			it.Key = cand
			it.Verdict = Verdict{
				Outcome: OutcomeHeld, Reason: ReasonIDConflict,
				Detail: fmt.Sprintf("cùng mã nguồn %s nhưng khác nội dung với mọi phiên bản (§2.4); bản này mang khoá có hậu tố băm %s", baseKey, hash[:n]),
			}
			return nil, nil
		}
		same, err := c.hasContent(cand, hash)
		if err != nil {
			return nil, err
		}
		if same {
			return &dupHit{reason: ReasonDuplicateKey, ref: cand}, nil
		}
	}
	return nil, contracts.Label(contracts.FaultData, fmt.Errorf("không cấp được khoá có hậu tố băm cho %q: mọi độ dài đã bị chiếm", it.Key))
}
