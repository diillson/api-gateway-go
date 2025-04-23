package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/diillson/api-gateway-go/internal/adapter/database"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Parsing de flags
	var (
		username string
		dbDriver string
		dbDSN    string
		verbose  bool
	)

	flag.StringVar(&username, "username", "", "Nome de usuário a ser diagnosticado")
	flag.StringVar(&dbDriver, "driver", "postgres", "Driver do banco de dados (sqlite, mysql, postgres)")
	flag.StringVar(&dbDSN, "dsn", "postgres://postgres:postgres@localhost:5432/apigateway?sslmode=disable", "DSN do banco de dados")
	flag.BoolVar(&verbose, "verbose", false, "Mostrar logs detalhados")
	flag.Parse()

	// Validar entradas
	if username == "" {
		fmt.Println("Erro: username não pode ser vazio.")
		flag.Usage()
		os.Exit(1)
	}

	// Configurar logger
	cfg := zap.NewProductionConfig()
	if !verbose {
		cfg.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
		cfg.OutputPaths = []string{"stderr"}
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

	// Criar repo de usuário
	userRepo := database.NewUserRepository(db.DB())

	// Executar diagnóstico
	report, err := userRepo.DiagnoseUserStorage(username)
	if err != nil {
		fmt.Printf("Erro ao diagnosticar usuário: %v\n", err)
		os.Exit(1)
	}

	// Exibir relatório
	fmt.Println("\n╭─────────────────────────────────────────╮")
	fmt.Println("│       DIAGNÓSTICO DE ARMAZENAMENTO       │")
	fmt.Println("├─────────────────────────────────────────┤")
	fmt.Println(report)
	fmt.Println("╰─────────────────────────────────────────╯")
}
