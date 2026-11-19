package database

import (
	"errors"
	"strings"
	"testing"

	"app/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func cfg() *config.Config {
	return &config.Config{AdminUsername: "admin", AdminPassword: "admin"}
}

func newDB(t *testing.T, name string) *gorm.DB {
	db, err := OpenSQLite(name)
	require.NoError(t, err)
	require.NoError(t, Migrate(db))
	return db
}

func failOn(db *gorm.DB, kind string) {
	fail := func(tx *gorm.DB) { _ = tx.AddError(errors.New("forced")) }
	switch kind {
	case "create":
		_ = db.Callback().Create().Before("gorm:create").Register("fail", fail)
	}
}

func TestConnectFails(t *testing.T) {
	_, err := Connect(&config.Config{DBHost: "127.0.0.1", DBPort: "1", DBUsername: "x", DBPassword: "x", DBDatabase: "x", DBSSLMode: "disable"})
	assert.Error(t, err)
}

func TestSeedIdempotent(t *testing.T) {
	db := newDB(t, "seed_ok")
	require.NoError(t, Seed(db, cfg()))
	require.NoError(t, Seed(db, cfg()))

	var users, posts int64
	db.Model(&User{}).Count(&users)
	db.Model(&Post{}).Count(&posts)
	assert.EqualValues(t, 2, users)
	assert.EqualValues(t, 3, posts)

	var admin User
	require.NoError(t, db.Where("username = ?", "admin").First(&admin).Error)
	assert.Equal(t, "admin", admin.Role)
	assert.NotEqual(t, "admin", admin.Password)
}

func TestSeedFindError(t *testing.T) {
	db, err := OpenSQLite("seed_find")
	require.NoError(t, err) // tabel belum dimigrasi
	assert.Error(t, Seed(db, cfg()))
}

func TestSeedHashError(t *testing.T) {
	db := newDB(t, "seed_hash")
	c := cfg()
	c.AdminPassword = strings.Repeat("x", 100)
	assert.Error(t, Seed(db, c))
}

func TestSeedCreateUserError(t *testing.T) {
	db := newDB(t, "seed_create_user")
	failOn(db, "create")
	assert.Error(t, Seed(db, cfg()))
}

func TestSeedCountError(t *testing.T) {
	db, err := OpenSQLite("seed_count")
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{})) // tabel posts tidak ada
	assert.Error(t, Seed(db, cfg()))
}

func TestSeedCreatePostsError(t *testing.T) {
	db := newDB(t, "seed_create_posts")
	h, _ := hash("x")
	require.NoError(t, db.Create(&User{Username: "admin", Email: "a@a.a", Password: h, Role: "admin"}).Error)
	require.NoError(t, db.Create(&User{Username: "member", Email: "m@m.m", Password: h, Role: "member"}).Error)
	failOn(db, "create")
	assert.Error(t, Seed(db, cfg()))
}

func TestOpenSQLiteError(t *testing.T) {
	// mode URI tidak valid membuat SQLite menolak koneksi
	db, err := OpenSQLite("bad?mode=bogus&x")
	assert.Error(t, err)
	assert.Nil(t, db)
}
