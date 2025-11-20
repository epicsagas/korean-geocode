package geoapi

import (
	"net/http"

	"github.com/epicsagas/korean-geocode/pkg/router"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes는 Gin 라우터에 geocoding 엔드포인트를 등록합니다
// 기본 경로는 /v1/geocode와 /health입니다
func (api *GeoAPI) RegisterRoutes(r *gin.Engine) {
	api.RegisterRoutesWithPrefix(r, "")
}

// RegisterRoutesWithPrefix는 지정된 prefix와 함께 Gin 라우터에 엔드포인트를 등록합니다
// 예: prefix가 "/api"인 경우, 엔드포인트는 /api/v1/geocode가 됩니다
func (api *GeoAPI) RegisterRoutesWithPrefix(r *gin.Engine, prefix string) {
	v1 := r.Group(prefix + "/v1")
	{
		v1.GET("/geocode", api.geocodeHandler())
	}

	r.GET(prefix+"/health", api.healthHandler())
}

// geocodeHandler는 주소를 좌표로 변환하는 Gin 핸들러입니다
func (api *GeoAPI) geocodeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Query("q")
		if query == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "MISSING_PARAMETER",
					"message": "Missing required parameter",
					"details": "Query parameter 'q' is required",
				},
			})
			return
		}

		result, err := api.Router.Geocode(c.Request.Context(), query)
		if err != nil {
			api.handleGinError(c, err, query)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    result.Data,
			"meta": gin.H{
				"fallback_history": result.FallbackHistory,
				"query":            query,
			},
		})
	}
}

// healthHandler는 서버 및 Provider 상태를 확인하는 Gin 핸들러입니다
func (api *GeoAPI) healthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		status := api.Router.GetProviderStatus()
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"status":    "healthy",
				"providers": status,
			},
		})
	}
}

// handleGinError는 Gin 컨텍스트에서 에러를 처리합니다
func (api *GeoAPI) handleGinError(c *gin.Context, err error, query string) {
	// Domain 에러에 따른 적절한 HTTP 상태 코드와 메시지 반환
	// 간단한 에러 처리 로직
	c.JSON(http.StatusInternalServerError, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "GEOCODING_FAILED",
			"message": "Failed to geocode address",
			"details": err.Error(),
		},
	})
}

// GeocodeResponse는 geocoding 응답 구조체입니다 (하위 호환성)
type GeocodeResponse struct {
	Data            interface{} `json:"data"`
	FallbackHistory []string    `json:"meta"`
}

// Geocode는 직접 geocoding을 수행하는 헬퍼 함수입니다 (Gin 없이 사용 가능)
func (api *GeoAPI) Geocode(query string) (*router.GeoRouteResult, error) {
	return api.Router.Geocode(nil, query)
}
