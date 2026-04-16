package embedding

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/beihai0xff/snowy/internal/pkg/config"
)

// openaiEmbedding 基于 OpenAI text-embedding-3-large 的实现。
// 当未配置 API Key 时会退化为本地确定性向量，便于开发和测试环境继续工作。
type openaiEmbedding struct {
	cfg        config.EmbeddingConfig
	httpClient *http.Client
}

// NewOpenAIEmbedding 创建 OpenAI Embedding Provider。
func NewOpenAIEmbedding(cfg config.EmbeddingConfig) Provider {
	return &openaiEmbedding{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

//nolint:cyclop // The embedding flow has distinct request construction, fallback, and decoding branches.
func (e *openaiEmbedding) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	if strings.TrimSpace(e.cfg.APIKey) == "" {
		return e.localEmbeddings(texts), nil
	}

	payload := map[string]any{"model": e.cfg.Model, "input": texts}
	if e.cfg.Dimensions > 0 {
		payload["dimensions"] = e.cfg.Dimensions
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}

	baseURL := strings.TrimRight(strings.TrimSpace(e.cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create embedding request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+e.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request embeddings: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read embedding response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf(
			"embedding request failed: status=%d body=%s",
			resp.StatusCode,
			strings.TrimSpace(string(responseBody)),
		)
	}

	var payloadResp struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(responseBody, &payloadResp); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}

	vectors := make([][]float64, 0, len(payloadResp.Data))
	for _, item := range payloadResp.Data {
		vectors = append(vectors, item.Embedding)
	}

	if len(vectors) == 0 {
		return e.localEmbeddings(texts), nil
	}

	return vectors, nil
}

func (e *openaiEmbedding) Dimensions() int {
	return e.cfg.Dimensions
}

func (e *openaiEmbedding) localEmbeddings(texts []string) [][]float64 {
	dimensions := e.cfg.Dimensions
	if dimensions <= 0 {
		dimensions = 32
	}

	vectors := make([][]float64, 0, len(texts))
	for _, text := range texts {
		hash := sha256.Sum256([]byte(text))

		vector := make([]float64, dimensions)
		for i := range dimensions {
			start := (i * 4) % len(hash)
			value := binary.BigEndian.Uint32([]byte{
				hash[start%len(hash)],
				hash[(start+1)%len(hash)],
				hash[(start+2)%len(hash)],
				hash[(start+3)%len(hash)],
			})
			vector[i] = float64(value%1000)/1000 - 0.5
		}

		normalize(vector)
		vectors = append(vectors, vector)
	}

	return vectors
}

func normalize(vector []float64) {
	sum := 0.0
	for _, value := range vector {
		sum += value * value
	}

	if sum == 0 {
		return
	}

	norm := math.Sqrt(sum)
	for i := range vector {
		vector[i] /= norm
	}
}
