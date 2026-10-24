package models

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// caller gửi POST JSON tới API của một nhà cung cấp và thử lại khi lỗi tạm — dùng chung cho Chat, Embeddings, Rerank.
// Messages giữ vòng thử lại riêng của nó: đổi nó là đổi điều kiện benchmark G7.
type caller struct {
	Provider string // nhãn nhà cung cấp trong nhãn lỗi third_party:<Provider>
	// MaxRetries: số lần THỬ LẠI khi lỗi tạm (429, 5xx, quá giờ, không tới được); 0 = không thử lại. 4xx khác không thử lại.
	MaxRetries int
	// Backoff: chờ trước lần thử lại thứ i là Backoff × 2^(i−1) (0 = 1 giây), hoặc lâu hơn nếu nhà cung cấp bảo (header Retry-After,
	// hoặc trường retryDelay trong thân lỗi) — hạn mức theo phút chỉ hồi khi đã qua phút đó.
	Backoff time.Duration
	Client  *http.Client // nil = timeout 60 giây, không theo chuyển hướng (khoá nằm ở header)
	sleep   func(ctx context.Context, d time.Duration) error
}

// maxProviderWait là trần thời gian chờ theo lời nhà cung cấp; maxResponse là trần thân phản hồi (vector của cả lô).
const (
	maxProviderWait = 75 * time.Second
	maxResponse     = 32 << 20
)

var retryDelayRe = regexp.MustCompile(`"retryDelay"\s*:\s*"(\d+(?:\.\d+)?)s"`)

// post gửi in (JSON) tới url kèm header, đọc phản hồi 200 vào out. Lỗi mang nhãn: lỗi tạm third_party:<nhà cung cấp>:<loại>,
// phản hồi không đọc được bad_response, 4xx khác là lỗi cấu hình của ta. Thông báo lỗi không chứa URL đầy đủ, header hay thân.
func (c *caller) post(ctx context.Context, url string, header http.Header, in, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return contracts.Label(contracts.FaultCode, err)
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	for attempt := 1; ; attempt++ {
		raw, hint, retry, err := c.once(ctx, client, url, header, body)
		if err == nil {
			if err := json.Unmarshal(raw, out); err != nil {
				return contracts.Label(contracts.ThirdParty(c.Provider, contracts.KindBadResponse), err)
			}
			return nil
		}
		if !retry || attempt > c.MaxRetries {
			return err
		}
		base := c.Backoff
		if base <= 0 {
			base = time.Second
		}
		sleep := c.sleep
		if sleep == nil {
			sleep = sleepCtx
		}
		if serr := sleep(ctx, max(backoffFor(base, attempt), min(hint, maxProviderWait))); serr != nil {
			return contracts.Label(contracts.ThirdParty(c.Provider, contracts.KindTimeout), fmt.Errorf("%w (lỗi lần gửi cuối: %v)", serr, err))
		}
	}
}

// once gửi MỘT lần; hint là thời gian nhà cung cấp bảo chờ (0 = không nói).
func (c *caller) once(ctx context.Context, client *http.Client, url string, header http.Header, body []byte) (raw []byte, hint time.Duration, retry bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		redactURL(err)
		return nil, 0, false, contracts.Label(contracts.FaultCode, err)
	}
	req.Header = header.Clone()
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		redactURL(err)
		kind := contracts.KindUnreachable
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "Client.Timeout") {
			kind = contracts.KindTimeout
		}
		return nil, 0, true, contracts.Label(contracts.ThirdParty(c.Provider, kind), err)
	}
	defer resp.Body.Close()
	raw, err = io.ReadAll(io.LimitReader(resp.Body, maxResponse))
	if err != nil {
		return nil, 0, true, contracts.Label(contracts.ThirdParty(c.Provider, contracts.KindUnreachable), err)
	}
	if s, err := strconv.ParseFloat(resp.Header.Get("Retry-After"), 64); err == nil && s > 0 {
		hint = time.Duration(s * float64(time.Second))
	} else if m := retryDelayRe.FindSubmatch(raw); m != nil {
		s, _ := strconv.ParseFloat(string(m[1]), 64)
		hint = time.Duration(s * float64(time.Second))
	}
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, hint, true, contracts.Errorf(contracts.ThirdParty(c.Provider, contracts.KindRateLimit), c.Provider+": hết hạn mức (429)")
	case resp.StatusCode >= 500:
		return nil, hint, true, contracts.Errorf(contracts.ThirdParty(c.Provider, contracts.KindUnavailable), fmt.Sprintf("%s: lỗi máy chủ %d", c.Provider, resp.StatusCode))
	case resp.StatusCode != http.StatusOK:
		return nil, 0, false, contracts.Errorf(contracts.FaultCode, fmt.Sprintf("%s: yêu cầu bị từ chối %d", c.Provider, resp.StatusCode))
	}
	return raw, 0, false, nil
}
