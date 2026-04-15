// Package mysql 提供 MySQL 连接池与 Repository 实现（基础设施层）。
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// Register the MySQL database/sql driver.
	_ "github.com/go-sql-driver/mysql"

	"github.com/beihai0xff/snowy/internal/pkg/config"
)

// NewDB 创建 MySQL 连接池。
func NewDB(cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	return db, nil
}
