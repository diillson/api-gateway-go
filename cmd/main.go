package main

import (
	"encoding/json"
	"github.com/diillson/api-gateway-go/internal/handler"
	"github.com/diillson/api-gateway-go/internal/logging"
	"github.com/diillson/api-gateway-go/internal/middleware"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
)

type Route struct {
	Path       string   `json:"path"`
	ServiceURL string   `json:"serviceURL"`
	Methods    []string `json:"methods"`
	Headers    []string `json:"headers"`
}

func loadRoutes(filePath string) ([]Route, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	routes := []Route{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&routes)
	return routes, err
}

func main() {
	logger := logging.NewLogger()

	routes, err := loadRoutes("./routes/routes.json")
	if err != nil {
		logger.Fatal("Failed to load routes", zap.Error(err))
	}

	httpHandler := handler.NewHandler(logger)
	for _, route := range routes {
		httpHandler.AddRoute(route.Path, route.ServiceURL, route.Method)
	}

	middleware := middleware.NewMiddleware(logger)

	chain := alice.New(
		middleware.Authenticate,
		middleware.RateLimit,
		middleware.ValidateHeaders,
		middleware.Analytics,
	).Then(httpHandler)

	http.Handle("/", chain)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
