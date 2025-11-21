package ratelimit

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite" // CGO-free SQLite driver

	"github.com/epicsagas/korean-geocode/pkg/domain"
)

// SQLiteRateLimiter는 SQLite 기반 Rate Limiter입니다
// Redis를 사용할 수 없는 환경에서 영구 저장소로 사용합니다
type SQLiteRateLimiter struct {
	db     *sql.DB
	mu     sync.RWMutex
	dbPath string
}

// NewSQLiteRateLimiter는 새로운 SQLite 기반 Rate Limiter를 생성합니다
func NewSQLiteRateLimiter(dataDir string) (*SQLiteRateLimiter, error) {
	// 데이터 디렉토리가 비어있으면 현재 디렉토리 사용
	if dataDir == "" {
		dataDir = "."
	}

	// 데이터 디렉토리 생성 (존재하지 않을 경우)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "ratelimit.db")

	// SQLite 데이터베이스 열기
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// 테이블 생성 (date 컬럼 추가)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS rate_limits (
			provider TEXT NOT NULL,
			date TEXT NOT NULL,
			quota INTEGER NOT NULL,
			usage INTEGER NOT NULL DEFAULT 0,
			reset_at TIMESTAMP NOT NULL,
			PRIMARY KEY (provider, date)
		);
		CREATE INDEX IF NOT EXISTS idx_provider_date ON rate_limits(provider, date);
		CREATE INDEX IF NOT EXISTS idx_reset_at ON rate_limits(reset_at);
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	limiter := &SQLiteRateLimiter{
		db:     db,
		dbPath: dbPath,
	}

	// 오래된 데이터 정리 (시작 시 한 번)
	limiter.cleanupExpiredData()

	return limiter, nil
}

// SetQuota는 Provider의 일일 쿼터를 설정합니다
func (s *SQLiteRateLimiter) SetQuota(provider string, quota int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	resetAt := s.getNextMidnight()

	_, err := s.db.Exec(`
		INSERT INTO rate_limits (provider, date, quota, usage, reset_at)
		VALUES (?, ?, ?, 0, ?)
		ON CONFLICT(provider, date) DO UPDATE SET
			quota = excluded.quota,
			usage = 0,
			reset_at = excluded.reset_at
	`, provider, today, quota, resetAt)

	return err
}

// Allow는 요청이 쿼터 내에 있는지 확인하고, 카운터를 증가시킵니다
func (s *SQLiteRateLimiter) Allow(ctx context.Context, provider string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	today := time.Now().Format("2006-01-02")

	// 트랜잭션 시작
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	// 현재 데이터 조회
	var quota, usage int
	var resetAt time.Time
	err = tx.QueryRow("SELECT quota, usage, reset_at FROM rate_limits WHERE provider = ? AND date = ?", provider, today).
		Scan(&quota, &usage, &resetAt)

	if err == sql.ErrNoRows {
		// 쿼터 미설정 시 무제한 허용
		return true, nil
	}
	if err != nil {
		return false, err
	}

	// 자동 리셋 확인
	now := time.Now()
	if now.After(resetAt) {
		usage = 0
		resetAt = s.getNextMidnight()

		_, err = tx.Exec("UPDATE rate_limits SET usage = 0, reset_at = ? WHERE provider = ? AND date = ?", resetAt, provider, today)
		if err != nil {
			return false, err
		}
	}

	// 쿼터 확인
	if usage >= quota {
		return false, domain.ErrQuotaExceeded
	}

	// 사용량 증가
	_, err = tx.Exec("UPDATE rate_limits SET usage = usage + 1 WHERE provider = ? AND date = ?", provider, today)
	if err != nil {
		return false, err
	}

	// 트랜잭션 커밋
	if err = tx.Commit(); err != nil {
		return false, err
	}

	return true, nil
}

// GetUsage는 현재 사용량을 반환합니다
func (s *SQLiteRateLimiter) GetUsage(ctx context.Context, provider string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	today := time.Now().Format("2006-01-02")
	var usage int
	err := s.db.QueryRow("SELECT usage FROM rate_limits WHERE provider = ? AND date = ?", provider, today).Scan(&usage)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return usage, nil
}

// GetQuota는 설정된 쿼터를 반환합니다
func (s *SQLiteRateLimiter) GetQuota(ctx context.Context, provider string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	today := time.Now().Format("2006-01-02")
	var quota int
	err := s.db.QueryRow("SELECT quota FROM rate_limits WHERE provider = ? AND date = ?", provider, today).Scan(&quota)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return quota, nil
}

// Reset은 쿼터를 수동으로 초기화합니다
func (s *SQLiteRateLimiter) Reset(ctx context.Context, provider string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	resetAt := s.getNextMidnight()

	_, err := s.db.Exec("UPDATE rate_limits SET usage = 0, reset_at = ? WHERE provider = ? AND date = ?", resetAt, provider, today)
	return err
}

// Close는 데이터베이스 연결을 종료합니다
func (s *SQLiteRateLimiter) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// getNextMidnight는 다음 자정(00:00) 시간을 반환합니다
func (s *SQLiteRateLimiter) getNextMidnight() time.Time {
	now := time.Now()
	year, month, day := now.Date()
	midnight := time.Date(year, month, day+1, 0, 0, 0, 0, now.Location())
	return midnight
}

// cleanupExpiredData는 오래된 리셋 데이터를 정리합니다 (시작 시 자동 호출)
func (s *SQLiteRateLimiter) cleanupExpiredData() {
	// 7일 이상 지난 데이터 삭제
	threshold := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	s.db.Exec("DELETE FROM rate_limits WHERE date < ?", threshold)
}

// CleanupOldData는 지정한 일자 이전의 데이터를 삭제합니다
func (s *SQLiteRateLimiter) CleanupOldData(ctx context.Context, beforeDate string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.ExecContext(ctx, "DELETE FROM rate_limits WHERE date < ?", beforeDate)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowsAffected), nil
}
