package post

import (
	"testing"
	"time"

	"app/internal/middleware"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func signToken(t *testing.T, id uint, role string) string {
	claims := middleware.Claims{
		UserID: id, Username: "u", Role: role,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	require.NoError(t, err)
	return s
}
