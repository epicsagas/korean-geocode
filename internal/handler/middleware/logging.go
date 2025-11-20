package middleware

import (
	"log"
	"net/http"
	"time"
)

// responseWriter는 상태 코드를 캡처하는 래퍼입니다
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += int64(n)
	return n, err
}

// Logging은 HTTP 요청을 로깅하는 미들웨어입니다
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// ResponseWriter 래핑
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// 다음 핸들러 실행
		next.ServeHTTP(rw, r)

		// 로그 출력
		duration := time.Since(start)
		log.Printf(
			"%s %s %d %s %dB",
			r.Method,
			r.RequestURI,
			rw.statusCode,
			duration,
			rw.written,
		)
	})
}
