package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/MicahParks/keyfunc"
	"github.com/golang-jwt/jwt/v4"
	"github.com/modfin/epoxy/internal/log"
	"github.com/modfin/epoxy/internal/simplecache"
	"io"
	"net/http"
	"time"
)

func SwapJwtMiddleware(jwtHeader string, jwkUrl string, swapTokenUrl string) Middleware {
	if jwtHeader == "" || jwkUrl == "" || swapTokenUrl == "" {
		log.New().Fatal("missing required env variables for SwapJwtMiddleware")
	}
	return func(next http.Handler) http.Handler {

		cache := simplecache.New(time.Minute * 30)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get JWK
			jwkCacheKey := "JWK:" + time.Now().Format("2006-01-02T15")
			jwkJson := cache.Get(jwkCacheKey)
			if jwkJson == "" {
				jwkJsonBytes, err := getUrl(r.Context(), jwkUrl)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				jwkJson = string(jwkJsonBytes)
			}
			jwks, err := keyfunc.NewJSON([]byte(jwkJson))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			cache.Set(jwkCacheKey, jwkJson)

			jwtToken := r.Header.Get(jwtHeader)
			swappedToken := cache.Get("JWT:" + jwtToken)
			var token *jwt.Token
			if swappedToken != "" {
				var err error
				token, err = jwt.Parse(swappedToken, jwks.Keyfunc)
				if err != nil || !token.Valid {
					swappedToken = ""
					token = nil
				}
			}

			// JWT from [jwtHeader] -> New swapped token
			if swappedToken == "" {
				var err error
				swappedToken, err = swapToken(r.Context(), swapTokenUrl, jwtToken)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				cache.Set("JWT:"+jwtToken, swappedToken)
			}

			if token == nil {
				token, err = jwt.Parse(swappedToken, jwks.Keyfunc)
				if err != nil || !token.Valid {
					w.WriteHeader(http.StatusForbidden)
					return
				}
			}

			r.Header.Set("Epoxy-Token", swappedToken)
			next.ServeHTTP(w, r)
		})
	}
}

func swapToken(ctx context.Context, swapTokenUrl string, jwtToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, swapTokenUrl, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))
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

func getUrl(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return getRequestBody(req)
}
