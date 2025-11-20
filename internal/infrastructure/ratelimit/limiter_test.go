package ratelimit

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryRateLimiter_Allow(t *testing.T) {
	limiter := NewMemoryRateLimiter()
	limiter.SetQuota("test-provider", 10)

	ctx := context.Background()

	// Test successful requests within quota
	for i := 0; i < 10; i++ {
		allowed, err := limiter.Allow(ctx, "test-provider")
		assert.NoError(t, err)
		assert.True(t, allowed, "Request %d should be allowed", i+1)
	}

	// Test quota exceeded
	allowed, err := limiter.Allow(ctx, "test-provider")
	assert.Error(t, err)
	assert.False(t, allowed)
}

func TestMemoryRateLimiter_GetUsage(t *testing.T) {
	limiter := NewMemoryRateLimiter()
	limiter.SetQuota("test-provider", 100)

	ctx := context.Background()

	// Initial usage should be 0
	usage, err := limiter.GetUsage(ctx, "test-provider")
	assert.NoError(t, err)
	assert.Equal(t, 0, usage)

	// Make 5 requests
	for i := 0; i < 5; i++ {
		limiter.Allow(ctx, "test-provider")
	}

	// Usage should be 5
	usage, err = limiter.GetUsage(ctx, "test-provider")
	assert.NoError(t, err)
	assert.Equal(t, 5, usage)
}

func TestMemoryRateLimiter_Reset(t *testing.T) {
	limiter := NewMemoryRateLimiter()
	limiter.SetQuota("test-provider", 10)

	ctx := context.Background()

	// Use up some quota
	for i := 0; i < 5; i++ {
		limiter.Allow(ctx, "test-provider")
	}

	usage, _ := limiter.GetUsage(ctx, "test-provider")
	assert.Equal(t, 5, usage)

	// Reset quota
	err := limiter.Reset(ctx, "test-provider")
	assert.NoError(t, err)

	// Usage should be 0 after reset
	usage, _ = limiter.GetUsage(ctx, "test-provider")
	assert.Equal(t, 0, usage)
}

func TestMemoryRateLimiter_UnlimitedProvider(t *testing.T) {
	limiter := NewMemoryRateLimiter()
	// Don't set quota for this provider

	ctx := context.Background()

	// Should allow unlimited requests
	for i := 0; i < 1000; i++ {
		allowed, err := limiter.Allow(ctx, "unlimited-provider")
		assert.NoError(t, err)
		assert.True(t, allowed)
	}
}

func TestMemoryRateLimiter_ConcurrentAccess(t *testing.T) {
	limiter := NewMemoryRateLimiter()
	limiter.SetQuota("concurrent-test", 100)

	ctx := context.Background()
	var wg sync.WaitGroup

	// Launch 100 concurrent goroutines
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			limiter.Allow(ctx, "concurrent-test")
		}()
	}

	wg.Wait()

	// All 100 requests should have been counted
	usage, _ := limiter.GetUsage(ctx, "concurrent-test")
	assert.Equal(t, 100, usage)

	// Next request should fail (quota exceeded)
	allowed, err := limiter.Allow(ctx, "concurrent-test")
	assert.Error(t, err)
	assert.False(t, allowed)
}

func TestMemoryRateLimiter_MultipleProviders(t *testing.T) {
	limiter := NewMemoryRateLimiter()
	limiter.SetQuota("provider-a", 10)
	limiter.SetQuota("provider-b", 20)

	ctx := context.Background()

	// Use provider-a quota
	for i := 0; i < 10; i++ {
		allowed, err := limiter.Allow(ctx, "provider-a")
		assert.NoError(t, err)
		assert.True(t, allowed)
	}

	// Provider-a should be exhausted
	allowed, err := limiter.Allow(ctx, "provider-a")
	assert.Error(t, err)
	assert.False(t, allowed)

	// Provider-b should still have quota
	allowed, err = limiter.Allow(ctx, "provider-b")
	assert.NoError(t, err)
	assert.True(t, allowed)

	// Check individual usage
	usageA, _ := limiter.GetUsage(ctx, "provider-a")
	usageB, _ := limiter.GetUsage(ctx, "provider-b")
	assert.Equal(t, 10, usageA)
	assert.Equal(t, 1, usageB)
}

func TestMemoryRateLimiter_AutoReset(t *testing.T) {
	limiter := NewMemoryRateLimiter()
	limiter.SetQuota("auto-reset-test", 5)

	ctx := context.Background()

	// Use up quota
	for i := 0; i < 5; i++ {
		limiter.Allow(ctx, "auto-reset-test")
	}

	// Manually set resetAt to past time (simulate midnight passing)
	limiter.mu.Lock()
	limiter.resetAt["auto-reset-test"] = time.Now().Add(-1 * time.Hour)
	limiter.mu.Unlock()

	// Next request should trigger auto-reset and succeed
	allowed, err := limiter.Allow(ctx, "auto-reset-test")
	assert.NoError(t, err)
	assert.True(t, allowed)

	// Usage should be 1 after auto-reset
	usage, _ := limiter.GetUsage(ctx, "auto-reset-test")
	assert.Equal(t, 1, usage)
}

func TestGetNextMidnight(t *testing.T) {
	limiter := NewMemoryRateLimiter()
	midnight := limiter.getNextMidnight()

	// Midnight should be in the future
	assert.True(t, midnight.After(time.Now()))

	// Midnight should be at 00:00:00
	assert.Equal(t, 0, midnight.Hour())
	assert.Equal(t, 0, midnight.Minute())
	assert.Equal(t, 0, midnight.Second())
}

// SQLite Rate Limiter Tests

func TestSQLiteRateLimiter_Allow(t *testing.T) {
	tmpDir := t.TempDir()
	limiter, err := NewSQLiteRateLimiter(tmpDir)
	require.NoError(t, err)
	defer limiter.Close()

	err = limiter.SetQuota("test-provider", 10)
	require.NoError(t, err)

	ctx := context.Background()

	// Test successful requests within quota
	for i := 0; i < 10; i++ {
		allowed, err := limiter.Allow(ctx, "test-provider")
		assert.NoError(t, err)
		assert.True(t, allowed, "Request %d should be allowed", i+1)
	}

	// Test quota exceeded
	allowed, err := limiter.Allow(ctx, "test-provider")
	assert.Error(t, err)
	assert.False(t, allowed)
}

func TestSQLiteRateLimiter_GetUsage(t *testing.T) {
	tmpDir := t.TempDir()
	limiter, err := NewSQLiteRateLimiter(tmpDir)
	require.NoError(t, err)
	defer limiter.Close()

	err = limiter.SetQuota("test-provider", 100)
	require.NoError(t, err)

	ctx := context.Background()

	// Initial usage should be 0
	usage, err := limiter.GetUsage(ctx, "test-provider")
	assert.NoError(t, err)
	assert.Equal(t, 0, usage)

	// Make 5 requests
	for i := 0; i < 5; i++ {
		limiter.Allow(ctx, "test-provider")
	}

	// Usage should be 5
	usage, err = limiter.GetUsage(ctx, "test-provider")
	assert.NoError(t, err)
	assert.Equal(t, 5, usage)
}

func TestSQLiteRateLimiter_Reset(t *testing.T) {
	tmpDir := t.TempDir()
	limiter, err := NewSQLiteRateLimiter(tmpDir)
	require.NoError(t, err)
	defer limiter.Close()

	err = limiter.SetQuota("test-provider", 10)
	require.NoError(t, err)

	ctx := context.Background()

	// Use up some quota
	for i := 0; i < 5; i++ {
		limiter.Allow(ctx, "test-provider")
	}

	usage, _ := limiter.GetUsage(ctx, "test-provider")
	assert.Equal(t, 5, usage)

	// Reset quota
	err = limiter.Reset(ctx, "test-provider")
	assert.NoError(t, err)

	// Usage should be 0 after reset
	usage, _ = limiter.GetUsage(ctx, "test-provider")
	assert.Equal(t, 0, usage)
}

func TestSQLiteRateLimiter_UnlimitedProvider(t *testing.T) {
	tmpDir := t.TempDir()
	limiter, err := NewSQLiteRateLimiter(tmpDir)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()

	// Should allow unlimited requests for provider without quota
	for i := 0; i < 100; i++ {
		allowed, err := limiter.Allow(ctx, "unlimited-provider")
		assert.NoError(t, err)
		assert.True(t, allowed)
	}
}

func TestSQLiteRateLimiter_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	limiter, err := NewSQLiteRateLimiter(tmpDir)
	require.NoError(t, err)
	defer limiter.Close()

	err = limiter.SetQuota("concurrent-test", 100)
	require.NoError(t, err)

	ctx := context.Background()
	var wg sync.WaitGroup

	// Launch 100 concurrent goroutines
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			limiter.Allow(ctx, "concurrent-test")
		}()
	}

	wg.Wait()

	// All 100 requests should have been counted
	usage, _ := limiter.GetUsage(ctx, "concurrent-test")
	assert.Equal(t, 100, usage)

	// Next request should fail (quota exceeded)
	allowed, err := limiter.Allow(ctx, "concurrent-test")
	assert.Error(t, err)
	assert.False(t, allowed)
}

func TestSQLiteRateLimiter_MultipleProviders(t *testing.T) {
	tmpDir := t.TempDir()
	limiter, err := NewSQLiteRateLimiter(tmpDir)
	require.NoError(t, err)
	defer limiter.Close()

	err = limiter.SetQuota("provider-a", 10)
	require.NoError(t, err)
	err = limiter.SetQuota("provider-b", 20)
	require.NoError(t, err)

	ctx := context.Background()

	// Use provider-a quota
	for i := 0; i < 10; i++ {
		allowed, err := limiter.Allow(ctx, "provider-a")
		assert.NoError(t, err)
		assert.True(t, allowed)
	}

	// Provider-a should be exhausted
	allowed, err := limiter.Allow(ctx, "provider-a")
	assert.Error(t, err)
	assert.False(t, allowed)

	// Provider-b should still have quota
	allowed, err = limiter.Allow(ctx, "provider-b")
	assert.NoError(t, err)
	assert.True(t, allowed)

	// Check individual usage
	usageA, _ := limiter.GetUsage(ctx, "provider-a")
	usageB, _ := limiter.GetUsage(ctx, "provider-b")
	assert.Equal(t, 10, usageA)
	assert.Equal(t, 1, usageB)
}

func TestSQLiteRateLimiter_AutoReset(t *testing.T) {
	tmpDir := t.TempDir()
	limiter, err := NewSQLiteRateLimiter(tmpDir)
	require.NoError(t, err)
	defer limiter.Close()

	err = limiter.SetQuota("auto-reset-test", 5)
	require.NoError(t, err)

	ctx := context.Background()

	// Use up quota
	for i := 0; i < 5; i++ {
		limiter.Allow(ctx, "auto-reset-test")
	}

	// Simulate midnight passing by manipulating the database directly
	resetAt := time.Now().Add(-1 * time.Hour)
	_, err = limiter.db.Exec("UPDATE rate_limits SET reset_at = ? WHERE provider = ?", resetAt, "auto-reset-test")
	require.NoError(t, err)

	// Next request should trigger auto-reset and succeed
	allowed, err := limiter.Allow(ctx, "auto-reset-test")
	assert.NoError(t, err)
	assert.True(t, allowed)

	// Usage should be 1 after auto-reset
	usage, _ := limiter.GetUsage(ctx, "auto-reset-test")
	assert.Equal(t, 1, usage)
}

func TestSQLiteRateLimiter_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create first limiter and set quota
	limiter1, err := NewSQLiteRateLimiter(tmpDir)
	require.NoError(t, err)

	err = limiter1.SetQuota("persistence-test", 100)
	require.NoError(t, err)

	// Make some requests
	for i := 0; i < 25; i++ {
		limiter1.Allow(ctx, "persistence-test")
	}

	usage1, _ := limiter1.GetUsage(ctx, "persistence-test")
	assert.Equal(t, 25, usage1)

	// Close first limiter
	limiter1.Close()

	// Create second limiter pointing to same database
	limiter2, err := NewSQLiteRateLimiter(tmpDir)
	require.NoError(t, err)
	defer limiter2.Close()

	// Usage should persist
	usage2, _ := limiter2.GetUsage(ctx, "persistence-test")
	assert.Equal(t, 25, usage2)

	// Make more requests
	for i := 0; i < 10; i++ {
		limiter2.Allow(ctx, "persistence-test")
	}

	// Total usage should be 35
	usage3, _ := limiter2.GetUsage(ctx, "persistence-test")
	assert.Equal(t, 35, usage3)
}

func TestSQLiteRateLimiter_DatabaseCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "ratelimit.db")

	limiter, err := NewSQLiteRateLimiter(tmpDir)
	require.NoError(t, err)

	// Database file should exist
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)

	// Close limiter
	err = limiter.Close()
	assert.NoError(t, err)

	// Database file should still exist (persistence)
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)
}
