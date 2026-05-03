package mysql

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/beihai0xff/snowy/internal/modeling/generative"
)

type generativeModelPackageRepo struct{ db *gorm.DB }

func NewGenerativeModelPackageRepository(db *gorm.DB) generative.Repository {
	return &generativeModelPackageRepo{db: db}
}

func (r *generativeModelPackageRepo) Save(ctx context.Context, pkg *generative.GenerativeModelPackage) error {
	if pkg == nil {
		return fmt.Errorf("generative model package is nil")
	}
	return dbFromContext(ctx, r.db).WithContext(ctx).Create(newGenerativeModelPackageRow(pkg)).Error
}

func (r *generativeModelPackageRepo) GetByID(ctx context.Context, id uuid.UUID) (*generative.GenerativeModelPackage, error) {
	row := &generativeModelPackageRow{}
	if err := dbFromContext(ctx, r.db).WithContext(ctx).Where("id = ?", id).First(row).Error; err != nil {
		return nil, fmt.Errorf("get generative model package: %w", err)
	}
	return row.toDomain(), nil
}
