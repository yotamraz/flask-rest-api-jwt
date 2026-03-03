package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthHandler handles health check endpoints.
type HealthHandler struct {
	DB *gorm.DB
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{DB: db}
}

// HealthCheck performs a liveness/readiness probe.
// GET /health/
// Returns 200 with {"status": "healthy", "database": "healthy"} when DB is reachable.
// Returns 503 with {"status": "unhealthy", "database": "unhealthy"} when DB is not reachable.
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	sqlDB, err := h.DB.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "unhealthy",
			"database": "unhealthy",
		})
		return
	}

	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "unhealthy",
			"database": "unhealthy",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "healthy",
		"database": "healthy",
	})
}

// RegisterHealthRoutes registers health check routes.
func (h *HealthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/", h.HealthCheck)
}
