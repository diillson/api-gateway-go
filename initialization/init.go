package initialization

import (
	"encoding/json"
	"github.com/diillson/api-gateway-go/internal/config"
	"github.com/diillson/api-gateway-go/internal/database"
	"github.com/diillson/api-gateway-go/internal/handler"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"os"
)

func LoadRoutes(filePath string) ([]config.Route, error) {
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

func LoadAndSaveRoutes(r *gin.Engine, filePath string, db *database.Database, logger *zap.Logger) error {
	routes, err := LoadRoutes(filePath)
	if err != nil {
		return err
	}

	for _, route := range routes {
		// Verificar e adicionar a rota ao banco de dados
		if !handler.RouteExists(r, route.Methods, route.Path) {
			err = db.AddRoute(&route)
			if err != nil {
				logger.Error("Failed to add route to database", zap.Error(err))
				return err // Retornar o erro e interromper o processo se não puder adicionar a rota
			}
		} else {
			// Se a rota já existir, apenas logue um aviso e continue
			logger.Warn("Route already exists", zap.String("path", route.Path))
			// Não retornar erro, apenas continuar para a próxima rota
		}
	}

	return nil // Retornar nil ao final indicando que não houve erro crítico
}
