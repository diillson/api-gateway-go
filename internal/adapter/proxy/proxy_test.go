package proxy_test

import (
	"github.com/diillson/api-gateway-go/internal/adapter/proxy"
	"github.com/diillson/api-gateway-go/internal/domain/model"
	"github.com/diillson/api-gateway-go/internal/mocks"
	"github.com/diillson/api-gateway-go/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReverseProxy_ProxyRequest(t *testing.T) {
	logger := testutils.TestLogger(t)
	mockCache := new(mocks.MockCache)

	// Setup test server as our backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	}))
	defer backend.Close()

	// Create the proxy
	reverseProxy := proxy.NewReverseProxy(mockCache, logger)

	// Create a test route
	route := &model.Route{
		Path:        "/test",
		ServiceURL:  backend.URL,
		Methods:     []string{"GET"},
		Description: "Test route",
		IsActive:    true,
	}

	// Create a test request
	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()

	// Test proxy request
	err = reverseProxy.ProxyRequest(route, rr, req)
	require.NoError(t, err)

	// Verify the response
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"status":"success"`)
}
