package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

const secret = "test-secret"

func sign(t *testing.T, method jwt.SigningMethod, role string, exp time.Duration) string {
	claims := Claims{UserID: 7, Username: "u", Role: role,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp))}}
	s, err := jwt.NewWithClaims(method, claims).SignedString([]byte(secret))
	assert.NoError(t, err)
	return s
}

func run(r *gin.Engine, method, path, auth string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	w := run(r, http.MethodGet, "/x", "")
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

	w = run(r, http.MethodOptions, "/x", "")
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestAuthAndRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/me", Auth(secret), func(c *gin.Context) { c.JSON(200, gin.H{"id": UserID(c)}) })
	r.GET("/admin", Auth(secret), RequireRole("admin"), func(c *gin.Context) { c.String(200, "ok") })

	assert.Equal(t, 401, run(r, "GET", "/me", "").Code)
	assert.Equal(t, 401, run(r, "GET", "/me", sign(t, jwt.SigningMethodHS256, "member", time.Hour)).Code) // tanpa "Bearer"
	assert.Equal(t, 401, run(r, "GET", "/me", "Bearer token.palsu.tidak").Code)
	assert.Equal(t, 401, run(r, "GET", "/me", "Bearer "+sign(t, jwt.SigningMethodHS256, "member", -time.Hour)).Code)
	assert.Equal(t, 401, run(r, "GET", "/me", "Bearer "+sign(t, jwt.SigningMethodHS512, "member", time.Hour)).Code)

	w := run(r, "GET", "/me", "Bearer "+sign(t, jwt.SigningMethodHS256, "member", time.Hour))
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"id":7`)

	assert.Equal(t, 403, run(r, "GET", "/admin", "Bearer "+sign(t, jwt.SigningMethodHS256, "member", time.Hour)).Code)
	assert.Equal(t, 200, run(r, "GET", "/admin", "Bearer "+sign(t, jwt.SigningMethodHS256, "admin", time.Hour)).Code)
}

func TestUserIDMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	assert.Equal(t, uint(0), UserID(c))
}

func TestRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimiter(2, 50*time.Millisecond))
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	assert.Equal(t, 200, run(r, "GET", "/x", "").Code)
	assert.Equal(t, 200, run(r, "GET", "/x", "").Code)
	assert.Equal(t, 429, run(r, "GET", "/x", "").Code)
	time.Sleep(70 * time.Millisecond)
	assert.Equal(t, 200, run(r, "GET", "/x", "").Code)
}
