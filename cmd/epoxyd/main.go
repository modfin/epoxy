package main

import (
	"context"
	"github.com/modfin/epoxy/internal/basicauth"
	"github.com/modfin/epoxy/internal/cf"
	"github.com/modfin/epoxy/internal/config"
	"github.com/modfin/epoxy/internal/epoxytoken"
	"github.com/modfin/epoxy/internal/extjwt"
	"github.com/modfin/epoxy/internal/log"
	"github.com/modfin/epoxy/pkg/epoxy"
	"io/fs"
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
	var epoxies []epoxy.Epoxy

	if cfg.CfAppAud != "" {
		middlewares := []epoxy.Middleware{
			epoxytoken.MiddlewareExt(cfg.ExtJwtEcKey, cfg.ExtJwtSubjectPath),
			extjwt.Middleware(cfg.ExtJwkUrl, cfg.ExtJwtUrl),
			cf.Middleware(cfg.CfAppAud, cfg.CfJwkUrl),
			log.Middleware,
		}
		e, err := epoxy.New(cfg.CfAddr, middlewares, publicFs, cfg.PublicPrefix, cfg.Routes...)
		if err != nil {
			log.New().WithError(err).Fatal("failed to init cf epoxy")
		}
		epoxies = append(epoxies, e)
	}

	if cfg.BasicAuthPass != "" {
		middlewares := []epoxy.Middleware{
			epoxytoken.MiddlewareBasic(cfg.ExtJwtEcKey, cfg.BasicAuthUserSuffix),
			basicauth.Middleware(cfg.BasicAuthPass),
			log.Middleware,
		}
		e, err := epoxy.New(cfg.BasicAuthAddr, middlewares, publicFs, cfg.PublicPrefix, cfg.Routes...)
		if err != nil {
			log.New().WithError(err).Fatal("failed to init basic auth epoxy")
		}
		epoxies = append(epoxies, e)
	}

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	err := epoxy.Serve(ctx, epoxies...)
	log.New().WithError(err).Info("shutting down")
	log.Drain(context.Background())
}
