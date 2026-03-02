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

const testSecret = "test-secret-key"

// setupTestDB creates an in-memory SQLite database with all models migrated.
func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic("failed to open test database: " + err.Error())
	}
	if err := db.AutoMigrate(&models.User{}, &models.Store{}, &models.Item{}, &models.Tag{}); err != nil {
		panic("failed to migrate test database: " + err.Error())
	}
	return db
}

// setupTestRouter creates a Gin engine with all routes registered for testing.
func setupTestRouter(db *gorm.DB) (*gin.Engine, *middleware.JWTManager) {
	jwtMgr := middleware.NewJWTManager(testSecret)
	r := gin.New() // No logger/recovery for cleaner test output
	RegisterRoutes(r, db, jwtMgr)
	return r, jwtMgr
}
