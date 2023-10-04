package auth

import (
	"github.com/golang-jwt/jwt/v4"
	"time"
)

// GenerateJWT creates a new JWT for a given username
func GenerateJWT(username string) (string, error) {
	// Setting the token expiration time
	expirationTime := time.Now().Add(24 * time.Hour)

	// Creating the claims for the token, including the username and expiration time
	claims := &Claims{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			// Including the expiration time in Unix time
			ExpiresAt: expirationTime.Unix(),
		},
	}

	// Creating a new JWT token with the claims and signing it with the secret key
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Converting the token into a string
	return token.SignedString(JwtKey)
}
