package post

import (
	"errors"
	"net/http"
	"strconv"

	"app/internal/database"
	"app/internal/middleware"
	"app/internal/response"

	"github.com/gin-gonic/gin"
)

// Handler adalah controller /api/public/*, /api/posts/*, dan /api/admin/*.
type Handler struct{ svc *Service }

// NewHandler membuat handler artikel.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrForbidden):
		response.Fail(c, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrInvalid):
		response.Fail(c, http.StatusBadRequest, err.Error())
	default:
		response.Error(c, "Terjadi kesalahan pada server")
	}
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID harus berupa angka")
		return 0, false
	}
	return uint(id), true
}

func responses(posts []database.Post) []PostResponse {
	out := make([]PostResponse, 0, len(posts))
	for i := range posts {
		out = append(out, toResponse(&posts[i]))
	}
	return out
}

// ---------- Publik ----------

// PublicCategories godoc: GET /api/public/categories
func (h *Handler) PublicCategories(c *gin.Context) {
	response.Success(c, http.StatusOK, "Kategori berhasil diambil", Categories)
}

// PublicPosts godoc: GET /api/public/posts?q=&category=
func (h *Handler) PublicPosts(c *gin.Context) {
	posts, err := h.svc.PublicList(c.Query("q"), c.Query("category"))
	if err != nil {
		h.fail(c, err)
		return
	}
	out := make([]PublicPostResponse, 0, len(posts))
	for i := range posts {
		out = append(out, toPublic(&posts[i]))
	}
	response.Success(c, http.StatusOK, "Artikel berhasil diambil", out)
}

// PublicPostDetail godoc: GET /api/public/posts/:slug
func (h *Handler) PublicPostDetail(c *gin.Context) {
	p, err := h.svc.PublicDetail(c.Param("slug"))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Artikel berhasil diambil", toPublic(p))
}

// ---------- Penulis ----------

// List godoc: GET /api/posts
func (h *Handler) List(c *gin.Context) {
	posts, err := h.svc.List(middleware.UserID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Artikel berhasil diambil", responses(posts))
}

// Create godoc: POST /api/posts
func (h *Handler) Create(c *gin.Context) {
	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.BindMessage(err))
		return
	}
	p, err := h.svc.Create(middleware.UserID(c), req)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Artikel berhasil dibuat", toResponse(p))
}

// Stats godoc: GET /api/posts/stats
func (h *Handler) Stats(c *gin.Context) {
	st, err := h.svc.Stats(middleware.UserID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Statistik berhasil diambil", st)
}

// Get godoc: GET /api/posts/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := h.svc.Get(middleware.UserID(c), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Artikel berhasil diambil", toResponse(p))
}

// Update godoc: PUT /api/posts/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.BindMessage(err))
		return
	}
	p, err := h.svc.Update(middleware.UserID(c), id, req)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Artikel berhasil diperbarui", toResponse(p))
}

// TogglePublish godoc: PATCH /api/posts/:id/toggle-publish
func (h *Handler) TogglePublish(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := h.svc.TogglePublish(middleware.UserID(c), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Status publikasi berhasil diubah", toResponse(p))
}

// Delete godoc: DELETE /api/posts/:id
func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(middleware.UserID(c), id); err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Artikel berhasil dihapus", nil)
}

// ---------- Admin ----------

// AdminUsers godoc: GET /api/admin/users
func (h *Handler) AdminUsers(c *gin.Context) {
	users, err := h.svc.AdminUsers()
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Pengguna berhasil diambil", users)
}

// AdminPosts godoc: GET /api/admin/posts
func (h *Handler) AdminPosts(c *gin.Context) {
	posts, err := h.svc.AdminPosts()
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Artikel berhasil diambil", responses(posts))
}

// AdminUserPosts godoc: GET /api/admin/users/:id/posts
func (h *Handler) AdminUserPosts(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	posts, err := h.svc.List(id)
	if err != nil {
		h.fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Artikel pengguna berhasil diambil", responses(posts))
}
