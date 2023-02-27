package jwk

import (
	"context"
	"errors"
	"fmt"
	"github.com/MicahParks/keyfunc"
	"github.com/golang-jwt/jwt/v4"
	"io"
	"net/http"
	"time"
)

type Cache interface {
	Set(key string, value string)
	Get(key string) string
}

func ParseWithUrl(ctx context.Context, cache Cache, jwkUrl string, jwtToken string) (*jwt.Token, error) {
	return ParseWithUrlIntoClaims(ctx, cache, jwkUrl, jwtToken, nil)
}

func ParseWithUrlIntoClaims(ctx context.Context, cache Cache, jwkUrl string, jwtToken string, claims jwt.Claims) (*jwt.Token, error) {
	jwkCacheKey := time.Now().Format("2006-01-02T15")
	var jwkJson string
	if cache != nil {
		jwkJson = cache.Get(jwkCacheKey)
	}
	if jwkJson == "" {
		jwkJsonBytes, err := getUrl(ctx, jwkUrl)
		if err != nil {
			return nil, err
		}
		jwkJson = string(jwkJsonBytes)
	}
	jwks, err := keyfunc.NewJSON([]byte(jwkJson))
	if err != nil {
		return nil, err
	}
	if cache != nil {
		cache.Set(jwkCacheKey, jwkJson)
	}
	token, err := jwt.Parse(jwtToken, jwks.Keyfunc)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("token not valid")
	}
	if claims != nil {
		_, err := jwt.ParseWithClaims(jwtToken, claims, jwks.Keyfunc)
		if err != nil {
			return nil, err
		}
	}
	return token, nil
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
