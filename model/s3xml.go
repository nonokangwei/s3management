package model

import "encoding/xml"

// VersioningConfiguration represents the S3 XML schema for bucket versioning.
// GCS supports VersioningEnabled (bool) but does not support MfaDelete.
type VersioningConfiguration struct {
	XMLName   xml.Name `xml:"VersioningConfiguration"`
	Status    string   `xml:"Status,omitempty"`    // "Enabled" or "Suspended"
	MfaDelete string   `xml:"MfaDelete,omitempty"` // Not supported by GCS
}

// CORSConfiguration represents the S3 XML schema for bucket CORS rules.
type CORSConfiguration struct {
	XMLName  xml.Name   `xml:"CORSConfiguration"`
	CORSRule []CORSRule `xml:"CORSRule"`
}

// CORSRule represents a single CORS rule in S3 format.
// GCS supports Origins, Methods, ResponseHeaders, and MaxAge.
// S3's ID field has no GCS equivalent and is silently dropped.
// S3's AllowedHeader has no direct GCS equivalent (GCS allows all request headers).
type CORSRule struct {
	ID            string   `xml:"ID,omitempty"`
	AllowedHeader []string `xml:"AllowedHeader,omitempty"`
	AllowedMethod []string `xml:"AllowedMethod"`
	AllowedOrigin []string `xml:"AllowedOrigin"`
	ExposeHeader  []string `xml:"ExposeHeader,omitempty"`
	MaxAgeSeconds int      `xml:"MaxAgeSeconds,omitempty"`
}

// BucketLoggingStatus represents the S3 XML schema for bucket logging.
type BucketLoggingStatus struct {
	XMLName        xml.Name        `xml:"BucketLoggingStatus"`
	LoggingEnabled *LoggingEnabled `xml:"LoggingEnabled,omitempty"`
}

// LoggingEnabled contains the logging target configuration.
// GCS supports LogBucket and LogObjectPrefix but not TargetGrants.
type LoggingEnabled struct {
	TargetBucket string        `xml:"TargetBucket"`
	TargetPrefix string        `xml:"TargetPrefix"`
	TargetGrants *TargetGrants `xml:"TargetGrants,omitempty"` // Not supported by GCS
}

// TargetGrants represents S3 logging grants (not supported by GCS).
type TargetGrants struct {
	Grant []Grant `xml:"Grant"`
}

// Grant represents a single S3 logging grant entry.
type Grant struct {
	Grantee    Grantee `xml:"Grantee"`
	Permission string  `xml:"Permission"`
}

// Grantee represents the grant recipient in S3 logging.
type Grantee struct {
	Type         string `xml:"type,attr"`
	URI          string `xml:"URI,omitempty"`
	ID           string `xml:"ID,omitempty"`
	DisplayName  string `xml:"DisplayName,omitempty"`
	EmailAddress string `xml:"EmailAddress,omitempty"`
}

// Tagging represents the S3 XML schema for bucket tagging.
// GCS maps tags to bucket labels (map[string]string with lowercase keys).
type Tagging struct {
	XMLName xml.Name `xml:"Tagging"`
	TagSet  TagSet   `xml:"TagSet"`
}

// TagSet contains a list of tags.
type TagSet struct {
	Tag []Tag `xml:"Tag"`
}

// Tag represents a single key-value tag.
type Tag struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}
