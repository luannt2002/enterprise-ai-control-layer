// Package models là CỬA DUY NHẤT tới model (CLAUDE.md §30): pipeline không gọi thẳng provider nào.
// Đầu ra của model là ĐỀ XUẤT — luôn qua cổng kiểm S9 trước khi giao.
package models

import (
	"context"
	"fmt"
	"sync"

	"github.com/luannt2002/enterprise-ai-systems/core/contracts"
)

// Request là đầu vào một lượt sinh: câu hỏi + bằng chứng ĐÃ được hệ thống chọn và lọc quyền.
type Request struct {
	Question string
	Evidence []contracts.Evidence
}

// Gateway sinh một bản nháp từ bằng chứng.
type Gateway interface {
	Generate(ctx context.Context, req Request) (contracts.Draft, error)
}

// Extractive là "model" TẤT ĐỊNH, không sáng tác: trả nguyên văn đoạn bằng chứng đứng đầu (truy
// xuất đã xếp theo mức liên quan). Dùng cho cổng G4/G5 (đo được, lặp lại được) và khi tenant chưa
// cấu hình model thật. Đếm số lần được gọi để cổng G4 chứng minh route tất định không gọi model.
type Extractive struct {
	mu    sync.Mutex
	calls int
}

// Generate trả đoạn bằng chứng đầu tiên, kèm claim trỏ về đúng bằng chứng đó; không có bằng
// chứng thì trả bản nháp rỗng (không claim — cổng kiểm sẽ chặn).
func (m *Extractive) Generate(_ context.Context, req Request) (contracts.Draft, error) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	if len(req.Evidence) == 0 {
		return contracts.Draft{Model: "extractive"}, nil
	}
	e := req.Evidence[0]
	return contracts.Draft{Text: e.Text, Model: "extractive", Claims: []contracts.Claim{{Text: e.Text, EvidenceIDs: []string{e.ID}}}}, nil
}

// Calls trả số lần đã gọi kể từ lần Reset gần nhất.
func (m *Extractive) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

// Reset đặt lại bộ đếm.
func (m *Extractive) Reset() {
	m.mu.Lock()
	m.calls = 0
	m.mu.Unlock()
}

// FromConfig chọn gateway theo cấu hình (không hard code provider). Provider chưa hỗ trợ là lỗi
// cấu hình — chặn lúc khởi động, không âm thầm rơi về model khác.
func FromConfig(provider string) (Gateway, error) {
	switch provider {
	case "", "extractive":
		return &Extractive{}, nil
	}
	return nil, contracts.Errorf(contracts.FaultCode, fmt.Sprintf("provider model %q chưa được hỗ trợ", provider))
}
