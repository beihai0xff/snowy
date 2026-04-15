// Package mysql 提供 MySQL 连接池与 Repository 实现（基础设施层）。
package mysql

import (
	"context"
	"fmt"
	"time"

	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/beihai0xff/snowy/internal/pkg/config"
)

// NewDB 创建 GORM MySQL 客户端，并配置底层连接池。
func NewDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(gormmysql.New(gormmysql.Config{DSN: cfg.DSN()}), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db from gorm: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()

		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	return db, nil
}
