package model

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/kangwe/s3management/observability"
)

// S3Error represents an S3-compatible XML error response.
type S3Error struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	Resource  string   `xml:"Resource,omitempty"`
	RequestId string   `xml:"RequestId"`
}

// WriteS3Error writes an S3-compatible XML error response to the HTTP response writer.
func WriteS3Error(ctx context.Context, w http.ResponseWriter, code string, message string, httpStatus int, resource string) {
	requestID := observability.RequestIDFromContext(ctx)
	if requestID == "" {
		requestID = uuid.NewString()
	}
	s3Err := S3Error{
		Code:      code,
		Message:   message,
		Resource:  resource,
		RequestId: requestID,
	}
	body, err := xml.MarshalIndent(s3Err, "", "  ")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(httpStatus)
	fmt.Fprintf(w, "%s%s", xml.Header, body)
}

// Common S3 error constructors

func WriteNoSuchBucket(ctx context.Context, w http.ResponseWriter, bucket string) {
	WriteS3Error(ctx, w, "NoSuchBucket", "The specified bucket does not exist", http.StatusNotFound, bucket)
}

func WriteInvalidArgument(ctx context.Context, w http.ResponseWriter, message string) {
	WriteS3Error(ctx, w, "InvalidArgument", message, http.StatusBadRequest, "")
}

func WriteMalformedXML(ctx context.Context, w http.ResponseWriter) {
	WriteS3Error(ctx, w, "MalformedXML", "The XML you provided was not well-formed or did not validate against our published schema", http.StatusBadRequest, "")
}

func WriteAccessDenied(ctx context.Context, w http.ResponseWriter) {
	WriteS3Error(ctx, w, "AccessDenied", "Access Denied", http.StatusForbidden, "")
}

func WriteInternalError(ctx context.Context, w http.ResponseWriter, message string) {
	WriteS3Error(ctx, w, "InternalError", message, http.StatusInternalServerError, "")
}
