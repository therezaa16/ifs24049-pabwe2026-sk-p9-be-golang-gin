package post

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"app/internal/database"

	"gorm.io/gorm"
)

// Galat domain artikel.
var (
	ErrNotFound  = errors.New("artikel tidak ditemukan")
	ErrForbidden = errors.New("anda tidak berhak mengakses artikel ini")
	ErrInvalid   = errors.New("judul dan konten wajib diisi")
)

var slugCounter atomic.Uint64

// Slugify mengubah judul menjadi slug huruf kecil [a-z0-9-].
func Slugify(title string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(title) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	s := strings.Trim(b.String(), "-")
	if len(s) > 180 {
		s = strings.Trim(s[:180], "-")
	}
	if s == "" {
		s = "artikel"
	}
	return s
}

func uniqueSlug(title string) string {
	n := strconv.FormatInt(time.Now().UnixNano()/1000, 10) + strconv.FormatUint(slugCounter.Add(1)%100, 10)
	return Slugify(title) + "-" + n
}

// ReadingTime menghitung estimasi menit baca (kata / 200 wpm, minimal 1).
func ReadingTime(content string) int {
	m := int(math.Ceil(float64(len(strings.Fields(content))) / 200.0))
	if m < 1 {
		return 1
	}
	return m
}

func normalizeCategory(c string) string {
	c = strings.TrimSpace(c)
	if c == "" {
		return "Umum"
	}
	return c
}

// Service berisi logika bisnis artikel.
type Service struct{ repo *Repository }

// NewService membuat service artikel.
func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func mapErr(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

func setPublished(p *database.Post, v bool) {
	p.IsPublished = v
	if !v {
		p.PublishedAt = nil
	} else if p.PublishedAt == nil {
		now := time.Now()
		p.PublishedAt = &now
	}
}

// Create membuat artikel baru dengan slug unik.
func (s *Service) Create(userID uint, req CreatePostRequest) (*database.Post, error) {
	title, content := strings.TrimSpace(req.Title), strings.TrimSpace(req.Content)
	if title == "" || content == "" {
		return nil, ErrInvalid
	}
	p := &database.Post{
		UserID: userID, Title: title, Slug: uniqueSlug(title),
		Category: normalizeCategory(req.Category), Excerpt: strings.TrimSpace(req.Excerpt),
		Content: req.Content, CoverImagePath: req.CoverImagePath,
	}
	setPublished(p, req.IsPublished)
	if err := s.repo.Create(p); err != nil {
		return nil, err
	}
	return p, nil
}

// owned memuat artikel dan memastikan kepemilikan.
func (s *Service) owned(userID, id uint) (*database.Post, error) {
	p, err := s.repo.FindByID(id)
	if err != nil {
		return nil, mapErr(err)
	}
	if p.UserID != userID {
		return nil, ErrForbidden
	}
	return p, nil
}

// Get mengambil artikel milik penulis.
func (s *Service) Get(userID, id uint) (*database.Post, error) { return s.owned(userID, id) }

// Update memperbarui artikel; field kosong tidak menimpa nilai lama.
func (s *Service) Update(userID, id uint, req UpdatePostRequest) (*database.Post, error) {
	p, err := s.owned(userID, id)
	if err != nil {
		return nil, err
	}
	if v := strings.TrimSpace(req.Title); v != "" {
		p.Title = v
	}
	if v := strings.TrimSpace(req.Category); v != "" {
		p.Category = v
	}
	if v := strings.TrimSpace(req.Excerpt); v != "" {
		p.Excerpt = v
	}
	if strings.TrimSpace(req.Content) != "" {
		p.Content = req.Content
	}
	if req.CoverImagePath != "" {
		p.CoverImagePath = req.CoverImagePath
	}
	if req.IsPublished != nil {
		setPublished(p, *req.IsPublished)
	}
	if err := s.repo.Save(p); err != nil {
		return nil, err
	}
	return p, nil
}

// TogglePublish membalik status Draf <-> Terbit.
func (s *Service) TogglePublish(userID, id uint) (*database.Post, error) {
	p, err := s.owned(userID, id)
	if err != nil {
		return nil, err
	}
	setPublished(p, !p.IsPublished)
	if err := s.repo.Save(p); err != nil {
		return nil, err
	}
	return p, nil
}

// Delete menghapus artikel milik penulis.
func (s *Service) Delete(userID, id uint) error {
	p, err := s.owned(userID, id)
	if err != nil {
		return err
	}
	return s.repo.Delete(p)
}

// List mengambil artikel milik penulis.
func (s *Service) List(userID uint) ([]database.Post, error) { return s.repo.ListByUser(userID) }

// Stats menghitung 3 kartu statistik penulis.
func (s *Service) Stats(userID uint) (*Stats, error) {
	total, pub, err := s.repo.Count(userID)
	if err != nil {
		return nil, err
	}
	return &Stats{TotalArticles: total, TotalSelesai: pub, BelumSelesai: total - pub}, nil
}

// PublicList mengambil artikel terbit (filter q & category).
func (s *Service) PublicList(q, category string) ([]database.Post, error) {
	return s.repo.ListPublished(q, category)
}

// PublicDetail mengambil satu artikel terbit berdasarkan slug.
func (s *Service) PublicDetail(slug string) (*database.Post, error) {
	p, err := s.repo.FindPublishedBySlug(slug)
	return p, mapErr(err)
}

// AdminPosts mengambil seluruh artikel.
func (s *Service) AdminPosts() ([]database.Post, error) { return s.repo.ListAll() }

// AdminUsers mengambil seluruh user beserta statistik artikel.
func (s *Service) AdminUsers() ([]UserSummary, error) { return s.repo.UsersWithCounts() }
