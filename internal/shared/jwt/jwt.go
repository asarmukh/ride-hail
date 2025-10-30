package jwt

import (
	"time"

	"ride-hail/internal/shared/models"

	"github.com/golang-jwt/jwt/v5"
)

var JwtKey = []byte("awdwkamdawdnhbkdl")

func GenerateJWT(role string) (string, error) {
	expirationTime := time.Now().Add(15 * time.Hour)
	claims := &models.Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(JwtKey)
}
