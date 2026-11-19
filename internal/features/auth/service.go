package auth

import (
	"errors"
	"strings"
	"time"

	"app/internal/database"
	"app/internal/middleware"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Galat domain autentikasi.
var (
	ErrDuplicate          = errors.New("username atau email sudah terdaftar")
	ErrInvalidCredentials = errors.New("username/email atau password salah")
	ErrNotFound           = errors.New("pengguna tidak ditemukan")
)

// Service berisi logika bisnis autentikasi.
type Service struct {
	repo   *Repository
	secret string
	ttl    time.Duration
}

// NewService membuat service autentikasi.
func NewService(repo *Repository, secret string) *Service {
	return &Service{repo: repo, secret: secret, ttl: 24 * time.Hour}
}

func (s *Service) issue(u *database.User) *AuthResponse {
	claims := middleware.Claims{
		UserID:   u.ID,
		Username: u.Username,
		Role:     u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	// Penandatanganan HS256 dengan kunci []byte tidak pernah gagal.
	t, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.secret))
	return &AuthResponse{Token: t, User: toUserResponse(u)}
}

// Register membuat akun baru. Role selain "admin"/"member" dipaksa "member".
func (s *Service) Register(req RegisterRequest) (*AuthResponse, error) {
	username := strings.TrimSpace(req.Username)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	exists, err := s.repo.Exists(username, email, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicate
	}
	h, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	role := "member"
	if req.Role == "admin" {
		role = "admin"
	}
	u := &database.User{Username: username, Email: email, Password: string(h), Role: role}
	if err := s.repo.Create(u); err != nil {
		return nil, err
	}
	return s.issue(u), nil
}

// Login memverifikasi kredensial (username atau email) dan menerbitkan JWT.
func (s *Service) Login(req LoginRequest) (*AuthResponse, error) {
	identity := strings.TrimSpace(req.Username)
	u, err := s.repo.FindByLogin(identity)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		u, err = s.repo.FindByLogin(strings.ToLower(identity))
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)) != nil {
		return nil, ErrInvalidCredentials
	}
	return s.issue(u), nil
}

// Profile mengambil profil akun.
func (s *Service) Profile(id uint) (*UserResponse, error) {
	u, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	r := toUserResponse(u)
	return &r, nil
}

// UpdateProfile mengubah email dan/atau password.
func (s *Service) UpdateProfile(id uint, req UpdateProfileRequest) (*UserResponse, error) {
	u, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if email := strings.ToLower(strings.TrimSpace(req.Email)); email != "" && email != u.Email {
		exists, err := s.repo.Exists("", email, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrDuplicate
		}
		u.Email = email
	}
	if req.Password != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		u.Password = string(h)
	}
	if err := s.repo.Save(u); err != nil {
		return nil, err
	}
	r := toUserResponse(u)
	return &r, nil
}
