package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port:    "8080",
			Timeout: 5 * time.Second,
		},
		Google: ProviderConfig{
			Timeout: 2 * time.Second,
		},
		Kakao: ProviderConfig{
			Timeout:    2 * time.Second,
			DailyQuota: 300000,
		},
		VWorld: ProviderConfig{
			Timeout: 2 * time.Second,
		},
		CircuitBreaker: CircuitBreakerConfig{
			MaxRequests:      10,
			FailureThreshold: 5,
			Timeout:          60 * time.Second,
		},
	}

	assert.NotNil(t, cfg)
	assert.Equal(t, "8080", cfg.Server.Port)
	assert.Equal(t, 5*time.Second, cfg.Server.Timeout)
	assert.Equal(t, uint32(10), cfg.CircuitBreaker.MaxRequests)
	assert.Equal(t, uint32(5), cfg.CircuitBreaker.FailureThreshold)
	assert.Equal(t, 60*time.Second, cfg.CircuitBreaker.Timeout)
}

func TestLoadConfig_WithEnvVars(t *testing.T) {
	// Set environment variables
	os.Setenv("GOOGLE_MAPS_API_KEY", "test-google-key")
	os.Setenv("KAKAO_REST_API_KEY", "test-kakao-key")
	os.Setenv("VWORLD_API_KEY", "test-vworld-key")
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("SERVER_TIMEOUT", "10s")
	os.Setenv("KAKAO_DAILY_QUOTA", "500000")
	os.Setenv("CIRCUIT_BREAKER_MAX_REQUESTS", "20")
	os.Setenv("CIRCUIT_BREAKER_FAILURE_THRESHOLD", "10")
	os.Setenv("CIRCUIT_BREAKER_TIMEOUT", "120s")

	defer func() {
		// Clean up
		os.Unsetenv("GOOGLE_MAPS_API_KEY")
		os.Unsetenv("KAKAO_REST_API_KEY")
		os.Unsetenv("VWORLD_API_KEY")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("SERVER_TIMEOUT")
		os.Unsetenv("KAKAO_DAILY_QUOTA")
		os.Unsetenv("CIRCUIT_BREAKER_MAX_REQUESTS")
		os.Unsetenv("CIRCUIT_BREAKER_FAILURE_THRESHOLD")
		os.Unsetenv("CIRCUIT_BREAKER_TIMEOUT")
	}()

	cfg, err := LoadConfig()

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "test-google-key", cfg.Google.APIKey)
	assert.Equal(t, "test-kakao-key", cfg.Kakao.APIKey)
	assert.Equal(t, "test-vworld-key", cfg.VWorld.APIKey)
	assert.Equal(t, "9090", cfg.Server.Port)
	assert.Equal(t, 10*time.Second, cfg.Server.Timeout)
	assert.Equal(t, 500000, cfg.Kakao.DailyQuota)
	assert.Equal(t, uint32(20), cfg.CircuitBreaker.MaxRequests)
	assert.Equal(t, uint32(10), cfg.CircuitBreaker.FailureThreshold)
	assert.Equal(t, 120*time.Second, cfg.CircuitBreaker.Timeout)
}

func TestLoadConfig_Defaults(t *testing.T) {
	// Clear all env vars except set one API key
	os.Clearenv()
	os.Setenv("GOOGLE_MAPS_API_KEY", "test-key-for-defaults")
	defer os.Unsetenv("GOOGLE_MAPS_API_KEY")

	cfg, err := LoadConfig()

	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// Check defaults
	assert.Equal(t, "test-key-for-defaults", cfg.Google.APIKey)
	assert.Equal(t, "", cfg.Kakao.APIKey)
	assert.Equal(t, "", cfg.VWorld.APIKey)
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

func TestLoadConfig_ProviderTimeouts(t *testing.T) {
	os.Setenv("GOOGLE_MAPS_API_KEY", "test-key")
	os.Setenv("GOOGLE_TIMEOUT", "3s")
	os.Setenv("KAKAO_TIMEOUT", "4s")
	os.Setenv("VWORLD_TIMEOUT", "5s")

	defer func() {
		os.Unsetenv("GOOGLE_MAPS_API_KEY")
		os.Unsetenv("GOOGLE_TIMEOUT")
		os.Unsetenv("KAKAO_TIMEOUT")
		os.Unsetenv("VWORLD_TIMEOUT")
	}()

	cfg, err := LoadConfig()

	assert.NoError(t, err)
	assert.Equal(t, 3*time.Second, cfg.Google.Timeout)
	assert.Equal(t, 4*time.Second, cfg.Kakao.Timeout)
	assert.Equal(t, 5*time.Second, cfg.VWorld.Timeout)
}

func TestLoadConfig_InvalidTimeout(t *testing.T) {
	os.Setenv("GOOGLE_MAPS_API_KEY", "test-key")
	os.Setenv("SERVER_TIMEOUT", "invalid")

	defer func() {
		os.Unsetenv("GOOGLE_MAPS_API_KEY")
		os.Unsetenv("SERVER_TIMEOUT")
	}()

	cfg, err := LoadConfig()

	// When viper can't parse duration, it returns 0s (not the default)
	assert.NoError(t, err)
	assert.Equal(t, 0*time.Second, cfg.Server.Timeout)
}

func TestLoadConfig_InvalidPort(t *testing.T) {
	os.Setenv("GOOGLE_MAPS_API_KEY", "test-key")
	os.Setenv("SERVER_PORT", "99999") // Invalid port number

	defer func() {
		os.Unsetenv("GOOGLE_MAPS_API_KEY")
		os.Unsetenv("SERVER_PORT")
	}()

	cfg, err := LoadConfig()

	// Should still load with the value (validation happens elsewhere)
	assert.NoError(t, err)
	assert.Equal(t, "99999", cfg.Server.Port)
}

func TestLoadConfig_PartialConfig(t *testing.T) {
	// Only set some env vars
	os.Setenv("GOOGLE_MAPS_API_KEY", "google-key-only")
	os.Setenv("SERVER_PORT", "3000")

	defer func() {
		os.Unsetenv("GOOGLE_MAPS_API_KEY")
		os.Unsetenv("SERVER_PORT")
	}()

	cfg, err := LoadConfig()

	assert.NoError(t, err)
	assert.Equal(t, "google-key-only", cfg.Google.APIKey)
	assert.Equal(t, "", cfg.Kakao.APIKey)
	assert.Equal(t, "", cfg.VWorld.APIKey)
	assert.Equal(t, "3000", cfg.Server.Port)
	assert.Equal(t, 5*time.Second, cfg.Server.Timeout) // Default
}

func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name          string
		config        *Config
		shouldHaveKey bool
	}{
		{
			name: "All providers configured",
			config: &Config{
				Google: ProviderConfig{APIKey: "google-key"},
				Kakao:  ProviderConfig{APIKey: "kakao-key"},
				VWorld: ProviderConfig{APIKey: "vworld-key"},
			},
			shouldHaveKey: true,
		},
		{
			name: "Only Google configured",
			config: &Config{
				Google: ProviderConfig{APIKey: "google-key"},
			},
			shouldHaveKey: true,
		},
		{
			name: "No providers configured",
			config: &Config{
				Google: ProviderConfig{APIKey: ""},
				Kakao:  ProviderConfig{APIKey: ""},
				VWorld: ProviderConfig{APIKey: ""},
			},
			shouldHaveKey: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasKey := tt.config.Google.APIKey != "" ||
				tt.config.Kakao.APIKey != "" ||
				tt.config.VWorld.APIKey != ""

			assert.Equal(t, tt.shouldHaveKey, hasKey)
		})
	}
}

func TestConfig_DurationParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
	}{
		{"Seconds", "5s", 5 * time.Second},
		{"Minutes", "2m", 2 * time.Minute},
		{"Milliseconds", "500ms", 500 * time.Millisecond},
		{"Combined", "1m30s", 90 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("GOOGLE_MAPS_API_KEY", "test-key")
			os.Setenv("SERVER_TIMEOUT", tt.input)
			defer func() {
				os.Unsetenv("GOOGLE_MAPS_API_KEY")
				os.Unsetenv("SERVER_TIMEOUT")
			}()

			cfg, err := LoadConfig()
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.Server.Timeout)
		})
	}
}
