package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func setJWTCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    token,
		Path:     "/",
		HttpOnly: true,  // prevents JS access
		Secure:   false, // set true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600, // 1 hour
	})
}
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func extractToken(r *http.Request) string {
	cookie, err := r.Cookie("jwt") // Return jwt token directly if it exists in cookie
	if err == nil {
		return cookie.Value
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return parts[1] //Omits Authorization Token's "Bearer "
}

func GenerateJWT(studentNo string) (string, error) {
	JwtSecret := os.Getenv("JWT_SECRET")

	claims := &jwt.RegisteredClaims{
		Subject:   studentNo,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(JwtSecret))
}
