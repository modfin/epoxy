package main

import (
	"context"
	"github.com/modfin/epoxy/internal/cf"
	"github.com/modfin/epoxy/internal/config"
	"github.com/modfin/epoxy/internal/dev"
	"github.com/modfin/epoxy/internal/epoxytoken"
	"github.com/modfin/epoxy/internal/extjwt"
	"github.com/modfin/epoxy/internal/log"
	"github.com/modfin/epoxy/internal/nocache"
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
			extjwt.Middleware(cfg.ExtJwkUrl, cfg.ExtJwtUrl),
			cf.Middleware(cfg.CfAppAud, cfg.CfJwkUrl),
			log.Middleware,
			nocache.Middleware,
		}
		if cfg.JwtEc256 != nil {
			middlewares = append([]epoxy.Middleware{epoxytoken.MiddlewareExt(cfg.JwtEc256, cfg.ExtJwtSubjectPath)}, middlewares...)
		}
		e, err := epoxy.New(cfg.CfAddr, middlewares, publicFs, cfg.PublicPrefix, cfg.Routes...)
		if err != nil {
			log.New().WithError(err).Fatal("failed to init cf epoxy")
		}
		epoxies = append(epoxies, e)
	}

	if cfg.DevPass != "" {
		middlewares := []epoxy.Middleware{
			epoxytoken.MiddlewareDev(cfg.JwtEc256, cfg.DevAllowedUserSuffix),
			dev.Middleware(cfg.DevPass, cfg.DevSessionDuration, cfg.JwtEc256, cfg.JwtEc256Pub),
			log.Middleware,
			nocache.Middleware,
		}

		e, err := epoxy.New(cfg.DevAddr, middlewares, publicFs, cfg.PublicPrefix, cfg.Routes...)
		if err != nil {
			log.New().WithError(err).Fatal("failed to init dev epoxy")
		}
		epoxies = append(epoxies, e)
	}

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	err := epoxy.Serve(ctx, epoxies...)
	log.New().WithError(err).Info("shutting down")
	log.Drain(context.Background())
}
