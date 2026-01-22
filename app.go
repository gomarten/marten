package marten

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// App is the core of Marten.
type App struct {
	*Router
	pool       sync.Pool
	onError    func(*Ctx, error)
	onStart    []func()
	onShutdown []func()
}

// New creates a new Marten application.
func New() *App {
	app := &App{
		Router: NewRouter(),
		onError: func(c *Ctx, err error) {
			if !c.written {
				_ = c.Text(http.StatusInternalServerError, "Internal Server Error")
			}
		},
	}
	app.pool = sync.Pool{
		New: func() any {
			return &Ctx{
				params: make(map[string]string),
				store:  make(map[string]any),
			}
		},
	}
	return app
}

// OnError sets a custom error handler.
func (a *App) OnError(fn func(*Ctx, error)) {
	a.onError = fn
}

// OnStart registers a callback to run when the server starts.
func (a *App) OnStart(fn func()) {
	a.onStart = append(a.onStart, fn)
}

// OnShutdown registers a callback to run when the server shuts down.
func (a *App) OnShutdown(fn func()) {
	a.onShutdown = append(a.onShutdown, fn)
}

// SetTrailingSlash configures trailing slash handling.
func (a *App) SetTrailingSlash(mode TrailingSlashMode) {
	a.Router.SetTrailingSlash(mode)
}

// ServeHTTP implements http.Handler.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := a.pool.Get().(*Ctx)
	c.Reset(w, r)
	defer a.pool.Put(c)

	handler, routeMw, allowed, redirect := a.lookupWithTrailingSlash(r.Method, r.URL.Path, c.params)

	// Handle trailing slash redirect
	if redirect != "" {
		w.Header().Set("Location", redirect)
		w.WriteHeader(http.StatusMovedPermanently)
		return
	}

	if handler == nil {
		if len(allowed) > 0 {
			// Path exists but method not allowed
			w.Header().Set("Allow", strings.Join(allowed, ", "))
			handler = func(c *Ctx) error {
				return c.Text(http.StatusMethodNotAllowed, "Method Not Allowed")
			}
		} else {
			handler = a.notFound
		}
	}

	// Apply route-specific middleware
	if len(routeMw) > 0 {
		handler = Chain(routeMw...)(handler)
	}

	// Apply global middleware
	if len(a.middleware) > 0 {
		handler = Chain(a.middleware...)(handler)
	}

	if err := handler(c); err != nil {
		a.onError(c, err)
	}
}

// Run starts the server on the given address.
func (a *App) Run(addr string) error {
	// Run OnStart callbacks
	for _, fn := range a.onStart {
		fn()
	}

	server := &http.Server{
		Addr:    addr,
		Handler: a,
	}
	return server.ListenAndServe()
}

// RunGraceful starts the server with graceful shutdown support.
func (a *App) RunGraceful(addr string, timeout time.Duration) error {
	server := &http.Server{
		Addr:    addr,
		Handler: a,
	}

	// Run OnStart callbacks
	for _, fn := range a.onStart {
		fn()
	}

	done := make(chan error, 1)
	go func() {
		done <- server.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-done:
		if err != http.ErrServerClosed {
			return err
		}
	case <-quit:
		// Run OnShutdown callbacks
		for _, fn := range a.onShutdown {
			fn()
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		return server.Shutdown(ctx)
	}
	return nil
}
