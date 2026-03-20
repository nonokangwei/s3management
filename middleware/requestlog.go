package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/kangwe/s3management/observability"
)

// responseRecorder wraps http.ResponseWriter to capture the status code.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

// RequestLog is middleware that logs each request with method, path, status, and duration.
func RequestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rr := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rr, r)

		duration := time.Since(start).Round(time.Millisecond)
		requestID := observability.RequestIDFromContext(r.Context())
		op := observability.OperationFromContext(r.Context())
		bucket := observability.BucketFromContext(r.Context())

		log.Printf("req_id=%s method=%s path=%s bucket=%s operation=%s status=%d latency=%s", requestID, r.Method, r.URL.String(), bucket, op, rr.statusCode, duration)
		observability.RecordRequest(op, bucket, rr.statusCode, duration)
	})
}
