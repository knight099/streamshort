package middleware

import (
	"log"
	"net/http"
	"net/http/httputil"
	"time"
)

// RequestLoggerMiddleware logs the incoming request and outgoing response details
func RequestLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log Request
		// DumpRequest(r, true) prints the request body too.
		// Use false if you only want headers, or check Content-Type/Size to avoid dumping huge files.
		reqDump, err := httputil.DumpRequest(r, true)
		if err != nil {
			log.Printf("Failed to dump request: %v", err)
		} else {
			log.Printf("\n--- Incoming Request ---\n%s\n------------------------", string(reqDump))
		}

		// Wrap ResponseWriter to capture status code
		rw := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		// Log Response Summary (Body logging is tricky with ResponseWriter unless you intercept Write)
		// For now, we log status code and duration.
		log.Printf("\n--- Response Summary ---\nMethod: %s\nPath: %s\nStatus: %d\nDuration: %v\n------------------------",
			r.Method, r.URL.Path, rw.statusCode, duration)
	})
}

// responseWriterWrapper captures the status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

