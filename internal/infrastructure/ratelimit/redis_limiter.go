package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/epicsagas/korean-geocode/pkg/domain"
	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter는 Redis 기반 Rate Limiter입니다
type RedisRateLimiter struct {
	client *redis.Client
}

// NewRedisRateLimiter는 새로운 Redis 기반 Rate Limiter를 생성합니다
func NewRedisRateLimiter(host, port, password string, db int) (*RedisRateLimiter, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: password,
		DB:       db,
	})

	// 연결 테스트
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisRateLimiter{client: client}, nil
}

// SetQuota는 Provider의 일일 쿼터를 설정합니다
func (r *RedisRateLimiter) SetQuota(provider string, quota int) error {
	ctx := context.Background()
	quotaKey := fmt.Sprintf("ratelimit:%s:quota", provider)
	usageKey := fmt.Sprintf("ratelimit:%s:usage", provider)
	resetKey := fmt.Sprintf("ratelimit:%s:reset_at", provider)

	// 쿼터 설정
	if err := r.client.Set(ctx, quotaKey, quota, 0).Err(); err != nil {
		return err
	}

	// 사용량 초기화 (키가 없을 때만)
	exists, err := r.client.Exists(ctx, usageKey).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		if err := r.client.Set(ctx, usageKey, 0, 0).Err(); err != nil {
			return err
		}
	}

	// 리셋 시간 설정 (키가 없을 때만)
	exists, err = r.client.Exists(ctx, resetKey).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		resetAt := getNextMidnight()
		if err := r.client.Set(ctx, resetKey, resetAt.Unix(), 0).Err(); err != nil {
			return err
		}
	}

	return nil
}

// Allow는 요청이 쿼터 내에 있는지 확인하고, 카운터를 증가시킵니다
func (r *RedisRateLimiter) Allow(ctx context.Context, provider string) (bool, error) {
	quotaKey := fmt.Sprintf("ratelimit:%s:quota", provider)
	usageKey := fmt.Sprintf("ratelimit:%s:usage", provider)
	resetKey := fmt.Sprintf("ratelimit:%s:reset_at", provider)

	// 쿼터 조회
	quota, err := r.client.Get(ctx, quotaKey).Int()
	if err == redis.Nil {
		// 쿼터 미설정 시 무제한 허용
		return true, nil
	}
	if err != nil {
		return false, err
	}

	// 리셋 시간 확인
	resetUnix, err := r.client.Get(ctx, resetKey).Int64()
	if err == nil {
		resetAt := time.Unix(resetUnix, 0)
		if time.Now().After(resetAt) {
			// 자동 리셋
			if err := r.client.Set(ctx, usageKey, 0, 0).Err(); err != nil {
				return false, err
			}
			newResetAt := getNextMidnight()
			if err := r.client.Set(ctx, resetKey, newResetAt.Unix(), 0).Err(); err != nil {
				return false, err
			}
		}
	}

	// 현재 사용량 조회
	usage, err := r.client.Get(ctx, usageKey).Int()
	if err != nil && err != redis.Nil {
		return false, err
	}

	// 쿼터 확인
	if usage >= quota {
		return false, domain.ErrQuotaExceeded
	}

	// 사용량 증가
	if err := r.client.Incr(ctx, usageKey).Err(); err != nil {
		return false, err
	}

	return true, nil
}

// GetUsage는 현재 사용량을 반환합니다
func (r *RedisRateLimiter) GetUsage(ctx context.Context, provider string) (int, error) {
	usageKey := fmt.Sprintf("ratelimit:%s:usage", provider)
	usage, err := r.client.Get(ctx, usageKey).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return usage, err
}

// GetQuota는 설정된 쿼터를 반환합니다
func (r *RedisRateLimiter) GetQuota(ctx context.Context, provider string) (int, error) {
	quotaKey := fmt.Sprintf("ratelimit:%s:quota", provider)
	quota, err := r.client.Get(ctx, quotaKey).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return quota, err
}

// Reset은 쿼터를 수동으로 초기화합니다
func (r *RedisRateLimiter) Reset(ctx context.Context, provider string) error {
	usageKey := fmt.Sprintf("ratelimit:%s:usage", provider)
	resetKey := fmt.Sprintf("ratelimit:%s:reset_at", provider)

	if err := r.client.Set(ctx, usageKey, 0, 0).Err(); err != nil {
		return err
	}

	resetAt := getNextMidnight()
	return r.client.Set(ctx, resetKey, resetAt.Unix(), 0).Err()
}

// Close는 Redis 연결을 종료합니다
func (r *RedisRateLimiter) Close() error {
	return r.client.Close()
}

// getNextMidnight는 다음 자정(00:00) 시간을 반환합니다
func getNextMidnight() time.Time {
	now := time.Now()
	year, month, day := now.Date()
	midnight := time.Date(year, month, day+1, 0, 0, 0, 0, now.Location())
	return midnight
}
