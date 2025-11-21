package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/epicsagas/korean-geocode/docs/swagger" // swagger docs
	handlerhttp "github.com/epicsagas/korean-geocode/internal/handler/http"
	"github.com/epicsagas/korean-geocode/internal/infrastructure/breaker"
	"github.com/epicsagas/korean-geocode/internal/infrastructure/config"
	"github.com/epicsagas/korean-geocode/internal/infrastructure/metrics"
	"github.com/epicsagas/korean-geocode/internal/infrastructure/ratelimit"
	"github.com/epicsagas/korean-geocode/pkg/domain"
	"github.com/epicsagas/korean-geocode/pkg/provider"
	"github.com/epicsagas/korean-geocode/pkg/router"
)

// @title GeoCoding Hybrid API
// @version 1.0
// @description 고가용성 하이브리드 Geocoding 서비스 - Google Maps, Kakao Local, vWorld API 통합
// @description
// @description ## 주요 특징
// @description - 🌏 하이브리드 전략: 비용 최적화 + 고가용성
// @description - 🔄 Smart Router: 한글/영문 자동 감지 및 최적 Provider 선택
// @description - ⚡ Circuit Breaker: 장애 격리 및 자동 복구
// @description - 📊 Rate Limiting: 일일 쿼터 관리
// @description - 🗺️ WGS84 통일: 모든 좌표계를 WGS84로 정규화
// @description
// @description ## Provider 우선순위
// @description - 한글 주소: Kakao → vWorld → Google
// @description - 영문 주소: Google → Kakao

// @contact.name API Support
// @contact.url https://github.com/epicsagas/korean-geocode/issues

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /
// @schemes http https

// @tag.name geocoding
// @tag.description 주소-좌표 변환 API

// @tag.name health
// @tag.description 서버 상태 확인

func main() {
	// 설정 로드
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Rate Limiter 초기화 (Redis 명시적 설정 시 Redis, 미설정 시 SQLite 사용)
	var rateLimiter ratelimit.RateLimiter
	if cfg.RateLimiter.RedisHost != "" {
		// Redis 설정이 있으면 Redis 사용
		redisLimiter, err := ratelimit.NewRedisRateLimiter(
			cfg.RateLimiter.RedisHost,
			cfg.RateLimiter.RedisPort,
			cfg.RateLimiter.RedisPassword,
			cfg.RateLimiter.RedisDB,
		)
		if err != nil {
			log.Fatalf("Failed to create Redis rate limiter: %v", err)
		}
		defer redisLimiter.Close()

		if cfg.VWorld.DailyQuota > 0 {
			redisLimiter.SetQuota("vworld", cfg.VWorld.DailyQuota)
		}
		if cfg.Kakao.DailyQuota > 0 {
			redisLimiter.SetQuota("kakao", cfg.Kakao.DailyQuota)
		}
		if cfg.Naver.ClientID != "" && cfg.Naver.ClientSecret != "" {
			if cfg.Naver.DailyQuota > 0 {
				redisLimiter.SetQuota("naver", cfg.Naver.DailyQuota)
			} else {
				redisLimiter.SetQuota("naver", 600) // Default: 600/month free tier
			}
		}
		if cfg.Google.MonthlyQuota > 0 {
			redisLimiter.SetQuota("google", cfg.Google.MonthlyQuota)
		}

		rateLimiter = redisLimiter
		log.Printf("✅ Redis Rate Limiter initialized (%s:%s)",
			cfg.RateLimiter.RedisHost, cfg.RateLimiter.RedisPort)
	} else {
		// Redis 미설정 시 SQLite 사용
		sqliteLimiter, err := ratelimit.NewSQLiteRateLimiter("./data")
		if err != nil {
			log.Fatalf("Failed to create SQLite rate limiter: %v", err)
		}
		defer sqliteLimiter.Close()

		if cfg.VWorld.DailyQuota > 0 {
			sqliteLimiter.SetQuota("vworld", cfg.VWorld.DailyQuota)
		}
		if cfg.Kakao.DailyQuota > 0 {
			sqliteLimiter.SetQuota("kakao", cfg.Kakao.DailyQuota)
		}
		if cfg.Naver.ClientID != "" && cfg.Naver.ClientSecret != "" {
			if cfg.Naver.DailyQuota > 0 {
				sqliteLimiter.SetQuota("naver", cfg.Naver.DailyQuota)
			} else {
				sqliteLimiter.SetQuota("naver", 600) // Default: 600/month free tier
			}
		}
		if cfg.Google.MonthlyQuota > 0 {
			sqliteLimiter.SetQuota("google", cfg.Google.MonthlyQuota)
		}

		rateLimiter = sqliteLimiter
		log.Println("✅ SQLite Rate Limiter initialized (./data/ratelimit.db)")
	}

	// Prometheus Exporter 초기화
	activeProviders := []string{}
	if cfg.Google.APIKey != "" {
		activeProviders = append(activeProviders, "google")
	}
	if cfg.Kakao.APIKey != "" {
		activeProviders = append(activeProviders, "kakao")
	}
	if cfg.VWorld.APIKey != "" {
		activeProviders = append(activeProviders, "vworld")
	}
	if cfg.Naver.ClientID != "" && cfg.Naver.ClientSecret != "" {
		activeProviders = append(activeProviders, "naver")
	}

	metricsExporter := metrics.NewExporter(rateLimiter, activeProviders)
	if err := metricsExporter.Register(); err != nil {
		log.Printf("⚠️  Failed to register Prometheus exporter: %v", err)
	} else {
		log.Println("✅ Prometheus metrics exporter registered (/metrics)")
	}

	// Provider 초기화 및 Circuit Breaker, Rate Limiter 적용
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
		vworldWithLimiter := ratelimit.NewRateLimiterWrapper(vworldWithBreaker, rateLimiter)
		providersMap["vworld"] = vworldWithLimiter
		log.Println("✅ vWorld Provider initialized")
	}

	if cfg.Naver.ClientID != "" && cfg.Naver.ClientSecret != "" {
		naverProvider := provider.NewNaverProvider(cfg.Naver.ClientID, cfg.Naver.ClientSecret)
		naverWithBreaker := breaker.NewCircuitBreakerWrapper(naverProvider, cfg.CircuitBreaker)
		naverWithLimiter := ratelimit.NewRateLimiterWrapper(naverWithBreaker, rateLimiter)
		providersMap["naver"] = naverWithLimiter
		log.Println("✅ Naver Maps Provider initialized")
	}

	if len(providersMap) == 0 {
		log.Fatal("❌ No providers configured. Please set at least one API key.")
	}

	// Smart Router 초기화 (설정된 프로바이더 순서 사용)
	smartRouter := router.NewSmartRouterWithOrder(
		providersMap,
		cfg.Server.Timeout,
		cfg.KoreanProviderOrder,
		cfg.GlobalProviderOrder,
	)
	log.Printf("✅ Smart Router initialized (Korean: %v, Global: %v)",
		cfg.KoreanProviderOrder, cfg.GlobalProviderOrder)

	// HTTP 라우트 설정 (새로운 Handler Layer 사용)
	handler := handlerhttp.SetupRoutes(smartRouter, rateLimiter)

	// HTTP 서버 설정
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Graceful Shutdown 설정
	go func() {
		log.Printf("🚀 Server starting on port %s", cfg.Server.Port)
		log.Printf("📚 Swagger UI: http://localhost:%s/swagger/", cfg.Server.Port)
		log.Printf("📖 API Docs: http://localhost:%s/docs", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// 종료 시그널 대기
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server exited gracefully")
}
