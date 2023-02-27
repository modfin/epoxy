package main

import (
	"context"
	"github.com/modfin/epoxy/internal/config"
	"github.com/modfin/epoxy/internal/log"
	"github.com/modfin/epoxy/internal/middleware"
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
	middlewares := []middleware.Middleware{
		middleware.SwapJwtMiddleware(cfg.JwtHeader, cfg.JwkUrl, cfg.SwapTokenUrl),
		middleware.Logging,
	}
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	err := epoxy.Serve(ctx, middlewares, publicFs, cfg.PublicPrefix, cfg.Routes...)
	log.New().WithError(err).Info("shutting down")
	log.Drain(context.Background())
}
