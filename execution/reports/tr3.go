package reports

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Bước 3b: TR3, nghi trùng trong cửa sổ W (HOP-DONG §5.1, §5.6).
//
// settle là MỘT bước duy nhất chạy ở cuối MỌI thao tác làm đổi kho (nạp, quyết định, thu hồi, sửa tin), trong cùng giao dịch, dưới khoá của các ô bị chạm. Nó không biết
// sự kiện nào vừa xảy ra: nó chỉ nhìn các ô (thực thể, ngày xảy ra) đã chạm và đưa chúng về trạng thái giữ ba điều (điểm bất động; mọi bước chỉ SIẾT, không bao giờ nới):
//
//	(1) hai bản liền kề trong nhóm, cách ≤ W, mà người chưa xét cặp thì bản đứng sau không ACCEPTED (§5.6 mục 1, 3, 4 (b));
//	(2) bản HELD possible_duplicate luôn trỏ đúng bản đứng ngay trước nó trong nhóm, hoặc rỗng kèm chi tiết khi không còn (§5.6 mục 1, 4 (a));
//	(3) bản REJECTED bởi MARK_DUPLICATE mà bản được giữ của nó thôi được tính, vì BẤT KỲ lý do gì, thì quay lại HELD possible_duplicate (§5.6 mục 4 (c); R6-01).
//
// "Được tính" của bản được giữ k đối với bản bị bác b: k đang ở đúng nhóm của b (ACCEPTED hoặc HELD possible_duplicate), hoặc k cũng đang REJECTED bởi MARK_DUPLICATE (khi
// đó k đã đi theo bản được giữ của nó; nếu bản đó rời nhóm thì chính k quay lại HELD và từ đó k được tính). Mọi trạng thái khác của k (RETRACTED, SUPERSEDED, đổi nhóm, HELD vì
// lý do khác, QUARANTINED, mất bản ghi) là "thôi được tính". Vì kiểm theo trạng thái chứ không theo đường đi, một đường rời nhóm mới thêm sau này không phải nhớ gọi thêm việc nào.

const maxSettleRounds = 64

// settle chạy tới điểm bất động trên các ô đã chạm. Mỗi vòng chỉ có thể: ACCEPTED hoặc REJECTED thành HELD, hoặc nối lại duplicate_of; không có bước ngược lại, nên dừng.
func (c *opctx) settle() error {
	for round := 0; round < maxSettleRounds; round++ {
		slots := c.sortedSlots()
		if err := c.ensureLocks(slots); err != nil { // mọi ô sẽ đọc được khoá một lượt, theo thứ tự khoá tăng dần
			return err
		}
		changed := false
		for _, s := range slots {
			recs, err := c.slotRecs(s)
			if err != nil {
				return err
			}
			ch, err := c.settleSlot(recs)
			if err != nil {
				return err
			}
			changed = changed || ch
		}
		if !changed {
			return nil
		}
	}
	return contracts.Errorf(contracts.FaultCode, "TR3 không tới được điểm bất động sau "+strconv.Itoa(maxSettleRounds)+" vòng (mọi bước chỉ siết nên đáng lẽ phải dừng)")
}

// settleSlot đưa các bản ghi của MỘT ô về điểm bất động (một vòng); trả true nếu có bản đổi.
func (c *opctx) settleSlot(recs []*Record) (bool, error) {
	changed := false
	// (3) bản bị bác mà bản được giữ thôi được tính
	for _, b := range recs {
		if b.Disposition != DispRejected || b.Decision != DecMarkDuplicate {
			continue
		}
		kept, err := c.keptOf(b, recs)
		if err != nil {
			return false, err
		}
		if keptCounts(b, kept) {
			continue
		}
		c.returnToHeld(b)
		changed = true
	}
	// (1) và (2) theo từng nhóm, theo thứ tự chuẩn (sent_at, unit_key)
	w := c.opt.window()
	groups := map[groupKey][]*Record{}
	var order []groupKey
	for _, r := range recs { // recs đã sắp theo unit_key nên thứ tự các nhóm tất định
		if !inGroup(r) {
			continue
		}
		g := groupOf(r)
		if _, ok := groups[g]; !ok {
			order = append(order, g)
		}
		groups[g] = append(groups[g], r)
	}
	for _, g := range order {
		list := groups[g]
		sort.Slice(list, func(i, j int) bool { return before(list[i], list[j]) })
		for i, r := range list {
			var p *Record
			if i > 0 {
				p = list[i-1]
			}
			if r.Disposition == DispAccepted {
				if p != nil && r.SentAt.Sub(p.SentAt) <= w && !r.reviewedWith(p.Key) {
					c.hold(r, p, w)
					changed = true
				}
				continue
			}
			want := ""
			if p != nil {
				want = p.Key
			}
			if r.DuplicateOf != want {
				c.relink(r, p, w)
				changed = true
			}
		}
	}
	return changed, nil
}

// keptOf tìm bản được giữ của b: trong ô trước (bản mới nhất), không có thì đọc theo khoá (bản đã chuyển ô).
func (c *opctx) keptOf(b *Record, recs []*Record) (*Record, error) {
	for _, r := range recs {
		if r.Key == b.DuplicateOf {
			return r, nil
		}
	}
	if b.DuplicateOf == "" {
		return nil, nil
	}
	return c.get(b.DuplicateOf)
}

func keptCounts(b, k *Record) bool {
	if k == nil {
		return false
	}
	if k.Disposition == DispRejected && k.Decision == DecMarkDuplicate {
		return true
	}
	return inGroup(k) && groupOf(k) == groupOf(b)
}

// returnToHeld đưa bản bị bác về hàng chờ: HELD possible_duplicate, có người nhận; quyết định MARK_DUPLICATE cũ còn ở bản ghi và lịch sử (BB7 ngoại lệ có tên). duplicate_of giữ tạm
// giá trị cũ (bản được giữ đã rời nhóm) để vòng nhóm ngay sau nối lại sang bản đứng ngay trước, hoặc để trống kèm chi tiết.
func (c *opctx) returnToHeld(b *Record) {
	from := b.Disposition
	c.touch(b)
	b.Disposition, b.Reason = DispHeld, ReasonPossibleDuplicate
	b.Detail = "quay lại hàng chờ: bản được giữ " + b.DuplicateOf + " đã rời nhóm (" + c.leftWhy(b.DuplicateOf, b) + "); quyết định MARK_DUPLICATE cũ còn ở lịch sử"
	c.finishVerdict(b)
	c.returned[b.Key] = true
	c.commit(b)
	c.hist(b, EventReturnedToHeld, from, ActorTR3, b.Detail+c.causeSuffix(), "")
}

// hold chuyển bản ACCEPTED thành HELD possible_duplicate trỏ bản đứng ngay trước (§5.6 mục 1: đánh giá lại chỉ siết).
func (c *opctx) hold(r, p *Record, w time.Duration) {
	from := r.Disposition
	c.touch(r)
	r.Disposition, r.Reason, r.DuplicateOf = DispHeld, ReasonPossibleDuplicate, p.Key
	r.Detail = fmt.Sprintf("nghi trùng bản %s: cùng thực thể, ngày xảy ra, giá trị, đơn vị và loại, gửi cách %s (cửa sổ W = %s); chờ người quyết KEEP_BOTH hoặc MARK_DUPLICATE",
		p.Key, durationText(r.SentAt.Sub(p.SentAt)), durationText(w))
	r.Recipient = c.recipientFor(r.Entity)
	c.commit(r)
	c.hist(r, EventTR3Hold, from, ActorTR3, r.Detail+c.causeSuffix(), "")
}

// relink nối lại duplicate_of của bản HELD possible_duplicate sang bản đứng ngay trước mới (p), hoặc để trống kèm chi tiết khi không còn. Bản vẫn HELD: máy không nhả.
func (c *opctx) relink(r, p *Record, w time.Duration) {
	old := r.DuplicateOf
	c.touch(r)
	if p == nil {
		r.DuplicateOf = ""
		r.Detail = fmt.Sprintf("bản đứng trước %s đã rời nhóm: %s; không còn bản nào để so, bản vẫn chờ người quyết (KEEP_BOTH), máy không tự nhận", old, c.leftWhy(old, r))
	} else {
		gap := r.SentAt.Sub(p.SentAt)
		r.DuplicateOf = p.Key
		r.Detail = fmt.Sprintf("nghi trùng bản %s (cách %s); bản đứng trước cũ %s đã rời nhóm hoặc đổi vị trí: %s", p.Key, durationText(gap), old, c.leftWhy(old, r))
		if gap > w {
			r.Detail += fmt.Sprintf("; cách hơn cửa sổ W = %s nhưng bản vẫn chờ người (chỉ siết, không tự nới)", durationText(w))
		}
	}
	c.commit(r)
	if !c.returned[r.Key] {
		c.hist(r, EventTR3Relink, r.Disposition, ActorTR3, r.Detail+c.causeSuffix(), "")
	}
}

// leftWhy mô tả vì sao bản oldKey không còn là bản đứng trước (đọc từ trạng thái hiện tại của nó); r là bản đang so.
func (c *opctx) leftWhy(oldKey string, r *Record) string {
	o, _ := c.get(oldKey)
	switch {
	case o == nil:
		return "không còn bản ghi"
	case o.Disposition == DispRetracted:
		return "đã thu hồi"
	case o.Disposition == DispSuperseded:
		return "bị thay bởi " + o.SupersededBy
	case o.Disposition == DispRejected:
		return "đã bị bác"
	case o.Disposition == DispQuarantined:
		return "chuyển sang cách ly"
	case o.Disposition == DispHeld && o.Reason != ReasonPossibleDuplicate:
		return "chuyển sang chờ vì " + string(o.Reason)
	case !inGroup(o) || groupOf(o) != groupOf(r):
		return "đổi nhóm (thực thể, ngày xảy ra, giá trị, đơn vị hoặc loại)"
	default:
		return "đổi vị trí trong nhóm (giờ gửi)"
	}
}

func (c *opctx) causeSuffix() string {
	if c.cause == "" {
		return ""
	}
	return "; do: " + c.cause
}
