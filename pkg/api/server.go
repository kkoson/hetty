// Package api provides the HTTP server and API handler for Hetty.
package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const (
	// defaultReadTimeout is the default timeout for reading the entire request.
	defaultReadTimeout = 30 * time.Second
	// defaultWriteTimeout is the default timeout for writing the response.
	defaultWriteTimeout = 30 * time.Second
	// defaultIdleTimeout is the default timeout for idle connections.
	// Increased from 60s to 120s to reduce reconnection overhead during longer sessions.
	defaultIdleTimeout = 120 * time.Second
)

// Server represents the Hetty API HTTP server.
type Server struct {
	httpServer *http.Server
	logger     *zap.Logger
	addr       string
}

// Config holds configuration options for the API server.
type Config struct {
	// Addr is the TCP address for the server to listen on, in the form "host:port".
	Addr string
	// Logger is the structured logger instance.
	Logger *zap.Logger
	// ReadTimeout overrides the default read timeout.
	ReadTimeout time.Duration
	// WriteTimeout overrides the default write timeout.
	WriteTimeout time.Duration
	// IdleTimeout overrides the default idle timeout.
	IdleTimeout time.Duration
}

// NewServer creates a new API server with the provided configuration.
func NewServer(cfg Config) *Server {
	if cfg.Logger == nil {
		cfg.Logger, _ = zap.NewProduction()
	}
	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = defaultReadTimeout
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = defaultWriteTimeout
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = defaultIdleTimeout
	}

	mux := http.NewServeMux()
	registerRoutes(mux, cfg.Logger)

	httpServer := &http.Server{
		Addr:         cfg.Addr,
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return &Server{
		httpServer: httpServer,
		logger:     cfg.Logger,
		addr:       cfg.Addr,
	}
}

// Start begins listening and serving HTTP requests.
func (s *Server) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("api: failed to listen on %s: %w", s.addr, err)
	}

	s.logger.Info("API server listening", zap.String("addr", ln.Addr().String()))

	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("api: server error: %w", err)
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		return s.Shutdown()
	case err := <-errCh:
		return err
	}
}

// Shutdown gracefully shuts down the server without interrupting active connections.
func (s *Server) Shutdown() error {
	s.logger.Info("Shutting down API server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("api: graceful shutdown failed: %w", err)
	}
	return nil
}

// registerRoutes sets up the HTTP routes on the given mux.
func registerRoutes(mux *http.ServeMux, logger *za