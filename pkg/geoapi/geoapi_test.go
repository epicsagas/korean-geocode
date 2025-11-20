package geoapi

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/epicsagas/korean-geocode/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, 5*time.Second, cfg.Server.Timeout)
	assert.Equal(t, 2*time.Second, cfg.Google.Timeout)
	assert.Equal(t, 2*time.Second, cfg.Kakao.Timeout)
	assert.Equal(t, 2*time.Second, cfg.VWorld.Timeout)
	assert.Equal(t, 300000, cfg.Kakao.DailyQuota)
	assert.Equal(t, uint32(10), cfg.CircuitBreaker.MaxRequests)
	assert.Equal(t, uint32(5), cfg.CircuitBreaker.FailureThreshold)
	assert.Equal(t, 60*time.Second, cfg.CircuitBreaker.Timeout)
}

func TestNew_WithValidConfig(t *testing.T) {
	// Set environment variables
	os.Setenv("GOOGLE_MAPS_API_KEY", "test-google-key")
	os.Setenv("KAKAO_REST_API_KEY", "test-kakao-key")
	os.Setenv("VWORLD_API_KEY", "test-vworld-key")

	defer func() {
		os.Unsetenv("GOOGLE_MAPS_API_KEY")
		os.Unsetenv("KAKAO_REST_API_KEY")
		os.Unsetenv("VWORLD_API_KEY")
	}()

	api, err := New()

	assert.NoError(t, err)
	assert.NotNil(t, api)
	assert.NotNil(t, api.Router)
	assert.NotNil(t, api.Config)
	assert.NotNil(t, api.RateLimiter)
}

func TestNew_WithoutAPIKeys(t *testing.T) {
	// Clear all environment variables
	os.Clearenv()

	api, err := New()

	// Should error when no API keys configured
	assert.Error(t, err)
	assert.Nil(t, api)
	assert.Contains(t, err.Error(), "at least one API key must be configured")
}

func TestNewWithConfig_GoogleOnly(t *testing.T) {
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
	assert.NotNil(t, api.Router)
	assert.NotNil(t, api.Config)
	assert.Equal(t, cfg, api.Config)
}

func TestNewWithConfig_KakaoOnly(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:    "8080",
			Timeout: 5 * time.Second,
		},
		Google: config.ProviderConfig{
			Timeout: 2 * time.Second,
		},
		Kakao: config.ProviderConfig{
			APIKey:     "test-kakao-key",
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
	assert.NotNil(t, api.RateLimiter)

	// Verify rate limiter can track Kakao usage
	usage, err := api.RateLimiter.GetUsage(context.Background(), "kakao")
	assert.NoError(t, err)
	assert.Equal(t, 0, usage) // Initial usage should be 0
}

func TestNewWithConfig_VWorldOnly(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:    "8080",
			Timeout: 5 * time.Second,
		},
		Google: config.ProviderConfig{
			Timeout: 2 * time.Second,
		},
		Kakao: config.ProviderConfig{
			Timeout:    2 * time.Second,
			DailyQuota: 300000,
		},
		VWorld: config.ProviderConfig{
			APIKey:  "test-vworld-key",
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
	assert.NotNil(t, api.Router)
}

func TestNewWithConfig_AllProviders(t *testing.T) {
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
			APIKey:     "test-kakao-key",
			Timeout:    2 * time.Second,
			DailyQuota: 300000,
		},
		VWorld: config.ProviderConfig{
			APIKey:  "test-vworld-key",
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
	assert.NotNil(t, api.Router)
	assert.NotNil(t, api.Config)
	assert.NotNil(t, api.RateLimiter)

	// Verify all providers are registered
	status := api.Router.GetProviderStatus()
	assert.NotEmpty(t, status)
}

func TestNewWithConfig_NoProviders(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:    "8080",
			Timeout: 5 * time.Second,
		},
		Google: config.ProviderConfig{
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
	assert.Nil(t, api) // Should return nil when no providers configured
}

func TestNewWithConfig_CustomCircuitBreakerSettings(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:    "9090",
			Timeout: 10 * time.Second,
		},
		Google: config.ProviderConfig{
			APIKey:  "test-google-key",
			Timeout: 3 * time.Second,
		},
		Kakao: config.ProviderConfig{
			Timeout:    2 * time.Second,
			DailyQuota: 300000,
		},
		VWorld: config.ProviderConfig{
			Timeout: 2 * time.Second,
		},
		CircuitBreaker: config.CircuitBreakerConfig{
			MaxRequests:      20,
			FailureThreshold: 10,
			Timeout:          120 * time.Second,
		},
	}

	api, err := NewWithConfig(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, api)
	assert.Equal(t, uint32(20), api.Config.CircuitBreaker.MaxRequests)
	assert.Equal(t, uint32(10), api.Config.CircuitBreaker.FailureThreshold)
	assert.Equal(t, 120*time.Second, api.Config.CircuitBreaker.Timeout)
}

func TestNewWithConfig_CustomKakaoQuota(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:    "8080",
			Timeout: 5 * time.Second,
		},
		Google: config.ProviderConfig{
			Timeout: 2 * time.Second,
		},
		Kakao: config.ProviderConfig{
			APIKey:     "test-kakao-key",
			Timeout:    2 * time.Second,
			DailyQuota: 500000, // Custom quota
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

	// Verify rate limiter is working
	usage, err := api.RateLimiter.GetUsage(context.Background(), "kakao")
	assert.NoError(t, err)
	assert.Equal(t, 0, usage) // Initial usage should be 0
}

func TestGeoAPI_Geocode(t *testing.T) {
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

	// Verify Geocode method exists and works with proper context
	// Use context with timeout to prevent hanging on real API call
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := api.Router.Geocode(ctx, "서울특별시 강남구")

	// Should get an error since we're using a fake API key or timeout
	assert.Error(t, err)
	assert.Nil(t, result)
}
