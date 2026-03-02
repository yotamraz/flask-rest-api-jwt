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
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	database, err := db.Connect(cfg.DatabaseDSN(), cfg.AppEnv)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	jwtManager := middleware.NewJWTManager(cfg.SecretKey)

	r := gin.Default()
	handlers.RegisterRoutes(r, database, jwtManager)

	log.Printf("Starting server on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
