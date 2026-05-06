package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type ctxKey string

const (
	ctxKeyRequestID ctxKey = "request_id"
	ctxKeyUserID    ctxKey = "user_id"
)

func WithRequestID(next http.Handler) http.Handler {
	logger := auditLogger()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-Id")
		if rid == "" {
			rid = newRequestID()
		}
		w.Header().Set("X-Request-Id", rid)
		ctx := context.WithValue(r.Context(), ctxKeyRequestID, rid)
		next.ServeHTTP(w, r.WithContext(ctx))

		// best-effort: ensure logger is referenced so it isn't optimized away in tiny programs
		_ = logger
	})
}

func AuditMiddleware(next http.Handler) http.Handler {
	log := auditLogger()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rr := &responseRecorder{ResponseWriter: w, status: 200}

		next.ServeHTTP(rr, r)

		dur := time.Since(start)
		rid, _ := r.Context().Value(ctxKeyRequestID).(string)
		uid := r.Context().Value(ctxKeyUserID)

		attrs := []slog.Attr{
			slog.String("event", auditEventName(r)),
			slog.String("request_id", rid),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rr.status),
			slog.Int("bytes", rr.bytes),
			slog.String("ip", clientIP(r)),
			slog.String("user_agent", r.UserAgent()),
			slog.Duration("duration_ms", dur.Round(time.Millisecond)),
		}
		if uid != nil {
			switch v := uid.(type) {
			case int:
				attrs = append(attrs, slog.Int("user_id", v))
			case int64:
				attrs = append(attrs, slog.Int64("user_id", v))
			case string:
				attrs = append(attrs, slog.String("user_id", v))
			}
		}

		// level based on outcome
		if rr.status >= 500 {
			log.Error("audit", slog.Attr{Key: "audit", Value: slog.GroupValue(attrs...)})
			return
		}
		log.Info("audit", slog.Attr{Key: "audit", Value: slog.GroupValue(attrs...)})
	})
}

func auditLogger() *slog.Logger {
	level := new(slog.LevelVar)
	level.Set(slog.LevelInfo)
	if os.Getenv("AUDIT_LOG_LEVEL") == "debug" {
		level.Set(slog.LevelDebug)
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

func auditEventName(r *http.Request) string {
	switch r.URL.Path {
	case "/login":
		return "auth.login"
	case "/register":
		return "auth.register"
	case "/profile":
		return "auth.profile"
	default:
		return "http.request"
	}
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(b[:])
}
