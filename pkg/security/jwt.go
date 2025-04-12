package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type KeyManager struct {
	secretKey []byte
	logger    *zap.Logger
}

func NewKeyManager(logger *zap.Logger) (*KeyManager, error) {
	// Buscando o secret do config - mesmo valor usado no seu generate_token.go
	secretKey := GetJWTSecret()

	if len(secretKey) < 32 {
		return nil, errors.New("jwt secret key muito curta")
	}

	return &KeyManager{
		secretKey: secretKey,
		logger:    logger,
	}, nil
}

func (km *KeyManager) GenerateToken(userID, role string, duration time.Duration) (string, error) {
	expireTime := time.Now().Add(duration)

	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(km.secretKey)
	if err != nil {
		km.logger.Error("falha ao gerar token JWT", zap.Error(err))
		return "", err
	}

	return tokenString, nil
}

func (km *KeyManager) VerifyToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verificar o método de assinatura
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de assinatura inesperado: %v", token.Header["alg"])
		}
		return km.secretKey, nil
	})

	if err != nil {
		if err.Error() == "token has expired" {
			return nil, errors.New("token expirado")
		}
		km.logger.Error("falha ao validar token JWT", zap.Error(err))
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("token inválido")
}
