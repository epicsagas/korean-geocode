# Custom Configuration Example

환경변수 대신 코드에서 직접 설정을 구성하는 예제입니다.

## 특징

- .env 파일 불필요
- 코드에서 직접 API 키 설정
- 타임아웃, Circuit Breaker 등 세밀한 조정 가능
- 커스텀 URL prefix 사용 예제 (`/api` prefix)

## 사용 방법

### 1. API 키 설정

`main.go` 파일을 열어 API 키를 직접 수정:

```go
cfg.Google.APIKey = "your_google_api_key"
cfg.Kakao.APIKey = "your_kakao_api_key"
cfg.Naver.ClientID = "your_naver_client_id"
cfg.Naver.ClientSecret = "your_naver_client_secret"
cfg.VWorld.APIKey = "your_vworld_api_key"
```

### 2. 의존성 설치

```bash
go mod tidy
```

### 3. 실행

```bash
go run main.go
```

### 4. 테스트

```bash
# 국내 주소 (prefix: /api)
curl "http://localhost:8080/api/v1/geocode?q=서울특별시 강남구 테헤란로 152"

# 헬스체크
curl "http://localhost:8080/api/health"

# 설정 확인
curl "http://localhost:8080/"
```

## 커스터마이징 가능한 설정

```go
cfg := geoapi.DefaultConfig()

// 타임아웃
cfg.Server.Timeout = 10 * time.Second
cfg.Google.Timeout = 3 * time.Second

// Circuit Breaker
cfg.CircuitBreaker.FailureThreshold = 3
cfg.CircuitBreaker.Timeout = 30 * time.Second

// Rate Limiting
cfg.Kakao.DailyQuota = 300000
cfg.Naver.DailyQuota = 100000
```
