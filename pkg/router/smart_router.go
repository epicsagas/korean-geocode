package router

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/epicsagas/korean-geocode/pkg/domain"
)

// SmartRouter는 지능형 라우터로 주소 패턴에 따라 적절한 Provider를 선택합니다
type SmartRouter struct {
	providers           map[string]domain.Geocoder
	timeout             time.Duration
	koreanProviderOrder []string // 한국 주소용 프로바이더 순서
	globalProviderOrder []string // 해외 주소용 프로바이더 순서
}

// NewSmartRouter는 새로운 Smart Router를 생성합니다
func NewSmartRouter(providers map[string]domain.Geocoder, timeout time.Duration) *SmartRouter {
	return &SmartRouter{
		providers:           providers,
		timeout:             timeout,
		koreanProviderOrder: []string{"vworld", "kakao", "naver", "google"}, // 기본값
		globalProviderOrder: []string{"google", "naver", "kakao"},           // 기본값
	}
}

// NewSmartRouterWithOrder는 프로바이더 순서를 지정하여 Smart Router를 생성합니다
func NewSmartRouterWithOrder(providers map[string]domain.Geocoder, timeout time.Duration, koreanOrder, globalOrder []string) *SmartRouter {
	// 빈 슬라이스인 경우 기본값 사용
	if len(koreanOrder) == 0 {
		koreanOrder = []string{"vworld", "kakao", "naver", "google"}
	}
	if len(globalOrder) == 0 {
		globalOrder = []string{"google", "naver", "kakao"}
	}

	return &SmartRouter{
		providers:           providers,
		timeout:             timeout,
		koreanProviderOrder: koreanOrder,
		globalProviderOrder: globalOrder,
	}
}

// GeoRouteResult는 라우팅 결과와 메타데이터를 포함합니다
type GeoRouteResult struct {
	Data            *domain.GeoResult `json:"data"`
	FallbackHistory []string          `json:"meta"`
}

// Geocode는 주소 패턴을 분석하여 최적의 Provider 순서로 요청을 시도합니다
func (r *SmartRouter) Geocode(ctx context.Context, query string) (*GeoRouteResult, error) {
	// Context timeout 설정
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// 주소 정제: 괄호 안 동 정보 제거, 공백 정리
	cleanedQuery, _ := domain.ParseAddress(query)
	log.Printf("📝 Address cleaned: '%s' → '%s'", query, cleanedQuery)

	// 주소 패턴 분석 및 Provider 순서 결정
	providerOrder := r.determineProviderOrder(cleanedQuery)

	// Fallback 히스토리 추적
	var fallbackHistory []string

	// Provider 순서대로 시도
	for _, providerName := range providerOrder {
		provider, exists := r.providers[providerName]
		if !exists {
			log.Printf("⚠️  Provider '%s' not configured", providerName)
			fallbackHistory = append(fallbackHistory, fmt.Sprintf("%s_not_configured", providerName))
			continue
		}

		// Provider에 개별 timeout 설정 (2초)
		providerCtx, providerCancel := context.WithTimeout(ctx, 2*time.Second)
		result, err := provider.Geocode(providerCtx, cleanedQuery)
		providerCancel()

		if err != nil {
			// 실패 기록 및 상세 로깅
			log.Printf("❌ Provider '%s' failed for query '%s': %v", providerName, cleanedQuery, err)
			fallbackHistory = append(fallbackHistory, fmt.Sprintf("%s_failed: %v", providerName, err))
			continue
		}

		// 성공
		log.Printf("✅ Provider '%s' succeeded for query '%s'", providerName, cleanedQuery)
		fallbackHistory = append(fallbackHistory, fmt.Sprintf("%s_success", providerName))
		return &GeoRouteResult{
			Data:            result,
			FallbackHistory: fallbackHistory,
		}, nil
	}

	// 모든 Provider 실패
	return nil, fmt.Errorf("all providers failed: %v", fallbackHistory)
}

// determineProviderOrder는 주소 패턴에 따라 Provider 우선순위를 결정합니다
func (r *SmartRouter) determineProviderOrder(query string) []string {
	// 한글 패턴 감지 (유니코드 범위: 가-힣, ㄱ-ㅎ, ㅏ-ㅣ)
	koreanPattern := regexp.MustCompile(`[\x{AC00}-\x{D7A3}\x{1100}-\x{11FF}\x{3130}-\x{318F}]`)
	isKorean := koreanPattern.MatchString(query)

	if isKorean {
		// 국내 주소: 설정된 순서 사용 (기본: vWorld → Kakao → Naver → Google)
		return r.koreanProviderOrder
	}

	// 해외 주소: 설정된 순서 사용 (기본: Google → Kakao)
	return r.globalProviderOrder
}

// GetProviderStatus는 모든 Provider의 상태를 반환합니다 (헬스체크용)
func (r *SmartRouter) GetProviderStatus() map[string]string {
	status := make(map[string]string)
	for name := range r.providers {
		status[name] = "available"
	}
	return status
}
