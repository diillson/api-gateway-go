package mocks

import (
	"github.com/diillson/api-gateway-go/internal/domain/model"
	"github.com/stretchr/testify/mock"
)

// MockAuthService é um mock para o serviço de autenticação
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Login(username, password string) (string, error) {
	args := m.Called(username, password)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) ValidateToken(tokenString string) (*model.User, error) {
	args := m.Called(tokenString)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockAuthService) IsAdmin(user *model.User) bool {
	args := m.Called(user)
	return args.Bool(0)
}

func (m *MockAuthService) GenerateToken(userID string, role string) (string, error) {
	args := m.Called(userID, role)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) RevokeToken(userID string) error {
	args := m.Called(userID)
	return args.Error(0)
}
