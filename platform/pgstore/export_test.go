package pgstore

// LoadLockKey cho test ngoài gói (pgstore_test) mượn khoá tư vấn mà Load giữ theo tenant: TestLoadRecordsMoiNhatTheoThuTuCommit dựng tay một lần nạp
// bắt đầu sớm nhưng lấy khoá muộn, và phải lấy ĐÚNG khoá của Load. Tệp _test.go nên không vào binary.
var LoadLockKey = loadLockKey
