package breaker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/epicsagas/korean-geocode/internal/infrastructure/config"
	"github.com/epicsagas/korean-geocode/pkg/domain"
	"github.com/sony/gobreaker"
	"github.com/stretchr/testify/assert"
)

// MockGeocoder for testing
type MockGeocoder struct {
	name       string
	shouldFail bool
	failCount  int
	callCount  int
}

func (m *MockGeocoder) Name() string {
	return m.name
}

func (m *MockGeocoder) Geocode(ctx context.Context, query string) (*domain.GeoResult, error) {
	m.callCount++

	if m.shouldFail {
		m.failCount++
		return nil, errors.New("mock provider error")
	}

	return &domain.GeoResult{
		Provider:  m.name,
		Latitude:  37.5,
		Longitude: 127.0,
		Address:   query,
		CRS:       "WGS84",
	}, nil
}

func TestCircuitBreakerWrapper_Name(t *testing.T) {
	mockProvider := &MockGeocoder{name: "test-provider"}
	cfg := config.CircuitBreakerConfig{
		MaxRequests:      10,
		FailureThreshold: 5,
		Timeout:          60 * time.Second,
	}

	wrapper := NewCircuitBreakerWrapper(mockProvider, cfg)
	assert.Equal(t, "test-provider", wrapper.Name())
}

func TestCircuitBreakerWrapper_Geocode_Success(t *testing.T) {
	mockProvider := &MockGeocoder{name: "test-provider", shouldFail: false}
	cfg := config.CircuitBreakerConfig{
		MaxRequests:      10,
		FailureThreshold: 5,
		Timeout:          60 * time.Second,
	}

	wrapper := NewCircuitBreakerWrapper(mockProvider, cfg)

	result, err := wrapper.Geocode(context.Background(), "test address")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-provider", result.Provider)
	assert.Equal(t, 37.5, result.Latitude)
	assert.Equal(t, 127.0, result.Longitude)
	assert.Equal(t, gobreaker.StateClosed, wrapper.State())
}

func TestCircuitBreakerWrapper_Geocode_ProviderFailure(t *testing.T) {
	mockProvider := &MockGeocoder{name: "test-provider", shouldFail: true}
	cfg := config.CircuitBreakerConfig{
		MaxRequests:      10,
		FailureThreshold: 5,
		Timeout:          60 * time.Second,
	}

	wrapper := NewCircuitBreakerWrapper(mockProvider, cfg)

	result, err := wrapper.Geocode(context.Background(), "test address")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 1, mockProvider.failCount)
}

func TestCircuitBreakerWrapper_CircuitOpens(t *testing.T) {
	mockProvider := &MockGeocoder{name: "test-provider", shouldFail: true}
	cfg := config.CircuitBreakerConfig{
		MaxRequests:      10,
		FailureThreshold: 5,
		Timeout:          60 * time.Second,
	}

	wrapper := NewCircuitBreakerWrapper(mockProvider, cfg)

	// Generate enough failures to trip the circuit
	// Need FailureThreshold requests with 50%+ failure rate
	for i := 0; i < 10; i++ {
		wrapper.Geocode(context.Background(), "test address")
	}

	// Circuit should be open now
	assert.Equal(t, gobreaker.StateOpen, wrapper.State())

	// Next request should fail immediately without calling provider
	prevCallCount := mockProvider.callCount
	result, err := wrapper.Geocode(context.Background(), "test address")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, domain.ErrProviderUnavailable, err)
	assert.Equal(t, prevCallCount, mockProvider.callCount, "Provider should not be called when circuit is open")
}

func TestCircuitBreakerWrapper_CircuitHalfOpen(t *testing.T) {
	mockProvider := &MockGeocoder{name: "test-provider", shouldFail: true}
	cfg := config.CircuitBreakerConfig{
		MaxRequests:      1,
		FailureThreshold: 3,
		Timeout:          100 * time.Millisecond, // Short timeout for testing
	}

	wrapper := NewCircuitBreakerWrapper(mockProvider, cfg)

	// Trip the circuit
	for i := 0; i < 5; i++ {
		wrapper.Geocode(context.Background(), "test address")
	}

	assert.Equal(t, gobreaker.StateOpen, wrapper.State())

	// Wait for circuit to become half-open
	time.Sleep(150 * time.Millisecond)

	// Fix the provider
	mockProvider.shouldFail = false

	// Next request should succeed and close the circuit
	result, err := wrapper.Geocode(context.Background(), "test address")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, gobreaker.StateClosed, wrapper.State())
}

func TestCircuitBreakerWrapper_MixedSuccessFailure(t *testing.T) {
	mockProvider := &MockGeocoder{name: "test-provider", shouldFail: false}
	cfg := config.CircuitBreakerConfig{
		MaxRequests:      10,
		FailureThreshold: 5,
		Timeout:          60 * time.Second,
	}

	wrapper := NewCircuitBreakerWrapper(mockProvider, cfg)

	// Mix of successes and failures (less than 50% failure rate)
	for i := 0; i < 10; i++ {
		if i%3 == 0 {
			mockProvider.shouldFail = true
		} else {
			mockProvider.shouldFail = false
		}
		wrapper.Geocode(context.Background(), "test address")
	}

	// Circuit should remain closed (failure rate < 50%)
	assert.Equal(t, gobreaker.StateClosed, wrapper.State())
}

func TestCircuitBreakerWrapper_ContextCancellation(t *testing.T) {
	mockProvider := &MockGeocoder{name: "test-provider", shouldFail: false}
	cfg := config.CircuitBreakerConfig{
		MaxRequests:      10,
		FailureThreshold: 5,
		Timeout:          60 * time.Second,
	}

	wrapper := NewCircuitBreakerWrapper(mockProvider, cfg)

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(2 * time.Millisecond) // Ensure timeout expires

	result, err := wrapper.Geocode(ctx, "test address")

	// Circuit breaker may not catch context cancellation before executing
	// This test verifies the circuit breaker doesn't panic on cancelled context
	_ = result
	_ = err
}

func TestCircuitBreakerWrapper_RecoverFromOpen(t *testing.T) {
	mockProvider := &MockGeocoder{name: "test-provider", shouldFail: true}
	cfg := config.CircuitBreakerConfig{
		MaxRequests:      2,
		FailureThreshold: 3,
		Timeout:          100 * time.Millisecond,
	}

	wrapper := NewCircuitBreakerWrapper(mockProvider, cfg)

	// Trip the circuit
	for i := 0; i < 5; i++ {
		wrapper.Geocode(context.Background(), "test address")
	}
	assert.Equal(t, gobreaker.StateOpen, wrapper.State())

	// Wait for half-open
	time.Sleep(150 * time.Millisecond)

	// Provider still failing
	wrapper.Geocode(context.Background(), "test address")

	// Should go back to open
	assert.Equal(t, gobreaker.StateOpen, wrapper.State())

	// Wait again
	time.Sleep(150 * time.Millisecond)

	// Fix provider
	mockProvider.shouldFail = false

	// Allow MaxRequests to succeed
	for i := uint32(0); i < cfg.MaxRequests; i++ {
		result, err := wrapper.Geocode(context.Background(), "test address")
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}

	// Circuit should be closed now
	assert.Equal(t, gobreaker.StateClosed, wrapper.State())
}
