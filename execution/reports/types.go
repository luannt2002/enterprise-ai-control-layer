package reports

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-control-layer/core/decimal"
)

// Outcome là kết cục của MỘT LẦN GIAO (HOP-DONG §3.1). Bốn giá trị; DUPLICATE chỉ có ở sổ lần giao, không bao giờ là trạng thái của bản
// ghi.
type Outcome string

const (
	OutcomeAccepted    Outcome = "ACCEPTED"
	OutcomeHeld        Outcome = "HELD"
	OutcomeQuarantined Outcome = "QUARANTINED"
	OutcomeDuplicate   Outcome = "DUPLICATE"
)

// Disposition là trạng thái hiện hành của một BẢN GHI (§3.1): ba kết cục đầu cộng ba trạng thái vòng đời thêm sau quyết định hoặc
// đánh giá lại. Ba trạng thái vòng đời (SUPERSEDED, RETRACTED, REJECTED) không vào tổng và không vào nhóm nghi trùng.
type Disposition string

const (
	DispAccepted    Disposition = "ACCEPTED"
	DispHeld        Disposition = "HELD"
	DispQuarantined Disposition = "QUARANTINED"
	DispSuperseded  Disposition = "SUPERSEDED"
	DispRetracted   Disposition = "RETRACTED"
	DispRejected    Disposition = "REJECTED"
)

// Kind là loại tin (§3.3). Gói chỉ giữ và so sánh; việc gán loại từ chữ là bước 4.
type Kind string

const (
	KindOccurred   Kind = "OCCURRED"
	KindPlan       Kind = "PLAN"
	KindCorrection Kind = "CORRECTION"
)

// Reason là mã lý do thuộc danh sách ĐÓNG (§3.4). Mã nằm ngoài danh sách không bao giờ được ghi.
type Reason string

const (
	ReasonMissingValue           Reason = "missing_value"
	ReasonUnknownEntity          Reason = "unknown_entity"
	ReasonAmbiguousEntity        Reason = "ambiguous_entity"
	ReasonOutOfRange             Reason = "out_of_range"
	ReasonUnitDimension          Reason = "unit_dimension"
	ReasonUnparseable            Reason = "unparseable"
	ReasonMultipleItems          Reason = "multiple_items"
	ReasonNumberReadingUncertain Reason = "number_reading_uncertain"
	ReasonUnsupportedMedia       Reason = "unsupported_media"
	ReasonUnrecognizedLine       Reason = "unrecognized_line"
	ReasonUnreadableFile         Reason = "unreadable_file"
	ReasonUnknownSender          Reason = "unknown_sender"
	ReasonSenderNotAuthorized    Reason = "sender_not_authorized"
	ReasonDateConflict           Reason = "date_conflict"
	ReasonCorrectionUnconfirmed  Reason = "correction_unconfirmed"
	ReasonPossibleDuplicate      Reason = "possible_duplicate"
	ReasonAmbiguousKind          Reason = "ambiguous_kind"
	ReasonIDConflict             Reason = "id_conflict"
	ReasonDuplicateKey           Reason = "duplicate_key"
	ReasonDuplicateFingerprint   Reason = "duplicate_fingerprint"
)

// reasonOutcome là bảng §3.4: mỗi mã thuộc đúng một kết cục.
var reasonOutcome = map[Reason]Outcome{
	ReasonMissingValue:           OutcomeQuarantined,
	ReasonUnknownEntity:          OutcomeQuarantined,
	ReasonAmbiguousEntity:        OutcomeHeld,
	ReasonOutOfRange:             OutcomeQuarantined,
	ReasonUnitDimension:          OutcomeQuarantined,
	ReasonUnparseable:            OutcomeQuarantined,
	ReasonMultipleItems:          OutcomeQuarantined,
	ReasonNumberReadingUncertain: OutcomeHeld,
	ReasonUnsupportedMedia:       OutcomeQuarantined,
	ReasonUnrecognizedLine:       OutcomeQuarantined,
	ReasonUnreadableFile:         OutcomeQuarantined,
	ReasonUnknownSender:          OutcomeQuarantined,
	ReasonSenderNotAuthorized:    OutcomeHeld,
	ReasonDateConflict:           OutcomeHeld,
	ReasonCorrectionUnconfirmed:  OutcomeHeld,
	ReasonPossibleDuplicate:      OutcomeHeld,
	ReasonAmbiguousKind:          OutcomeHeld,
	ReasonIDConflict:             OutcomeHeld,
	ReasonDuplicateKey:           OutcomeDuplicate,
	ReasonDuplicateFingerprint:   OutcomeDuplicate,
}

// Outcome trả kết cục mà mã thuộc về; ok = false khi mã không thuộc danh sách đóng.
func (r Reason) Outcome() (Outcome, bool) {
	o, ok := reasonOutcome[r]
	return o, ok
}

// AllReasons trả toàn bộ mã lý do, sắp theo chữ.
func AllReasons() []Reason {
	out := make([]Reason, 0, len(reasonOutcome))
	for r := range reasonOutcome {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// ReasonsOf trả các mã lý do của một kết cục, sắp theo chữ.
func ReasonsOf(o Outcome) []Reason {
	var out []Reason
	for _, r := range AllReasons() {
		if reasonOutcome[r] == o {
			out = append(out, r)
		}
	}
	return out
}

// Decision là quyết định của người (§6.2). DISMISS và REPROCESS chưa có (xem doc.go).
type Decision string

const (
	DecKeepBoth      Decision = "KEEP_BOTH"
	DecMarkDuplicate Decision = "MARK_DUPLICATE"
	DecReject        Decision = "REJECT"
	DecChoose        Decision = "CHOOSE"
	DecCorrect       Decision = "CORRECT"
	DecAuthorize     Decision = "AUTHORIZE"
	DecApply         Decision = "APPLY"
)

// Tên sự kiện của lịch sử bản ghi (audit: ai, khi nào, vì sao).
const (
	EventDelivered      = "DELIVERED"        // bản ghi được tạo bởi một lần giao
	EventTR3Hold        = "TR3_HOLD"         // ACCEPTED thành HELD possible_duplicate (đánh giá lại chỉ siết)
	EventTR3Relink      = "TR3_RELINK"       // bản HELD possible_duplicate đổi duplicate_of sang bản đứng trước mới
	EventReturnedToHeld = "RETURNED_TO_HELD" // bản REJECTED bởi MARK_DUPLICATE quay lại hàng chờ (R5-01, R6-01)
	EventDecision       = "DECISION"         // quyết định của người
	EventRefused        = "DECISION_REFUSED" // lệnh bị từ chối (ví dụ màn hình cũ): bản ghi không đổi, audit vẫn ghi
	EventRetracted      = "RETRACTED"
	EventAmended        = "AMENDED"
	EventSuperseded     = "SUPERSEDED"
)

// DefaultRecipient là người nhận khi không xác định được máy (§6.3 mục 2).
const DefaultRecipient = "director"

// Quantity là lượng ở đơn vị chuẩn. Number là chuỗi thập phân chuẩn (core/decimal), rỗng khi chưa đọc được lượng.
type Quantity struct {
	Number string `json:"number"`
	Unit   string `json:"unit"`
}

// Attachment là tham chiếu tới ảnh hoặc tệp đính kèm: không đọc, chỉ giữ (§2.2, LC04, LC05).
type Attachment struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
}

// Verdict là kết cục mà LUẬT ĐỌC (chuẩn hoá, thực thể, loại tin, ngày, quyền, miền: bước 4 đến 6) đề xuất cho một đơn vị. Gói này
// không tự đọc chữ: nó nhận Verdict rồi áp chống trùng lên trên. Outcome chỉ là ACCEPTED, HELD hoặc QUARANTINED; Reason rỗng khi
// ACCEPTED, còn lại thuộc danh sách đóng và đúng kết cục. Ba mã do chống trùng sở hữu (possible_duplicate, duplicate_key,
// duplicate_fingerprint) không được đề xuất.
type Verdict struct {
	Outcome Outcome
	Reason  Reason
	Detail  string
}

// Item là MỘT đơn vị đầu vào ở một lần giao (§2.4). Không có trường tenant: tenant chỉ đến từ ngữ cảnh (BB8, LC07).
type Item struct {
	Key         string // unit_key tất định (§2.4), bắt buộc
	Channel     string
	SourceType  string
	SenderID    string
	SentAt      time.Time // zero: đơn vị không có sent_at (dòng PDF, hàng bảng): không vào nhóm TR3
	SourceSeq   string    // mã nguồn của đơn vị trong tệp (số dòng PDF, record_id): dùng cho vân tay TR2 của đơn vị không có sent_at
	OccurredOn  string    // YYYY-MM-DD, rỗng khi chưa đọc được
	Entity      string
	Metric      string
	Value       Quantity
	Approx      bool
	Kind        Kind
	Corrects    string
	Raw         string // nguyên văn, giữ từng byte (BB4)
	Attachments []Attachment
	Note        string
	Verdict     Verdict
	PackVersion string
	TraceID     string
}

// Record là bản ghi báo cáo hiện hành của một unit_key (§3.2). Raw không vào JSON của ảnh chụp phiên bản: nó nằm ở cột riêng để
// giữ từng byte (kể cả UTF-8 hỏng) mà JSON không giữ được.
type Record struct {
	Tenant        string       `json:"-"`
	Key           string       `json:"unit_key"`
	Channel       string       `json:"channel"`
	SourceType    string       `json:"source_type"`
	SenderID      string       `json:"sender_id"`
	SentAt        time.Time    `json:"sent_at"`
	OccurredOn    string       `json:"occurred_on"`
	Entity        string       `json:"entity"`
	Metric        string       `json:"metric"`
	Value         Quantity     `json:"value"`
	Approx        bool         `json:"approx"`
	Kind          Kind         `json:"kind"`
	Corrects      string       `json:"corrects"`
	Raw           string       `json:"-"`
	Attachments   []Attachment `json:"attachments"`
	Note          string       `json:"note"`
	Disposition   Disposition  `json:"disposition"`
	Reason        Reason       `json:"reason"`
	Detail        string       `json:"detail"`
	Recipient     string       `json:"recipient"`
	DuplicateOf   string       `json:"duplicate_of"`
	SupersededBy  string       `json:"superseded_by"`
	Version       int          `json:"version"`
	DecidedBy     string       `json:"decided_by"`
	DecidedAt     time.Time    `json:"decided_at"`
	Decision      Decision     `json:"decision"`
	ReviewedPairs []string     `json:"reviewed_pairs"` // unit_key của các bản kia mà người đã xét bằng KEEP_BOTH, cộng dồn (§5.6 mục 3)
	PackVersion   string       `json:"pack_version"`
	TraceID       string       `json:"trace_id"`
	Fingerprint   string       `json:"fingerprint"`    // vân tay TR2 của phiên bản hiện hành
	ContentHash   string       `json:"content_sha256"` // băm nội dung chuẩn hoá (§5.2) của phiên bản hiện hành, cho TR1
}

func (r *Record) clone() Record {
	c := *r
	c.Attachments = append([]Attachment(nil), r.Attachments...)
	c.ReviewedPairs = append([]string(nil), r.ReviewedPairs...)
	return c
}

func (r *Record) reviewedWith(key string) bool {
	for _, k := range r.ReviewedPairs {
		if k == key {
			return true
		}
	}
	return false
}

// DeliveryEntry là một dòng của SỔ LẦN GIAO (§3.8): mỗi đơn vị của mỗi lô đúng một dòng, kể cả ACCEPTED và DUPLICATE; chỉ thêm, không
// sửa. Outcome là kết cục LÚC XỬ LÝ; kết cục sau này đổi (quyết định, đánh giá lại TR3) thì ghi vào lịch sử bản ghi, không sửa sổ.
type DeliveryEntry struct {
	Tenant      string
	BatchID     string
	Ordinal     int
	Key         string // unit_key của đơn vị ĐÃ GIAO
	Outcome     Outcome
	Reason      Reason // rỗng khi ACCEPTED
	Detail      string
	RecordRef   string // unit_key của bản ghi mà lần giao này tạo hoặc trùng; khác Key khi id_conflict (khoá có hậu tố băm)
	PackVersion string
	TraceID     string
	RecordedAt  time.Time
}

// HistoryEntry là một dòng lịch sử của bản ghi: ai, khi nào, vì sao, từ trạng thái nào sang trạng thái nào. Chỉ thêm.
type HistoryEntry struct {
	Tenant      string
	Key         string
	Seq         int
	At          time.Time
	Actor       string
	Event       string
	From        Disposition
	To          Disposition
	Reason      Reason
	Detail      string
	Version     int
	DuplicateOf string
	Decision    Decision
	PackVersion string
}

// Total là một dòng của tổng đọc (§3.6): chỉ bản ghi ACCEPTED loại OCCURRED. Approx đúng khi có ít nhất một bản ước lượng.
type Total struct {
	Entity     string
	OccurredOn string
	Unit       string
	Number     string
	Count      int
	Approx     bool
}

// cleanText bỏ NUL và thay chuỗi UTF-8 hỏng bằng U+FFFD, cho các trường mà PostgreSQL lưu kiểu text (khoá, mã người gửi, ...). Raw
// KHÔNG qua đây: nó lưu kiểu bytea để giữ từng byte (BB4, LC08).
func cleanText(s string) string {
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "�")
	}
	if strings.IndexByte(s, 0) >= 0 {
		s = strings.ReplaceAll(s, "\x00", "")
	}
	return s
}

const dateLayout = "2006-01-02"

// sanitized trả bản đã chuẩn hoá của Item: trường text sạch, giờ gửi về UTC làm tròn xuống micro-giây (độ chính xác của PostgreSQL, để
// kho bộ nhớ và kho PG cho cùng kết quả), loại mặc định, lượng ở dạng chuẩn.
func (it Item) sanitized() Item {
	it.Key = cleanText(it.Key)
	it.Channel = cleanText(it.Channel)
	it.SourceType = cleanText(it.SourceType)
	it.SenderID = cleanText(it.SenderID)
	it.SourceSeq = cleanText(it.SourceSeq)
	it.OccurredOn = cleanText(it.OccurredOn)
	it.Entity = cleanText(it.Entity)
	it.Metric = cleanText(it.Metric)
	it.Corrects = cleanText(it.Corrects)
	it.Note = cleanText(it.Note)
	it.PackVersion = cleanText(it.PackVersion)
	it.TraceID = cleanText(it.TraceID)
	it.Verdict.Detail = cleanText(it.Verdict.Detail)
	it.Value.Unit = cleanText(it.Value.Unit)
	if c, ok := decimal.Canonical(it.Value.Number); ok {
		it.Value.Number = c
	}
	if !it.SentAt.IsZero() {
		it.SentAt = it.SentAt.UTC().Truncate(time.Microsecond)
	}
	if it.Kind == "" {
		it.Kind = KindOccurred
	}
	if len(it.Attachments) > 0 {
		at := make([]Attachment, len(it.Attachments))
		for i, a := range it.Attachments {
			at[i] = Attachment{Kind: cleanText(a.Kind), Ref: cleanText(a.Ref)}
		}
		it.Attachments = at
	}
	return it
}

// validate kiểm Item đã qua sanitized. Lỗi là lỗi của bên gọi (đầu vào), không bao giờ làm rơi đơn vị im lặng: bên gọi phải thấy lỗi.
func (it Item) validate() error {
	bad := func(format string, a ...any) error { return inputErr(ErrInvalidItem, format, a...) }
	if it.Key == "" {
		return bad("thiếu unit_key")
	}
	switch it.Kind {
	case KindOccurred, KindPlan, KindCorrection:
	default:
		return bad("loại tin %q ngoài {OCCURRED, PLAN, CORRECTION}", it.Kind)
	}
	if it.OccurredOn != "" {
		if _, err := time.Parse(dateLayout, it.OccurredOn); err != nil {
			return bad("ngày xảy ra %q không phải YYYY-MM-DD", it.OccurredOn)
		}
	}
	if it.Value.Number != "" {
		if _, ok := decimal.Canonical(it.Value.Number); !ok {
			return bad("lượng %q không phải số thập phân", it.Value.Number)
		}
	}
	v := it.Verdict
	switch v.Outcome {
	case OutcomeAccepted:
		if v.Reason != "" {
			return bad("kết cục ACCEPTED không mang mã lý do (%s)", v.Reason)
		}
		if it.Entity == "" || it.OccurredOn == "" || it.Value.Number == "" || it.Value.Unit == "" {
			return bad("đơn vị ACCEPTED phải có thực thể, ngày xảy ra, lượng và đơn vị (BB6: không nhận bản đoán)")
		}
	case OutcomeHeld, OutcomeQuarantined:
		o, ok := v.Reason.Outcome()
		if !ok || o != v.Outcome {
			return bad("mã lý do %q không thuộc kết cục %s trong danh sách đóng (§3.4)", v.Reason, v.Outcome)
		}
		switch v.Reason {
		case ReasonPossibleDuplicate:
			return bad("mã possible_duplicate do TR3 sở hữu, luật đọc không được đề xuất")
		}
	default:
		return bad("kết cục đề xuất %q ngoài {ACCEPTED, HELD, QUARANTINED}: DUPLICATE do chống trùng quyết, không do luật đọc", v.Outcome)
	}
	return nil
}

// durationText in khoảng cách giờ gửi cho dòng detail, dạng ổn định giữa hai lần chạy.
func durationText(d time.Duration) string {
	return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
}
