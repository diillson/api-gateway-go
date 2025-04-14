package database

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/diillson/api-gateway-go/internal/domain/model"
	"go.uber.org/zap"
)

// JSONRouteLoader carrega rotas de um arquivo JSON
type JSONRouteLoader struct {
	db     *Database
	logger *zap.Logger
}

// NewJSONRouteLoader cria um novo carregador de rotas JSON
func NewJSONRouteLoader(db *Database, logger *zap.Logger) *JSONRouteLoader {
	return &JSONRouteLoader{
		db:     db,
		logger: logger,
	}
}

// LoadRoutesFromJSON carrega rotas de um arquivo JSON para o banco de dados
func (l *JSONRouteLoader) LoadRoutesFromJSON(filePath string) error {
	// Verificar se o arquivo existe
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		l.logger.Warn("Arquivo de rotas não encontrado", zap.String("path", filePath))
		return nil // Não é erro, apenas não há arquivo
	}

	// Ler o arquivo
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		l.logger.Error("Erro ao ler arquivo de rotas", zap.String("path", filePath), zap.Error(err))
		return err
	}

	// Deserializar o JSON
	var routes []*model.Route
	if err := json.Unmarshal(data, &routes); err != nil {
		l.logger.Error("Erro ao deserializar arquivo de rotas", zap.String("path", filePath), zap.Error(err))
		return err
	}

	// Se não há rotas, não fazer nada
	if len(routes) == 0 {
		l.logger.Info("Nenhuma rota encontrada no arquivo", zap.String("path", filePath))
		return nil
	}

	// Criar o repositório de rotas
	repo := NewRouteRepository(l.db.DB(), l.logger)

	// Criar um contexto válido com timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Inserir ou atualizar cada rota
	for _, route := range routes {
		// Verificar se a rota já existe
		_, err := repo.GetRouteByPath(ctx, route.Path)
		if err == nil {
			// Rota existe, atualizar
			l.logger.Debug("Atualizando rota existente", zap.String("path", route.Path))
			if err := repo.UpdateRoute(ctx, route); err != nil {
				l.logger.Error("Erro ao atualizar rota", zap.String("path", route.Path), zap.Error(err))
			}
		} else {
			// Rota não existe, inserir
			l.logger.Debug("Inserindo nova rota", zap.String("path", route.Path))
			if err := repo.AddRoute(ctx, route); err != nil {
				l.logger.Error("Erro ao inserir rota", zap.String("path", route.Path), zap.Error(err))
			}
		}
	}

	l.logger.Info("Rotas carregadas com sucesso", zap.Int("count", len(routes)), zap.String("file", filepath.Base(filePath)))
	return nil
}
