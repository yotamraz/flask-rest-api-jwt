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

func TestHashPasswordAndCheck(t *testing.T) {
	hash, err := HashPassword("secret123")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "secret123", hash)

	// Correct password should match.
	assert.NoError(t, CheckPassword(hash, "secret123"))

	// Wrong password should fail.
	assert.Error(t, CheckPassword(hash, "wrongpass"))
}

func TestCreateAndParseAccessToken(t *testing.T) {
	mgr := NewJWTManager("test-secret")

	token, err := mgr.CreateAccessToken(42)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := mgr.ParseToken(token)
	require.NoError(t, err)
	assert.Equal(t, uint(42), claims.UserID)
	assert.Equal(t, "42", claims.Subject)
	assert.Equal(t, "access", claims.TokenType)
	assert.NotEmpty(t, claims.ID) // JTI
}

func TestCreateAndParseRefreshToken(t *testing.T) {
	mgr := NewJWTManager("test-secret")

	token, err := mgr.CreateRefreshToken(7)
	require.NoError(t, err)

	claims, err := mgr.ParseToken(token)
	require.NoError(t, err)
	assert.Equal(t, uint(7), claims.UserID)
	assert.Equal(t, "refresh", claims.TokenType)
}

func TestParseTokenInvalidSignature(t *testing.T) {
	mgr1 := NewJWTManager("secret-one")
	mgr2 := NewJWTManager("secret-two")

	token, err := mgr1.CreateAccessToken(1)
	require.NoError(t, err)

	_, err = mgr2.ParseToken(token)
	assert.Error(t, err)
}

func TestParseTokenExpired(t *testing.T) {
	mgr := NewJWTManager("test-secret")

	// Create an already-expired token.
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "1",
			ID:        "test-jti",
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
		},
		TokenType: "access",
		UserID:    1,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("test-secret"))
	require.NoError(t, err)

	_, err = mgr.ParseToken(tokenStr)
	assert.Error(t, err)
}

func TestBlacklist(t *testing.T) {
	mgr := NewJWTManager("test-secret")

	assert.False(t, mgr.IsBlacklisted("some-jti"))

	mgr.BlacklistToken("some-jti")
	assert.True(t, mgr.IsBlacklisted("some-jti"))

	// Other JTIs should not be blacklisted.
	assert.False(t, mgr.IsBlacklisted("other-jti"))
}

func TestAuthRequiredMiddlewareNoHeader(t *testing.T) {
	mgr := NewJWTManager("test-secret")

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/protected", mgr.AuthRequired(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	c.Request, _ = http.NewRequest("GET", "/protected", nil)
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthRequiredMiddlewareValidToken(t *testing.T) {
	mgr := NewJWTManager("test-secret")

	token, err := mgr.CreateAccessToken(5)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/protected", mgr.AuthRequired(), func(c *gin.Context) {
		uid, ok := GetUserID(c)
		assert.True(t, ok)
		assert.Equal(t, uint(5), uid)

		jti, ok := GetJTI(c)
		assert.True(t, ok)
		assert.NotEmpty(t, jti)

		c.JSON(http.StatusOK, gin.H{"user_id": uid})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthRequiredMiddlewareBlacklistedToken(t *testing.T) {
	mgr := NewJWTManager("test-secret")

	token, err := mgr.CreateAccessToken(5)
	require.NoError(t, err)

	// Blacklist the token's JTI.
	claims, err := mgr.ParseToken(token)
	require.NoError(t, err)
	mgr.BlacklistToken(claims.ID)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/protected", mgr.AuthRequired(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthRequiredRejectsRefreshToken(t *testing.T) {
	mgr := NewJWTManager("test-secret")

	token, err := mgr.CreateRefreshToken(5)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/protected", mgr.AuthRequired(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRefreshRequiredAcceptsRefreshToken(t *testing.T) {
	mgr := NewJWTManager("test-secret")

	token, err := mgr.CreateRefreshToken(5)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.POST("/refresh", mgr.RefreshRequired(), func(c *gin.Context) {
		uid, ok := GetUserID(c)
		assert.True(t, ok)
		assert.Equal(t, uint(5), uid)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req, _ := http.NewRequest("POST", "/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRefreshRequiredRejectsAccessToken(t *testing.T) {
	mgr := NewJWTManager("test-secret")

	token, err := mgr.CreateAccessToken(5)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.POST("/refresh", mgr.RefreshRequired(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req, _ := http.NewRequest("POST", "/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthRequiredMalformedHeader(t *testing.T) {
	mgr := NewJWTManager("test-secret")

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/protected", mgr.AuthRequired(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
