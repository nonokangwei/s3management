package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/kangwe/s3management/gcs"
	"github.com/kangwe/s3management/handler"
	"github.com/kangwe/s3management/observability"
)

// Router dispatches S3 API requests to the appropriate handler based on query parameters.
type Router struct {
	client            gcs.BucketOperator
	gcsRequestTimeout time.Duration // Timeout for individual GCS API calls
}

// NewRouter creates a new Router with the given GCS client and request timeout.
func NewRouter(client gcs.BucketOperator, gcsRequestTimeout time.Duration) *Router {
	return &Router{
		client:            client,
		gcsRequestTimeout: gcsRequestTimeout,
	}
}

// ServeHTTP routes incoming requests based on query parameters (?versioning, ?cors, ?logging, ?tagging).
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bucket := extractBucketName(r)
	if bucket == "" {
		http.Error(w, "Missing bucket name", http.StatusBadRequest)
		return
	}

	r = observability.EnsureRequestContext(r, "", bucket)

	// Apply per-request GCS timeout so a slow backend doesn't block indefinitely
	ctx, cancel := context.WithTimeout(r.Context(), rt.gcsRequestTimeout)
	defer cancel()
	r = r.WithContext(ctx)

	query := r.URL.Query()

	// Dispatch based on query parameter presence.
	// S3 uses bare query params like ?versioning (value is irrelevant).
	switch {
	case query.Has("versioning"):
		rt.handleVersioning(w, r, bucket)
	case query.Has("cors"):
		rt.handleCORS(w, r, bucket)
	case query.Has("logging"):
		rt.handleLogging(w, r, bucket)
	case query.Has("tagging"):
		rt.handleTagging(w, r, bucket)
	default:
		http.Error(w, "Unsupported operation", http.StatusBadRequest)
	}
}

func (rt *Router) handleVersioning(w http.ResponseWriter, r *http.Request, bucket string) {
	switch r.Method {
	case http.MethodGet:
		r = observability.EnsureRequestContext(r, "versioning:get", bucket)
		handler.GetVersioning(w, r, bucket, rt.client)
	case http.MethodPut:
		r = observability.EnsureRequestContext(r, "versioning:put", bucket)
		handler.PutVersioning(w, r, bucket, rt.client)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (rt *Router) handleCORS(w http.ResponseWriter, r *http.Request, bucket string) {
	switch r.Method {
	case http.MethodGet:
		r = observability.EnsureRequestContext(r, "cors:get", bucket)
		handler.GetCORS(w, r, bucket, rt.client)
	case http.MethodPut:
		r = observability.EnsureRequestContext(r, "cors:put", bucket)
		handler.PutCORS(w, r, bucket, rt.client)
	case http.MethodDelete:
		r = observability.EnsureRequestContext(r, "cors:delete", bucket)
		handler.DeleteCORS(w, r, bucket, rt.client)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (rt *Router) handleLogging(w http.ResponseWriter, r *http.Request, bucket string) {
	// S3 logging only supports GET and PUT. To disable logging, PUT with empty BucketLoggingStatus.
	switch r.Method {
	case http.MethodGet:
		r = observability.EnsureRequestContext(r, "logging:get", bucket)
		handler.GetLogging(w, r, bucket, rt.client)
	case http.MethodPut:
		r = observability.EnsureRequestContext(r, "logging:put", bucket)
		handler.PutLogging(w, r, bucket, rt.client)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (rt *Router) handleTagging(w http.ResponseWriter, r *http.Request, bucket string) {
	switch r.Method {
	case http.MethodGet:
		r = observability.EnsureRequestContext(r, "tagging:get", bucket)
		handler.GetTagging(w, r, bucket, rt.client)
	case http.MethodPut:
		r = observability.EnsureRequestContext(r, "tagging:put", bucket)
		handler.PutTagging(w, r, bucket, rt.client)
	case http.MethodDelete:
		r = observability.EnsureRequestContext(r, "tagging:delete", bucket)
		handler.DeleteTagging(w, r, bucket, rt.client)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// extractBucketName extracts the bucket name from the request.
// Supports both virtual-hosted style (mybucket.s3.amazonaws.com)
// and path style (/mybucket) addressing.
func extractBucketName(r *http.Request) string {
	host := r.Host

	// Virtual-hosted style: mybucket.s3.amazonaws.com or mybucket.localhost:8080
	if parts := strings.SplitN(host, ".", 2); len(parts) >= 2 {
		candidate := parts[0]
		if candidate != "s3" && candidate != "localhost" && candidate != "127" {
			return candidate
		}
	}

	// Path style: /mybucket or /mybucket/
	path := strings.TrimPrefix(r.URL.Path, "/")
	if idx := strings.Index(path, "/"); idx > 0 {
		path = path[:idx]
	}

	return path
}
