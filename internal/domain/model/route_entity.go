package model

import (
	"time"
)

// RouteEntity é a representação de banco de dados de uma rota
type RouteEntity struct {
	ID                  uint      `gorm:"primaryKey"`
	Path                string    `gorm:"uniqueIndex;not null"`
	ServiceURL          string    `gorm:"not null"`
	MethodsJSON         string    `gorm:"column:methods;type:text"`
	HeadersJSON         string    `gorm:"column:headers;type:text"`
	Description         string    `gorm:"type:text"`
	IsActive            bool      `gorm:"default:true"`
	CallCount           int64     `gorm:"default:0"`
	TotalResponse       int64     `gorm:"default:0"` // Armazenado em nanossegundos
	RequiredHeadersJSON string    `gorm:"column:required_headers;type:text"`
	CreatedAt           time.Time `gorm:"autoCreateTime"`
	UpdatedAt           time.Time `gorm:"autoUpdateTime"`
	LastUpdatedAt       time.Time
}

// TableName define o nome da tabela
func (RouteEntity) TableName() string {
	return "routes"
}
