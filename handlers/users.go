package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"flask-rest-api-jwt/middleware"
	"flask-rest-api-jwt/models"
)

// UserHandler handles user-related endpoints.
type UserHandler struct {
	DB  *gorm.DB
	JWT *middleware.JWTManager
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(db *gorm.DB, jwtManager *middleware.JWTManager) *UserHandler {
	return &UserHandler{DB: db, JWT: jwtManager}
}

// Register creates a new user.
// POST /user/register
// Request: {"username": "...", "password": "..."}
// Response 201: {"id": int, "username": string}
// Response 400: {"message": "User exists"}
func (h *UserHandler) Register(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Username and password are required"})
		return
	}

	// Check if user already exists
	var existing models.User
	if err := h.DB.Where("username = ?", input.Username).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "User exists"})
		return
	}

	// Hash password
	hash, err := middleware.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error creating user"})
		return
	}

	user := models.User{
		Username:     input.Username,
		PasswordHash: hash,
	}

	if err := h.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error creating user"})
		return
	}

	// Return user data (matching Flask's UserSchema which excludes password_hash)
	c.JSON(http.StatusCreated, gin.H{
		"id":       user.ID,
		"username": user.Username,
	})
}

// Login authenticates a user and returns JWT tokens.
// POST /user/login
// Request: {"username": "...", "password": "..."}
// Response 200: {"access_token": "...", "refresh_token": "..."}
// Response 401: {"message": "Invalid credentials"}
func (h *UserHandler) Login(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid credentials"})
		return
	}

	// Find user
	var user models.User
	if err := h.DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid credentials"})
		return
	}

	// Check password
	if !middleware.CheckPassword(user.PasswordHash, input.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid credentials"})
		return
	}

	// Create tokens
	accessToken, err := h.JWT.CreateAccessToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error creating token"})
		return
	}

	refreshToken, err := h.JWT.CreateRefreshToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error creating token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// Logout revokes the current access token.
// POST /user/logout
// Response 200: {"message": "Logged out"}
func (h *UserHandler) Logout(c *gin.Context) {
	jti := middleware.GetJTI(c)
	h.JWT.RevokeToken(jti)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

// Refresh issues a new access token using a valid refresh token.
// POST /user/refresh
// Response 200: {"access_token": "..."}
func (h *UserHandler) Refresh(c *gin.Context) {
	identity := middleware.GetUserID(c)

	userID, err := strconv.ParseUint(identity, 10, 64)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid token identity"})
		return
	}

	accessToken, err := h.JWT.CreateAccessToken(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error creating token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
	})
}

// GetUser returns user details by ID.
// GET /user/:id
// Response 200: {"id": int, "username": string}
// Response 401: {"message": "Unauthorized"}
// Response 404: 404 page not found (or {"message": "User not found"})
func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	// Check authorization - user can only access their own profile
	identity := middleware.GetUserID(c)
	currentUserID, _ := strconv.ParseUint(identity, 10, 64)
	if currentUserID != id {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	var user models.User
	if err := h.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"username": user.Username,
	})
}

// DeleteUser deletes a user and all their associated data (cascade).
// DELETE /user/:id
// Response 200: {"message": "Deleted"}
// Response 401: {"message": "Unauthorized"}
// Response 404: {"message": "User not found"}
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	// Check authorization - user can only delete their own profile
	identity := middleware.GetUserID(c)
	currentUserID, _ := strconv.ParseUint(identity, 10, 64)
	if currentUserID != id {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	var user models.User
	if err := h.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	// Delete user (cascade will handle stores, items, tags via GORM constraints)
	// We need to manually handle cascade for SQLite since it doesn't always
	// enforce FK constraints. Delete stores first to trigger their cascades.
	var stores []models.Store
	h.DB.Where("user_id = ?", user.ID).Find(&stores)
	for _, store := range stores {
		h.DB.Where("store_id = ?", store.ID).Delete(&models.Item{})
		h.DB.Where("store_id = ?", store.ID).Delete(&models.Tag{})
	}
	h.DB.Where("user_id = ?", user.ID).Delete(&models.Store{})
	h.DB.Delete(&user)

	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

// RegisterRoutes registers user routes on the given router group.
func (h *UserHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/register", h.Register)
	rg.POST("/login", h.Login)
	rg.POST("/logout", h.JWT.AuthRequired(), h.Logout)
	rg.POST("/refresh", h.JWT.RefreshRequired(), h.Refresh)
	rg.GET("/:id", h.JWT.AuthRequired(), h.GetUser)
	rg.DELETE("/:id", h.JWT.AuthRequired(), h.DeleteUser)
}
