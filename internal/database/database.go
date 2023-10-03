package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/diillson/api-gateway-go/internal/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Database struct {
	DB *gorm.DB
}

func (db *Database) UpdateMetrics(route *config.Route) error {
	return db.DB.Model(route).Where("path = ?", route.Path).Updates(map[string]interface{}{
		"call_count":     route.CallCount,
		"total_response": route.TotalResponse,
	}).Error
}

func NewDatabase() (*Database, error) {
	db, err := gorm.Open(sqlite.Open("./routes.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	database := &Database{DB: db}

	if err := database.initialize(); err != nil {
		return nil, err
	}

	return database, nil
}

func (db *Database) initialize() error {
	err := db.DB.AutoMigrate(&config.Route{})
	return err
}

func (db *Database) GetRoutes() ([]*config.Route, error) {
	if db == nil || db.DB == nil {
		return nil, errors.New("database not initialized")
	}

	var routeEntities []struct {
		config.Route
		MethodsJSON         string `gorm:"column:methods"`
		HeadersJSON         string `gorm:"column:headers"`
		RequiredHeadersJSON string `gorm:"column:required_headers"`
	}

	// Query usando métodos GORM
	result := db.DB.Table("routes").Scan(&routeEntities)
	if result.Error != nil {
		return nil, result.Error
	}

	var routes []*config.Route
	for _, entity := range routeEntities {
		if err := json.Unmarshal([]byte(entity.MethodsJSON), &entity.Methods); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(entity.HeadersJSON), &entity.Headers); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(entity.RequiredHeadersJSON), &entity.RequiredHeaders); err != nil {
			return nil, err
		}
		route := entity.Route
		routes = append(routes, &route)
	}

	return routes, nil
}

func (db *Database) AddRoute(route *config.Route) error {
	// Convertendo os slices para JSON
	methods, err := json.Marshal(route.Methods)
	if err != nil {
		return errors.New("failed to marshal methods: " + err.Error())
	}

	headers, err := json.Marshal(route.Headers)
	if err != nil {
		return errors.New("failed to marshal headers: " + err.Error())
	}

	requiredHeaders, err := json.Marshal(route.RequiredHeaders)
	if err != nil {
		return errors.New("failed to marshal required headers: " + err.Error())
	}

	// Criando um mapa para armazenar os valores que serão salvos no DB
	data := map[string]interface{}{
		"path":             route.Path,
		"service_url":      route.ServiceURL,
		"methods":          string(methods),
		"headers":          string(headers),
		"description":      route.Description,
		"is_active":        route.IsActive,
		"call_count":       route.CallCount,
		"total_response":   route.TotalResponse,
		"required_headers": string(requiredHeaders),
	}

	// Armazenando os dados no banco de dados
	if err := db.DB.Model(&config.Route{}).Create(&data).Error; err != nil {
		return errors.New("failed to add route: " + err.Error())
	}

	return nil
}

func (db *Database) UpdateRoute(route *config.Route) error {
	if db == nil || db.DB == nil {
		return errors.New("database not initialized")
	}

	// Não é necessário passar um ponteiro aqui, GORM pode lidar com o valor diretamente
	if err := db.DB.Save(route).Error; err != nil {
		return fmt.Errorf("failed to update route: %w", err)
	}
	return nil
}

func (db *Database) DeleteRoute(path string) error {
	if db == nil || db.DB == nil {
		return errors.New("database not initialized")
	}

	// Certifique-se de que o path não está vazio
	if path == "" {
		return errors.New("path cannot be empty")
	}

	// Não é necessário criar uma instância de config.Route se você está apenas excluindo por path
	if err := db.DB.Where("path = ?", path).Delete(&config.Route{}).Error; err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}
	return nil
}
