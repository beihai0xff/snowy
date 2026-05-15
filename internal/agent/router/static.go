package router

import (
	"context"
	"errors"
	"strings"

	"github.com/beihai0xff/snowy/internal/pkg/config"
)

type staticRouter struct {
	models []ModelInfo
}

// NewStaticRouter 创建基于配置声明顺序的静态模型路由器。
func NewStaticRouter(cfg config.LLMConfig) Router {
	configuredModels := cfg.EffectiveModels()

	models := make([]ModelInfo, 0, len(configuredModels))
	for i, model := range configuredModels {
		models = append(models, ModelInfo{
			Provider: normalizeProvider(model.Provider),
			Model:    model.EffectiveModel(),
			Order:    i + 1,
		})
	}

	return &staticRouter{models: models}
}

func (r *staticRouter) Route(_ context.Context, _ TaskType) (*ModelInfo, error) {
	if len(r.models) == 0 || r.models[0].Model == "" {
		return nil, errors.New("model list is not configured")
	}

	model := r.models[0]

	return &model, nil
}

func normalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}
