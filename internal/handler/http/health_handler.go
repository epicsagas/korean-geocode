package http

import (
	"net/http"
)

// HealthHandler는 서버 상태 확인을 처리합니다
type HealthHandler struct {
	router Router
}

// NewHealthHandler는 새로운 HealthHandler를 생성합니다
func NewHealthHandler(r Router) *HealthHandler {
	return &HealthHandler{
		router: r,
	}
}

// Handle godoc
// @Summary 서버 상태 확인
// @Description 서버 및 모든 Provider의 상태를 확인합니다. Kubernetes Liveness/Readiness Probe에 사용할 수 있습니다.
// @Tags health
// @Produce json
// @Success 200 {object} Response "서버 정상 동작"
// @Router /health [get]
func (h *HealthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	status := h.router.GetProviderStatus()

	data := map[string]interface{}{
		"status":    "healthy",
		"providers": status,
	}

	WriteJSON(w, data, http.StatusOK)
}
