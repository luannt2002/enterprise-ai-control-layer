package models

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
)

// Giao thức của dịch vụ phụ cho truy hồi. Code chỉ biết giao thức; nhà cung cấp, endpoint, model, khoá đến từ cấu hình.
const (
	ProtocolEmbeddings = "embeddings" // POST {base}/embeddings {model, input[], dimensions} → data[].embedding
	ProtocolRerank     = "rerank"     // POST {base}/rerank {model, query, documents[], top_n} → results[]{index, relevance_score}
)

// ServiceConfig là cấu hình một dịch vụ phụ của truy hồi (embedding, rerank) — đọc ở tầng khởi động, không hard code.
type ServiceConfig struct {
	Provider   string // giao thức (Protocol*); "" = tắt
	BaseURL    string // bắt buộc — không có endpoint mặc định
	APIKey     string
	Model      string // bắt buộc — không có model mặc định
	Dim        int    // số chiều vector (chỉ embedding); bắt buộc vì vector đã lưu gắn với nó
	MaxRetries int
}

// ServiceFromEnv đọc EAIS_<role>_PROVIDER, _MODEL, _BASE_URL, _API_KEY, _DIM, _MAX_RETRIES (trống = 2); role là "EMBED" hoặc
// "RERANK". Số viết sai thành -1 để constructor báo lỗi cấu hình, không lặng lẽ về mặc định.
func ServiceFromEnv(role string) ServiceConfig {
	env := func(k string) string { return os.Getenv("EAIS_" + role + "_" + k) }
	num := func(k string, def int) int {
		v := env(k)
		if v == "" {
			return def
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return -1
		}
		return n
	}
	return ServiceConfig{Provider: env("PROVIDER"), BaseURL: env("BASE_URL"), APIKey: env("API_KEY"), Model: env("MODEL"),
		Dim: num("DIM", 0), MaxRetries: num("MAX_RETRIES", 2)}
}

// check kiểm cấu hình chung của role và trả BaseURL đã bỏ "/" cuối.
func (c ServiceConfig) check(role, protocol string) (string, error) {
	if c.Provider != protocol {
		return "", contracts.Errorf(contracts.FaultCode, fmt.Sprintf("EAIS_%s_PROVIDER %q chưa được hỗ trợ (có: %s)", role, c.Provider, protocol))
	}
	if c.Model == "" || c.APIKey == "" || c.BaseURL == "" {
		return "", contracts.Errorf(contracts.FaultCode, fmt.Sprintf("EAIS_%s_PROVIDER %s cần EAIS_%s_MODEL, EAIS_%s_API_KEY và EAIS_%s_BASE_URL", role, protocol, role, role, role))
	}
	if !secureEndpoint(c.BaseURL) {
		return "", contracts.Errorf(contracts.FaultCode, fmt.Sprintf("EAIS_%s_BASE_URL %q: khoá chỉ được gửi qua https, hoặc http tới loopback", role, EndpointForLog(c.BaseURL)))
	}
	if c.MaxRetries < 0 || c.MaxRetries > MaxRetriesLimit {
		return "", contracts.Errorf(contracts.FaultCode, fmt.Sprintf("EAIS_%s_MAX_RETRIES phải là số trong [0, %d]", role, MaxRetriesLimit))
	}
	return strings.TrimRight(c.BaseURL, "/"), nil
}

// NewEmbedder dựng embedder theo cấu hình. Provider trống là lỗi: gọi hàm này nghĩa là đã chọn bật embedding.
func NewEmbedder(c ServiceConfig) (*Embeddings, error) {
	base, err := c.check("EMBED", ProtocolEmbeddings)
	if err != nil {
		return nil, err
	}
	if c.Dim <= 0 {
		return nil, contracts.Errorf(contracts.FaultCode, "EAIS_EMBED_DIM phải là số nguyên dương")
	}
	return &Embeddings{caller: caller{Provider: c.Provider, MaxRetries: c.MaxRetries}, BaseURL: base, APIKey: c.APIKey, Model: c.Model, Dim: c.Dim}, nil
}

// NewReranker dựng reranker theo cấu hình.
func NewReranker(c ServiceConfig) (*Rerank, error) {
	base, err := c.check("RERANK", ProtocolRerank)
	if err != nil {
		return nil, err
	}
	return &Rerank{caller: caller{Provider: c.Provider, MaxRetries: c.MaxRetries}, BaseURL: base, APIKey: c.APIKey, Model: c.Model}, nil
}

// Embeddings nhúng chữ qua giao thức /embeddings. Dữ liệu ra ngoài: mọi chữ được nhúng.
type Embeddings struct {
	caller
	BaseURL, APIKey, Model string
	Dim                    int // số chiều yêu cầu (dimensions) và phải nhận về
	Batch                  int // số chữ mỗi yêu cầu; 0 = 40 (hạn mức theo số yêu cầu mỗi phút)
}

// Name định danh không gian vector: vector của hai Name khác nhau không so được với nhau.
func (e *Embeddings) Name() string { return fmt.Sprintf("%s/%d", e.Model, e.Dim) }

// Embed trả vector theo đúng thứ tự texts. Câu hỏi và tài liệu nhúng cùng một cách (giao thức chung không có kiểu nhiệm vụ).
func (e *Embeddings) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	batch := e.Batch
	if batch <= 0 {
		batch = 40
	}
	out := make([][]float32, 0, len(texts))
	for i := 0; i < len(texts); i += batch {
		lot := texts[i:min(i+batch, len(texts))]
		var r struct {
			Data []struct {
				Index     *int      `json:"index"` // có nơi không trả: khi đó theo thứ tự gửi
				Embedding []float32 `json:"embedding"`
			} `json:"data"`
		}
		in := map[string]any{"model": e.Model, "input": lot, "dimensions": e.Dim}
		if err := e.post(ctx, e.BaseURL+"/embeddings", http.Header{"Authorization": {"Bearer " + e.APIKey}}, in, &r); err != nil {
			return nil, err
		}
		if len(r.Data) != len(lot) {
			return nil, contracts.Errorf(contracts.ThirdParty(e.Provider, contracts.KindBadResponse), fmt.Sprintf("gửi %d chữ, nhận %d vector", len(lot), len(r.Data)))
		}
		got := make([][]float32, len(lot))
		for j, d := range r.Data {
			at := j
			if d.Index != nil {
				at = *d.Index
			}
			if at < 0 || at >= len(lot) || got[at] != nil || len(d.Embedding) != e.Dim {
				return nil, contracts.Errorf(contracts.ThirdParty(e.Provider, contracts.KindBadResponse),
					fmt.Sprintf("vector #%d: chỉ số %d lạ / lặp, hoặc %d chiều (cấu hình %d)", j, at, len(d.Embedding), e.Dim))
			}
			got[at] = d.Embedding
		}
		out = append(out, got...)
	}
	return out, nil
}

// Rerank chấm lại mức liên quan của từng đoạn với câu hỏi qua giao thức /rerank. Dữ liệu ra ngoài: câu hỏi + các đoạn đưa vào.
type Rerank struct {
	caller
	BaseURL, APIKey, Model string
}

// Name định danh reranker (ngưỡng "biết im" gắn với nó).
func (r *Rerank) Name() string { return r.Model }

// Rerank trả điểm theo đúng thứ tự docs.
func (r *Rerank) Rerank(ctx context.Context, query string, docs []string) ([]float64, error) {
	if len(docs) == 0 {
		return nil, nil
	}
	var resp struct {
		Results []struct {
			Index int     `json:"index"`
			Score float64 `json:"relevance_score"`
		} `json:"results"`
	}
	in := map[string]any{"model": r.Model, "query": query, "documents": docs, "top_n": len(docs), "return_documents": false}
	if err := r.post(ctx, r.BaseURL+"/rerank", http.Header{"Authorization": {"Bearer " + r.APIKey}}, in, &resp); err != nil {
		return nil, err
	}
	if len(resp.Results) != len(docs) {
		return nil, contracts.Errorf(contracts.ThirdParty(r.Provider, contracts.KindBadResponse), fmt.Sprintf("gửi %d đoạn, nhận %d điểm", len(docs), len(resp.Results)))
	}
	scores, seen := make([]float64, len(docs)), make([]bool, len(docs))
	for _, x := range resp.Results {
		if x.Index < 0 || x.Index >= len(docs) || seen[x.Index] {
			return nil, contracts.Errorf(contracts.ThirdParty(r.Provider, contracts.KindBadResponse), fmt.Sprintf("chỉ số đoạn %d lạ hoặc lặp", x.Index))
		}
		scores[x.Index], seen[x.Index] = x.Score, true
	}
	return scores, nil
}
