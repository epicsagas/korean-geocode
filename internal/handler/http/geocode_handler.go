package http

import (
	"fmt"
	"net/http"

	"github.com/epicsagas/korean-geocode/pkg/domain"
)

// GeocodeHandler는 geocoding 요청을 처리합니다
type GeocodeHandler struct {
	router Router
}

// NewGeocodeHandler는 새로운 GeocodeHandler를 생성합니다
func NewGeocodeHandler(r Router) *GeocodeHandler {
	return &GeocodeHandler{
		router: r,
	}
}

// Handle godoc
// @Summary 주소를 좌표로 변환
// @Description 입력된 주소를 위도/경도 좌표로 변환합니다. 주소 패턴에 따라 최적의 Provider가 자동으로 선택되며, 실패 시 자동으로 다른 Provider로 Failover됩니다.
// @Tags geocoding
// @Accept json
// @Produce json
// @Param q query string true "검색할 주소 (한글 또는 영문)" example(서울특별시 강남구 테헤란로 152)
// @Success 200 {object} Response "성공적으로 좌표를 반환"
// @Failure 400 {object} Response "잘못된 요청 (쿼리 파라미터 누락)"
// @Failure 500 {object} Response "서버 내부 오류 또는 모든 Provider 실패"
// @Failure 503 {object} Response "서비스 일시 중단 (Circuit Breaker Open)"
// @Router /v1/geocode [get]
func (h *GeocodeHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// GET 메서드만 허용
	if r.Method != http.MethodGet {
		WriteError(w, "METHOD_NOT_ALLOWED", "Method not allowed", "Only GET method is supported", http.StatusMethodNotAllowed)
		return
	}

	// 쿼리 파라미터 추출
	query := r.URL.Query().Get("q")
	if query == "" {
		WriteError(w, "MISSING_PARAMETER", "Missing required parameter", "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	// Geocoding 수행
	result, err := h.router.Geocode(r.Context(), query)
	if err != nil {
		h.handleError(w, err, query)
		return
	}

	// 성공 응답
	WriteJSONWithMeta(w, result.Data, map[string]interface{}{
		"fallback_history": result.FallbackHistory,
		"query":            query,
	}, http.StatusOK)
}

// handleError는 에러를 적절한 HTTP 응답으로 변환합니다
func (h *GeocodeHandler) handleError(w http.ResponseWriter, err error, query string) {
	code := "GEOCODING_FAILED"
	message := "Failed to geocode address"
	details := fmt.Sprintf("Query: %s, Error: %v", query, err)
	status := http.StatusInternalServerError

	// Domain 에러 타입별 처리
	switch err {
	case domain.ErrInvalidAddress:
		code = "INVALID_ADDRESS"
		message = "Invalid address format"
		status = http.StatusBadRequest
	case domain.ErrProviderTimeout:
		code = "PROVIDER_TIMEOUT"
		message = "Provider request timeout"
		status = http.StatusGatewayTimeout
	case domain.ErrProviderUnavailable:
		code = "PROVIDER_UNAVAILABLE"
		message = "Provider service unavailable"
		status = http.StatusServiceUnavailable
	case domain.ErrQuotaExceeded:
		code = "QUOTA_EXCEEDED"
		message = "Provider quota exceeded"
		status = http.StatusTooManyRequests
	case domain.ErrNoResultsFound:
		code = "NO_RESULTS"
		message = "No results found for the given address"
		status = http.StatusNotFound
	}

	WriteError(w, code, message, details, status)
}
