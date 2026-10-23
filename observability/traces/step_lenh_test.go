package traces_test

import (
	"testing"

	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
)

// U9 (1): ValidStage nhận chín bước của luồng lệnh (C0…C7 và C4b, HOP-DONG §7.5) mà không làm hỏng chỗ nào khác: mã bước của luồng hỏi, của tầng nạp
// và mọi dạng gần giống mã lệnh (sai số, sai chữ, thừa ký tự) vẫn bị từ chối.
func TestValidStageLenh(t *testing.T) {
	for stage, want := range map[string]bool{
		"C0": true, "C1": true, "C2": true, "C3": true, "C4": true, "C4b": true, "C5": true, "C6": true, "C7": true,
		"C8": false, "C9": false, "C": false, "C04": false, "C10": false, "c0": false, "C4B": false, "C4c": false, "C-1": false,
		"C 4": false, "C4bb": false, "C0b": false, "C3b": false, "C5b": false, "CC4": false, "C٤": false,
		// luồng hỏi và tầng nạp giữ nguyên
		"S0": true, "S10": true, "S11": false, "S?": true, "L1.read": true, "L1.": false, "": false,
	} {
		if got := traces.ValidStage(stage); got != want {
			t.Errorf("ValidStage(%q) = %v, muốn %v", stage, got, want)
		}
	}
}

// Một bản ghi có chín bước lệnh qua Record.Validate (nơi kho vết kiểm trước khi lưu); mã bước lạ vẫn bị từ chối.
func TestRecordLenhValidate(t *testing.T) {
	stages := []string{"C0", "C1", "C2", "C3", "C4", "C4b", "C5", "C6", "C7"}
	rec := traces.Record{Decision: traces.Decision{TraceID: "t-1"}}
	for i, s := range stages {
		rec.Steps = append(rec.Steps, traces.Step{TraceID: "t-1", Seq: i, Stage: s, Name: "x", Status: traces.StatusOK})
	}
	if err := rec.Validate(); err != nil {
		t.Fatalf("bản ghi có chín bước lệnh phải hợp lệ: %v", err)
	}
	rec.Steps[3].Stage = "C8"
	if err := rec.Validate(); err == nil {
		t.Fatal("mã bước C8 không tồn tại mà bản ghi vẫn hợp lệ")
	}
}
