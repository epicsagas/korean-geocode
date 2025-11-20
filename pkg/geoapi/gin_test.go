package geoapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/epicsagas/korean-geocode/internal/infrastructure/config"
	"github.com/epicsagas/korean-geocode/pkg/domain"
	"github.com/epicsagas/korean-geocode/pkg/router"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
}

// MockGeocoder for testing
type MockGeocoder struct {
	name       string
	shouldFail bool
	result     *domain.GeoResult
	err        error
}

func (m *MockGeocoder) Name() string {
	return m.name
}

func (m *MockGeocoder) Geocode(ctx context.Context, query string) (*domain.GeoResult, error) {
	if m.shouldFail {
		if m.err != nil {
			return nil, m.err
		}
		return nil, errors.New("mock error")
	}
	if m.result != nil {
		return m.result, nil
	}
	return &domain.GeoResult{
		Provider:  m.name,
		Latitude:  37.5665,
		Longitude: 126.9780,
		Address:   query,
		CRS:       "WGS84",
	}, nil
}

func setupTestAPIWithRouter() *GeoAPI {
	// Use "kakao" as provider name so SmartRouter recognizes it for Korean addresses
	mockProvider := &MockGeocoder{
		name: "kakao",
		result: &domain.GeoResult{
			Provider:  "kakao",
			Latitude:  37.5665,
			Longitude: 126.9780,
			Address:   "서울",
			CRS:       "WGS84",
		},
	}

	providers := map[string]domain.Geocoder{
		"kakao": mockProvider,
	}

	smartRouter := router.NewSmartRouter(providers, 5*time.Second)

	return &GeoAPI{
		Router: smartRouter,
		Config: DefaultConfig(),
	}
}

func TestRegisterRoutes(t *testing.T) {
	api := setupTestAPIWithRouter()
	r := gin.New()

	api.RegisterRoutes(r)

	// Verify routes are registered
	routes := r.Routes()

	hasGeocodeRoute := false
	hasHealthRoute := false

	for _, route := range routes {
		if route.Path == "/v1/geocode" && route.Method == "GET" {
			hasGeocodeRoute = true
		}
		if route.Path == "/health" && route.Method == "GET" {
			hasHealthRoute = true
		}
	}

	assert.True(t, hasGeocodeRoute, "Should register /v1/geocode route")
	assert.True(t, hasHealthRoute, "Should register /health route")
}

func TestRegisterRoutesWithPrefix(t *testing.T) {
	api := setupTestAPIWithRouter()
	r := gin.New()

	api.RegisterRoutesWithPrefix(r, "/api")

	routes := r.Routes()

	hasGeocodeRoute := false
	hasHealthRoute := false

	for _, route := range routes {
		if route.Path == "/api/v1/geocode" && route.Method == "GET" {
			hasGeocodeRoute = true
		}
		if route.Path == "/api/health" && route.Method == "GET" {
			hasHealthRoute = true
		}
	}

	assert.True(t, hasGeocodeRoute, "Should register /api/v1/geocode route")
	assert.True(t, hasHealthRoute, "Should register /api/health route")
}

func TestGeocodeHandler_Success(t *testing.T) {
	api := setupTestAPIWithRouter()
	r := gin.New()
	api.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/v1/geocode?q=서울특별시", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "\"success\":true")
	assert.Contains(t, rec.Body.String(), "37.5665")
	assert.Contains(t, rec.Body.String(), "126.978")
}

func TestGeocodeHandler_MissingParameter(t *testing.T) {
	api := setupTestAPIWithRouter()
	r := gin.New()
	api.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/v1/geocode", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "\"success\":false")
	assert.Contains(t, rec.Body.String(), "MISSING_PARAMETER")
	assert.Contains(t, rec.Body.String(), "Query parameter 'q' is required")
}

func TestGeocodeHandler_GeocodingError(t *testing.T) {
	mockProvider := &MockGeocoder{
		name:       "kakao",
		shouldFail: true,
		err:        errors.New("geocoding failed"),
	}

	providers := map[string]domain.Geocoder{
		"kakao": mockProvider,
	}

	api := &GeoAPI{
		Router: router.NewSmartRouter(providers, 5*time.Second),
		Config: DefaultConfig(),
	}

	r := gin.New()
	api.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/v1/geocode?q=invalid", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "\"success\":false")
	assert.Contains(t, rec.Body.String(), "GEOCODING_FAILED")
}

func TestGeocodeHandler_EmptyQuery(t *testing.T) {
	api := setupTestAPIWithRouter()
	r := gin.New()
	api.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/v1/geocode?q=", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "MISSING_PARAMETER")
}

func TestHealthHandler_Success(t *testing.T) {
	api := setupTestAPIWithRouter()
	r := gin.New()
	api.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "\"success\":true")
	assert.Contains(t, rec.Body.String(), "\"status\":\"healthy\"")
	assert.Contains(t, rec.Body.String(), "kakao")
}

func TestHealthHandler_WithPrefix(t *testing.T) {
	api := setupTestAPIWithRouter()
	r := gin.New()
	api.RegisterRoutesWithPrefix(r, "/api")

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "\"success\":true")
	assert.Contains(t, rec.Body.String(), "healthy")
}

func TestGeocodeHandler_WithFallback(t *testing.T) {
	mockProvider1 := &MockGeocoder{
		name:       "kakao",
		shouldFail: true,
	}
	mockProvider2 := &MockGeocoder{
		name: "google",
		result: &domain.GeoResult{
			Provider:  "google",
			Latitude:  37.5665,
			Longitude: 126.9780,
			Address:   "서울",
			CRS:       "WGS84",
		},
	}

	providers := map[string]domain.Geocoder{
		"kakao":  mockProvider1,
		"google": mockProvider2,
	}

	api := &GeoAPI{
		Router: router.NewSmartRouter(providers, 5*time.Second),
		Config: DefaultConfig(),
	}

	r := gin.New()
	api.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/v1/geocode?q=서울", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "fallback_history")
}

func TestMultipleRequests(t *testing.T) {
	api := setupTestAPIWithRouter()
	r := gin.New()
	api.RegisterRoutes(r)

	// First request
	req1 := httptest.NewRequest(http.MethodGet, "/v1/geocode?q=서울", nil)
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Second request
	req2 := httptest.NewRequest(http.MethodGet, "/v1/geocode?q=부산", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)
}

func TestGeoAPI_IntegrationWithRealConfig(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:    "8080",
			Timeout: 5 * time.Second,
		},
		Google: config.ProviderConfig{
			APIKey:  "test-google-key",
			Timeout: 2 * time.Second,
		},
		Kakao: config.ProviderConfig{
			Timeout:    2 * time.Second,
			DailyQuota: 300000,
		},
		VWorld: config.ProviderConfig{
			Timeout: 2 * time.Second,
		},
		CircuitBreaker: config.CircuitBreakerConfig{
			MaxRequests:      10,
			FailureThreshold: 5,
			Timeout:          60 * time.Second,
		},
	}

	api, err := NewWithConfig(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, api)

	r := gin.New()
	api.RegisterRoutes(r)

	// Test health endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "healthy")
}
