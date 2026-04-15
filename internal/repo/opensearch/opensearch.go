// Package opensearch 提供 OpenSearch 搜索引擎适配实现。
// 参考技术方案 §6.2.2。
package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	internalsearch "github.com/beihai0xff/snowy/internal/repo/search"

	"github.com/beihai0xff/snowy/internal/content"
	"github.com/beihai0xff/snowy/internal/pkg/config"
)

// OpenSearchAdapter OpenSearch 搜索适配器。
// 实现 search.Repository 和 content/indexer.Indexer 接口。
type OpenSearchAdapter struct {
	cfg        config.OpenSearchConfig
	endpoint   string
	index      string
	httpClient *http.Client
}

// NewOpenSearchAdapter 创建 OpenSearch 适配器。
func NewOpenSearchAdapter(cfg config.OpenSearchConfig) *OpenSearchAdapter {
	endpoint := "http://127.0.0.1:9200"
	if len(cfg.Addresses) > 0 && strings.TrimSpace(cfg.Addresses[0]) != "" {
		endpoint = strings.TrimRight(cfg.Addresses[0], "/")
	}
	indexName := strings.TrimSpace(os.Getenv("SNOWY_OPENSEARCH_INDEX"))
	if indexName == "" {
		indexName = "snowy-content"
	}

	return &OpenSearchAdapter{
		cfg:      cfg,
		endpoint: endpoint,
		index:    indexName,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Search 执行混合检索（全文+向量+标签）。
func (a *OpenSearchAdapter) Search(
	ctx context.Context,
	query *internalsearch.ParsedQuery,
	filters internalsearch.Filters,
	offset, limit int,
) ([]internalsearch.Result, int64, error) {
	if err := a.ensureIndex(ctx); err != nil {
		return nil, 0, err
	}

	queryText := ""
	keywords := []string{}
	entities := []string{}
	if query != nil {
		queryText = strings.TrimSpace(query.Original)
		if queryText == "" {
			queryText = strings.TrimSpace(strings.Join(query.Keywords, " "))
		}
		keywords = query.Keywords
		entities = query.Entities
	}

	payload := a.buildSearchPayload(query, queryText, keywords, entities, filters, offset, limit)

	body, err := a.doJSON(ctx, http.MethodPost, "/"+a.index+"/_search", payload)
	if err != nil {
		return nil, 0, fmt.Errorf("opensearch search request: %w", err)
	}

	var resp openSearchSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, 0, fmt.Errorf("decode opensearch search response: %w", err)
	}

	results := make([]internalsearch.Result, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		results = append(results, internalsearch.Result{
			DocID:      hit.Source.DocID,
			SourceType: hit.Source.SourceType,
			Subject:    hit.Source.Subject,
			Chapter:    hit.Source.Chapter,
			Snippet:    hit.Source.Content,
			Score:      hit.Score,
			Tags:       hit.Source.Tags,
		})
	}
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, resp.Hits.Total.Value, nil
}

// GetByDocID 根据文档 ID 获取详情。
func (a *OpenSearchAdapter) GetByDocID(ctx context.Context, docID string) (*internalsearch.Result, error) {
	payload := map[string]any{
		"size": 1,
		"query": map[string]any{
			"term": map[string]any{"doc_id": docID},
		},
	}
	body, err := a.doJSON(ctx, http.MethodPost, "/"+a.index+"/_search", payload)
	if err != nil {
		return nil, err
	}

	var resp openSearchSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode get by doc id response: %w", err)
	}
	if len(resp.Hits.Hits) == 0 {
		return nil, fmt.Errorf("document %s not found", docID)
	}

	hit := resp.Hits.Hits[0]
	result := internalsearch.Result{
		DocID:      hit.Source.DocID,
		SourceType: hit.Source.SourceType,
		Subject:    hit.Source.Subject,
		Chapter:    hit.Source.Chapter,
		Snippet:    hit.Source.Content,
		Score:      hit.Score,
		Tags:       hit.Source.Tags,
	}
	return &result, nil
}

// Index 将切片写入 OpenSearch 索引。
func (a *OpenSearchAdapter) Index(ctx context.Context, chunks []*content.Chunk) error {
	if err := a.ensureIndex(ctx); err != nil {
		return err
	}
	if len(chunks) == 0 {
		return nil
	}

	var bulk strings.Builder
	encoder := json.NewEncoder(&bulk)
	for _, chunk := range chunks {
		docID := chunk.DocumentID.String()
		if docID == uuid.Nil.String() {
			docID = chunk.ID.String()
		}
		subject, grade, chapter, sourceType := parseChunkTags(chunk.Tags)
		doc := openSearchDocument{
			DocID:      docID,
			DocumentID: chunk.DocumentID.String(),
			ChunkIndex: chunk.ChunkIndex,
			Content:    chunk.Content,
			Embedding:  chunk.Embedding,
			Tags:       chunk.Tags,
			ChunkType:  chunk.ChunkType,
			Subject:    subject,
			Grade:      grade,
			Chapter:    chapter,
			SourceType: sourceType,
			CreatedAt:  chunk.CreatedAt.UTC().Format(time.RFC3339Nano),
		}
		metaID := fmt.Sprintf("%s_%d", docID, chunk.ChunkIndex)
		if chunk.ID != uuid.Nil {
			metaID = chunk.ID.String()
		}
		if err := encoder.Encode(map[string]any{"index": map[string]any{"_id": metaID}}); err != nil {
			return fmt.Errorf("encode opensearch bulk meta: %w", err)
		}
		if err := encoder.Encode(doc); err != nil {
			return fmt.Errorf("encode opensearch bulk doc: %w", err)
		}
	}

	path := "/" + a.index + "/_bulk?refresh=true"
	body, err := a.do(ctx, http.MethodPost, path, strings.NewReader(bulk.String()), "application/x-ndjson")
	if err != nil {
		return fmt.Errorf("opensearch bulk index request: %w", err)
	}

	var resp struct {
		Errors bool `json:"errors"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decode opensearch bulk response: %w", err)
	}
	if resp.Errors {
		return fmt.Errorf("opensearch bulk index returned errors")
	}
	return nil
}

// Delete 从索引中删除文档切片。
func (a *OpenSearchAdapter) Delete(ctx context.Context, documentID string) error {
	if err := a.ensureIndex(ctx); err != nil {
		return err
	}
	payload := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"should": []any{
					map[string]any{"term": map[string]any{"document_id": documentID}},
					map[string]any{"term": map[string]any{"doc_id": documentID}},
				},
				"minimum_should_match": 1,
			},
		},
	}
	_, err := a.doJSON(ctx, http.MethodPost, "/"+a.index+"/_delete_by_query?refresh=true", payload)
	if err != nil {
		return fmt.Errorf("opensearch delete by query: %w", err)
	}
	return nil
}

type openSearchDocument struct {
	DocID      string    `json:"doc_id"`
	DocumentID string    `json:"document_id"`
	ChunkIndex int       `json:"chunk_index"`
	Content    string    `json:"content"`
	Embedding  []float64 `json:"embedding,omitempty"`
	Tags       []string  `json:"tags,omitempty"`
	ChunkType  string    `json:"chunk_type,omitempty"`
	Subject    string    `json:"subject,omitempty"`
	Grade      string    `json:"grade,omitempty"`
	Chapter    string    `json:"chapter,omitempty"`
	SourceType string    `json:"source_type,omitempty"`
	CreatedAt  string    `json:"created_at,omitempty"`
}

type openSearchSearchResponse struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []struct {
			Score  float64            `json:"_score"`
			Source openSearchDocument `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func (a *OpenSearchAdapter) ensureIndex(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, a.endpoint+"/"+a.index, nil)
	if err != nil {
		return fmt.Errorf("create opensearch head request: %w", err)
	}
	a.applyAuth(req)
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("head opensearch index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}
	if resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("head index unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	payload := map[string]any{
		"settings": map[string]any{
			"index.knn":          true,
			"number_of_shards":   1,
			"number_of_replicas": 0,
		},
		"mappings": map[string]any{
			"properties": map[string]any{
				"doc_id":      map[string]any{"type": "keyword"},
				"document_id": map[string]any{"type": "keyword"},
				"chunk_index": map[string]any{"type": "integer"},
				"content":     map[string]any{"type": "text"},
				"embedding": map[string]any{
					"type":      "knn_vector",
					"dimension": a.vectorDimension(),
					"method": map[string]any{
						"name":       "hnsw",
						"space_type": "cosinesimil",
						"engine":     "lucene",
					},
				},
				"tags":        map[string]any{"type": "keyword"},
				"chunk_type":  map[string]any{"type": "keyword"},
				"subject":     map[string]any{"type": "keyword"},
				"grade":       map[string]any{"type": "keyword"},
				"chapter":     map[string]any{"type": "keyword"},
				"source_type": map[string]any{"type": "keyword"},
				"created_at":  map[string]any{"type": "date"},
			},
		},
	}
	_, err = a.doJSON(ctx, http.MethodPut, "/"+a.index, payload)
	if err != nil {
		return fmt.Errorf("create opensearch index: %w", err)
	}
	return nil
}

func (a *OpenSearchAdapter) buildSearchPayload(query *internalsearch.ParsedQuery, queryText string, keywords, entities []string, filters internalsearch.Filters, offset, limit int) map[string]any {
	if limit <= 0 {
		limit = 10
	}
	candidateLimit := maxInt(limit, 10)
	if query != nil && len(query.Embedding) > 0 {
		candidateLimit = maxInt(limit*5, 50)
	}

	filter := make([]any, 0, 4)
	textShould := make([]any, 0, 8)

	if filters.Subject != "" {
		filter = append(filter, map[string]any{"term": map[string]any{"subject": filters.Subject}})
	}
	if filters.Grade != "" {
		filter = append(filter, map[string]any{"term": map[string]any{"grade": filters.Grade}})
	}
	if filters.Chapter != "" {
		filter = append(filter, map[string]any{"term": map[string]any{"chapter": filters.Chapter}})
	}
	if filters.Source != "" {
		filter = append(filter, map[string]any{"term": map[string]any{"source_type": filters.Source}})
	}

	if queryText != "" {
		textShould = append(textShould,
			map[string]any{"multi_match": map[string]any{
				"query":  queryText,
				"fields": []string{"content^4", "tags^2", "subject^2", "chapter^2", "source_type"},
				"type":   "best_fields",
			}},
			map[string]any{"match_phrase": map[string]any{
				"content": map[string]any{"query": queryText, "boost": 3},
			}},
		)
	}
	for _, keyword := range keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			continue
		}
		textShould = append(textShould,
			map[string]any{"match": map[string]any{"content": map[string]any{"query": keyword, "boost": 2}}},
			map[string]any{"term": map[string]any{"tags": keyword}},
		)
	}
	for _, entity := range entities {
		entity = strings.TrimSpace(entity)
		if entity == "" {
			continue
		}
		textShould = append(textShould,
			map[string]any{"term": map[string]any{"subject": entity}},
			map[string]any{"term": map[string]any{"chapter": entity}},
			map[string]any{"term": map[string]any{"tags": entity}},
		)
	}

	var queryBody map[string]any
	hasVector := query != nil && len(query.Embedding) > 0
	hasText := len(textShould) > 0

	switch {
	case hasVector && hasText:
		clauses := []any{
			map[string]any{
				"bool": map[string]any{
					"should":               textShould,
					"minimum_should_match": 1,
				},
			},
			buildKNNClause(query.Embedding, candidateLimit),
		}
		queryBody = map[string]any{
			"bool": map[string]any{
				"filter":               filter,
				"should":               clauses,
				"minimum_should_match": 1,
			},
		}
	case hasVector:
		knn := buildKNNClause(query.Embedding, candidateLimit)
		if len(filter) == 0 {
			queryBody = knn
		} else {
			queryBody = map[string]any{
				"bool": map[string]any{
					"filter": filter,
					"must":   []any{knn},
				},
			}
		}
	case hasText || len(filter) > 0:
		boolQuery := map[string]any{"filter": filter}
		if hasText {
			boolQuery["should"] = textShould
			boolQuery["minimum_should_match"] = 1
		}
		queryBody = map[string]any{"bool": boolQuery}
	default:
		queryBody = map[string]any{"match_all": map[string]any{}}
	}

	return map[string]any{
		"from":             offset,
		"size":             limit,
		"track_total_hits": true,
		"_source":          []string{"doc_id", "document_id", "chunk_index", "content", "embedding", "tags", "chunk_type", "subject", "grade", "chapter", "source_type", "created_at"},
		"query":            queryBody,
	}
}

func buildKNNClause(embedding []float64, k int) map[string]any {
	return map[string]any{
		"knn": map[string]any{
			"embedding": map[string]any{
				"vector": embedding,
				"k":      k,
			},
		},
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (a *OpenSearchAdapter) vectorDimension() int {
	if raw := strings.TrimSpace(os.Getenv("SNOWY_OPENSEARCH_VECTOR_DIM")); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			return value
		}
	}
	return 3072
}

func (a *OpenSearchAdapter) doJSON(ctx context.Context, method, path string, payload any) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal opensearch payload: %w", err)
		}
		body = bytes.NewReader(b)
	}
	return a.do(ctx, method, path, body, "application/json")
}

func (a *OpenSearchAdapter) do(ctx context.Context, method, path string, body io.Reader, contentType string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, a.endpoint+path, body)
	if err != nil {
		return nil, fmt.Errorf("create opensearch request: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	a.applyAuth(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform opensearch request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read opensearch response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("opensearch status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return respBody, nil
}

func (a *OpenSearchAdapter) applyAuth(req *http.Request) {
	if a.cfg.Username != "" {
		req.SetBasicAuth(a.cfg.Username, a.cfg.Password)
	}
}

func parseChunkTags(tags []string) (subject, grade, chapter, sourceType string) {
	for _, tag := range tags {
		key, value, ok := strings.Cut(tag, ":")
		if !ok {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "subject":
			if subject == "" {
				subject = strings.TrimSpace(value)
			}
		case "grade":
			if grade == "" {
				grade = strings.TrimSpace(value)
			}
		case "chapter":
			if chapter == "" {
				chapter = strings.TrimSpace(value)
			}
		case "source", "source_type":
			if sourceType == "" {
				sourceType = strings.TrimSpace(value)
			}
		}
	}
	return subject, grade, chapter, sourceType
}
