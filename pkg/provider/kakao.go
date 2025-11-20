package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/epicsagas/korean-geocode/pkg/domain"
)

// KakaoProvider는 Kakao Local API를 사용하는 공급자입니다
type KakaoProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewKakaoProvider는 새로운 Kakao Provider를 생성합니다
func NewKakaoProvider(apiKey string) *KakaoProvider {
	return &KakaoProvider{
		apiKey:  apiKey,
		baseURL: "https://dapi.kakao.com/v2/local/search/address.json",
		client:  &http.Client{},
	}
}

// Name은 공급자 이름을 반환합니다
func (k *KakaoProvider) Name() string {
	return "kakao"
}

// Geocode는 주소를 좌표로 변환합니다
func (k *KakaoProvider) Geocode(ctx context.Context, query string) (*domain.GeoResult, error) {
	// URL 생성
	reqURL := k.buildURL(query)

	// Context를 포함한 HTTP 요청 생성
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Kakao REST API 키 헤더 추가
	req.Header.Set("Authorization", fmt.Sprintf("KakaoAK %s", k.apiKey))

	// 요청 실행
	resp, err := k.client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, domain.ErrProviderTimeout
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// HTTP 상태 코드 확인
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, domain.ErrQuotaExceeded
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// 응답 파싱
	var kakaoResp KakaoGeocodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&kakaoResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 결과 검증
	if len(kakaoResp.Documents) == 0 {
		return nil, domain.ErrNoResultsFound
	}

	// 첫 번째 결과 사용
	firstResult := kakaoResp.Documents[0]

	// 도로명 주소가 있으면 우선, 없으면 지번 주소 사용
	address := firstResult.RoadAddress.AddressName
	if address == "" {
		address = firstResult.Address.AddressName
	}

	// Kakao는 WGS84 좌표계를 사용합니다
	result := &domain.GeoResult{
		Provider:  k.Name(),
		Latitude:  firstResult.Y,
		Longitude: firstResult.X,
		Address:   address,
		CRS:       "WGS84",
	}

	// 좌표 유효성 검증
	if err := result.ValidateCoordinates(); err != nil {
		return nil, err
	}

	return result, nil
}

// buildURL은 Kakao Local API URL을 생성합니다
func (k *KakaoProvider) buildURL(address string) string {
	params := url.Values{}
	params.Set("query", address)
	return fmt.Sprintf("%s?%s", k.baseURL, params.Encode())
}

// KakaoGeocodeResponse는 Kakao Local API 응답 구조체입니다
type KakaoGeocodeResponse struct {
	Documents []struct {
		Address struct {
			AddressName string `json:"address_name"`
		} `json:"address"`
		RoadAddress struct {
			AddressName string `json:"address_name"`
		} `json:"road_address"`
		X float64 `json:"x,string"` // 경도 (WGS84)
		Y float64 `json:"y,string"` // 위도 (WGS84)
	} `json:"documents"`
	Meta struct {
		TotalCount int `json:"total_count"`
	} `json:"meta"`
}
