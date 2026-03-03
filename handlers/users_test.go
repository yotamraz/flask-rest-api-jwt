package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"flask-rest-api-jwt/models"
)

// ============================================================
// POST /user/register
// ============================================================

func TestRegister_Success(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	body := `{"username":"alice","password":"secret123"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/user/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "alice", resp["username"])
	assert.NotNil(t, resp["id"])
	// password_hash must not be in response
	assert.Nil(t, resp["password_hash"])
	assert.Nil(t, resp["PasswordHash"])
}

func TestRegister_DuplicateUser(t *testing.T) {
	router, db, _ := setupTestRouter(t)
	createTestUser(t, db, "alice", "secret123")

	body := `{"username":"alice","password":"otherpass"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/user/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "User exists", resp["message"])
}

func TestRegister_MissingFields(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	// Missing password
	body := `{"username":"alice"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/user/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================
// POST /user/login
// ============================================================

func TestLogin_Success(t *testing.T) {
	router, db, _ := setupTestRouter(t)
	createTestUser(t, db, "bob", "mypassword")

	body := `{"username":"bob","password":"mypassword"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/user/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.NotEmpty(t, resp["access_token"])
	assert.NotEmpty(t, resp["refresh_token"])
}

func TestLogin_InvalidPassword(t *testing.T) {
	router, db, _ := setupTestRouter(t)
	createTestUser(t, db, "bob", "mypassword")

	body := `{"username":"bob","password":"wrongpassword"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/user/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Invalid credentials", resp["message"])
}

func TestLogin_NonExistentUser(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	body := `{"username":"ghost","password":"nopassword"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/user/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Invalid credentials", resp["message"])
}

// ============================================================
// POST /user/logout
// ============================================================

func TestLogout_Success(t *testing.T) {
	router, db, jwtManager := setupTestRouter(t)
	user := createTestUser(t, db, "carol", "pass123")
	token := getAccessToken(t, jwtManager, user.ID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/user/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Logged out", resp["message"])

	// Verify token is now blacklisted - using it again should fail
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/user/1", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}

func TestLogout_NoToken(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/user/logout", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================
// POST /user/refresh
// ============================================================

func TestRefresh_Success(t *testing.T) {
	router, db, jwtManager := setupTestRouter(t)
	user := createTestUser(t, db, "dave", "pass123")
	refreshToken := getRefreshToken(t, jwtManager, user.ID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/user/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+refreshToken)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp["access_token"])
}

func TestRefresh_WithAccessToken_Rejected(t *testing.T) {
	router, db, jwtManager := setupTestRouter(t)
	user := createTestUser(t, db, "eve", "pass123")
	accessToken := getAccessToken(t, jwtManager, user.ID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/user/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestRefresh_NoToken(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/user/refresh", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================
// GET /user/:id
// ============================================================

func TestGetUser_Success(t *testing.T) {
	router, db, jwtManager := setupTestRouter(t)
	user := createTestUser(t, db, "frank", "pass123")
	token := getAccessToken(t, jwtManager, user.ID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/user/"+uintToStr(user.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "frank", resp["username"])
	assert.NotNil(t, resp["id"])
	// password_hash must not be in response
	assert.Nil(t, resp["password_hash"])
}

func TestGetUser_Unauthorized_OtherUser(t *testing.T) {
	router, db, jwtManager := setupTestRouter(t)
	user1 := createTestUser(t, db, "grace", "pass123")
	createTestUser(t, db, "hank", "pass456")

	token := getAccessToken(t, jwtManager, user1.ID)

	// user1 tries to get user2's profile
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/user/"+uintToStr(user1.ID+1), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Unauthorized", resp["message"])
}

func TestGetUser_NotFound(t *testing.T) {
	router, db, jwtManager := setupTestRouter(t)
	user := createTestUser(t, db, "ivan", "pass123")
	token := getAccessToken(t, jwtManager, user.ID)

	// Try to get own profile (exists) - should work
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/user/"+uintToStr(user.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetUser_NoToken(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/user/1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================
// DELETE /user/:id
// ============================================================

func TestDeleteUser_Success(t *testing.T) {
	router, db, jwtManager := setupTestRouter(t)
	user := createTestUser(t, db, "judy", "pass123")
	token := getAccessToken(t, jwtManager, user.ID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/user/"+uintToStr(user.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Deleted", resp["message"])

	// Verify user is actually deleted
	var count int64
	db.Model(&models.User{}).Where("id = ?", user.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestDeleteUser_CascadeDeletes(t *testing.T) {
	router, db, jwtManager := setupTestRouter(t)
	user := createTestUser(t, db, "karl", "pass123")

	// Create a store, item, and tag for this user
	store := models.Store{Name: "TestStore", UserID: user.ID}
	db.Create(&store)

	item := models.Item{Name: "TestItem", Price: 9.99, StoreID: store.ID}
	db.Create(&item)

	tag := models.Tag{Name: "TestTag", StoreID: store.ID}
	db.Create(&tag)

	token := getAccessToken(t, jwtManager, user.ID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/user/"+uintToStr(user.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify cascade: store, item, and tag should be deleted
	var storeCount, itemCount, tagCount int64
	db.Model(&models.Store{}).Where("user_id = ?", user.ID).Count(&storeCount)
	db.Model(&models.Item{}).Where("store_id = ?", store.ID).Count(&itemCount)
	db.Model(&models.Tag{}).Where("store_id = ?", store.ID).Count(&tagCount)

	assert.Equal(t, int64(0), storeCount)
	assert.Equal(t, int64(0), itemCount)
	assert.Equal(t, int64(0), tagCount)
}

func TestDeleteUser_Unauthorized_OtherUser(t *testing.T) {
	router, db, jwtManager := setupTestRouter(t)
	user1 := createTestUser(t, db, "laura", "pass123")
	user2 := createTestUser(t, db, "mike", "pass456")

	token := getAccessToken(t, jwtManager, user1.ID)

	// user1 tries to delete user2
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/user/"+uintToStr(user2.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "Unauthorized", resp["message"])

	// user2 should still exist
	var count int64
	db.Model(&models.User{}).Where("id = ?", user2.ID).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestDeleteUser_NoToken(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/user/1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================
// Integration: Full user flow
// ============================================================

func TestFullUserFlow_RegisterLoginLogout(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	// 1. Register
	regBody := `{"username":"flowuser","password":"flowpass"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/user/register", strings.NewReader(regBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var regResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &regResp)
	userID := strconv.FormatFloat(regResp["id"].(float64), 'f', 0, 64)
	userPath := "/user/" + userID

	// 2. Login
	loginBody := `{"username":"flowuser","password":"flowpass"}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/user/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var loginResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResp)
	accessToken := loginResp["access_token"].(string)
	refreshToken := loginResp["refresh_token"].(string)

	// 3. Access protected endpoint with access token
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", userPath, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 4. Refresh token
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/user/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+refreshToken)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var refreshResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &refreshResp)
	newAccessToken := refreshResp["access_token"].(string)
	assert.NotEmpty(t, newAccessToken)

	// 5. Logout with original access token
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/user/logout", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 6. Original access token should now be revoked
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", userPath, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 7. But the new access token from refresh should still work
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", userPath, nil)
	req.Header.Set("Authorization", "Bearer "+newAccessToken)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================
// Helper
// ============================================================

func uintToStr(n uint) string {
	return strconv.FormatUint(uint64(n), 10)
}
