package observability

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ctxKey string

const (
	requestIDKey ctxKey = "request_id"
	operationKey ctxKey = "operation"
	bucketKey    ctxKey = "bucket"

	RequestIDHeader = "X-Request-ID"
)

var (
	requestLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "s3_proxy_request_latency_seconds",
		Help:    "Latency of S3 management proxy requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation", "bucket", "status"})

	upstreamErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "s3_proxy_upstream_errors_total",
		Help: "Count of upstream GCS errors by operation and bucket",
	}, []string{"operation", "bucket", "reason"})
)

// EnsureRequestContext injects request ID, operation, and bucket onto the context.
func EnsureRequestContext(r *http.Request, operation, bucket string) *http.Request {
	ctx := r.Context()
	if rid := RequestIDFromContext(ctx); rid == "" {
		ctx = context.WithValue(ctx, requestIDKey, uuid.NewString())
	}
	if operation != "" {
		ctx = context.WithValue(ctx, operationKey, operation)
	}
	if bucket != "" {
		ctx = context.WithValue(ctx, bucketKey, bucket)
	}
	return r.WithContext(ctx)
}

// WithRequestID explicitly sets the request ID on the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

func OperationFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(operationKey).(string); ok {
		return v
	}
	return ""
}

func BucketFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(bucketKey).(string); ok {
		return v
	}
	return ""
}

// RecordRequest captures latency with operation/bucket/status labels.
func RecordRequest(operation, bucket string, status int, duration time.Duration) {
	requestLatency.WithLabelValues(operation, bucket, strconv.Itoa(status)).Observe(duration.Seconds())
}

// RecordUpstreamError increments the upstream error counter.
func RecordUpstreamError(operation, bucket, reason string) {
	upstreamErrors.WithLabelValues(operation, bucket, reason).Inc()
}

// MetricsHandler exposes Prometheus metrics.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
