package generative

import (
	"context"

	"github.com/google/uuid"
)

// Service compiles a user question into a v4 generative model package.
type Service interface {
	Compile(ctx context.Context, req *CompileRequest) (*GenerativeModelPackage, error)
	GetPackage(ctx context.Context, id string) (*GenerativeModelPackage, error)
	ListPackages(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*GenerativeModelPackage, int64, error)
	// Recompute v7 §3：基于 overrides 把变量默认值与 outcomes 重算后产出新快照（不调用 LLM）。
	Recompute(ctx context.Context, packageID string, overrides map[string]float64) (*GenerativeModelPackage, error)
	// Regenerate v7 §3：以 parentID 为上下文，按 reason + ctx 增量调用 LLM 重新生成。
	Regenerate(
		ctx context.Context,
		parentID string,
		reason string,
		hint CompileContext,
	) (*GenerativeModelPackage, error)
}
