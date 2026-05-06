package middleware

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type tokenBucket struct {
	tokens float64
	last   time.Time
}

type rateLimiter struct {
	mu sync.Mutex

	rps   float64
	burst float64

	// key: client identifier (typically IP)
	buckets map[string]*tokenBucket
}

func newRateLimiter(rps float64, burst int) *rateLimiter {
	if rps <= 0 {
		rps = 5
	}
	if burst <= 0 {
		burst = 10
	}
	return &rateLimiter{
		rps:     rps,
		burst:   float64(burst),
		buckets: make(map[string]*tokenBucket),
	}
}

func (rl *rateLimiter) allow(key string, now time.Time) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[key]
	if !ok {
		rl.buckets[key] = &tokenBucket{tokens: rl.burst - 1, last: now}
		return true
	}

	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens = minFloat(rl.burst, b.tokens+(elapsed*rl.rps))
		b.last = now
	}

	if b.tokens < 1 {
		return false
	}
	b.tokens -= 1
	return true
}

func (rl *rateLimiter) cleanup(olderThan time.Duration) {
	cutoff := time.Now().Add(-olderThan)
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for k, b := range rl.buckets {
		if b.last.Before(cutoff) {
			delete(rl.buckets, k)
		}
	}
}

func RateLimitMiddleware(next http.Handler) http.Handler {
	rps := envFloat("RATE_LIMIT_RPS", 5)
	burst := envInt("RATE_LIMIT_BURST", 10)
	rl := newRateLimiter(rps, burst)

	// best-effort memory cleanup
	go func() {
		t := time.NewTicker(10 * time.Minute)
		defer t.Stop()
		for range t.C {
			rl.cleanup(30 * time.Minute)
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := clientIP(r)
		if key == "" {
			key = "unknown"
		}
		if !rl.allow(key, time.Now()) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			http.Error(w, `{"error":"rate_limited"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	// If behind a reverse proxy, prefer X-Forwarded-For (first IP).
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}
	if xrip := strings.TrimSpace(r.Header.Get("X-Real-IP")); xrip != "" {
		if net.ParseIP(xrip) != nil {
			return xrip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && net.ParseIP(host) != nil {
		return host
	}
	if net.ParseIP(r.RemoteAddr) != nil {
		return r.RemoteAddr
	}
	return ""
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

func envFloat(key string, def float64) float64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
