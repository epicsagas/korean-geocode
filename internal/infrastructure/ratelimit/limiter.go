package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/epicsagas/korean-geocode/pkg/domain"
)

// RateLimiter는 API 쿼터 관리를 위한 인터페이스입니다
type RateLimiter interface {
	// Allow는 요청이 쿼터 내에 있는지 확인합니다
	Allow(ctx context.Context, provider string) (bool, error)

	// GetUsage는 현재 사용량을 반환합니다
	GetUsage(ctx context.Context, provider string) (int, error)

	// GetQuota는 설정된 쿼터를 반환합니다
	GetQuota(ctx context.Context, provider string) (int, error)

	// Reset은 쿼터를 초기화합니다
	Reset(ctx context.Context, provider string) error

	// CleanupOldData는 지정한 일자 이전의 데이터를 삭제합니다
	CleanupOldData(ctx context.Context, beforeDate string) (int, error)
}

// MemoryRateLimiter는 메모리 기반 Rate Limiter입니다
// Production에서는 Redis 기반으로 교체 권장
type MemoryRateLimiter struct {
	mu      sync.RWMutex
	quotas  map[string]int       // Provider별 일일 쿼터
	usage   map[string]int       // Provider별 현재 사용량
	resetAt map[string]time.Time // Provider별 리셋 시간
}

// NewMemoryRateLimiter는 새로운 메모리 기반 Rate Limiter를 생성합니다
func NewMemoryRateLimiter() *MemoryRateLimiter {
	return &MemoryRateLimiter{
		quotas:  make(map[string]int),
		usage:   make(map[string]int),
		resetAt: make(map[string]time.Time),
	}
}

// SetQuota는 Provider의 일일 쿼터를 설정합니다
func (m *MemoryRateLimiter) SetQuota(provider string, quota int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.quotas[provider] = quota
	m.usage[provider] = 0
	m.resetAt[provider] = m.getNextMidnight()
}

// Allow는 요청이 쿼터 내에 있는지 확인하고, 카운터를 증가시킵니다
func (m *MemoryRateLimiter) Allow(ctx context.Context, provider string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 자동 리셋 확인
	if time.Now().After(m.resetAt[provider]) {
		m.usage[provider] = 0
		m.resetAt[provider] = m.getNextMidnight()
	}

	quota, exists := m.quotas[provider]
	if !exists {
		// 쿼터 미설정 시 무제한 허용
		return true, nil
	}

	currentUsage := m.usage[provider]
	if currentUsage >= quota {
		return false, domain.ErrQuotaExceeded
	}

	// 사용량 증가
	m.usage[provider]++
	return true, nil
}

// GetUsage는 현재 사용량을 반환합니다
func (m *MemoryRateLimiter) GetUsage(ctx context.Context, provider string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	usage, exists := m.usage[provider]
	if !exists {
		return 0, nil
	}

	return usage, nil
}

// GetQuota는 설정된 쿼터를 반환합니다
func (m *MemoryRateLimiter) GetQuota(ctx context.Context, provider string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	quota, exists := m.quotas[provider]
	if !exists {
		return 0, nil
	}

	return quota, nil
}

// Reset은 쿼터를 수동으로 초기화합니다
func (m *MemoryRateLimiter) Reset(ctx context.Context, provider string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.usage[provider] = 0
	m.resetAt[provider] = m.getNextMidnight()
	return nil
}

// getNextMidnight는 다음 자정(00:00) 시간을 반환합니다
func (m *MemoryRateLimiter) getNextMidnight() time.Time {
	now := time.Now()
	year, month, day := now.Date()
	midnight := time.Date(year, month, day+1, 0, 0, 0, 0, now.Location())
	return midnight
}

// CleanupOldData는 메모리 기반이므로 항상 0을 반환합니다 (자동 정리됨)
func (m *MemoryRateLimiter) CleanupOldData(ctx context.Context, beforeDate string) (int, error) {
	return 0, nil
}

// RateLimiterWrapper는 Geocoder를 Rate Limiter로 감싸는 래퍼입니다
type RateLimiterWrapper struct {
	provider domain.Geocoder
	limiter  RateLimiter
}

// NewRateLimiterWrapper는 새로운 Rate Limiter Wrapper를 생성합니다
func NewRateLimiterWrapper(provider domain.Geocoder, limiter RateLimiter) *RateLimiterWrapper {
	return &RateLimiterWrapper{
		provider: provider,
		limiter:  limiter,
	}
}

// Name은 공급자 이름을 반환합니다
func (r *RateLimiterWrapper) Name() string {
	return r.provider.Name()
}

// Geocode는 Rate Limiter를 통해 Geocoding을 수행합니다
func (r *RateLimiterWrapper) Geocode(ctx context.Context, query string) (*domain.GeoResult, error) {
	// 쿼터 확인
	allowed, err := r.limiter.Allow(ctx, r.provider.Name())
	if err != nil {
		return nil, err
	}

	if !allowed {
		return nil, fmt.Errorf("rate limit exceeded for %s", r.provider.Name())
	}

	// 실제 Geocoding 수행
	return r.provider.Geocode(ctx, query)
}
