package converter

import (
	"fmt"

	"cloud.google.com/go/storage"
	"github.com/kangwe/s3management/model"
)

// VersioningToGCS converts an S3 VersioningConfiguration to a GCS BucketAttrsToUpdate.
// Returns an error if MfaDelete is set, as GCS does not support it.
func VersioningToGCS(vc *model.VersioningConfiguration) (storage.BucketAttrsToUpdate, error) {
	update := storage.BucketAttrsToUpdate{}

	// GCS does not support MFA Delete
	if vc.MfaDelete == "Enabled" {
		return update, fmt.Errorf("MfaDelete is not supported by GCS")
	}

	switch vc.Status {
	case "Enabled":
		update.VersioningEnabled = true
	case "Suspended":
		// GCS uses a boolean; Suspended maps to false
		update.VersioningEnabled = false
	default:
		return update, fmt.Errorf("invalid versioning status: %s", vc.Status)
	}

	return update, nil
}

// VersioningFromGCS converts GCS bucket attributes to an S3 VersioningConfiguration.
func VersioningFromGCS(attrs *storage.BucketAttrs) *model.VersioningConfiguration {
	vc := &model.VersioningConfiguration{}
	if attrs.VersioningEnabled {
		vc.Status = "Enabled"
	} else {
		vc.Status = "Suspended"
	}
	return vc
}
