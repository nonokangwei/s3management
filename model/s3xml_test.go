package model

import (
	"encoding/xml"
	"strings"
	"testing"
)

// Test XML round-trip for VersioningConfiguration
func TestVersioningConfigurationXML(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		status string
		mfaDel string
	}{
		{
			name:   "versioning enabled",
			input:  `<VersioningConfiguration><Status>Enabled</Status></VersioningConfiguration>`,
			status: "Enabled",
		},
		{
			name:   "versioning suspended",
			input:  `<VersioningConfiguration><Status>Suspended</Status></VersioningConfiguration>`,
			status: "Suspended",
		},
		{
			name:   "versioning with MfaDelete",
			input:  `<VersioningConfiguration><Status>Enabled</Status><MfaDelete>Enabled</MfaDelete></VersioningConfiguration>`,
			status: "Enabled",
			mfaDel: "Enabled",
		},
		{
			name:   "empty versioning",
			input:  `<VersioningConfiguration></VersioningConfiguration>`,
			status: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var vc VersioningConfiguration
			if err := xml.Unmarshal([]byte(tt.input), &vc); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if vc.Status != tt.status {
				t.Errorf("Status = %q, want %q", vc.Status, tt.status)
			}
			if vc.MfaDelete != tt.mfaDel {
				t.Errorf("MfaDelete = %q, want %q", vc.MfaDelete, tt.mfaDel)
			}

			// Round-trip: marshal back and verify key fields are present
			out, err := xml.Marshal(vc)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
			if tt.status != "" && !strings.Contains(string(out), tt.status) {
				t.Errorf("Marshaled XML missing status %q: %s", tt.status, out)
			}
		})
	}
}

// Test XML round-trip for CORSConfiguration
func TestCORSConfigurationXML(t *testing.T) {
	input := `<CORSConfiguration>
  <CORSRule>
    <AllowedOrigin>http://example.com</AllowedOrigin>
    <AllowedOrigin>http://example2.com</AllowedOrigin>
    <AllowedMethod>GET</AllowedMethod>
    <AllowedMethod>PUT</AllowedMethod>
    <AllowedHeader>Content-Type</AllowedHeader>
    <ExposeHeader>x-custom-header</ExposeHeader>
    <MaxAgeSeconds>3600</MaxAgeSeconds>
    <ID>rule1</ID>
  </CORSRule>
</CORSConfiguration>`

	var cc CORSConfiguration
	if err := xml.Unmarshal([]byte(input), &cc); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(cc.CORSRule) != 1 {
		t.Fatalf("Expected 1 CORS rule, got %d", len(cc.CORSRule))
	}

	rule := cc.CORSRule[0]
	if len(rule.AllowedOrigin) != 2 {
		t.Errorf("Expected 2 AllowedOrigins, got %d", len(rule.AllowedOrigin))
	}
	if len(rule.AllowedMethod) != 2 {
		t.Errorf("Expected 2 AllowedMethods, got %d", len(rule.AllowedMethod))
	}
	if len(rule.AllowedHeader) != 1 || rule.AllowedHeader[0] != "Content-Type" {
		t.Errorf("AllowedHeader = %v, want [Content-Type]", rule.AllowedHeader)
	}
	if len(rule.ExposeHeader) != 1 || rule.ExposeHeader[0] != "x-custom-header" {
		t.Errorf("ExposeHeader = %v, want [x-custom-header]", rule.ExposeHeader)
	}
	if rule.MaxAgeSeconds != 3600 {
		t.Errorf("MaxAgeSeconds = %d, want 3600", rule.MaxAgeSeconds)
	}
	if rule.ID != "rule1" {
		t.Errorf("ID = %q, want %q", rule.ID, "rule1")
	}

	// Round-trip
	out, err := xml.Marshal(cc)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if !strings.Contains(string(out), "http://example.com") {
		t.Error("Marshaled XML missing AllowedOrigin")
	}
}

// Test XML round-trip for BucketLoggingStatus
func TestBucketLoggingStatusXML(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		hasLogging   bool
		targetBucket string
		targetPrefix string
	}{
		{
			name: "logging enabled",
			input: `<BucketLoggingStatus>
  <LoggingEnabled>
    <TargetBucket>log-bucket</TargetBucket>
    <TargetPrefix>logs/</TargetPrefix>
  </LoggingEnabled>
</BucketLoggingStatus>`,
			hasLogging:   true,
			targetBucket: "log-bucket",
			targetPrefix: "logs/",
		},
		{
			name:       "logging disabled",
			input:      `<BucketLoggingStatus></BucketLoggingStatus>`,
			hasLogging: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ls BucketLoggingStatus
			if err := xml.Unmarshal([]byte(tt.input), &ls); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if tt.hasLogging {
				if ls.LoggingEnabled == nil {
					t.Fatal("Expected LoggingEnabled to be non-nil")
				}
				if ls.LoggingEnabled.TargetBucket != tt.targetBucket {
					t.Errorf("TargetBucket = %q, want %q", ls.LoggingEnabled.TargetBucket, tt.targetBucket)
				}
				if ls.LoggingEnabled.TargetPrefix != tt.targetPrefix {
					t.Errorf("TargetPrefix = %q, want %q", ls.LoggingEnabled.TargetPrefix, tt.targetPrefix)
				}
			} else {
				if ls.LoggingEnabled != nil {
					t.Error("Expected LoggingEnabled to be nil")
				}
			}
		})
	}
}

// Test XML round-trip for Tagging
func TestTaggingXML(t *testing.T) {
	input := `<Tagging>
  <TagSet>
    <Tag>
      <Key>Environment</Key>
      <Value>Production</Value>
    </Tag>
    <Tag>
      <Key>Project</Key>
      <Value>MyApp</Value>
    </Tag>
  </TagSet>
</Tagging>`

	var tag Tagging
	if err := xml.Unmarshal([]byte(input), &tag); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(tag.TagSet.Tag) != 2 {
		t.Fatalf("Expected 2 tags, got %d", len(tag.TagSet.Tag))
	}

	if tag.TagSet.Tag[0].Key != "Environment" || tag.TagSet.Tag[0].Value != "Production" {
		t.Errorf("Tag[0] = %+v, want Key=Environment, Value=Production", tag.TagSet.Tag[0])
	}
	if tag.TagSet.Tag[1].Key != "Project" || tag.TagSet.Tag[1].Value != "MyApp" {
		t.Errorf("Tag[1] = %+v, want Key=Project, Value=MyApp", tag.TagSet.Tag[1])
	}

	// Round-trip
	out, err := xml.Marshal(tag)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if !strings.Contains(string(out), "Environment") || !strings.Contains(string(out), "Production") {
		t.Error("Marshaled XML missing tag content")
	}
}

// Test XML parsing with S3 namespace
func TestVersioningConfigurationWithNamespace(t *testing.T) {
	input := `<?xml version="1.0" encoding="UTF-8"?>
<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Status>Enabled</Status>
</VersioningConfiguration>`

	var vc VersioningConfiguration
	if err := xml.Unmarshal([]byte(input), &vc); err != nil {
		t.Fatalf("Unmarshal with namespace failed: %v", err)
	}
	if vc.Status != "Enabled" {
		t.Errorf("Status = %q, want Enabled", vc.Status)
	}
}

// Test logging with TargetGrants (unsupported by GCS)
func TestBucketLoggingWithTargetGrants(t *testing.T) {
	input := `<BucketLoggingStatus>
  <LoggingEnabled>
    <TargetBucket>log-bucket</TargetBucket>
    <TargetPrefix>logs/</TargetPrefix>
    <TargetGrants>
      <Grant>
        <Grantee type="CanonicalUser">
          <ID>user-id</ID>
        </Grantee>
        <Permission>FULL_CONTROL</Permission>
      </Grant>
    </TargetGrants>
  </LoggingEnabled>
</BucketLoggingStatus>`

	var ls BucketLoggingStatus
	if err := xml.Unmarshal([]byte(input), &ls); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if ls.LoggingEnabled == nil {
		t.Fatal("Expected LoggingEnabled to be non-nil")
	}
	if ls.LoggingEnabled.TargetGrants == nil {
		t.Fatal("Expected TargetGrants to be non-nil")
	}
	if len(ls.LoggingEnabled.TargetGrants.Grant) != 1 {
		t.Errorf("Expected 1 grant, got %d", len(ls.LoggingEnabled.TargetGrants.Grant))
	}
}
