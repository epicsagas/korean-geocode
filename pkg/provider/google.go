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

// GoogleProvider는 Google Maps Geocoding API를 사용하는 공급자입니다
type GoogleProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewGoogleProvider는 새로운 Google Provider를 생성합니다
func NewGoogleProvider(apiKey string) *GoogleProvider {
	return &GoogleProvider{
		apiKey:  apiKey,
		baseURL: "https://maps.googleapis.com/maps/api/geocode/json",
		client:  &http.Client{},
	}
}

// Name은 공급자 이름을 반환합니다
func (g *GoogleProvider) Name() string {
	return "google"
}

// Geocode는 주소를 좌표로 변환합니다
func (g *GoogleProvider) Geocode(ctx context.Context, query string) (*domain.GeoResult, error) {
	// URL 생성 (EPSG:4326 WGS84 좌표계)
	reqURL := g.buildURL(query)

	// Context를 포함한 HTTP 요청 생성
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 요청 실행 (Context timeout 자동 적용)
	resp, err := g.client.Do(req)
	if err != nil {
		// Context timeout 또는 네트워크 에러
		if ctx.Err() == context.DeadlineExceeded {
			return nil, domain.ErrProviderTimeout
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// HTTP 상태 코드 확인
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// 응답 파싱
	var googleResp GoogleGeocodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&googleResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 결과 검증
	if googleResp.Status != "OK" {
		if googleResp.Status == "ZERO_RESULTS" {
			return nil, domain.ErrNoResultsFound
		}
		if googleResp.Status == "OVER_QUERY_LIMIT" {
			return nil, domain.ErrQuotaExceeded
		}
		return nil, fmt.Errorf("google api error: %s", googleResp.Status)
	}

	if len(googleResp.Results) == 0 {
		return nil, domain.ErrNoResultsFound
	}

	// 첫 번째 결과 사용
	firstResult := googleResp.Results[0]
	result := &domain.GeoResult{
		Provider:  g.Name(),
		Latitude:  firstResult.Geometry.Location.Lat,
		Longitude: firstResult.Geometry.Location.Lng,
		Address:   firstResult.FormattedAddress,
		CRS:       "WGS84",
	}

	// 좌표 유효성 검증
	if err := result.ValidateCoordinates(); err != nil {
		return nil, err
	}

	return result, nil
}

// buildURL은 Google Geocoding API URL을 생성합니다
func (g *GoogleProvider) buildURL(address string) string {
	params := url.Values{}
	params.Set("address", address)
	params.Set("key", g.apiKey)
	// EPSG:4326 (WGS84) 명시 - Google은 기본적으로 WGS84 사용
	return fmt.Sprintf("%s?%s", g.baseURL, params.Encode())
}

// GoogleGeocodeResponse는 Google Geocoding API 응답 구조체입니다
type GoogleGeocodeResponse struct {
	Results []struct {
		FormattedAddress string `json:"formatted_address"`
		Geometry         struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
	} `json:"results"`
	Status string `json:"status"`
}
