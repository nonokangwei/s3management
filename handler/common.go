package handler

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"

	"github.com/kangwe/s3management/model"
	"github.com/kangwe/s3management/observability"
)

// writeXMLResponse marshals the given value to XML and writes it as the HTTP response.
func writeXMLResponse(ctx context.Context, w http.ResponseWriter, status int, v interface{}) {
	body, err := xml.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal XML response: %v", err)
		model.WriteInternalError(ctx, w, "Failed to generate response")
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	fmt.Fprintf(w, "%s%s", xml.Header, body)
}

// handleGCSError maps GCS SDK errors to S3-compatible error responses.
// Uses Google API error codes for reliable error classification.
func handleGCSError(ctx context.Context, w http.ResponseWriter, err error, bucket, operation string) {
	if err == storage.ErrBucketNotExist {
		log.Printf("Bucket not found: %s", bucket)
		model.WriteNoSuchBucket(ctx, w, bucket)
		return
	}

	if errors.Is(err, context.DeadlineExceeded) {
		log.Printf("GCS request timeout for bucket %s", bucket)
		model.WriteS3Error(ctx, w, "RequestTimeout", "GCS request timed out", http.StatusRequestTimeout, bucket)
		observability.RecordUpstreamError(operation, bucket, "timeout")
		return
	}

	if errors.Is(err, context.Canceled) {
		log.Printf("GCS request canceled for bucket %s", bucket)
		model.WriteS3Error(ctx, w, "RequestCanceled", "Request was canceled", 499, bucket)
		observability.RecordUpstreamError(operation, bucket, "canceled")
		return
	}

	// Use Google API error codes for reliable classification
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		log.Printf("GCS API error for bucket %s: %d %s", bucket, apiErr.Code, apiErr.Message)
		switch apiErr.Code {
		case http.StatusForbidden:
			model.WriteAccessDenied(ctx, w)
		case http.StatusNotFound:
			model.WriteNoSuchBucket(ctx, w, bucket)
		case http.StatusTooManyRequests:
			model.WriteS3Error(ctx, w, "SlowDown", "Rate limit exceeded", http.StatusServiceUnavailable, bucket)
		case http.StatusConflict:
			model.WriteS3Error(ctx, w, "OperationAborted", "A conflicting operation is in progress", http.StatusConflict, bucket)
		case http.StatusPreconditionFailed:
			model.WriteS3Error(ctx, w, "PreconditionFailed", "Precondition failed", http.StatusPreconditionFailed, bucket)
		default:
			model.WriteInternalError(ctx, w, "GCS operation failed")
		}
		observability.RecordUpstreamError(operation, bucket, fmt.Sprintf("%d", apiErr.Code))
		return
	}

	log.Printf("GCS error for bucket %s: %v", bucket, err)
	model.WriteInternalError(ctx, w, "GCS operation failed")
	observability.RecordUpstreamError(operation, bucket, "unknown")
}
