package ratelimit

import (
	"context"
	"fmt"
	"strings"
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
	today := time.Now().Format("2006-01-02")
	quotaKey := fmt.Sprintf("ratelimit:%s:%s:quota", provider, today)
	usageKey := fmt.Sprintf("ratelimit:%s:%s:usage", provider, today)
	resetKey := fmt.Sprintf("ratelimit:%s:%s:reset_at", provider, today)

	// 다음 자정까지 TTL 계산
	ttl := time.Until(getNextMidnight())

	// 쿼터 설정 (TTL 적용)
	if err := r.client.Set(ctx, quotaKey, quota, ttl).Err(); err != nil {
		return err
	}

	// 사용량 초기화 (키가 없을 때만, TTL 적용)
	exists, err := r.client.Exists(ctx, usageKey).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		if err := r.client.Set(ctx, usageKey, 0, ttl).Err(); err != nil {
			return err
		}
	}

	// 리셋 시간 설정 (키가 없을 때만, TTL 적용)
	exists, err = r.client.Exists(ctx, resetKey).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		resetAt := getNextMidnight()
		if err := r.client.Set(ctx, resetKey, resetAt.Unix(), ttl).Err(); err != nil {
			return err
		}
	}

	return nil
}

// Allow는 요청이 쿼터 내에 있는지 확인하고, 카운터를 증가시킵니다
func (r *RedisRateLimiter) Allow(ctx context.Context, provider string) (bool, error) {
	today := time.Now().Format("2006-01-02")
	quotaKey := fmt.Sprintf("ratelimit:%s:%s:quota", provider, today)
	usageKey := fmt.Sprintf("ratelimit:%s:%s:usage", provider, today)
	resetKey := fmt.Sprintf("ratelimit:%s:%s:reset_at", provider, today)

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
	today := time.Now().Format("2006-01-02")
	usageKey := fmt.Sprintf("ratelimit:%s:%s:usage", provider, today)
	usage, err := r.client.Get(ctx, usageKey).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return usage, err
}

// GetQuota는 설정된 쿼터를 반환합니다
func (r *RedisRateLimiter) GetQuota(ctx context.Context, provider string) (int, error) {
	today := time.Now().Format("2006-01-02")
	quotaKey := fmt.Sprintf("ratelimit:%s:%s:quota", provider, today)
	quota, err := r.client.Get(ctx, quotaKey).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return quota, err
}

// Reset은 쿼터를 수동으로 초기화합니다
func (r *RedisRateLimiter) Reset(ctx context.Context, provider string) error {
	today := time.Now().Format("2006-01-02")
	usageKey := fmt.Sprintf("ratelimit:%s:%s:usage", provider, today)
	resetKey := fmt.Sprintf("ratelimit:%s:%s:reset_at", provider, today)

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

// CleanupOldData는 지정한 일자 이전의 데이터를 삭제합니다
func (r *RedisRateLimiter) CleanupOldData(ctx context.Context, beforeDate string) (int, error) {
	// Redis는 패턴 매칭으로 모든 키를 찾아야 함
	pattern := "ratelimit:*:*:*"
	var cursor uint64
	deletedCount := 0

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return deletedCount, err
		}

		// 각 키에서 날짜 추출 및 삭제
		for _, key := range keys {
			// ratelimit:provider:YYYY-MM-DD:type 형식
			parts := strings.Split(key, ":")
			if len(parts) >= 3 {
				keyDate := parts[2]
				if keyDate < beforeDate {
					if err := r.client.Del(ctx, key).Err(); err == nil {
						deletedCount++
					}
				}
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return deletedCount, nil
}
