package post

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"app/internal/database"
	"app/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const secret = "post-test-secret"

type env struct {
	r  *gin.Engine
	db *gorm.DB
}

func tokenFor(t *testing.T, id uint, role string) string { return signToken(t, id, role) }

func setup(t *testing.T, name string) *env {
	gin.SetMode(gin.TestMode)
	db, err := database.OpenSQLite(name)
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	for _, u := range []database.User{
		{Username: "penulis", Email: "p@x.y", Password: "h", Role: "member"},
		{Username: "lain", Email: "l@x.y", Password: "h", Role: "member"},
		{Username: "admin", Email: "a@x.y", Password: "h", Role: "admin"},
	} {
		u := u
		require.NoError(t, db.Create(&u).Error)
	}
	h := NewHandler(NewService(NewRepository(db)))
	r := gin.New()
	pub := r.Group("/public")
	pub.GET("/categories", h.PublicCategories)
	pub.GET("/posts", h.PublicPosts)
	pub.GET("/posts/:slug", h.PublicPostDetail)
	guard := middleware.Auth(secret)
	p := r.Group("/posts", guard)
	p.GET("", h.List)
	p.POST("", h.Create)
	p.GET("/stats", h.Stats)
	p.GET("/:id", h.Get)
	p.PUT("/:id", h.Update)
	p.PATCH("/:id/toggle-publish", h.TogglePublish)
	p.DELETE("/:id", h.Delete)
	a := r.Group("/admin", guard, middleware.RequireRole("admin"))
	a.GET("/users", h.AdminUsers)
	a.GET("/posts", h.AdminPosts)
	a.GET("/users/:id/posts", h.AdminUserPosts)
	return &env{r: r, db: db}
}

var (
	penulis = func(t *testing.T) string { return tokenFor(t, 1, "member") }
	lain    = func(t *testing.T) string { return tokenFor(t, 2, "member") }
	admin   = func(t *testing.T) string { return tokenFor(t, 3, "admin") }
)

func (e *env) call(method, path, token string, body any) (int, map[string]any) {
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
	e.r.ServeHTTP(w, req)
	var m map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	return w.Code, m
}

func obj(m map[string]any) map[string]any { d, _ := m["data"].(map[string]any); return d }
func list(m map[string]any) []any         { d, _ := m["data"].([]any); return d }

var counter atomic.Int64

func (e *env) create(t *testing.T, token string, extra map[string]any) map[string]any {
	body := map[string]any{
		"title": fmt.Sprintf("Artikel Uji %d", counter.Add(1)), "category": "Teknologi",
		"excerpt": "ringkasan", "content": "# Isi\n\nKonten tentang Golang.", "is_published": true,
	}
	for k, v := range extra {
		body[k] = v
	}
	code, m := e.call("POST", "/posts", token, body)
	require.Equal(t, 201, code)
	return obj(m)
}

func TestHelpers(t *testing.T) {
	assert.Equal(t, "artikel-uji-abc-123", Slugify("Artikel Uji ABC 123!"))
	assert.Equal(t, "artikel", Slugify("!!!"))
	assert.Equal(t, "a-b", Slugify("  a --- b  "))
	assert.LessOrEqual(t, len(Slugify(strings.Repeat("kata ", 100))), 180)
	assert.NotEqual(t, uniqueSlug("x"), uniqueSlug("x"))

	assert.Equal(t, 1, ReadingTime(""))
	assert.Equal(t, 1, ReadingTime(strings.Repeat("kata ", 200)))
	assert.Equal(t, 2, ReadingTime(strings.Repeat("kata ", 201)))
	assert.Equal(t, "Umum", normalizeCategory("  "))
	assert.Equal(t, "Bisnis", normalizeCategory(" Bisnis "))
}

func TestPublicEndpoints(t *testing.T) {
	e := setup(t, "post_public")
	tk := penulis(t)
	e.create(t, tk, map[string]any{"title": "Belajar Golang", "content": "Isi tentang golang yang panjang."})
	e.create(t, tk, map[string]any{"title": "Bisnis Kecil", "category": "Bisnis", "content": "Isi bisnis."})
	e.create(t, tk, map[string]any{"title": "Draf Rahasia", "is_published": false})

	code, m := e.call("GET", "/public/categories", "", nil)
	assert.Equal(t, 200, code)
	assert.Len(t, list(m), 6)

	code, m = e.call("GET", "/public/posts", "", nil)
	assert.Equal(t, 200, code)
	assert.Len(t, list(m), 2) // draf tidak tampil
	raw, _ := json.Marshal(m)
	assert.NotContains(t, strings.ToLower(string(raw)), "user_id")
	first := list(m)[0].(map[string]any)
	assert.NotEmpty(t, first["author_name"])
	assert.GreaterOrEqual(t, first["reading_time"].(float64), float64(1))

	_, m = e.call("GET", "/public/posts?category=Bisnis", "", nil)
	assert.Len(t, list(m), 1)
	_, m = e.call("GET", "/public/posts?category=Semua", "", nil)
	assert.Len(t, list(m), 2)
	_, m = e.call("GET", "/public/posts?q=GOLANG", "", nil)
	assert.Len(t, list(m), 1)
	_, m = e.call("GET", "/public/posts?q=zzztidakada", "", nil)
	assert.NotNil(t, m["data"])
	assert.Len(t, list(m), 0)

	slug := first["slug"].(string)
	code, m = e.call("GET", "/public/posts/"+slug, "", nil)
	assert.Equal(t, 200, code)
	assert.Equal(t, slug, obj(m)["slug"])
	code, m = e.call("GET", "/public/posts/tidak-ada", "", nil)
	assert.Equal(t, 404, code)
	assert.Equal(t, "fail", m["status"])

	draf := e.create(t, tk, map[string]any{"is_published": false})
	code, _ = e.call("GET", "/public/posts/"+draf["slug"].(string), "", nil)
	assert.Equal(t, 404, code)
}

func TestAuthorCRUD(t *testing.T) {
	e := setup(t, "post_crud")
	tk, other := penulis(t), lain(t)

	p := e.create(t, tk, nil)
	id := fmt.Sprint(p["id"])
	assert.Regexp(t, `^[a-z0-9-]+$`, p["slug"])
	assert.Contains(t, p["slug"], "artikel-uji")
	assert.NotNil(t, p["published_at"])
	assert.EqualValues(t, 1, p["user_id"])

	// kategori kosong -> Umum, draf tanpa published_at
	d := e.create(t, tk, map[string]any{"category": "", "is_published": false})
	assert.Equal(t, "Umum", d["category"])
	assert.Nil(t, d["published_at"])

	// create gagal
	code, _ := e.call("POST", "/posts", tk, map[string]any{"content": "x"})
	assert.Equal(t, 400, code)
	code, _ = e.call("POST", "/posts", tk, map[string]any{"title": "x"})
	assert.Equal(t, 400, code)
	code, _ = e.call("POST", "/posts", tk, map[string]any{"title": "  ", "content": "x"})
	assert.Equal(t, 400, code)
	code, _ = e.call("POST", "/posts", "", map[string]any{"title": "x", "content": "y"})
	assert.Equal(t, 401, code)

	// read
	code, m := e.call("GET", "/posts", tk, nil)
	assert.Equal(t, 200, code)
	assert.Len(t, list(m), 2)
	_, m = e.call("GET", "/posts", other, nil)
	assert.Len(t, list(m), 0)

	code, m = e.call("GET", "/posts/"+id, tk, nil)
	assert.Equal(t, 200, code)
	assert.Equal(t, p["title"], obj(m)["title"])
	assert.Equal(t, 404, func() int { c, _ := e.call("GET", "/posts/99999999", tk, nil); return c }())
	assert.Equal(t, 400, func() int { c, _ := e.call("GET", "/posts/abc", tk, nil); return c }())
	assert.Equal(t, 403, func() int { c, _ := e.call("GET", "/posts/"+id, other, nil); return c }())
	assert.Equal(t, 401, func() int { c, _ := e.call("GET", "/posts/"+id, "", nil); return c }())

	// stats
	code, m = e.call("GET", "/posts/stats", tk, nil)
	assert.Equal(t, 200, code)
	st := obj(m)
	assert.EqualValues(t, 2, st["total_articles"])
	assert.EqualValues(t, 1, st["total_selesai"])
	assert.EqualValues(t, 1, st["belum_selesai"])

	// update
	code, m = e.call("PUT", "/posts/"+id, tk, map[string]any{
		"title": "Judul Baru", "category": "Opini", "excerpt": "ex", "content": "Konten baru", "cover_image_path": "/img.png",
	})
	assert.Equal(t, 200, code)
	assert.Equal(t, "Judul Baru", obj(m)["title"])
	assert.Equal(t, "Opini", obj(m)["category"])
	code, m = e.call("PUT", "/posts/"+id, tk, map[string]any{}) // kosong: tidak menimpa
	assert.Equal(t, 200, code)
	assert.Equal(t, "Judul Baru", obj(m)["title"])
	code, m = e.call("PUT", "/posts/"+id, tk, map[string]any{"is_published": false})
	assert.Nil(t, obj(m)["published_at"])
	code, m = e.call("PUT", "/posts/"+id, tk, map[string]any{"is_published": true})
	assert.NotNil(t, obj(m)["published_at"])
	assert.Equal(t, 400, func() int { c, _ := e.call("PUT", "/posts/"+id, tk, "{rusak"); return c }())
	assert.Equal(t, 400, func() int { c, _ := e.call("PUT", "/posts/abc", tk, map[string]any{}); return c }())
	assert.Equal(t, 404, func() int { c, _ := e.call("PUT", "/posts/99999999", tk, map[string]any{}); return c }())
	assert.Equal(t, 403, func() int { c, _ := e.call("PUT", "/posts/"+id, other, map[string]any{"title": "x"}); return c }())
	assert.Equal(t, 401, func() int { c, _ := e.call("PUT", "/posts/"+id, "", map[string]any{}); return c }())

	// toggle
	did := fmt.Sprint(d["id"])
	code, m = e.call("PATCH", "/posts/"+did+"/toggle-publish", tk, nil)
	assert.Equal(t, 200, code)
	assert.Equal(t, true, obj(m)["is_published"])
	assert.NotNil(t, obj(m)["published_at"])
	_, m = e.call("PATCH", "/posts/"+did+"/toggle-publish", tk, nil)
	assert.Equal(t, false, obj(m)["is_published"])
	assert.Equal(t, 400, func() int { c, _ := e.call("PATCH", "/posts/abc/toggle-publish", tk, nil); return c }())
	assert.Equal(t, 404, func() int { c, _ := e.call("PATCH", "/posts/99999999/toggle-publish", tk, nil); return c }())
	assert.Equal(t, 403, func() int { c, _ := e.call("PATCH", "/posts/"+did+"/toggle-publish", other, nil); return c }())

	// delete
	assert.Equal(t, 400, func() int { c, _ := e.call("DELETE", "/posts/abc", tk, nil); return c }())
	assert.Equal(t, 404, func() int { c, _ := e.call("DELETE", "/posts/99999999", tk, nil); return c }())
	assert.Equal(t, 403, func() int { c, _ := e.call("DELETE", "/posts/"+id, other, nil); return c }())
	assert.Equal(t, 401, func() int { c, _ := e.call("DELETE", "/posts/"+id, "", nil); return c }())
	code, m = e.call("DELETE", "/posts/"+id, tk, nil)
	assert.Equal(t, 200, code)
	assert.Equal(t, "success", m["status"])
	assert.Equal(t, 404, func() int { c, _ := e.call("GET", "/posts/"+id, tk, nil); return c }())
}

func TestAdminEndpoints(t *testing.T) {
	e := setup(t, "post_admin")
	e.create(t, penulis(t), nil)
	e.create(t, penulis(t), map[string]any{"is_published": false})
	e.create(t, lain(t), nil)
	adm := admin(t)

	code, m := e.call("GET", "/admin/users", adm, nil)
	assert.Equal(t, 200, code)
	users := list(m)
	assert.Len(t, users, 3)
	first := users[0].(map[string]any)
	assert.EqualValues(t, 2, first["total_articles"])
	assert.EqualValues(t, 1, first["total_published"])
	raw, _ := json.Marshal(m)
	assert.NotContains(t, strings.ToLower(string(raw)), "password")

	_, m = e.call("GET", "/admin/posts", adm, nil)
	assert.Len(t, list(m), 3)

	_, m = e.call("GET", "/admin/users/1/posts", adm, nil)
	assert.Len(t, list(m), 2)
	code, m = e.call("GET", "/admin/users/99999999/posts", adm, nil)
	assert.Equal(t, 200, code)
	assert.Len(t, list(m), 0)
	assert.Equal(t, 400, func() int { c, _ := e.call("GET", "/admin/users/abc/posts", adm, nil); return c }())

	assert.Equal(t, 401, func() int { c, _ := e.call("GET", "/admin/users", "", nil); return c }())
	assert.Equal(t, 403, func() int { c, _ := e.call("GET", "/admin/users", penulis(t), nil); return c }())
	assert.Equal(t, 403, func() int { c, _ := e.call("GET", "/admin/posts", penulis(t), nil); return c }())
}

func forceErr(tx *gorm.DB) { _ = tx.AddError(errors.New("forced")) }

func TestServerErrorsWithClosedDB(t *testing.T) {
	e := setup(t, "post_closed")
	tk, adm := penulis(t), admin(t)
	p := e.create(t, tk, nil)
	id := fmt.Sprint(p["id"])
	sqlDB, _ := e.db.DB()
	require.NoError(t, sqlDB.Close())

	for _, c := range []struct{ method, path, token string }{
		{"GET", "/public/posts", ""},
		{"GET", "/public/posts/x", ""},
		{"GET", "/posts", tk},
		{"GET", "/posts/stats", tk},
		{"GET", "/posts/" + id, tk},
		{"PUT", "/posts/" + id, tk},
		{"PATCH", "/posts/" + id + "/toggle-publish", tk},
		{"DELETE", "/posts/" + id, tk},
		{"GET", "/admin/users", adm},
		{"GET", "/admin/posts", adm},
		{"GET", "/admin/users/1/posts", adm},
	} {
		code, m := e.call(c.method, c.path, c.token, map[string]any{})
		assert.Equal(t, 500, code, c.method+" "+c.path)
		assert.Equal(t, "error", m["status"])
	}
	code, _ := e.call("POST", "/posts", tk, map[string]any{"title": "x", "content": "y"})
	assert.Equal(t, 500, code)
}

func TestWriteErrors(t *testing.T) {
	e := setup(t, "post_write_err")
	tk := penulis(t)
	p := e.create(t, tk, nil)
	id := fmt.Sprint(p["id"])

	_ = e.db.Callback().Update().Before("gorm:update").Register("f_update", forceErr)
	_ = e.db.Callback().Delete().Before("gorm:delete").Register("f_delete", forceErr)
	_ = e.db.Callback().Create().Before("gorm:create").Register("f_create", forceErr)

	assert.Equal(t, 500, func() int { c, _ := e.call("PUT", "/posts/"+id, tk, map[string]any{"title": "x"}); return c }())
	assert.Equal(t, 500, func() int { c, _ := e.call("PATCH", "/posts/"+id+"/toggle-publish", tk, nil); return c }())
	assert.Equal(t, 500, func() int { c, _ := e.call("DELETE", "/posts/"+id, tk, nil); return c }())
	assert.Equal(t, 500, func() int {
		c, _ := e.call("POST", "/posts", tk, map[string]any{"title": "x", "content": "y"})
		return c
	}())
}

func TestStatsSecondCountError(t *testing.T) {
	e := setup(t, "post_stats_err")
	tk := penulis(t)
	var n int
	_ = e.db.Callback().Query().Before("gorm:query").Register("f_second", func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*int64); ok {
			n++
			if n == 2 {
				_ = tx.AddError(errors.New("forced"))
			}
		}
	})
	code, _ := e.call("GET", "/posts/stats", tk, nil)
	assert.Equal(t, 500, code)
}
