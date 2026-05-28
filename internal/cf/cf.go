package cf

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/modfin/epoxy/internal/log"
	"github.com/modfin/epoxy/internal/simplecache"
	"github.com/modfin/epoxy/pkg/epoxy"
	"github.com/modfin/epoxy/pkg/jwk"
)

type contextKey struct{}

type accessToken struct {
	token  *jwt.Token
	claims Claims
}

func Middleware(cfAppAud string, cfJwksUrl string) epoxy.Middleware {
	if cfAppAud == "" || cfJwksUrl == "" {
		log.New().Fatal("cf: CF_APP_AUD and CF_JWKS_URL required")
	}
	return func(next http.Handler) http.Handler {

		jwkCache := simplecache.New(time.Minute * 30)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var claims Claims
			token, err := jwk.ParseWithUrlIntoClaims(r.Context(), jwkCache, cfJwksUrl, r.Header.Get("Cf-Access-Jwt-Assertion"), &claims)
			if err != nil {
				log.New().WithError(fmt.Errorf("cf: error parsing jwt token: %w", err)).AddToContext(r.Context())
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			foundAppAud := false
			for _, aud := range claims.Audience {
				if aud == cfAppAud {
					foundAppAud = true
				}
			}
			if !foundAppAud {
				log.New().WithError(errors.New("cf: aud not matching CF_APP_AUD")).AddToContext(r.Context())
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			log.New().
				WithField("email", claims.Email).
				WithField("service_account_common_name", claims.CommonName).
				AddToContext(r.Context())
			ctx := context.WithValue(r.Context(), contextKey{}, accessToken{token: token, claims: claims})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AccessToken(ctx context.Context) (*jwt.Token, error) {
	if a, ok := ctx.Value(contextKey{}).(accessToken); ok {
		return a.token, nil
	}
	return nil, errors.New("couldn't get parsed 'Cf-Access-Jwt-Assertion' from context, make sure cf.Middleware has run")
}

func AccessClaims(ctx context.Context) (Claims, error) {
	if a, ok := ctx.Value(contextKey{}).(accessToken); ok {
		return a.claims, nil
	}
	return Claims{}, errors.New("couldn't get parsed Cloudflare Access claims from context, make sure cf.Middleware has run")
}

type Claims struct {
	jwt.RegisteredClaims
	Email         string `json:"email"`
	Type          string `json:"type"`
	IdentityNonce string `json:"identity_nonce"`
	Country       string `json:"country"`
	CommonName    string `json:"common_name"`
}
