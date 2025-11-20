# Security Policy

## Supported Versions

다음 버전들에 대해 보안 업데이트를 제공합니다:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

**보안 취약점을 발견하셨나요?** 책임 있는 공개를 위해 다음 절차를 따라주세요.

### 보고 방법

보안 취약점은 **공개 이슈로 보고하지 마세요**. 대신 다음 방법을 사용해주세요:

**GitHub Security Advisory** (권장)
   - [Security Advisories](https://github.com/epicsagas/korean-geocode/security/advisories) 페이지에서 "Report a vulnerability" 클릭
   - 비공개로 취약점 세부 정보 제출


### 보고 시 포함할 정보

다음 정보를 최대한 상세히 제공해주세요:

- **취약점 유형**: 예) API 키 노출, SQL Injection, XSS 등
- **영향 범위**: 어떤 컴포넌트/기능이 영향을 받는지
- **재현 방법**: 단계별 재현 절차
- **영향도**: 잠재적 피해 범위와 심각도
- **제안 해결책**: 가능한 경우 패치 제안
- **환경 정보**: Go 버전, OS, 관련 설정

### 예시 보고서

```
## 취약점 요약
API 키가 로그에 평문으로 기록될 수 있습니다.

## 영향 범위
- 영향받는 버전: v1.0.0 - v1.2.0
- 영향받는 컴포넌트: pkg/provider/google.go

## 재현 방법
1. DEBUG 로그 레벨 활성화
2. Google Provider로 지오코딩 요청
3. 로그 파일 확인 시 API 키 노출 확인

## 영향도
- 심각도: High
- 로그 파일 접근 권한이 있는 공격자가 API 키 탈취 가능

## 제안 해결책
API 키를 `***` 로 마스킹하여 로깅
```

## 응답 시간

- **초기 응답**: 보고 후 48시간 이내
- **취약점 확인**: 5 영업일 이내
- **패치 제공**: 심각도에 따라 다름
  - Critical: 7일 이내
  - High: 30일 이내
  - Medium: 60일 이내
  - Low: 90일 이내

## 보안 업데이트 프로세스

1. **접수**: 보안 취약점 보고 접수
2. **확인**: 취약점 재현 및 영향 범위 분석
3. **패치 개발**: 수정 코드 작성 및 내부 테스트
4. **조율**: 보고자와 공개 일정 조율
5. **릴리스**: 보안 패치 버전 릴리스
6. **공개**: GitHub Security Advisory 및 릴리스 노트 공개
7. **크레딧**: 보고자에게 크레딧 제공 (원하는 경우)

## 보안 모범 사례

이 라이브러리를 사용할 때 다음 보안 모범 사례를 따라주세요:

### 1. API 키 관리

**❌ 나쁜 예**:
```go
// 코드에 하드코딩 - 절대 하지 마세요!
provider := provider.NewGoogleProvider("AIzaSyABC123...")
```

**✅ 좋은 예**:
```go
// 환경 변수 사용
apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
provider := provider.NewGoogleProvider(apiKey)
```

### 2. 환경 변수 보호

```bash
# .env 파일을 반드시 .gitignore에 추가
echo ".env" >> .gitignore

# .env.example 파일로 템플릿 제공
cp .env .env.example
# .env.example에서 실제 값 제거
```

### 3. API 키 제한 설정

각 Provider의 콘솔에서 다음 제한을 설정하세요:

- **IP 제한**: 서버 IP만 허용
- **Referer 제한**: 허용된 도메인만
- **일일 쿼터**: 예상 사용량의 110% 정도로 제한
- **API 범위**: 필요한 API만 활성화

### 4. 로깅 보안

```go
// API 키를 로그에 남기지 마세요
log.Printf("Request: %s", url) // ❌ URL에 API 키 포함 가능

// 민감한 정보는 마스킹하세요
log.Printf("Provider: %s, Query: %s", provider.Name(), query) // ✅
```

### 5. Rate Limiting

```go
// Rate limiter를 사용하여 비용 폭탄 방지
limiter := ratelimit.NewSQLiteRateLimiter("./data")
limiter.SetQuota("google", 10000) // 일일 10,000건으로 제한
```

## 알려진 보안 고려사항

### 1. 좌표계 변환
- 모든 좌표는 WGS84 (EPSG:4326)로 정규화됨
- 좌표 유효성 검증 자동 수행 (`ValidateCoordinates()`)

### 2. 외부 API 타임아웃
- 기본 타임아웃: 2초 (Provider별)
- Context를 통한 타임아웃 제어 가능

### 3. 의존성 보안
- 정기적으로 `go mod tidy` 실행
- Dependabot 활성화 권장
- 알려진 취약점이 있는 패키지 업데이트

```bash
# 의존성 취약점 검사
go list -json -m all | nancy sleuth
```

## 보안 연락처

보안 취약점은 **GitHub Security Advisory**를 통해 보고해주세요:

- [Security Advisories 페이지](https://github.com/epicsagas/korean-geocode/security/advisories)에서 "Report a vulnerability" 클릭
- 비공개로 취약점을 안전하게 논의하고 해결할 수 있습니다

## Hall of Fame

보안 취약점을 책임감 있게 보고해주신 분들:

<!-- 보고자 이름과 날짜가 여기에 추가됩니다 -->

- 아직 보고자가 없습니다. 첫 번째 기여자가 되어주세요!

---

**참고**: 이 문서는 [GitHub의 보안 정책 가이드](https://docs.github.com/en/code-security/getting-started/adding-a-security-policy-to-your-repository)를 따릅니다.
