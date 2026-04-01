package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
)

type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipLimiter struct {
	mu      sync.Mutex
	entries map[string]*entry
	rps     rate.Limit
	burst   int
}

func newIPLimiter(rps rate.Limit, burst int) *ipLimiter {
	return &ipLimiter{
		entries: make(map[string]*entry),
		rps:     rps,
		burst:   burst,
	}
}

func (l *ipLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	e, ok := l.entries[ip]
	if !ok {
		e = &entry{limiter: rate.NewLimiter(l.rps, l.burst)}
		l.entries[ip] = e
	}
	e.lastSeen = time.Now()
	return e.limiter.Allow()
}

func (l *ipLimiter) cleanup(interval, ttl time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		for ip, e := range l.entries {
			if time.Since(e.lastSeen) > ttl {
				delete(l.entries, ip)
			}
		}
		l.mu.Unlock()
	}
}

// RateLimit returns a per-IP token-bucket rate limiting middleware.
// Stale IP entries are pruned every minute after 3 minutes of inactivity.
func RateLimit(rps float64, burst int, logger *zerolog.Logger) func(http.Handler) http.Handler {
	lim := newIPLimiter(rate.Limit(rps), burst)
	go lim.cleanup(time.Minute, 3*time.Minute)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r)
			if !lim.allow(ip) {
				logger.Warn().Str("ip", ip).Str("path", r.URL.Path).Msg("rate limit exceeded")
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"jsonrpc": "2.0",
					"id":      nil,
					"error": map[string]any{
						"code":    -32005,
						"message": "rate limit exceeded",
					},
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// realIP extracts the client IP, respecting X-Real-IP and X-Forwarded-For.
func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.SplitN(fwd, ",", 2)[0])
	}
	addr := r.RemoteAddr
	if i := strings.LastIndex(addr, ":"); i > 0 {
		return addr[:i]
	}
	return addr
}
