package converter

import (
	"time"

	"cloud.google.com/go/storage"
	"github.com/kangwe/s3management/model"
)

// CORSToGCS converts S3 CORSConfiguration to GCS CORS rules.
// S3 AllowedHeader has no direct GCS equivalent (GCS allows all request headers) - silently accepted.
// S3 ID field has no GCS equivalent - silently dropped.
func CORSToGCS(cc *model.CORSConfiguration) []storage.CORS {
	gcsCORS := make([]storage.CORS, 0, len(cc.CORSRule))
	for _, rule := range cc.CORSRule {
		cors := storage.CORS{
			Origins:         rule.AllowedOrigin,
			Methods:         rule.AllowedMethod,
			ResponseHeaders: rule.ExposeHeader,
			MaxAge:          time.Duration(rule.MaxAgeSeconds) * time.Second,
		}
		gcsCORS = append(gcsCORS, cors)
	}
	return gcsCORS
}

// CORSFromGCS converts GCS CORS rules to S3 CORSConfiguration.
func CORSFromGCS(gcsCORS []storage.CORS) *model.CORSConfiguration {
	cc := &model.CORSConfiguration{}
	for _, cors := range gcsCORS {
		rule := model.CORSRule{
			AllowedOrigin: cors.Origins,
			AllowedMethod: cors.Methods,
			ExposeHeader:  cors.ResponseHeaders,
			MaxAgeSeconds: int(cors.MaxAge.Seconds()),
		}
		cc.CORSRule = append(cc.CORSRule, rule)
	}
	return cc
}
