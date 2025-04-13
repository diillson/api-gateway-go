package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/diillson/api-gateway-go/pkg/config"
	"gopkg.in/yaml.v3"
)

func main() {
	var (
		outputPath string
		force      bool
	)

	flag.StringVar(&outputPath, "output", "config.yaml", "Caminho para o arquivo de configuração de saída")
	flag.BoolVar(&force, "force", false, "Sobrescrever arquivo se existir")
	flag.Parse()

	// Verificar se o arquivo já existe
	if _, err := os.Stat(outputPath); err == nil && !force {
		fmt.Printf("Erro: arquivo %s já existe. Use --force para sobrescrever.\n", outputPath)
		os.Exit(1)
	}

	// Criar configuração com valores padrão
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:           8080,
			Host:           "0.0.0.0",
			ReadTimeout:    5 * time.Second,
			WriteTimeout:   10 * time.Second,
			IdleTimeout:    30 * time.Second,
			MaxHeaderBytes: 1 << 20, // 1 MB
			TLS:            false,
			CertFile:       "/path/to/cert.pem",
			KeyFile:        "/path/to/key.pem",
			BaseURL:        "https://api.example.com",
			Domains:        []string{"api.example.com"},
		},
		Database: config.DatabaseConfig{
			Driver:          "postgres",
			DSN:             "./routes.db",
			MaxIdleConns:    10,
			MaxOpenConns:    50,
			ConnMaxLifetime: 1 * time.Hour,
			LogLevel:        "warn",
			SlowThreshold:   200 * time.Millisecond,
			MigrationDir:    "./migrations",
			SkipMigrations:  false, // Opção: false aplica migrações (padrão), true pula
		},
		Cache: config.CacheConfig{
			Enabled:     true,
			Type:        "memory",
			Address:     "localhost:6379",
			Password:    "",
			DB:          0,
			TTL:         5 * time.Minute,
			MaxItems:    10000,
			MaxMemoryMB: 100,
		},
		Auth: config.AuthConfig{
			Enabled:          true,
			JWTSecret:        "your-secret-key-here",
			TokenExpiration:  24 * time.Hour,
			RefreshEnabled:   true,
			RefreshDuration:  168 * time.Hour,
			AllowedOrigins:   []string{"*"},
			AdminUsers:       []string{"admin"},
			PasswordMinLen:   8,
			RequireTwoFactor: false,
		},
		Metrics: config.MetricsConfig{
			Enabled:        true,
			PrometheusPath: "/metrics",
			ReportInterval: 15 * time.Second,
		},
		Logging: config.LoggingConfig{
			Level:      "info",
			Format:     "json",
			OutputPath: "stdout",
			ErrorPath:  "stderr",
			Production: true,
		},
		Tracing: config.TracingConfig{
			Enabled:       false,
			Provider:      "opentelemetry",
			Endpoint:      "localhost:4317",
			ServiceName:   "api-gateway",
			SamplingRatio: 0.1,
		},
		Features: config.FeaturesConfig{
			RateLimiter:       true,
			CircuitBreaker:    true,
			Caching:           true,
			HealthCheck:       true,
			AdminAPI:          true,
			Analytics:         true,
			Monitoring:        true,
			AutoRouteRegister: false,
		},
	}

	// Converter para YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		fmt.Printf("Erro ao serializar configuração: %v\n", err)
		os.Exit(1)
	}

	// Adicionar comentário documentando a opção skipmigrations
	yamlStr := string(data)

	// Usar regex para encontrar a linha com skipmigrations e adicionar o comentário após o valor
	re := regexp.MustCompile(`(\s+skipmigrations:\s+false)`)
	yamlStr = re.ReplaceAllString(yamlStr, `$1  # Opção: false aplica migrações (padrão), true pula`)

	// Escrever arquivo
	err = os.WriteFile(outputPath, []byte(yamlStr), 0644)
	if err != nil {
		fmt.Printf("Erro ao escrever arquivo: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Arquivo de configuração gerado em: %s\n", outputPath)
}
