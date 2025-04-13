package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/diillson/api-gateway-go/internal/adapter/database"
	"github.com/diillson/api-gateway-go/internal/domain/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func main() {
	// Parsing de flags
	var (
		username string
		password string
		email    string
		dbDriver string
		dbDSN    string
		verbose  bool
	)

	flag.StringVar(&username, "username", "", "Nome de usuário do admin")
	flag.StringVar(&password, "password", "", "Senha do admin")
	flag.StringVar(&email, "email", "", "Email do admin")
	flag.StringVar(&dbDriver, "driver", "postgres", "Driver do banco de dados (sqlite, mysql, postgres)")
	flag.StringVar(&dbDSN, "dsn", "postgres://postgres:postgres@localhost:5432/apigateway?sslmode=disable", "DSN do banco de dados")
	flag.BoolVar(&verbose, "verbose", false, "Mostrar logs detalhados")
	flag.Parse()

	// Validar entradas
	if username == "" || password == "" || email == "" {
		fmt.Println("Erro: username, password e email não podem ser vazios.")
		flag.Usage()
		os.Exit(1)
	}

	// Configurar logger com nível apropriado
	cfg := zap.NewProductionConfig()
	if !verbose {
		cfg.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel) // Só mostra erros fatais
		cfg.OutputPaths = []string{"stderr"}                 // Só envia para stderr
	}

	logger, err := cfg.Build()
	if err != nil {
		fmt.Printf("Erro ao inicializar logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Configuração do banco de dados
	dbConfig := database.Config{
		Driver:          dbDriver,
		DSN:             dbDSN,
		MaxIdleConns:    5,
		MaxOpenConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
		LogLevel:        4,
		SlowThreshold:   200 * time.Millisecond,
		SkipMigrations:  true, // Pular migrações para esta ferramenta
	}

	// Inicializar banco de dados
	ctx := context.Background()
	db, err := database.NewDatabase(ctx, dbConfig, logger)
	if err != nil {
		fmt.Printf("Erro ao conectar ao banco de dados: %v\n", err)
		os.Exit(1)
	}

	// Verificar se a tabela de usuários existe, se não, criá-la
	if !db.DB().Migrator().HasTable(&model.UserEntity{}) {
		if err := db.DB().AutoMigrate(&model.UserEntity{}); err != nil {
			fmt.Printf("Erro ao criar tabela de usuários: %v\n", err)
			os.Exit(1)
		}
	}

	// Verificar se o usuário já existe
	var existingUser model.UserEntity
	result := db.DB().Where("username = ?", username).First(&existingUser)

	// Flag para controlar se é uma nova criação ou atualização
	isUpdate := false

	// Se o usuário existir, verificar se queremos substituir
	if result.Error == nil {
		isUpdate = true
		fmt.Printf("Usuário '%s' já existe. Deseja sobrescrevê-lo? (s/n): ", username)
		var response string
		fmt.Scanln(&response)

		if response != "s" && response != "S" {
			fmt.Println("Operação cancelada pelo usuário.")
			os.Exit(0)
		}

		db.DB().Delete(&existingUser)
	} else if result.Error != gorm.ErrRecordNotFound {
		fmt.Printf("Erro ao verificar usuário existente: %v\n", result.Error)
		os.Exit(1)
	}

	// Hash da senha
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("Erro ao processar senha: %v\n", err)
		os.Exit(1)
	}

	// Criar usuário admin
	adminUser := model.UserEntity{
		ID:        uuid.New().String(),
		Username:  username,
		Password:  string(hashedPassword),
		Email:     email,
		Role:      "admin",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Salvar no banco de dados
	if err := db.DB().Create(&adminUser).Error; err != nil {
		fmt.Printf("Erro ao salvar usuário no banco de dados: %v\n", err)
		os.Exit(1)
	}

	// Mostrar apenas informações relevantes e não sensíveis
	fmt.Println("\n╭──────────────────────────────────────────╮")
	if isUpdate {
		fmt.Println("│  Usuário admin atualizado com sucesso    │")
	} else {
		fmt.Println("│      Usuário admin criado com sucesso      │")
	}
	fmt.Println("├──────────────────────────────────────────┤")
	fmt.Printf("│ ID: %-35s │\n", adminUser.ID)
	fmt.Printf("│ Username: %-30s │\n", username)
	fmt.Printf("│ Email: %-33s │\n", email)
	fmt.Printf("│ Role: %-34s │\n", "admin")
	fmt.Println("╰──────────────────────────────────────────╯")
	fmt.Println("\nUse este ID para gerar um token de acesso com:")
	fmt.Printf("go run cmd/tools/generate_token.go -user_id=%s\n\n", adminUser.ID)
}
