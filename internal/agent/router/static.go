package router

import (
	"context"
	"errors"
	"strings"

	"github.com/beihai0xff/snowy/internal/pkg/config"
)

type staticRouter struct {
	primary  ModelInfo
	fallback ModelInfo
}

// NewStaticRouter 创建基于配置的静态模型路由器。
func NewStaticRouter(cfg config.LLMConfig) Router {
	return &staticRouter{
		primary: ModelInfo{
			Provider:  normalizeProvider(cfg.Primary.Provider),
			Model:     cfg.Primary.Model,
			IsPrimary: true,
		},
		fallback: ModelInfo{
			Provider:  normalizeProvider(cfg.Fallback.Provider),
			Model:     cfg.Fallback.Model,
			IsPrimary: false,
		},
	}
}

func (r *staticRouter) Route(_ context.Context, _ TaskType) (*ModelInfo, error) {
	if r.primary.Model == "" {
		return nil, errors.New("primary model is not configured")
	}

	model := r.primary

	return &model, nil
}

func (r *staticRouter) Fallback(_ context.Context, _ TaskType) (*ModelInfo, error) {
	if r.fallback.Model == "" {
		return nil, errors.New("fallback model is not configured")
	}

	model := r.fallback

	return &model, nil
}

func normalizeProvider(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	switch provider {
	case "google":
		return "gemini"
	default:
		return provider
	}
}
