package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/epicsagas/korean-geocode/pkg/domain"
	"github.com/epicsagas/korean-geocode/pkg/router"
	"github.com/stretchr/testify/assert"
)

// MockRouter는 테스트용 SmartRouter 목업입니다
type MockRouter struct {
	geocodeFunc func(ctx context.Context, query string) (*router.GeoRouteResult, error)
}

func (m *MockRouter) Geocode(ctx context.Context, query string) (*router.GeoRouteResult, error) {
	if m.geocodeFunc != nil {
		return m.geocodeFunc(ctx, query)
	}
	return nil, errors.New("not implemented")
}

func (m *MockRouter) GetProviderStatus() map[string]string {
	return map[string]string{
		"test": "available",
	}
}

func TestGeocodeHandler_Success(t *testing.T) {
	mockRouter := &MockRouter{
		geocodeFunc: func(ctx context.Context, query string) (*router.GeoRouteResult, error) {
			return &router.GeoRouteResult{
				Data: &domain.GeoResult{
					Provider:  "kakao",
					Latitude:  37.498095,
					Longitude: 127.027610,
					Address:   "서울 강남구 테헤란로 152",
					CRS:       "WGS84",
				},
				FallbackHistory: []string{"kakao_success"},
			}, nil
		},
	}

	handler := NewGeocodeHandler(mockRouter)

	req := httptest.NewRequest(http.MethodGet, "/v1/geocode?q=서울+강남구+테헤란로+152", nil)
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
	assert.Contains(t, w.Body.String(), "kakao")
}

func TestGeocodeHandler_MissingParameter(t *testing.T) {
	handler := NewGeocodeHandler(&MockRouter{})

	req := httptest.NewRequest(http.MethodGet, "/v1/geocode", nil)
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "MISSING_PARAMETER")
}

func TestGeocodeHandler_MethodNotAllowed(t *testing.T) {
	handler := NewGeocodeHandler(&MockRouter{})

	req := httptest.NewRequest(http.MethodPost, "/v1/geocode?q=test", nil)
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestGeocodeHandler_GeocodingError(t *testing.T) {
	mockRouter := &MockRouter{
		geocodeFunc: func(ctx context.Context, query string) (*router.GeoRouteResult, error) {
			return nil, domain.ErrNoResultsFound
		},
	}

	handler := NewGeocodeHandler(mockRouter)

	req := httptest.NewRequest(http.MethodGet, "/v1/geocode?q=invalid", nil)
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "NO_RESULTS")
}

func TestGeocodeHandler_ProviderTimeout(t *testing.T) {
	mockRouter := &MockRouter{
		geocodeFunc: func(ctx context.Context, query string) (*router.GeoRouteResult, error) {
			return nil, domain.ErrProviderTimeout
		},
	}

	handler := NewGeocodeHandler(mockRouter)

	req := httptest.NewRequest(http.MethodGet, "/v1/geocode?q=test", nil)
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	assert.Equal(t, http.StatusGatewayTimeout, w.Code)
	assert.Contains(t, w.Body.String(), "PROVIDER_TIMEOUT")
}

func TestGeocodeHandler_InvalidAddress(t *testing.T) {
	mockRouter := &MockRouter{
		geocodeFunc: func(ctx context.Context, query string) (*router.GeoRouteResult, error) {
			return nil, domain.ErrInvalidAddress
		},
	}

	handler := NewGeocodeHandler(mockRouter)

	req := httptest.NewRequest(http.MethodGet, "/v1/geocode?q=invalid", nil)
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_ADDRESS")
}

func TestGeocodeHandler_ProviderUnavailable(t *testing.T) {
	mockRouter := &MockRouter{
		geocodeFunc: func(ctx context.Context, query string) (*router.GeoRouteResult, error) {
			return nil, domain.ErrProviderUnavailable
		},
	}

	handler := NewGeocodeHandler(mockRouter)

	req := httptest.NewRequest(http.MethodGet, "/v1/geocode?q=test", nil)
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "PROVIDER_UNAVAILABLE")
}

func TestGeocodeHandler_QuotaExceeded(t *testing.T) {
	mockRouter := &MockRouter{
		geocodeFunc: func(ctx context.Context, query string) (*router.GeoRouteResult, error) {
			return nil, domain.ErrQuotaExceeded
		},
	}

	handler := NewGeocodeHandler(mockRouter)

	req := httptest.NewRequest(http.MethodGet, "/v1/geocode?q=test", nil)
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "QUOTA_EXCEEDED")
}

func TestGeocodeHandler_GenericError(t *testing.T) {
	mockRouter := &MockRouter{
		geocodeFunc: func(ctx context.Context, query string) (*router.GeoRouteResult, error) {
			return nil, errors.New("unknown error")
		},
	}

	handler := NewGeocodeHandler(mockRouter)

	req := httptest.NewRequest(http.MethodGet, "/v1/geocode?q=test", nil)
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "GEOCODING_FAILED")
}

func TestGeocodeHandler_SuccessWithMetadata(t *testing.T) {
	mockRouter := &MockRouter{
		geocodeFunc: func(ctx context.Context, query string) (*router.GeoRouteResult, error) {
			return &router.GeoRouteResult{
				Data: &domain.GeoResult{
					Provider:  "google",
					Latitude:  37.498095,
					Longitude: 127.027610,
					Address:   "서울 강남구 테헤란로 152",
					CRS:       "WGS84",
				},
				FallbackHistory: []string{"vworld_failed", "kakao_failed", "google_success"},
			}, nil
		},
	}

	handler := NewGeocodeHandler(mockRouter)

	req := httptest.NewRequest(http.MethodGet, "/v1/geocode?q=서울+강남구+테헤란로+152", nil)
	w := httptest.NewRecorder()

	handler.Handle(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
	assert.Contains(t, w.Body.String(), "google")
	assert.Contains(t, w.Body.String(), "fallback_history")
	assert.Contains(t, w.Body.String(), "vworld_failed")
	assert.Contains(t, w.Body.String(), "kakao_failed")
	assert.Contains(t, w.Body.String(), "google_success")
	assert.Contains(t, w.Body.String(), "query")
}
