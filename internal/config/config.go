package config

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"github.com/caarlos0/env/v11"
	"github.com/golang-jwt/jwt/v5"
	"github.com/modfin/epoxy/internal/log"
	"github.com/modfin/epoxy/pkg/epoxy"
	"strings"
	"sync"
	"time"
)

var cfg Config
var once sync.Once

type config struct {
	Routes string `env:"ROUTES"`

	PublicDir    string `env:"PUBLIC_DIR"`
	PublicPrefix string `env:"PUBLIC_PREFIX"`

	CfAddr    string `env:"CF_ADDR" envDefault:"127.0.0.1:8080"`
	CfJwksUrl string `env:"CF_JWKS_URL"`
	CfAppAud  string `env:"CF_APP_AUD"`

	DevAddr                string        `env:"DEV_ADDR" envDefault:":7070"`
	DevAllowedUserSuffix   string        `env:"DEV_ALLOWED_USER_SUFFIX"`
	DevBcryptHash          string        `env:"DEV_BCRYPT_HASH"`
	DevSessionDuration     time.Duration `env:"DEV_SESSION_DURATION"`
	DevDisableSecureCookie bool          `env:"DEV_DISABLE_SECURE_COOKIE"`

	ExtJwksUrl        string `env:"EXT_JWKS_URL"`
	ExtJwtUrl         string `env:"EXT_JWT_URL"`
	ExtJwtSubjectPath string `env:"EXT_JWT_SUBJECT_PATH"`

	NoAuthEnable bool   `env:"NO_AUTH_ENABLE"`
	NoAuthAddr   string `env:"NO_AUTH_ADDR"`

	JwtEc256    string `env:"JWT_EC_256"`
	JwtEc256Pub string `env:"JWT_EC_256_PUB"`
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

		if c.DevSessionDuration.Milliseconds() < 0 {
			log.New().Fatal("error: DEV_SESSION_DURATION is negative")
		}

		cfg = Config{
			Routes:                 routes,
			PublicDir:              strings.TrimSpace(c.PublicDir),
			PublicPrefix:           strings.TrimSpace(c.PublicPrefix),
			CfAddr:                 strings.TrimSpace(c.CfAddr),
			CfJwkUrl:               strings.TrimSpace(c.CfJwksUrl),
			CfAppAud:               strings.TrimSpace(c.CfAppAud),
			ExtJwkUrl:              strings.TrimSpace(c.ExtJwksUrl),
			ExtJwtUrl:              strings.TrimSpace(c.ExtJwtUrl),
			ExtJwtSubjectPath:      strings.TrimSpace(c.ExtJwtSubjectPath),
			DevAddr:                strings.TrimSpace(c.DevAddr),
			DevBcryptHash:          strings.TrimSpace(c.DevBcryptHash),
			DevAllowedUserSuffix:   strings.TrimSpace(c.DevAllowedUserSuffix),
			NoAuthEnable:           c.NoAuthEnable,
			NoAuthAddr:             strings.TrimSpace(c.NoAuthAddr),
			DevSessionDuration:     c.DevSessionDuration,
			DevDisableSecureCookie: c.DevDisableSecureCookie,
		}

		if strings.TrimSpace(c.JwtEc256) != "" {
			key, err := jwt.ParseECPrivateKeyFromPEM([]byte(strings.TrimSpace(c.JwtEc256)))
			if err != nil {
				log.New().WithError(err).Fatal("error to parsing ECDSA private key")
			}
			cfg.JwtEc256 = key
		}

		if strings.TrimSpace(c.JwtEc256Pub) != "" {
			key, err := jwt.ParseECPublicKeyFromPEM([]byte(strings.TrimSpace(c.JwtEc256Pub)))
			if err != nil {
				log.New().WithError(err).Fatal("error to parsing ECDSA public key")
			}
			cfg.JwtEc256Pub = key
		}
	})
	return cfg
}

type Config struct {
	Routes                 []epoxy.Route
	PublicDir              string
	PublicPrefix           string
	CfAddr                 string
	CfJwkUrl               string
	CfAppAud               string
	DevAddr                string
	DevAllowedUserSuffix   string
	DevBcryptHash          string
	DevSessionDuration     time.Duration
	DevDisableSecureCookie bool
	ExtJwkUrl              string
	ExtJwtUrl              string
	ExtJwtSubjectPath      string
	NoAuthEnable           bool
	NoAuthAddr             string
	JwtEc256               *ecdsa.PrivateKey
	JwtEc256Pub            *ecdsa.PublicKey
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
