package epoxy

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"

	"github.com/modfin/epoxy/internal/fallbackfs"
	"github.com/modfin/epoxy/internal/log"
)

type Epoxy interface {
	serve(ctx context.Context) error
	WithMiddlewares(middlewares []Middleware) Epoxy
	Finalize(name string, addr string) Epoxy
}

func New(publicDir fs.FS, publicPrefix string, routes ...Route) (Epoxy, error) {
	mux := http.NewServeMux()

	proxiedRoot := false

	for _, r := range routes {
		target, err := url.Parse(r.Target)
		if err != nil {
			return nil, fmt.Errorf("could parse target route target: %w", err)
		}
		p := httputil.NewSingleHostReverseProxy(target)

		// attempt to fix issue with Host header NOT being set to target host on reverse proxy
		if r.RewriteHost {
			p = &httputil.ReverseProxy{
				Rewrite: func(pr *httputil.ProxyRequest) {
					pr.SetURL(target)
				},
			}
		}

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
	return &epoxy{
		Handler: mux,
	}, nil
}

type epoxy struct {
	http.Handler
	name string
	addr string
}

func (e epoxy) WithMiddlewares(middlewares []Middleware) Epoxy {
	var handler = e.Handler
	for _, m := range middlewares {
		handler = m(handler)
	}
	return &epoxy{
		Handler: handler,
		addr:    e.addr,
		name:    e.name,
	}
}

func (e epoxy) Finalize(name string, addr string) Epoxy {
	return &epoxy{
		Handler: e.Handler,
		addr:    addr,
		name:    name,
	}
}

func (e epoxy) serve(ctx context.Context) error {
	if e.addr == "" {
		return errors.New("must call Finalize on epoxy before serving")
	}
	log.New().WithField("addr", e.addr).Info(fmt.Sprintf("[%s] listening", e.name))
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
