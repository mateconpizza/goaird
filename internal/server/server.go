package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

var ErrServerAlreadyRunning = errors.New("server is already running")

type Middleware func(http.Handler, *slog.Logger) http.Handler

// Server represents an HTTP server with middleware support.
type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
	isActive   int32
}

// ServerOptFn defines functional options for Server configuration.
type ServerOptFn func(*serverOpt)

type serverOpt struct {
	addr        string
	router      *http.ServeMux
	logger      *slog.Logger
	token       string
	middlewares []Middleware
}

// WithMux sets the HTTP router/mux for the server.
func WithMux(r *http.ServeMux) ServerOptFn {
	return func(o *serverOpt) {
		o.router = r
	}
}

// WithAddr sets the server address.
func WithAddr(addr string) ServerOptFn {
	return func(o *serverOpt) {
		o.addr = addr
	}
}

// WithMiddleware adds middleware to the server.
func WithMiddleware(mw ...Middleware) ServerOptFn {
	return func(o *serverOpt) {
		o.middlewares = append(o.middlewares, mw...)
	}
}

func WithLogger(l *slog.Logger) ServerOptFn {
	return func(o *serverOpt) {
		o.logger = l
	}
}

// New creates a new Server with the given options.
func New(opts ...ServerOptFn) *Server {
	o := &serverOpt{
		router: http.NewServeMux(),
		addr:   ":8080",
	}

	// Apply options
	for _, optFn := range opts {
		optFn(o)
	}

	if o.logger == nil {
		o.logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	// Apply middlewares in reverse order (last middleware wraps first)
	var handler http.Handler = o.router
	for i := len(o.middlewares) - 1; i >= 0; i-- {
		handler = o.middlewares[i](handler, o.logger)
	}

	return &Server{
		httpServer: &http.Server{
			Addr:         o.addr,
			Handler:      handler,
			ErrorLog:     slog.NewLogLogger(o.logger.Handler(), slog.LevelError),
			IdleTimeout:  time.Minute,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		logger: o.logger,
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	if !atomic.CompareAndSwapInt32(&s.isActive, 0, 1) {
		return ErrServerAlreadyRunning
	}

	s.logger.Info("starting server", "addr", s.httpServer.Addr)

	err := s.httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.logger.Error("server start failed", "addr", s.httpServer.Addr, "error", err)
		return err
	}

	s.logger.Info("server stopped")

	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
