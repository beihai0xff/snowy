// Package config 提供配置加载能力，基于 Viper 实现多环境配置管理。
// 参考技术方案 §6.1.3。
package config

import (
	"errors"
	"fmt"
	"os"
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
	RunMode         string        `mapstructure:"run_mode"`
	Mode            string        `mapstructure:"mode"` // debug / release / test
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

const (
	RunModeAll    = "all"
	RunModeAPI    = "api"
	RunModeWorker = "worker"
)

// Addr 返回监听地址。
func (s ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// EffectiveRunMode 返回规范化后的运行模式。
func (s ServerConfig) EffectiveRunMode() string {
	return NormalizeRunMode(s.RunMode)
}

// APIEnabled 返回当前运行模式是否包含 HTTP API 运行面。
func (s ServerConfig) APIEnabled() bool {
	switch s.EffectiveRunMode() {
	case RunModeAll, RunModeAPI:
		return true
	default:
		return false
	}
}

// WorkerEnabled 返回当前运行模式是否包含 Worker 运行面。
func (s ServerConfig) WorkerEnabled() bool {
	switch s.EffectiveRunMode() {
	case RunModeAll, RunModeWorker:
		return true
	default:
		return false
	}
}

// ValidateRunMode 校验运行模式是否合法。
func (s ServerConfig) ValidateRunMode() error {
	switch s.EffectiveRunMode() {
	case RunModeAll, RunModeAPI, RunModeWorker:
		return nil
	default:
		return fmt.Errorf(
			"invalid server.run_mode %q: must be one of %s, %s, %s",
			s.RunMode,
			RunModeAll,
			RunModeAPI,
			RunModeWorker,
		)
	}
}

// NormalizeRunMode 返回规范化后的运行模式；空值默认 all。
func NormalizeRunMode(mode string) string {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	if normalized == "" {
		return RunModeAll
	}

	return normalized
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

// ModelProviderConfig 单个模型供应商配置。
type ModelProviderConfig struct {
	Provider            string        `mapstructure:"provider"`
	ModelProvider       string        `mapstructure:"model_provider"`
	Model               string        `mapstructure:"model"`
	ModelName           string        `mapstructure:"model_name"`
	APIKey              string        `mapstructure:"api_key"`
	BaseURL             string        `mapstructure:"base_url"`
	BaseURLNoUnderscore string        `mapstructure:"baseurl"`
	Timeout             time.Duration `mapstructure:"timeout"`
	Temperature         float64       `mapstructure:"temperature"`
	MaxTokens           int           `mapstructure:"max_tokens"`
	MaxRetries          int           `mapstructure:"max_retries"`
	RetryInterval       time.Duration `mapstructure:"retry_interval"`
}

// EffectiveModel 返回最终模型名，兼容 model_name 与 model 两种配置键。
func (m ModelProviderConfig) EffectiveModel() string {
	if modelName := strings.TrimSpace(m.ModelName); modelName != "" {
		return modelName
	}

	return strings.TrimSpace(m.Model)
}

// EffectiveBaseURL 返回最终模型服务地址，兼容 baseurl 与 base_url 两种配置键。
func (m ModelProviderConfig) EffectiveBaseURL() string {
	if baseURL := strings.TrimSpace(m.BaseURLNoUnderscore); baseURL != "" {
		return baseURL
	}

	return strings.TrimSpace(m.BaseURL)
}

// LLMConfig 大模型配置。
//
// v5 只支持 models[] 显式模型列表；调用顺序严格等于配置声明顺序。
type LLMConfig struct {
	Models []ModelProviderConfig `mapstructure:"models"`
}

// EffectiveModels returns configured models in declaration order.
func (c LLMConfig) EffectiveModels() []ModelProviderConfig {
	return filterConfiguredModels(c.Models)
}

func filterConfiguredModels(models []ModelProviderConfig) []ModelProviderConfig {
	out := make([]ModelProviderConfig, 0, len(models))
	for _, model := range models {
		if strings.TrimSpace(model.Provider) == "" && model.EffectiveModel() == "" && model.EffectiveBaseURL() == "" {
			continue
		}

		out = append(out, model)
	}

	return out
}

// EmbeddingConfig Embedding 模型配置。
type EmbeddingConfig struct {
	Provider            string `mapstructure:"provider"`
	Model               string `mapstructure:"model"`
	ModelName           string `mapstructure:"model_name"`
	APIKey              string `mapstructure:"api_key"`
	BaseURL             string `mapstructure:"base_url"`
	BaseURLNoUnderscore string `mapstructure:"baseurl"`
	Dimensions          int    `mapstructure:"dimensions"`
}

// EffectiveModel 返回最终 Embedding 模型名，兼容 model_name 与 model 两种配置键。
func (e EmbeddingConfig) EffectiveModel() string {
	if modelName := strings.TrimSpace(e.ModelName); modelName != "" {
		return modelName
	}

	return strings.TrimSpace(e.Model)
}

// EffectiveBaseURL 返回最终 Embedding 服务地址，兼容 baseurl 与 base_url 两种配置键。
func (e EmbeddingConfig) EffectiveBaseURL() string {
	if baseURL := strings.TrimSpace(e.BaseURLNoUnderscore); baseURL != "" {
		return baseURL
	}

	return strings.TrimSpace(e.BaseURL)
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

	if err := bindEnvironment(v); err != nil {
		return nil, err
	}

	if err := v.ReadInConfig(); err != nil {
		if isConfigMissing(err) {
			return nil, fmt.Errorf(
				"read config %q: %w; create local runtime config with `cp configs/config.example.yaml configs/config.yaml`, then set llm.models[].base_url, model, and api_key",
				configPath,
				err,
			)
		}

		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	cfg.Server.RunMode = cfg.Server.EffectiveRunMode()
	if err := cfg.Server.ValidateRunMode(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func isConfigMissing(err error) bool {
	var notFound viper.ConfigFileNotFoundError

	return errors.As(err, &notFound) || os.IsNotExist(err)
}

func bindEnvironment(v *viper.Viper) error {
	keys := []string{
		"server.run_mode",
		"llm.models",
		"embedding.provider",
		"embedding.model",
		"embedding.model_name",
		"embedding.api_key",
		"embedding.base_url",
		"embedding.baseurl",
		"embedding.dimensions",
	}
	for _, key := range keys {
		if err := v.BindEnv(key); err != nil {
			return fmt.Errorf("bind env %s: %w", key, err)
		}
	}

	return nil
}
