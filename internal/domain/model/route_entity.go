package model

import (
	"time"
)

// RouteEntity é a representação de banco de dados de uma rota
type RouteEntity struct {
	ID                  uint      `gorm:"primaryKey"`
	Path                string    `gorm:"uniqueIndex;not null"`
	ServiceURL          string    `gorm:"not null"`
	MethodsJSON         string    `gorm:"column:methods;type:json;not null"`
	HeadersJSON         string    `gorm:"column:headers;type:json"`
	Description         string    `gorm:"type:text"`
	IsActive            bool      `gorm:"default:true"`
	CallCount           int64     `gorm:"default:0"`
	TotalResponse       int64     `gorm:"default:0"` // Armazenado em nanossegundos
	RequiredHeadersJSON string    `gorm:"column:required_headers;type:json"`
	CreatedAt           time.Time `gorm:"autoCreateTime"`
	UpdatedAt           time.Time `gorm:"autoUpdateTime"`
	LastUpdatedAt       time.Time
}

// TableName define o nome da tabela
func (RouteEntity) TableName() string {
	return "routes"
}
