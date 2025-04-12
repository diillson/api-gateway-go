package model

import "time"

// User representa um usuário do sistema
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Email    string `json:"email,omitempty"`
}

// UserEntity é a representação de banco de dados de um usuário
type UserEntity struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Username  string    `gorm:"uniqueIndex;not null;size:50"`
	Password  string    `gorm:"not null"`
	Email     string    `gorm:"uniqueIndex;size:100"`
	Role      string    `gorm:"default:user;size:20"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName define o nome da tabela
func (UserEntity) TableName() string {
	return "users"
}
