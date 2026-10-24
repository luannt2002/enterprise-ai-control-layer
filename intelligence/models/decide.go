package models

import (
	"context"
	"fmt"
	"maps"
	"math"
	"net/http"
	"slices"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// ProtocolDecide: model quyết định có kiểu (không sinh chữ).
// POST {base_url} {model, state, questions{tên: {type: "choice", instructions, criteria{nhãn: mô tả}} | {type: "noul", instructions}}}
// → {model, answers{tên: {probabilities{nhãn: p}} | {noul: p}}}. base_url là URL ĐẦY ĐỦ của điểm cuối (đường dẫn mỗi nơi một khác).
// Dữ liệu ra ngoài: state và câu hỏi của mỗi lần gọi. Đầu ra là ĐỀ XUẤT có xác suất (P3): xác suất không phải chắc chắn, ngưỡng phải
// chọn trên bộ có nhãn của chính mình; cùng đầu vào gọi lại có thể lệch vài phần trăm (không tất định).
const ProtocolDecide = "typed_questions"

// decideTimeout: một quyết định nằm trên đường hỏi (S4) — quá hạn thì luồng giữ kết quả của luật, không chờ như sinh câu trả lời.
const decideTimeout = 6 * time.Second

// Decide gọi model quyết định có kiểu.
type Decide struct {
	caller
	URL, APIKey, Model string
}

// NewDecider dựng client theo cấu hình EAIS_DECIDE_*. Model nên ghim mã phiên bản, không dùng bí danh tự chuyển phiên bản.
func NewDecider(c ServiceConfig) (*Decide, error) {
	url, err := c.check("DECIDE", ProtocolDecide)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: decideTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return &Decide{caller: caller{Provider: c.Provider, MaxRetries: c.MaxRetries, Client: client}, URL: url, APIKey: c.APIKey, Model: c.Model}, nil
}

// Choice trả xác suất từng nhãn của MỘT câu hỏi chọn một trên state, và mã model nhà cung cấp báo (ghi vết).
func (d *Decide) Choice(ctx context.Context, state any, instructions string, labels map[string]string) (map[string]float64, string, error) {
	var resp decideResp
	q := map[string]any{"q": map[string]any{"type": "choice", "instructions": instructions, "criteria": labels}}
	if err := d.post(ctx, d.URL, http.Header{"Authorization": {"Bearer " + d.APIKey}}, map[string]any{"model": d.Model, "state": state, "questions": q}, &resp); err != nil {
		return nil, "", err
	}
	p := resp.Answers["q"].Probabilities
	sum := 0.0
	for _, v := range p {
		sum += v
	}
	if !slices.Equal(slices.Sorted(maps.Keys(p)), slices.Sorted(maps.Keys(labels))) || !okProb(p) || math.Abs(sum-1) > 0.01 {
		return nil, "", d.bad(fmt.Sprintf("xác suất của câu chọn một không khớp %d nhãn đã gửi hoặc không cộng thành 1", len(labels)))
	}
	return p, resp.Model, nil
}

// Propositions trả xác suất "đúng" của từng mệnh đề (có / không) trên CÙNG state trong một lần gọi; các mệnh đề được chấm độc lập nhau.
func (d *Decide) Propositions(ctx context.Context, state any, props map[string]string) (map[string]float64, string, error) {
	var resp decideResp
	q := map[string]any{}
	for name, text := range props {
		q[name] = map[string]any{"type": "noul", "instructions": text}
	}
	if err := d.post(ctx, d.URL, http.Header{"Authorization": {"Bearer " + d.APIKey}}, map[string]any{"model": d.Model, "state": state, "questions": q}, &resp); err != nil {
		return nil, "", err
	}
	out := map[string]float64{}
	for name := range props {
		a, ok := resp.Answers[name]
		if !ok || a.Noul == nil {
			return nil, "", d.bad("thiếu xác suất của mệnh đề " + name)
		}
		out[name] = *a.Noul
	}
	if !okProb(out) {
		return nil, "", d.bad("xác suất mệnh đề ngoài [0, 1]")
	}
	return out, resp.Model, nil
}

type decideResp struct {
	Model   string `json:"model"`
	Answers map[string]struct {
		Probabilities map[string]float64 `json:"probabilities"`
		Noul          *float64           `json:"noul"`
	} `json:"answers"`
}

func okProb(p map[string]float64) bool {
	for _, v := range p {
		if math.IsNaN(v) || v < 0 || v > 1 {
			return false
		}
	}
	return true
}

func (d *Decide) bad(msg string) error {
	return contracts.Errorf(contracts.ThirdParty(d.Provider, contracts.KindBadResponse), msg)
}
