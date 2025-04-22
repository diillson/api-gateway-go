package database

import (
	"context"
	"fmt"
	"time"

	"github.com/diillson/api-gateway-go/internal/domain/model"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config contém configurações para o banco de dados
type Config struct {
	Driver          string
	DSN             string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	LogLevel        logger.LogLevel
	SlowThreshold   time.Duration
	MigrationDir    string
	SkipMigrations  bool
}

// Database gerencia a conexão com o banco de dados
type Database struct {
	db        *gorm.DB
	logger    *zap.Logger
	migration *MigrationManager
}

// NewDatabase cria uma nova instância do banco de dados
func NewDatabase(ctx context.Context, config Config, zapLogger *zap.Logger) (*Database, error) {
	// Configurar GORM Logger
	gormLogger := logger.New(
		GormLogAdapter{zapLogger},
		logger.Config{
			SlowThreshold:             config.SlowThreshold,
			LogLevel:                  config.LogLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	// Configurações do GORM
	gormConfig := &gorm.Config{
		Logger:                                   gormLogger,
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
		PrepareStmt:                              true,
	}

	// Conectar ao banco de dados
	var db *gorm.DB
	var err error

	switch config.Driver {
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(config.DSN), gormConfig)
	case "mysql":
		db, err = gorm.Open(mysql.Open(config.DSN), gormConfig)
	case "postgres":
		db, err = gorm.Open(postgres.Open(config.DSN), gormConfig)
	default:
		return nil, fmt.Errorf("driver de banco de dados não suportado: %s", config.Driver)
	}

	if err != nil {
		return nil, fmt.Errorf("falha ao conectar ao banco de dados: %w", err)
	}

	// Configurar pool de conexões
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("falha ao obter instância do banco de dados: %w", err)
	}

	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Testar conexão
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("falha ao testar conexão com banco de dados: %w", err)
	}

	// Criar gerenciador de migração
	migration := NewMigrationManager(db, zapLogger, config.MigrationDir)

	database := &Database{
		db:        db,
		logger:    zapLogger,
		migration: migration,
	}

	// Aplicar migrações apenas se não forem puladas
	if !config.SkipMigrations {
		if err := database.migrate(ctx); err != nil {
			return nil, fmt.Errorf("falha ao aplicar migrações: %w", err)
		}
	} else {
		zapLogger.Info("Migrações foram puladas devido à configuração")
	}

	return database, nil
}

// DB retorna a instância do GORM DB
func (d *Database) DB() *gorm.DB {
	return d.db
}

// Ping verifica a conexão com o banco de dados
func (d *Database) Ping(ctx context.Context) error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Close fecha a conexão com o banco de dados
func (d *Database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// migrate aplica migrações de banco de dados
func (d *Database) migrate(ctx context.Context) error {
	// Auto migração para garantir que a tabela de rotas existe
	if err := d.db.AutoMigrate(&model.RouteEntity{}); err != nil {
		return fmt.Errorf("falha ao aplicar auto migração: %w", err)
	}

	// Aplicar migrações SQL
	if d.migration != nil {
		if err := d.migration.ApplyMigrations(ctx); err != nil {
			// Log do erro, mas não falhe em caso de problemas com migrações
			d.logger.Error("falha ao aplicar migrações SQL", zap.Error(err))
		}
	}

	return nil
}

// CreateMigration cria um novo arquivo de migração
func (d *Database) CreateMigration(name string) (string, error) {
	if d.migration == nil {
		return "", fmt.Errorf("gerenciador de migrações não configurado")
	}
	return d.migration.CreateMigration(name)
}

// GormLogAdapter adapta o zap.Logger para uso com GORM
type GormLogAdapter struct {
	ZapLogger *zap.Logger
}

// Printf implementa a interface de Logger do GORM
func (l GormLogAdapter) Printf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.ZapLogger.Debug(msg)
}
