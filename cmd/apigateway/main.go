package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/diillson/api-gateway-go/pkg/config"
	"github.com/diillson/api-gateway-go/pkg/telemetry"
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
func setupServer(router *gin.Engine, logger *zap.Logger) *http.Server {
	// Verificar ambiente
	env := os.Getenv("ENV")

	// Modo de desenvolvimento (HTTP)
	if env == "development" {
		return &http.Server{
			Addr:    ":8080",
			Handler: router,
		}
	}

	// Modo de produção (HTTPS)
	domains := strings.Split(os.Getenv("SERVER_DOMAINS"), ",")
	if len(domains) == 0 || domains[0] == "" {
		logger.Warn("Nenhum domínio configurado para HTTPS. Usando HTTP em produção.")
		return &http.Server{
			Addr:    ":8080",
			Handler: router,
		}
	}

	// Gerenciador de certificados Let's Encrypt
	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(domains...),
		Cache:      autocert.DirCache("./certs"),
		Email:      os.Getenv("LETSENCRYPT_EMAIL"),
	}

	// Servidor HTTPS
	server := &http.Server{
		Addr: ":443",
		TLSConfig: &tls.Config{
			GetCertificate:           certManager.GetCertificate,
			MinVersion:               tls.VersionTLS13,
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				//tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				//tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				//tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				//tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				//tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				//tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_AES_128_GCM_SHA256,
				tls.TLS_AES_256_GCM_SHA384,
				tls.TLS_CHACHA20_POLY1305_SHA256,
			},
		},
		Handler: router,
	}

	// Iniciar servidor HTTP para redirecionamento HTTPS
	go func() {
		httpServer := &http.Server{
			Addr:    ":80",
			Handler: certManager.HTTPHandler(http.HandlerFunc(redirectHTTPS)),
		}

		logger.Info("Iniciando servidor HTTP para redirecionamento HTTPS", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Erro no servidor HTTP", zap.Error(err))
		}
	}()

	return server
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

	// Carregar configuração
	cfg, err := config.LoadConfig("./config")
	if err != nil {
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
	application, err := app.NewApp(logger)
	if err != nil {
		logger.Fatal("Falha ao inicializar aplicação", zap.Error(err))
	}

	// Configurar o router
	router := gin.Default()
	application.RegisterRoutes(router)

	// Configurar servidor HTTP
	server := setupServer(router, logger)

	// Iniciar o servidor em uma goroutine
	go func() {
		logger.Info("Servidor iniciado", zap.String("addr", server.Addr))
		var err error

		if server.TLSConfig != nil {
			// HTTPS
			err = server.ListenAndServeTLS("", "")
		} else {
			// HTTP
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
