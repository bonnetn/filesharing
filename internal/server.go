package internal

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
)

func RunServer(ctx context.Context, handler http.Handler) error {
	srv := http.Server{
		Addr:    ":8080",
		Handler: handler,
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		log.Printf("received SIGINT, shutting down server")

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("HTTP server Shutdown: %w", err)
		}
		close(idleConnsClosed)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server ListenAndServe: %v", err)
	}

	<-idleConnsClosed
	return nil
}
