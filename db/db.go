package db

import (
	"flask-rest-api-jwt/models"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Connect establishes a database connection using the appropriate driver
// based on the application environment. It runs AutoMigrate for all models.
func Connect(dsn string, appEnv string) (*gorm.DB, error) {
	var dialector gorm.Dialector

	if appEnv == "production" {
		dialector = postgres.Open(dsn)
	} else {
		dialector = sqlite.Open(dsn)
	}

	database, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Enable foreign key enforcement for SQLite
	if appEnv != "production" {
		sqlDB, err := database.DB()
		if err != nil {
			return nil, err
		}
		_, err = sqlDB.Exec("PRAGMA foreign_keys = ON")
		if err != nil {
			return nil, err
		}
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
