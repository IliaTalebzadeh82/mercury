package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/IliaTalebzadeh82/mercury/internal/advertiser"
	"github.com/IliaTalebzadeh82/mercury/internal/api"
	"github.com/IliaTalebzadeh82/mercury/internal/budget"
	"github.com/IliaTalebzadeh82/mercury/internal/campaign"
	"github.com/IliaTalebzadeh82/mercury/internal/decision"
	"github.com/IliaTalebzadeh82/mercury/internal/platform/config"
	"github.com/IliaTalebzadeh82/mercury/internal/platform/database"
	"github.com/IliaTalebzadeh82/mercury/internal/platform/server"
)

func main() {
	bootstrapLogger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(bootstrapLogger)

	if err := run(); err != nil {
		slog.Error("mercury stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	startupCtx, cancelStartup := context.WithTimeout(context.Background(), cfg.DatabasePingTimeout)
	pool, err := database.Open(startupCtx, cfg.DatabaseURL)
	cancelStartup()
	if err != nil {
		return err
	}
	defer pool.Close()

	advertisers := advertiser.NewStore(pool, cfg.DatabaseOperationTimeout)
	campaigns := campaign.NewStore(pool, cfg.DatabaseOperationTimeout)
	budgets := budget.NewStore(pool, cfg.DatabaseOperationTimeout)
	decisions := decision.NewEngine(pool, cfg.DatabaseOperationTimeout)
	httpAPI := api.New(logger, advertisers, campaigns, budgets, decisions, cfg.DiagnosticAPIEnabled)
	httpServer := server.New(cfg.DatabasePingTimeout, logger, pool, httpAPI)
	listener, err := net.Listen("tcp", cfg.HTTPAddress)
	if err != nil {
		return fmt.Errorf("listen on configured HTTP address: %w", err)
	}
	defer listener.Close()

	signalCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- httpServer.Serve(listener)
	}()
	logger.Info("http server listening", "address", listener.Addr().String())

	select {
	case err := <-serveErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-signalCtx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancelShutdown()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return err
	}

	if err := <-serveErrors; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	logger.Info("graceful shutdown complete")
	return nil
}
