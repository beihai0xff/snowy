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

func TestLoad_EnvOverride(t *testing.T) {
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
}
