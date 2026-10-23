package reports

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Lỗi có tên: bên gọi phân loại bằng errors.Is. Lỗi của bên gọi mang nhãn contracts.FaultInput; lỗi hạ tầng mang infra:postgres.
var (
	ErrNotFound           = errors.New("không có bản ghi với unit_key này")
	ErrInvalidItem        = errors.New("đơn vị đầu vào không hợp lệ")
	ErrDecisionNotAllowed = errors.New("quyết định không được phép cho bản ghi này")
	ErrMissingSeen        = errors.New("lệnh thiếu seen_duplicate_of")
	ErrNoCounterpart      = errors.New("không có bản kia để trùng")
	ErrStaleDecision      = errors.New("quyết định trên màn hình cũ")
)

func inputErr(sentinel error, format string, a ...any) error {
	return contracts.Label(contracts.FaultInput, fmt.Errorf("%w: %s", sentinel, fmt.Sprintf(format, a...)))
}

// StaleDecisionError là lỗi của lệnh KEEP_BOTH hoặc MARK_DUPLICATE mang duplicate_of khác duplicate_of hiện tại (§6.2): bản kia đã đổi,
// đã rời nhóm, hoặc có bản mới chen vào. Bản ghi KHÔNG đổi (không REJECTED, không thêm cặp đã xét); lệnh bị từ chối được ghi vào lịch
// sử bản ghi làm audit, kèm hai giá trị để người quyết làm mới màn hình rồi quyết lại.
type StaleDecisionError struct{ Key, Seen, Current string }

func (e *StaleDecisionError) Error() string {
	return fmt.Sprintf("%v: bản %s: người quyết thấy duplicate_of = %q, hiện tại là %q", ErrStaleDecision, e.Key, e.Seen, e.Current)
}
func (e *StaleDecisionError) Is(target error) bool { return target == ErrStaleDecision }
func (e *StaleDecisionError) keepAudit() bool      { return true }

// Options là cấu hình của kho. Giá trị 0 dùng mặc định an toàn.
type Options struct {
	// Window là W của TR3 (§5.1): hai bản liền kề trong nhóm cách nhau ≤ W thì bản đứng sau bị giữ. Mặc định 60 phút (Q-0A-5, đề xuất).
	Window time.Duration
	// Now là đồng hồ của hệ (decided_at, lịch sử); mặc định time.Now. Trường của hệ không tất định (§3.7): test đặt đồng hồ cố định.
	Now func() time.Time
	// Recipient trả người nhận của một mục HELD hoặc QUARANTINED theo thực thể (§6.3); rỗng hoặc nil thì DefaultRecipient.
	Recipient func(entity string) string
}

func (o Options) window() time.Duration {
	if o.Window <= 0 {
		return 60 * time.Minute
	}
	return o.Window
}

func (o Options) now() time.Time {
	if o.Now == nil {
		return time.Now().UTC().Truncate(time.Microsecond)
	}
	return o.Now().UTC().Truncate(time.Microsecond)
}

// Command là một quyết định của người (§6.2). Gói không kiểm capability reports.decide hay phạm vi thực thể: đó là bước 6.
type Command struct {
	Key      string
	Decision Decision
	Actor    string
	Why      string
	// SeenDuplicateOf là duplicate_of của bản ghi trên màn hình lúc người quyết (KEEP_BOTH, MARK_DUPLICATE). nil là THIẾU: lệnh bị
	// từ chối (đóng mặc định). Con trỏ tới chuỗi rỗng là "màn hình ghi bản đứng trước đã rời nhóm": hợp lệ, khác thiếu (R4-03).
	SeenDuplicateOf *string
	Entity          string    // CHOOSE (bắt buộc); CORRECT (tuỳ chọn)
	Value           *Quantity // CORRECT (tuỳ chọn)
	OccurredOn      string    // CORRECT (tuỳ chọn)
	NewKind         Kind      // CORRECT (tuỳ chọn)
	Target          string    // APPLY: unit_key của bản gốc bị thay
}

// Receipt là biên nhận lô (§3.8): số dòng sổ của lô theo kết cục và theo mã lý do, kèm "trùng với bản nào" ở từng dòng.
type Receipt struct {
	BatchID   string
	Entries   []DeliveryEntry
	ByOutcome map[Outcome]int
	ByReason  map[Reason]int
}

// BuildReceipt dựng biên nhận từ các dòng sổ của một lô.
func BuildReceipt(batchID string, entries []DeliveryEntry) Receipt {
	r := Receipt{BatchID: batchID, Entries: entries, ByOutcome: map[Outcome]int{}, ByReason: map[Reason]int{}}
	for _, e := range entries {
		r.ByOutcome[e.Outcome]++
		if e.Reason != "" {
			r.ByReason[e.Reason]++
		}
	}
	return r
}

// Conserved kiểm BB1 cho một lô đã chạy tới cuối: manifest là các unit_key đã đưa vào, theo thứ tự. Số đơn vị vào = số dòng sổ = tổng
// các kết cục; dòng thứ i của sổ là đơn vị thứ i; mỗi dòng đúng một kết cục thuộc bốn giá trị; mỗi dòng DUPLICATE trỏ về một bản ghi.
func (r Receipt) Conserved(manifest []string) error {
	if len(r.Entries) != len(manifest) {
		return fmt.Errorf("BB1: %d đơn vị vào nhưng %d dòng sổ (lô %s)", len(manifest), len(r.Entries), r.BatchID)
	}
	sum := 0
	for _, n := range r.ByOutcome {
		sum += n
	}
	if sum != len(manifest) {
		return fmt.Errorf("BB1: tổng các kết cục %d khác số đơn vị vào %d", sum, len(manifest))
	}
	for i, e := range r.Entries {
		if e.Ordinal != i+1 || e.Key != manifest[i] {
			return fmt.Errorf("BB1: dòng sổ %d là (%d, %q), muốn (%d, %q)", i, e.Ordinal, e.Key, i+1, manifest[i])
		}
		switch e.Outcome {
		case OutcomeAccepted, OutcomeHeld, OutcomeQuarantined, OutcomeDuplicate:
		default:
			return fmt.Errorf("BB1: dòng %q có kết cục %q ngoài bốn giá trị", e.Key, e.Outcome)
		}
		if e.RecordRef == "" {
			return fmt.Errorf("BB1: dòng %q không trỏ về bản ghi nào", e.Key)
		}
	}
	return nil
}

// Store là giao diện công khai của kho bản ghi. Mọi lệnh lấy tenant từ ngữ cảnh (contracts.TenantFrom); không có tenant thì từ chối
// bằng lỗi chính sách, không chạm dữ liệu (BB8). Hai cài đặt: Memory (test) và PG.
type Store interface {
	// Deliver xử lý MỘT lần giao: TR1, TR2, rồi Verdict của luật đọc, rồi TR3 (thứ tự áp: HOP-DONG §3.8), trong MỘT giao dịch ghi bản
	// ghi (nếu có), phiên bản, lịch sử và đúng một dòng sổ. Trả dòng sổ.
	Deliver(ctx context.Context, batchID string, ordinal int, it Item) (DeliveryEntry, error)
	// DeliverBatch xử lý lần lượt các đơn vị của MỘT lô mới (mã lô cấp mới mỗi lần gọi, §3.8), mỗi đơn vị một giao dịch. Lỗi giữa chừng
	// trả biên nhận của các đơn vị đã xong cùng lỗi: lô dừng giữa chừng chỉ có dòng của đơn vị đã xử lý xong (§7.5).
	DeliverBatch(ctx context.Context, items []Item) (Receipt, error)
	// Decide áp một quyết định của người (§6.2). Lệnh KEEP_BOTH, MARK_DUPLICATE trên màn hình cũ trả *StaleDecisionError và vẫn ghi audit.
	Decide(ctx context.Context, cmd Command) (Record, error)
	// Retract đưa bản ghi sang RETRACTED. Nguyên thủy ÁP DỤNG của lớp phục hồi (bước 7): không kiểm người gửi, không tạo dấu thu hồi.
	Retract(ctx context.Context, key, actor, why string) error
	// Amend thêm một phiên bản nội dung mới cho bản ghi (version + 1). Nguyên thủy ÁP DỤNG của lớp phục hồi (bước 7); it.Key bị bỏ qua.
	Amend(ctx context.Context, key string, it Item, actor, why string) error

	Record(ctx context.Context, key string) (Record, bool, error)
	Records(ctx context.Context) ([]Record, error)                       // sắp theo unit_key (so byte)
	Ledger(ctx context.Context, batchID string) ([]DeliveryEntry, error) // batchID rỗng: mọi lô, sắp theo (lô, thứ tự)
	History(ctx context.Context, key string) ([]HistoryEntry, error)
	Receipt(ctx context.Context, batchID string) (Receipt, error)
	Totals(ctx context.Context) ([]Total, error)
}

func tenantFrom(ctx context.Context) (string, error) {
	t, ok := contracts.TenantFrom(ctx)
	if !ok {
		return "", contracts.Errorf(contracts.FaultPolicy, "truy vấn không mang tenant trong ngữ cảnh")
	}
	return string(t), nil
}

// sortRecords sắp theo unit_key so byte: cả hai kho trả cùng thứ tự.
func sortRecords(rs []Record) { sort.Slice(rs, func(i, j int) bool { return rs[i].Key < rs[j].Key }) }

func sortLedger(es []DeliveryEntry) {
	sort.Slice(es, func(i, j int) bool {
		if es[i].BatchID != es[j].BatchID {
			return es[i].BatchID < es[j].BatchID
		}
		return es[i].Ordinal < es[j].Ordinal
	})
}
