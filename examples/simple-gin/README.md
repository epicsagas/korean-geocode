# Simple Gin Integration Example

가장 간단한 Gin 프로젝트 통합 예제입니다.

## 사용 방법

### 1. 환경변수 설정

```bash
cp .env.example .env
# .env 파일을 편집하여 API 키 입력
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
# 국내 주소
curl "http://localhost:8080/v1/geocode?q=서울특별시 강남구 테헤란로 152"

# 해외 주소
curl "http://localhost:8080/v1/geocode?q=1600 Amphitheatre Parkway, Mountain View, CA"

# 헬스체크
curl "http://localhost:8080/health"
```

## 핵심 코드

```go
// 1. GeoAPI 초기화
geoAPI, err := geoapi.New()

// 2. Gin 라우터 생성
router := gin.Default()

// 3. 라우트 등록
geoAPI.RegisterRoutes(router)

// 4. 서버 실행
router.Run(":8080")
```

단 4줄로 완성!
