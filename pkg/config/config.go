package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config representa a configuração completa da aplicação
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Cache    CacheConfig
	Auth     AuthConfig
	Metrics  MetricsConfig
	Logging  LoggingConfig
	Tracing  TracingConfig
	Features FeaturesConfig
}

// ServerConfig contém configurações do servidor HTTP
type ServerConfig struct {
	Port           int
	Host           string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	MaxHeaderBytes int
	TLS            bool
	CertFile       string
	KeyFile        string
	BaseURL        string
	Domains        []string
}

// DatabaseConfig contém configurações do banco de dados
type DatabaseConfig struct {
	Driver          string
	DSN             string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	LogLevel        string
	SlowThreshold   time.Duration
	MigrationDir    string
	SkipMigrations  bool
}

// RedisOptions contém configurações específicas para Redis
type RedisOptions struct {
	Address            string
	Password           string
	DB                 int
	PoolSize           int
	MinIdleConns       int
	MaxRetries         int
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	DialTimeout        time.Duration
	PoolTimeout        time.Duration
	IdleTimeout        time.Duration
	MaxConnAge         time.Duration
	ConnectionPoolName string
}

// CacheConfig contém configurações do cache
type CacheConfig struct {
	Enabled     bool
	Type        string // redis, memory
	TTL         time.Duration
	MaxItems    int // apenas para cache em memória
	MaxMemoryMB int // apenas para cache em memória
	Redis       RedisOptions
}

// AuthConfig contém configurações de autenticação
type AuthConfig struct {
	Enabled          bool
	JWTSecret        string
	TokenExpiration  time.Duration
	RefreshEnabled   bool
	RefreshDuration  time.Duration
	AllowedOrigins   []string
	AdminUsers       []string
	PasswordMinLen   int
	RequireTwoFactor bool
}

// MetricsConfig contém configurações de métricas
type MetricsConfig struct {
	Enabled        bool
	PrometheusPath string
	ReportInterval time.Duration
}

// LoggingConfig contém configurações de logging
type LoggingConfig struct {
	Level      string
	Format     string // json, console
	OutputPath string // stdout, file path
	ErrorPath  string
	Production bool
}

// TracingConfig contém configurações de rastreamento
type TracingConfig struct {
	Enabled       bool
	Provider      string // opentelemetry, jaeger
	Endpoint      string
	ServiceName   string
	SamplingRatio float64
}

// FeaturesConfig contém flags de recursos
type FeaturesConfig struct {
	RateLimiter       bool
	CircuitBreaker    bool
	Caching           bool
	HealthCheck       bool
	AdminAPI          bool
	Analytics         bool
	Monitoring        bool
	AutoRouteRegister bool
}

// LoadConfig carrega a configuração de diversas fontes (arquivos, env, defaults)
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Definir valores padrão
	setDefaults(v)

	// Configuração de leitura
	v.SetConfigName("config") // nome do arquivo de configuração
	v.SetConfigType("yaml")   // tipo do arquivo de configuração

	// Locais para procurar arquivos de configuração
	if configPath != "" {
		v.AddConfigPath(configPath)
	}
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/apigateway")

	// Ler arquivo de configuração
	if err := v.ReadInConfig(); err != nil {
		// Ignorar se o arquivo não for encontrado
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("erro ao ler arquivo de configuração: %w", err)
		}
	}

	// Ler variáveis de ambiente com prefixo AG_
	v.SetEnvPrefix("AG")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Mapear configuração para a estrutura
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("erro ao mapear configuração: %w", err)
	}

	// Validar a configuração
	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// setDefaults define valores padrão para a configuração
func setDefaults(v *viper.Viper) {
	// Servidor
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.readTimeout", "5s")
	v.SetDefault("server.writeTimeout", "10s")
	v.SetDefault("server.idleTimeout", "30s")
	v.SetDefault("server.maxHeaderBytes", 1<<20) // 1 MB
	v.SetDefault("server.tls", false)

	// Banco de dados
	v.SetDefault("database.driver", "postgres")
	v.SetDefault("database.dsn", "postgres://postgres:postgres@postgres:5432/apigateway?sslmode=disable")
	v.SetDefault("database.maxIdleConns", 10)
	v.SetDefault("database.maxOpenConns", 50)
	v.SetDefault("database.connMaxLifetime", "1h")
	v.SetDefault("database.logLevel", "warn")
	v.SetDefault("database.slowThreshold", "200ms")
	v.SetDefault("database.migrationDir", "./migrations")

	// Redis
	v.SetDefault("cache.redis.address", "localhost:6379")
	v.SetDefault("cache.redis.db", 0)
	v.SetDefault("cache.redis.pool_size", 10)
	v.SetDefault("cache.redis.min_idle_conns", 5)
	v.SetDefault("cache.redis.max_retries", 3)
	v.SetDefault("cache.redis.read_timeout", "3s")
	v.SetDefault("cache.redis.write_timeout", "3s")
	v.SetDefault("cache.redis.dial_timeout", "5s")
	v.SetDefault("cache.redis.pool_timeout", "4s")
	v.SetDefault("cache.redis.idle_timeout", "5m")
	v.SetDefault("cache.redis.max_conn_age", "30m")

	// Cache
	v.SetDefault("cache.enabled", true)
	v.SetDefault("cache.type", "memory")
	v.SetDefault("cache.ttl", "5m")
	v.SetDefault("cache.maxItems", 10000)
	v.SetDefault("cache.maxMemoryMB", 100)

	// Autenticação
	v.SetDefault("auth.enabled", true)
	v.SetDefault("auth.tokenExpiration", "24h")
	v.SetDefault("auth.refreshEnabled", true)
	v.SetDefault("auth.refreshDuration", "168h") // 7 dias
	v.SetDefault("auth.passwordMinLen", 8)
	v.SetDefault("auth.requireTwoFactor", false)

	// Métricas
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.prometheusPath", "/metrics")
	v.SetDefault("metrics.reportInterval", "15s")

	// Logging
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.outputPath", "stdout")
	v.SetDefault("logging.errorPath", "stderr")
	v.SetDefault("logging.production", true)

	// Tracing
	v.SetDefault("tracing.enabled", false)
	v.SetDefault("tracing.provider", "opentelemetry")
	v.SetDefault("tracing.samplingRatio", 0.1) // 10% das requisições
	v.SetDefault("tracing.serviceName", "api-gateway")

	// Features
	v.SetDefault("features.rateLimiter", true)
	v.SetDefault("features.circuitBreaker", true)
	v.SetDefault("features.caching", true)
	v.SetDefault("features.healthCheck", true)
	v.SetDefault("features.adminAPI", true)
	v.SetDefault("features.analytics", true)
	v.SetDefault("features.monitoring", true)
	v.SetDefault("features.autoRouteRegister", false)
}

// validateConfig valida a configuração
func validateConfig(config *Config) error {
	// Validar JWT Secret
	if config.Auth.Enabled && config.Auth.JWTSecret == "" {
		// Gerar um aviso se o segredo JWT não estiver definido
		fmt.Println("AVISO: JWT_SECRET_KEY não está definido. Uma chave temporária será gerada, mas isso não é recomendado para produção.")
	}

	// Validar configuração de TLS
	if config.Server.TLS {
		if config.Server.CertFile == "" || config.Server.KeyFile == "" {
			return fmt.Errorf("TLS habilitado, mas CertFile ou KeyFile não estão definidos")
		}
	}

	// Validar configuração do banco de dados
	validDrivers := map[string]bool{"sqlite": true, "mysql": true, "postgres": true}
	if !validDrivers[config.Database.Driver] {
		return fmt.Errorf("driver de banco de dados inválido: %s", config.Database.Driver)
	}

	// Validar configuração de cache
	if config.Cache.Enabled {
		validTypes := map[string]bool{"memory": true, "redis": true}
		if !validTypes[config.Cache.Type] {
			return fmt.Errorf("tipo de cache inválido: %s", config.Cache.Type)
		}

		if config.Cache.Type == "redis" && config.Cache.Redis.Address == "" {
			return fmt.Errorf("tipo de cache redis requer um endereço")
		}
	}

	return nil
}
