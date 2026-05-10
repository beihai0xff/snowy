package llmroute

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/beihai0xff/snowy/internal/repo/llm"
)

type fakeProvider struct {
	name     string
	model    string
	calls    int
	generate func(*llm.Request) (*llm.Response, error)
}

func (f *fakeProvider) Generate(_ context.Context, req *llm.Request) (*llm.Response, error) {
	f.calls++
	if f.generate != nil {
		return f.generate(req)
	}
	return &llm.Response{Content: "ok", Model: req.Model}, nil
}
func (f *fakeProvider) GenerateStream(context.Context, *llm.Request, chan<- llm.StreamChunk) error {
	return nil
}
func (f *fakeProvider) HealthCheck(context.Context) error { return nil }
func (f *fakeProvider) EstimateCost(context.Context, *llm.Request) (*llm.Cost, error) {
	return &llm.Cost{}, nil
}
func (f *fakeProvider) Name() string                    { return f.name }
func (f *fakeProvider) ConfiguredModel() string         { return f.model }
func (f *fakeProvider) ConfiguredBaseURL() string       { return "" }
func (f *fakeProvider) ConfiguredModelProvider() string { return "" }

func TestChainFallsBackAndInjectsProviderModel(t *testing.T) {
	primary := &fakeProvider{name: "primary", model: "m1", generate: func(*llm.Request) (*llm.Response, error) {
		return nil, errors.New("down")
	}}
	fallback := &fakeProvider{name: "fallback", model: "m2", generate: func(req *llm.Request) (*llm.Response, error) {
		assert.Equal(t, "m2", req.Model)
		return &llm.Response{Content: "ok", Model: req.Model}, nil
	}}

	provider := NewChain("test", primary, fallback)
	resp, err := provider.Generate(context.Background(), &llm.Request{})

	require.NoError(t, err)
	assert.Equal(t, "m2", resp.Model)
	assert.Equal(t, 1, primary.calls)
	assert.Equal(t, 1, fallback.calls)
}

func TestRetryingProviderRetriesOnlyRetryableErrors(t *testing.T) {
	base := &fakeProvider{name: "base", model: "m1", generate: func(req *llm.Request) (*llm.Response, error) {
		return nil, llm.NewProviderError("temporary", true)
	}}
	provider := NewRetryingProvider(base, 2, time.Millisecond)

	_, err := provider.Generate(context.Background(), &llm.Request{})

	require.Error(t, err)
	assert.Equal(t, 3, base.calls)
}
