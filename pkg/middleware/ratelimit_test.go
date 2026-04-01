package middleware_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/sudores/twt-test-task/pkg/middleware"
)

var nopLogger = zerolog.Nop()

// okHandler is a simple upstream that always returns 200.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":"ok"}`)
})

func TestRateLimit_AllowsUnderLimit(t *testing.T) {
	// burst=10 means first 10 requests go through immediately.
	rl := middleware.RateLimit(100, 10, nopLogger)
	h := rl(okHandler)

	for i := range 5 {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: want 200, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimit_Blocks_WhenBurstExceeded(t *testing.T) {
	// rps=1, burst=2: first 2 requests pass, 3rd is rejected.
	rl := middleware.RateLimit(1, 2, nopLogger)
	h := rl(okHandler)

	pass := 0
	blocked := 0
	for range 5 {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = "10.0.0.2:5678"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		switch w.Code {
		case http.StatusOK:
			pass++
		case http.StatusTooManyRequests:
			blocked++
		default:
			t.Fatalf("unexpected status %d", w.Code)
		}
	}

	if pass != 2 {
		t.Errorf("want 2 passing requests, got %d", pass)
	}
	if blocked != 3 {
		t.Errorf("want 3 blocked requests, got %d", blocked)
	}
}

func TestRateLimit_BlockedResponse_IsValidJSON(t *testing.T) {
	rl := middleware.RateLimit(0, 0, nopLogger) // burst=0 blocks everything
	h := rl(okHandler)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "10.0.0.3:9999"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("want 429, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: want application/json, got %q", ct)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatal("missing error object in response")
	}
	if errObj["code"] != float64(-32005) {
		t.Errorf("error.code: want -32005, got %v", errObj["code"])
	}
}

func TestRateLimit_SeparateLimitsPerIP(t *testing.T) {
	// burst=1: each IP gets exactly 1 request before being throttled.
	rl := middleware.RateLimit(0, 1, nopLogger)
	h := rl(okHandler)

	for _, ip := range []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"} {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = ip + ":1234"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("first request from %s: want 200, got %d", ip, w.Code)
		}
	}
}

func TestRateLimit_ReadsXRealIP(t *testing.T) {
	rl := middleware.RateLimit(0, 1, nopLogger)
	h := rl(okHandler)

	// First request from "real" IP via header — should pass (burst=1).
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "127.0.0.1:0"
	req.Header.Set("X-Real-IP", "5.5.5.5")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}

	// Second request from same real IP — burst exhausted.
	req2 := httptest.NewRequest(http.MethodPost, "/", nil)
	req2.RemoteAddr = "127.0.0.1:0"
	req2.Header.Set("X-Real-IP", "5.5.5.5")
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("want 429, got %d", w2.Code)
	}
}
