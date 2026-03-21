package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/kangwe/s3management/observability"
)

// RequestID middleware injects a request ID into the context and response header.
// If the incoming request already provides X-Request-ID, it is propagated.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if existing := r.Header.Get(observability.RequestIDHeader); existing != "" {
			ctx = observability.WithRequestID(ctx, existing)
		} else {
			ctx = observability.WithRequestID(ctx, uuid.NewString())
		}

		r = r.WithContext(ctx)

		requestID := observability.RequestIDFromContext(ctx)
		w.Header().Set(observability.RequestIDHeader, requestID)

		next.ServeHTTP(w, r)
	})
}
