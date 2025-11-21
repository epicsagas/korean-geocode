package config

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// LoadConfig는 .env 파일과 환경변수에서 설정을 로드합니다
func LoadConfig() (*Config, error) {
	// .env 파일 로드 (선택적, 파일이 없어도 에러 무시)
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, using environment variables or defaults")
	}

	// Viper 설정
	setupViper()

	// Config 구조체 생성
	cfg := &Config{
		Server: ServerConfig{
			Port:    viper.GetString("server.port"),
			Timeout: viper.GetDuration("server.timeout"),
		},
		Google: ProviderConfig{
			APIKey:       viper.GetString("google.api_key"),
			Timeout:      viper.GetDuration("google.timeout"),
			MonthlyQuota: viper.GetInt("google.monthly_credit"),
		},
		Kakao: ProviderConfig{
			APIKey:     viper.GetString("kakao.api_key"),
			Timeout:    viper.GetDuration("kakao.timeout"),
			DailyQuota: viper.GetInt("kakao.daily_quota"),
		},
		VWorld: ProviderConfig{
			APIKey:     viper.GetString("vworld.api_key"),
			Timeout:    viper.GetDuration("vworld.timeout"),
			DailyQuota: viper.GetInt("vworld.daily_quota"),
		},
		Naver: NaverConfig{
			ClientID:     viper.GetString("naver.client_id"),
			ClientSecret: viper.GetString("naver.client_secret"),
			Timeout:      viper.GetDuration("naver.timeout"),
			DailyQuota:   viper.GetInt("naver.daily_quota"),
		},
		CircuitBreaker: CircuitBreakerConfig{
			MaxRequests:      uint32(viper.GetInt("circuit_breaker.max_requests")),
			FailureThreshold: uint32(viper.GetInt("circuit_breaker.failure_threshold")),
			Timeout:          viper.GetDuration("circuit_breaker.timeout"),
		},
		RateLimiter: RateLimiterConfig{
			RedisHost:     viper.GetString("redis.host"),
			RedisPort:     viper.GetString("redis.port"),
			RedisPassword: viper.GetString("redis.password"),
			RedisDB:       viper.GetInt("redis.db"),
		},
		KoreanProviderOrder: parseProviderOrder(viper.GetString("korean.provider_order")),
		GlobalProviderOrder: parseProviderOrder(viper.GetString("global.provider_order")),
	}

	// 필수 API 키 검증
	naverConfigured := cfg.Naver.ClientID != "" && cfg.Naver.ClientSecret != ""
	if cfg.Google.APIKey == "" && cfg.Kakao.APIKey == "" && cfg.VWorld.APIKey == "" && !naverConfigured {
		return nil, fmt.Errorf("at least one API key must be configured")
	}

	return cfg, nil
}

// setupViper는 Viper 기본값 및 환경변수 바인딩을 설정합니다
func setupViper() {
	// 환경변수 자동 바인딩
	viper.AutomaticEnv()

	// 환경변수 키 매핑 (SNAKE_CASE → dot.notation)
	viper.BindEnv("server.port", "SERVER_PORT")
	viper.BindEnv("server.timeout", "SERVER_TIMEOUT")

	viper.BindEnv("google.api_key", "GOOGLE_MAPS_API_KEY")
	viper.BindEnv("google.timeout", "GOOGLE_TIMEOUT")
	viper.BindEnv("google.monthly_credit", "GOOGLE_MONTHLY_CREDIT")

	viper.BindEnv("kakao.api_key", "KAKAO_REST_API_KEY")
	viper.BindEnv("kakao.timeout", "KAKAO_TIMEOUT")
	viper.BindEnv("kakao.daily_quota", "KAKAO_DAILY_QUOTA")

	viper.BindEnv("vworld.api_key", "VWORLD_API_KEY")
	viper.BindEnv("vworld.timeout", "VWORLD_TIMEOUT")
	viper.BindEnv("vworld.daily_quota", "VWORLD_DAILY_QUOTA")

	viper.BindEnv("naver.client_id", "NAVER_CLIENT_ID")
	viper.BindEnv("naver.client_secret", "NAVER_CLIENT_SECRET")
	viper.BindEnv("naver.timeout", "NAVER_TIMEOUT")
	viper.BindEnv("naver.daily_quota", "NAVER_DAILY_QUOTA")

	viper.BindEnv("circuit_breaker.max_requests", "CIRCUIT_BREAKER_MAX_REQUESTS")
	viper.BindEnv("circuit_breaker.failure_threshold", "CIRCUIT_BREAKER_FAILURE_THRESHOLD")
	viper.BindEnv("circuit_breaker.timeout", "CIRCUIT_BREAKER_TIMEOUT")

	viper.BindEnv("redis.host", "REDIS_HOST")
	viper.BindEnv("redis.port", "REDIS_PORT")
	viper.BindEnv("redis.password", "REDIS_PASSWORD")
	viper.BindEnv("redis.db", "REDIS_DB")

	viper.BindEnv("korean.provider_order", "KOREAN_PROVIDER_ORDER")
	viper.BindEnv("global.provider_order", "GLOBAL_PROVIDER_ORDER")

	// 기본값 설정
	setDefaults()
}

// setDefaults는 모든 설정의 기본값을 설정합니다
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.timeout", 5*time.Second)

	// Provider timeouts
	viper.SetDefault("google.timeout", 2*time.Second)
	viper.SetDefault("kakao.timeout", 2*time.Second)
	viper.SetDefault("vworld.timeout", 2*time.Second)
	viper.SetDefault("naver.timeout", 2*time.Second)

	// Provider quotas
	viper.SetDefault("vworld.daily_quota", 100000)
	viper.SetDefault("kakao.daily_quota", 300000)
	viper.SetDefault("naver.daily_quota", 100000)

	// Circuit Breaker defaults
	viper.SetDefault("circuit_breaker.max_requests", 10)
	viper.SetDefault("circuit_breaker.failure_threshold", 5)
	viper.SetDefault("circuit_breaker.timeout", 60*time.Second)

	// Redis defaults - 기본값을 설정하지 않아 명시적 설정 시에만 사용
	// viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", "6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// Provider order defaults
	viper.SetDefault("korean.provider_order", "vworld,kakao,naver,google")
	viper.SetDefault("global.provider_order", "google,naver,kakao")
}

// parseProviderOrder는 콤마로 구분된 문자열을 슬라이스로 파싱합니다
func parseProviderOrder(order string) []string {
	if order == "" {
		return []string{}
	}

	parts := strings.Split(order, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
