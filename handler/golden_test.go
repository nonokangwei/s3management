package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"

	"github.com/kangwe/s3management/observability"
)

func TestGoldenResponses(t *testing.T) {
	tests := []struct {
		name       string
		request    *http.Request
		mock       *mockBucketOperator
		handlerFn  func(http.ResponseWriter, *http.Request, string, *mockBucketOperator)
		goldenFile string
		status     int
	}{
		{
			name: "versioning_get",
			request: httptest.NewRequest(http.MethodGet, "/?versioning", nil).WithContext(
				observability.WithRequestID(context.Background(), "golden-versioning")),
			mock: &mockBucketOperator{
				attrs: &storage.BucketAttrs{VersioningEnabled: true},
			},
			handlerFn: func(w http.ResponseWriter, r *http.Request, bucket string, mock *mockBucketOperator) {
				GetVersioning(w, r, bucket, mock)
			},
			goldenFile: "testdata/versioning_get.golden",
			status:     http.StatusOK,
		},
		{
			name: "cors_get",
			request: httptest.NewRequest(http.MethodGet, "/?cors", nil).WithContext(
				observability.WithRequestID(context.Background(), "golden-cors")),
			mock: &mockBucketOperator{
				attrs: &storage.BucketAttrs{
					CORS: []storage.CORS{{
						Origins:         []string{"http://example.com"},
						Methods:         []string{"GET", "POST"},
						ResponseHeaders: []string{"X-Test"},
						MaxAge:          time.Hour,
					}},
				},
			},
			handlerFn: func(w http.ResponseWriter, r *http.Request, bucket string, mock *mockBucketOperator) {
				GetCORS(w, r, bucket, mock)
			},
			goldenFile: "testdata/cors_get.golden",
			status:     http.StatusOK,
		},
		{
			name: "logging_get",
			request: httptest.NewRequest(http.MethodGet, "/?logging", nil).WithContext(
				observability.WithRequestID(context.Background(), "golden-logging")),
			mock: &mockBucketOperator{
				attrs: &storage.BucketAttrs{
					Logging: &storage.BucketLogging{
						LogBucket:       "logs-bucket",
						LogObjectPrefix: "prefix/",
					},
				},
			},
			handlerFn: func(w http.ResponseWriter, r *http.Request, bucket string, mock *mockBucketOperator) {
				GetLogging(w, r, bucket, mock)
			},
			goldenFile: "testdata/logging_get.golden",
			status:     http.StatusOK,
		},
		{
			name: "tagging_get",
			request: httptest.NewRequest(http.MethodGet, "/?tagging", nil).WithContext(
				observability.WithRequestID(context.Background(), "golden-tagging")),
			mock: &mockBucketOperator{
				attrs: &storage.BucketAttrs{
					Labels: map[string]string{
						"env": "prod",
						"app": "demo",
					},
				},
			},
			handlerFn: func(w http.ResponseWriter, r *http.Request, bucket string, mock *mockBucketOperator) {
				GetTagging(w, r, bucket, mock)
			},
			goldenFile: "testdata/tagging_get.golden",
			status:     http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.handlerFn(w, tt.request, "test-bucket", tt.mock)

			if w.Code != tt.status {
				t.Fatalf("status = %d, want %d", w.Code, tt.status)
			}

			body := strings.TrimSpace(w.Body.String())
			expected := readGolden(t, tt.goldenFile)
			if body != expected {
				t.Fatalf("response mismatch\nexpected:\n%s\n\ngot:\n%s", expected, body)
			}
		})
	}
}

func readGolden(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed reading golden file %s: %v", path, err)
	}
	return strings.TrimSpace(string(data))
}
