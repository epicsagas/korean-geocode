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

func TestNaverProvider_Name(t *testing.T) {
	provider := NewNaverProvider("test-client-id", "test-client-secret")
	assert.Equal(t, "naver", provider.Name())
}

func TestNaverProvider_Geocode_Success(t *testing.T) {
	// Mock Naver API response
	mockResponse := NaverGeocodeResponse{
		Status: "OK",
		Meta: struct {
			TotalCount int `json:"totalCount"`
			Page       int `json:"page"`
			Count      int `json:"count"`
		}{
			TotalCount: 1,
			Page:       1,
			Count:      1,
		},
		Addresses: []struct {
			RoadAddress    string  `json:"roadAddress"`
			JibunAddress   string  `json:"jibunAddress"`
			EnglishAddress string  `json:"englishAddress"`
			X              string  `json:"x"`
			Y              string  `json:"y"`
			Distance       float64 `json:"distance"`
		}{
			{
				RoadAddress:  "경기도 성남시 분당구 불정로 6",
				JibunAddress: "경기도 성남시 분당구 정자동 178-1",
				X:            "127.1054328",
				Y:            "37.3595316",
				Distance:     0.0,
			},
		},
	}

	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		assert.Equal(t, "test-client-id", r.Header.Get("x-ncp-apigw-api-key-id"))
		assert.Equal(t, "test-client-secret", r.Header.Get("x-ncp-apigw-api-key"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	// Create provider with mock server URL
	provider := NewNaverProvider("test-client-id", "test-client-secret")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "경기도 성남시 분당구 불정로 6")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "naver", result.Provider)
	assert.Equal(t, 37.3595316, result.Latitude)
	assert.Equal(t, 127.1054328, result.Longitude)
	assert.Equal(t, "경기도 성남시 분당구 불정로 6", result.Address)
	assert.Equal(t, "WGS84", result.CRS)
}

func TestNaverProvider_Geocode_ZeroResults(t *testing.T) {
	mockResponse := NaverGeocodeResponse{
		Status: "OK",
		Meta: struct {
			TotalCount int `json:"totalCount"`
			Page       int `json:"page"`
			Count      int `json:"count"`
		}{
			TotalCount: 0,
			Page:       1,
			Count:      0,
		},
		Addresses: []struct {
			RoadAddress    string  `json:"roadAddress"`
			JibunAddress   string  `json:"jibunAddress"`
			EnglishAddress string  `json:"englishAddress"`
			X              string  `json:"x"`
			Y              string  `json:"y"`
			Distance       float64 `json:"distance"`
		}{},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	provider := NewNaverProvider("test-client-id", "test-client-secret")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "nonexistent address")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, domain.ErrNoResultsFound, err)
}

func TestNaverProvider_Geocode_Timeout(t *testing.T) {
	// Create slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	provider := NewNaverProvider("test-client-id", "test-client-secret")
	provider.baseURL = server.URL

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := provider.Geocode(ctx, "test query")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, domain.ErrProviderTimeout, err)
}

func TestNaverProvider_Geocode_QuotaExceeded(t *testing.T) {
	// Mock 429 Too Many Requests response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"errorMessage": "quota exceeded"}`))
	}))
	defer server.Close()

	provider := NewNaverProvider("test-client-id", "test-client-secret")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "test query")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, domain.ErrQuotaExceeded, err)
}

func TestNaverProvider_Geocode_ServerError(t *testing.T) {
	// Mock 500 Internal Server Error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	provider := NewNaverProvider("test-client-id", "test-client-secret")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "test query")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unexpected status code 500")
}

func TestNaverProvider_Geocode_InvalidCoordinates(t *testing.T) {
	// Mock response with invalid coordinates
	mockResponse := NaverGeocodeResponse{
		Status: "OK",
		Addresses: []struct {
			RoadAddress    string  `json:"roadAddress"`
			JibunAddress   string  `json:"jibunAddress"`
			EnglishAddress string  `json:"englishAddress"`
			X              string  `json:"x"`
			Y              string  `json:"y"`
			Distance       float64 `json:"distance"`
		}{
			{
				RoadAddress: "Invalid Location",
				X:           "invalid",
				Y:           "invalid",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	provider := NewNaverProvider("test-client-id", "test-client-secret")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "test query")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestNaverProvider_Geocode_JibunAddressFallback(t *testing.T) {
	// Mock response with only jibun address (no road address)
	mockResponse := NaverGeocodeResponse{
		Status: "OK",
		Addresses: []struct {
			RoadAddress    string  `json:"roadAddress"`
			JibunAddress   string  `json:"jibunAddress"`
			EnglishAddress string  `json:"englishAddress"`
			X              string  `json:"x"`
			Y              string  `json:"y"`
			Distance       float64 `json:"distance"`
		}{
			{
				RoadAddress:  "", // Empty road address
				JibunAddress: "경기도 성남시 분당구 정자동 178-1",
				X:            "127.1054328",
				Y:            "37.3595316",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	provider := NewNaverProvider("test-client-id", "test-client-secret")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "정자동 178-1")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "경기도 성남시 분당구 정자동 178-1", result.Address)
}

func TestNaverProvider_Geocode_APIError(t *testing.T) {
	// Mock response with API error message
	mockResponse := NaverGeocodeResponse{
		Status:       "ERROR",
		ErrorMessage: "Invalid API key",
		Addresses: []struct {
			RoadAddress    string  `json:"roadAddress"`
			JibunAddress   string  `json:"jibunAddress"`
			EnglishAddress string  `json:"englishAddress"`
			X              string  `json:"x"`
			Y              string  `json:"y"`
			Distance       float64 `json:"distance"`
		}{},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	provider := NewNaverProvider("test-client-id", "test-client-secret")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "test query")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Invalid API key")
}

func TestNaverProvider_BuildURL(t *testing.T) {
	provider := NewNaverProvider("test-client-id", "test-client-secret")
	url := provider.buildURL("경기도 성남시 분당구 불정로 6")

	assert.Contains(t, url, "naveropenapi.apigw.ntruss.com")
	assert.Contains(t, url, "query=")
}
