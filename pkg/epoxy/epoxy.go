package epoxy

import (
	"context"
	"fmt"
	"github.com/modfin/epoxy/internal/fallbackfs"
	"github.com/modfin/epoxy/internal/log"
	"github.com/modfin/epoxy/internal/middleware"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
)

func Serve(ctx context.Context, middlewares []middleware.Middleware, publicDir fs.FS, publicPrefix string, routes ...Route) error {
	mux := http.NewServeMux()

	if publicDir != nil {
		publicPrefix = path.Clean("/" + strings.TrimPrefix(publicPrefix, "/"))
		f := fallbackfs.New(publicDir, "index.html")
		h := http.StripPrefix(publicPrefix, http.FileServer(http.FS(f)))
		attachToMux(mux, publicPrefix, h)
		if publicPrefix != "/" {
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, publicPrefix, http.StatusMovedPermanently)
				return
			})
		}
		log.New().WithField("prefix", publicPrefix).Info("hosting assets directory")
	}

	for _, r := range routes {
		target, err := url.Parse(r.Target)
		if err != nil {
			return fmt.Errorf("could parse target route target: %w", err)
		}
		p := httputil.NewSingleHostReverseProxy(target)
		prefix := strings.TrimSuffix(r.Prefix, "/")
		h := http.Handler(p)
		if r.Strip {
			h = http.StripPrefix(prefix, h)
		}
		attachToMux(mux, prefix, h)
		log.New().
			WithField("prefix", r.Prefix).
			WithField("target", r.Target).
			WithField("strip", r.Strip).
			Info("hosting reverse proxy")
	}

	var handler http.Handler = mux
	for _, m := range middlewares {
		handler = m(handler)
	}
	server := &http.Server{Addr: ":8080", Handler: handler}
	return waitAll(func() error {
		<-ctx.Done()
		return server.Shutdown(context.Background())
	}, func() error {
		return server.ListenAndServe()
	})
}
