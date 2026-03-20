package converter

import (
	"strings"

	"cloud.google.com/go/storage"
	"github.com/kangwe/s3management/model"
)

// TaggingToGCS converts S3 Tagging to GCS label set operations on BucketAttrsToUpdate.
// GCS label keys must be lowercase, so tag keys are automatically lowercased.
func TaggingToGCS(t *model.Tagging) storage.BucketAttrsToUpdate {
	update := storage.BucketAttrsToUpdate{}
	for _, tag := range t.TagSet.Tag {
		// GCS requires lowercase label keys
		key := strings.ToLower(tag.Key)
		update.SetLabel(key, tag.Value)
	}
	return update
}

// TaggingDeleteToGCS creates a BucketAttrsToUpdate that removes all existing labels.
// Requires current labels to know which keys to delete.
func TaggingDeleteToGCS(currentLabels map[string]string) storage.BucketAttrsToUpdate {
	update := storage.BucketAttrsToUpdate{}
	for key := range currentLabels {
		update.DeleteLabel(key)
	}
	return update
}

// TaggingFromGCS converts GCS bucket labels to S3 Tagging format.
func TaggingFromGCS(attrs *storage.BucketAttrs) *model.Tagging {
	t := &model.Tagging{}
	for key, value := range attrs.Labels {
		t.TagSet.Tag = append(t.TagSet.Tag, model.Tag{
			Key:   key,
			Value: value,
		})
	}
	return t
}
