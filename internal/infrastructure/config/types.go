package config

import "time"

// Config는 전체 시스템 설정을 담습니다
type Config struct {
	Server              ServerConfig
	Google              ProviderConfig
	Kakao               ProviderConfig
	VWorld              ProviderConfig
	Naver               NaverConfig
	CircuitBreaker      CircuitBreakerConfig
	RateLimiter         RateLimiterConfig
	KoreanProviderOrder []string // 한국 주소용 프로바이더 순서
	GlobalProviderOrder []string // 해외 주소용 프로바이더 순서
}

// ServerConfig는 HTTP 서버 설정입니다
type ServerConfig struct {
	Port    string
	Timeout time.Duration
}

// ProviderConfig는 각 공급자별 설정입니다
type ProviderConfig struct {
	APIKey       string
	Timeout      time.Duration
	DailyQuota   int
	MonthlyQuota int
}

// NaverConfig는 Naver Cloud Platform 설정입니다
// Naver는 두 개의 API 키가 필요합니다
type NaverConfig struct {
	ClientID     string
	ClientSecret string
	Timeout      time.Duration
	DailyQuota   int
}

// CircuitBreakerConfig는 Circuit Breaker 설정입니다
type CircuitBreakerConfig struct {
	MaxRequests      uint32
	FailureThreshold uint32
	Timeout          time.Duration
}

// RateLimiterConfig는 Rate Limiter 설정입니다
type RateLimiterConfig struct {
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int
}
