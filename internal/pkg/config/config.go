// Package config 提供配置加载能力，基于 Viper 实现多环境配置管理。
// 参考技术方案 §6.1.3。
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 是 Snowy 的顶层配置结构体。
type Config struct {
	Server        ServerConfig        `mapstructure:"server"`
	Database      DatabaseConfig      `mapstructure:"database"`
	Redis         RedisConfig         `mapstructure:"redis"`
	OpenSearch    OpenSearchConfig    `mapstructure:"opensearch"`
	MinIO         MinIOConfig         `mapstructure:"minio"`
	LLM           LLMConfig           `mapstructure:"llm"`
	Embedding     EmbeddingConfig     `mapstructure:"embedding"`
	Auth          AuthConfig          `mapstructure:"auth"`
	RateLimit     RateLimitConfig     `mapstructure:"ratelimit"`
	TokenBudget   TokenBudgetConfig   `mapstructure:"token_budget"`
	Observability ObservabilityConfig `mapstructure:"observability"`
}

// ServerConfig HTTP 服务配置。
type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Mode            string        `mapstructure:"mode"` // debug / release / test
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// Addr 返回监听地址。
func (s ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// DatabaseConfig MySQL 连接配置。
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	Charset         string        `mapstructure:"charset"`
	ParseTime       bool          `mapstructure:"parse_time"`
	Loc             string        `mapstructure:"loc"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// DSN 返回 MySQL 连接字符串 (go-sql-driver/mysql 格式)。
func (d DatabaseConfig) DSN() string {
	charset := d.Charset
	if charset == "" {
		charset = "utf8mb4"
	}

	loc := d.Loc
	if loc == "" {
		loc = "Local"
	}

	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, charset, d.ParseTime, loc,
	)
}

// RedisConfig Redis 连接配置。
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

// OpenSearchConfig OpenSearch 连接配置。
type OpenSearchConfig struct {
	Addresses          []string `mapstructure:"addresses"`
	Username           string   `mapstructure:"username"`
	Password           string   `mapstructure:"password"`
	InsecureSkipVerify bool     `mapstructure:"insecure_skip_verify"`
}

// MinIOConfig 对象存储配置。
type MinIOConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Bucket    string `mapstructure:"bucket"`
	UseSSL    bool   `mapstructure:"use_ssl"`
}

// ModelProviderConfig 单个模型供应商配置。
type ModelProviderConfig struct {
	Provider   string        `mapstructure:"provider"`
	Model      string        `mapstructure:"model"`
	APIKey     string        `mapstructure:"api_key"`
	BaseURL    string        `mapstructure:"base_url"`
	Timeout    time.Duration `mapstructure:"timeout"`
	MaxRetries int           `mapstructure:"max_retries"`
}

// LLMConfig 大模型配置（主 + 备选）。
type LLMConfig struct {
	Primary  ModelProviderConfig `mapstructure:"primary"`
	Fallback ModelProviderConfig `mapstructure:"fallback"`
}

// EmbeddingConfig Embedding 模型配置。
type EmbeddingConfig struct {
	Provider   string `mapstructure:"provider"`
	Model      string `mapstructure:"model"`
	APIKey     string `mapstructure:"api_key"`
	BaseURL    string `mapstructure:"base_url"`
	Dimensions int    `mapstructure:"dimensions"`
}

// AuthConfig 鉴权配置。
type AuthConfig struct {
	JWTSecret       string            `mapstructure:"jwt_secret"`
	AccessTokenTTL  time.Duration     `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration     `mapstructure:"refresh_token_ttl"`
	GoogleOAuth     GoogleOAuthConfig `mapstructure:"google_oauth"`
}

// GoogleOAuthConfig Google OAuth 2.0 配置。
type GoogleOAuthConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURI  string `mapstructure:"redirect_uri"`
}

// RateLimitConfig 限流配置。
type RateLimitConfig struct {
	AuthenticatedRPM int `mapstructure:"authenticated_rpm"`
	AnonymousRPM     int `mapstructure:"anonymous_rpm"`
}

// TokenBudgetConfig Token 预算管控，参考技术方案 §18B。
type TokenBudgetConfig struct {
	PerRequestInput  int `mapstructure:"per_request_input"`
	PerRequestOutput int `mapstructure:"per_request_output"`
	PerSession       int `mapstructure:"per_session"`
	PerUserDaily     int `mapstructure:"per_user_daily"`
}

// ObservabilityConfig 可观测性配置。
type ObservabilityConfig struct {
	OTelEndpoint   string `mapstructure:"otel_endpoint"`
	PrometheusPath string `mapstructure:"prometheus_path"`
	LogLevel       string `mapstructure:"log_level"`
	LogFormat      string `mapstructure:"log_format"`
}

// Load 从指定路径加载配置文件，支持环境变量覆盖。
// configPath 为配置文件路径（不含扩展名），如 "configs/config"。
func Load(configPath string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(configPath)
	v.SetEnvPrefix("SNOWY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}
