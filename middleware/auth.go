package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	// TokenTypeAccess identifies an access token.
	TokenTypeAccess = "access"
	// TokenTypeRefresh identifies a refresh token.
	TokenTypeRefresh = "refresh"

	// AccessTokenExpiry is the lifetime of an access token.
	AccessTokenExpiry = 15 * time.Minute
	// RefreshTokenExpiry is the lifetime of a refresh token.
	RefreshTokenExpiry = 30 * 24 * time.Hour
)

// Claims represents the custom JWT claims used by this application.
type Claims struct {
	jwt.RegisteredClaims
	TokenType string `json:"type"`
	UserID    uint   `json:"user_id"`
}

// JWTManager handles JWT token creation, validation, and blacklisting.
type JWTManager struct {
	secretKey []byte
	blacklist sync.Map // stores revoked JTIs
}

// NewJWTManager creates a new JWTManager with the given secret key.
func NewJWTManager(secretKey string) *JWTManager {
	return &JWTManager{
		secretKey: []byte(secretKey),
	}
}

// CreateAccessToken creates a new JWT access token for the given user ID.
func (m *JWTManager) CreateAccessToken(userID uint) (string, error) {
	return m.createToken(userID, TokenTypeAccess, AccessTokenExpiry)
}

// CreateRefreshToken creates a new JWT refresh token for the given user ID.
func (m *JWTManager) CreateRefreshToken(userID uint) (string, error) {
	return m.createToken(userID, TokenTypeRefresh, RefreshTokenExpiry)
}

func (m *JWTManager) createToken(userID uint, tokenType string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		TokenType: tokenType,
		UserID:    userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// ParseToken parses and validates a JWT token string and returns its claims.
func (m *JWTManager) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return m.secretKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	return claims, nil
}

// BlacklistToken adds a token's JTI to the blacklist.
func (m *JWTManager) BlacklistToken(jti string) {
	m.blacklist.Store(jti, struct{}{})
}

// IsBlacklisted checks if a token's JTI has been blacklisted.
func (m *JWTManager) IsBlacklisted(jti string) bool {
	_, exists := m.blacklist.Load(jti)
	return exists
}

// RequireAuth returns a Gin middleware that requires a valid access token.
func (m *JWTManager) RequireAuth() gin.HandlerFunc {
	return m.requireToken(TokenTypeAccess)
}

// RequireRefresh returns a Gin middleware that requires a valid refresh token.
func (m *JWTManager) RequireRefresh() gin.HandlerFunc {
	return m.requireToken(TokenTypeRefresh)
}

func (m *JWTManager) requireToken(expectedType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"description": "Request does not contain an access token.",
				"error":       "authorization_required",
				"status_code": 401,
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"description": "Request does not contain an access token.",
				"error":       "authorization_required",
				"status_code": 401,
			})
			return
		}

		tokenString := parts[1]
		claims, err := m.ParseToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"description": "Signature verification failed.",
				"error":       "invalid_token",
				"status_code": 401,
			})
			return
		}

		if claims.TokenType != expectedType {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"description": "Only " + expectedType + " tokens are allowed.",
				"error":       "invalid_token",
				"status_code": 401,
			})
			return
		}

		if m.IsBlacklisted(claims.ID) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"description": "Token has been revoked.",
				"error":       "token_revoked",
				"status_code": 401,
			})
			return
		}

		// Store user identity and token claims in context
		c.Set("user_id", claims.UserID)
		c.Set("jwt_claims", claims)
		c.Next()
	}
}

// GetUserID extracts the authenticated user's ID from the Gin context.
func GetUserID(c *gin.Context) (uint, bool) {
	val, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	userID, ok := val.(uint)
	return userID, ok
}

// GetJWTClaims extracts the JWT claims from the Gin context.
func GetJWTClaims(c *gin.Context) (*Claims, bool) {
	val, exists := c.Get("jwt_claims")
	if !exists {
		return nil, false
	}
	claims, ok := val.(*Claims)
	return claims, ok
}

// HashPassword hashes a plaintext password using bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword compares a bcrypt hash with a plaintext password.
func CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
