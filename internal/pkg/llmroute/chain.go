package llmroute

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/beihai0xff/snowy/internal/repo/llm"
)

// Chain is an llm.Provider that tries multiple providers in configured order.
// It keeps the business layer dependent on a single Provider while preserving
// model-level failover and error visibility.
type Chain struct {
	name      string
	providers []llm.Provider
}

// RetryingProvider wraps one provider with same-model retry semantics.
type RetryingProvider struct {
	next          llm.Provider
	maxRetries    int
	retryInterval time.Duration
}

// NewRetryingProvider returns provider decorated with retry. Non-positive retries keep provider unchanged.
func NewRetryingProvider(provider llm.Provider, maxRetries int, retryInterval time.Duration) llm.Provider {
	if provider == nil || maxRetries <= 0 {
		return provider
	}

	if retryInterval <= 0 {
		retryInterval = time.Second
	}

	return &RetryingProvider{next: provider, maxRetries: maxRetries, retryInterval: retryInterval}
}

func (p *RetryingProvider) Generate(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		resp, err := p.next.Generate(ctx, cloneRequest(req))
		if err == nil {
			return resp, nil
		}

		lastErr = err
		if !llm.IsRetryable(err) || attempt == p.maxRetries {
			break
		}

		if err := sleepWithContext(ctx, p.retryInterval); err != nil {
			return nil, err
		}
	}

	return nil, lastErr
}

func (p *RetryingProvider) GenerateStream(ctx context.Context, req *llm.Request, chunks chan<- llm.StreamChunk) error {
	var lastErr error

	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		err := p.next.GenerateStream(ctx, cloneRequest(req), chunks)
		if err == nil {
			return nil
		}

		lastErr = err
		if !llm.IsRetryable(err) || attempt == p.maxRetries {
			break
		}

		if err := sleepWithContext(ctx, p.retryInterval); err != nil {
			return err
		}
	}

	return lastErr
}

func (p *RetryingProvider) HealthCheck(ctx context.Context) error { return p.next.HealthCheck(ctx) }
func (p *RetryingProvider) EstimateCost(ctx context.Context, req *llm.Request) (*llm.Cost, error) {
	return p.next.EstimateCost(ctx, req)
}
func (p *RetryingProvider) Name() string            { return p.next.Name() }
func (p *RetryingProvider) ConfiguredModel() string { return providerModel(p.next) }
func (p *RetryingProvider) ConfiguredBaseURL() string {
	if configured, ok := p.next.(llm.ConfiguredProvider); ok {
		return configured.ConfiguredBaseURL()
	}

	return ""
}

func (p *RetryingProvider) ConfiguredModelProvider() string {
	if configured, ok := p.next.(llm.ConfiguredProvider); ok {
		return configured.ConfiguredModelProvider()
	}

	return ""
}

// NewChain creates a provider chain. Nil providers are ignored.
func NewChain(name string, providers ...llm.Provider) llm.Provider {
	chain := &Chain{name: strings.TrimSpace(name)}
	if chain.name == "" {
		chain.name = "ordered-model-chain"
	}

	for _, provider := range providers {
		if provider != nil {
			chain.providers = append(chain.providers, provider)
		}
	}

	if len(chain.providers) == 1 {
		return chain.providers[0]
	}

	return chain
}

func (c *Chain) Generate(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	if c == nil || len(c.providers) == 0 {
		return nil, errors.New("llm provider chain is empty")
	}

	failures := make([]string, 0, len(c.providers))
	for _, provider := range c.providers {
		if provider == nil {
			continue
		}

		attempt := cloneRequest(req)
		if attempt != nil && strings.TrimSpace(attempt.Model) == "" {
			attempt.Model = providerModel(provider)
		}

		resp, err := provider.Generate(ctx, attempt)
		if err == nil {
			return resp, nil
		}

		failures = append(failures, fmt.Sprintf("%s: %v", provider.Name(), err))
	}

	if len(failures) == 0 {
		return nil, errors.New("llm provider chain has no runnable provider")
	}

	return nil, errors.New(strings.Join(failures, "; "))
}

func (c *Chain) GenerateStream(ctx context.Context, req *llm.Request, chunks chan<- llm.StreamChunk) error {
	if c == nil || len(c.providers) == 0 {
		return errors.New("llm provider chain is empty")
	}

	failures := make([]string, 0, len(c.providers))
	for _, provider := range c.providers {
		if provider == nil {
			continue
		}

		attempt := cloneRequest(req)
		if attempt != nil && strings.TrimSpace(attempt.Model) == "" {
			attempt.Model = providerModel(provider)
		}

		err := provider.GenerateStream(ctx, attempt, chunks)
		if err == nil {
			return nil
		}

		failures = append(failures, fmt.Sprintf("%s: %v", provider.Name(), err))
	}

	if len(failures) == 0 {
		return errors.New("llm provider chain has no runnable provider")
	}

	return errors.New(strings.Join(failures, "; "))
}

func (c *Chain) HealthCheck(ctx context.Context) error {
	if c == nil || len(c.providers) == 0 {
		return errors.New("llm provider chain is empty")
	}

	failures := make([]string, 0, len(c.providers))
	for _, provider := range c.providers {
		if provider == nil {
			continue
		}

		err := provider.HealthCheck(ctx)
		if err == nil {
			return nil
		}

		failures = append(failures, fmt.Sprintf("%s: %v", provider.Name(), err))
	}

	return errors.New(strings.Join(failures, "; "))
}

func (c *Chain) EstimateCost(ctx context.Context, req *llm.Request) (*llm.Cost, error) {
	if c == nil || len(c.providers) == 0 {
		return &llm.Cost{}, nil
	}

	for _, provider := range c.providers {
		if provider != nil {
			return provider.EstimateCost(ctx, req)
		}
	}

	return &llm.Cost{}, nil
}

func (c *Chain) Name() string {
	if c == nil || c.name == "" {
		return "ordered-model-chain"
	}

	return c.name
}

func (c *Chain) ConfiguredModel() string {
	if c == nil {
		return ""
	}

	for _, provider := range c.providers {
		if model := providerModel(provider); model != "" {
			return model
		}
	}

	return ""
}

func (c *Chain) ConfiguredBaseURL() string {
	if c == nil {
		return ""
	}

	for _, provider := range c.providers {
		if configured, ok := provider.(llm.ConfiguredProvider); ok {
			if baseURL := strings.TrimSpace(configured.ConfiguredBaseURL()); baseURL != "" {
				return baseURL
			}
		}
	}

	return ""
}

func (c *Chain) ConfiguredModelProvider() string {
	if c == nil {
		return ""
	}

	for _, provider := range c.providers {
		if configured, ok := provider.(llm.ConfiguredProvider); ok {
			if modelProvider := strings.TrimSpace(configured.ConfiguredModelProvider()); modelProvider != "" {
				return modelProvider
			}
		}
	}

	return ""
}

func cloneRequest(req *llm.Request) *llm.Request {
	if req == nil {
		return nil
	}

	cloned := *req
	if req.Messages != nil {
		cloned.Messages = append([]llm.Message(nil), req.Messages...)
	}

	return &cloned
}

func providerModel(provider llm.Provider) string {
	if configured, ok := provider.(llm.ConfiguredProvider); ok {
		return strings.TrimSpace(configured.ConfiguredModel())
	}

	return ""
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
