// Package database mengelola koneksi, migrasi, dan seeder.
package database

import (
	"time"

	"app/internal/config"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect membuka koneksi PostgreSQL untuk runtime aplikasi.
func Connect(cfg *config.Config) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
}

// OpenSQLite membuka SQLite in-memory (dipakai unit test mandiri).
func OpenSQLite(name string) (*gorm.DB, error) {
	db, err := gorm.Open(
		sqlite.Open("file:"+name+"?mode=memory&cache=shared"),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)},
	)
	if err != nil {
		return nil, err
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	return db, nil
}

// Migrate menjalankan AutoMigrate seluruh tabel.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&User{}, &Post{})
}

func hash(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

// Seed membuat akun admin & member bawaan serta artikel contoh bila tabel masih kosong.
func Seed(db *gorm.DB, cfg *config.Config) error {
	accounts := []struct{ username, email, password, role string }{
		{cfg.AdminUsername, "admin@delcom.org", cfg.AdminPassword, "admin"},
		{"member", "member@delcom.org", "member", "member"},
	}
	var admin User
	for _, a := range accounts {
		var u User
		res := db.Where("username = ?", a.username).Limit(1).Find(&u)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			h, err := hash(a.password)
			if err != nil {
				return err
			}
			u = User{Username: a.username, Email: a.email, Password: h, Role: a.role}
			if err := db.Create(&u).Error; err != nil {
				return err
			}
		}
		if a.role == "admin" {
			admin = u
		}
	}

	var total int64
	if err := db.Model(&Post{}).Count(&total).Error; err != nil {
		return err
	}
	if total > 0 {
		return nil
	}
	now := time.Now()
	samples := []Post{
		{
			Title: "Memulai Backend dengan Golang dan Gin", Slug: "memulai-backend-dengan-golang-dan-gin-100001",
			Category: "Teknologi",
			Excerpt:  "Golang menawarkan performa tinggi dan sintaks sederhana untuk membangun REST API modern.",
			Content: "# Memulai Backend dengan Golang\n\nGolang adalah bahasa pemrograman yang cepat dan efisien. " +
				"Dipadukan dengan framework Gin, kita dapat membangun REST API dengan routing yang ringkas, " +
				"middleware yang fleksibel, dan performa yang sangat baik.\n\n## Mengapa Gin?\n\n" +
				"- Routing cepat berbasis radix tree\n- Middleware yang mudah disusun\n- Binding dan validasi JSON bawaan\n",
		},
		{
			Title: "Belajar Efektif di Era Digital", Slug: "belajar-efektif-di-era-digital-100002",
			Category: "Pendidikan",
			Excerpt:  "Tips menjaga fokus dan konsistensi belajar ketika informasi datang dari segala arah.",
			Content: "# Belajar Efektif di Era Digital\n\nBelajar membutuhkan fokus. Tetapkan tujuan kecil setiap hari, " +
				"gunakan teknik pomodoro, dan evaluasi kemajuan secara berkala agar kebiasaan belajar tetap terjaga.\n",
		},
		{
			Title: "Menyusun Rutinitas Pagi yang Sehat", Slug: "menyusun-rutinitas-pagi-yang-sehat-100003",
			Category: "Gaya Hidup",
			Excerpt:  "Rutinitas pagi sederhana dapat meningkatkan energi dan suasana hati sepanjang hari.",
			Content: "# Rutinitas Pagi yang Sehat\n\nMulailah hari dengan minum air putih, peregangan ringan, " +
				"dan sarapan seimbang. Konsistensi lebih penting daripada durasi.\n",
		},
	}
	for i := range samples {
		t := now.Add(-time.Duration(i) * time.Hour)
		samples[i].UserID = admin.ID
		samples[i].IsPublished = true
		samples[i].PublishedAt = &t
	}
	return db.Create(&samples).Error
}
