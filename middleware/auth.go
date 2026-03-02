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
	// Token expiration durations matching Flask configuration.
	AccessTokenExpiry  = 15 * time.Minute
	RefreshTokenExpiry = 30 * 24 * time.Hour

	// Context keys used to store values in Gin context.
	ContextUserID = "user_id"
	ContextJTI    = "jti"
)

// JWTManager handles JWT token creation, validation, and blacklisting.
type JWTManager struct {
	secretKey []byte
	blacklist sync.Map // map[string]struct{} — stores revoked JTIs
}

// NewJWTManager creates a new JWTManager with the given signing secret.
func NewJWTManager(secretKey string) *JWTManager {
	return &JWTManager{
		secretKey: []byte(secretKey),
	}
}

// Claims represents the custom JWT claims used in access and refresh tokens.
type Claims struct {
	jwt.RegisteredClaims
	TokenType string `json:"type"`
	UserID    uint   `json:"user_id"`
}

// CreateAccessToken generates a signed JWT access token for the given user ID.
func (m *JWTManager) CreateAccessToken(userID uint) (string, error) {
	return m.createToken(userID, "access", AccessTokenExpiry)
}

// CreateRefreshToken generates a signed JWT refresh token for the given user ID.
func (m *JWTManager) CreateRefreshToken(userID uint) (string, error) {
	return m.createToken(userID, "refresh", RefreshTokenExpiry)
}

func (m *JWTManager) createToken(userID uint, tokenType string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			ID:        uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
		TokenType: tokenType,
		UserID:    userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// ParseToken parses and validates a JWT token string, returning its claims.
func (m *JWTManager) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secretKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// BlacklistToken adds a token's JTI to the in-memory blacklist.
func (m *JWTManager) BlacklistToken(jti string) {
	m.blacklist.Store(jti, struct{}{})
}

// IsBlacklisted checks whether a token JTI has been revoked.
func (m *JWTManager) IsBlacklisted(jti string) bool {
	_, ok := m.blacklist.Load(jti)
	return ok
}

// AuthRequired returns a Gin middleware that requires a valid access token.
func (m *JWTManager) AuthRequired() gin.HandlerFunc {
	return m.requireToken("access")
}

// RefreshRequired returns a Gin middleware that requires a valid refresh token.
func (m *JWTManager) RefreshRequired() gin.HandlerFunc {
	return m.requireToken("refresh")
}

func (m *JWTManager) requireToken(expectedType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"msg": "Missing Authorization Header",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Invalid authorization header format",
			})
			return
		}

		claims, err := m.ParseToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Invalid or expired token",
			})
			return
		}

		if claims.TokenType != expectedType {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": fmt.Sprintf("Expected %s token", expectedType),
			})
			return
		}

		if m.IsBlacklisted(claims.ID) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Token has been revoked",
			})
			return
		}

		// Store user identity and JTI in context for downstream handlers.
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextJTI, claims.ID)
		c.Next()
	}
}

// GetUserID extracts the authenticated user's ID from the Gin context.
func GetUserID(c *gin.Context) (uint, bool) {
	val, exists := c.Get(ContextUserID)
	if !exists {
		return 0, false
	}
	id, ok := val.(uint)
	return id, ok
}

// GetJTI extracts the current token's JTI from the Gin context.
func GetJTI(c *gin.Context) (string, bool) {
	val, exists := c.Get(ContextJTI)
	if !exists {
		return "", false
	}
	jti, ok := val.(string)
	return jti, ok
}

// HashPassword hashes a plaintext password using bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword verifies a plaintext password against a bcrypt hash.
// Returns nil on success, an error on mismatch.
func CheckPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
