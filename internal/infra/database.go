// Package infra provides shared infrastructure components used across all slices.
package infra

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// NewDB opens a GORM MySQL connection using the provided DSN.
// The DSN must include parseTime=true and charset=utf8mb4 parameters.
func NewDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("infra.NewDB: open: %w", err)
	}

	return db, nil
}
