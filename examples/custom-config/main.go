package main

import (
	"log"
	"time"

	"github.com/epicsagas/korean-geocode/pkg/geoapi"
	"github.com/gin-gonic/gin"
)

func main() {
	// 커스텀 설정으로 GeoAPI 초기화
	cfg := geoapi.DefaultConfig()

	// API 키 설정 (환경변수 대신)
	cfg.Google.APIKey = "your_google_api_key"
	cfg.Kakao.APIKey = "your_kakao_api_key"
	cfg.VWorld.APIKey = "your_vworld_api_key"

	// 타임아웃 커스터마이징
	cfg.Server.Timeout = 10 * time.Second
	cfg.Google.Timeout = 3 * time.Second
	cfg.Kakao.Timeout = 3 * time.Second

	// Circuit Breaker 설정 조정
	cfg.CircuitBreaker.FailureThreshold = 3
	cfg.CircuitBreaker.Timeout = 30 * time.Second

	// GeoAPI 생성
	geoAPI, err := geoapi.NewWithConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize GeoAPI: %v", err)
	}

	// Gin 라우터 생성
	router := gin.Default()

	// 커스텀 prefix와 함께 라우트 등록
	geoAPI.RegisterRoutesWithPrefix(router, "/api")

	// 루트 경로
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Geocoding API with custom configuration",
			"endpoints": map[string]string{
				"geocode": "/api/v1/geocode?q=<address>",
				"health":  "/api/health",
			},
			"config": map[string]interface{}{
				"server_timeout":  cfg.Server.Timeout.String(),
				"circuit_breaker": cfg.CircuitBreaker.FailureThreshold,
			},
		})
	})

	// 서버 실행
	log.Println("🚀 Server starting on :8080")
	log.Println("📍 Geocoding: http://localhost:8080/api/v1/geocode?q=서울 강남구 테헤란로 152")
	log.Println("💚 Health: http://localhost:8080/api/health")

	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
