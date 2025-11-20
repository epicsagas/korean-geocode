package http

import (
	"net/http"

	"github.com/epicsagas/korean-geocode/internal/handler/middleware"
	"github.com/epicsagas/korean-geocode/pkg/router"
	httpSwagger "github.com/swaggo/http-swagger"
)

// SetupRoutes는 HTTP 라우트를 설정합니다
func SetupRoutes(smartRouter *router.SmartRouter) http.Handler {
	mux := http.NewServeMux()

	// 핸들러 초기화
	geocodeHandler := NewGeocodeHandler(smartRouter)
	healthHandler := NewHealthHandler(smartRouter)

	// 라우트 등록
	mux.HandleFunc("/v1/geocode", geocodeHandler.Handle)
	mux.HandleFunc("/health", healthHandler.Handle)

	// Swagger UI
	mux.HandleFunc("/swagger/", httpSwagger.WrapHandler)
	mux.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})

	// 미들웨어 체인 적용
	handler := middleware.Recovery(mux)
	handler = middleware.Logging(handler)
	handler = middleware.CORS(middleware.DefaultCORSConfig())(handler)

	return handler
}
