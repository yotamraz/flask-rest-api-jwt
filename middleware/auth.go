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
	// Token expiration durations matching Flask-JWT-Extended defaults
	AccessTokenExpiry  = 15 * time.Minute
	RefreshTokenExpiry = 30 * 24 * time.Hour // 30 days

	// Context keys for storing JWT data in Gin context
	contextKeyUserID   = "jwt_user_id"
	contextKeyJTI      = "jwt_jti"
	contextKeyTokenRaw = "jwt_token_raw"
)

// JWTManager handles JWT creation, validation, and blacklist management.
type JWTManager struct {
	secretKey []byte
	blacklist sync.Map // map[string]struct{} - stores revoked JTIs
}

// NewJWTManager creates a new JWTManager with the given signing secret.
func NewJWTManager(secretKey string) *JWTManager {
	return &JWTManager{
		secretKey: []byte(secretKey),
	}
}

// Claims represents the JWT claims structure.
type Claims struct {
	jwt.RegisteredClaims
	TokenType string `json:"type"`           // "access" or "refresh"
	Identity  string `json:"sub"`            // user ID as string (matches Flask behavior)
}

// CreateAccessToken generates a new access token for the given user ID.
func (m *JWTManager) CreateAccessToken(userID uint) (string, error) {
	return m.createToken(userID, "access", AccessTokenExpiry)
}

// CreateRefreshToken generates a new refresh token for the given user ID.
func (m *JWTManager) CreateRefreshToken(userID uint) (string, error) {
	return m.createToken(userID, "refresh", RefreshTokenExpiry)
}

func (m *JWTManager) createToken(userID uint, tokenType string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
		TokenType: tokenType,
		Identity:  fmt.Sprintf("%d", userID),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// ParseToken validates and parses a JWT token string, returning the claims.
func (m *JWTManager) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// RevokeToken adds a token's JTI to the blacklist.
func (m *JWTManager) RevokeToken(jti string) {
	m.blacklist.Store(jti, struct{}{})
}

// IsRevoked checks if a token's JTI is blacklisted.
func (m *JWTManager) IsRevoked(jti string) bool {
	_, exists := m.blacklist.Load(jti)
	return exists
}

// AuthRequired returns a Gin middleware that validates access tokens.
func (m *JWTManager) AuthRequired() gin.HandlerFunc {
	return m.requireToken("access")
}

// RefreshRequired returns a Gin middleware that validates refresh tokens.
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
				"msg": "Missing Authorization Header",
			})
			return
		}

		tokenString := parts[1]
		claims, err := m.ParseToken(tokenString)
		if err != nil {
			// Check if the error is an expiration error
			if strings.Contains(err.Error(), "token is expired") {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"msg": "Token has expired",
				})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
				"msg": "Bad Authorization header. Expected 'Bearer <JWT>'",
			})
			return
		}

		// Check token type
		if claims.TokenType != expectedType {
			c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
				"msg": fmt.Sprintf("Only %s tokens are allowed", expectedType),
			})
			return
		}

		// Check blacklist
		if m.IsRevoked(claims.ID) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"msg": "Token has been revoked",
			})
			return
		}

		// Store user identity and token info in context
		c.Set(contextKeyUserID, claims.Identity)
		c.Set(contextKeyJTI, claims.ID)
		c.Set(contextKeyTokenRaw, tokenString)

		c.Next()
	}
}

// GetUserID extracts the authenticated user's ID string from the Gin context.
func GetUserID(c *gin.Context) string {
	val, exists := c.Get(contextKeyUserID)
	if !exists {
		return ""
	}
	return val.(string)
}

// GetJTI extracts the JWT ID from the Gin context.
func GetJTI(c *gin.Context) string {
	val, exists := c.Get(contextKeyJTI)
	if !exists {
		return ""
	}
	return val.(string)
}

// HashPassword generates a bcrypt hash of the given plain-text password.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword compares a bcrypt hashed password with a plain-text candidate.
func CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
