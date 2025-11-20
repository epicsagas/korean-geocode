package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecovery_NoPanic(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	middleware := Recovery(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "success", rec.Body.String())
}

func TestRecovery_StringPanic(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	})

	middleware := Recovery(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	// Verify 500 error response
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Internal Server Error")
	assert.Contains(t, rec.Body.String(), "something went wrong")

	// Verify panic was logged
	logOutput := buf.String()
	assert.Contains(t, logOutput, "Panic recovered")
	assert.Contains(t, logOutput, "something went wrong")
}

func TestRecovery_ErrorPanic(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	})

	middleware := Recovery(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Internal Server Error")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "Panic recovered")
}

func TestRecovery_StructPanic(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	type customError struct {
		Message string
		Code    int
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(customError{Message: "custom error", Code: 42})
	})

	middleware := Recovery(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Internal Server Error")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "Panic recovered")
	assert.Contains(t, logOutput, "custom error")
}

func TestRecovery_NilPanic(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ptr *int
		_ = *ptr // Nil pointer dereference
	})

	middleware := Recovery(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Internal Server Error")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "Panic recovered")
}

func TestRecovery_StackTrace(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic for stack trace")
	})

	middleware := Recovery(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	logOutput := buf.String()

	// Verify stack trace is included
	assert.Contains(t, logOutput, "Panic recovered")
	assert.Contains(t, logOutput, "test panic for stack trace")

	// Stack trace should contain goroutine info and file paths
	hasGoroutine := strings.Contains(logOutput, "goroutine")
	hasFilePath := strings.Contains(logOutput, ".go:")
	assert.True(t, hasGoroutine || hasFilePath, "Log should contain stack trace information")
}

func TestRecovery_MultiplePanics(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("first panic")
	})

	middleware := Recovery(handler)

	// First panic
	req1 := httptest.NewRequest(http.MethodGet, "/test1", nil)
	rec1 := httptest.NewRecorder()
	middleware.ServeHTTP(rec1, req1)

	assert.Equal(t, http.StatusInternalServerError, rec1.Code)

	// Clear buffer
	buf.Reset()

	// Second panic
	req2 := httptest.NewRequest(http.MethodGet, "/test2", nil)
	rec2 := httptest.NewRecorder()
	middleware.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusInternalServerError, rec2.Code)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "Panic recovered")
}

func TestRecovery_PanicAfterWrite(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("partial response"))
		panic("panic after write")
	})

	middleware := Recovery(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	// Note: Status code might still be 200 if header was already written
	// But error message should be appended to response
	responseBody := rec.Body.String()
	assert.Contains(t, responseBody, "partial response")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "Panic recovered")
	assert.Contains(t, logOutput, "panic after write")
}

func TestRecovery_DifferentHTTPMethods(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			buf.Reset()

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("test panic")
			})

			middleware := Recovery(handler)
			req := httptest.NewRequest(method, "/test", nil)
			rec := httptest.NewRecorder()

			middleware.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusInternalServerError, rec.Code)

			logOutput := buf.String()
			assert.Contains(t, logOutput, "Panic recovered")
		})
	}
}

func TestRecovery_ChainedMiddleware(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	// Handler that panics
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("chained panic")
	})

	// Chain recovery with logging
	middleware := Recovery(Logging(handler))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	logOutput := buf.String()
	// Should have both recovery log and request log
	assert.Contains(t, logOutput, "Panic recovered")
	assert.Contains(t, logOutput, "chained panic")
}
