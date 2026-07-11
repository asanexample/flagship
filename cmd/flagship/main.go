// Command flagship is the platform feature-flag service (ADR-099): the control-plane management API plus
// the flagd sync source. It owns flag definitions + per-Environment config and projects them into flagd's
// schema; evaluation happens client-side in flagd (via OpenFeature). v1 runs against an in-memory store;
// the Postgres (CNPG) store is the next step.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asanexample/flagship/internal/api"
	"github.com/asanexample/flagship/internal/store"
	"github.com/asanexample/flagship/internal/syncsource"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	st := store.NewMemory() // TODO(phase1): swap for the Postgres store.

	mux := http.NewServeMux()
	api.API{Store: st}.Register(mux)         // control-plane management (write side)
	syncsource.HTTP{Store: st}.Register(mux) // flagd sync source (read side, per Environment)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	addr := getenv("ADDR", ":8080")
	srv := &http.Server{Addr: addr, Handler: mux, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}

	go func() {
		logger.Info("flagship listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
