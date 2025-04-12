package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/diillson/api-gateway-go/internal/adapter/database"
	"github.com/diillson/api-gateway-go/pkg/logging"
	"go.uber.org/zap"
)

func main() {
	// Configurações
	var (
		action       string
		name         string
		driver       string
		dsn          string
		migrationDir string
	)

	flag.StringVar(&action, "action", "migrate", "Ação (migrate, create)")
	flag.StringVar(&name, "name", "", "Nome da migração (apenas para action=create)")
	flag.StringVar(&driver, "driver", "sqlite", "Driver de banco de dados (sqlite, mysql, postgres)")
	flag.StringVar(&dsn, "dsn", "./routes.db", "DSN do banco de dados")
	flag.StringVar(&migrationDir, "dir", "./migrations", "Diretório de migrações")
	flag.Parse()

	// Inicializar logger
	logger, err := logging.NewLogger()
	if err != nil {
		fmt.Printf("Erro ao inicializar logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Configurar banco de dados
	dbConfig := database.Config{
		Driver:          driver,
		DSN:             dsn,
		MaxIdleConns:    5,
		MaxOpenConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
		LogLevel:        database.LogLevelInfo,
		SlowThreshold:   200 * time.Millisecond,
		MigrationDir:    migrationDir,
	}

	ctx := context.Background()

	switch action {
	case "migrate":
		// Executar migrações
		db, err := database.NewDatabase(ctx, dbConfig, logger)
		if err != nil {
			logger.Fatal("Falha ao inicializar banco de dados", zap.Error(err))
		}
		defer db.Close()

		logger.Info("Migrações aplicadas com sucesso")

	case "create":
		if name == "" {
			logger.Fatal("Nome da migração é obrigatório para action=create")
		}

		// Criar nova migração
		db, err := database.NewDatabase(ctx, dbConfig, logger)
		if err != nil {
			logger.Fatal("Falha ao inicializar banco de dados", zap.Error(err))
		}
		defer db.Close()

		filepath, err := db.CreateMigration(name)
		if err != nil {
			logger.Fatal("Falha ao criar migração", zap.Error(err))
		}

		logger.Info("Migração criada", zap.String("path", filepath))

	default:
		logger.Fatal("Ação desconhecida", zap.String("action", action))
	}
}
