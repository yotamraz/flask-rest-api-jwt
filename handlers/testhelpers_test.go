package handlers

import (
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"flask-rest-api-jwt/middleware"
	"flask-rest-api-jwt/models"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupTestDB creates an in-memory SQLite database with all models migrated.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	err = db.AutoMigrate(&models.User{}, &models.Store{}, &models.Item{}, &models.Tag{})
	if err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}

	// Clean tables for test isolation
	db.Exec("DELETE FROM items")
	db.Exec("DELETE FROM tags")
	db.Exec("DELETE FROM stores")
	db.Exec("DELETE FROM users")

	return db
}

// setupTestRouter creates a Gin engine with all routes registered for testing.
func setupTestRouter(t *testing.T) (*gin.Engine, *gorm.DB, *middleware.JWTManager) {
	t.Helper()
	db := setupTestDB(t)
	jwtManager := middleware.NewJWTManager("test-secret-key")

	r := gin.New() // Use gin.New() instead of gin.Default() to avoid logger noise in tests
	RegisterRoutes(r, db, jwtManager)

	return r, db, jwtManager
}

// createTestUser creates a user in the database and returns the user model.
func createTestUser(t *testing.T, db *gorm.DB, username, password string) models.User {
	t.Helper()
	hash, err := middleware.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := models.User{
		Username:     username,
		PasswordHash: hash,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

// getAccessToken creates a user, logs them in, and returns their access token.
func getAccessToken(t *testing.T, jwtManager *middleware.JWTManager, userID uint) string {
	t.Helper()
	token, err := jwtManager.CreateAccessToken(userID)
	if err != nil {
		t.Fatalf("failed to create access token: %v", err)
	}
	return token
}

// getRefreshToken creates a refresh token for the given user ID.
func getRefreshToken(t *testing.T, jwtManager *middleware.JWTManager, userID uint) string {
	t.Helper()
	token, err := jwtManager.CreateRefreshToken(userID)
	if err != nil {
		t.Fatalf("failed to create refresh token: %v", err)
	}
	return token
}
