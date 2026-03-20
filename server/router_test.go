package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractBucketName_PathStyle(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		host   string
		expect string
	}{
		{
			name:   "path style simple",
			path:   "/mybucket",
			host:   "localhost:8080",
			expect: "mybucket",
		},
		{
			name:   "path style with trailing slash",
			path:   "/mybucket/",
			host:   "localhost:8080",
			expect: "mybucket",
		},
		{
			name:   "path style with query",
			path:   "/mybucket",
			host:   "s3.amazonaws.com",
			expect: "mybucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path+"?versioning", nil)
			req.Host = tt.host
			bucket := extractBucketName(req)
			if bucket != tt.expect {
				t.Errorf("extractBucketName() = %q, want %q", bucket, tt.expect)
			}
		})
	}
}

func TestExtractBucketName_VirtualHosted(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?versioning", nil)
	req.Host = "mybucket.s3.amazonaws.com"
	bucket := extractBucketName(req)
	if bucket != "mybucket" {
		t.Errorf("extractBucketName() = %q, want mybucket", bucket)
	}
}

func TestExtractBucketName_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "localhost:8080"
	bucket := extractBucketName(req)
	if bucket != "" {
		t.Errorf("extractBucketName() = %q, want empty", bucket)
	}
}
