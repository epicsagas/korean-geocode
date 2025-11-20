package breaker

import (
	"context"
	"time"

	"github.com/epicsagas/korean-geocode/internal/infrastructure/config"
	"github.com/epicsagas/korean-geocode/pkg/domain"
	"github.com/sony/gobreaker"
)

// CircuitBreakerWrapper는 Geocoder를 Circuit Breaker로 감싸는 래퍼입니다
type CircuitBreakerWrapper struct {
	provider domain.Geocoder
	breaker  *gobreaker.CircuitBreaker
}

// NewCircuitBreakerWrapper는 새로운 Circuit Breaker Wrapper를 생성합니다
func NewCircuitBreakerWrapper(provider domain.Geocoder, cfg config.CircuitBreakerConfig) *CircuitBreakerWrapper {
	// Circuit Breaker 설정
	settings := gobreaker.Settings{
		Name:        provider.Name(),
		MaxRequests: cfg.MaxRequests,
		Interval:    10 * time.Second, // 통계 수집 간격
		Timeout:     cfg.Timeout,      // Open 상태 유지 시간 (60초)
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// 최근 요청 중 실패율이 임계값 이상이면 Circuit Open
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= uint32(cfg.FailureThreshold) &&
				failureRatio >= 0.5 // 50% 이상 실패
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			// 상태 변경 로깅 (production에서는 proper logger 사용)
			// fmt.Printf("[CircuitBreaker] %s: %s -> %s\n", name, from, to)
		},
	}

	return &CircuitBreakerWrapper{
		provider: provider,
		breaker:  gobreaker.NewCircuitBreaker(settings),
	}
}

// Name은 공급자 이름을 반환합니다
func (c *CircuitBreakerWrapper) Name() string {
	return c.provider.Name()
}

// Geocode는 Circuit Breaker를 통해 Geocoding을 수행합니다
func (c *CircuitBreakerWrapper) Geocode(ctx context.Context, query string) (*domain.GeoResult, error) {
	// Circuit Breaker Execute
	result, err := c.breaker.Execute(func() (interface{}, error) {
		return c.provider.Geocode(ctx, query)
	})

	if err != nil {
		// Circuit Breaker Open 상태이거나 요청 실패
		if err == gobreaker.ErrOpenState {
			return nil, domain.ErrProviderUnavailable
		}
		return nil, err
	}

	// 타입 변환
	geoResult, ok := result.(*domain.GeoResult)
	if !ok {
		return nil, domain.ErrProviderUnavailable
	}

	return geoResult, nil
}

// State는 현재 Circuit Breaker 상태를 반환합니다 (디버깅용)
func (c *CircuitBreakerWrapper) State() gobreaker.State {
	return c.breaker.State()
}
