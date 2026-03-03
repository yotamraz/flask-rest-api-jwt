package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"flask-rest-api-jwt/config"
	"flask-rest-api-jwt/db"
	"flask-rest-api-jwt/handlers"
	"flask-rest-api-jwt/middleware"
)

func main() {
	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Connect to database
	database, err := db.Connect(cfg.DatabaseDSN(), cfg.AppEnv)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}

	// Initialize JWT manager
	jwtManager := middleware.NewJWTManager(cfg.SecretKey)

	// Create Gin router
	r := gin.Default()

	// Register all routes
	handlers.RegisterRoutes(r, database, jwtManager)

	// Start the HTTP server
	log.Printf("Starting server on port %s (env: %s)", cfg.Port, cfg.AppEnv)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
