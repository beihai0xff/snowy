package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseConfig_DSN(t *testing.T) {
	tests := []struct {
		name string
		cfg  DatabaseConfig
		want string
	}{
		{
			name: "with defaults",
			cfg: DatabaseConfig{
				User: "root", Password: "pass", Host: "localhost", Port: 3306,
				Name: "snowy", ParseTime: true,
			},
			want: "root:pass@tcp(localhost:3306)/snowy?charset=utf8mb4&parseTime=true&loc=Local",
		},
		{
			name: "with explicit charset and loc",
			cfg: DatabaseConfig{
				User: "admin", Password: "secret", Host: "db.host", Port: 3307,
				Name: "test_db", Charset: "utf8", Loc: "UTC", ParseTime: false,
			},
			want: "admin:secret@tcp(db.host:3307)/test_db?charset=utf8&parseTime=false&loc=UTC",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.cfg.DSN())
		})
	}
}

func TestServerConfig_Addr(t *testing.T) {
	tests := []struct {
		host string
		port int
		want string
	}{
		{"0.0.0.0", 8080, "0.0.0.0:8080"},
		{"", 3000, ":3000"},
		{"localhost", 443, "localhost:443"},
	}
	for _, tt := range tests {
		cfg := ServerConfig{Host: tt.host, Port: tt.port}
		assert.Equal(t, tt.want, cfg.Addr())
	}
}

func TestServerConfig_RunMode(t *testing.T) {
	t.Run("default to all", func(t *testing.T) {
		cfg := ServerConfig{}
		assert.Equal(t, RunModeAll, cfg.EffectiveRunMode())
		assert.True(t, cfg.APIEnabled())
		assert.True(t, cfg.WorkerEnabled())
		require.NoError(t, cfg.ValidateRunMode())
	})

	t.Run("api only", func(t *testing.T) {
		cfg := ServerConfig{RunMode: " api "}
		assert.Equal(t, RunModeAPI, cfg.EffectiveRunMode())
		assert.True(t, cfg.APIEnabled())
		assert.False(t, cfg.WorkerEnabled())
		require.NoError(t, cfg.ValidateRunMode())
	})

	t.Run("worker only", func(t *testing.T) {
		cfg := ServerConfig{RunMode: "WORKER"}
		assert.Equal(t, RunModeWorker, cfg.EffectiveRunMode())
		assert.False(t, cfg.APIEnabled())
		assert.True(t, cfg.WorkerEnabled())
		require.NoError(t, cfg.ValidateRunMode())
	})

	t.Run("invalid", func(t *testing.T) {
		cfg := ServerConfig{RunMode: "invalid"}
		require.Error(t, cfg.ValidateRunMode())
	})
}

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("SNOWY_SERVER_RUN_MODE", "worker")
	t.Setenv("SNOWY_DATABASE_HOST", "127.0.0.1")
	t.Setenv("SNOWY_REDIS_ADDR", "127.0.0.1:6379")

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(`database:
  host: "localhost"
  port: 3306
redis:
  addr: "localhost:6379"
`), 0o600))

	cfg, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, "127.0.0.1", cfg.Database.Host)
	assert.Equal(t, "127.0.0.1:6379", cfg.Redis.Addr)
	assert.Equal(t, RunModeWorker, cfg.Server.EffectiveRunMode())
}

func TestLoad_LLMModels(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(`llm:
  models:
    - provider: "openai"
      model_provider: "gateway"
      model: "gateway-test-model"
      api_key: "local-test-key"
      base_url: "https://gateway.example.test/v1"
      timeout: 10m
      temperature: 0.2
      max_tokens: 131072
      max_retries: 2
      retry_interval: 1s
    - provider: "openai"
      model_name: "gpt-test"
      baseurl: "https://openai.example.test/v1"
`), 0o600))

	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.Len(t, cfg.LLM.Models, 2)

	assert.Equal(t, "openai", cfg.LLM.Models[0].Provider)
	assert.Equal(t, "gateway", cfg.LLM.Models[0].ModelProvider)
	assert.Equal(t, "gateway-test-model", cfg.LLM.Models[0].EffectiveModel())
	assert.Equal(t, "local-test-key", cfg.LLM.Models[0].APIKey)
	assert.Equal(t, "https://gateway.example.test/v1", cfg.LLM.Models[0].EffectiveBaseURL())
	assert.Equal(t, 600, int(cfg.LLM.Models[0].Timeout.Seconds()))
	assert.Equal(t, 0.2, cfg.LLM.Models[0].Temperature)
	assert.Equal(t, 131072, cfg.LLM.Models[0].MaxTokens)
	assert.Equal(t, 2, cfg.LLM.Models[0].MaxRetries)
	assert.Equal(t, "gpt-test", cfg.LLM.Models[1].EffectiveModel())
	assert.Equal(t, "https://openai.example.test/v1", cfg.LLM.Models[1].EffectiveBaseURL())
}

func TestLoad_MissingLocalConfigHint(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")

	_, err := Load(configPath)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "configs/config.example.yaml")
	assert.Contains(t, err.Error(), "configs/config.yaml")
	assert.Contains(t, err.Error(), "llm.models[].base_url")
	assert.Contains(t, err.Error(), "api_key")
}

func TestModelProviderConfig_EffectiveAliases(t *testing.T) {
	cfg := ModelProviderConfig{
		Model:               " file-model ",
		ModelName:           " alias-model ",
		BaseURL:             " https://file.example.test/v1 ",
		BaseURLNoUnderscore: " https://alias.example.test/v1 ",
	}

	assert.Equal(t, "alias-model", cfg.EffectiveModel())
	assert.Equal(t, "https://alias.example.test/v1", cfg.EffectiveBaseURL())
}

func TestLLMConfig_EffectiveModelsKeepsDeclarationOrder(t *testing.T) {
	cfg := LLMConfig{
		Models: []ModelProviderConfig{
			{Provider: "openai", Model: "first", BaseURL: "https://first.example.test/v1"},
			{Provider: "openai", Model: "second", BaseURL: "https://second.example.test/v1"},
		},
	}

	models := cfg.EffectiveModels()
	require.Len(t, models, 2)
	assert.Equal(t, "first", models[0].EffectiveModel())
	assert.Equal(t, "second", models[1].EffectiveModel())
}

func TestLLMConfig_EffectiveModelsFiltersEmptyEntries(t *testing.T) {
	cfg := LLMConfig{
		Models: []ModelProviderConfig{
			{},
			{Provider: "openai", Model: "configured", BaseURL: "https://configured.example.test/v1"},
		},
	}

	models := cfg.EffectiveModels()
	require.Len(t, models, 1)
	assert.Equal(t, "configured", models[0].EffectiveModel())
}

func TestEmbeddingConfig_EffectiveAliases(t *testing.T) {
	cfg := EmbeddingConfig{
		Model:               " file-embedding ",
		ModelName:           " alias-embedding ",
		BaseURL:             " https://file-embedding.example.test/v1 ",
		BaseURLNoUnderscore: " https://alias-embedding.example.test/v1 ",
	}

	assert.Equal(t, "alias-embedding", cfg.EffectiveModel())
	assert.Equal(t, "https://alias-embedding.example.test/v1", cfg.EffectiveBaseURL())
}
