package epoxy

import (
	"net/http"
	"sync"
)

func attachToMux(mux *http.ServeMux, dirPath string, handler http.Handler) {
	mux.Handle(dirPath, handler)
	mux.Handle(dirPath+"/", handler)
}

func waitAll(functions ...func() error) error {
	var wg sync.WaitGroup
	wg.Add(len(functions))
	errs := make([]error, len(functions))
	for i, f := range functions {
		i := i
		f := f
		go func() {
			errs[i] = f()
			wg.Done()
		}()
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

type Middleware func(next http.Handler) http.Handler
