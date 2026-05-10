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

func TestLoad_LLMEnvOverride(t *testing.T) {
	t.Setenv("SNOWY_LLM_PRIMARY_PROVIDER", "mimo")
	t.Setenv("SNOWY_LLM_PRIMARY_MODEL_PROVIDER", "mimo-env")
	t.Setenv("SNOWY_LLM_PRIMARY_MODEL", "mimo-env-model")
	t.Setenv("SNOWY_LLM_PRIMARY_MODEL_NAME", "mimo-env-model-name")
	t.Setenv("SNOWY_LLM_PRIMARY_BASE_URL", "https://llm.example.test/v1")
	t.Setenv("SNOWY_LLM_PRIMARY_BASEURL", "https://llm-baseurl.example.test/v1")
	t.Setenv("SNOWY_LLM_FALLBACK_PROVIDER", "openai")
	t.Setenv("SNOWY_LLM_FALLBACK_MODEL", "fallback-env-model")
	t.Setenv("SNOWY_LLM_FALLBACK_BASE_URL", "https://fallback.example.test/v1")

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(`llm:
  primary:
    provider: "openai"
    model_provider: "openai"
    model: "file-model"
    base_url: "https://file.example.test/v1"
  fallback:
    provider: "google"
    model: "file-fallback-model"
    base_url: "https://file-fallback.example.test/v1"
`), 0o600))

	cfg, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, "mimo", cfg.LLM.Primary.Provider)
	assert.Equal(t, "mimo-env", cfg.LLM.Primary.ModelProvider)
	assert.Equal(t, "mimo-env-model", cfg.LLM.Primary.Model)
	assert.Equal(t, "mimo-env-model-name", cfg.LLM.Primary.EffectiveModel())
	assert.Equal(t, "https://llm.example.test/v1", cfg.LLM.Primary.BaseURL)
	assert.Equal(t, "https://llm-baseurl.example.test/v1", cfg.LLM.Primary.EffectiveBaseURL())
	assert.Equal(t, "openai", cfg.LLM.Fallback.Provider)
	assert.Equal(t, "fallback-env-model", cfg.LLM.Fallback.Model)
	assert.Equal(t, "https://fallback.example.test/v1", cfg.LLM.Fallback.BaseURL)
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

func TestLLMConfig_EffectiveModelsPriority(t *testing.T) {
	cfg := LLMConfig{
		Models: []ModelProviderConfig{
			{Provider: "openai", Model: "slow", BaseURL: "https://slow.example.test/v1", Priority: 30},
			{Provider: "mimo", Model: "fast", BaseURL: "https://fast.example.test/v1", Priority: 10},
		},
	}

	models := cfg.EffectiveModels()
	require.Len(t, models, 2)
	assert.Equal(t, "fast", models[0].EffectiveModel())
	assert.Equal(t, "slow", models[1].EffectiveModel())
}

func TestLLMConfig_EffectiveModelsBackwardCompatibility(t *testing.T) {
	cfg := LLMConfig{
		Primary:  ModelProviderConfig{Provider: "mimo", Model: "primary", BaseURL: "https://primary.example.test/v1"},
		Fallback: ModelProviderConfig{Provider: "openai", Model: "fallback", BaseURL: "https://fallback.example.test/v1"},
	}

	models := cfg.EffectiveModels()
	require.Len(t, models, 2)
	assert.Equal(t, "primary", models[0].EffectiveModel())
	assert.Equal(t, "fallback", models[1].EffectiveModel())
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
