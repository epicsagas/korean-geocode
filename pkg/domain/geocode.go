package domain

import (
	"context"
	"errors"
)

// Geocoder 인터페이스: 모든 공급자는 이 인터페이스를 구현해야 함
type Geocoder interface {
	// Name은 공급자의 이름을 반환합니다
	Name() string

	// Geocode는 주소 문자열을 좌표로 변환합니다
	Geocode(ctx context.Context, query string) (*GeoResult, error)
}

// GeoResult 공통 결과 구조체 (좌표계는 WGS84로 통일)
type GeoResult struct {
	Provider  string  `json:"provider"` // 응답한 공급자 이름
	Latitude  float64 `json:"lat"`      // 위도 (WGS84)
	Longitude float64 `json:"lng"`      // 경도 (WGS84)
	Address   string  `json:"address"`  // 정규화된 주소
	CRS       string  `json:"crs"`      // 좌표계 (항상 "WGS84")
}

// 공통 에러 정의
var (
	ErrInvalidAddress      = errors.New("invalid address")
	ErrProviderTimeout     = errors.New("provider timeout")
	ErrProviderUnavailable = errors.New("provider unavailable")
	ErrQuotaExceeded       = errors.New("quota exceeded")
	ErrInvalidCoordinates  = errors.New("invalid coordinates")
	ErrNoResultsFound      = errors.New("no results found")
)

// ValidateCoordinates는 좌표가 유효 범위 내인지 검증합니다
// Lat: -90~90, Lng: -180~180
func (r *GeoResult) ValidateCoordinates() error {
	if r.Latitude < -90 || r.Latitude > 90 {
		return ErrInvalidCoordinates
	}
	if r.Longitude < -180 || r.Longitude > 180 {
		return ErrInvalidCoordinates
	}
	return nil
}
