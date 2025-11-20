package main

import (
	"log"

	"github.com/epicsagas/korean-geocode/pkg/geoapi"
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. GeoAPI 초기화 (.env 파일 자동 로드)
	geoAPI, err := geoapi.New()
	if err != nil {
		log.Fatalf("Failed to initialize GeoAPI: %v", err)
	}

	// 2. Gin 라우터 생성
	router := gin.Default()

	// 3. Geocoding 엔드포인트 등록
	geoAPI.RegisterRoutes(router)

	// 추가 커스텀 라우트 예시
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Geocoding API is running",
			"endpoints": map[string]string{
				"geocode": "/v1/geocode?q=<address>",
				"health":  "/health",
			},
		})
	})

	// 서버 실행
	log.Println("🚀 Server starting on :8080")
	log.Println("📍 Geocoding: http://localhost:8080/v1/geocode?q=서울 강남구 테헤란로 152")
	log.Println("💚 Health: http://localhost:8080/health")

	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
