// Package auth implements password hashing (bcrypt), JWT issuing /
// validation, and a simple in-memory login rate limiter.
// Passwords are never stored in plaintext. JWTs are HS256, signed with
// JWT_SECRET from the environment.
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plaintext password with bcrypt (cost 12).
// The returned string is the only thing ever stored.
func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword compares a plaintext password against a stored bcrypt hash.
func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// Claims is the JWT payload.
type Claims struct {
	UserID   string `json:"sub"`
	Email    string `json:"email"`
	Username string `json:"username"`
	DeviceID string `json:"device_id,omitempty"`
	jwt.RegisteredClaims
}

// GenerateJWT issues a signed token for a user.
func GenerateJWT(secret, userID, email, username, deviceID string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID, Email: email, Username: username, DeviceID: deviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateJWT parses and verifies a token, returning its claims.
func ValidateJWT(secret, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
