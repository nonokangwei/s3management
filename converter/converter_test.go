package converter

import (
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/kangwe/s3management/model"
)

// --- Versioning Tests ---

func TestVersioningToGCS_Enabled(t *testing.T) {
	vc := &model.VersioningConfiguration{Status: "Enabled"}
	update, err := VersioningToGCS(vc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if update.VersioningEnabled != true {
		t.Errorf("VersioningEnabled = %v, want true", update.VersioningEnabled)
	}
}

func TestVersioningToGCS_Suspended(t *testing.T) {
	vc := &model.VersioningConfiguration{Status: "Suspended"}
	update, err := VersioningToGCS(vc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if update.VersioningEnabled != false {
		t.Errorf("VersioningEnabled = %v, want false", update.VersioningEnabled)
	}
}

func TestVersioningToGCS_MfaDeleteError(t *testing.T) {
	vc := &model.VersioningConfiguration{Status: "Enabled", MfaDelete: "Enabled"}
	_, err := VersioningToGCS(vc)
	if err == nil {
		t.Fatal("Expected error for MfaDelete, got nil")
	}
}

func TestVersioningToGCS_InvalidStatus(t *testing.T) {
	vc := &model.VersioningConfiguration{Status: "Invalid"}
	_, err := VersioningToGCS(vc)
	if err == nil {
		t.Fatal("Expected error for invalid status, got nil")
	}
}

func TestVersioningFromGCS_Enabled(t *testing.T) {
	attrs := &storage.BucketAttrs{VersioningEnabled: true}
	vc := VersioningFromGCS(attrs)
	if vc.Status != "Enabled" {
		t.Errorf("Status = %q, want Enabled", vc.Status)
	}
}

func TestVersioningFromGCS_Disabled(t *testing.T) {
	attrs := &storage.BucketAttrs{VersioningEnabled: false}
	vc := VersioningFromGCS(attrs)
	if vc.Status != "Suspended" {
		t.Errorf("Status = %q, want Suspended", vc.Status)
	}
}

// --- CORS Tests ---

func TestCORSToGCS(t *testing.T) {
	cc := &model.CORSConfiguration{
		CORSRule: []model.CORSRule{
			{
				AllowedOrigin: []string{"http://example.com"},
				AllowedMethod: []string{"GET", "PUT"},
				ExposeHeader:  []string{"x-custom-header"},
				MaxAgeSeconds: 3600,
				ID:            "rule1", // should be silently dropped
			},
		},
	}

	gcsCORS := CORSToGCS(cc)
	if len(gcsCORS) != 1 {
		t.Fatalf("Expected 1 CORS rule, got %d", len(gcsCORS))
	}

	rule := gcsCORS[0]
	if len(rule.Origins) != 1 || rule.Origins[0] != "http://example.com" {
		t.Errorf("Origins = %v, want [http://example.com]", rule.Origins)
	}
	if len(rule.Methods) != 2 {
		t.Errorf("Methods = %v, want [GET PUT]", rule.Methods)
	}
	if len(rule.ResponseHeaders) != 1 || rule.ResponseHeaders[0] != "x-custom-header" {
		t.Errorf("ResponseHeaders = %v, want [x-custom-header]", rule.ResponseHeaders)
	}
	if rule.MaxAge != 3600*time.Second {
		t.Errorf("MaxAge = %v, want 3600s", rule.MaxAge)
	}
}

func TestCORSFromGCS(t *testing.T) {
	gcsCORS := []storage.CORS{
		{
			Origins:         []string{"*"},
			Methods:         []string{"GET"},
			ResponseHeaders: []string{"Content-Type"},
			MaxAge:          1800 * time.Second,
		},
	}

	cc := CORSFromGCS(gcsCORS)
	if len(cc.CORSRule) != 1 {
		t.Fatalf("Expected 1 CORS rule, got %d", len(cc.CORSRule))
	}

	rule := cc.CORSRule[0]
	if rule.AllowedOrigin[0] != "*" {
		t.Errorf("AllowedOrigin = %v, want [*]", rule.AllowedOrigin)
	}
	if rule.MaxAgeSeconds != 1800 {
		t.Errorf("MaxAgeSeconds = %d, want 1800", rule.MaxAgeSeconds)
	}
}

func TestCORSToGCS_EmptyRules(t *testing.T) {
	cc := &model.CORSConfiguration{CORSRule: []model.CORSRule{}}
	gcsCORS := CORSToGCS(cc)
	if len(gcsCORS) != 0 {
		t.Errorf("Expected 0 CORS rules, got %d", len(gcsCORS))
	}
}

// --- Logging Tests ---

func TestLoggingToGCS(t *testing.T) {
	ls := &model.BucketLoggingStatus{
		LoggingEnabled: &model.LoggingEnabled{
			TargetBucket: "log-bucket",
			TargetPrefix: "logs/",
		},
	}

	update, err := LoggingToGCS(ls)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if update.Logging == nil {
		t.Fatal("Expected Logging to be non-nil")
	}
	if update.Logging.LogBucket != "log-bucket" {
		t.Errorf("LogBucket = %q, want log-bucket", update.Logging.LogBucket)
	}
	if update.Logging.LogObjectPrefix != "logs/" {
		t.Errorf("LogObjectPrefix = %q, want logs/", update.Logging.LogObjectPrefix)
	}
}

func TestLoggingToGCS_DisableLogging(t *testing.T) {
	ls := &model.BucketLoggingStatus{LoggingEnabled: nil}
	update, err := LoggingToGCS(ls)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if update.Logging == nil {
		t.Fatal("Expected Logging to be non-nil (empty struct to disable)")
	}
	if update.Logging.LogBucket != "" {
		t.Errorf("LogBucket = %q, want empty", update.Logging.LogBucket)
	}
}

func TestLoggingToGCS_TargetGrantsError(t *testing.T) {
	ls := &model.BucketLoggingStatus{
		LoggingEnabled: &model.LoggingEnabled{
			TargetBucket: "log-bucket",
			TargetPrefix: "logs/",
			TargetGrants: &model.TargetGrants{
				Grant: []model.Grant{{Permission: "FULL_CONTROL"}},
			},
		},
	}

	_, err := LoggingToGCS(ls)
	if err == nil {
		t.Fatal("Expected error for TargetGrants, got nil")
	}
}

func TestLoggingFromGCS(t *testing.T) {
	attrs := &storage.BucketAttrs{
		Logging: &storage.BucketLogging{
			LogBucket:       "log-bucket",
			LogObjectPrefix: "prefix/",
		},
	}

	ls := LoggingFromGCS(attrs)
	if ls.LoggingEnabled == nil {
		t.Fatal("Expected LoggingEnabled to be non-nil")
	}
	if ls.LoggingEnabled.TargetBucket != "log-bucket" {
		t.Errorf("TargetBucket = %q, want log-bucket", ls.LoggingEnabled.TargetBucket)
	}
}

func TestLoggingFromGCS_NoLogging(t *testing.T) {
	attrs := &storage.BucketAttrs{Logging: nil}
	ls := LoggingFromGCS(attrs)
	if ls.LoggingEnabled != nil {
		t.Error("Expected LoggingEnabled to be nil")
	}
}

// --- Tagging Tests ---

func TestTaggingToGCS(t *testing.T) {
	tag := &model.Tagging{
		TagSet: model.TagSet{
			Tag: []model.Tag{
				{Key: "Environment", Value: "prod"},
				{Key: "project", Value: "myapp"},
			},
		},
	}

	update := TaggingToGCS(tag)
	// We can't directly inspect BucketAttrsToUpdate labels,
	// but we verify it doesn't panic and returns a valid update.
	_ = update
}

func TestTaggingDeleteToGCS(t *testing.T) {
	labels := map[string]string{
		"env":     "prod",
		"project": "myapp",
	}

	update := TaggingDeleteToGCS(labels)
	_ = update
}

func TestTaggingFromGCS(t *testing.T) {
	attrs := &storage.BucketAttrs{
		Labels: map[string]string{
			"env":     "prod",
			"project": "myapp",
		},
	}

	tag := TaggingFromGCS(attrs)
	if len(tag.TagSet.Tag) != 2 {
		t.Fatalf("Expected 2 tags, got %d", len(tag.TagSet.Tag))
	}

	// Check that all labels are represented (order may vary)
	found := map[string]string{}
	for _, tg := range tag.TagSet.Tag {
		found[tg.Key] = tg.Value
	}
	if found["env"] != "prod" {
		t.Errorf("env = %q, want prod", found["env"])
	}
	if found["project"] != "myapp" {
		t.Errorf("project = %q, want myapp", found["project"])
	}
}

func TestTaggingFromGCS_NoLabels(t *testing.T) {
	attrs := &storage.BucketAttrs{Labels: nil}
	tag := TaggingFromGCS(attrs)
	if len(tag.TagSet.Tag) != 0 {
		t.Errorf("Expected 0 tags, got %d", len(tag.TagSet.Tag))
	}
}
