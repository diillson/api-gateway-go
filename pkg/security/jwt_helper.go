package security

import (
	"github.com/diillson/api-gateway-go/pkg/config"
	"os"
)

// GetJWTSecret obtém o segredo JWT de diferentes fontes na seguinte ordem:
// 1. Variável de ambiente JWT_SECRET
// 2. Arquivo de configuração
// 3. Fallback para valor padrão (apenas em desenvolvimento)
func GetJWTSecret() []byte {
	// Primeiro, tentar obter da variável de ambiente
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret != "" {
		return []byte(secret)
	}

	// Segundo, tentar obter da variável específica AG_AUTH_JWTSECRET
	secret = os.Getenv("AG_AUTH_JWT_SECRET_KEY")
	if secret != "" {
		return []byte(secret)
	}

	// Terceiro, obter da configuração
	cfg, err := config.LoadConfig("./config")
	if err == nil && cfg.Auth.JWTSecret != "" {
		return []byte(cfg.Auth.JWTSecret)
	}

	// Fallback para o valor padrão (apenas para desenvolvimento)
	// Em ambientes de produção, isso não deve ser usado
	//return []byte("seu token padrão aqui")
	return nil
}
