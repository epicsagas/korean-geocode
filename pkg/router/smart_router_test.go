package router

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/epicsagas/korean-geocode/pkg/domain"
	"github.com/stretchr/testify/assert"
)

// MockGeocoder는 테스트용 Geocoder 구현입니다
type MockGeocoder struct {
	name   string
	result *domain.GeoResult
	err    error
}

func (m *MockGeocoder) Name() string {
	return m.name
}

func (m *MockGeocoder) Geocode(ctx context.Context, query string) (*domain.GeoResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func TestSmartRouter_DetermineProviderOrder(t *testing.T) {
	router := NewSmartRouter(nil, 5*time.Second)

	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "Korean address - Hangul",
			query:    "서울특별시 강남구 테헤란로 152",
			expected: []string{"vworld", "kakao", "naver", "google"},
		},
		{
			name:     "Korean address - Mixed",
			query:    "서울시 강남구 역삼동",
			expected: []string{"vworld", "kakao", "naver", "google"},
		},
		{
			name:     "Korean address - with numbers",
			query:    "경기도 성남시 분당구 정자일로 95",
			expected: []string{"vworld", "kakao", "naver", "google"},
		},
		{
			name:     "English address - USA",
			query:    "1600 Amphitheatre Parkway, Mountain View, CA",
			expected: []string{"google", "naver", "kakao"},
		},
		{
			name:     "English address - UK",
			query:    "10 Downing Street, London",
			expected: []string{"google", "naver", "kakao"},
		},
		{
			name:     "Numeric only",
			query:    "123-456",
			expected: []string{"google", "naver", "kakao"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := router.determineProviderOrder(tt.query)
			assert.Equal(t, tt.expected, order)
		})
	}
}

func TestSmartRouter_Geocode_Success(t *testing.T) {
	mockResult := &domain.GeoResult{
		Provider:  "kakao",
		Latitude:  37.498095,
		Longitude: 127.027610,
		Address:   "서울 강남구 테헤란로 152",
		CRS:       "WGS84",
	}

	providers := map[string]domain.Geocoder{
		"kakao": &MockGeocoder{
			name:   "kakao",
			result: mockResult,
			err:    nil,
		},
	}

	router := NewSmartRouter(providers, 5*time.Second)

	result, err := router.Geocode(context.Background(), "서울 강남구 테헤란로 152")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, mockResult, result.Data)
	assert.Contains(t, result.FallbackHistory, "kakao_success")
}

func TestSmartRouter_Geocode_Failover(t *testing.T) {
	googleResult := &domain.GeoResult{
		Provider:  "google",
		Latitude:  37.498095,
		Longitude: 127.027610,
		Address:   "서울특별시 강남구 테헤란로 152",
		CRS:       "WGS84",
	}

	providers := map[string]domain.Geocoder{
		"kakao": &MockGeocoder{
			name: "kakao",
			err:  errors.New("kakao failed"),
		},
		"vworld": &MockGeocoder{
			name: "vworld",
			err:  errors.New("vworld failed"),
		},
		"google": &MockGeocoder{
			name:   "google",
			result: googleResult,
			err:    nil,
		},
	}

	router := NewSmartRouter(providers, 5*time.Second)

	result, err := router.Geocode(context.Background(), "서울 강남구 테헤란로 152")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, googleResult, result.Data)

	// Fallback history should show failures and final success
	assert.Contains(t, result.FallbackHistory, "kakao_failed: kakao failed")
	assert.Contains(t, result.FallbackHistory, "vworld_failed: vworld failed")
	assert.Contains(t, result.FallbackHistory, "google_success")
}

func TestSmartRouter_Geocode_AllProvidersFail(t *testing.T) {
	providers := map[string]domain.Geocoder{
		"kakao": &MockGeocoder{
			name: "kakao",
			err:  errors.New("kakao failed"),
		},
		"vworld": &MockGeocoder{
			name: "vworld",
			err:  errors.New("vworld failed"),
		},
		"google": &MockGeocoder{
			name: "google",
			err:  errors.New("google failed"),
		},
	}

	router := NewSmartRouter(providers, 5*time.Second)

	result, err := router.Geocode(context.Background(), "서울 강남구 테헤란로 152")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "all providers failed")
}

func TestSmartRouter_Geocode_Timeout(t *testing.T) {
	providers := map[string]domain.Geocoder{
		"kakao": &MockGeocoder{
			name: "kakao",
			err:  domain.ErrProviderTimeout,
		},
	}

	router := NewSmartRouter(providers, 100*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := router.Geocode(ctx, "서울 강남구")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestSmartRouter_GetProviderStatus(t *testing.T) {
	providers := map[string]domain.Geocoder{
		"kakao":  &MockGeocoder{name: "kakao"},
		"google": &MockGeocoder{name: "google"},
		"vworld": &MockGeocoder{name: "vworld"},
	}

	router := NewSmartRouter(providers, 5*time.Second)
	status := router.GetProviderStatus()

	assert.Len(t, status, 3)
	assert.Equal(t, "available", status["kakao"])
	assert.Equal(t, "available", status["google"])
	assert.Equal(t, "available", status["vworld"])
}

// NewSmartRouterWithOrder Tests

func TestNewSmartRouterWithOrder_CustomKoreanOrder(t *testing.T) {
	providers := map[string]domain.Geocoder{
		"kakao":  &MockGeocoder{name: "kakao"},
		"google": &MockGeocoder{name: "google"},
		"vworld": &MockGeocoder{name: "vworld"},
		"naver":  &MockGeocoder{name: "naver"},
	}

	// Custom Korean order: kakao → google → naver → vworld
	customKoreanOrder := []string{"kakao", "google", "naver", "vworld"}
	router := NewSmartRouterWithOrder(providers, 5*time.Second, customKoreanOrder, nil)

	// Verify Korean order
	koreanOrder := router.determineProviderOrder("서울특별시 강남구")
	assert.Equal(t, customKoreanOrder, koreanOrder)

	// Verify global order uses defaults
	globalOrder := router.determineProviderOrder("New York")
	assert.Equal(t, []string{"google", "naver", "kakao"}, globalOrder)
}

func TestNewSmartRouterWithOrder_CustomGlobalOrder(t *testing.T) {
	providers := map[string]domain.Geocoder{
		"kakao":  &MockGeocoder{name: "kakao"},
		"google": &MockGeocoder{name: "google"},
		"naver":  &MockGeocoder{name: "naver"},
	}

	// Custom global order: naver → google → kakao
	customGlobalOrder := []string{"naver", "google", "kakao"}
	router := NewSmartRouterWithOrder(providers, 5*time.Second, nil, customGlobalOrder)

	// Verify global order
	globalOrder := router.determineProviderOrder("London")
	assert.Equal(t, customGlobalOrder, globalOrder)

	// Verify Korean order uses defaults
	koreanOrder := router.determineProviderOrder("서울시")
	assert.Equal(t, []string{"vworld", "kakao", "naver", "google"}, koreanOrder)
}

func TestNewSmartRouterWithOrder_BothCustomOrders(t *testing.T) {
	providers := map[string]domain.Geocoder{
		"kakao":  &MockGeocoder{name: "kakao"},
		"google": &MockGeocoder{name: "google"},
		"vworld": &MockGeocoder{name: "vworld"},
		"naver":  &MockGeocoder{name: "naver"},
	}

	customKoreanOrder := []string{"naver", "kakao", "vworld", "google"}
	customGlobalOrder := []string{"naver", "google"}

	router := NewSmartRouterWithOrder(providers, 5*time.Second, customKoreanOrder, customGlobalOrder)

	// Verify Korean order
	koreanOrder := router.determineProviderOrder("부산광역시")
	assert.Equal(t, customKoreanOrder, koreanOrder)

	// Verify global order
	globalOrder := router.determineProviderOrder("Paris, France")
	assert.Equal(t, customGlobalOrder, globalOrder)
}

func TestNewSmartRouterWithOrder_EmptyOrdersUseDefaults(t *testing.T) {
	providers := map[string]domain.Geocoder{
		"kakao":  &MockGeocoder{name: "kakao"},
		"google": &MockGeocoder{name: "google"},
		"vworld": &MockGeocoder{name: "vworld"},
		"naver":  &MockGeocoder{name: "naver"},
	}

	// Pass empty slices - should use defaults
	router := NewSmartRouterWithOrder(providers, 5*time.Second, []string{}, []string{})

	// Verify default Korean order
	koreanOrder := router.determineProviderOrder("인천광역시")
	assert.Equal(t, []string{"vworld", "kakao", "naver", "google"}, koreanOrder)

	// Verify default global order
	globalOrder := router.determineProviderOrder("Tokyo")
	assert.Equal(t, []string{"google", "naver", "kakao"}, globalOrder)
}

func TestNewSmartRouterWithOrder_CustomOrderRespectedInGeocoding(t *testing.T) {
	// Setup providers where only naver succeeds
	naverResult := &domain.GeoResult{
		Provider:  "naver",
		Latitude:  37.498095,
		Longitude: 127.027610,
		Address:   "서울 강남구 테헤란로 152",
		CRS:       "WGS84",
	}

	providers := map[string]domain.Geocoder{
		"kakao": &MockGeocoder{
			name: "kakao",
			err:  errors.New("kakao failed"),
		},
		"vworld": &MockGeocoder{
			name: "vworld",
			err:  errors.New("vworld failed"),
		},
		"naver": &MockGeocoder{
			name:   "naver",
			result: naverResult,
			err:    nil,
		},
		"google": &MockGeocoder{
			name: "google",
			err:  errors.New("google failed"),
		},
	}

	// Custom order: naver should be tried first
	customKoreanOrder := []string{"naver", "kakao", "vworld", "google"}
	router := NewSmartRouterWithOrder(providers, 5*time.Second, customKoreanOrder, nil)

	result, err := router.Geocode(context.Background(), "서울 강남구")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, naverResult, result.Data)

	// Verify naver was tried first and succeeded
	assert.Contains(t, result.FallbackHistory, "naver_success")
	assert.NotContains(t, result.FallbackHistory, "kakao")
	assert.NotContains(t, result.FallbackHistory, "vworld")
}

func TestNewSmartRouterWithOrder_NilOrdersUseDefaults(t *testing.T) {
	providers := map[string]domain.Geocoder{
		"kakao":  &MockGeocoder{name: "kakao"},
		"google": &MockGeocoder{name: "google"},
		"vworld": &MockGeocoder{name: "vworld"},
		"naver":  &MockGeocoder{name: "naver"},
	}

	// Pass nil slices - should use defaults
	router := NewSmartRouterWithOrder(providers, 5*time.Second, nil, nil)

	// Verify default Korean order
	koreanOrder := router.determineProviderOrder("대구광역시")
	assert.Equal(t, []string{"vworld", "kakao", "naver", "google"}, koreanOrder)

	// Verify default global order
	globalOrder := router.determineProviderOrder("Berlin")
	assert.Equal(t, []string{"google", "naver", "kakao"}, globalOrder)
}
