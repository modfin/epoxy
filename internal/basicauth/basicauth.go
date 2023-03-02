package basicauth

import (
	"context"
	"crypto/subtle"
	"errors"
	"github.com/modfin/epoxy/internal/log"
	"github.com/modfin/epoxy/pkg/epoxy"
	"net/http"
)

type contextKey struct{}

func Middleware(basicAuthPass string) epoxy.Middleware {
	if basicAuthPass == "" {
		log.New().Fatal("basicauth: password required")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok {
				log.New().WithError(errors.New("basicauth: couldn't parse Authorization header")).AddToContext(r.Context())
				w.Header().Set("WWW-Authenticate", "Basic")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if subtle.ConstantTimeCompare([]byte(basicAuthPass), []byte(password)) != 1 {
				log.New().WithError(errors.New("basicauth: wrong password")).AddToContext(r.Context())
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			log.New().WithField("basic_auth_username", username).AddToContext(r.Context())
			ctx := context.WithValue(r.Context(), contextKey{}, username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func Username(ctx context.Context) (string, error) {
	if u, ok := ctx.Value(contextKey{}).(string); ok {
		return u, nil
	}
	return "", errors.New("couldn't get basic auth username from context, make sure basicauth.Middleware has run")
}
