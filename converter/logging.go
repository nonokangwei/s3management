package converter

import (
	"fmt"

	"cloud.google.com/go/storage"
	"github.com/kangwe/s3management/model"
)

// LoggingToGCS converts S3 BucketLoggingStatus to a GCS BucketAttrsToUpdate.
// Returns an error if TargetGrants is set, as GCS does not support it.
func LoggingToGCS(ls *model.BucketLoggingStatus) (storage.BucketAttrsToUpdate, error) {
	update := storage.BucketAttrsToUpdate{}

	if ls.LoggingEnabled == nil {
		// No logging configuration means disable logging
		update.Logging = &storage.BucketLogging{}
		return update, nil
	}

	// GCS does not support TargetGrants
	if ls.LoggingEnabled.TargetGrants != nil && len(ls.LoggingEnabled.TargetGrants.Grant) > 0 {
		return update, fmt.Errorf("TargetGrants is not supported by GCS")
	}

	update.Logging = &storage.BucketLogging{
		LogBucket:       ls.LoggingEnabled.TargetBucket,
		LogObjectPrefix: ls.LoggingEnabled.TargetPrefix,
	}

	return update, nil
}

// LoggingFromGCS converts GCS bucket attributes to an S3 BucketLoggingStatus.
func LoggingFromGCS(attrs *storage.BucketAttrs) *model.BucketLoggingStatus {
	ls := &model.BucketLoggingStatus{}

	if attrs.Logging != nil && attrs.Logging.LogBucket != "" {
		ls.LoggingEnabled = &model.LoggingEnabled{
			TargetBucket: attrs.Logging.LogBucket,
			TargetPrefix: attrs.Logging.LogObjectPrefix,
		}
	}

	return ls
}
