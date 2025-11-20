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

func TestGoogleProvider_Name(t *testing.T) {
	provider := NewGoogleProvider("test-api-key")
	assert.Equal(t, "google", provider.Name())
}

func TestGoogleProvider_Geocode_Success(t *testing.T) {
	// Mock Google API response
	mockResponse := GoogleGeocodeResponse{
		Results: []struct {
			FormattedAddress string `json:"formatted_address"`
			Geometry         struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		}{
			{
				FormattedAddress: "1600 Amphitheatre Parkway, Mountain View, CA 94043, USA",
				Geometry: struct {
					Location struct {
						Lat float64 `json:"lat"`
						Lng float64 `json:"lng"`
					} `json:"location"`
				}{
					Location: struct {
						Lat float64 `json:"lat"`
						Lng float64 `json:"lng"`
					}{
						Lat: 37.4224764,
						Lng: -122.0842499,
					},
				},
			},
		},
		Status: "OK",
	}

	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	// Create provider with mock server URL
	provider := NewGoogleProvider("test-api-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "1600 Amphitheatre Parkway")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "google", result.Provider)
	assert.Equal(t, 37.4224764, result.Latitude)
	assert.Equal(t, -122.0842499, result.Longitude)
	assert.Equal(t, "1600 Amphitheatre Parkway, Mountain View, CA 94043, USA", result.Address)
	assert.Equal(t, "WGS84", result.CRS)
}

func TestGoogleProvider_Geocode_ZeroResults(t *testing.T) {
	mockResponse := GoogleGeocodeResponse{
		Results: []struct {
			FormattedAddress string `json:"formatted_address"`
			Geometry         struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		}{},
		Status: "ZERO_RESULTS",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	provider := NewGoogleProvider("test-api-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "nonexistent address")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, domain.ErrNoResultsFound, err)
}

func TestGoogleProvider_Geocode_QuotaExceeded(t *testing.T) {
	mockResponse := GoogleGeocodeResponse{
		Results: []struct {
			FormattedAddress string `json:"formatted_address"`
			Geometry         struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		}{},
		Status: "OVER_QUERY_LIMIT",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	provider := NewGoogleProvider("test-api-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "test query")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, domain.ErrQuotaExceeded, err)
}

func TestGoogleProvider_Geocode_Timeout(t *testing.T) {
	// Create slow server that doesn't respond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	provider := NewGoogleProvider("test-api-key")
	provider.baseURL = server.URL

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := provider.Geocode(ctx, "test query")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, domain.ErrProviderTimeout, err)
}

func TestGoogleProvider_Geocode_InvalidCoordinates(t *testing.T) {
	// Mock response with invalid coordinates
	mockResponse := GoogleGeocodeResponse{
		Results: []struct {
			FormattedAddress string `json:"formatted_address"`
			Geometry         struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		}{
			{
				FormattedAddress: "Invalid Location",
				Geometry: struct {
					Location struct {
						Lat float64 `json:"lat"`
						Lng float64 `json:"lng"`
					} `json:"location"`
				}{
					Location: struct {
						Lat float64 `json:"lat"`
						Lng float64 `json:"lng"`
					}{
						Lat: 91.0, // Invalid latitude
						Lng: 0.0,
					},
				},
			},
		},
		Status: "OK",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	provider := NewGoogleProvider("test-api-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "test query")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, domain.ErrInvalidCoordinates, err)
}

func TestGoogleProvider_BuildURL(t *testing.T) {
	provider := NewGoogleProvider("my-secret-key")
	url := provider.buildURL("1600 Amphitheatre Parkway")

	assert.Contains(t, url, "https://maps.googleapis.com/maps/api/geocode/json")
	assert.Contains(t, url, "address=1600+Amphitheatre+Parkway")
	assert.Contains(t, url, "key=my-secret-key")
}
