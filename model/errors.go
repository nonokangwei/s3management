package model

import (
	"encoding/xml"
	"fmt"
	"net/http"
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
func WriteS3Error(w http.ResponseWriter, code string, message string, httpStatus int, resource string) {
	s3Err := S3Error{
		Code:      code,
		Message:   message,
		Resource:  resource,
		RequestId: "00000000-0000-0000-0000-000000000000", // Placeholder request ID
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

func WriteNoSuchBucket(w http.ResponseWriter, bucket string) {
	WriteS3Error(w, "NoSuchBucket", "The specified bucket does not exist", http.StatusNotFound, bucket)
}

func WriteInvalidArgument(w http.ResponseWriter, message string) {
	WriteS3Error(w, "InvalidArgument", message, http.StatusBadRequest, "")
}

func WriteMalformedXML(w http.ResponseWriter) {
	WriteS3Error(w, "MalformedXML", "The XML you provided was not well-formed or did not validate against our published schema", http.StatusBadRequest, "")
}

func WriteAccessDenied(w http.ResponseWriter) {
	WriteS3Error(w, "AccessDenied", "Access Denied", http.StatusForbidden, "")
}

func WriteInternalError(w http.ResponseWriter, message string) {
	WriteS3Error(w, "InternalError", message, http.StatusInternalServerError, "")
}
