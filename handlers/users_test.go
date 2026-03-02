package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- helpers ----------

func registerUser(t *testing.T, router http.Handler, username, password string) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"username":"%s","password":"%s"}`, username, password)
	req, _ := http.NewRequest("POST", "/user/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func loginUser(t *testing.T, router http.Handler, username, password string) (accessToken, refreshToken string) {
	t.Helper()
	body := fmt.Sprintf(`{"username":"%s","password":"%s"}`, username, password)
	req, _ := http.NewRequest("POST", "/user/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp["access_token"], resp["refresh_token"]
}

// ---------- Registration tests ----------

func TestRegisterSuccess(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	w := registerUser(t, r, "alice", "pass123")

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "alice", resp["username"])
	assert.NotZero(t, resp["id"])
	// Password hash must not be in response.
	assert.Nil(t, resp["password_hash"])
}

func TestRegisterDuplicate(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	w := registerUser(t, r, "bob", "pass123")
	assert.Equal(t, http.StatusCreated, w.Code)

	// Second registration with same username.
	w = registerUser(t, r, "bob", "pass456")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "User exists", resp["message"])
}

// ---------- Login tests ----------

func TestLoginSuccess(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	registerUser(t, r, "carol", "password")

	body := `{"username":"carol","password":"password"}`
	req, _ := http.NewRequest("POST", "/user/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["access_token"])
	assert.NotEmpty(t, resp["refresh_token"])
}

func TestLoginInvalidCredentials(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	registerUser(t, r, "dave", "correctpass")

	body := `{"username":"dave","password":"wrongpass"}`
	req, _ := http.NewRequest("POST", "/user/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Invalid credentials", resp["message"])
}

func TestLoginNonExistentUser(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	body := `{"username":"nobody","password":"whatever"}`
	req, _ := http.NewRequest("POST", "/user/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Invalid credentials", resp["message"])
}

// ---------- Logout tests ----------

func TestLogoutSuccess(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	registerUser(t, r, "eve", "pass")
	access, _ := loginUser(t, r, "eve", "pass")

	req, _ := http.NewRequest("POST", "/user/logout", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Logged out", resp["message"])
}

func TestLogoutTokenBlacklisted(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	registerUser(t, r, "frank", "pass")
	access, _ := loginUser(t, r, "frank", "pass")

	// Logout (blacklists the token).
	req, _ := http.NewRequest("POST", "/user/logout", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Try to use the same token — should be rejected.
	req, _ = http.NewRequest("GET", "/user/1", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogoutWithoutToken(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	req, _ := http.NewRequest("POST", "/user/logout", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------- Refresh tests ----------

func TestRefreshSuccess(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	registerUser(t, r, "grace", "pass")
	_, refresh := loginUser(t, r, "grace", "pass")

	req, _ := http.NewRequest("POST", "/user/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+refresh)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["access_token"])
}

func TestRefreshWithAccessTokenFails(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	registerUser(t, r, "heidi", "pass")
	access, _ := loginUser(t, r, "heidi", "pass")

	// Using access token on refresh endpoint should fail.
	req, _ := http.NewRequest("POST", "/user/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------- Get User tests ----------

func TestGetUserSuccess(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	w := registerUser(t, r, "ivan", "pass")
	var regResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &regResp))
	userID := int(regResp["id"].(float64))

	access, _ := loginUser(t, r, "ivan", "pass")

	req, _ := http.NewRequest("GET", fmt.Sprintf("/user/%d", userID), nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "ivan", resp["username"])
}

func TestGetUserUnauthorizedOtherUser(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	registerUser(t, r, "judy", "pass")
	w := registerUser(t, r, "mallory", "pass")
	var regResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &regResp))
	malloryID := int(regResp["id"].(float64))

	// Login as judy, try to access mallory.
	access, _ := loginUser(t, r, "judy", "pass")

	req, _ := http.NewRequest("GET", fmt.Sprintf("/user/%d", malloryID), nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Unauthorized", resp["message"])
}

func TestGetUserNotFound(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	registerUser(t, r, "niaj", "pass")
	access, _ := loginUser(t, r, "niaj", "pass")

	req, _ := http.NewRequest("GET", "/user/9999", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetUserNoAuth(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	req, _ := http.NewRequest("GET", "/user/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------- Delete User tests ----------

func TestDeleteUserSuccess(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	w := registerUser(t, r, "oscar", "pass")
	var regResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &regResp))
	userID := int(regResp["id"].(float64))

	access, _ := loginUser(t, r, "oscar", "pass")

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/user/%d", userID), nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Deleted", resp["message"])
}

func TestDeleteUserUnauthorized(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	registerUser(t, r, "pat", "pass")
	w := registerUser(t, r, "quinn", "pass")
	var regResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &regResp))
	quinnID := int(regResp["id"].(float64))

	access, _ := loginUser(t, r, "pat", "pass")

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/user/%d", quinnID), nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Unauthorized", resp["message"])
}

func TestDeleteUserNotFound(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	registerUser(t, r, "rosa", "pass")
	access, _ := loginUser(t, r, "rosa", "pass")

	req, _ := http.NewRequest("DELETE", "/user/9999", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
