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

// GetTagging handles GET /?tagging requests.
// Retrieves bucket labels from GCS and returns them as S3-compatible tagging XML.
func GetTagging(w http.ResponseWriter, r *http.Request, bucket string, client gcs.BucketOperator) {
	attrs, err := client.GetBucketAttrs(r.Context(), bucket)
	if err != nil {
		handleGCSError(w, err, bucket)
		return
	}

	t := converter.TaggingFromGCS(attrs)
	writeXMLResponse(w, http.StatusOK, t)
}

// PutTagging handles PUT /?tagging requests.
// Parses S3 Tagging XML and sets GCS bucket labels accordingly.
func PutTagging(w http.ResponseWriter, r *http.Request, bucket string, client gcs.BucketOperator) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		model.WriteInternalError(w, "Failed to read request body")
		return
	}

	var t model.Tagging
	if err := xml.Unmarshal(body, &t); err != nil {
		log.Printf("Failed to parse tagging XML: %v", err)
		model.WriteMalformedXML(w)
		return
	}

	update := converter.TaggingToGCS(&t)

	if _, err := client.UpdateBucket(r.Context(), bucket, update); err != nil {
		handleGCSError(w, err, bucket)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// DeleteTagging handles DELETE /?tagging requests.
// Removes all labels from the GCS bucket by fetching current labels and deleting each one.
func DeleteTagging(w http.ResponseWriter, r *http.Request, bucket string, client gcs.BucketOperator) {
	// First fetch current labels to know which keys to delete
	attrs, err := client.GetBucketAttrs(r.Context(), bucket)
	if err != nil {
		handleGCSError(w, err, bucket)
		return
	}

	if len(attrs.Labels) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	update := converter.TaggingDeleteToGCS(attrs.Labels)

	if _, err := client.UpdateBucket(r.Context(), bucket, update); err != nil {
		handleGCSError(w, err, bucket)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
