package csp

import (
	"net/http"

	"github.com/modfin/epoxy/pkg/epoxy"
)

func Middleware(policy string) epoxy.Middleware {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Security-Policy", policy)
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
