package handler

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/kangwe/s3management/converter"
	"github.com/kangwe/s3management/gcs"
	"github.com/kangwe/s3management/model"
)

// GetVersioning handles GET /?versioning requests.
// Retrieves bucket versioning status from GCS and returns S3-compatible XML.
func GetVersioning(w http.ResponseWriter, r *http.Request, bucket string, client gcs.BucketOperator) {
	attrs, err := client.GetBucketAttrs(r.Context(), bucket)
	if err != nil {
		handleGCSError(w, err, bucket)
		return
	}

	vc := converter.VersioningFromGCS(attrs)
	writeXMLResponse(w, http.StatusOK, vc)
}

// PutVersioning handles PUT /?versioning requests.
// Parses S3 VersioningConfiguration XML and updates GCS bucket versioning.
func PutVersioning(w http.ResponseWriter, r *http.Request, bucket string, client gcs.BucketOperator) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		model.WriteInternalError(w, "Failed to read request body")
		return
	}

	var vc model.VersioningConfiguration
	if err := xml.Unmarshal(body, &vc); err != nil {
		log.Printf("Failed to parse versioning XML: %v", err)
		model.WriteMalformedXML(w)
		return
	}

	update, err := converter.VersioningToGCS(&vc)
	if err != nil {
		model.WriteInvalidArgument(w, err.Error())
		return
	}

	if _, err := client.UpdateBucket(r.Context(), bucket, update); err != nil {
		handleGCSError(w, err, bucket)
		return
	}

	// S3 returns 200 with empty body for PUT versioning
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "")
}
