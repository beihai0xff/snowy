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
}
