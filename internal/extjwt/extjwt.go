package extjwt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"text/template"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/modfin/epoxy/internal/cf"
	"github.com/modfin/epoxy/internal/log"
	"github.com/modfin/epoxy/internal/simplecache"
	"github.com/modfin/epoxy/pkg/epoxy"
	"github.com/modfin/epoxy/pkg/jwk"
)

type contextKey struct{}

type ServiceAccountConfig struct {
	Allow         bool
	EmailTemplate string
}

func Middleware(extJwkUrl string, extJwtUrl string, serviceAccount ServiceAccountConfig) epoxy.Middleware {
	if extJwkUrl == "" || extJwtUrl == "" {
		log.New().Fatal("extjwt: missing required parameters")
	}
	serviceAccountTemplate := parseServiceAccountTemplate(serviceAccount)
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
			cfClaims, err := cf.AccessClaims(r.Context())
			if err != nil {
				log.New().WithError(err).AddToContext(r.Context())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if serviceAccount.Allow && serviceAccountTemplate != nil && isServiceAccount(cfClaims) {
				claims, err := serviceAccountClaims(cfClaims, serviceAccountTemplate)
				if err != nil {
					log.New().WithError(fmt.Errorf("extjwt: error rendering service account claims: %w", err)).AddToContext(r.Context())
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				ctx := context.WithValue(r.Context(), contextKey{}, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
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

func parseServiceAccountTemplate(config ServiceAccountConfig) *template.Template {
	if !config.Allow {
		return nil
	}
	if strings.TrimSpace(config.EmailTemplate) == "" {
		log.New().Fatal("extjwt: CF_SERVICE_ACCOUNT_EMAIL_TEMPLATE required when CF_ALLOW_SERVICE_ACCOUNT=true")
	}
	t, err := template.New("service_account_email").Parse(config.EmailTemplate)
	if err != nil {
		log.New().WithError(err).Fatal("extjwt: error parsing service account email template")
	}
	return t
}

func isServiceAccount(claims cf.Claims) bool {
	return claims.Email == "" && claims.CommonName != ""
}

func serviceAccountClaims(claims cf.Claims, emailTemplate *template.Template) (jwt.MapClaims, error) {
	var b strings.Builder
	err := emailTemplate.Execute(&b, struct {
		CommonName string
	}{
		CommonName: claims.CommonName,
	})
	if err != nil {
		return nil, err
	}
	email := strings.TrimSpace(b.String())
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return nil, fmt.Errorf("invalid service account email %q: %w", email, err)
	}
	if addr.Address != email {
		return nil, fmt.Errorf("invalid service account email %q: display names are not allowed", email)
	}
	return jwt.MapClaims{
		"email":           email,
		"sub":             email,
		"service_account": true,
		"cf_common_name":  claims.CommonName,
	}, nil
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
