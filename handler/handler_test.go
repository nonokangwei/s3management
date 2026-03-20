package handler

import (
	"bytes"
	"context"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/kangwe/s3management/model"
)

// mockBucketOperator implements gcs.BucketOperator for testing.
type mockBucketOperator struct {
	attrs     *storage.BucketAttrs
	attrsErr  error
	updateErr error
}

func (m *mockBucketOperator) GetBucketAttrs(ctx context.Context, bucket string) (*storage.BucketAttrs, error) {
	if m.attrsErr != nil {
		return nil, m.attrsErr
	}
	return m.attrs, nil
}

func (m *mockBucketOperator) UpdateBucket(ctx context.Context, bucket string, attrs storage.BucketAttrsToUpdate) (*storage.BucketAttrs, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return m.attrs, nil
}

// --- Versioning Handler Tests ---

func TestGetVersioning(t *testing.T) {
	mock := &mockBucketOperator{
		attrs: &storage.BucketAttrs{VersioningEnabled: true},
	}

	req := httptest.NewRequest(http.MethodGet, "/?versioning", nil)
	w := httptest.NewRecorder()

	GetVersioning(w, req, "test-bucket", mock)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "application/xml") {
		t.Errorf("Content-Type = %q, want application/xml", w.Header().Get("Content-Type"))
	}

	var vc model.VersioningConfiguration
	if err := xml.Unmarshal(w.Body.Bytes(), &vc); err != nil {
		t.Fatalf("Failed to parse response XML: %v", err)
	}
	if vc.Status != "Enabled" {
		t.Errorf("Status = %q, want Enabled", vc.Status)
	}
}

func TestPutVersioning_Enabled(t *testing.T) {
	mock := &mockBucketOperator{
		attrs: &storage.BucketAttrs{},
	}

	body := `<VersioningConfiguration><Status>Enabled</Status></VersioningConfiguration>`
	req := httptest.NewRequest(http.MethodPut, "/?versioning", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	PutVersioning(w, req, "test-bucket", mock)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestPutVersioning_MfaDeleteError(t *testing.T) {
	mock := &mockBucketOperator{attrs: &storage.BucketAttrs{}}

	body := `<VersioningConfiguration><Status>Enabled</Status><MfaDelete>Enabled</MfaDelete></VersioningConfiguration>`
	req := httptest.NewRequest(http.MethodPut, "/?versioning", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	PutVersioning(w, req, "test-bucket", mock)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var s3Err model.S3Error
	if err := xml.Unmarshal(w.Body.Bytes(), &s3Err); err != nil {
		t.Fatalf("Failed to parse error XML: %v", err)
	}
	if s3Err.Code != "InvalidArgument" {
		t.Errorf("Error code = %q, want InvalidArgument", s3Err.Code)
	}
}

func TestPutVersioning_MalformedXML(t *testing.T) {
	mock := &mockBucketOperator{attrs: &storage.BucketAttrs{}}

	body := `<not valid xml`
	req := httptest.NewRequest(http.MethodPut, "/?versioning", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	PutVersioning(w, req, "test-bucket", mock)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGetVersioning_BucketNotFound(t *testing.T) {
	mock := &mockBucketOperator{
		attrsErr: storage.ErrBucketNotExist,
	}

	req := httptest.NewRequest(http.MethodGet, "/?versioning", nil)
	w := httptest.NewRecorder()

	GetVersioning(w, req, "nonexistent-bucket", mock)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- CORS Handler Tests ---

func TestGetCORS(t *testing.T) {
	mock := &mockBucketOperator{
		attrs: &storage.BucketAttrs{
			CORS: []storage.CORS{
				{
					Origins: []string{"http://example.com"},
					Methods: []string{"GET"},
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/?cors", nil)
	w := httptest.NewRecorder()

	GetCORS(w, req, "test-bucket", mock)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var cc model.CORSConfiguration
	if err := xml.Unmarshal(w.Body.Bytes(), &cc); err != nil {
		t.Fatalf("Failed to parse response XML: %v", err)
	}
	if len(cc.CORSRule) != 1 {
		t.Errorf("Expected 1 CORS rule, got %d", len(cc.CORSRule))
	}
}

func TestPutCORS(t *testing.T) {
	mock := &mockBucketOperator{attrs: &storage.BucketAttrs{}}

	body := `<CORSConfiguration><CORSRule><AllowedOrigin>*</AllowedOrigin><AllowedMethod>GET</AllowedMethod></CORSRule></CORSConfiguration>`
	req := httptest.NewRequest(http.MethodPut, "/?cors", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	PutCORS(w, req, "test-bucket", mock)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestDeleteCORS(t *testing.T) {
	mock := &mockBucketOperator{attrs: &storage.BucketAttrs{}}

	req := httptest.NewRequest(http.MethodDelete, "/?cors", nil)
	w := httptest.NewRecorder()

	DeleteCORS(w, req, "test-bucket", mock)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

// --- Logging Handler Tests ---

func TestGetLogging(t *testing.T) {
	mock := &mockBucketOperator{
		attrs: &storage.BucketAttrs{
			Logging: &storage.BucketLogging{
				LogBucket:       "log-bucket",
				LogObjectPrefix: "logs/",
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/?logging", nil)
	w := httptest.NewRecorder()

	GetLogging(w, req, "test-bucket", mock)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var ls model.BucketLoggingStatus
	if err := xml.Unmarshal(w.Body.Bytes(), &ls); err != nil {
		t.Fatalf("Failed to parse response XML: %v", err)
	}
	if ls.LoggingEnabled == nil {
		t.Fatal("Expected LoggingEnabled to be non-nil")
	}
	if ls.LoggingEnabled.TargetBucket != "log-bucket" {
		t.Errorf("TargetBucket = %q, want log-bucket", ls.LoggingEnabled.TargetBucket)
	}
}

func TestPutLogging(t *testing.T) {
	mock := &mockBucketOperator{attrs: &storage.BucketAttrs{}}

	body := `<BucketLoggingStatus><LoggingEnabled><TargetBucket>log-bucket</TargetBucket><TargetPrefix>logs/</TargetPrefix></LoggingEnabled></BucketLoggingStatus>`
	req := httptest.NewRequest(http.MethodPut, "/?logging", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	PutLogging(w, req, "test-bucket", mock)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestPutLogging_TargetGrantsError(t *testing.T) {
	mock := &mockBucketOperator{attrs: &storage.BucketAttrs{}}

	body := `<BucketLoggingStatus><LoggingEnabled><TargetBucket>log-bucket</TargetBucket><TargetPrefix>logs/</TargetPrefix><TargetGrants><Grant><Grantee type="CanonicalUser"><ID>id</ID></Grantee><Permission>FULL_CONTROL</Permission></Grant></TargetGrants></LoggingEnabled></BucketLoggingStatus>`
	req := httptest.NewRequest(http.MethodPut, "/?logging", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	PutLogging(w, req, "test-bucket", mock)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPutLogging_DisableLogging(t *testing.T) {
	mock := &mockBucketOperator{attrs: &storage.BucketAttrs{}}

	// S3 disables logging via PUT with empty BucketLoggingStatus (no LoggingEnabled element)
	body := `<BucketLoggingStatus></BucketLoggingStatus>`
	req := httptest.NewRequest(http.MethodPut, "/?logging", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	PutLogging(w, req, "test-bucket", mock)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

// --- Tagging Handler Tests ---

func TestGetTagging(t *testing.T) {
	mock := &mockBucketOperator{
		attrs: &storage.BucketAttrs{
			Labels: map[string]string{
				"env":     "prod",
				"project": "myapp",
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/?tagging", nil)
	w := httptest.NewRecorder()

	GetTagging(w, req, "test-bucket", mock)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var tag model.Tagging
	if err := xml.Unmarshal(w.Body.Bytes(), &tag); err != nil {
		t.Fatalf("Failed to parse response XML: %v", err)
	}
	if len(tag.TagSet.Tag) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tag.TagSet.Tag))
	}
}

func TestPutTagging(t *testing.T) {
	mock := &mockBucketOperator{attrs: &storage.BucketAttrs{}}

	body := `<Tagging><TagSet><Tag><Key>env</Key><Value>prod</Value></Tag></TagSet></Tagging>`
	req := httptest.NewRequest(http.MethodPut, "/?tagging", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	PutTagging(w, req, "test-bucket", mock)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestDeleteTagging(t *testing.T) {
	mock := &mockBucketOperator{
		attrs: &storage.BucketAttrs{
			Labels: map[string]string{"env": "prod"},
		},
	}

	req := httptest.NewRequest(http.MethodDelete, "/?tagging", nil)
	w := httptest.NewRecorder()

	DeleteTagging(w, req, "test-bucket", mock)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestDeleteTagging_NoLabels(t *testing.T) {
	mock := &mockBucketOperator{
		attrs: &storage.BucketAttrs{Labels: map[string]string{}},
	}

	req := httptest.NewRequest(http.MethodDelete, "/?tagging", nil)
	w := httptest.NewRecorder()

	DeleteTagging(w, req, "test-bucket", mock)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNoContent)
	}
}
