package main

import (
	"github.com/diillson/api-gateway-go/initialization"
	"github.com/diillson/api-gateway-go/internal/auth"
	"github.com/diillson/api-gateway-go/internal/config"
	"github.com/diillson/api-gateway-go/internal/database"
	"github.com/diillson/api-gateway-go/internal/handler"
	"github.com/diillson/api-gateway-go/internal/logging"
	"github.com/diillson/api-gateway-go/internal/middleware"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
) // This should be the same secret key used in the IsAuthenticated middleware

func main() {
	logger, err := logging.NewLogger()
	if err != nil {
		// handle error
		logger.Error("Error initializing logger: %v\n", zap.Error(err))
		return
	}

	db, err := database.NewDatabase()
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}

	r := gin.Default()
	r.Use(auth.IsAuthenticated())

	err = initialization.LoadAndSaveRoutes(r, "./routes/routes.json", db, logger)
	if err != nil {
		logger.Error("Failed to load routes", zap.Error(err))
	}

	token, err := auth.GenerateJWT("admin")
	if err != nil {
		logger.Error("Error generating the token:", zap.Error(err))
		return
	}

	logger.Info("Generated JWT token:", zap.String("token", token))

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

	for _, route := range routes {
		if !handler.RouteExists(r, route.Methods, route.Path) {
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
		} else {
			logger.Warn("Route already exists", zap.String("path", route.Path))
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
