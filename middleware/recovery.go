package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/kangwe/s3management/model"
)

// Recovery is middleware that catches panics in handlers and returns
// a 500 Internal Server Error instead of crashing the process.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("PANIC recovered: %v\n%s", rec, debug.Stack())
				model.WriteInternalError(w, "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
