package handlers

import (
	"net/http"
	"strconv"

	"flask-rest-api-jwt/middleware"
	"flask-rest-api-jwt/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UserHandler contains endpoints for user registration, login, logout,
// token refresh, and user CRUD operations.
type UserHandler struct {
	DB         *gorm.DB
	JWTManager *middleware.JWTManager
}

// Register handles POST /user/register.
// Creates a new user with a hashed password. Returns 201 on success,
// 400 if the username already exists.
func (h *UserHandler) Register(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid input"})
		return
	}

	// Check for duplicate username.
	var existing models.User
	if err := h.DB.Where("username = ?", input.Username).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "User exists"})
		return
	}

	hash, err := middleware.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to hash password"})
		return
	}

	user := models.User{
		Username:     input.Username,
		PasswordHash: hash,
	}
	if err := h.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create user"})
		return
	}

	// Return user object (password_hash excluded via json:"-" tag).
	c.JSON(http.StatusCreated, gin.H{
		"id":       user.ID,
		"username": user.Username,
	})
}

// Login handles POST /user/login.
// Validates credentials and returns access + refresh tokens on success.
func (h *UserHandler) Login(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid credentials"})
		return
	}

	var user models.User
	if err := h.DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid credentials"})
		return
	}

	if err := middleware.CheckPassword(user.PasswordHash, input.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid credentials"})
		return
	}

	accessToken, err := h.JWTManager.CreateAccessToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create token"})
		return
	}

	refreshToken, err := h.JWTManager.CreateRefreshToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// Logout handles POST /user/logout.
// Blacklists the current access token's JTI and returns a confirmation.
func (h *UserHandler) Logout(c *gin.Context) {
	jti, ok := middleware.GetJTI(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Missing token identity"})
		return
	}

	h.JWTManager.BlacklistToken(jti)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

// Refresh handles POST /user/refresh.
// Issues a new access token using a valid refresh token.
func (h *UserHandler) Refresh(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Missing user identity"})
		return
	}

	accessToken, err := h.JWTManager.CreateAccessToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
	})
}

// GetUser handles GET /user/:id.
// Returns the user object if the authenticated user matches the requested ID.
func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid user ID"})
		return
	}

	currentUserID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Missing user identity"})
		return
	}

	var user models.User
	if err := h.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	if currentUserID != user.ID {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"username": user.Username,
	})
}

// DeleteUser handles DELETE /user/:id.
// Deletes the user (and cascades to their stores/items/tags) if the
// authenticated user matches the requested ID.
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid user ID"})
		return
	}

	currentUserID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Missing user identity"})
		return
	}

	var user models.User
	if err := h.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	if currentUserID != user.ID {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	// Use Select to cascade delete associated stores (which in turn cascade to items/tags).
	if err := h.DB.Select("Stores").Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}
