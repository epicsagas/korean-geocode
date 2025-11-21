# Contributing to Korean Geocoder

먼저, Korean Geocoder에 기여해주셔서 감사합니다! 🎉

## 🤝 기여 방법

### 1. Fork & Clone

```bash
# Fork this repository
# Then clone your fork
git clone https://github.com/YOUR_USERNAME/korean-geocode.git
cd korean-geocode
```

### 2. 브랜치 생성

```bash
git checkout -b feature/your-feature-name
# 또는
git checkout -b fix/your-bug-fix
```

**브랜치 네이밍 규칙:**
- `feature/` - 새로운 기능 추가
- `fix/` - 버그 수정
- `docs/` - 문서 개선
- `refactor/` - 코드 리팩토링
- `test/` - 테스트 추가/개선

### 3. 개발 환경 설정

```bash
# 의존성 설치
go mod download

# 환경변수 설정
cp .env.example .env
# .env 파일을 편집하여 API 키 입력

# 빌드 테스트
make build

# 테스트 실행
go test ./...
```

### 4. 코드 작성

#### 코드 스타일

- **Go 표준 스타일** 준수: `gofmt`, `goimports` 사용
- **Linting**: `golangci-lint` 통과 필수
- **주석**: 공개 함수/타입에는 godoc 주석 작성

```go
// GoodExample은 올바른 주석 예시입니다.
// 함수의 목적과 동작을 명확히 설명합니다.
func GoodExample(param string) error {
    // 구현...
}
```

#### 테스트 작성

모든 새로운 기능과 버그 수정에는 테스트가 필요합니다:

```go
func TestYourFeature(t *testing.T) {
    // Arrange
    expected := "expected result"

    // Act
    result := YourFunction()

    // Assert
    if result != expected {
        t.Errorf("got %v, want %v", result, expected)
    }
}
```

**테스트 실행:**

```bash
# 모든 테스트
go test ./...

# 특정 패키지
go test ./internal/providers

# 커버리지
go test -cover ./...

# 자세한 출력
go test -v ./...
```

### 5. 커밋

**커밋 메시지 규칙:**

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Type:**
- `feat`: 새로운 기능
- `fix`: 버그 수정
- `docs`: 문서 변경
- `style`: 코드 포맷팅 (기능 변경 없음)
- `refactor`: 리팩토링
- `test`: 테스트 추가/수정
- `chore`: 빌드 프로세스, 도구 설정 등

**예시:**

```bash
git commit -m "feat(providers): add Naver Maps provider support

- Add NaverProvider implementation
- Update Smart Router to include Naver
- Add tests for Naver provider

Closes #42"
```

### 6. Push & Pull Request

```bash
# Push to your fork
git push origin feature/your-feature-name
```

GitHub에서 Pull Request를 생성하세요.

**PR 제목 예시:**
- `[Feature] Add Naver Maps provider support`
- `[Fix] Resolve timeout issue in Circuit Breaker`
- `[Docs] Update installation guide`

**PR 설명 템플릿:**

```markdown
## 변경 사항
- 변경 내용을 간단히 설명

## 동기
- 왜 이 변경이 필요한지

## 테스트 방법
- 어떻게 테스트했는지

## 체크리스트
- [ ] 테스트 작성 및 통과
- [ ] 문서 업데이트
- [ ] Linter 통과
- [ ] Breaking change 여부 명시
```

## 📝 코드 리뷰 프로세스

1. **자동 검사**: CI/CD가 자동으로 테스트 및 린팅 실행
2. **코드 리뷰**: 유지보수자가 코드를 검토
3. **피드백 반영**: 필요시 수정
4. **승인 & 머지**: 리뷰 통과 후 main 브랜치에 병합

## 🐛 버그 리포트

버그를 발견하셨나요? [Issue](https://github.com/epicsagas/korean-geocode/issues)를 열어주세요!

**버그 리포트에 포함할 내용:**
- 명확한 제목
- 재현 단계
- 예상 동작 vs 실제 동작
- 환경 정보 (Go 버전, OS 등)
- 에러 메시지 / 로그

**템플릿:**

```markdown
## 버그 설명
간단한 버그 설명

## 재현 방법
1. 이렇게 하고
2. 저렇게 하면
3. 버그 발생

## 예상 동작
이렇게 되어야 함

## 실제 동작
이렇게 됨

## 환경
- Go 버전: 1.23.0
- OS: macOS 14.0
- Korean Geocoder 버전: v0.1.0

## 추가 정보
스크린샷, 로그 등
```

## 💡 기능 제안

새로운 기능을 제안하고 싶으신가요?

1. 먼저 [Issue](https://github.com/epicsagas/korean-geocode/issues)에서 중복 여부 확인
2. 새 Issue 생성
3. 기능의 목적과 사용 사례 설명
4. 가능하면 구현 방안 제시

## 🧪 테스트 가이드

### Unit Tests

```bash
# 전체 테스트
go test ./...

# 특정 패키지
go test ./internal/providers

# 커버리지 리포트
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Tests

```bash
# 실제 API 키가 필요한 통합 테스트
TEST_INTEGRATION=true go test ./...
```

### Benchmark Tests

```bash
go test -bench=. ./internal/router
```

## 📚 문서 기여

문서 개선도 큰 기여입니다!

- **README.md**: 프로젝트 개요, 빠른 시작
- **USAGE.md**: 상세 사용 가이드
- **godoc**: 코드 주석 개선
- **예제**: `examples/` 디렉토리에 새로운 예제 추가

## 🔒 보안 취약점 보고

보안 취약점을 발견하셨다면, 공개 Issue를 열지 마시고 다음으로 연락주세요:

**이메일**: [보안 관련 이메일 주소]

## ⚖️ 라이선스

기여한 코드는 프로젝트의 [Apache License 2.0](LICENSE)에 따라 배포됩니다.

## 🙏 감사의 말

모든 기여자분들께 감사드립니다!

<!-- ALL-CONTRIBUTORS-LIST:START -->
<!-- 기여자 목록이 여기에 자동으로 추가됩니다 -->
<!-- ALL-CONTRIBUTORS-LIST:END -->

---

**질문이 있으신가요?**

- 💬 [Discussions](https://github.com/epicsagas/korean-geocode/discussions)에서 질문하세요
- 📧 또는 Issue를 열어주세요

다시 한번 기여해주셔서 감사합니다! 🚀
