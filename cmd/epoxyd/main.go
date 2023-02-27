package main

import (
	"context"
	"github.com/modfin/epoxy/internal/cf"
	"github.com/modfin/epoxy/internal/config"
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
	var middlewares []epoxy.Middleware
	if cfg.ExtJwtUrl != "" || cfg.ExtJwkUrl != "" || cfg.ExtJwtEcKey != nil {
		middlewares = append(middlewares, extjwt.Middleware(cfg.ExtJwkUrl, cfg.ExtJwtUrl, cfg.ExtJwtEcKey))
	}
	middlewares = append(middlewares, cf.Middleware(cfg.CfAppAud, cfg.CfJwkUrl))
	middlewares = append(middlewares, log.Middleware)
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	err := epoxy.Serve(ctx, middlewares, publicFs, cfg.PublicPrefix, cfg.Routes...)
	log.New().WithError(err).Info("shutting down")
	log.Drain(context.Background())
}
