package middleware

import (
	"log"
	"net/http"
	"strings"
)

// SignatureBypass is middleware that detects S3 v4 signature headers
// and passes the request through without validation.
// Per design requirements, signature verification is intentionally skipped.
func SignatureBypass(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "AWS4-HMAC-SHA256") {
			log.Printf("S3 v4 signature detected for %s %s (bypassed)", r.Method, r.URL.String())
		}
		next.ServeHTTP(w, r)
	})
}
