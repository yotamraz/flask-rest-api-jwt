package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestNewJWTManager(t *testing.T) {
	m := NewJWTManager("test-secret")
	assert.NotNil(t, m)
}

func TestCreateAccessToken(t *testing.T) {
	m := NewJWTManager("test-secret")
	token, err := m.CreateAccessToken(1)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := m.ParseToken(token)
	require.NoError(t, err)
	assert.Equal(t, uint(1), claims.UserID)
	assert.Equal(t, TokenTypeAccess, claims.TokenType)
	assert.NotEmpty(t, claims.ID) // JTI
}

func TestCreateRefreshToken(t *testing.T) {
	m := NewJWTManager("test-secret")
	token, err := m.CreateRefreshToken(42)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := m.ParseToken(token)
	require.NoError(t, err)
	assert.Equal(t, uint(42), claims.UserID)
	assert.Equal(t, TokenTypeRefresh, claims.TokenType)
}

func TestParseTokenInvalidSignature(t *testing.T) {
	m1 := NewJWTManager("secret-1")
	m2 := NewJWTManager("secret-2")

	token, err := m1.CreateAccessToken(1)
	require.NoError(t, err)

	_, err = m2.ParseToken(token)
	assert.Error(t, err)
}

func TestParseTokenExpired(t *testing.T) {
	m := NewJWTManager("test-secret")

	// Create a token that's already expired
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "1",
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ID:        "test-jti",
		},
		TokenType: TokenTypeAccess,
		UserID:    1,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("test-secret"))
	require.NoError(t, err)

	_, err = m.ParseToken(tokenString)
	assert.Error(t, err)
}

func TestBlacklist(t *testing.T) {
	m := NewJWTManager("test-secret")

	assert.False(t, m.IsBlacklisted("some-jti"))

	m.BlacklistToken("some-jti")
	assert.True(t, m.IsBlacklisted("some-jti"))

	// Other JTIs should not be blacklisted
	assert.False(t, m.IsBlacklisted("other-jti"))
}

func TestRequireAuthMiddlewareNoToken(t *testing.T) {
	m := NewJWTManager("test-secret")

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/test", m.RequireAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	c.Request = httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireAuthMiddlewareValidToken(t *testing.T) {
	m := NewJWTManager("test-secret")
	token, _ := m.CreateAccessToken(5)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/test", m.RequireAuth(), func(c *gin.Context) {
		userID, ok := GetUserID(c)
		assert.True(t, ok)
		assert.Equal(t, uint(5), userID)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireAuthMiddlewareBlacklistedToken(t *testing.T) {
	m := NewJWTManager("test-secret")
	token, _ := m.CreateAccessToken(1)

	// Parse to get JTI and blacklist
	claims, _ := m.ParseToken(token)
	m.BlacklistToken(claims.ID)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/test", m.RequireAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireAuthMiddlewareRefreshTokenRejected(t *testing.T) {
	m := NewJWTManager("test-secret")
	// Create a refresh token and try to use it as an access token
	token, _ := m.CreateRefreshToken(1)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/test", m.RequireAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireRefreshMiddlewareAcceptsRefreshToken(t *testing.T) {
	m := NewJWTManager("test-secret")
	token, _ := m.CreateRefreshToken(7)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.POST("/test", m.RequireRefresh(), func(c *gin.Context) {
		userID, ok := GetUserID(c)
		assert.True(t, ok)
		assert.Equal(t, uint(7), userID)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireRefreshMiddlewareRejectsAccessToken(t *testing.T) {
	m := NewJWTManager("test-secret")
	token, _ := m.CreateAccessToken(1)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.POST("/test", m.RequireRefresh(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireAuthMiddlewareMalformedHeader(t *testing.T) {
	m := NewJWTManager("test-secret")

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/test", m.RequireAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("mysecret")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "mysecret", hash)

	assert.True(t, CheckPassword(hash, "mysecret"))
	assert.False(t, CheckPassword(hash, "wrongpassword"))
}
