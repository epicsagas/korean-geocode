package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/epicsagas/korean-geocode/pkg/domain"
	"github.com/stretchr/testify/assert"
)

func TestKakaoProvider_Name(t *testing.T) {
	provider := NewKakaoProvider("test-api-key")
	assert.Equal(t, "kakao", provider.Name())
}

func TestKakaoProvider_Geocode_Success(t *testing.T) {
	// Mock Kakao API response
	mockResponse := KakaoGeocodeResponse{
		Documents: []struct {
			Address struct {
				AddressName string `json:"address_name"`
			} `json:"address"`
			RoadAddress struct {
				AddressName string `json:"address_name"`
			} `json:"road_address"`
			X float64 `json:"x,string"`
			Y float64 `json:"y,string"`
		}{
			{
				Address: struct {
					AddressName string `json:"address_name"`
				}{
					AddressName: "서울 강남구 테헤란로 152",
				},
				RoadAddress: struct {
					AddressName string `json:"address_name"`
				}{
					AddressName: "서울 강남구 테헤란로 152",
				},
				X: 127.027610,
				Y: 37.498095,
			},
		},
		Meta: struct {
			TotalCount int `json:"total_count"`
		}{
			TotalCount: 1,
		},
	}

	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Authorization header
		auth := r.Header.Get("Authorization")
		assert.Contains(t, auth, "KakaoAK")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	// Create provider with mock server URL
	provider := NewKakaoProvider("test-api-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "서울 강남구 테헤란로 152")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "kakao", result.Provider)
	assert.Equal(t, 37.498095, result.Latitude)
	assert.Equal(t, 127.027610, result.Longitude)
	assert.Equal(t, "서울 강남구 테헤란로 152", result.Address)
	assert.Equal(t, "WGS84", result.CRS)
}

func TestKakaoProvider_Geocode_ZeroResults(t *testing.T) {
	mockResponse := KakaoGeocodeResponse{
		Documents: []struct {
			Address struct {
				AddressName string `json:"address_name"`
			} `json:"address"`
			RoadAddress struct {
				AddressName string `json:"address_name"`
			} `json:"road_address"`
			X float64 `json:"x,string"`
			Y float64 `json:"y,string"`
		}{},
		Meta: struct {
			TotalCount int `json:"total_count"`
		}{
			TotalCount: 0,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	provider := NewKakaoProvider("test-api-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "nonexistent address")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, domain.ErrNoResultsFound, err)
}

func TestKakaoProvider_Geocode_QuotaExceeded(t *testing.T) {
	// Mock 429 Too Many Requests response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "quota exceeded"}`))
	}))
	defer server.Close()

	provider := NewKakaoProvider("test-api-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "test query")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, domain.ErrQuotaExceeded, err)
}

func TestKakaoProvider_Geocode_Timeout(t *testing.T) {
	// Create slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	provider := NewKakaoProvider("test-api-key")
	provider.baseURL = server.URL

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := provider.Geocode(ctx, "test query")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, domain.ErrProviderTimeout, err)
}

func TestKakaoProvider_Geocode_InvalidCoordinates(t *testing.T) {
	// Mock response with zero coordinates (valid but unusual)
	mockResponse := KakaoGeocodeResponse{
		Documents: []struct {
			Address struct {
				AddressName string `json:"address_name"`
			} `json:"address"`
			RoadAddress struct {
				AddressName string `json:"address_name"`
			} `json:"road_address"`
			X float64 `json:"x,string"`
			Y float64 `json:"y,string"`
		}{
			{
				Address: struct {
					AddressName string `json:"address_name"`
				}{
					AddressName: "Test Location",
				},
				X: 0.0,
				Y: 0.0,
			},
		},
		Meta: struct {
			TotalCount int `json:"total_count"`
		}{
			TotalCount: 1,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	provider := NewKakaoProvider("test-api-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "test query")

	// Zero coordinates are technically valid (Gulf of Guinea)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0.0, result.Latitude)
	assert.Equal(t, 0.0, result.Longitude)
}

func TestKakaoProvider_BuildURL(t *testing.T) {
	provider := NewKakaoProvider("my-secret-key")
	url := provider.buildURL("서울 강남구")

	assert.Contains(t, url, "dapi.kakao.com")
	assert.Contains(t, url, "query=")
}

func TestKakaoProvider_Geocode_ServerError(t *testing.T) {
	// Mock 500 Internal Server Error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	provider := NewKakaoProvider("test-api-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "test query")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unexpected status code 500")
}
