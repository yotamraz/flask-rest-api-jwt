package handlers

import (
	"flask-rest-api-jwt/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes sets up all route groups and handlers on the Gin engine.
func RegisterRoutes(r *gin.Engine, db *gorm.DB, jwtManager *middleware.JWTManager) {
	// Health endpoints
	healthHandler := NewHealthHandler(db)
	healthGroup := r.Group("/health")
	healthHandler.RegisterRoutes(healthGroup)

	// User endpoints
	userHandler := NewUserHandler(db, jwtManager)
	userGroup := r.Group("/user")
	userHandler.RegisterRoutes(userGroup)
}
