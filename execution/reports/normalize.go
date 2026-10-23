package reports

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/unicode/norm"
)

// NormalizeText là chuẩn hoá chữ để băm (§5.2): NFC, bỏ khoảng trắng đầu cuối, gộp khoảng trắng liên tiếp. Cố ý KHÔNG bỏ dấu và KHÔNG
// đổi hoa thường: "máy 03" và "may 03" là hai nội dung; chuẩn hoá mạnh hơn sẽ gộp oan (§5.3). Khoảng trắng theo unicode.IsSpace (gồm
// xuống dòng, tab, NBSP). UTF-8 hỏng không làm panic: byte hỏng đi qua nguyên vẹn nên hai lần chạy cho cùng kết quả.
func NormalizeText(s string) string {
	return strings.Join(strings.Fields(norm.NFC.String(s)), " ")
}

func hexSum(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		// mã hoá có độ dài đứng trước: ("ab","c") và ("a","bc") không bao giờ cùng một chuỗi băm
		h.Write([]byte(strconv.Itoa(len(p))))
		h.Write([]byte{':'})
		h.Write([]byte(p))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// contentHashOf là băm nội dung chuẩn hoá của một phiên bản: TR1 so nó với MỌI phiên bản đã có của cùng unit_key.
func contentHashOf(raw string) string {
	sum := sha256.Sum256([]byte(NormalizeText(raw)))
	return hex.EncodeToString(sum[:])
}

// fingerprintOf là vân tay TR2 (§5.1). Không bao giờ gồm unit_key hay đường dẫn tệp: tệp xuất chồng lấp có mã tin khác, tệp chép sang đường
// dẫn khác có khoá khác, và vân tay chính là thứ để bắt hai trường hợp đó.
//   - (a) đơn vị có sent_at (tin): (tenant, kênh, người gửi, sent_at đến giây, chữ chuẩn hoá);
//   - (b) đơn vị không có sent_at (dòng PDF, hàng bảng): (tenant, loại nguồn, ngày xảy ra, mã nguồn trong tệp, chữ chuẩn hoá). Có ngày
//     xảy ra để cùng một dòng chữ ở hai ngày khác nhau vẫn là hai đơn vị; có mã nguồn trong tệp để hai dòng cùng chữ trong một tệp vẫn là
//     hai đơn vị (không khử trùng theo giá trị, §5.3).
//
// Hai vế mang nhãn miền riêng ("tr2a", "tr2b"): một tin và một dòng không bao giờ cùng vân tay.
func fingerprintOf(tenant string, it Item) string {
	text := NormalizeText(it.Raw)
	if !it.SentAt.IsZero() {
		return hexSum("tr2a", tenant, it.Channel, it.SenderID, it.SentAt.UTC().Truncate(time.Second).Format(time.RFC3339), text)
	}
	return hexSum("tr2b", tenant, it.SourceType, it.OccurredOn, it.SourceSeq, text)
}

// MakeUnitKey dựng unit_key tất định của một đơn vị (§2.4): "<nguồn>#<mã nguồn>" khi có mã nguồn (record_id, mã tin, số dòng PDF), ngược
// lại "<nguồn>#L<thứ tự dòng>". origin là kênh hoặc đường dẫn tệp tương đối. Hai đơn vị khác nội dung mà vẫn chung khoá (cùng mã nguồn) thì
// kho xử lý như id_conflict (TR1), không bao giờ gộp.
func MakeUnitKey(origin, sourceID string, line int) string {
	if sourceID != "" {
		return origin + "#" + sourceID
	}
	return origin + "#L" + strconv.Itoa(line)
}
