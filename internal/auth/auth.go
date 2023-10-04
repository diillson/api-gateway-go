package auth

import (
	"fmt"
	"github.com/diillson/api-gateway-go/internal/logging"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

var JwtKey = []byte("your-secret-key")

func IsAuthenticated() gin.HandlerFunc {
	logger, err := logging.NewLogger()
	if err != nil {
		// handle error
		logger.Error("Error initializing logger: %v\n", zap.Error(err))
		return nil
	}

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header not provided"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return JwtKey, nil
		})

		if err != nil {
			logger.Error("Error parsing token: %v", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Error parsing token"})
			return
		}

		if !token.Valid {
			logger.Error("Invalid token", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		c.Next()
	}
}
