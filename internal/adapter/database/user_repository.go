package database

import (
	"errors"
	"fmt"
	"github.com/diillson/api-gateway-go/internal/domain/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetUserByCredentials(username, password string) (*model.User, error) {
	var user model.UserEntity

	// Adicionar logs para diagnóstico
	result := r.db.Where("username = ?", username).First(&user)
	if result.Error != nil {
		// Verificar tipo específico de erro
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("usuário não encontrado")
		}

		// Registrar detalhes do erro
		fmt.Printf("Erro ao buscar usuário: %v, Database: %s\n",
			result.Error, r.db.Dialector.Name())
		return nil, result.Error
	}

	// Verificar e registrar hash para diagnóstico
	fmt.Printf("Verificando senha para usuário: %s, Hash armazenado (len=%d): %s\n",
		username, len(user.Password), user.Password)

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		fmt.Printf("Falha na verificação de senha: %v\n", err)
		return nil, errors.New("senha inválida")
	}

	return &model.User{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
		Email:    user.Email,
	}, nil
}

func (r *UserRepository) GetUserByID(id string) (*model.User, error) {
	var user model.UserEntity
	if err := r.db.Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("usuário não encontrado")
		}
		return nil, err
	}

	return &model.User{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
		Email:    user.Email,
	}, nil
}

// DiagnoseUserStorage ajuda a diagnosticar problemas com armazenamento de usuários
func (r *UserRepository) DiagnoseUserStorage(username string) (string, error) {
	var user model.UserEntity
	result := r.db.Where("username = ?", username).First(&user)
	if result.Error != nil {
		return "", fmt.Errorf("erro ao buscar usuário: %w", result.Error)
	}

	// Obter informações sobre o banco de dados
	dbType := r.db.Dialector.Name()

	// Construir relatório de diagnóstico
	report := fmt.Sprintf(
		"Diagnóstico para usuário: %s\n"+
			"----------------------------\n"+
			"ID: %s\n"+
			"Tipo de banco: %s\n"+
			"Tamanho do hash de senha: %d\n"+
			"Username: %s\n"+
			"Email: %s\n"+
			"Role: %s\n"+
			"CreatedAt: %v\n",
		username,
		user.ID,
		dbType,
		len(user.Password),
		user.Username,
		user.Email,
		user.Role,
		user.CreatedAt,
	)

	return report, nil
}
