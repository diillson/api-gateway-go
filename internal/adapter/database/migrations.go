package database

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Migration representa uma migração de banco de dados
type Migration struct {
	ID        uint  `gorm:"primaryKey"`
	Version   int64 `gorm:"uniqueIndex"`
	Name      string
	AppliedAt time.Time
}

// MigrationManager gerencia migrações de banco de dados
type MigrationManager struct {
	db        *gorm.DB
	logger    *zap.Logger
	directory string
}

// NewMigrationManager cria um novo gerenciador de migrações
func NewMigrationManager(db *gorm.DB, logger *zap.Logger, directory string) *MigrationManager {
	return &MigrationManager{
		db:        db,
		logger:    logger,
		directory: directory,
	}
}

// Initialize inicializa a tabela de migrações
func (m *MigrationManager) Initialize(ctx context.Context) error {
	// Cria a tabela de migrações se não existir
	if err := m.db.WithContext(ctx).AutoMigrate(&Migration{}); err != nil {
		return fmt.Errorf("falha ao criar tabela de migrações: %w", err)
	}

	return nil
}

// ApplyMigrations aplica todas as migrações pendentes
func (m *MigrationManager) ApplyMigrations(ctx context.Context) error {
	// Inicializar a tabela de migrações
	if err := m.Initialize(ctx); err != nil {
		return err
	}

	// Buscar migrações já aplicadas
	var appliedMigrations []Migration
	if err := m.db.WithContext(ctx).Order("version").Find(&appliedMigrations).Error; err != nil {
		return fmt.Errorf("falha ao buscar migrações aplicadas: %w", err)
	}

	// Mapear migrações aplicadas pelo número de versão
	appliedVersions := make(map[int64]bool)
	for _, migration := range appliedMigrations {
		appliedVersions[migration.Version] = true
	}

	// Listar arquivos de migração
	migrationFiles, err := m.findMigrationFiles()
	if err != nil {
		return fmt.Errorf("falha ao listar arquivos de migração: %w", err)
	}

	// Ordenar os arquivos por número de versão
	sort.Slice(migrationFiles, func(i, j int) bool {
		return migrationFiles[i].Version < migrationFiles[j].Version
	})

	// Aplicar migrações pendentes em transação
	for _, file := range migrationFiles {
		if appliedVersions[file.Version] {
			m.logger.Info("Migração já aplicada", zap.Int64("version", file.Version), zap.String("name", file.Name))
			continue
		}

		m.logger.Info("Aplicando migração", zap.Int64("version", file.Version), zap.String("name", file.Name))

		// Ler o conteúdo do arquivo
		content, err := os.ReadFile(file.Path)
		if err != nil {
			return fmt.Errorf("falha ao ler arquivo de migração: %w", err)
		}

		// Iniciar uma transação
		tx := m.db.WithContext(ctx).Begin()
		if tx.Error != nil {
			return fmt.Errorf("falha ao iniciar transação: %w", tx.Error)
		}

		// Dividir o conteúdo em comandos SQL individuais
		sqlCommands := splitSQLCommands(string(content))

		// Executar cada comando SQL individualmente
		for _, sqlCmd := range sqlCommands {
			sqlCmd = strings.TrimSpace(sqlCmd)
			if sqlCmd == "" {
				continue
			}

			if err := tx.Exec(sqlCmd).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("falha ao executar migração: %w", err)
			}
		}

		// Registrar a migração como aplicada
		if err := tx.Create(&Migration{
			Version:   file.Version,
			Name:      file.Name,
			AppliedAt: time.Now(),
		}).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("falha ao registrar migração: %w", err)
		}

		// Commit da transação
		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("falha ao confirmar transação: %w", err)
		}

		m.logger.Info("Migração aplicada com sucesso", zap.Int64("version", file.Version), zap.String("name", file.Name))
	}

	return nil
}

// Função auxiliar para dividir o SQL em comandos individuais
func splitSQLCommands(sql string) []string {
	// Dividir por ponto e vírgula, mas ignorar ponto e vírgula dentro de strings ou comentários
	var commands []string
	var currentCommand strings.Builder
	inString := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(sql); i++ {
		ch := sql[i]

		// Tratamento de comentários de linha
		if !inString && !inBlockComment && i < len(sql)-1 && ch == '-' && sql[i+1] == '-' {
			inLineComment = true
			currentCommand.WriteByte(ch)
			continue
		}

		// Fim de comentário de linha
		if inLineComment && ch == '\n' {
			inLineComment = false
			currentCommand.WriteByte(ch)
			continue
		}

		// Tratamento de comentários de bloco
		if !inString && !inLineComment && i < len(sql)-1 && ch == '/' && sql[i+1] == '*' {
			inBlockComment = true
			currentCommand.WriteByte(ch)
			continue
		}

		// Fim de comentário de bloco
		if inBlockComment && i < len(sql)-1 && ch == '*' && sql[i+1] == '/' {
			inBlockComment = false
			currentCommand.WriteString("*/")
			i++ // Pular o próximo caractere
			continue
		}

		// Tratamento de strings
		if !inLineComment && !inBlockComment && ch == '\'' {
			inString = !inString
		}

		// Identificar comandos separados por ponto e vírgula
		if !inString && !inLineComment && !inBlockComment && ch == ';' {
			currentCommand.WriteByte(ch)
			commands = append(commands, currentCommand.String())
			currentCommand.Reset()
			continue
		}

		// Adicionar caractere ao comando atual
		currentCommand.WriteByte(ch)
	}

	// Adicionar o último comando se não estiver vazio
	lastCommand := strings.TrimSpace(currentCommand.String())
	if lastCommand != "" {
		commands = append(commands, lastCommand)
	}

	return commands
}

// MigrationFile representa um arquivo de migração
type MigrationFile struct {
	Version int64
	Name    string
	Path    string
}

// findMigrationFiles encontra todos os arquivos de migração .sql
func (m *MigrationManager) findMigrationFiles() ([]MigrationFile, error) {
	var files []MigrationFile

	err := filepath.Walk(m.directory, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(info.Name(), ".sql") {
			return nil
		}

		// Extrair versão e nome do arquivo (formato: YYYYMMDDHHMMSS_name.sql)
		parts := strings.SplitN(info.Name(), "_", 2)
		if len(parts) != 2 {
			m.logger.Warn("Formato de arquivo de migração inválido", zap.String("file", info.Name()))
			return nil
		}

		version, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			m.logger.Warn("Versão de migração inválida", zap.String("file", info.Name()))
			return nil
		}

		name := strings.TrimSuffix(parts[1], ".sql")

		files = append(files, MigrationFile{
			Version: version,
			Name:    name,
			Path:    path,
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, errors.New("nenhum arquivo de migração encontrado")
	}

	return files, nil
}

// CreateMigration cria um novo arquivo de migração
func (m *MigrationManager) CreateMigration(name string) (string, error) {
	// Sanitizar o nome
	name = strings.ReplaceAll(strings.ToLower(name), " ", "_")

	// Gerar timestamp para versão
	timestamp := time.Now().Format("20060102150405")

	// Criar o diretório se não existir
	if err := os.MkdirAll(m.directory, 0755); err != nil {
		return "", fmt.Errorf("falha ao criar diretório: %w", err)
	}

	// Nome do arquivo
	filename := fmt.Sprintf("%s_%s.sql", timestamp, name)
	filepath := filepath.Join(m.directory, filename)

	// Criar arquivo vazio
	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("falha ao criar arquivo: %w", err)
	}

	if err := file.Close(); err != nil {
		return "", fmt.Errorf("falha ao fechar arquivo: %w", err)
	}

	return filepath, nil
}
