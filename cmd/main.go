package main

import (
	"encoding/json"
	"github.com/diillson/api-gateway-go/internal/database"
	"github.com/diillson/api-gateway-go/internal/handler"
	"github.com/diillson/api-gateway-go/internal/logging"
	"github.com/diillson/api-gateway-go/internal/middleware"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"time"
)

type Route struct {
	Path          string        `json:"path"`
	ServiceURL    string        `json:"serviceURL"`
	Methods       []string      `json:"methods"`
	Headers       []string      `json:"headers"`
	Description   string        `json:"description"`
	IsActive      bool          `json:"isActive"`
	CallCount     int64         `json:"callCount"`
	TotalResponse time.Duration `json:"totalResponse"`
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
	db := database.NewDatabase()
	routes, err := db.GetRoutes()
	if err != nil {
		log.Fatalf("Failed to load routes: %v", err)
	}

	httpHandler := handler.NewHandler(db, logger)
	for _, route := range routes {
		httpHandler.AddRoute(route.Path, route.ServiceURL, route.Method)
	}

	middleware := middleware.NewMiddleware(logger)

	chain := alice.New(
		middleware.Authenticate,
		middleware.RateLimit,
		middleware.ValidateHeaders,
		middleware.Analytics,
		middleware.RecoverPanic,
		middleware.Authenticate,
	).Then(httpHandler)

	http.Handle("/", chain)
	http.Handle("/admin/register", middleware.AuthenticateAdmin(h.RegisterAPI))
	// Rota para listar todas as APIs registradas
	http.Handle("/admin/apis", middleware.AuthenticateAdmin(h.ListAPIs))
	// Rota para atualizar uma API existente
	http.Handle("/admin/update", middleware.AuthenticateAdmin(h.UpdateAPI))
	// Rota para deletar uma API existente
	http.Handle("/admin/delete", middleware.AuthenticateAdmin(h.DeleteAPI))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
