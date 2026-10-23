package reports

// KHUNG TẠM của commit đỏ-trước (bước 3, TR1 và TR2): chưa chống trùng. Bị thay bằng cài đặt thật ở commit kế tiếp.

// dupHit là kết quả của TR1 hoặc TR2: lần giao trùng bản đã có.
type dupHit struct {
	reason Reason
	ref    string
}

func contentHashOf(raw string) string                            { return "" }
func fingerprintOf(tenant string, it Item) string                { return "" }
func (c *opctx) tr12(it *Item, hash, fp string) (*dupHit, error) { return nil, nil }
