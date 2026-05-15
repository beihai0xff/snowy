package generative

import (
	"context"

	"github.com/google/uuid"
)

// Repository persists generated model packages.
type Repository interface {
	Save(ctx context.Context, pkg *GenerativeModelPackage) error
	GetByID(ctx context.Context, id uuid.UUID) (*GenerativeModelPackage, error)
	ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*GenerativeModelPackage, int64, error)
}
