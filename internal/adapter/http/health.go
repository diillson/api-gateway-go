package http

import (
	"context"
	"github.com/diillson/api-gateway-go/internal/app/route"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HealthChecker implementa endpoints de health check
type HealthChecker struct {
	router       *route.Service
	db           DatabaseChecker
	cache        CacheChecker
	logger       *zap.Logger
	dependencies []Dependency
}

// DatabaseChecker define a interface para verificar o banco de dados
type DatabaseChecker interface {
	Ping(ctx context.Context) error
}

// CacheChecker define a interface para verificar o cache
type CacheChecker interface {
	Ping(ctx context.Context) error
}

// Dependency representa um componente do qual o sistema depende
type Dependency struct {
	Name     string
	Check    func(context.Context) error
	Critical bool // Se true, falha deste componente faz o health check falhar
}

// NewHealthChecker cria um novo health checker
func NewHealthChecker(router *route.Service, db DatabaseChecker, cache CacheChecker, logger *zap.Logger) *HealthChecker {
	hc := &HealthChecker{
		router: router,
		db:     db,
		cache:  cache,
		logger: logger,
	}

	// Adicionar dependências
	hc.dependencies = []Dependency{
		{
			Name:     "database",
			Check:    db.Ping,
			Critical: true,
		},
		{
			Name:     "cache",
			Check:    cache.Ping,
			Critical: false,
		},
	}

	return hc
}

// LivenessCheck verifica se o aplicativo está vivo (execução básica)
func (h *HealthChecker) LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "UP",
		"time":   time.Now(),
	})
}

// ReadinessCheck verifica se o aplicativo está pronto para receber tráfego
func (h *HealthChecker) ReadinessCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	status := http.StatusOK
	result := gin.H{
		"status": "UP",
		"time":   time.Now(),
		"checks": make(map[string]interface{}),
	}

	checks := result["checks"].(map[string]interface{})
	var wg sync.WaitGroup

	// Verificar cada dependência em paralelo
	for _, dep := range h.dependencies {
		wg.Add(1)
		go func(d Dependency) {
			defer wg.Done()

			start := time.Now()
			err := d.Check(ctx)
			duration := time.Since(start)

			depStatus := "UP"
			if err != nil {
				depStatus = "DOWN"
				h.logger.Error("health check falhou",
					zap.String("dependency", d.Name),
					zap.Error(err))

				if d.Critical {
					status = http.StatusServiceUnavailable
				}
			}

			checks[d.Name] = gin.H{
				"status":   depStatus,
				"time":     duration.String(),
				"critical": d.Critical,
			}
		}(dep)
	}

	// Adicionar check do roteador
	wg.Add(1)
	go func() {
		defer wg.Done()

		start := time.Now()
		routes, err := h.router.GetRoutes(ctx)
		duration := time.Since(start)

		routeStatus := "UP"
		routeDetails := gin.H{
			"status":   routeStatus,
			"time":     duration.String(),
			"critical": true,
		}

		if err != nil {
			routeStatus = "DOWN"
			routeDetails["status"] = routeStatus
			status = http.StatusServiceUnavailable
			h.logger.Error("health check do roteador falhou", zap.Error(err))
		} else {
			routeDetails["count"] = len(routes)
		}

		checks["router"] = routeDetails
	}()

	wg.Wait()

	// Atualizar status geral
	if status != http.StatusOK {
		result["status"] = "DOWN"
	}

	c.JSON(status, result)
}

// DetailedHealth fornece informações detalhadas sobre o sistema
func (h *HealthChecker) DetailedHealth(c *gin.Context) {
	// Apenas para administradores
	//if !isAdmin(c) {
	//	c.JSON(http.StatusForbidden, gin.H{"error": "Acesso negado"})
	//	return
	//}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	details := gin.H{
		"status":      "UP",
		"time":        time.Now(),
		"version":     getVersion(),
		"environment": getEnvironment(),
		"checks":      make(map[string]interface{}),
		"system":      getSystemInfo(),
	}

	checks := details["checks"].(map[string]interface{})

	// Verificações de saúde básicas (reutilizar código do ReadinessCheck)
	status := http.StatusOK
	var wg sync.WaitGroup

	for _, dep := range h.dependencies {
		wg.Add(1)
		go func(d Dependency) {
			defer wg.Done()

			start := time.Now()
			err := d.Check(ctx)
			duration := time.Since(start)

			depStatus := "UP"
			if err != nil {
				depStatus = "DOWN"
				if d.Critical {
					status = http.StatusServiceUnavailable
				}
			}

			checks[d.Name] = gin.H{
				"status":   depStatus,
				"time":     duration.String(),
				"critical": d.Critical,
				"error": func() interface{} {
					if err != nil {
						return err.Error()
					}
					return nil
				}(),
			}
		}(dep)
	}

	wg.Wait()

	if status != http.StatusOK {
		details["status"] = "DOWN"
	}

	c.JSON(status, details)
}

// getVersion retorna a versão do aplicativo
func getVersion() string {
	return os.Getenv("APP_VERSION")
}

// getEnvironment retorna o ambiente atual
func getEnvironment() string {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		return "development"
	}
	return env
}

// getSystemInfo retorna informações sobre o sistema
func getSystemInfo() gin.H {
	return gin.H{
		"go_version":    runtime.Version(),
		"go_os":         runtime.GOOS,
		"go_arch":       runtime.GOARCH,
		"num_cpu":       runtime.NumCPU(),
		"num_goroutine": runtime.NumGoroutine(),
		"memory_alloc":  getMemoryStats(),
	}
}

// getMemoryStats retorna estatísticas de memória
func getMemoryStats() gin.H {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return gin.H{
		"alloc_mb":       float64(m.Alloc) / 1024 / 1024,
		"total_alloc_mb": float64(m.TotalAlloc) / 1024 / 1024,
		"sys_mb":         float64(m.Sys) / 1024 / 1024,
		"num_gc":         m.NumGC,
	}
}
