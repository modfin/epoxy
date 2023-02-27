package config

import (
	"encoding/json"
	"github.com/caarlos0/env/v7"
	"github.com/modfin/epoxy/internal/log"
	"github.com/modfin/epoxy/pkg/epoxy"
	"strings"
	"sync"
)

var cfg Config
var once sync.Once

type Config struct {
	Routes       []epoxy.Route
	PublicDir    string
	PublicPrefix string
	JwtHeader    string
	JwkUrl       string
	SwapTokenUrl string
}

type config struct {
	PublicDir    string `env:"PUBLIC_DIR"`
	PublicPrefix string `env:"PUBLIC_PREFIX"`
	Routes       string `env:"ROUTES"`
	JwtHeader    string `env:"JWT_HEADER"`
	JwkUrl       string `env:"JWK_URL"`
	SwapTokenUrl string `env:"SWAP_TOKEN_URL"`
}

func Get() Config {
	once.Do(func() {
		var c config
		err := env.Parse(&c)
		if err != nil {
			log.New().WithError(err).Fatal("error parsing env")
		}
		var routes []epoxy.Route
		if c.Routes != "" {
			err = json.Unmarshal([]byte(c.Routes), &routes)
			if err != nil {
				log.New().WithError(err).Fatal("error parsing ROUTES")
			}
		}
		cfg = Config{
			Routes:       routes,
			PublicDir:    strings.TrimSpace(c.PublicDir),
			PublicPrefix: strings.TrimSpace(c.PublicPrefix),
			JwtHeader:    strings.TrimSpace(c.JwtHeader),
			JwkUrl:       strings.TrimSpace(c.JwkUrl),
			SwapTokenUrl: strings.TrimSpace(c.SwapTokenUrl),
		}
	})
	return cfg
}
