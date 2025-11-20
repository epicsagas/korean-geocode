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

func TestVWorldProvider_Name(t *testing.T) {
	provider := NewVWorldProvider("test-api-key")
	assert.Equal(t, "vworld", provider.Name())
}

func TestVWorldProvider_Geocode_Success(t *testing.T) {
	// Mock vWorld API response
	mockResponse := VWorldGeocodeResponse{
		Response: struct {
			Status string `json:"status"`
			Result *struct {
				CRS   string `json:"crs"`
				Type  string `json:"type"`
				Point *struct {
					X string `json:"x"`
					Y string `json:"y"`
				} `json:"point"`
				Items []struct {
					Address string `json:"address"`
					Point   struct {
						X string `json:"x"`
						Y string `json:"y"`
					} `json:"point"`
				} `json:"items"`
			} `json:"result"`
		}{
			Status: "OK",
			Result: &struct {
				CRS   string `json:"crs"`
				Type  string `json:"type"`
				Point *struct {
					X string `json:"x"`
					Y string `json:"y"`
				} `json:"point"`
				Items []struct {
					Address string `json:"address"`
					Point   struct {
						X string `json:"x"`
						Y string `json:"y"`
					} `json:"point"`
				} `json:"items"`
			}{
				CRS:  "EPSG:4326",
				Type: "address",
				Items: []struct {
					Address string `json:"address"`
					Point   struct {
						X string `json:"x"`
						Y string `json:"y"`
					} `json:"point"`
				}{
					{
						Address: "서울특별시 강남구 테헤란로 152",
						Point: struct {
							X string `json:"x"`
							Y string `json:"y"`
						}{
							X: "127.027610",
							Y: "37.498095",
						},
					},
				},
			},
		},
	}

	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	// Create provider with mock server URL
	provider := NewVWorldProvider("test-api-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "서울특별시 강남구 테헤란로 152")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "vworld", result.Provider)
	assert.Equal(t, 37.498095, result.Latitude)
	assert.Equal(t, 127.027610, result.Longitude)
	assert.Equal(t, "서울특별시 강남구 테헤란로 152", result.Address)
	assert.Equal(t, "WGS84", result.CRS)
}

func TestVWorldProvider_Geocode_ZeroResults(t *testing.T) {
	mockResponse := VWorldGeocodeResponse{
		Response: struct {
			Status string `json:"status"`
			Result *struct {
				CRS   string `json:"crs"`
				Type  string `json:"type"`
				Point *struct {
					X string `json:"x"`
					Y string `json:"y"`
				} `json:"point"`
				Items []struct {
					Address string `json:"address"`
					Point   struct {
						X string `json:"x"`
						Y string `json:"y"`
					} `json:"point"`
				} `json:"items"`
			} `json:"result"`
		}{
			Status: "NOT_FOUND",
			Result: &struct {
				CRS   string `json:"crs"`
				Type  string `json:"type"`
				Point *struct {
					X string `json:"x"`
					Y string `json:"y"`
				} `json:"point"`
				Items []struct {
					Address string `json:"address"`
					Point   struct {
						X string `json:"x"`
						Y string `json:"y"`
					} `json:"point"`
				} `json:"items"`
			}{
				Items: []struct {
					Address string `json:"address"`
					Point   struct {
						X string `json:"x"`
						Y string `json:"y"`
					} `json:"point"`
				}{},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	provider := NewVWorldProvider("test-api-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "nonexistent address")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, domain.ErrNoResultsFound, err)
}

func TestVWorldProvider_Geocode_Timeout(t *testing.T) {
	// Create slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	provider := NewVWorldProvider("test-api-key")
	provider.baseURL = server.URL

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := provider.Geocode(ctx, "test query")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, domain.ErrProviderTimeout, err)
}

func TestVWorldProvider_Geocode_InvalidCoordinates(t *testing.T) {
	// Mock response with invalid coordinates
	mockResponse := VWorldGeocodeResponse{
		Response: struct {
			Status string `json:"status"`
			Result *struct {
				CRS   string `json:"crs"`
				Type  string `json:"type"`
				Point *struct {
					X string `json:"x"`
					Y string `json:"y"`
				} `json:"point"`
				Items []struct {
					Address string `json:"address"`
					Point   struct {
						X string `json:"x"`
						Y string `json:"y"`
					} `json:"point"`
				} `json:"items"`
			} `json:"result"`
		}{
			Status: "OK",
			Result: &struct {
				CRS   string `json:"crs"`
				Type  string `json:"type"`
				Point *struct {
					X string `json:"x"`
					Y string `json:"y"`
				} `json:"point"`
				Items []struct {
					Address string `json:"address"`
					Point   struct {
						X string `json:"x"`
						Y string `json:"y"`
					} `json:"point"`
				} `json:"items"`
			}{
				Items: []struct {
					Address string `json:"address"`
					Point   struct {
						X string `json:"x"`
						Y string `json:"y"`
					} `json:"point"`
				}{
					{
						Address: "Invalid Location",
						Point: struct {
							X string `json:"x"`
							Y string `json:"y"`
						}{
							X: "invalid",
							Y: "invalid",
						},
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	provider := NewVWorldProvider("test-api-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "test query")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestVWorldProvider_BuildURL(t *testing.T) {
	provider := NewVWorldProvider("my-secret-key")
	url := provider.buildURL("서울 강남구")

	assert.Contains(t, url, "api.vworld.kr")
	assert.Contains(t, url, "address=")
	assert.Contains(t, url, "key=my-secret-key")
}

func TestVWorldProvider_Geocode_ServerError(t *testing.T) {
	// Mock 500 Internal Server Error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	provider := NewVWorldProvider("test-api-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "test query")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unexpected status code 500")
}

func TestVWorldProvider_DetermineAddressType(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected string // "ROAD" or "PARCEL"
	}{
		{
			name:     "Road address with 로",
			address:  "서울 강남구 테헤란로 152",
			expected: "ROAD",
		},
		{
			name:     "Road address with 길",
			address:  "서울 종로구 사직로 161",
			expected: "ROAD",
		},
		{
			name:     "Parcel address with 동",
			address:  "서울 강남구 역삼동 123",
			expected: "PARCEL",
		},
		{
			name:     "Parcel address with 리",
			address:  "경기도 양평군 양평읍 양평리 123",
			expected: "PARCEL",
		},
	}

	provider := NewVWorldProvider("test-key")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This would test the internal determineAddressType method
			// Since it's a private method, we test it indirectly through buildURL
			url := provider.buildURL(tt.address)

			if tt.expected == "ROAD" {
				assert.Contains(t, url, "type=ROAD", "Expected ROAD type in URL")
			} else {
				assert.Contains(t, url, "type=PARCEL", "Expected PARCEL type in URL")
			}
		})
	}
}

func TestVWorldProvider_Geocode_QuotaExceeded(t *testing.T) {
	// Mock 429 Too Many Requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	provider := NewVWorldProvider("test-api-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "test query")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, domain.ErrQuotaExceeded, err)
}

func TestVWorldProvider_Geocode_AuthError(t *testing.T) {
	mockResponse := VWorldGeocodeResponse{
		Response: struct {
			Status string `json:"status"`
			Result *struct {
				CRS   string `json:"crs"`
				Type  string `json:"type"`
				Point *struct {
					X string `json:"x"`
					Y string `json:"y"`
				} `json:"point"`
				Items []struct {
					Address string `json:"address"`
					Point   struct {
						X string `json:"x"`
						Y string `json:"y"`
					} `json:"point"`
				} `json:"items"`
			} `json:"result"`
		}{
			Status: "AUTH_ERROR",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	provider := NewVWorldProvider("invalid-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "test query")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "authentication failed")
}

func TestVWorldProvider_Geocode_AccessDenied(t *testing.T) {
	mockResponse := VWorldGeocodeResponse{
		Response: struct {
			Status string `json:"status"`
			Result *struct {
				CRS   string `json:"crs"`
				Type  string `json:"type"`
				Point *struct {
					X string `json:"x"`
					Y string `json:"y"`
				} `json:"point"`
				Items []struct {
					Address string `json:"address"`
					Point   struct {
						X string `json:"x"`
						Y string `json:"y"`
					} `json:"point"`
				} `json:"items"`
			} `json:"result"`
		}{
			Status: "ACCESS_DENIED",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	provider := NewVWorldProvider("test-api-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "test query")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "access denied")
}

func TestVWorldProvider_Geocode_InternalError(t *testing.T) {
	mockResponse := VWorldGeocodeResponse{
		Response: struct {
			Status string `json:"status"`
			Result *struct {
				CRS   string `json:"crs"`
				Type  string `json:"type"`
				Point *struct {
					X string `json:"x"`
					Y string `json:"y"`
				} `json:"point"`
				Items []struct {
					Address string `json:"address"`
					Point   struct {
						X string `json:"x"`
						Y string `json:"y"`
					} `json:"point"`
				} `json:"items"`
			} `json:"result"`
		}{
			Status: "ERROR",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	provider := NewVWorldProvider("test-api-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "test query")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "internal error")
}

func TestVWorldProvider_Geocode_UnknownStatus(t *testing.T) {
	mockResponse := VWorldGeocodeResponse{
		Response: struct {
			Status string `json:"status"`
			Result *struct {
				CRS   string `json:"crs"`
				Type  string `json:"type"`
				Point *struct {
					X string `json:"x"`
					Y string `json:"y"`
				} `json:"point"`
				Items []struct {
					Address string `json:"address"`
					Point   struct {
						X string `json:"x"`
						Y string `json:"y"`
					} `json:"point"`
				} `json:"items"`
			} `json:"result"`
		}{
			Status: "UNKNOWN_ERROR_CODE",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	provider := NewVWorldProvider("test-api-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "test query")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "UNKNOWN_ERROR_CODE")
}

func TestVWorldProvider_Geocode_PointOnlyResponse(t *testing.T) {
	// Test when response has Point but no Items
	mockResponse := VWorldGeocodeResponse{
		Response: struct {
			Status string `json:"status"`
			Result *struct {
				CRS   string `json:"crs"`
				Type  string `json:"type"`
				Point *struct {
					X string `json:"x"`
					Y string `json:"y"`
				} `json:"point"`
				Items []struct {
					Address string `json:"address"`
					Point   struct {
						X string `json:"x"`
						Y string `json:"y"`
					} `json:"point"`
				} `json:"items"`
			} `json:"result"`
		}{
			Status: "OK",
			Result: &struct {
				CRS   string `json:"crs"`
				Type  string `json:"type"`
				Point *struct {
					X string `json:"x"`
					Y string `json:"y"`
				} `json:"point"`
				Items []struct {
					Address string `json:"address"`
					Point   struct {
						X string `json:"x"`
						Y string `json:"y"`
					} `json:"point"`
				} `json:"items"`
			}{
				CRS:  "EPSG:4326",
				Type: "address",
				Point: &struct {
					X string `json:"x"`
					Y string `json:"y"`
				}{
					X: "127.027610",
					Y: "37.498095",
				},
				Items: []struct {
					Address string `json:"address"`
					Point   struct {
						X string `json:"x"`
						Y string `json:"y"`
					} `json:"point"`
				}{}, // Empty items
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	provider := NewVWorldProvider("test-api-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "서울 강남구")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "vworld", result.Provider)
	assert.Equal(t, 37.498095, result.Latitude)
	assert.Equal(t, 127.027610, result.Longitude)
	assert.Equal(t, "서울 강남구", result.Address) // Should use query as address
	assert.Equal(t, "WGS84", result.CRS)
}

func TestVWorldProvider_Geocode_NullResult(t *testing.T) {
	mockResponse := VWorldGeocodeResponse{
		Response: struct {
			Status string `json:"status"`
			Result *struct {
				CRS   string `json:"crs"`
				Type  string `json:"type"`
				Point *struct {
					X string `json:"x"`
					Y string `json:"y"`
				} `json:"point"`
				Items []struct {
					Address string `json:"address"`
					Point   struct {
						X string `json:"x"`
						Y string `json:"y"`
					} `json:"point"`
				} `json:"items"`
			} `json:"result"`
		}{
			Status: "OK",
			Result: nil, // Null result
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	provider := NewVWorldProvider("test-api-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "invalid address")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, domain.ErrNoResultsFound, err)
}

func TestVWorldProvider_Geocode_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json {{{"))
	}))
	defer server.Close()

	provider := NewVWorldProvider("test-api-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "test query")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestVWorldProvider_Geocode_NoPointNoItems(t *testing.T) {
	// Test when response has neither Point nor Items
	mockResponse := VWorldGeocodeResponse{
		Response: struct {
			Status string `json:"status"`
			Result *struct {
				CRS   string `json:"crs"`
				Type  string `json:"type"`
				Point *struct {
					X string `json:"x"`
					Y string `json:"y"`
				} `json:"point"`
				Items []struct {
					Address string `json:"address"`
					Point   struct {
						X string `json:"x"`
						Y string `json:"y"`
					} `json:"point"`
				} `json:"items"`
			} `json:"result"`
		}{
			Status: "OK",
			Result: &struct {
				CRS   string `json:"crs"`
				Type  string `json:"type"`
				Point *struct {
					X string `json:"x"`
					Y string `json:"y"`
				} `json:"point"`
				Items []struct {
					Address string `json:"address"`
					Point   struct {
						X string `json:"x"`
						Y string `json:"y"`
					} `json:"point"`
				} `json:"items"`
			}{
				Point: nil,
				Items: []struct {
					Address string `json:"address"`
					Point   struct {
						X string `json:"x"`
						Y string `json:"y"`
					} `json:"point"`
				}{},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	provider := NewVWorldProvider("test-api-key")
	provider.baseURL = server.URL

	result, err := provider.Geocode(context.Background(), "invalid")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, domain.ErrNoResultsFound, err)
}
