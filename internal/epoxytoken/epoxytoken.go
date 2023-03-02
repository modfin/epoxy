package epoxytoken

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/modfin/epoxy/internal/basicauth"
	"github.com/modfin/epoxy/internal/extjwt"
	"github.com/modfin/epoxy/internal/log"
	"github.com/modfin/epoxy/pkg/epoxy"
	"net/http"
	"strings"
	"time"
)

type EpoxyClaims struct {
	jwt.RegisteredClaims
	ExtClaims jwt.Claims `json:"ext_claims"`
}

func MiddlewareExt(epoxyJwtKey *ecdsa.PrivateKey, subjectPath string) epoxy.Middleware {
	if epoxyJwtKey == nil {
		log.New().Fatal("epoxytoken: jwt key required")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			extClaims, err := extjwt.ExtValidationClaims(r.Context())
			if err != nil {
				log.New().WithError(fmt.Errorf("epoxytoken: %w", err)).AddToContext(r.Context())
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			subject, err := getPath(extClaims, subjectPath)
			if err != nil {
				log.New().WithError(fmt.Errorf("epoxytoken: subject not found: %w", err)).AddToContext(r.Context())
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			claims := EpoxyClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:    "epoxy",
					Subject:   subject,
					IssuedAt:  &jwt.NumericDate{Time: time.Now()},
					ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Minute)},
				},
				ExtClaims: extClaims,
			}
			epoxyJwt, err := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(epoxyJwtKey)
			if err != nil {
				log.New().WithError(err).AddToContext(r.Context())
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			r.Header.Set("Epoxy-Token", epoxyJwt)
			next.ServeHTTP(w, r)
		})
	}
}

func MiddlewareBasic(epoxyJwtKey *ecdsa.PrivateKey, allowedSuffix string) epoxy.Middleware {
	if epoxyJwtKey == nil {
		log.New().Fatal("epoxytoken: jwt key required")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, err := basicauth.Username(r.Context())
			if err != nil {
				log.New().WithError(fmt.Errorf("epoxytoken: %w", err)).AddToContext(r.Context())
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if !strings.HasSuffix(username, allowedSuffix) || strings.TrimSpace(strings.TrimSuffix(username, allowedSuffix)) == "" {
				log.New().WithError(errors.New("epoxytoken: basic auth username not allowed")).AddToContext(r.Context())
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			claims := EpoxyClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:    "epoxy",
					Subject:   username,
					IssuedAt:  &jwt.NumericDate{Time: time.Now()},
					ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Minute)},
				},
			}
			epoxyJwt, err := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(epoxyJwtKey)
			if err != nil {
				log.New().WithError(err).AddToContext(r.Context())
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			r.Header.Set("Epoxy-Token", epoxyJwt)
			next.ServeHTTP(w, r)
		})
	}
}

func getPath(a any, path string) (string, error) {
	b, err := json.Marshal(a)
	if err != nil {
		return "", err
	}
	var m any
	err = json.Unmarshal(b, &m)
	if err != nil {
		return "", err
	}
	p := strings.Split(path, ".")
	for ; len(p) > 0; p = p[1:] {
		if x, ok := m.(map[string]any); ok {
			m = x[p[0]]
		}
	}
	r, ok := m.(string)
	if ok && strings.TrimSpace(r) != "" {
		return r, nil
	}
	return "", fmt.Errorf("couldn't find string with path '%s'", path)
}
