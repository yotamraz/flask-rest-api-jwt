package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthCheckHealthy(t *testing.T) {
	db := setupTestDB()
	r, _ := setupTestRouter(db)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "healthy", body["status"])
	assert.Equal(t, "healthy", body["database"])
}
