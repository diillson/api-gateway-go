package auth

import (
	"errors"
	"time"

	"github.com/diillson/api-gateway-go/internal/domain/model"
	"github.com/diillson/api-gateway-go/pkg/security"
	"go.uber.org/zap"
)

// UserRepository define a interface para acesso a dados de usuário
type UserRepository interface {
	GetUserByCredentials(username, password string) (*model.User, error)
	GetUserByID(id string) (*model.User, error)
}

// AuthService gerencia operações de autenticação
type AuthService struct {
	keyManager *security.KeyManager
	userRepo   UserRepository
	logger     *zap.Logger
}

// NewAuthService cria um novo serviço de autenticação
func NewAuthService(keyManager *security.KeyManager, userRepo UserRepository, logger *zap.Logger) *AuthService {
	return &AuthService{
		keyManager: keyManager,
		userRepo:   userRepo,
		logger:     logger,
	}
}

// Login autentica um usuário e gera um token JWT
func (s *AuthService) Login(username, password string) (string, error) {
	user, err := s.userRepo.GetUserByCredentials(username, password)
	if err != nil {
		s.logger.Warn("Falha na autenticação", zap.String("username", username), zap.Error(err))
		return "", errors.New("credenciais inválidas")
	}

	// Gerar token com duração de 24 horas
	token, err := s.keyManager.GenerateToken(user.ID, user.Role, 24*time.Hour)
	if err != nil {
		s.logger.Error("Falha ao gerar token", zap.String("user_id", user.ID), zap.Error(err))
		return "", err
	}

	s.logger.Info("Login bem-sucedido", zap.String("user_id", user.ID))
	return token, nil
}

// ValidateToken valida um token JWT e retorna o usuário correspondente
func (s *AuthService) ValidateToken(tokenString string) (*model.User, error) {
	claims, err := s.keyManager.VerifyToken(tokenString)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetUserByID(claims.UserID)
	if err != nil {
		s.logger.Error("Usuário do token não encontrado", zap.String("user_id", claims.UserID), zap.Error(err))
		return nil, errors.New("usuário inválido")
	}

	return user, nil
}

// IsAdmin verifica se um usuário tem permissão administrativa
func (s *AuthService) IsAdmin(user *model.User) bool {
	return user != nil && user.Role == "admin"
}
