package main

import (
	"context"
	"github.com/klauspost/compress/gzhttp"
	"github.com/modfin/epoxy/internal/cf"
	"github.com/modfin/epoxy/internal/config"
	"github.com/modfin/epoxy/internal/dev"
	"github.com/modfin/epoxy/internal/epoxytoken"
	"github.com/modfin/epoxy/internal/extjwt"
	"github.com/modfin/epoxy/internal/log"
	"github.com/modfin/epoxy/internal/nocache"
	"github.com/modfin/epoxy/pkg/epoxy"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.Get()
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
			extjwt.Middleware(cfg.ExtJwkUrl, cfg.ExtJwtUrl),
			cf.Middleware(cfg.CfAppAud, cfg.CfJwkUrl),
			nocache.Middleware,
			gzipMiddleware,
			log.Middleware,
		}
		if cfg.JwtEc256 != nil {
			middlewares = append([]epoxy.Middleware{epoxytoken.MiddlewareExt(cfg.JwtEc256, cfg.ExtJwtSubjectPath)}, middlewares...)
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
		epoxies = append(epoxies, e.WithMiddlewares(middlewares).Finalize("dev", cfg.DevAddr))
	}

	if cfg.NoAuthEnable && cfg.NoAuthAddr != "" {
		middlewares := []epoxy.Middleware{
			nocache.Middleware,
			log.Middleware,
		}
		epoxies = append(epoxies, e.WithMiddlewares(middlewares).Finalize("no-auth", cfg.NoAuthAddr))
	}

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	err = epoxy.Serve(ctx, epoxies...)
	log.New().WithError(err).Info("shutting down")
	log.Drain(context.Background())
}
