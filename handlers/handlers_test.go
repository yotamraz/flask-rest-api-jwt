package handlers

import (
	"flask-rest-api-jwt/middleware"
	"flask-rest-api-jwt/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to test database: " + err.Error())
	}

	sqlDB, _ := db.DB()
	sqlDB.Exec("PRAGMA foreign_keys = ON")

	err = db.AutoMigrate(&models.User{}, &models.Store{}, &models.Item{}, &models.Tag{})
	if err != nil {
		panic("failed to migrate test database: " + err.Error())
	}

	return db
}

// setupTestRouter creates a Gin router with all handlers registered for testing.
func setupTestRouter(db *gorm.DB) (*gin.Engine, *middleware.JWTManager) {
	jwtManager := middleware.NewJWTManager("test-secret-key")
	r := gin.New()
	RegisterRoutes(r, db, jwtManager)
	return r, jwtManager
}
