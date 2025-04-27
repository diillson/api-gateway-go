package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/diillson/api-gateway-go/pkg/config"
	"github.com/diillson/api-gateway-go/pkg/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/diillson/api-gateway-go/internal/app"
	"github.com/diillson/api-gateway-go/pkg/logging"
	"github.com/gin-gonic/gin"
)

// Função para configurar servidor HTTPS
func setupServer(router *gin.Engine, cfg *config.Config, logger *zap.Logger) *http.Server {
	// Verificar ambiente
	env := os.Getenv("ENV")

	// Modo de desenvolvimento ou TLS desabilitado (HTTP)
	if env == "development" || !cfg.Server.TLS {
		logger.Info("Iniciando em modo HTTP",
			zap.Bool("tls_disabled", !cfg.Server.TLS),
			zap.String("env", env),
			zap.Int("port", cfg.Server.Port))

		return &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
			Handler: router,
		}
	}

	// Verificar se o usuário forneceu certificados próprios
	hasCertificates := cfg.Server.CertFile != "" && cfg.Server.KeyFile != ""
	if hasCertificates {
		// Verificar se os arquivos existem
		if _, err := os.Stat(cfg.Server.CertFile); os.IsNotExist(err) {
			logger.Error("Arquivo de certificado não encontrado",
				zap.String("certFile", cfg.Server.CertFile))
			hasCertificates = false
		}

		if _, err := os.Stat(cfg.Server.KeyFile); os.IsNotExist(err) {
			logger.Error("Arquivo de chave privada não encontrado",
				zap.String("keyFile", cfg.Server.KeyFile))
			hasCertificates = false
		}
	}

	// Se o usuário forneceu certificados próprios, use-os
	if hasCertificates {
		logger.Info("Usando certificados TLS fornecidos pelo usuário",
			zap.String("certFile", cfg.Server.CertFile),
			zap.String("keyFile", cfg.Server.KeyFile))

		// Servidor HTTPS com certificados fornecidos
		server := &http.Server{
			Addr:    ":443",
			Handler: router,
			TLSConfig: &tls.Config{
				MinVersion:               tls.VersionTLS13,
				PreferServerCipherSuites: true,
				CipherSuites: []uint16{
					tls.TLS_AES_128_GCM_SHA256,
					tls.TLS_AES_256_GCM_SHA384,
					tls.TLS_CHACHA20_POLY1305_SHA256,
				},
			},
		}

		// Iniciar o redirecionador HTTP -> HTTPS
		go startHTTPRedirector(logger)

		// Este servidor deverá ser iniciado com:
		// server.ListenAndServeTLS(cfg.Server.CertFile, cfg.Server.KeyFile)
		return server
	}

	// CASO NÃO TENHA CERTIFICADOS PRÓPRIOS - TENTAR LET'S ENCRYPT

	// Modo de produção com Let's Encrypt (HTTPS automático)
	var domains []string

	// Priorizar variável de ambiente se estiver definida
	serverDomains := os.Getenv("SERVER_DOMAINS")
	if serverDomains != "" {
		domains = strings.Split(serverDomains, ",")
		logger.Info("Usando domínios da variável SERVER_DOMAINS",
			zap.Strings("domains", domains))
	} else if len(cfg.Server.Domains) > 0 {
		domains = cfg.Server.Domains
		logger.Info("Usando domínios do arquivo de configuração",
			zap.Strings("domains", domains))
	}

	// Verificar se temos domínios válidos (e remover 'localhost')
	validDomains := make([]string, 0)
	for _, domain := range domains {
		if domain != "" && domain != "localhost" && domain != "127.0.0.1" {
			validDomains = append(validDomains, domain)
		}
	}

	if len(validDomains) == 0 {
		logger.Warn("Nenhum domínio válido configurado para Let's Encrypt. Usando HTTP.",
			zap.Strings("domains", domains))
		return &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
			Handler: router,
		}
	}

	// Obter email para Let's Encrypt
	email := os.Getenv("LETSENCRYPT_EMAIL")
	if email == "" {
		logger.Warn("Email para Let's Encrypt não configurado. Usando valor anônimo.")
	}

	// Configurar gerenciador de certificados Let's Encrypt
	logger.Info("Inicializando Let's Encrypt para domínios",
		zap.Strings("domains", validDomains))

	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(validDomains...),
		Cache:      autocert.DirCache("./certs"),
		Email:      email,
	}

	// Servidor HTTPS com Let's Encrypt
	server := &http.Server{
		Addr: ":443",
		TLSConfig: &tls.Config{
			GetCertificate:           certManager.GetCertificate,
			MinVersion:               tls.VersionTLS13,
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_AES_128_GCM_SHA256,
				tls.TLS_AES_256_GCM_SHA384,
				tls.TLS_CHACHA20_POLY1305_SHA256,
			},
		},
		Handler: router,
	}

	// Iniciar servidor HTTP para desafios Let's Encrypt e redirecionamento HTTPS
	go func() {
		httpServer := &http.Server{
			Addr:    ":80",
			Handler: certManager.HTTPHandler(http.HandlerFunc(redirectHTTPS)),
		}

		logger.Info("Iniciando servidor HTTP para desafios Let's Encrypt e redirecionamento",
			zap.String("addr", httpServer.Addr))

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Erro no servidor HTTP para Let's Encrypt", zap.Error(err))
		}
	}()

	logger.Info("Servidor HTTPS com Let's Encrypt configurado com sucesso",
		zap.Strings("domains", validDomains),
		zap.String("email", email))

	return server
}

// startHTTPRedirector inicia um servidor HTTP simples para redirecionar para HTTPS
func startHTTPRedirector(logger *zap.Logger) {
	httpServer := &http.Server{
		Addr:    ":80",
		Handler: http.HandlerFunc(redirectHTTPS),
	}

	logger.Info("Iniciando servidor HTTP para redirecionamento HTTPS",
		zap.String("addr", httpServer.Addr))

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("Erro no servidor HTTP para redirecionamento", zap.Error(err))
	}
}

// Redirecionamento HTTP -> HTTPS
func redirectHTTPS(w http.ResponseWriter, r *http.Request) {
	target := "https://" + r.Host + r.URL.Path
	if len(r.URL.RawQuery) > 0 {
		target += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, target, http.StatusMovedPermanently)
}

func main() {
	// Inicializar logger
	logger, err := logging.NewLogger()
	if err != nil {
		fmt.Printf("Erro ao inicializar logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	ctx, span := otel.Tracer("api-gateway.main").Start(context.Background(), "Server Initialization")
	defer span.End()

	// Carregar configuração
	cfg, err := config.LoadConfig("./config")
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		logger.Fatal("Falha ao carregar configuração", zap.Error(err))
	}

	// Inicializar o tracer se estiver habilitado
	if cfg.Tracing.Enabled {
		tp, err := telemetry.NewTracerProvider(
			context.Background(),
			cfg.Tracing.ServiceName,
			cfg.Tracing.Endpoint,
			logger,
		)
		if err != nil {
			logger.Error("Falha ao inicializar tracer", zap.Error(err))
		} else {
			logger.Info("Tracer inicializado com sucesso",
				zap.String("provider", cfg.Tracing.Provider),
				zap.String("endpoint", cfg.Tracing.Endpoint))
			defer tp.Shutdown(context.Background())
		}
	}

	// Inicializar aplicação
	application, err := app.NewApp(logger, cfg)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		logger.Fatal("Falha ao inicializar aplicação", zap.Error(err))
	}

	// Configurar o router
	router := gin.Default()
	application.RegisterRoutes(router)

	// Configurar servidor HTTP
	server := setupServer(router, cfg, logger)

	// Iniciar o servidor em uma goroutine
	go func() {
		var err error

		// Se TLS está configurado
		if server.TLSConfig != nil {
			logger.Info("Iniciando servidor HTTPS", zap.String("addr", server.Addr))

			// Verificar se devemos usar certificados próprios
			if cfg.Server.CertFile != "" && cfg.Server.KeyFile != "" {
				logger.Info("Usando certificados próprios",
					zap.String("certFile", cfg.Server.CertFile),
					zap.String("keyFile", cfg.Server.KeyFile))

				// Iniciar com certificados fornecidos
				err = server.ListenAndServeTLS(cfg.Server.CertFile, cfg.Server.KeyFile)
			} else {
				// Iniciar com Let's Encrypt (certificados gerenciados automaticamente)
				logger.Info("Usando Let's Encrypt para certificados")
				err = server.ListenAndServeTLS("", "")
			}
		} else {
			// HTTP
			logger.Info("Iniciando servidor HTTP", zap.String("addr", server.Addr))
			err = server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			logger.Fatal("Erro ao iniciar servidor", zap.Error(err))
		}
	}()

	// Esperar por sinal de interrupção para shutdown gracioso
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Encerrando servidor...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Erro ao encerrar servidor", zap.Error(err))
	}

	logger.Info("Servidor encerrado com sucesso")
}
