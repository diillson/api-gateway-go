package main

import (
	"flag"
	"fmt"
	"github.com/diillson/api-gateway-go/pkg/security"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	// Define a flag para o userID
	var userID string
	flag.StringVar(&userID, "user_id", "", "ID do usuário admin")
	flag.Parse()

	// Verifica se o userID foi fornecido
	if userID == "" {
		fmt.Println("Erro: O ID do usuário admin não pode ser vazio.")
		fmt.Println("Uso: go run cmd/tools/generate_token.go -userid=<ID do usuário>")
		os.Exit(1)
	}

	// Obter a chave secreta do seu arquivo config.yaml
	secretKey := security.GetJWTSecret()

	// Imprimir um aviso se o valor padrão estiver sendo usado
	if len(secretKey) == 0 {
		fmt.Println("AVISO: Nenhum segredo JWT configurado. Utilizando valor padrão inseguro!")
		fmt.Println("Para segurança adequada, configure JWT_SECRET_KEY ou AG_AUTH_JWT_SECRET_KEY ou defina auth.jwtsecret no config.yaml")
	}

	// Criar os claims do JWT exatamente no formato que o sistema espera
	claims := jwt.MapClaims{
		"user_id": userID, // ID do usuário admin que foi criado
		"role":    "admin",
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
		"nbf":     time.Now().Unix(),
	}

	// Criar o token com o algoritmo de assinatura correto
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Assinar o token com a chave secreta
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		fmt.Printf("Erro ao gerar token: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nToken JWT gerado:")
	fmt.Println("------------------------------------------")
	fmt.Println(tokenString)
	fmt.Println("------------------------------------------")
	fmt.Printf("\nDetalhes do token:\n")
	fmt.Printf("ID do usuário: %s\n", userID)
	fmt.Printf("Papel: admin\n")
	fmt.Printf("Expira em: %s\n", time.Now().Add(24*time.Hour).Format(time.RFC3339))
	fmt.Println("\nUse este token no cabeçalho Authorization:")
	fmt.Printf("Authorization: Bearer %s\n", tokenString)

	// Dica adicional sobre configuração
	fmt.Println("\nPara configurar sua própria chave secreta:")
	fmt.Println("1. Como variável de ambiente: export JWT_SECRET_KEY=sua-chave-secreta")
	fmt.Println("2. No arquivo config.yaml: jwtsecret: sua-chave-secreta")
	fmt.Println("3. Via variável AG: export AG_AUTH_JWT_SECRET_KEY=sua-chave-secreta")
}
