.PHONY: build run test clean help

# 기본 타겟
all: build

# 빌드
build:
	@echo "🔨 Building korean-geocode api..."
	@go build -o bin/korean-geocode-api cmd/api/main.go
	@echo "✅ Build complete: ./bin/korean-geocode-api"

# 실행
run:
	@echo "🚀 Starting korean-geocode api server..."
	@go run cmd/api/main.go

# Hot reload 개발 모드 (air 필요)
dev:
	@echo "🔥 Starting development server with hot reload..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "⚠️  air not found. Install it with: go install github.com/air-verse/air@latest"; \
		echo "📖 Or use: make run"; \
	fi

# 테스트
test:
	@echo "🧪 Running tests..."
	@go test -v ./...

# 커버리지 테스트
coverage:
	@echo "📊 Generating coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out
	@echo ""
	@echo "💡 To view HTML coverage report, run:"
	@echo "   go tool cover -html=coverage.out"

# 커버리지 HTML 보기
coverage-html:
	@echo "🌐 Opening HTML coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out

# Swagger 문서 생성
swagger:
	@echo "📚 Generating Swagger documentation..."
	@swag init -g cmd/api/main.go -o docs/swagger
	@echo "✅ Swagger docs generated at docs/"
	@echo "💡 Access Swagger UI at http://localhost:8080/swagger/"

# 의존성 정리
tidy:
	@echo "📦 Tidying dependencies..."
	@go mod tidy

# 정리
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf bin/korean-geocode-api tmp/ coverage.out build-errors.log
	@echo "✅ Clean complete"

# 도움말
help:
	@echo "GeoCoding Hybrid API Server - Makefile Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build         - Build the server binary"
	@echo "  run           - Run the server directly"
	@echo "  dev           - Run with hot reload (requires air)"
	@echo "  test          - Run all tests"
	@echo "  coverage      - Generate test coverage report"
	@echo "  coverage-html - Open HTML coverage report in browser"
	@echo "  swagger       - Generate Swagger/OpenAPI documentation"
	@echo "  tidy          - Tidy Go module dependencies"
	@echo "  clean         - Remove build artifacts"
	@echo "  help          - Show this help message"
