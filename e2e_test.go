package main

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/kangwe/s3management/gcs"
	"github.com/kangwe/s3management/middleware"
	"github.com/kangwe/s3management/server"
)

const testBucket = "s3managementtest"

// setupE2E starts the proxy server backed by a real GCS client and returns
// an AWS S3 SDK client configured to talk to the proxy.
func setupE2E(t *testing.T) (*s3.Client, *httptest.Server, func()) {
	t.Helper()

	ctx := context.Background()

	// Create a real GCS client using Application Default Credentials
	gcsClient, err := gcs.NewClient(ctx, "")
	if err != nil {
		t.Skipf("Skipping E2E tests: %v", err)
	}

	// Start proxy server with 30s GCS request timeout
	router := server.NewRouter(gcsClient, 30*time.Second)
	handler := middleware.SignatureBypass(router)
	ts := httptest.NewServer(handler)

	// Create AWS S3 SDK client pointing to our proxy with path-style addressing
	s3Client := s3.New(s3.Options{
		BaseEndpoint: aws.String(ts.URL),
		Region:       "us-east-1",
		Credentials:  credentials.NewStaticCredentialsProvider("dummy", "dummy", ""),
		UsePathStyle: true,
	})

	cleanup := func() {
		ts.Close()
		gcsClient.Close()
	}

	return s3Client, ts, cleanup
}

// TestE2E_Versioning tests PUT and GET bucket versioning against the real GCS bucket.
func TestE2E_Versioning(t *testing.T) {
	s3Client, _, cleanup := setupE2E(t)
	defer cleanup()
	ctx := context.Background()

	// Enable versioning
	_, err := s3Client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(testBucket),
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: types.BucketVersioningStatusEnabled,
		},
	})
	if err != nil {
		t.Fatalf("PutBucketVersioning (enable) failed: %v", err)
	}

	// Get versioning - should be Enabled
	getResult, err := s3Client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: aws.String(testBucket),
	})
	if err != nil {
		t.Fatalf("GetBucketVersioning failed: %v", err)
	}
	if getResult.Status != types.BucketVersioningStatusEnabled {
		t.Errorf("Versioning status = %q, want Enabled", getResult.Status)
	}

	// Suspend versioning
	_, err = s3Client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(testBucket),
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: types.BucketVersioningStatusSuspended,
		},
	})
	if err != nil {
		t.Fatalf("PutBucketVersioning (suspend) failed: %v", err)
	}

	// Get versioning - should be Suspended
	getResult, err = s3Client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: aws.String(testBucket),
	})
	if err != nil {
		t.Fatalf("GetBucketVersioning after suspend failed: %v", err)
	}
	if getResult.Status != types.BucketVersioningStatusSuspended {
		t.Errorf("Versioning status = %q, want Suspended", getResult.Status)
	}
}

// TestE2E_CORS tests PUT, GET, and DELETE bucket CORS against the real GCS bucket.
func TestE2E_CORS(t *testing.T) {
	s3Client, _, cleanup := setupE2E(t)
	defer cleanup()
	ctx := context.Background()

	// Put CORS configuration
	_, err := s3Client.PutBucketCors(ctx, &s3.PutBucketCorsInput{
		Bucket: aws.String(testBucket),
		CORSConfiguration: &types.CORSConfiguration{
			CORSRules: []types.CORSRule{
				{
					AllowedOrigins: []string{"http://example.com", "http://test.com"},
					AllowedMethods: []string{"GET", "PUT"},
					AllowedHeaders: []string{"Content-Type"},
					ExposeHeaders:  []string{"x-custom-header"},
					MaxAgeSeconds:  aws.Int32(3600),
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("PutBucketCors failed: %v", err)
	}

	// Get CORS - should match what we set
	getResult, err := s3Client.GetBucketCors(ctx, &s3.GetBucketCorsInput{
		Bucket: aws.String(testBucket),
	})
	if err != nil {
		t.Fatalf("GetBucketCors failed: %v", err)
	}
	if len(getResult.CORSRules) != 1 {
		t.Fatalf("Expected 1 CORS rule, got %d", len(getResult.CORSRules))
	}

	rule := getResult.CORSRules[0]
	if len(rule.AllowedOrigins) != 2 {
		t.Errorf("Expected 2 AllowedOrigins, got %d: %v", len(rule.AllowedOrigins), rule.AllowedOrigins)
	}
	if len(rule.AllowedMethods) != 2 {
		t.Errorf("Expected 2 AllowedMethods, got %d: %v", len(rule.AllowedMethods), rule.AllowedMethods)
	}

	// Delete CORS
	_, err = s3Client.DeleteBucketCors(ctx, &s3.DeleteBucketCorsInput{
		Bucket: aws.String(testBucket),
	})
	if err != nil {
		t.Fatalf("DeleteBucketCors failed: %v", err)
	}

	// Get CORS after delete - should be empty
	getResult, err = s3Client.GetBucketCors(ctx, &s3.GetBucketCorsInput{
		Bucket: aws.String(testBucket),
	})
	if err != nil {
		t.Fatalf("GetBucketCors after delete failed: %v", err)
	}
	if len(getResult.CORSRules) != 0 {
		t.Errorf("Expected 0 CORS rules after delete, got %d", len(getResult.CORSRules))
	}
}

// TestE2E_Tagging tests PUT, GET, and DELETE bucket tagging against the real GCS bucket.
func TestE2E_Tagging(t *testing.T) {
	s3Client, _, cleanup := setupE2E(t)
	defer cleanup()
	ctx := context.Background()

	// Put tags
	_, err := s3Client.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{
		Bucket: aws.String(testBucket),
		Tagging: &types.Tagging{
			TagSet: []types.Tag{
				{Key: aws.String("environment"), Value: aws.String("test")},
				{Key: aws.String("project"), Value: aws.String("s3management")},
			},
		},
	})
	if err != nil {
		t.Fatalf("PutBucketTagging failed: %v", err)
	}

	// Get tags - should match what we set
	getResult, err := s3Client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
		Bucket: aws.String(testBucket),
	})
	if err != nil {
		t.Fatalf("GetBucketTagging failed: %v", err)
	}
	if len(getResult.TagSet) != 2 {
		t.Fatalf("Expected 2 tags, got %d", len(getResult.TagSet))
	}

	// Verify tag values
	tags := map[string]string{}
	for _, tag := range getResult.TagSet {
		tags[*tag.Key] = *tag.Value
	}
	if tags["environment"] != "test" {
		t.Errorf("Tag environment = %q, want test", tags["environment"])
	}
	if tags["project"] != "s3management" {
		t.Errorf("Tag project = %q, want s3management", tags["project"])
	}

	// Delete tags
	_, err = s3Client.DeleteBucketTagging(ctx, &s3.DeleteBucketTaggingInput{
		Bucket: aws.String(testBucket),
	})
	if err != nil {
		t.Fatalf("DeleteBucketTagging failed: %v", err)
	}

	// Get tags after delete - should be empty
	getResult, err = s3Client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
		Bucket: aws.String(testBucket),
	})
	if err != nil {
		t.Fatalf("GetBucketTagging after delete failed: %v", err)
	}
	if len(getResult.TagSet) != 0 {
		t.Errorf("Expected 0 tags after delete, got %d", len(getResult.TagSet))
	}
}

// TestE2E_Logging tests PUT and GET bucket logging against the real GCS bucket.
// Note: S3 has no DELETE for logging. Disable logging by PUT with empty BucketLoggingStatus.
func TestE2E_Logging(t *testing.T) {
	s3Client, _, cleanup := setupE2E(t)
	defer cleanup()
	ctx := context.Background()

	// Enable logging - use the same bucket as both source and log target for testing
	_, err := s3Client.PutBucketLogging(ctx, &s3.PutBucketLoggingInput{
		Bucket: aws.String(testBucket),
		BucketLoggingStatus: &types.BucketLoggingStatus{
			LoggingEnabled: &types.LoggingEnabled{
				TargetBucket: aws.String(testBucket),
				TargetPrefix: aws.String("logs/"),
			},
		},
	})
	if err != nil {
		t.Fatalf("PutBucketLogging (enable) failed: %v", err)
	}

	// Get logging - should have logging enabled
	getResult, err := s3Client.GetBucketLogging(ctx, &s3.GetBucketLoggingInput{
		Bucket: aws.String(testBucket),
	})
	if err != nil {
		t.Fatalf("GetBucketLogging failed: %v", err)
	}
	if getResult.LoggingEnabled == nil {
		t.Fatal("Expected LoggingEnabled to be non-nil")
	}
	if *getResult.LoggingEnabled.TargetBucket != testBucket {
		t.Errorf("TargetBucket = %q, want %q", *getResult.LoggingEnabled.TargetBucket, testBucket)
	}
	if *getResult.LoggingEnabled.TargetPrefix != "logs/" {
		t.Errorf("TargetPrefix = %q, want logs/", *getResult.LoggingEnabled.TargetPrefix)
	}

	// Disable logging by PUT with empty BucketLoggingStatus (no LoggingEnabled)
	_, err = s3Client.PutBucketLogging(ctx, &s3.PutBucketLoggingInput{
		Bucket:              aws.String(testBucket),
		BucketLoggingStatus: &types.BucketLoggingStatus{},
	})
	if err != nil {
		t.Fatalf("PutBucketLogging (disable) failed: %v", err)
	}

	// Get logging after disable - should have no logging
	getResult, err = s3Client.GetBucketLogging(ctx, &s3.GetBucketLoggingInput{
		Bucket: aws.String(testBucket),
	})
	if err != nil {
		t.Fatalf("GetBucketLogging after disable failed: %v", err)
	}
	if getResult.LoggingEnabled != nil {
		t.Errorf("Expected LoggingEnabled to be nil after disable, got %+v", getResult.LoggingEnabled)
	}
}

// TestE2E_Versioning_MfaDeleteError tests that MfaDelete is rejected with InvalidArgument.
func TestE2E_Versioning_MfaDeleteError(t *testing.T) {
	s3Client, _, cleanup := setupE2E(t)
	defer cleanup()
	ctx := context.Background()

	_, err := s3Client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(testBucket),
		VersioningConfiguration: &types.VersioningConfiguration{
			Status:    types.BucketVersioningStatusEnabled,
			MFADelete: types.MFADeleteEnabled,
		},
	})
	if err == nil {
		t.Fatal("Expected error for MfaDelete, got nil")
	}
	t.Logf("MfaDelete correctly rejected: %v", err)
}
