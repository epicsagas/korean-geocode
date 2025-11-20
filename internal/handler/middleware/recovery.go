package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
)

// Recovery는 패닉을 복구하고 500 에러를 반환하는 미들웨어입니다
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// 스택 트레이스 로깅
				log.Printf("Panic recovered: %v\n%s", err, debug.Stack())

				// 500 Internal Server Error 응답
				http.Error(w, fmt.Sprintf("Internal Server Error: %v", err), http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
