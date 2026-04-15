// Package opensearch 提供 OpenSearch 搜索引擎适配实现。
// 参考技术方案 §6.2.2。
package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/beihai0xff/snowy/internal/content"
	"github.com/beihai0xff/snowy/internal/pkg/config"
	internalsearch "github.com/beihai0xff/snowy/internal/repo/search"
	"github.com/google/uuid"
)

// OpenSearchAdapter OpenSearch 搜索适配器。
// 实现 search.Repository 和 content/indexer.Indexer 接口。
//
//nolint:revive // Adapter suffix is intentional to distinguish this infrastructure implementation from domain ports.
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
		if err := encodeBulkChunk(encoder, chunk); err != nil {
			return err
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
		return errors.New("opensearch bulk index returned errors")
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

func (a *OpenSearchAdapter) buildSearchPayload(
	query *internalsearch.ParsedQuery,
	queryText string,
	keywords, entities []string,
	filters internalsearch.Filters,
	offset, limit int,
) map[string]any {
	if limit <= 0 {
		limit = 10
	}

	candidateLimit := candidateLimit(limit, query)
	filter := buildFilterClauses(filters)
	textShould := buildTextShouldClauses(queryText, keywords, entities)
	queryBody := buildQueryBody(query, filter, textShould, candidateLimit)

	return map[string]any{
		"from":             offset,
		"size":             limit,
		"track_total_hits": true,
		"_source": []string{
			"doc_id",
			"document_id",
			"chunk_index",
			"content",
			"embedding",
			"tags",
			"chunk_type",
			"subject",
			"grade",
			"chapter",
			"source_type",
			"created_at",
		},
		"query": queryBody,
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

func encodeBulkChunk(encoder *json.Encoder, chunk *content.Chunk) error {
	docID := chunkDocID(chunk)

	doc := buildOpenSearchDocument(chunk, docID)
	if err := encoder.Encode(map[string]any{"index": map[string]any{"_id": bulkMetaID(chunk, docID)}}); err != nil {
		return fmt.Errorf("encode opensearch bulk meta: %w", err)
	}

	if err := encoder.Encode(doc); err != nil {
		return fmt.Errorf("encode opensearch bulk doc: %w", err)
	}

	return nil
}

func chunkDocID(chunk *content.Chunk) string {
	if chunk.DocumentID != uuid.Nil {
		return chunk.DocumentID.String()
	}

	return chunk.ID.String()
}

func bulkMetaID(chunk *content.Chunk, docID string) string {
	if chunk.ID != uuid.Nil {
		return chunk.ID.String()
	}

	return fmt.Sprintf("%s_%d", docID, chunk.ChunkIndex)
}

func buildOpenSearchDocument(chunk *content.Chunk, docID string) openSearchDocument {
	subject, grade, chapter, sourceType := parseChunkTags(chunk.Tags)

	return openSearchDocument{
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
}

func candidateLimit(limit int, query *internalsearch.ParsedQuery) int {
	if query != nil && len(query.Embedding) > 0 {
		return maxInt(limit*5, 50)
	}

	return maxInt(limit, 10)
}

func buildFilterClauses(filters internalsearch.Filters) []any {
	clauses := make([]any, 0, 4)
	clauses = appendTermFilter(clauses, "subject", filters.Subject)
	clauses = appendTermFilter(clauses, "grade", filters.Grade)
	clauses = appendTermFilter(clauses, "chapter", filters.Chapter)
	clauses = appendTermFilter(clauses, "source_type", filters.Source)

	return clauses
}

func appendTermFilter(clauses []any, field, value string) []any {
	if value == "" {
		return clauses
	}

	return append(clauses, map[string]any{"term": map[string]any{field: value}})
}

func buildTextShouldClauses(queryText string, keywords, entities []string) []any {
	clauses := make([]any, 0, 8)
	if queryText != "" {
		clauses = append(clauses, textQueryClauses(queryText)...)
	}

	for _, keyword := range keywords {
		clauses = append(clauses, keywordClauses(keyword)...)
	}

	for _, entity := range entities {
		clauses = append(clauses, entityClauses(entity)...)
	}

	return clauses
}

func textQueryClauses(queryText string) []any {
	return []any{
		map[string]any{"multi_match": map[string]any{
			"query":  queryText,
			"fields": []string{"content^4", "tags^2", "subject^2", "chapter^2", "source_type"},
			"type":   "best_fields",
		}},
		map[string]any{"match_phrase": map[string]any{
			"content": map[string]any{"query": queryText, "boost": 3},
		}},
	}
}

func keywordClauses(keyword string) []any {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil
	}

	return []any{
		map[string]any{"match": map[string]any{"content": map[string]any{"query": keyword, "boost": 2}}},
		map[string]any{"term": map[string]any{"tags": keyword}},
	}
}

func entityClauses(entity string) []any {
	entity = strings.TrimSpace(entity)
	if entity == "" {
		return nil
	}

	return []any{
		map[string]any{"term": map[string]any{"subject": entity}},
		map[string]any{"term": map[string]any{"chapter": entity}},
		map[string]any{"term": map[string]any{"tags": entity}},
	}
}

func buildQueryBody(query *internalsearch.ParsedQuery, filter, textShould []any, knnCandidates int) map[string]any {
	hasVector := query != nil && len(query.Embedding) > 0
	hasText := len(textShould) > 0

	if hasVector && hasText {
		return hybridQueryBody(query.Embedding, filter, textShould, knnCandidates)
	}

	if hasVector {
		return vectorOnlyQueryBody(query.Embedding, filter, knnCandidates)
	}

	if hasText || len(filter) > 0 {
		return textOnlyQueryBody(filter, textShould)
	}

	return map[string]any{"match_all": map[string]any{}}
}

func hybridQueryBody(embedding []float64, filter, textShould []any, knnCandidates int) map[string]any {
	clauses := []any{
		map[string]any{
			"bool": map[string]any{
				"should":               textShould,
				"minimum_should_match": 1,
			},
		},
		buildKNNClause(embedding, knnCandidates),
	}

	return map[string]any{
		"bool": map[string]any{
			"filter":               filter,
			"should":               clauses,
			"minimum_should_match": 1,
		},
	}
}

func vectorOnlyQueryBody(embedding []float64, filter []any, knnCandidates int) map[string]any {
	knn := buildKNNClause(embedding, knnCandidates)
	if len(filter) == 0 {
		return knn
	}

	return map[string]any{
		"bool": map[string]any{
			"filter": filter,
			"must":   []any{knn},
		},
	}
}

func textOnlyQueryBody(filter, textShould []any) map[string]any {
	boolQuery := map[string]any{"filter": filter}
	if len(textShould) > 0 {
		boolQuery["should"] = textShould
		boolQuery["minimum_should_match"] = 1
	}

	return map[string]any{"bool": boolQuery}
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

func (a *OpenSearchAdapter) do(
	ctx context.Context,
	method, path string,
	body io.Reader,
	contentType string,
) ([]byte, error) {
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
	fields := map[string]*string{
		"subject":     &subject,
		"grade":       &grade,
		"chapter":     &chapter,
		"source":      &sourceType,
		"source_type": &sourceType,
	}

	for _, tag := range tags {
		key, value, ok := strings.Cut(tag, ":")
		if !ok {
			continue
		}

		target, exists := fields[strings.ToLower(strings.TrimSpace(key))]
		if !exists {
			continue
		}

		trimmed := strings.TrimSpace(value)
		if trimmed == "" || *target != "" {
			continue
		}

		*target = trimmed
	}

	return subject, grade, chapter, sourceType
}
