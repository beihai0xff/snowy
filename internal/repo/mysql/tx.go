package mysql

import (
	"context"
	"errors"

	"gorm.io/gorm"

	irepo "github.com/beihai0xff/snowy/internal/repo"
)

type txContextKey struct{}

// WithTx 将 GORM 事务句柄写入 context，供 repository 自动复用。
func WithTx(ctx context.Context, tx *gorm.DB) context.Context {
	if tx == nil {
		return ctx
	}

	return context.WithValue(ctx, txContextKey{}, tx)
}

func txFromContext(ctx context.Context) *gorm.DB {
	tx, _ := ctx.Value(txContextKey{}).(*gorm.DB)

	return tx
}

func dbFromContext(ctx context.Context, base *gorm.DB) *gorm.DB {
	if tx := txFromContext(ctx); tx != nil {
		return tx.WithContext(ctx)
	}

	return base.WithContext(ctx)
}

type transactor struct {
	db *gorm.DB
}

// NewTransactor 创建基于 GORM 的事务执行器。
func NewTransactor(db *gorm.DB) irepo.Transactor {
	return &transactor{db: db}
}

// Transaction 在一个事务中执行回调；若 context 已携带事务，则复用该事务而不再嵌套开启。
func (t *transactor) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return Transaction(ctx, t.db, fn)
}

// Transaction 在一个事务中执行回调；若 context 已携带事务，则复用该事务而不再嵌套开启。
func Transaction(ctx context.Context, db *gorm.DB, fn func(ctx context.Context) error) error {
	if fn == nil {
		return nil
	}

	if tx := txFromContext(ctx); tx != nil {
		return fn(ctx)
	}

	if db == nil {
		return errors.New("transaction db is nil")
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(WithTx(ctx, tx))
	})
}
