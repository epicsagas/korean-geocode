package middleware

import (
	"net/http"
)

// CORSConfig는 CORS 설정입니다
type CORSConfig struct {
	AllowOrigins []string
	AllowMethods []string
	AllowHeaders []string
}

// DefaultCORSConfig는 기본 CORS 설정을 반환합니다
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Content-Type", "Authorization"},
	}
}

// CORS는 CORS 헤더를 추가하는 미들웨어입니다
func CORS(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// CORS 헤더 설정
			if len(config.AllowOrigins) > 0 {
				w.Header().Set("Access-Control-Allow-Origin", config.AllowOrigins[0])
			}

			if len(config.AllowMethods) > 0 {
				methods := ""
				for i, method := range config.AllowMethods {
					if i > 0 {
						methods += ", "
					}
					methods += method
				}
				w.Header().Set("Access-Control-Allow-Methods", methods)
			}

			if len(config.AllowHeaders) > 0 {
				headers := ""
				for i, header := range config.AllowHeaders {
					if i > 0 {
						headers += ", "
					}
					headers += header
				}
				w.Header().Set("Access-Control-Allow-Headers", headers)
			}

			// Preflight 요청 처리
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
