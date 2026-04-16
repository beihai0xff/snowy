//go:build integration

package integration

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/beihai0xff/snowy/internal/pkg/config"
	mysqlrepo "github.com/beihai0xff/snowy/internal/repo/mysql"
	redisrepo "github.com/beihai0xff/snowy/internal/repo/redis"
)

var (
	integrationDB    *gorm.DB
	integrationRedis *goredis.Client
	integrationMinIO *minio.Client
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	if strings.TrimSpace(os.Getenv("SNOWY_OPENSEARCH_INDEX")) == "" {
		_ = os.Setenv("SNOWY_OPENSEARCH_INDEX", "snowy-content-integration")
	}
	if strings.TrimSpace(os.Getenv("SNOWY_OPENSEARCH_VECTOR_DIM")) == "" {
		_ = os.Setenv("SNOWY_OPENSEARCH_VECTOR_DIM", "4")
	}

	db, err := mysqlrepo.NewDB(integrationDatabaseConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect mysql for integration tests: %v\n", err)
		os.Exit(1)
	}
	integrationDB = db

	if err := applyMySQLMigrations(ctx, integrationDB); err != nil {
		fmt.Fprintf(os.Stderr, "failed to apply mysql migrations for integration tests: %v\n", err)
		closeIntegrationDB(integrationDB)
		os.Exit(1)
	}

	rdb, err := redisrepo.NewClient(integrationRedisConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect redis for integration tests: %v\n", err)
		closeIntegrationDB(integrationDB)
		os.Exit(1)
	}
	integrationRedis = rdb

	minioClient, err := newMinIOClient(integrationMinIOConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect minio for integration tests: %v\n", err)
		_ = integrationRedis.Close()
		closeIntegrationDB(integrationDB)
		os.Exit(1)
	}
	integrationMinIO = minioClient

	if err := resetMySQL(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to reset mysql state for integration tests: %v\n", err)
		_ = integrationRedis.Close()
		closeIntegrationDB(integrationDB)
		os.Exit(1)
	}
	if err := resetRedis(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to reset redis state for integration tests: %v\n", err)
		_ = integrationRedis.Close()
		closeIntegrationDB(integrationDB)
		os.Exit(1)
	}
	if err := resetOpenSearch(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to reset opensearch state for integration tests: %v\n", err)
		_ = integrationRedis.Close()
		closeIntegrationDB(integrationDB)
		os.Exit(1)
	}
	if err := resetMinIO(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to reset minio state for integration tests: %v\n", err)
		_ = integrationRedis.Close()
		closeIntegrationDB(integrationDB)
		os.Exit(1)
	}

	code := m.Run()

	_ = resetMinIO(ctx)
	_ = resetOpenSearch(ctx)
	_ = resetRedis(ctx)
	_ = integrationRedis.Close()
	closeIntegrationDB(integrationDB)
	os.Exit(code)
}

func integrationDatabaseConfig() config.DatabaseConfig {
	return config.DatabaseConfig{
		Host:            getenv("SNOWY_DATABASE_HOST", "127.0.0.1"),
		Port:            getenvInt("SNOWY_DATABASE_PORT", 3306),
		User:            getenv("SNOWY_DATABASE_USER", "snowy"),
		Password:        getenv("SNOWY_DATABASE_PASSWORD", "snowy_secret"),
		Name:            getenv("SNOWY_DATABASE_NAME", "snowy"),
		Charset:         "utf8mb4",
		ParseTime:       true,
		Loc:             "Local",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}
}

func integrationRedisConfig() config.RedisConfig {
	return config.RedisConfig{
		Addr:     getenv("SNOWY_REDIS_ADDR", "127.0.0.1:6379"),
		Password: getenv("SNOWY_REDIS_PASSWORD", ""),
		DB:       getenvInt("SNOWY_REDIS_DB", 0),
		PoolSize: 10,
	}
}

func integrationAuthConfig() config.AuthConfig {
	return config.AuthConfig{
		JWTSecret:       getenv("SNOWY_AUTH_JWT_SECRET", "integration-secret"),
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	}
}

func integrationOpenSearchConfig() config.OpenSearchConfig {
	return config.OpenSearchConfig{
		Addresses: []string{getenv("SNOWY_OPENSEARCH_ADDR", "http://127.0.0.1:9200")},
		Username:  getenv("SNOWY_OPENSEARCH_USERNAME", "admin"),
		Password:  getenv("SNOWY_OPENSEARCH_PASSWORD", "admin"),
	}
}

func integrationMinIOConfig() config.MinIOConfig {
	return config.MinIOConfig{
		Endpoint:  getenv("SNOWY_MINIO_ENDPOINT", "127.0.0.1:9000"),
		AccessKey: getenv("SNOWY_MINIO_ACCESS_KEY", "snowy_admin"),
		SecretKey: getenv("SNOWY_MINIO_SECRET_KEY", "snowy_minio_secret"),
		Bucket:    getenv("SNOWY_MINIO_BUCKET", "snowy"),
		UseSSL:    false,
	}
}

func integrationMinIOBucketConfig(bucket string) config.MinIOConfig {
	cfg := integrationMinIOConfig()
	cfg.Bucket = bucket
	return cfg
}

func integrationMinIOBuckets() []string {
	seen := map[string]struct{}{}
	buckets := []string{}
	for _, bucket := range []string{
		integrationMinIOConfig().Bucket,
		"snowy-content",
		"snowy-charts",
		"snowy-exports",
	} {
		bucket = strings.TrimSpace(bucket)
		if bucket == "" {
			continue
		}
		if _, ok := seen[bucket]; ok {
			continue
		}
		seen[bucket] = struct{}{}
		buckets = append(buckets, bucket)
	}
	return buckets
}

func getenv(key, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return defaultValue
}

func getenvInt(key string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func projectRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func applyMySQLMigrations(ctx context.Context, db *gorm.DB) error {
	return mysqlrepo.RunMigrations(ctx, db)
}

func splitSQLStatements(content string) []string {
	var (
		statements []string
		builder    strings.Builder
	)

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}

		builder.WriteString(line)
		builder.WriteByte('\n')
		if strings.Contains(trimmed, ";") {
			stmt := strings.TrimSpace(builder.String())
			stmt = strings.TrimSuffix(stmt, ";")
			if stmt != "" {
				statements = append(statements, stmt)
			}
			builder.Reset()
		}
	}

	return statements
}

func resetMySQL(ctx context.Context) error {
	queries := []string{
		"SET FOREIGN_KEY_CHECKS = 0",
		"TRUNCATE TABLE agent_tool_calls",
		"TRUNCATE TABLE agent_runs",
		"TRUNCATE TABLE agent_messages",
		"TRUNCATE TABLE agent_sessions",
		"TRUNCATE TABLE favorites",
		"TRUNCATE TABLE history_items",
		"TRUNCATE TABLE concept_graph_snapshots",
		"TRUNCATE TABLE biology_runs",
		"TRUNCATE TABLE physics_runs",
		"TRUNCATE TABLE search_logs",
		"TRUNCATE TABLE content_chunks",
		"TRUNCATE TABLE content_documents",
		"TRUNCATE TABLE prompt_templates",
		"TRUNCATE TABLE users",
		"SET FOREIGN_KEY_CHECKS = 1",
	}

	for _, query := range queries {
		if err := integrationDB.WithContext(ctx).Exec(query).Error; err != nil {
			return fmt.Errorf("exec reset query %q: %w", query, err)
		}
	}
	return nil
}

func closeIntegrationDB(db *gorm.DB) {
	if db == nil {
		return
	}

	sqlDB, err := db.DB()
	if err != nil {
		return
	}

	_ = sqlDB.Close()
}

func resetRedis(ctx context.Context) error {
	if integrationRedis == nil {
		return nil
	}
	return integrationRedis.FlushDB(ctx).Err()
}

func integrationOpenSearchIndex() string {
	return getenv("SNOWY_OPENSEARCH_INDEX", "snowy-content-integration")
}

func resetOpenSearch(ctx context.Context) error {
	return withRetry(ctx, 12, 2*time.Second, func() error {
		return resetOpenSearchOnce(ctx)
	})
}

func resetOpenSearchOnce(ctx context.Context) error {
	cfg := integrationOpenSearchConfig()
	endpoint := strings.TrimRight(cfg.Addresses[0], "/")
	indexName := integrationOpenSearchIndex()
	httpClient := &http.Client{Timeout: 15 * time.Second}

	deleteReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint+"/"+indexName, nil)
	if err != nil {
		return fmt.Errorf("create opensearch delete request: %w", err)
	}
	if cfg.Username != "" {
		deleteReq.SetBasicAuth(cfg.Username, cfg.Password)
	}
	deleteResp, err := httpClient.Do(deleteReq)
	if err != nil {
		return fmt.Errorf("delete opensearch index: %w", err)
	}
	_, _ = io.Copy(io.Discard, deleteResp.Body)
	_ = deleteResp.Body.Close()
	if deleteResp.StatusCode != http.StatusOK && deleteResp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("delete opensearch index unexpected status: %d", deleteResp.StatusCode)
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
					"dimension": getenvInt("SNOWY_OPENSEARCH_VECTOR_DIM", 4),
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
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal opensearch create payload: %w", err)
	}
	createReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		endpoint+"/"+indexName,
		strings.NewReader(string(body)),
	)
	if err != nil {
		return fmt.Errorf("create opensearch create request: %w", err)
	}
	createReq.Header.Set("Content-Type", "application/json")
	if cfg.Username != "" {
		createReq.SetBasicAuth(cfg.Username, cfg.Password)
	}
	createResp, err := httpClient.Do(createReq)
	if err != nil {
		return fmt.Errorf("create opensearch index request: %w", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode >= http.StatusBadRequest {
		respBody, _ := io.ReadAll(createResp.Body)
		return fmt.Errorf(
			"create opensearch index status %d: %s",
			createResp.StatusCode,
			strings.TrimSpace(string(respBody)),
		)
	}
	return nil
}

func newMinIOClient(cfg config.MinIOConfig) (*minio.Client, error) {
	return minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
}

func resetMinIO(ctx context.Context) error {
	return withRetry(ctx, 10, time.Second, func() error {
		return resetMinIOOnce(ctx)
	})
}

func resetMinIOOnce(ctx context.Context) error {
	if integrationMinIO == nil {
		return nil
	}
	for _, bucket := range integrationMinIOBuckets() {
		exists, err := integrationMinIO.BucketExists(ctx, bucket)
		if err != nil {
			return fmt.Errorf("check minio bucket %s: %w", bucket, err)
		}
		if !exists {
			if err := integrationMinIO.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
				return fmt.Errorf("create minio bucket %s: %w", bucket, err)
			}
			continue
		}

		for objectInfo := range integrationMinIO.ListObjects(ctx, bucket, minio.ListObjectsOptions{Recursive: true}) {
			if objectInfo.Err != nil {
				return fmt.Errorf("list minio objects in %s: %w", bucket, objectInfo.Err)
			}
			if err := integrationMinIO.RemoveObject(ctx, bucket, objectInfo.Key, minio.RemoveObjectOptions{}); err != nil {
				return fmt.Errorf("remove minio object %s/%s: %w", bucket, objectInfo.Key, err)
			}
		}
	}
	return nil
}

func withRetry(ctx context.Context, attempts int, delay time.Duration, fn func() error) error {
	var lastErr error
	for i := 0; i < attempts; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			return errors.Join(ctx.Err(), lastErr)
		case <-time.After(delay):
		}
	}
	return lastErr
}
