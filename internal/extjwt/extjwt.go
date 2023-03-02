package extjwt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/modfin/epoxy/internal/cf"
	"github.com/modfin/epoxy/internal/log"
	"github.com/modfin/epoxy/internal/simplecache"
	"github.com/modfin/epoxy/pkg/epoxy"
	"github.com/modfin/epoxy/pkg/jwk"
	"io"
	"net/http"
	"time"
)

type contextKey struct{}

func Middleware(extJwkUrl string, extJwtUrl string) epoxy.Middleware {
	if extJwkUrl == "" || extJwtUrl == "" {
		log.New().Fatal("extjwt: missing required parameters")
	}
	return func(next http.Handler) http.Handler {

		jwkCache := simplecache.New(time.Minute * 30)
		extJwtCache := simplecache.New(time.Minute * 30)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfAuth, err := cf.AccessToken(r.Context())
			if err != nil {
				log.New().WithError(err).AddToContext(r.Context())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			extJwt, err := getAndParseExtJwt(r.Context(), jwkCache, extJwtCache, extJwkUrl, extJwtUrl, cfAuth.Raw)
			if err != nil {
				log.New().WithError(fmt.Errorf("extjwt: error getting and parsing token: %w", err)).AddToContext(r.Context())
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), contextKey{}, extJwt.Claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ExtValidationClaims(ctx context.Context) (jwt.MapClaims, error) {
	if c, ok := ctx.Value(contextKey{}).(jwt.MapClaims); ok {
		return c, nil
	}
	return nil, errors.New("couldn't get external validation claims, make sure extjwt.Middleware has run")
}

func getAndParseExtJwt(ctx context.Context, jwkCache simplecache.Cache, jwtCache simplecache.Cache, extJwkUrl string, extJwtUrl string, cfAuthRaw string) (*jwt.Token, error) {
	extJwtRaw := jwtCache.Get(cfAuthRaw)
	if extJwtRaw != "" {
		extJwt, err := jwk.ParseWithUrl(ctx, jwkCache, extJwkUrl, extJwtRaw)
		if err == nil {
			return extJwt, nil
		}
	}
	extJwtRaw, err := getExtJwt(ctx, cfAuthRaw, extJwtUrl)
	if err != nil {
		return nil, err
	}
	jwtCache.Set(cfAuthRaw, extJwtRaw)

	return jwk.ParseWithUrl(ctx, jwkCache, extJwkUrl, extJwtRaw)
}

func getExtJwt(ctx context.Context, cfJwtToken, extJwtUrl string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, extJwtUrl, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfJwtToken))
	b, err := getRequestBody(req)
	if err != nil {
		return "", err
	}
	var r struct {
		Token string `json:"token"`
	}
	err = json.Unmarshal(b, &r)
	return r.Token, err
}

func getRequestBody(req *http.Request) ([]byte, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bad status: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
