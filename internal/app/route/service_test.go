package route_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/diillson/api-gateway-go/internal/app/route"
	"github.com/diillson/api-gateway-go/internal/domain/model"
	"github.com/diillson/api-gateway-go/internal/mocks"
	"github.com/diillson/api-gateway-go/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRouteService_GetRoutes(t *testing.T) {
	logger := testutils.TestLogger(t)
	mockRepo := new(mocks.MockRouteRepository)
	mockCache := new(mocks.MockCache)

	service := route.NewService(mockRepo, mockCache, logger)

	t.Run("successfully from repository", func(t *testing.T) {
		ctx, cancel := testutils.ContextWithTimeout(t)
		defer cancel()

		expectedRoutes := []*model.Route{
			{
				Path:        "/test",
				ServiceURL:  "https://example.com",
				Methods:     []string{"GET"},
				Description: "Test Route",
				IsActive:    true,
			},
		}

		// Cache miss
		mockCache.On("Get", mock.Anything, "routes", mock.AnythingOfType("*[]*model.Route")).
			Return(false, nil).Once()

		// Repository returns data
		mockRepo.On("GetRoutes", mock.Anything).
			Return(expectedRoutes, nil).Once()

		// Cache is updated
		mockCache.On("Set", mock.Anything, "routes", expectedRoutes, 5*time.Minute).
			Return(nil).Once()

		// Execute the test
		routes, err := service.GetRoutes(ctx)

		// Assertions
		require.NoError(t, err)
		assert.Equal(t, expectedRoutes, routes)
		mockCache.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("successfully from cache", func(t *testing.T) {
		ctx, cancel := testutils.ContextWithTimeout(t)
		defer cancel()

		expectedRoutes := []*model.Route{
			{
				Path:        "/cached",
				ServiceURL:  "https://cached.example.com",
				Methods:     []string{"GET"},
				Description: "Cached Route",
				IsActive:    true,
			},
		}

		// Cache hit
		mockCache.On("Get", mock.Anything, "routes", mock.AnythingOfType("*[]*model.Route")).
			Run(func(args mock.Arguments) {
				// Modify the destination slice
				dest := args.Get(2).(*[]*model.Route)
				*dest = expectedRoutes
			}).
			Return(true, nil).Once()

		// Execute the test
		routes, err := service.GetRoutes(ctx)

		// Assertions
		require.NoError(t, err)
		assert.Equal(t, expectedRoutes, routes)
		mockCache.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "GetRoutes")
	})

	t.Run("error from repository", func(t *testing.T) {
		ctx, cancel := testutils.ContextWithTimeout(t)
		defer cancel()

		expectedError := errors.New("database error")

		// Cache miss
		mockCache.On("Get", mock.Anything, "routes", mock.AnythingOfType("*[]*model.Route")).
			Return(false, nil).Once()

		// Repository returns error
		mockRepo.On("GetRoutes", mock.Anything).
			Return(nil, expectedError).Once()

		// Execute the test
		routes, err := service.GetRoutes(ctx)

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, routes)
		assert.Equal(t, expectedError, err)
		mockCache.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("error from cache", func(t *testing.T) {
		ctx, cancel := testutils.ContextWithTimeout(t)
		defer cancel()

		expectedError := errors.New("cache error")

		// Cache error
		mockCache.On("Get", mock.Anything, "routes", mock.AnythingOfType("*[]*model.Route")).
			Return(false, expectedError).Once()

		// Execute the test
		routes, err := service.GetRoutes(ctx)

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, routes)
		assert.Equal(t, expectedError, err)
		mockCache.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "GetRoutes")
	})
}

func TestRouteService_GetRouteByPath(t *testing.T) {
	logger := testutils.TestLogger(t)
	mockRepo := new(mocks.MockRouteRepository)
	mockCache := new(mocks.MockCache)

	service := route.NewService(mockRepo, mockCache, logger)

	t.Run("route found", func(t *testing.T) {
		ctx, cancel := testutils.ContextWithTimeout(t)
		defer cancel()

		path := "/test"
		expectedRoute := &model.Route{
			Path:        path,
			ServiceURL:  "https://example.com",
			Methods:     []string{"GET"},
			Description: "Test Route",
			IsActive:    true,
		}

		// Cache miss
		cacheKey := "route:" + path
		mockCache.On("Get", mock.Anything, cacheKey, mock.AnythingOfType("*model.Route")).
			Return(false, nil).Once()

		// Repository returns data
		mockRepo.On("GetRouteByPath", mock.Anything, path).
			Return(expectedRoute, nil).Once()

		// Cache is updated
		mockCache.On("Set", mock.Anything, cacheKey, expectedRoute, 5*time.Minute).
			Return(nil).Once()

		// Execute the test
		route, err := service.GetRouteByPath(ctx, path)

		// Assertions
		require.NoError(t, err)
		assert.Equal(t, expectedRoute, route)
		mockCache.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})
}
