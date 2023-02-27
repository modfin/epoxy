package config

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"github.com/caarlos0/env/v7"
	"github.com/golang-jwt/jwt/v4"
	"github.com/modfin/epoxy/internal/log"
	"github.com/modfin/epoxy/pkg/epoxy"
	"strings"
	"sync"
)

var cfg Config
var once sync.Once

type config struct {
	Routes string `env:"ROUTES"`

	PublicDir    string `env:"PUBLIC_DIR"`
	PublicPrefix string `env:"PUBLIC_PREFIX"`

	CfJwkUrl string `env:"CF_JWK_URL"`
	CfAppAud string `env:"CF_APP_AUD"`

	ExtJwkUrl   string `env:"EXT_JWK_URL"`
	ExtJwtUrl   string `env:"EXT_JWT_URL"`
	ExtJwtEcKey string `env:"EXT_JWT_EC_KEY"`
}

func Get() Config {
	once.Do(func() {
		var c config
		err := env.Parse(&c)
		if err != nil {
			log.New().WithError(err).Fatal("error parsing env")
		}
		var routes []epoxy.Route
		if strings.TrimSpace(c.Routes) != "" {
			var err error
			routes, err = parseRoutes(c.Routes)
			if err != nil {
				log.New().WithError(err).Fatal("error parsing ROUTES")
			}
		}

		cfg = Config{
			Routes:       routes,
			PublicDir:    strings.TrimSpace(c.PublicDir),
			PublicPrefix: strings.TrimSpace(c.PublicPrefix),
			CfJwkUrl:     strings.TrimSpace(c.CfJwkUrl),
			CfAppAud:     strings.TrimSpace(c.CfAppAud),
			ExtJwkUrl:    strings.TrimSpace(c.ExtJwkUrl),
			ExtJwtUrl:    strings.TrimSpace(c.ExtJwtUrl),
		}

		if strings.TrimSpace(c.ExtJwtEcKey) != "" {
			key, err := jwt.ParseECPrivateKeyFromPEM([]byte(strings.TrimSpace(c.ExtJwtEcKey)))
			if err != nil {
				log.New().WithError(err).Fatal("error to parsing ECDSA private key")
			}
			cfg.ExtJwtEcKey = key
		}
	})
	return cfg
}

type Config struct {
	Routes       []epoxy.Route
	PublicDir    string
	PublicPrefix string
	CfJwkUrl     string
	CfAppAud     string
	ExtJwkUrl    string
	ExtJwtUrl    string
	ExtJwtEcKey  *ecdsa.PrivateKey
}

func parseRoutes(routesString string) ([]epoxy.Route, error) {
	var routes []epoxy.Route
	err := json.Unmarshal([]byte(routesString), &routes)
	if err == nil && len(routes) > 0 {
		return routes, nil
	}
	for _, l := range strings.Split(routesString, "\n") {
		parts := strings.Fields(l)
		if len(parts) != 3 {
			return nil, errors.New("3 tokens per line required")
		}
		var strip bool
		switch strings.ToLower(parts[0]) {
		case "prefixstrip":
			strip = true
		case "prefix":
			strip = false
		default:
			return nil, errors.New("route mode required (Prefix/PrefixStrip)")
		}
		routes = append(routes, epoxy.Route{
			Strip:  strip,
			Prefix: parts[1],
			Target: parts[2],
		})
	}
	return routes, nil
}
