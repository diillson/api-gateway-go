package auth

import (
	"github.com/dgrijalva/jwt-go"
	"net/http"
	"strings"
)

// estrutura para decodificar o JWT
type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

var jwtKey = []byte("your-secret-key") // Substitua pela sua chave secreta

func IsAuthenticated(r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil || !token.Valid {
		return false
	}
	return true
}
