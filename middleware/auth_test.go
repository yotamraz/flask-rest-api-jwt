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
	jm := NewJWTManager("test-secret")
	assert.NotNil(t, jm)
	assert.Equal(t, []byte("test-secret"), jm.secretKey)
}

func TestCreateAccessToken(t *testing.T) {
	jm := NewJWTManager("test-secret")
	token, err := jm.CreateAccessToken(1)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Parse and verify claims
	claims, err := jm.ParseToken(token)
	require.NoError(t, err)
	assert.Equal(t, "access", claims.TokenType)
	assert.Equal(t, "1", claims.Identity)
	assert.NotEmpty(t, claims.ID) // JTI
	assert.True(t, claims.ExpiresAt.Time.After(time.Now()))
	assert.True(t, claims.ExpiresAt.Time.Before(time.Now().Add(16*time.Minute)))
}

func TestCreateRefreshToken(t *testing.T) {
	jm := NewJWTManager("test-secret")
	token, err := jm.CreateRefreshToken(42)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := jm.ParseToken(token)
	require.NoError(t, err)
	assert.Equal(t, "refresh", claims.TokenType)
	assert.Equal(t, "42", claims.Identity)
	assert.True(t, claims.ExpiresAt.Time.After(time.Now().Add(29*24*time.Hour)))
}

func TestParseToken_InvalidSignature(t *testing.T) {
	jm1 := NewJWTManager("secret-1")
	jm2 := NewJWTManager("secret-2")

	token, err := jm1.CreateAccessToken(1)
	require.NoError(t, err)

	_, err = jm2.ParseToken(token)
	assert.Error(t, err)
}

func TestParseToken_ExpiredToken(t *testing.T) {
	jm := NewJWTManager("test-secret")

	// Create a token that's already expired
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        "test-jti",
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
		},
		TokenType: "access",
		Identity:  "1",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jm.secretKey)
	require.NoError(t, err)

	_, err = jm.ParseToken(tokenString)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestParseToken_MalformedToken(t *testing.T) {
	jm := NewJWTManager("test-secret")
	_, err := jm.ParseToken("not-a-valid-jwt")
	assert.Error(t, err)
}

func TestRevokeToken_And_IsRevoked(t *testing.T) {
	jm := NewJWTManager("test-secret")

	assert.False(t, jm.IsRevoked("some-jti"))

	jm.RevokeToken("some-jti")
	assert.True(t, jm.IsRevoked("some-jti"))

	// Other JTIs should not be affected
	assert.False(t, jm.IsRevoked("other-jti"))
}

func TestHashPassword_And_CheckPassword(t *testing.T) {
	hash, err := HashPassword("mypassword")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "mypassword", hash) // must not be plaintext

	assert.True(t, CheckPassword(hash, "mypassword"))
	assert.False(t, CheckPassword(hash, "wrongpassword"))
}

func TestAuthRequired_ValidAccessToken(t *testing.T) {
	jm := NewJWTManager("test-secret")
	token, _ := jm.CreateAccessToken(1)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/protected", jm.AuthRequired(), func(c *gin.Context) {
		userID := GetUserID(c)
		jti := GetJTI(c)
		c.JSON(http.StatusOK, gin.H{"user_id": userID, "jti": jti})
	})

	c.Request, _ = http.NewRequest("GET", "/protected", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"user_id":"1"`)
}

func TestAuthRequired_MissingHeader(t *testing.T) {
	jm := NewJWTManager("test-secret")

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/protected", jm.AuthRequired(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	c.Request, _ = http.NewRequest("GET", "/protected", nil)
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Missing Authorization Header")
}

func TestAuthRequired_MalformedHeader(t *testing.T) {
	jm := NewJWTManager("test-secret")

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/protected", jm.AuthRequired(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	c.Request, _ = http.NewRequest("GET", "/protected", nil)
	c.Request.Header.Set("Authorization", "NotBearer token")
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthRequired_InvalidToken(t *testing.T) {
	jm := NewJWTManager("test-secret")

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/protected", jm.AuthRequired(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	c.Request, _ = http.NewRequest("GET", "/protected", nil)
	c.Request.Header.Set("Authorization", "Bearer invalid-token-string")
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestAuthRequired_ExpiredToken(t *testing.T) {
	jm := NewJWTManager("test-secret")

	// Create an expired token
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        "expired-jti",
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
		},
		TokenType: "access",
		Identity:  "1",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(jm.secretKey)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/protected", jm.AuthRequired(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	c.Request, _ = http.NewRequest("GET", "/protected", nil)
	c.Request.Header.Set("Authorization", "Bearer "+tokenString)
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Token has expired")
}

func TestAuthRequired_BlacklistedToken(t *testing.T) {
	jm := NewJWTManager("test-secret")
	token, _ := jm.CreateAccessToken(1)

	// Parse to get JTI and blacklist it
	claims, _ := jm.ParseToken(token)
	jm.RevokeToken(claims.ID)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/protected", jm.AuthRequired(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	c.Request, _ = http.NewRequest("GET", "/protected", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Token has been revoked")
}

func TestAuthRequired_RefreshTokenRejected(t *testing.T) {
	jm := NewJWTManager("test-secret")
	token, _ := jm.CreateRefreshToken(1)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.GET("/protected", jm.AuthRequired(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	c.Request, _ = http.NewRequest("GET", "/protected", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "Only access tokens are allowed")
}

func TestRefreshRequired_ValidRefreshToken(t *testing.T) {
	jm := NewJWTManager("test-secret")
	token, _ := jm.CreateRefreshToken(1)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.POST("/refresh", jm.RefreshRequired(), func(c *gin.Context) {
		userID := GetUserID(c)
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	c.Request, _ = http.NewRequest("POST", "/refresh", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"user_id":"1"`)
}

func TestRefreshRequired_AccessTokenRejected(t *testing.T) {
	jm := NewJWTManager("test-secret")
	token, _ := jm.CreateAccessToken(1)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.POST("/refresh", jm.RefreshRequired(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	c.Request, _ = http.NewRequest("POST", "/refresh", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "Only refresh tokens are allowed")
}

func TestGetUserID_NotSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	assert.Equal(t, "", GetUserID(c))
}

func TestGetJTI_NotSet(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	assert.Equal(t, "", GetJTI(c))
}
