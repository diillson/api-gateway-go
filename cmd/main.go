package main

import (
	"encoding/json"
	"fmt"
	"github.com/diillson/api-gateway-go/internal/auth"
	"github.com/diillson/api-gateway-go/internal/config"
	"github.com/diillson/api-gateway-go/internal/database"
	"github.com/diillson/api-gateway-go/internal/handler"
	"github.com/diillson/api-gateway-go/internal/logging"
	"github.com/diillson/api-gateway-go/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
	"os"
	"time"
) // This should be the same secret key used in the IsAuthenticated middleware

// Function to generate a JWT token based on the username
func generateJWT(username string) (string, error) {
	// Setting the token expiration time
	expirationTime := time.Now().Add(24 * time.Hour)

	// Creating the claims for the token, including the username and expiration time
	claims := &auth.Claims{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			// Including the expiration time in Unix time
			ExpiresAt: expirationTime.Unix(),
		},
	}

	// Creating a new JWT token with the claims and signing it with the secret key
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Converting the token into a string
	tokenString, err := token.SignedString(auth.JwtKey)

	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func loadRoutes(filePath string) ([]config.Route, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	routes := []config.Route{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&routes)
	return routes, err
}

func main() {
	logger := logging.NewLogger()

	_, err := loadRoutes("./routes/routes.json")
	if err != nil {
		logger.Fatal("Failed to load routes", zap.Error(err))
	}

	token, err := generateJWT("admin")
	if err != nil {
		fmt.Println("Error generating the token:", err)
		return
	}

	fmt.Println("Generated JWT token:", token)

	db, err := database.NewDatabase()
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}
	routes, err := db.GetRoutes()
	if err != nil {
		logger.Fatal("Failed to load routes from database", zap.Error(err))
	}

	httpHandler := handler.NewHandler(db, logger)

	routesMap := make(map[string]*config.Route)
	for _, route := range routes {
		routesMap[route.Path] = route
	}
	// Passando a instância do banco de dados para o middleware
	mw := middleware.NewMiddleware(logger, routesMap, db)

	r := gin.Default()
	r.Use(auth.IsAuthenticated())

	for _, route := range routes {
		for _, method := range route.Methods {
			switch method {
			case "GET":
				r.GET(route.Path, mw.RateLimit, mw.Analytics, func(c *gin.Context) {
					httpHandler.ServeHTTP(c.Writer, c.Request)
				})
			case "POST":
				r.POST(route.Path, mw.RateLimit, mw.Analytics, func(c *gin.Context) {
					httpHandler.ServeHTTP(c.Writer, c.Request)
				})
				// Add other HTTP methods as needed
			}
		}
	}

	admin := r.Group("/admin")
	admin.Use(mw.AuthenticateAdmin) // ajustado para usar o middleware diretamente

	admin.POST("/register", httpHandler.RegisterAPI)
	admin.GET("/apis", httpHandler.ListAPIs)
	admin.PUT("/update", httpHandler.UpdateAPI)
	admin.DELETE("/delete", httpHandler.DeleteAPI)
	admin.GET("/metrics", httpHandler.GetMetrics)

	if err := r.Run(":8080"); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
