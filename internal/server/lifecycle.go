package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"qwq/internal/logger"
	"time"
)

// Start preserves the original blocking entrypoint. New callers should use Run
// so startup errors and cancellation can be handled by their owner.
func Start(address string) {
	if err := Run(context.Background(), address); err != nil {
		logger.Info("Console stopped: %v", err)
	}
}

// Run binds the listener before starting collectors and stops all work when the
// context is cancelled. Binding and serving failures are returned to the caller.
func Run(ctx context.Context, address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen %s: %w", address, err)
	}
	return serve(ctx, listener, NewHandler())
}

func serve(ctx context.Context, listener net.Listener, handler http.Handler) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return runCtx },
	}
	collectorDone := make(chan struct{})
	go func() {
		defer close(collectorDone)
		collectStatsLoopContext(runCtx)
	}()
	shutdownDone := make(chan error, 1)
	go func() {
		<-runCtx.Done()
		shutdownCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		err := server.Shutdown(shutdownCtx)
		if err != nil {
			server.Close()
		}
		shutdownDone <- err
	}()
	logger.Info("Console listening on %s", listener.Addr())
	err := server.Serve(listener)
	cancel()
	shutdownErr := <-shutdownDone
	<-collectorDone
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return shutdownErr
}
