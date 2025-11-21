# Korean Geocoder

[![Go Version](https://img.shields.io/badge/Go-1.23%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Go Reference](https://pkg.go.dev/badge/github.com/epicsagas/korean-geocode.svg)](https://pkg.go.dev/github.com/epicsagas/korean-geocode)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/epicsagas/korean-geocode)](https://goreportcard.com/report/github.com/epicsagas/korean-geocode)

**한국 주소에 최적화된 하이브리드 Geocoding 라이브러리** - Google Maps, Kakao Local, Naver Maps, vWorld API를 통합하여 비용 효율적이고 고가용성을 제공합니다.

🌍 **[English](#english)** | 🇰🇷 **한국어**

---

## 🎯 주요 특징

- 🇰🇷 **한국 주소 최적화**: vWorld → Kakao → Naver → Google 우선순위로 무료/저비용 API 우선 사용 (⚠️ Naver Maps는 새로운 맵스 API로 전환, 월 600건 무료)
- 🌏 **글로벌 주소 지원**: Google Maps 및 Naver Maps (영문 지원)로 전 세계 주소 처리 (⚠️ Naver Maps는 새로운 맵스 API로 전환, 월 600건 무료)
- 💰 **비용 최적화**: 무료 API를 우선 사용하여 운영 비용 절감 (월 최대 99% 절감)
- 🔄 **자동 Failover**: Provider 장애 시 자동으로 다음 Provider로 전환
- ⚡ **Circuit Breaker**: 장애 격리 및 자동 복구로 시스템 안정성 확보
- 📊 **Rate Limiting**: 일일 쿼터 관리로 예상치 못한 비용 방지
- 🎯 **Smart Routing**: 한글/영문 자동 감지 및 최적 Provider 선택
- 🗺️ **WGS84 통일**: 모든 좌표를 WGS84 (EPSG:4326)로 정규화
- 🔌 **Gin 통합**: 단 3줄의 코드로 기존 Gin 프로젝트에 통합

## 📦 설치

```bash
go get github.com/epicsagas/korean-geocode
```

**최소 요구사항**: Go 1.23 이상

## 🚀 빠른 시작

### 1. 환경변수 설정

`.env` 파일 생성:

```env
# 최소 하나 이상의 API 키 필요
GOOGLE_MAPS_API_KEY=your_google_maps_api_key
KAKAO_REST_API_KEY=your_kakao_rest_api_key
NAVER_CLIENT_ID=your_naver_client_id
NAVER_CLIENT_SECRET=your_naver_client_secret
VWORLD_API_KEY=your_vworld_api_key
```

### 2. Gin 프로젝트에 통합 (3줄!)

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/epicsagas/korean-geocode"
)

func main() {
    geoAPI, _ := geoapi.New()              // 1. 초기화
    router := gin.Default()                 // 2. Gin 라우터
    geoAPI.RegisterRoutes(router)          // 3. 라우트 등록
    router.Run(":8080")
}
```

### 3. 사용

```bash
# 한국 주소 검색
curl "http://localhost:8080/v1/geocode?q=서울특별시 강남구 테헤란로 152"

# 해외 주소 검색
curl "http://localhost:8080/v1/geocode?q=1600 Amphitheatre Parkway, Mountain View, CA"

# 헬스체크
curl "http://localhost:8080/health"
```

## 📖 사용 예제

### 예제 1: 기본 사용 (.env 파일)

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

### 예제 2: 커스텀 설정

```go
package main

import (
    "log"
    "time"
    "github.com/gin-gonic/gin"
    "github.com/epicsagas/korean-geocode"
)

func main() {
    // 커스텀 설정
    cfg := geoapi.DefaultConfig()
    cfg.Google.APIKey = "your_google_key"
    cfg.Kakao.APIKey = "your_kakao_key"
    cfg.Naver.ClientID = "your_naver_client_id"
    cfg.Naver.ClientSecret = "your_naver_client_secret"
    cfg.VWorld.APIKey = "your_vworld_api_key"
    cfg.Server.Timeout = 10 * time.Second
    cfg.CircuitBreaker.FailureThreshold = 3

    geoAPI, err := geoapi.NewWithConfig(cfg)
    if err != nil {
        log.Fatal(err)
    }

    router := gin.Default()
    geoAPI.RegisterRoutes(router)
    router.Run(":8080")
}
```

### 예제 3: 커스텀 URL Prefix

```go
router := gin.Default()

// /api prefix 추가
geoAPI.RegisterRoutesWithPrefix(router, "/api")

// 결과:
// - /api/v1/geocode
// - /api/health

router.Run(":8080")
```

### 예제 4: Gin 없이 직접 사용

```go
package main

import (
    "fmt"
    "log"
    "github.com/epicsagas/korean-geocode/pkg/geoapi"
)

func main() {
    geoAPI, _ := geoapi.New()

    // 직접 Geocoding 함수 호출
    result, err := geoAPI.Geocode("서울 강남구 테헤란로 152")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("위도: %f, 경도: %f\n", result.Data.Lat, result.Data.Lng)
    fmt.Printf("Provider: %s\n", result.Data.Provider)
}
```

### 예제 5: 고급 사용 - 커스텀 Provider 추가

```go
package main

import (
    "context"
    "time"
    "github.com/epicsagas/korean-geocode/pkg/domain"
    "github.com/epicsagas/korean-geocode/pkg/provider"
    "github.com/epicsagas/korean-geocode/pkg/router"
)

// 커스텀 Provider 구현
type MyCustomProvider struct {
    apiKey string
}

func (p *MyCustomProvider) Name() string {
    return "my-custom"
}

func (p *MyCustomProvider) Geocode(ctx context.Context, query string) (*domain.GeoResult, error) {
    // 커스텀 로직 구현
    return &domain.GeoResult{
        Provider:  "my-custom",
        Latitude:  37.498095,
        Longitude: 127.027610,
        Address:   query,
        CRS:       "WGS84",
    }, nil
}

func main() {
    // 기본 Provider 초기화
    kakao := provider.NewKakaoProvider("your_kakao_key")
    google := provider.NewGoogleProvider("your_google_key")
    custom := &MyCustomProvider{apiKey: "your_custom_key"}

    // Provider Map 구성
    providers := map[string]domain.Geocoder{
        "kakao":  kakao,
        "google": google,
        "custom": custom,  // 커스텀 Provider 추가
    }

    // SmartRouter 직접 생성
    smartRouter := router.NewSmartRouter(providers, 5*time.Second)

    // 사용
    result, _ := smartRouter.Geocode(context.Background(), "서울 강남구")
    fmt.Printf("Provider: %s\n", result.Data.Provider)
}
```

### 예제 6: 고급 사용 - Provider 직접 선택

```go
package main

import (
    "context"
    "github.com/epicsagas/korean-geocode/pkg/provider"
)

func main() {
    // Kakao Provider만 직접 사용
    kakao := provider.NewKakaoProvider("your_kakao_key")

    result, err := kakao.Geocode(context.Background(), "서울시 강남구")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("좌표: %f, %f\n", result.Latitude, result.Longitude)
}
```

## 🏗️ 아키텍처

```
[Client Request]
      │
      ▼
[Smart Router] ─── 한글 감지 → vWorld → Kakao → Naver → Google
                   영문 감지 → Google → Naver (language=eng) → Kakao
      │
      ▼
[Circuit Breaker & Rate Limiter (SQLite/Redis)]
      │
      ▼
[Provider Pool]
 ├─ Google Maps (글로벌, 유료)
 ├─ Kakao Local (한국, 무료 10만건/일)
 ├─ Naver Maps (한국+영문, 월 600건 무료, 이후 건당 0.1원) ⚠️ 새로운 맵스 API로 전환
 └─ vWorld (한국, 무료 공공 API, 일 4만건)
```

## 📊 Provider 선택 전략

### 한국 주소 (한글 포함)
1. **vWorld** (무료 공공 API, 일 4만건) ✅ 1순위
2. **Kakao Local** (무료 10만건/일) ✅ 2순위
3. **Naver Maps** ⚠️ (월 600건 무료, 이후 건당 0.1원) - 새로운 맵스 API로 전환
4. **Google Maps** (유료) ⚠️ 최후 수단

### 해외 주소 (영문)
1. **Google Maps** (글로벌 커버리지 우수) ✅ 1순위
2. **Naver Maps** ⚠️ (월 600건 무료, 이후 건당 0.1원) - 새로운 맵스 API로 전환
3. **Kakao Local** (보조) ✅ 3순위

### 🎛️ Provider 순서 커스터마이징

`.env` 파일에서 자유롭게 조정 가능:

```env
# 한국 주소용 (기본: vworld,kakao,naver,google)
# ⚠️ Naver Maps는 새로운 맵스 API로 전환 (월 600건 무료, 이후 건당 0.1원)
KOREAN_PROVIDER_ORDER=kakao,naver,vworld,google

# 해외 주소용 (기본: google,naver,kakao)
# ⚠️ Naver Maps는 새로운 맵스 API로 전환 (월 600건 무료, 이후 건당 0.1원)
GLOBAL_PROVIDER_ORDER=google,kakao
```

**✨ 사용 예시:**
- 비용 최우선: `vworld,kakao,naver,google`
- 속도 최우선: `google,naver,kakao,vworld`
- Kakao 중심: `kakao,vworld,naver,google`

## 📚 Provider 공식 문서

각 Provider의 상세한 API 사양 및 제약사항은 공식 문서를 참조하세요:

- **vWorld**: [Geocoder API 가이드](https://www.vworld.kr/dev/v4dv_geocoderguide2_s001.do) - 공공 데이터 무료 API, 일 4만건
- **Kakao Local**: [로컬 API 개발 가이드](https://developers.kakao.com/docs/latest/ko/local/dev-guide) - 일 10만건 무료
- **Naver Maps**: [Geocoding API 명세](https://api.ncloud-docs.com/docs/ai-naver-mapsgeocoding-geocode) - ⚠️ [새로운 맵스 API로 전환 공지](https://www.gov-ncloud.com/v2/support/notice/all/499) (월 600건 무료, 이후 건당 0.1원)
- **Google Maps**: [Geocoding API Overview](https://developers.google.com/maps/documentation/geocoding/overview) - 월 $200 크레딧

## 🔧 API 명세

### `GET /v1/geocode`

주소를 좌표로 변환합니다.

**Query Parameters:**
- `q` (required): 검색할 주소

**Response (성공):**

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

**Response (Fallback 발생):**

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

### `GET /health`

서버 및 Provider 상태를 확인합니다.

**Response:**

```json
{
  "status": "healthy",
  "providers": {
    "google": "available",
    "kakao": "available",
    "naver": "available",
    "vworld": "available"
  }
}
```

## ⚙️ 환경변수

| 변수 | 설명 | 기본값 | 필수 |
|------|------|--------|------|
| **API Keys** |||
| `GOOGLE_MAPS_API_KEY` | Google Maps API 키 | - | * |
| `KAKAO_REST_API_KEY` | Kakao REST API 키 | - | * |
| `NAVER_CLIENT_ID` | Naver Cloud Platform Client ID | - | * |
| `NAVER_CLIENT_SECRET` | Naver Cloud Platform Client Secret | - | * |
| `VWORLD_API_KEY` | vWorld API 키 | - | * |
| **Server** |||
| `SERVER_PORT` | 서버 포트 | `8080` | ✗ |
| `SERVER_TIMEOUT` | 요청 타임아웃 | `5s` | ✗ |
| **Rate Limiting** |||
| `KAKAO_DAILY_QUOTA` | Kakao 일일 쿼터 | `100000` | ✗ |
| `NAVER_DAILY_QUOTA` | Naver 일일 쿼터 | `100000` | ✗ |
| `REDIS_HOST` | Redis 호스트 (선택, 비어있으면 SQLite 사용) | `localhost` | ✗ |
| `REDIS_PORT` | Redis 포트 | `6379` | ✗ |
| **Provider Order** 🆕 |||
| `KOREAN_PROVIDER_ORDER` | 한국 주소 Provider 순서 | `vworld,kakao,naver,google` | ✗ |
| `GLOBAL_PROVIDER_ORDER` | 해외 주소 Provider 순서 | `google,naver,kakao` | ✗ |
| **Circuit Breaker** |||
| `CIRCUIT_BREAKER_FAILURE_THRESHOLD` | 실패 임계값 | `5` | ✗ |
| `CIRCUIT_BREAKER_TIMEOUT` | Circuit Open 유지 시간 | `60s` | ✗ |

\* 최소 하나 이상의 API 키 필요 (Naver는 Client ID + Secret 둘 다 필요)

## 🛡️ Circuit Breaker 동작

- **모니터링 간격**: 10초
- **실패 임계값**: 최근 10회 요청 중 5회 이상 실패
- **Open 유지 시간**: 60초
- **상태 전이**: Closed → Open → Half-Open → Closed

## 💾 Rate Limiter 선택 가이드 (SQLite vs Redis)

### SQLite (기본값) 🆕
**언제 사용:**
- 단일 서버 운영
- 간단한 배포 환경
- Redis 설치/관리 부담 제거
- 데이터 영구 저장 필요

**장점:**
- ✅ 설정 불필요 (자동 생성)
- ✅ 외부 의존성 없음
- ✅ 서버 재시작 시 쿼터 유지
- ✅ CGO-free 순수 Go 구현

**제약:**
- ⚠️ 단일 인스턴스만 지원
- ⚠️ 파일 기반 (./data/ratelimit.db)

```env
# SQLite 사용 (REDIS_HOST 비어있거나 localhost)
REDIS_HOST=
# 또는
REDIS_HOST=localhost
```

### Redis (권장: 프로덕션)
**언제 사용:**
- 다중 서버 환경 (수평 확장)
- 높은 동시성 처리
- 분산 Rate Limiting 필요

**장점:**
- ✅ 다중 인스턴스 지원
- ✅ 높은 성능과 동시성
- ✅ 클러스터 환경 최적

**요구사항:**
- Redis 서버 설치 및 운영

```env
# Redis 사용
REDIS_HOST=redis.example.com
REDIS_PORT=6379
REDIS_PASSWORD=your_password
```

### 선택 기준 요약

| 항목 | SQLite | Redis |
|------|--------|-------|
| **설정 복잡도** | 낮음 ⭐ | 중간 ⭐⭐ |
| **배포 환경** | 단일 서버 | 다중 서버 |
| **성능** | 중간 | 높음 |
| **데이터 영속성** | 파일 저장 | 메모리/영속화 |
| **확장성** | 수직 확장 | 수평 확장 |
| **권장 시나리오** | 개발/소규모 | 프로덕션/대규모 |

## 💰 비용 최적화 효과

### Before (Google Maps만 사용)
- 월 1,000,000건 요청
- 무료: 200 USD 크레딧
- **비용: ~600 USD/월**

### After (Korean Geocoder 사용)
- 한국 주소 90%: vWorld/Kakao (무료) + Naver (월 600건 무료, 이후 건당 0.1원)
- 해외 주소 10%: Google Maps (대부분 무료)
- **비용: ~6 USD/월** ✅ **99% 절감**

## 📁 프로젝트 구조

```
korean-geocode/
├── cmd/                   # Application entry points
│   └── api/main.go       # HTTP server
│
├── pkg/                   # 🔓 Public API (외부에서 import 가능)
│   ├── domain/           # 핵심 인터페이스 (Geocoder, GeoResult)
│   ├── provider/         # Provider 구현 (Google, Kakao, Naver, vWorld)
│   ├── router/           # SmartRouter (고급 사용)
│   └── geoapi/           # 고수준 통합 API (일반 사용)
│
├── internal/              # 🔒 Private (내부 전용)
│   ├── handler/          # HTTP 핸들러
│   └── infrastructure/   # Circuit Breaker, Config, Rate Limiter
│
├── examples/              # 예제 프로젝트
│   ├── simple-gin/       # 기본 통합
│   └── custom-config/    # 고급 커스터마이징
│
└── docs/                  # 문서 및 아키텍처 가이드
```

**사용 레벨:**
- 🟢 **일반**: `pkg/geoapi` - 3줄로 시작
- 🟡 **고급**: `pkg/domain`, `pkg/provider`, `pkg/router` - 커스터마이징
- 🔴 **전문가**: 전체 컴포넌트 조합

## 🔐 보안

- ✅ API 키는 환경변수로 관리 (코드 하드코딩 금지)
- ✅ `.env` 파일은 `.gitignore`에 포함
- ✅ CORS 헤더 설정
- ✅ 좌표 유효성 검증 (Lat: -90~90, Lng: -180~180)
- ✅ Rate Limiting으로 서비스 남용 방지

## 🧪 테스트

```bash
# 테스트 실행
go test ./...

# 커버리지 확인
go test -cover ./...

# 벤치마크
go test -bench=. ./...
```

## 🤝 기여하기

기여를 환영합니다! 다음 단계를 따라주세요:

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

자세한 내용은 [CONTRIBUTING.md](CONTRIBUTING.md)를 참조하세요.

## 📜 라이선스

이 프로젝트는 MIT 라이선스 하에 배포됩니다. 자세한 내용은 [LICENSE](LICENSE) 파일을 참조하세요.

**Third-Party API 이용약관:**
- [Google Maps Platform Terms of Service](https://cloud.google.com/maps-platform/terms)
- [Kakao Local API 이용약관](https://developers.kakao.com/terms)
- [Naver Cloud Platform 이용약관](https://www.ncloud.com/policy/terms/service)
- [vWorld API 공공데이터 이용약관](https://www.vworld.kr)

⚠️ **중요**: Geocoding 결과의 영구 저장(DB 적재)은 API 약관상 제한될 수 있습니다.

## 📚 관련 문서

- [USAGE.md](USAGE.md) - 상세 사용 가이드
- [Technical-Whitepaper.md](docs/Technical-Whitepaper.md) - 기술 백서
- [CONTRIBUTING.md](CONTRIBUTING.md) - 기여 가이드
- [examples/](examples/) - 실제 작동하는 예제

## 🙋 FAQ

**Q: 어떤 API 키가 필요한가요?**
A: 최소 하나 이상의 API 키가 필요합니다. 한국 주소만 처리한다면 Kakao 또는 vWorld만 있어도 됩니다.

**Q: 무료로 사용할 수 있나요?**
A: 네! Kakao (10만건/일), vWorld (4만건/일)를 사용하면 완전 무료입니다. Naver Maps는 새로운 맵스 API로 전환되어 월 600건 무료, 이후 건당 0.1원입니다. ([공지사항](https://www.gov-ncloud.com/v2/support/notice/all/499))

**Q: Gin이 아닌 다른 프레임워크에서도 사용할 수 있나요?**
A: 네! `geoAPI.Geocode()` 함수를 직접 호출하여 어떤 프레임워크에서도 사용 가능합니다.

**Q: Naver Maps API를 사용할 수 있나요?**
A: 네! Naver Maps는 새로운 맵스(Maps) API로 전환되었습니다. 월 600건까지 무료이며, 이후 건당 0.1원의 요금이 부과됩니다. 기존 AI NAVER API는 신규 이용 신청이 차단되었으므로 새로운 맵스 API를 사용해야 합니다. 자세한 내용은 [Naver Cloud Platform 공지사항](https://www.gov-ncloud.com/v2/support/notice/all/499)을 참조하세요.

**Q: 좌표를 주소로 변환(Reverse Geocoding)도 지원하나요?**
A: 현재는 주소→좌표 변환만 지원합니다. Reverse Geocoding은 향후 추가 예정입니다.

## ⭐ Star History

이 프로젝트가 유용하다면 ⭐️를 눌러주세요!

## 📞 연락처

프로젝트 관리자: [@epicsagas](https://github.com/epicsagas)

프로젝트 링크: [https://github.com/epicsagas/korean-geocode](https://github.com/epicsagas/korean-geocode)

---

<div align="center">

**Made with ❤️ in Korea**

</div>

---

<a name="english"></a>

# Korean Geocoder (English)

**Hybrid Geocoding Library Optimized for Korean Addresses** - Integrates Google Maps, Kakao Local, Naver Maps, and vWorld APIs for cost-effective and high-availability geocoding.

## Key Features

- 🇰🇷 **Korean Address Optimized**: Prioritizes free APIs (vWorld → Kakao → Naver → Google)
- 🌏 **Global Address Support**: Google Maps and Naver Maps (English support) for worldwide addresses
- 💰 **Cost Optimization**: Up to 99% cost reduction using free APIs
- 🔄 **Automatic Failover**: Seamless provider switching on failure
- ⚡ **Circuit Breaker**: System stability with fault isolation
- 📊 **Rate Limiting**: Prevent unexpected costs with quota management
- 🎯 **Smart Routing**: Auto-detect Korean/English and select optimal provider
- 🗺️ **WGS84 Unified**: All coordinates normalized to WGS84 (EPSG:4326)
- 🔌 **Gin Integration**: Add to existing Gin projects with just 3 lines

## Installation

```bash
go get github.com/epicsagas/korean-geocode
```

**Requirements**: Go 1.23+

## Quick Start

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/epicsagas/korean-geocode"
)

func main() {
    geoAPI, _ := geoapi.New()
    router := gin.Default()
    geoAPI.RegisterRoutes(router)
    router.Run(":8080")
}
```

For detailed documentation, see [USAGE.md](USAGE.md).

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
