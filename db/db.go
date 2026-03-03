package db

import (
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"flask-rest-api-jwt/models"
)

// Connect establishes a database connection based on the environment.
// For production, it uses PostgreSQL; otherwise SQLite.
func Connect(dsn string, appEnv string) (*gorm.DB, error) {
	var dialector gorm.Dialector

	if appEnv == "production" {
		dialector = postgres.Open(dsn)
	} else {
		// Strip sqlite:/// prefix if present for GORM compatibility
		cleanDSN := dsn
		if strings.HasPrefix(cleanDSN, "sqlite:///") {
			cleanDSN = strings.TrimPrefix(cleanDSN, "sqlite:///")
		} else if strings.HasPrefix(cleanDSN, "sqlite://") {
			cleanDSN = strings.TrimPrefix(cleanDSN, "sqlite://")
		}
		dialector = sqlite.Open(cleanDSN)
	}

	database, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// AutoMigrate all models
	if err := database.AutoMigrate(
		&models.User{},
		&models.Store{},
		&models.Item{},
		&models.Tag{},
	); err != nil {
		return nil, err
	}

	return database, nil
}
