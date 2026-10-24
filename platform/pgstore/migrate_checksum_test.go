package pgstore

import "testing"

// Checksum của migration không đổi theo kiểu xuống dòng của bản checkout (CRLF trên Windows với core.autocrlf), nhưng đổi khi nội dung đổi.
func TestChecksumMigrationBoQuaCRLF(t *testing.T) {
	lf := checksum([]byte("CREATE TABLE a (x int);\nGRANT SELECT ON a TO eais_app;\n"))
	if crlf := checksum([]byte("CREATE TABLE a (x int);\r\nGRANT SELECT ON a TO eais_app;\r\n")); crlf != lf {
		t.Errorf("CRLF ra checksum khác LF: %s ≠ %s", crlf, lf)
	}
	if other := checksum([]byte("CREATE TABLE a (x bigint);\nGRANT SELECT ON a TO eais_app;\n")); other == lf {
		t.Error("nội dung khác mà cùng checksum")
	}
}
