package auth

import (
	"net/http"
	"strings"
)

func IsAuthenticated(r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	// Implementar lógica real de validação de token aqui
	// Por exemplo, verificar o token JWT, consultar um serviço de autenticação, etc.
	return strings.HasPrefix(authHeader, "Bearer ")
}
