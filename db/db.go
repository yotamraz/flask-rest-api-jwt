package db

import (
	"flask-rest-api-jwt/models"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Connect initialises a GORM database connection.
// In production (appEnv == "production") it uses the PostgreSQL driver;
// otherwise it falls back to SQLite for development and testing.
// AutoMigrate is run on all models to ensure the schema is up to date.
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
