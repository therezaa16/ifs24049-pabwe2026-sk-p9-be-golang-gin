package server

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/config"
	"app/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRouter(t *testing.T, name string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	db, err := database.OpenSQLite(name)
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	cfg := &config.Config{
		AppKey: "router-secret", AdminUsername: "admin", AdminPassword: "admin",
		SearchEngineEndpoint: "http://127.0.0.1:1/search", OpenAIEndpoint: "http://127.0.0.1:1", OpenAIKey: "k", OpenAIModel: "m",
	}
	require.NoError(t, database.Seed(db, cfg))
	return NewRouter(cfg, db)
}

func do(r *gin.Engine, method, path, token, body string) (*httptest.ResponseRecorder, map[string]any) {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var m map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	return w, m
}

func TestRoutesEndToEnd(t *testing.T) {
	r := newRouter(t, "router_e2e")

	w, m := do(r, "GET", "/api/public/categories", "", "")
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "success", m["status"])

	w, m = do(r, "GET", "/api/public/posts", "", "")
	assert.Equal(t, 200, w.Code)
	assert.NotEmpty(t, m["data"])

	w, m = do(r, "GET", "/api/rute-tidak-ada", "", "")
	assert.Equal(t, 404, w.Code)
	assert.Equal(t, "fail", m["status"])

	w, _ = do(r, "GET", "/api/posts", "", "")
	assert.Equal(t, 401, w.Code)

	// login admin hasil seeder
	w, m = do(r, "POST", "/api/auth/login", "", `{"username":"admin","password":"admin"}`)
	require.Equal(t, 200, w.Code)
	admin := m["data"].(map[string]any)["token"].(string)

	w, _ = do(r, "GET", "/api/admin/users", admin, "")
	assert.Equal(t, 200, w.Code)

	// member biasa ditolak di panel admin
	w, m = do(r, "POST", "/api/auth/login", "", `{"username":"member","password":"member"}`)
	require.Equal(t, 200, w.Code)
	member := m["data"].(map[string]any)["token"].(string)
	w, _ = do(r, "GET", "/api/admin/users", member, "")
	assert.Equal(t, 403, w.Code)

	// AI: layanan tidak tersedia -> 500 error (menyebut OpenAI)
	w, m = do(r, "POST", "/api/posts/ai-generate", member, `{"topic":"Golang"}`)
	assert.Equal(t, 500, w.Code)
	assert.Equal(t, "error", m["status"])

	// /api/posts/stats tidak bentrok dengan /api/posts/:id
	w, m = do(r, "GET", "/api/posts/stats", member, "")
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, m["data"], "total_articles")
}

func TestSwaggerAndSpec(t *testing.T) {
	r := newRouter(t, "router_swagger")

	w, _ := do(r, "GET", "/swagger/index.html", "", "")
	assert.Equal(t, http.StatusOK, w.Code)
	w, _ = do(r, "GET", "/swagger/doc.json", "", "")
	assert.Equal(t, http.StatusOK, w.Code)

	w, m := do(r, "GET", "/api.json", "", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []any{"http"}, m["schemes"])

	req := httptest.NewRequest("GET", "/api.json", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req)
	assert.Contains(t, w2.Body.String(), `"https"`)

	req = httptest.NewRequest("GET", "/api.json", nil)
	req.TLS = &tls.ConnectionState{}
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req)
	assert.Contains(t, w3.Body.String(), `"https"`)
}
