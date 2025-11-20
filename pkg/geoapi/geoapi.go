package geoapi

import (
	"log"
	"time"

	"github.com/epicsagas/korean-geocode/internal/infrastructure/breaker"
	"github.com/epicsagas/korean-geocode/internal/infrastructure/config"
	"github.com/epicsagas/korean-geocode/internal/infrastructure/ratelimit"
	"github.com/epicsagas/korean-geocode/pkg/domain"
	"github.com/epicsagas/korean-geocode/pkg/provider"
	"github.com/epicsagas/korean-geocode/pkg/router"
)

// GeoAPI는 Geocoding API의 메인 인스턴스입니다
type GeoAPI struct {
	Router      *router.SmartRouter
	Config      *config.Config
	RateLimiter *ratelimit.MemoryRateLimiter
}

// New는 환경변수에서 설정을 로드하여 새로운 GeoAPI 인스턴스를 생성합니다
// .env 파일이 있으면 자동으로 로드됩니다
func New() (*GeoAPI, error) {
	// 설정 로드
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}

	return NewWithConfig(cfg)
}

// NewWithConfig는 제공된 설정으로 새로운 GeoAPI 인스턴스를 생성합니다
func NewWithConfig(cfg *config.Config) (*GeoAPI, error) {
	// Rate Limiter 초기화
	rateLimiter := ratelimit.NewMemoryRateLimiter()
	rateLimiter.SetQuota("kakao", cfg.Kakao.DailyQuota)

	// Provider 초기화
	providersMap := make(map[string]domain.Geocoder)

	if cfg.Google.APIKey != "" {
		googleProvider := provider.NewGoogleProvider(cfg.Google.APIKey)
		googleWithBreaker := breaker.NewCircuitBreakerWrapper(googleProvider, cfg.CircuitBreaker)
		providersMap["google"] = googleWithBreaker
		log.Println("✅ Google Maps Provider initialized")
	}

	if cfg.Kakao.APIKey != "" {
		kakaoProvider := provider.NewKakaoProvider(cfg.Kakao.APIKey)
		kakaoWithBreaker := breaker.NewCircuitBreakerWrapper(kakaoProvider, cfg.CircuitBreaker)
		kakaoWithLimiter := ratelimit.NewRateLimiterWrapper(kakaoWithBreaker, rateLimiter)
		providersMap["kakao"] = kakaoWithLimiter
		log.Println("✅ Kakao Local Provider initialized")
	}

	if cfg.VWorld.APIKey != "" {
		vworldProvider := provider.NewVWorldProvider(cfg.VWorld.APIKey)
		vworldWithBreaker := breaker.NewCircuitBreakerWrapper(vworldProvider, cfg.CircuitBreaker)
		providersMap["vworld"] = vworldWithBreaker
		log.Println("✅ vWorld Provider initialized")
	}

	if len(providersMap) == 0 {
		log.Println("⚠️  No providers configured. Please set at least one API key.")
		return nil, nil
	}

	// Smart Router 초기화
	smartRouter := router.NewSmartRouter(providersMap, cfg.Server.Timeout)
	log.Println("✅ Smart Router initialized")

	return &GeoAPI{
		Router:      smartRouter,
		Config:      cfg,
		RateLimiter: rateLimiter,
	}, nil
}

// DefaultConfig는 기본 설정을 반환합니다
func DefaultConfig() *config.Config {
	return &config.Config{
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
}
