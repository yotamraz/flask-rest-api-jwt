package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"flask-rest-api-jwt/middleware"
	"flask-rest-api-jwt/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// registerUser is a test helper that registers a user and returns the response.
func registerUser(r http.Handler, username, password string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/user/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

// loginUser is a test helper that logs in and returns the response.
func loginUser(r http.Handler, username, password string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/user/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

// getTokens is a test helper that registers and logs in a user, returning tokens.
func getTokens(t *testing.T, r http.Handler, username, password string) (accessToken, refreshToken string) {
	t.Helper()
	registerUser(r, username, password)
	w := loginUser(r, username, password)
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	return body["access_token"], body["refresh_token"]
}

// freshDB creates a fresh in-memory SQLite database (unique per test to avoid shared state).
func freshDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(
		sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())),
		&gorm.Config{},
	)
	require.NoError(t, err)

	sqlDB, _ := db.DB()
	sqlDB.Exec("PRAGMA foreign_keys = ON")

	err = db.AutoMigrate(&models.User{}, &models.Store{}, &models.Item{}, &models.Tag{})
	require.NoError(t, err)

	return db
}

// freshRouter creates a fresh router with a fresh DB for test isolation.
func freshRouter(t *testing.T) (http.Handler, *middleware.JWTManager) {
	t.Helper()
	db := freshDB(t)
	return setupTestRouter(db)
}

// --- Registration Tests ---

func TestRegisterSuccess(t *testing.T) {
	r, _ := freshRouter(t)

	w := registerUser(r, "alice", "password123")
	assert.Equal(t, http.StatusCreated, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "alice", body["username"])
	assert.NotNil(t, body["id"])
	// Ensure password_hash is NOT in the response
	assert.Nil(t, body["password_hash"])
}

func TestRegisterDuplicateUser(t *testing.T) {
	r, _ := freshRouter(t)

	w := registerUser(r, "bob", "password123")
	assert.Equal(t, http.StatusCreated, w.Code)

	w = registerUser(r, "bob", "password456")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "User exists", body["message"])
}

func TestRegisterMissingFields(t *testing.T) {
	r, _ := freshRouter(t)

	body, _ := json.Marshal(map[string]string{"username": "alice"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/user/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.(http.Handler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Login Tests ---

func TestLoginSuccess(t *testing.T) {
	r, _ := freshRouter(t)

	registerUser(r, "carol", "mypassword")
	w := loginUser(r, "carol", "mypassword")
	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.NotEmpty(t, body["access_token"])
	assert.NotEmpty(t, body["refresh_token"])
}

func TestLoginInvalidCredentials(t *testing.T) {
	r, _ := freshRouter(t)

	registerUser(r, "dave", "correctpassword")
	w := loginUser(r, "dave", "wrongpassword")
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "Invalid credentials", body["message"])
}

func TestLoginNonexistentUser(t *testing.T) {
	r, _ := freshRouter(t)

	w := loginUser(r, "nobody", "password")
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "Invalid credentials", body["message"])
}

// --- Logout Tests ---

func TestLogoutSuccess(t *testing.T) {
	r, _ := freshRouter(t)
	accessToken, _ := getTokens(t, r, "eve", "password123")

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/user/logout", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	r.(http.Handler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "Logged out", body["message"])

	// Using the same token after logout should fail
	w = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/user/logout", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	r.(http.Handler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogoutNoToken(t *testing.T) {
	r, _ := freshRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/user/logout", nil)
	r.(http.Handler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- Refresh Tests ---

func TestRefreshSuccess(t *testing.T) {
	r, _ := freshRouter(t)
	_, refreshToken := getTokens(t, r, "frank", "password123")

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/user/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+refreshToken)
	r.(http.Handler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.NotEmpty(t, body["access_token"])
}

func TestRefreshWithAccessTokenFails(t *testing.T) {
	r, _ := freshRouter(t)
	accessToken, _ := getTokens(t, r, "grace", "password123")

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/user/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	r.(http.Handler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- GetUser Tests ---

func TestGetUserSuccess(t *testing.T) {
	r, _ := freshRouter(t)
	accessToken, _ := getTokens(t, r, "heidi", "password123")

	// Find user ID (should be 1 in fresh DB)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/user/1", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	r.(http.Handler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "heidi", body["username"])
	assert.Nil(t, body["password_hash"])
}

func TestGetUserUnauthorized(t *testing.T) {
	r, _ := freshRouter(t)
	accessToken, _ := getTokens(t, r, "ivan", "password123")

	// Register another user to create user ID 2
	registerUser(r, "judy", "password456")

	// Try to access other user's profile
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/user/2", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	r.(http.Handler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "Unauthorized", body["message"])
}

func TestGetUserNotFound(t *testing.T) {
	r, _ := freshRouter(t)
	accessToken, _ := getTokens(t, r, "kate", "password123")

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/user/999", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	r.(http.Handler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetUserNoAuth(t *testing.T) {
	r, _ := freshRouter(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/user/1", nil)
	r.(http.Handler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- DeleteUser Tests ---

func TestDeleteUserSuccess(t *testing.T) {
	r, _ := freshRouter(t)
	accessToken, _ := getTokens(t, r, "leo", "password123")

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/user/1", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	r.(http.Handler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "Deleted", body["message"])

	// Verify user is gone (try to get)
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/user/1", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	r.(http.Handler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteUserUnauthorized(t *testing.T) {
	r, _ := freshRouter(t)
	accessToken, _ := getTokens(t, r, "mona", "password123")
	registerUser(r, "nick", "password456")

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/user/2", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	r.(http.Handler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDeleteUserNotFound(t *testing.T) {
	r, _ := freshRouter(t)
	accessToken, _ := getTokens(t, r, "olivia", "password123")

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/user/999", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	r.(http.Handler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
