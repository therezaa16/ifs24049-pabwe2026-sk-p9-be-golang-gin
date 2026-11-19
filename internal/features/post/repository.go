package post

import (
	"strings"

	"app/internal/database"

	"gorm.io/gorm"
)

// Repository mengakses tabel posts.
type Repository struct{ db *gorm.DB }

// NewRepository membuat repository artikel.
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) withAuthor() *gorm.DB {
	return r.db.Preload("User", func(tx *gorm.DB) *gorm.DB { return tx.Select("id", "username") })
}

// Create menyimpan artikel baru.
func (r *Repository) Create(p *database.Post) error { return r.db.Create(p).Error }

// Save menyimpan perubahan artikel.
func (r *Repository) Save(p *database.Post) error { return r.db.Save(p).Error }

// Delete menghapus artikel.
func (r *Repository) Delete(p *database.Post) error { return r.db.Delete(p).Error }

// FindByID mengambil artikel berdasarkan ID.
func (r *Repository) FindByID(id uint) (*database.Post, error) {
	var p database.Post
	err := r.withAuthor().First(&p, id).Error
	return &p, err
}

// FindPublishedBySlug mengambil artikel terbit berdasarkan slug.
func (r *Repository) FindPublishedBySlug(slug string) (*database.Post, error) {
	var p database.Post
	err := r.withAuthor().Where("slug = ? AND is_published = ?", slug, true).First(&p).Error
	return &p, err
}

// ListPublished mengambil artikel terbit dengan filter pencarian & kategori.
func (r *Repository) ListPublished(q, category string) ([]database.Post, error) {
	tx := r.withAuthor().Where("is_published = ?", true)
	if q = strings.ToLower(strings.TrimSpace(q)); q != "" {
		like := "%" + q + "%"
		tx = tx.Where("LOWER(title) LIKE ? OR LOWER(excerpt) LIKE ? OR LOWER(content) LIKE ?", like, like, like)
	}
	if category = strings.TrimSpace(category); category != "" && !strings.EqualFold(category, "Semua") {
		tx = tx.Where("LOWER(category) = ?", strings.ToLower(category))
	}
	posts := []database.Post{}
	err := tx.Order("published_at DESC, id DESC").Find(&posts).Error
	return posts, err
}

// ListByUser mengambil seluruh artikel milik satu penulis.
func (r *Repository) ListByUser(userID uint) ([]database.Post, error) {
	posts := []database.Post{}
	err := r.withAuthor().Where("user_id = ?", userID).Order("id DESC").Find(&posts).Error
	return posts, err
}

// ListAll mengambil seluruh artikel semua penulis.
func (r *Repository) ListAll() ([]database.Post, error) {
	posts := []database.Post{}
	err := r.withAuthor().Order("id DESC").Find(&posts).Error
	return posts, err
}

// Count menghitung total & terbit milik satu penulis.
func (r *Repository) Count(userID uint) (total, published int64, err error) {
	base := r.db.Model(&database.Post{}).Where("user_id = ?", userID)
	if err = base.Count(&total).Error; err != nil {
		return
	}
	err = r.db.Model(&database.Post{}).
		Where("user_id = ? AND is_published = ?", userID, true).Count(&published).Error
	return
}

// UsersWithCounts mengambil semua user beserta jumlah artikelnya.
func (r *Repository) UsersWithCounts() ([]UserSummary, error) {
	rows := []UserSummary{}
	err := r.db.Raw(`
		SELECT u.id, u.username, u.email, u.role, u.created_at,
		       COUNT(p.id) AS total_articles,
		       COALESCE(SUM(CASE WHEN p.is_published THEN 1 ELSE 0 END), 0) AS total_published
		FROM users u LEFT JOIN posts p ON p.user_id = u.id
		GROUP BY u.id, u.username, u.email, u.role, u.created_at
		ORDER BY u.id`).Scan(&rows).Error
	return rows, err
}
