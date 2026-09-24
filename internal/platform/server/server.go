package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	httpServer  *http.Server
	database    *pgxpool.Pool
	pingTimeout time.Duration
	ready       atomic.Bool
}

func New(pingTimeout time.Duration, logger *slog.Logger, database *pgxpool.Pool) *Server {
	server := &Server{database: database, pingTimeout: pingTimeout}
	server.ready.Store(true)

	router := chi.NewRouter()
	router.Get("/healthz", server.health)
	router.Get("/readyz", server.readiness)

	server.httpServer = &http.Server{
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
	return server
}

func (s *Server) Handler() http.Handler { return s.httpServer.Handler }

func (s *Server) Serve(listener net.Listener) error { return s.httpServer.Serve(listener) }

func (s *Server) Shutdown(ctx context.Context) error {
	s.ready.Store(false)
	if err := s.httpServer.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
