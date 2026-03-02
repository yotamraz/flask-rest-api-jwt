package handlers

import (
	"flask-rest-api-jwt/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes configures all route groups on the Gin engine,
// analogous to Flask blueprint registration.
func RegisterRoutes(r *gin.Engine, db *gorm.DB, jwtMgr *middleware.JWTManager) {
	// ---- Health ----
	healthHandler := &HealthHandler{DB: db}
	health := r.Group("/health")
	{
		health.GET("/", healthHandler.HealthCheck)
	}

	// ---- Users ----
	userHandler := &UserHandler{DB: db, JWTManager: jwtMgr}
	user := r.Group("/user")
	{
		user.POST("/register", userHandler.Register)
		user.POST("/login", userHandler.Login)
		user.POST("/logout", jwtMgr.AuthRequired(), userHandler.Logout)
		user.POST("/refresh", jwtMgr.RefreshRequired(), userHandler.Refresh)
		user.GET("/:id", jwtMgr.AuthRequired(), userHandler.GetUser)
		user.DELETE("/:id", jwtMgr.AuthRequired(), userHandler.DeleteUser)
	}
}
