package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/Gospoduu/graphs-service/internal/config"
)

func NewPostgresDB(cfg *config.PostgresConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.GetDSN()), &gorm.Config{})
	if err != nil {
		err = fmt.Errorf("failed to connect to postgres: %w", err)
		return nil, err
	}
	return db, nil
}
