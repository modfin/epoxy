package main

import (
	"context"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/klauspost/compress/gzhttp"
	"github.com/modfin/epoxy/internal/cf"
	"github.com/modfin/epoxy/internal/config"
	"github.com/modfin/epoxy/internal/csp"
	"github.com/modfin/epoxy/internal/dev"
	"github.com/modfin/epoxy/internal/epoxytoken"
	"github.com/modfin/epoxy/internal/extjwt"
	"github.com/modfin/epoxy/internal/log"
	"github.com/modfin/epoxy/internal/nocache"
	"github.com/modfin/epoxy/pkg/epoxy"
)

func main() {
	cfg := config.Get()

	log.New().
		WithField("CF_ADDR", cfg.CfAddr).
		WithField("CF_JWKS_URL", cfg.CfJwkUrl).
		WithField("CF_APP_AUD", cfg.CfAppAud).
		WithField("CF_SERVICE_TOKEN_ALLOWLIST", cfg.CfServiceTokenAllowlist).
		WithField("EXT_JWKS_URL", cfg.ExtJwkUrl).
		WithField("EXT_JWT_URL", cfg.ExtJwtUrl).
		WithField("EXT_JWT_SUBJECT_PATH", cfg.ExtJwtSubjectPath).
		WithField("DEV_ADDR", cfg.DevAddr).
		WithField("DEV_ALLOWED_USER_SUFFIX", cfg.DevAllowedUserSuffix).
		WithField("DEV_SESSION_DURATION", cfg.DevSessionDuration).
		WithField("DEV_DISABLE_SECURE_COOKIE", cfg.DevDisableSecureCookie).
		WithField("NO_AUTH_ENABLE", cfg.NoAuthEnable).
		WithField("NO_AUTH_ADDR", cfg.NoAuthAddr).
		WithField("PUBLIC_DIR", cfg.PublicDir).
		WithField("PUBLIC_PREFIX", cfg.PublicPrefix).
		WithField("CONTENT_SECURITY_POLICY", cfg.ContentSecurityPolicy).
		WithField("ROUTES", cfg.Routes).
		Info("epoxy config")

	var publicFs fs.FS
	if cfg.PublicDir != "" {
		publicFs = os.DirFS(cfg.PublicDir)
	}
	e, err := epoxy.New(publicFs, cfg.PublicPrefix, cfg.Routes...)
	if err != nil {
		log.New().WithError(err).Fatal("failed to init epoxy")
	}

	var epoxies []epoxy.Epoxy

	if cfg.CfAppAud != "" {
		var gzipMiddleware = func(h http.Handler) http.Handler {
			return gzhttp.GzipHandler(h)
		}
		middlewares := []epoxy.Middleware{
			extjwt.Middleware(cfg.ExtJwkUrl, cfg.ExtJwtUrl, cfg.CfServiceTokenAllowlist),
			cf.Middleware(cfg.CfAppAud, cfg.CfJwkUrl),
			nocache.Middleware,
			gzipMiddleware,
			log.Middleware,
		}
		if cfg.JwtEc256 != nil {
			middlewares = append([]epoxy.Middleware{epoxytoken.MiddlewareExt(cfg.JwtEc256, cfg.ExtJwtSubjectPath)}, middlewares...)
		}
		if cfg.ContentSecurityPolicy != "" {
			middlewares = append([]epoxy.Middleware{csp.Middleware(cfg.ContentSecurityPolicy)}, middlewares...)
		}
		epoxies = append(epoxies, e.WithMiddlewares(middlewares).Finalize("cf", cfg.CfAddr))
	}

	if cfg.DevBcryptHash != "" {
		middlewares := []epoxy.Middleware{
			epoxytoken.MiddlewareDev(cfg.JwtEc256, cfg.DevAllowedUserSuffix),
			dev.Middleware(cfg.DevBcryptHash, cfg.DevSessionDuration, cfg.JwtEc256, cfg.JwtEc256Pub, cfg.DevDisableSecureCookie),
			nocache.Middleware,
			log.Middleware,
		}
		if cfg.ContentSecurityPolicy != "" {
			middlewares = append([]epoxy.Middleware{csp.Middleware(cfg.ContentSecurityPolicy)}, middlewares...)
		}
		epoxies = append(epoxies, e.WithMiddlewares(middlewares).Finalize("dev", cfg.DevAddr))
	}

	if cfg.NoAuthEnable && cfg.NoAuthAddr != "" {
		middlewares := []epoxy.Middleware{
			nocache.Middleware,
			log.Middleware,
		}
		if cfg.ContentSecurityPolicy != "" {
			middlewares = append([]epoxy.Middleware{csp.Middleware(cfg.ContentSecurityPolicy)}, middlewares...)
		}
		epoxies = append(epoxies, e.WithMiddlewares(middlewares).Finalize("no-auth", cfg.NoAuthAddr))
	}

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	err = epoxy.Serve(ctx, epoxies...)
	log.New().WithError(err).Info("shutting down")
	log.Drain(context.Background())
}
