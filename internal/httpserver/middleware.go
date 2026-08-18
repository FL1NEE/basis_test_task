package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/FL1NEE/basis_test_task/internal/auth"
	"github.com/FL1NEE/basis_test_task/internal/domain"
	"github.com/google/uuid"
)

type contextKey string

const (
	userIDContextKey    contextKey = "userID"
	requestIDContextKey contextKey = "requestID"
)

func userIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDContextKey).(int64)
	return id, ok
}

func requestIDFromContext(ctx context.Context) string {
	id, ok := ctx.Value(requestIDContextKey).(string)
	if !ok {
		return ""
	}
	return id
}

func requireAuth(issuer *auth.TokenIssuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || token == "" {
				writeError(w, domain.ErrUnauthorized)
				return
			}

			claims, err := issuer.Parse(token)
			if err != nil {
				writeError(w, domain.ErrUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		ctx := context.WithValue(r.Context(), requestIDContextKey, reqID)
		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		reqID := requestIDFromContext(r.Context())
		userID, hasUser := userIDFromContext(r.Context())

		attrs := []slog.Attr{
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("request_id", reqID),
		}
		if hasUser {
			attrs = append(attrs, slog.Int64("user_id", userID))
		}

		next.ServeHTTP(rec, r)

		attrs = append(attrs,
			slog.Int("status", rec.status),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
		)

		slog.LogAttrs(r.Context(), slog.LevelInfo, "http request", attrs...)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
