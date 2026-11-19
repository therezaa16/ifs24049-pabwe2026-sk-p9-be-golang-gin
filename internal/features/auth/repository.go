package auth

import (
	"app/internal/database"

	"gorm.io/gorm"
)

// Repository mengakses tabel users.
type Repository struct{ db *gorm.DB }

// NewRepository membuat repository user.
func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// Create menyimpan user baru.
func (r *Repository) Create(u *database.User) error { return r.db.Create(u).Error }

// Exists memeriksa duplikat username/email, mengabaikan user dengan ID excludeID.
func (r *Repository) Exists(username, email string, excludeID uint) (bool, error) {
	q := r.db.Model(&database.User{}).Where("id <> ?", excludeID)
	switch {
	case username != "" && email != "":
		q = q.Where("username = ? OR email = ?", username, email)
	case username != "":
		q = q.Where("username = ?", username)
	default:
		q = q.Where("email = ?", email)
	}
	var n int64
	err := q.Count(&n).Error
	return n > 0, err
}

// FindByLogin mencari user berdasarkan username atau email.
func (r *Repository) FindByLogin(identity string) (*database.User, error) {
	var u database.User
	err := r.db.Where("username = ? OR email = ?", identity, identity).First(&u).Error
	return &u, err
}

// FindByID mencari user berdasarkan ID.
func (r *Repository) FindByID(id uint) (*database.User, error) {
	var u database.User
	err := r.db.First(&u, id).Error
	return &u, err
}

// Save menyimpan perubahan user.
func (r *Repository) Save(u *database.User) error { return r.db.Save(u).Error }
