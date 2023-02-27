package middleware

import (
	"github.com/modfin/epoxy/internal/log"
	"net/http"
	"time"
)

func Logging(next http.Handler) http.Handler {
	fn := func(respWriter http.ResponseWriter, r *http.Request) {
		w := wrapped(respWriter)
		t := time.Now()
		next.ServeHTTP(w, r)
		log.New().
			WithField("path", r.URL.Path).
			WithField("latency", time.Now().Sub(t).Round(time.Millisecond).String()).
			WithField("status", w.status).
			Info("access")
	}

	return http.HandlerFunc(fn)
}

type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func wrapped(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w}
}

func (rw *responseWriter) Status() int {
	return rw.status
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}

	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
	rw.wroteHeader = true

	return
}
