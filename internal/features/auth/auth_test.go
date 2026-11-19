package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"app/internal/database"
	"app/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const secret = "unit-test-secret"

func setup(t *testing.T, name string) (*gin.Engine, *gorm.DB) {
	gin.SetMode(gin.TestMode)
	db, err := database.OpenSQLite(name)
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	h := NewHandler(NewService(NewRepository(db), secret))
	r := gin.New()
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	r.POST("/logout", h.Logout)
	g := r.Group("/", middleware.Auth(secret))
	g.GET("/profile", h.Profile)
	g.PUT("/profile", h.UpdateProfile)
	return r, db
}

func call(r *gin.Engine, method, path, token string, body any) (int, map[string]any) {
	var buf bytes.Buffer
	switch b := body.(type) {
	case string:
		buf.WriteString(b)
	case nil:
	default:
		_ = json.NewEncoder(&buf).Encode(b)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var m map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	return w.Code, m
}

func data(m map[string]any) map[string]any { d, _ := m["data"].(map[string]any); return d }

func register(t *testing.T, r *gin.Engine, username string, extra map[string]any) string {
	body := map[string]any{"username": username, "email": username + "@uji.local", "password": "Password123!"}
	for k, v := range extra {
		body[k] = v
	}
	code, m := call(r, "POST", "/register", "", body)
	require.Equal(t, 201, code)
	return data(m)["token"].(string)
}

func failQuery(db *gorm.DB, name string, only func(tx *gorm.DB) bool) {
	_ = db.Callback().Query().Before("gorm:query").Register(name, func(tx *gorm.DB) {
		if only(tx) {
			_ = tx.AddError(errors.New("forced"))
		}
	})
}

func TestRegisterLoginFlow(t *testing.T) {
	r, _ := setup(t, "auth_flow")

	code, m := call(r, "POST", "/register", "", map[string]any{"username": "alice", "email": "Alice@Uji.Local", "password": "Password123!", "role": "admin"})
	assert.Equal(t, 201, code)
	assert.Equal(t, "admin", data(m)["user"].(map[string]any)["role"])
	assert.Equal(t, "alice@uji.local", data(m)["user"].(map[string]any)["email"])
	assert.Len(t, strings.Split(data(m)["token"].(string), "."), 3)

	code, m = call(r, "POST", "/register", "", map[string]any{"username": "bob", "email": "bob@uji.local", "password": "Password123!", "role": "superuser"})
	assert.Equal(t, 201, code)
	assert.Equal(t, "member", data(m)["user"].(map[string]any)["role"])

	code, m = call(r, "POST", "/register", "", map[string]any{"username": "alice", "email": "x@uji.local", "password": "Password123!"})
	assert.Equal(t, 400, code)
	assert.Equal(t, "fail", m["status"])

	for _, body := range []any{
		map[string]any{"username": "c", "password": "Password123!"},
		map[string]any{"username": "c", "email": "c@uji.local", "password": "12345"},
		map[string]any{"username": "c", "email": "bukan-email", "password": "Password123!"},
		"{rusak",
	} {
		code, _ = call(r, "POST", "/register", "", body)
		assert.Equal(t, 400, code)
	}

	// login: username, email, email huruf besar
	for _, id := range []string{"alice", "alice@uji.local", "ALICE@UJI.LOCAL"} {
		code, m = call(r, "POST", "/login", "", map[string]any{"username": id, "password": "Password123!"})
		assert.Equal(t, 200, code, id)
		assert.Equal(t, "alice", data(m)["user"].(map[string]any)["username"])
	}

	code, m = call(r, "POST", "/login", "", map[string]any{"username": "alice", "password": "salah!!"})
	assert.Equal(t, 401, code)
	assert.Nil(t, m["data"])
	code, _ = call(r, "POST", "/login", "", map[string]any{"username": "tidak-ada", "password": "Password123!"})
	assert.Equal(t, 401, code)
	code, _ = call(r, "POST", "/login", "", map[string]any{"username": "alice"})
	assert.Equal(t, 400, code)

	code, m = call(r, "POST", "/logout", "", nil)
	assert.Equal(t, 200, code)
	assert.Equal(t, "success", m["status"])
}

func TestProfile(t *testing.T) {
	r, db := setup(t, "auth_profile")
	token := register(t, r, "carol", nil)
	register(t, r, "dave", nil)

	code, m := call(r, "GET", "/profile", token, nil)
	assert.Equal(t, 200, code)
	assert.Equal(t, "carol", data(m)["username"])
	assert.NotContains(t, strings.ToLower(m["message"].(string)+toJSON(m)), "password")

	code, _ = call(r, "GET", "/profile", "", nil)
	assert.Equal(t, 401, code)

	// ubah email + password
	code, m = call(r, "PUT", "/profile", token, map[string]any{"email": "baru@uji.local", "password": "PasswordBaru2026!"})
	assert.Equal(t, 200, code)
	assert.Equal(t, "baru@uji.local", data(m)["email"])
	code, _ = call(r, "POST", "/login", "", map[string]any{"username": "carol", "password": "PasswordBaru2026!"})
	assert.Equal(t, 200, code)
	code, _ = call(r, "POST", "/login", "", map[string]any{"username": "carol", "password": "Password123!"})
	assert.Equal(t, 401, code)

	// email sama (tidak berubah) + body kosong
	code, _ = call(r, "PUT", "/profile", token, map[string]any{"email": "baru@uji.local"})
	assert.Equal(t, 200, code)
	code, _ = call(r, "PUT", "/profile", token, map[string]any{})
	assert.Equal(t, 200, code)

	// gagal: email tidak valid, duplikat, password pendek
	code, _ = call(r, "PUT", "/profile", token, map[string]any{"email": "bukan-email"})
	assert.Equal(t, 400, code)
	code, _ = call(r, "PUT", "/profile", token, map[string]any{"email": "dave@uji.local"})
	assert.Equal(t, 400, code)
	code, _ = call(r, "PUT", "/profile", token, map[string]any{"password": "123"})
	assert.Equal(t, 400, code)
	code, _ = call(r, "PUT", "/profile", "", map[string]any{"email": "x@uji.local"})
	assert.Equal(t, 401, code)

	// user terhapus -> 404
	require.NoError(t, db.Where("username = ?", "carol").Delete(&database.User{}).Error)
	code, _ = call(r, "GET", "/profile", token, nil)
	assert.Equal(t, 404, code)
	code, _ = call(r, "PUT", "/profile", token, map[string]any{"email": "z@uji.local"})
	assert.Equal(t, 404, code)
}

func toJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func TestServerErrors(t *testing.T) {
	// password terlalu panjang -> bcrypt gagal
	r, db := setup(t, "auth_errors")
	long := strings.Repeat("p", 80)
	code, m := call(r, "POST", "/register", "", map[string]any{"username": "eve", "email": "eve@uji.local", "password": long})
	assert.Equal(t, 500, code)
	assert.Equal(t, "error", m["status"])

	token := register(t, r, "frank", nil)
	code, _ = call(r, "PUT", "/profile", token, map[string]any{"password": long})
	assert.Equal(t, 500, code)

	// Save gagal
	_ = db.Callback().Update().Before("gorm:update").Register("fail_update", func(tx *gorm.DB) { _ = tx.AddError(errors.New("forced")) })
	code, _ = call(r, "PUT", "/profile", token, map[string]any{"email": "frank2@uji.local"})
	assert.Equal(t, 500, code)

	// Create gagal
	_ = db.Callback().Create().Before("gorm:create").Register("fail_create", func(tx *gorm.DB) { _ = tx.AddError(errors.New("forced")) })
	code, _ = call(r, "POST", "/register", "", map[string]any{"username": "gina", "email": "gina@uji.local", "password": "Password123!"})
	assert.Equal(t, 500, code)

	// Count (cek duplikat) gagal pada register & update profil
	failQuery(db, "fail_count", func(tx *gorm.DB) bool { _, ok := tx.Statement.Dest.(*int64); return ok })
	code, _ = call(r, "POST", "/register", "", map[string]any{"username": "hana", "email": "hana@uji.local", "password": "Password123!"})
	assert.Equal(t, 500, code)
	code, _ = call(r, "PUT", "/profile", token, map[string]any{"email": "frank3@uji.local"})
	assert.Equal(t, 500, code)
}

func TestQueryErrors(t *testing.T) {
	r, db := setup(t, "auth_query_errors")
	token := register(t, r, "ivan", nil)
	sqlDB, _ := db.DB()
	require.NoError(t, sqlDB.Close())

	code, _ := call(r, "POST", "/login", "", map[string]any{"username": "ivan", "password": "Password123!"})
	assert.Equal(t, 500, code)
	code, _ = call(r, "GET", "/profile", token, nil)
	assert.Equal(t, 500, code)
	code, _ = call(r, "PUT", "/profile", token, map[string]any{"email": "i2@uji.local"})
	assert.Equal(t, 500, code)
}

func TestExistsBranches(t *testing.T) {
	_, db := setup(t, "auth_exists")
	repo := NewRepository(db)
	require.NoError(t, repo.Create(&database.User{Username: "u1", Email: "u1@x.y", Password: "h", Role: "member"}))

	ok, err := repo.Exists("u1", "", 0)
	assert.NoError(t, err)
	assert.True(t, ok)
	ok, _ = repo.Exists("", "u1@x.y", 0)
	assert.True(t, ok)
	ok, _ = repo.Exists("zzz", "zzz@x.y", 0)
	assert.False(t, ok)

}
