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
			Type:        "memory", // Opções: "memory" ou "redis"
			TTL:         5 * time.Minute,
			MaxItems:    10000, // Apenas para cache em memória
			MaxMemoryMB: 100,   // Apenas para cache em memória
			Redis: config.RedisOptions{
				Address:      "localhost:6379",
				Password:     "",
				DB:           0,
				PoolSize:     10,
				MinIdleConns: 5,
				MaxRetries:   3,
				ReadTimeout:  3 * time.Second,
				WriteTimeout: 3 * time.Second,
				DialTimeout:  5 * time.Second,
				PoolTimeout:  4 * time.Second,
				IdleTimeout:  5 * time.Minute,
				MaxConnAge:   30 * time.Minute,
			},
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
			Provider:      "otlp",
			Endpoint:      "otel-collector:4317",
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

	// Adicionar comentários para documentação
	yamlStr := string(data)

	// Usar regex para adicionar comentários aos parâmetros específicos do certificado
	re := regexp.MustCompile(`(\s+certfile:\s+/path/to/cert.pem)`)
	yamlStr = re.ReplaceAllString(yamlStr, `$1  # Opcional: Caminhos para certificados próprios`)

	re = regexp.MustCompile(`(\s+keyfile:\s+/path/to/key.pem)`)
	yamlStr = re.ReplaceAllString(yamlStr, `$1  # Opcional: Caminhos para certificados próprios`)

	re = regexp.MustCompile(`(\s+domains:)`)
	yamlStr = re.ReplaceAllString(yamlStr, `$1  # Domínios para Let's Encrypt (ignorados se certificados próprios forem usados)`)

	re = regexp.MustCompile(`(\s+tls:\s+false)`)
	yamlStr = re.ReplaceAllString(yamlStr, `$1  # Habilitar HTTPS`)

	re = regexp.MustCompile(`(\s+port:\s+8080)`)
	yamlStr = re.ReplaceAllString(yamlStr, `$1  # Porta HTTP para o servidor`)

	// Usar regex para adicionar comentários aos parâmetros específicos do Redis
	re = regexp.MustCompile(`(\s+type:\s+memory)`)
	yamlStr = re.ReplaceAllString(yamlStr, `$1  # Opções: "memory" ou "redis"`)

	re = regexp.MustCompile(`(\s+skipmigrations:\s+false)`)
	yamlStr = re.ReplaceAllString(yamlStr, `$1  # Opção: false aplica migrações (padrão), true pula`)

	// Adicionar comentários para configurações do Redis
	re = regexp.MustCompile(`(\s+redis:)`)
	yamlStr = re.ReplaceAllString(yamlStr, `$1  # Configurações específicas para Redis`)

	re = regexp.MustCompile(`(\s+address:\s+localhost:6379)`)
	yamlStr = re.ReplaceAllString(yamlStr, `$1  # Endereço do servidor Redis (host:port)`)

	re = regexp.MustCompile(`(\s+poolsize:\s+10)`)
	yamlStr = re.ReplaceAllString(yamlStr, `$1  # Número máximo de conexões no pool`)

	re = regexp.MustCompile(`(\s+minidleconns:\s+5)`)
	yamlStr = re.ReplaceAllString(yamlStr, `$1  # Número mínimo de conexões ociosas mantidas abertas`)

	re = regexp.MustCompile(`(\s+maxretries:\s+3)`)
	yamlStr = re.ReplaceAllString(yamlStr, `$1  # Número máximo de tentativas de reconexão`)

	re = regexp.MustCompile(`(\s+readtimeout:\s+3s)`)
	yamlStr = re.ReplaceAllString(yamlStr, `$1  # Timeout para operações de leitura`)

	re = regexp.MustCompile(`(\s+writetimeout:\s+3s)`)
	yamlStr = re.ReplaceAllString(yamlStr, `$1  # Timeout para operações de escrita`)

	re = regexp.MustCompile(`(\s+dialtimeout:\s+5s)`)
	yamlStr = re.ReplaceAllString(yamlStr, `$1  # Timeout para estabelecer nova conexão`)

	re = regexp.MustCompile(`(\s+pooltimeout:\s+4s)`)
	yamlStr = re.ReplaceAllString(yamlStr, `$1  # Timeout para obter conexão do pool`)

	re = regexp.MustCompile(`(\s+idletimeout:\s+5m0s)`)
	yamlStr = re.ReplaceAllString(yamlStr, `$1  # Tempo máximo que uma conexão pode ficar ociosa`)

	re = regexp.MustCompile(`(\s+maxconnage:\s+30m0s)`)
	yamlStr = re.ReplaceAllString(yamlStr, `$1  # Tempo máximo de vida da conexão`)

	// Explicação sobre os modos de cache
	yamlStr = yamlStr + `
    # GUIA DE CONFIGURAÇÃO REDIS:
    # Para usar Redis como cache, configure as seguintes opções:
    #   cache:
    #     enabled: true
    #     type: "redis"
    #     ttl: "5m"
    #     redis:
    #       address: "redis-server:6379"
    #       password: "sua-senha-aqui" # Opcional
    #       db: 0 # Índice do banco de dados Redis
    #
    # Configurações avançadas do Redis para alta performance:
    # - poolsize: Aumentar para workloads com muitas requisições concorrentes
    # - minidleconns: Manter 25-30% do poolsize para inicializações rápidas
    # - maxretries: 3-5 é razoável para sistemas resilientes
    # - dialtimeout: 5-10s para ambientes com latência de rede
    #
    # Para ambientes de produção, recomenda-se:
    #   - Usar Redis em cluster para alta disponibilidade
    #   - Configurar maxretries adequadamente
    #   - Ajustar timeouts conforme a latência da rede
    #   - Monitorar performance do Redis com redis-cli INFO
    `

	// Escrever arquivo
	err = os.WriteFile(outputPath, []byte(yamlStr), 0644)
	if err != nil {
		fmt.Printf("Erro ao escrever arquivo: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Arquivo de configuração gerado em: %s\n", outputPath)
	fmt.Println("As configurações específicas do Redis estão documentadas no arquivo.")
}
