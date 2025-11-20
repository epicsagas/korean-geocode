package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/epicsagas/korean-geocode/pkg/domain"
)

// VWorldProvider는 vWorld (국토교통부) API를 사용하는 공급자입니다
type VWorldProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewVWorldProvider는 새로운 vWorld Provider를 생성합니다
func NewVWorldProvider(apiKey string) *VWorldProvider {
	return &VWorldProvider{
		apiKey:  apiKey,
		baseURL: "https://api.vworld.kr/req/address", // 명세: HTTPS 사용
		client:  &http.Client{},
	}
}

// Name은 공급자 이름을 반환합니다
func (v *VWorldProvider) Name() string {
	return "vworld"
}

// Geocode는 주소를 좌표로 변환합니다
func (v *VWorldProvider) Geocode(ctx context.Context, query string) (*domain.GeoResult, error) {
	// URL 생성 (WGS84 좌표계 명시)
	reqURL := v.buildURL(query)

	// Context를 포함한 HTTP 요청 생성
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 요청 실행
	resp, err := v.client.Do(req)
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

	// 응답 본문을 읽어서 로깅 및 파싱
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 디버깅: 응답 본문 로깅 (최대 2000자)
	if len(body) > 2000 {
		log.Printf("🔍 vWorld API response (truncated): %s...", string(body[:2000]))
	} else {
		log.Printf("🔍 vWorld API response: %s", string(body))
	}

	// 응답 파싱
	var vworldResp VWorldGeocodeResponse
	if err := json.Unmarshal(body, &vworldResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 결과 검증 및 상세 에러 처리
	if vworldResp.Response.Status != "OK" {
		// 에러 코드별 상세 메시지
		switch vworldResp.Response.Status {
		case "NOT_FOUND":
			return nil, domain.ErrNoResultsFound
		case "AUTH_ERROR":
			return nil, fmt.Errorf("vworld api key authentication failed")
		case "ACCESS_DENIED":
			return nil, fmt.Errorf("vworld api access denied (check IP/Referer restrictions)")
		case "ERROR":
			return nil, fmt.Errorf("vworld api internal error")
		default:
			return nil, fmt.Errorf("vworld api error: %s", vworldResp.Response.Status)
		}
	}

	if vworldResp.Response.Result == nil {
		return nil, domain.ErrNoResultsFound
	}

	var lat, lng float64
	var address string

	// items가 있으면 첫 번째 item 사용, 없으면 대표 point 사용
	if len(vworldResp.Response.Result.Items) > 0 {
		// items 사용 (상세 주소 포함)
		firstItem := vworldResp.Response.Result.Items[0]
		lat, err = strconv.ParseFloat(firstItem.Point.Y, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid latitude in items: %w", err)
		}
		lng, err = strconv.ParseFloat(firstItem.Point.X, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid longitude in items: %w", err)
		}
		address = firstItem.Address
	} else if vworldResp.Response.Result.Point != nil {
		// 대표 point 사용 (주소는 입력값 사용)
		lat, err = strconv.ParseFloat(vworldResp.Response.Result.Point.Y, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid latitude in point: %w", err)
		}
		lng, err = strconv.ParseFloat(vworldResp.Response.Result.Point.X, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid longitude in point: %w", err)
		}
		// items가 없으므로 입력 주소를 그대로 사용
		address = query
	} else {
		return nil, domain.ErrNoResultsFound
	}

	result := &domain.GeoResult{
		Provider:  v.Name(),
		Latitude:  lat,
		Longitude: lng,
		Address:   address,
		CRS:       "WGS84",
	}

	// 좌표 유효성 검증 (EPSG:4326 WGS84 범위)
	if err := result.ValidateCoordinates(); err != nil {
		return nil, err
	}

	return result, nil
}

// buildURL은 vWorld Geocoding API URL을 생성합니다
func (v *VWorldProvider) buildURL(address string) string {
	// 주소 유형 자동 판별 (도로명주소 vs 지번주소)
	addrType := domain.DetectAddressType(address)
	log.Printf("🔍 vWorld address type detected: %s for '%s'", addrType, address)

	params := url.Values{}
	params.Set("service", "address")
	params.Set("request", "getCoord") // 명세: 대문자 C (대소문자 구분!)
	params.Set("version", "2.0")
	params.Set("crs", "epsg:4326") // WGS84 좌표계 명시
	params.Set("address", address)
	params.Set("refine", "true")  // 정제된 주소 사용
	params.Set("simple", "false") // 상세 결과 반환
	params.Set("format", "json")
	params.Set("type", string(addrType)) // 주소 유형에 따라 자동 설정 (ROAD 또는 PARCEL)
	params.Set("key", v.apiKey)

	return fmt.Sprintf("%s?%s", v.baseURL, params.Encode())
}

// VWorldGeocodeResponse는 vWorld API 응답 구조체입니다
type VWorldGeocodeResponse struct {
	Response struct {
		Status string `json:"status"`
		Result *struct {
			CRS  string `json:"crs"`
			Type string `json:"type"`
			// 대표 좌표 (simple=true 또는 단일 결과)
			Point *struct {
				X string `json:"x"` // 경도
				Y string `json:"y"` // 위도
			} `json:"point"`
			// 상세 주소 목록 (simple=false 일 때)
			Items []struct {
				Address string `json:"address"`
				Point   struct {
					X string `json:"x"` // 경도
					Y string `json:"y"` // 위도
				} `json:"point"`
			} `json:"items"`
		} `json:"result"`
	} `json:"response"`
}
