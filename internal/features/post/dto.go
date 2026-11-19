package post

import (
	"time"

	"app/internal/database"
)

// Categories adalah daftar kategori resmi blog.
var Categories = []string{"Teknologi", "Pendidikan", "Gaya Hidup", "Bisnis", "Opini", "Umum"}

// CreatePostRequest payload pembuatan artikel.
type CreatePostRequest struct {
	Title          string `json:"title" binding:"required"`
	Category       string `json:"category"`
	Excerpt        string `json:"excerpt"`
	Content        string `json:"content" binding:"required"`
	CoverImagePath string `json:"cover_image_path"`
	IsPublished    bool   `json:"is_published"`
}

// UpdatePostRequest payload pembaruan; field kosong diabaikan.
type UpdatePostRequest struct {
	Title          string `json:"title"`
	Category       string `json:"category"`
	Excerpt        string `json:"excerpt"`
	Content        string `json:"content"`
	CoverImagePath string `json:"cover_image_path"`
	IsPublished    *bool  `json:"is_published"`
}

// PostResponse dipakai endpoint penulis & admin (memuat user_id).
type PostResponse struct {
	ID             uint       `json:"id"`
	UserID         uint       `json:"user_id"`
	AuthorName     string     `json:"author_name,omitempty"`
	Title          string     `json:"title"`
	Slug           string     `json:"slug"`
	Category       string     `json:"category"`
	Excerpt        string     `json:"excerpt"`
	Content        string     `json:"content"`
	CoverImagePath string     `json:"cover_image_path"`
	IsPublished    bool       `json:"is_published"`
	PublishedAt    *time.Time `json:"published_at"`
	ReadingTime    int        `json:"reading_time"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// PublicPostResponse dipakai endpoint publik (TANPA user_id).
type PublicPostResponse struct {
	ID             uint       `json:"id"`
	Title          string     `json:"title"`
	Slug           string     `json:"slug"`
	Category       string     `json:"category"`
	Excerpt        string     `json:"excerpt"`
	Content        string     `json:"content"`
	CoverImagePath string     `json:"cover_image_path"`
	AuthorName     string     `json:"author_name"`
	ReadingTime    int        `json:"reading_time"`
	PublishedAt    *time.Time `json:"published_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

// Stats adalah ringkasan 3 kartu statistik penulis.
type Stats struct {
	TotalArticles int64 `json:"total_articles"`
	TotalSelesai  int64 `json:"total_selesai"`
	BelumSelesai  int64 `json:"belum_selesai"`
}

// UserSummary adalah baris monitoring admin untuk satu pengguna.
type UserSummary struct {
	ID             uint      `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
	TotalArticles  int64     `json:"total_articles"`
	TotalPublished int64     `json:"total_published"`
}

func toResponse(p *database.Post) PostResponse {
	return PostResponse{
		ID: p.ID, UserID: p.UserID, AuthorName: p.User.Username,
		Title: p.Title, Slug: p.Slug, Category: p.Category, Excerpt: p.Excerpt,
		Content: p.Content, CoverImagePath: p.CoverImagePath,
		IsPublished: p.IsPublished, PublishedAt: p.PublishedAt,
		ReadingTime: ReadingTime(p.Content),
		CreatedAt:   p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toPublic(p *database.Post) PublicPostResponse {
	return PublicPostResponse{
		ID: p.ID, Title: p.Title, Slug: p.Slug, Category: p.Category,
		Excerpt: p.Excerpt, Content: p.Content, CoverImagePath: p.CoverImagePath,
		AuthorName: p.User.Username, ReadingTime: ReadingTime(p.Content),
		PublishedAt: p.PublishedAt, CreatedAt: p.CreatedAt,
	}
}
