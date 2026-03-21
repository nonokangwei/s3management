package handler

import (
	"encoding/xml"
	"io"
	"log"
	"net/http"

	"github.com/kangwe/s3management/converter"
	"github.com/kangwe/s3management/gcs"
	"github.com/kangwe/s3management/model"
)

// GetLogging handles GET /?logging requests.
// Retrieves bucket logging configuration from GCS and returns S3-compatible XML.
func GetLogging(w http.ResponseWriter, r *http.Request, bucket string, client gcs.BucketOperator) {
	attrs, err := client.GetBucketAttrs(r.Context(), bucket)
	if err != nil {
		handleGCSError(r.Context(), w, err, bucket, "logging:get")
		return
	}

	ls := converter.LoggingFromGCS(attrs)
	writeXMLResponse(r.Context(), w, http.StatusOK, ls)
}

// PutLogging handles PUT /?logging requests.
// Parses S3 BucketLoggingStatus XML and updates GCS bucket logging.
func PutLogging(w http.ResponseWriter, r *http.Request, bucket string, client gcs.BucketOperator) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		model.WriteInternalError(r.Context(), w, "Failed to read request body")
		return
	}

	var ls model.BucketLoggingStatus
	if err := xml.Unmarshal(body, &ls); err != nil {
		log.Printf("Failed to parse logging XML: %v", err)
		model.WriteMalformedXML(r.Context(), w)
		return
	}

	update, err := converter.LoggingToGCS(&ls)
	if err != nil {
		model.WriteInvalidArgument(r.Context(), w, err.Error())
		return
	}

	if _, err := client.UpdateBucket(r.Context(), bucket, update); err != nil {
		handleGCSError(r.Context(), w, err, bucket, "logging:put")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Note: S3 does not have a DELETE /?logging API.
// To disable logging, clients send PUT /?logging with an empty BucketLoggingStatus
// (no LoggingEnabled element). This is already handled by PutLogging + LoggingToGCS.
