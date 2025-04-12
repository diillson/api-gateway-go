package testutils

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// TestLogger cria um logger zap para testes
func TestLogger(t *testing.T) *zap.Logger {
	return zaptest.NewLogger(t)
}

// SetupTestRouter configura um router Gin para testes
func SetupTestRouter(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())
	return router
}

// MakeRequest executa uma requisição HTTP de teste
func MakeRequest(t *testing.T, router *gin.Engine, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	var reqBody io.Reader

	if body != nil {
		switch v := body.(type) {
		case string:
			reqBody = strings.NewReader(v)
		case []byte:
			reqBody = strings.NewReader(string(v))
		default:
			jsonData, err := json.Marshal(body)
			require.NoError(t, err, "Failed to marshal request body")
			reqBody = strings.NewReader(string(jsonData))
		}
	}

	req, err := http.NewRequest(method, path, reqBody)
	require.NoError(t, err, "Failed to create HTTP request")

	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Adicionar cabeçalhos personalizados
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	return resp
}

// ParseResponse analisa a resposta JSON para uma estrutura
func ParseResponse(t *testing.T, resp *httptest.ResponseRecorder, dst interface{}) {
	require.NotNil(t, resp, "Response recorder is nil")

	err := json.Unmarshal(resp.Body.Bytes(), dst)
	require.NoError(t, err, "Failed to parse response: %s", resp.Body.String())
}

// ContextWithTimeout cria um contexto com timeout para testes
func ContextWithTimeout(t *testing.T) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

// CheckError verifica se um erro contém uma mensagem específica
func CheckError(t *testing.T, err error, message string) {
	require.Error(t, err, "Expected an error but got nil")
	require.Contains(t, err.Error(), message, "Error message does not contain expected text")
}

// RequireHTTPStatus verifica o status HTTP da resposta
func RequireHTTPStatus(t *testing.T, resp *httptest.ResponseRecorder, status int) {
	require.Equal(t, status, resp.Code, "Expected HTTP status %d but got %d, body: %s",
		status, resp.Code, resp.Body.String())
}

// RequireJSONContentType verifica se o Content-Type da resposta é JSON
func RequireJSONContentType(t *testing.T, resp *httptest.ResponseRecorder) {
	contentType := resp.Header().Get("Content-Type")
	require.Contains(t, contentType, "application/json",
		"Expected Content-Type to contain application/json but got %s", contentType)
}
