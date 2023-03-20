package log

import (
	"bufio"
	"context"
	"errors"
	"github.com/google/uuid"
	"net"
	"net/http"
	"time"
)

type contextKey struct{}

func Middleware(next http.Handler) http.Handler {
	fn := func(respWriter http.ResponseWriter, r *http.Request) {
		w := wrapped(respWriter)
		t := time.Now()
		logFields := make(map[string]any)
		ctx := context.WithValue(r.Context(), contextKey{}, logFields)
		requestId := r.Header.Get("X-Request-Id")
		if requestId == "" {
			requestId = uuid.New().String()
			r.Header.Set("X-Request-Id", requestId)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
		New().
			WithField("path", r.URL.Path).
			WithField("latency", time.Now().Sub(t).Round(time.Millisecond).String()).
			WithField("status", w.status).
			WithField("encoding", w.Header().Get("Content-Encoding")).
			WithField("request_id", requestId).
			WithFields(logFields).
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

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("wrapped response writer doesn't support hijack")
	}
	return h.Hijack()
}
