package epoxy

import (
	"context"
	"fmt"
	"github.com/modfin/epoxy/internal/fallbackfs"
	"github.com/modfin/epoxy/internal/log"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
)

type Epoxy interface {
	serve(ctx context.Context) error
}

func New(addr string, middlewares []Middleware, publicDir fs.FS, publicPrefix string, routes ...Route) (Epoxy, error) {
	mux := http.NewServeMux()

	proxiedRoot := false

	for _, r := range routes {
		target, err := url.Parse(r.Target)
		if err != nil {
			return nil, fmt.Errorf("could parse target route target: %w", err)
		}
		p := httputil.NewSingleHostReverseProxy(target)
		prefix := strings.TrimSuffix(r.Prefix, "/")
		h := http.Handler(p)
		if r.Strip {
			h = http.StripPrefix(prefix, h)
		}
		if prefix == "" || prefix == "/" {
			proxiedRoot = true
		}
		attachToMux(mux, prefix, h)
		log.New().
			WithField("prefix", r.Prefix).
			WithField("target", r.Target).
			WithField("strip", r.Strip).
			Info("hosting reverse proxy")
	}

	if publicDir != nil {
		publicPrefix = path.Clean("/" + strings.TrimPrefix(publicPrefix, "/"))
		f := fallbackfs.New(publicDir, "index.html")
		h := http.StripPrefix(publicPrefix, http.FileServer(http.FS(f)))
		attachToMux(mux, publicPrefix, h)
		if !proxiedRoot && publicPrefix != "/" {
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, publicPrefix, http.StatusMovedPermanently)
				return
			})
		}
		log.New().WithField("prefix", publicPrefix).Info("hosting assets directory")
	}

	var handler http.Handler = mux
	for _, m := range middlewares {
		handler = m(handler)
	}
	return &epoxy{
		Handler: handler,
		addr:    addr,
	}, nil
}

type epoxy struct {
	http.Handler
	addr string
}

func (e epoxy) serve(ctx context.Context) error {
	server := &http.Server{Addr: e.addr, Handler: e}
	return waitAll(func() error {
		<-ctx.Done()
		return server.Shutdown(context.Background())
	}, func() error {
		return server.ListenAndServe()
	})
}

func Serve(ctx context.Context, es ...Epoxy) error {
	var functions []func() error
	for _, e := range es {
		e := e
		functions = append(functions, func() error {
			return e.serve(ctx)
		})
	}
	return waitAll(functions...)
}
