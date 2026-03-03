package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"flask-rest-api-jwt/middleware"
)

// RegisterRoutes sets up all route groups on the Gin engine.
// This mirrors the Flask blueprint registration in app/__init__.py.
func RegisterRoutes(r *gin.Engine, db *gorm.DB, jwtManager *middleware.JWTManager) {
	// Health check endpoints - /health/
	healthHandler := NewHealthHandler(db)
	healthGroup := r.Group("/health")
	healthHandler.RegisterRoutes(healthGroup)

	// User endpoints - /user/
	userHandler := NewUserHandler(db, jwtManager)
	userGroup := r.Group("/user")
	userHandler.RegisterRoutes(userGroup)
}
