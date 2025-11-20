package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/epicsagas/korean-geocode/pkg/domain"
)

// NaverProvider는 Naver Cloud Platform Maps API를 사용하는 공급자입니다
type NaverProvider struct {
	clientID     string
	clientSecret string
	baseURL      string
	client       *http.Client
}

// NaverGeocodeResponse는 Naver Maps Geocoding API 응답 구조입니다
type NaverGeocodeResponse struct {
	Status       string `json:"status"`
	ErrorMessage string `json:"errorMessage"`
	Meta         struct {
		TotalCount int `json:"totalCount"`
		Page       int `json:"page"`
		Count      int `json:"count"`
	} `json:"meta"`
	Addresses []struct {
		RoadAddress    string  `json:"roadAddress"`
		JibunAddress   string  `json:"jibunAddress"`
		EnglishAddress string  `json:"englishAddress"`
		X              string  `json:"x"` // 경도
		Y              string  `json:"y"` // 위도
		Distance       float64 `json:"distance"`
	} `json:"addresses"`
}

// NewNaverProvider는 새로운 Naver Provider를 생성합니다
func NewNaverProvider(clientID, clientSecret string) *NaverProvider {
	return &NaverProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		baseURL:      "https://naveropenapi.apigw.ntruss.com/map-geocode/v2/geocode",
		client:       &http.Client{},
	}
}

// Name은 공급자 이름을 반환합니다
func (n *NaverProvider) Name() string {
	return "naver"
}

// Geocode는 주소를 좌표로 변환합니다
func (n *NaverProvider) Geocode(ctx context.Context, query string) (*domain.GeoResult, error) {
	// URL 생성
	reqURL := n.buildURL(query)

	// Context를 포함한 HTTP 요청 생성
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Naver Cloud Platform API 헤더 추가
	req.Header.Set("x-ncp-apigw-api-key-id", n.clientID)
	req.Header.Set("x-ncp-apigw-api-key", n.clientSecret)
	req.Header.Set("Accept", "application/json")

	// 요청 실행
	resp, err := n.client.Do(req)
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
	var naverResp NaverGeocodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&naverResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 에러 메시지 확인
	if naverResp.ErrorMessage != "" {
		return nil, fmt.Errorf("naver api error: %s", naverResp.ErrorMessage)
	}

	// 결과 검증
	if len(naverResp.Addresses) == 0 {
		return nil, domain.ErrNoResultsFound
	}

	// 첫 번째 결과 사용
	firstResult := naverResp.Addresses[0]

	// 좌표 변환 (string to float64)
	longitude, err := strconv.ParseFloat(firstResult.X, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid longitude format: %w", err)
	}

	latitude, err := strconv.ParseFloat(firstResult.Y, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid latitude format: %w", err)
	}

	// 주소 선택 (도로명 주소 우선, 없으면 지번 주소)
	address := firstResult.RoadAddress
	if address == "" {
		address = firstResult.JibunAddress
	}

	return &domain.GeoResult{
		Provider:  "naver",
		Latitude:  latitude,
		Longitude: longitude,
		Address:   address,
		CRS:       "WGS84", // Naver Maps는 WGS84 좌표계 사용
	}, nil
}

// buildURL은 API 요청 URL을 생성합니다
func (n *NaverProvider) buildURL(query string) string {
	params := url.Values{}
	params.Set("query", query)

	// 한글이 없으면 영문 주소로 간주하여 language=eng 추가
	koreanPattern := regexp.MustCompile(`[\x{AC00}-\x{D7A3}\x{1100}-\x{11FF}\x{3130}-\x{318F}]`)
	if !koreanPattern.MatchString(query) {
		params.Set("language", "eng")
	}

	return fmt.Sprintf("%s?%s", n.baseURL, params.Encode())
}
