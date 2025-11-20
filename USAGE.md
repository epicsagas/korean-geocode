# 📖 사용 가이드 (Usage Guide)

다른 Go 프로젝트에서 이 패키지를 쉽게 사용하는 방법을 안내합니다.

## 🎯 빠른 시작 (Quick Start)

### 1단계: 패키지 설치

```bash
go get github.com/epicsagas/korean-geocode
```

### 2단계: .env 파일 생성

프로젝트 루트에 `.env` 파일 생성:

```env
GOOGLE_MAPS_API_KEY=your_google_api_key
KAKAO_REST_API_KEY=your_kakao_api_key
NAVER_CLIENT_ID=your_naver_client_id
NAVER_CLIENT_SECRET=your_naver_client_secret
VWORLD_API_KEY=your_vworld_api_key
```

### 3단계: 코드 작성

**최소 코드 (단 3줄!):**

```go
package main

import (
    "log"
    "github.com/gin-gonic/gin"
    "github.com/epicsagas/korean-geocode"
)

func main() {
    // 1. GeoAPI 초기화
    geoAPI, _ := geoapi.New()

    // 2. Gin 라우터 생성
    router := gin.Default()

    // 3. 라우트 등록
    geoAPI.RegisterRoutes(router)

    router.Run(":8080")
}
```

### 4단계: 실행 및 테스트

```bash
# 서버 실행
go run main.go

# 다른 터미널에서 테스트
curl "http://localhost:8080/v1/geocode?q=서울 강남구 테헤란로 152"
```

## 📋 상세 사용법

### 방법 1: 환경변수 사용 (.env 파일)

가장 권장하는 방법입니다.

```go
package main

import (
    "log"
    "github.com/gin-gonic/gin"
    "github.com/epicsagas/korean-geocode"
)

func main() {
    // .env 파일 자동 로드
    geoAPI, err := geoapi.New()
    if err != nil {
        log.Fatal(err)
    }

    router := gin.Default()
    geoAPI.RegisterRoutes(router)
    router.Run(":8080")
}
```

**엔드포인트:**
- `GET /v1/geocode?q=<주소>` - 주소 → 좌표 변환
- `GET /health` - 헬스체크

### 방법 2: 커스텀 설정

코드에서 직접 설정을 구성할 수 있습니다.

```go
package main

import (
    "log"
    "time"
    "github.com/gin-gonic/gin"
    "github.com/epicsagas/korean-geocode"
)

func main() {
    // 기본 설정 가져오기
    cfg := geoapi.DefaultConfig()

    // API 키 설정
    cfg.Google.APIKey = "your_google_key"
    cfg.Kakao.APIKey = "your_kakao_key"
    cfg.Naver.ClientID = "your_naver_client_id"
    cfg.Naver.ClientSecret = "your_naver_client_secret"
    cfg.VWorld.APIKey = "your_vworld_key"

    // 타임아웃 조정
    cfg.Server.Timeout = 10 * time.Second
    cfg.Google.Timeout = 3 * time.Second

    // Circuit Breaker 설정
    cfg.CircuitBreaker.FailureThreshold = 3
    cfg.CircuitBreaker.Timeout = 30 * time.Second

    // GeoAPI 생성
    geoAPI, err := geoapi.NewWithConfig(cfg)
    if err != nil {
        log.Fatal(err)
    }

    router := gin.Default()
    geoAPI.RegisterRoutes(router)
    router.Run(":8080")
}
```

### 방법 3: 커스텀 URL Prefix

API 엔드포인트에 prefix를 추가할 수 있습니다.

```go
router := gin.Default()

// /api prefix 추가
geoAPI.RegisterRoutesWithPrefix(router, "/api")

// 결과:
// - /api/v1/geocode
// - /api/health

router.Run(":8080")
```

### 방법 4: Gin 없이 직접 사용

HTTP 프레임워크 없이 직접 Geocoding 함수를 호출할 수 있습니다.

```go
package main

import (
    "fmt"
    "log"
    "github.com/epicsagas/korean-geocode"
)

func main() {
    geoAPI, err := geoapi.New()
    if err != nil {
        log.Fatal(err)
    }

    // 직접 Geocoding 함수 호출
    result, err := geoAPI.Geocode("서울 강남구 테헤란로 152")
    if err != nil {
        log.Printf("Geocoding failed: %v", err)
        return
    }

    fmt.Printf("Provider: %s\n", result.Data.Provider)
    fmt.Printf("위도: %f\n", result.Data.Lat)
    fmt.Printf("경도: %f\n", result.Data.Lng)
    fmt.Printf("주소: %s\n", result.Data.Address)
    fmt.Printf("Fallback History: %v\n", result.FallbackHistory)
}
```

## 🔧 설정 가능한 옵션

### Config 구조체

```go
type Config struct {
    Server struct {
        Port    string
        Timeout time.Duration
    }
    Google struct {
        APIKey  string
        Timeout time.Duration
    }
    Kakao struct {
        APIKey     string
        Timeout    time.Duration
        DailyQuota int
    }
    Naver struct {
        ClientID     string
        ClientSecret string
        Timeout      time.Duration
        DailyQuota   int
    }
    VWorld struct {
        APIKey  string
        Timeout time.Duration
    }
    CircuitBreaker struct {
        MaxRequests      uint32
        FailureThreshold uint32
        Timeout          time.Duration
    }
}
```

### 기본값

```go
cfg := geoapi.DefaultConfig()
// Server.Port: "8080"
// Server.Timeout: 5s
// Google.Timeout: 2s
// Kakao.Timeout: 2s
// Kakao.DailyQuota: 300000
// Naver.Timeout: 2s
// Naver.DailyQuota: 100000
// VWorld.Timeout: 2s
// CircuitBreaker.MaxRequests: 10
// CircuitBreaker.FailureThreshold: 5
// CircuitBreaker.Timeout: 60s
```

## 📦 응답 형식

### 성공 응답

```json
{
  "data": {
    "provider": "kakao",
    "lat": 37.498095,
    "lng": 127.027610,
    "address": "서울 강남구 테헤란로 152",
    "crs": "WGS84"
  },
  "meta": [
    "kakao_success"
  ]
}
```

### Fallback 발생 시

```json
{
  "data": {
    "provider": "google",
    "lat": 37.498095,
    "lng": 127.027610,
    "address": "서울특별시 강남구 테헤란로 152",
    "crs": "WGS84"
  },
  "meta": [
    "kakao_failed: rate limit exceeded",
    "vworld_failed: timeout",
    "google_success"
  ]
}
```

### 에러 응답

```json
{
  "error": "Geocoding failed: all providers failed"
}
```

## 🎓 예제 프로젝트

`examples/` 디렉토리에서 실제 작동하는 예제를 확인할 수 있습니다:

1. **simple-gin**: 가장 기본적인 통합 예제
2. **custom-config**: 커스텀 설정 사용 예제

각 예제는 독립적으로 실행 가능합니다:

```bash
cd examples/simple-gin
cp .env.example .env
# .env 파일 편집
go run main.go
```

## ⚠️ 주의사항

1. **API 키 관리**: 프로덕션 환경에서는 환경변수나 비밀 관리 시스템 사용 권장
2. **모듈 경로**: `go.mod`에서 `github.com/epicsagas/korean-geocode`를 실제 경로로 변경
3. **최소 요구사항**: Go 1.23 이상
4. **Rate Limiting**:
   - Kakao API: 일일 30만건 제한 (기본값)
   - Naver API: 일일 10만건 제한 (기본값)
   - vWorld: 무제한 (공공 API)

## 🔗 더 보기

- [README.md](./README.md) - 전체 프로젝트 문서
- [examples/](./examples/) - 실제 작동하는 예제
- [Technical-Whitepaper.md](./Technical-Whitepaper.md) - 기술 백서
