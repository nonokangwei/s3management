package handler

import (
	"encoding/xml"
	"io"
	"log"
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/kangwe/s3management/converter"
	"github.com/kangwe/s3management/gcs"
	"github.com/kangwe/s3management/model"
)

// GetCORS handles GET /?cors requests.
// Retrieves bucket CORS configuration from GCS and returns S3-compatible XML.
func GetCORS(w http.ResponseWriter, r *http.Request, bucket string, client gcs.BucketOperator) {
	attrs, err := client.GetBucketAttrs(r.Context(), bucket)
	if err != nil {
		handleGCSError(r.Context(), w, err, bucket, "cors:get")
		return
	}

	cc := converter.CORSFromGCS(attrs.CORS)
	writeXMLResponse(r.Context(), w, http.StatusOK, cc)
}

// PutCORS handles PUT /?cors requests.
// Parses S3 CORSConfiguration XML and updates GCS bucket CORS rules.
func PutCORS(w http.ResponseWriter, r *http.Request, bucket string, client gcs.BucketOperator) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		model.WriteInternalError(r.Context(), w, "Failed to read request body")
		return
	}

	var cc model.CORSConfiguration
	if err := xml.Unmarshal(body, &cc); err != nil {
		log.Printf("Failed to parse CORS XML: %v", err)
		model.WriteMalformedXML(r.Context(), w)
		return
	}

	gcsCORS := converter.CORSToGCS(&cc)
	update := storage.BucketAttrsToUpdate{
		CORS: gcsCORS,
	}

	if _, err := client.UpdateBucket(r.Context(), bucket, update); err != nil {
		handleGCSError(r.Context(), w, err, bucket, "cors:put")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// DeleteCORS handles DELETE /?cors requests.
// Removes all CORS rules from the GCS bucket by setting an empty CORS slice.
func DeleteCORS(w http.ResponseWriter, r *http.Request, bucket string, client gcs.BucketOperator) {
	update := storage.BucketAttrsToUpdate{
		CORS: []storage.CORS{},
	}

	if _, err := client.UpdateBucket(r.Context(), bucket, update); err != nil {
		handleGCSError(r.Context(), w, err, bucket, "cors:delete")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
