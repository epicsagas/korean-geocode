package http

import (
	"context"
	"net/http"
	"time"

	"github.com/epicsagas/korean-geocode/internal/infrastructure/ratelimit"
)

// QuotaHandler는 Rate Limiter 쿼터 조회를 처리합니다
type QuotaHandler struct {
	limiter ratelimit.RateLimiter
}

// NewQuotaHandler는 새로운 QuotaHandler를 생성합니다
func NewQuotaHandler(limiter ratelimit.RateLimiter) *QuotaHandler {
	return &QuotaHandler{
		limiter: limiter,
	}
}

// ProviderQuota는 Provider별 쿼터 정보입니다
type ProviderQuota struct {
	Provider  string `json:"provider"`
	Usage     int    `json:"usage"`
	Remaining int    `json:"remaining"`
	Quota     int    `json:"quota"`
}

// QuotaResponse는 쿼터 조회 응답입니다
type QuotaResponse struct {
	Timestamp string          `json:"timestamp"`
	Quotas    []ProviderQuota `json:"quotas"`
}

// Handle은 쿼터 조회 요청을 처리합니다
// @Summary 쿼터 조회
// @Description Provider별 현재 쿼터 사용량을 조회합니다
// @Tags quota
// @Accept json
// @Produce json
// @Param provider query string false "특정 Provider 조회 (vworld, kakao, naver, google)"
// @Success 200 {object} QuotaResponse
// @Failure 500 {object} QuotaResponse
// @Router /v1/quota [get]
func (h *QuotaHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// 쿼리 파라미터에서 provider 추출 (선택적)
	providerParam := r.URL.Query().Get("provider")

	var providers []string
	if providerParam != "" {
		providers = []string{providerParam}
	} else {
		// 모든 provider 조회
		providers = []string{"vworld", "kakao", "naver", "google"}
	}

	quotas := make([]ProviderQuota, 0, len(providers))

	for _, provider := range providers {
		usage, err := h.limiter.GetUsage(ctx, provider)
		if err != nil {
			WriteError(w, "INTERNAL_ERROR", "Failed to retrieve usage", err.Error(), http.StatusInternalServerError)
			return
		}

		quotaLimit, err := h.limiter.GetQuota(ctx, provider)
		if err != nil {
			WriteError(w, "INTERNAL_ERROR", "Failed to retrieve quota", err.Error(), http.StatusInternalServerError)
			return
		}

		remaining := 0
		if quotaLimit > 0 {
			remaining = quotaLimit - usage
			if remaining < 0 {
				remaining = 0
			}
		}

		quota := ProviderQuota{
			Provider:  provider,
			Usage:     usage,
			Remaining: remaining,
			Quota:     quotaLimit,
		}

		quotas = append(quotas, quota)
	}

	response := QuotaResponse{
		Timestamp: time.Now().Format(time.RFC3339),
		Quotas:    quotas,
	}

	WriteJSON(w, response, http.StatusOK)
}
