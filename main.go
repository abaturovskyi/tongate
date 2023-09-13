package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abaturovskyi/tongate/api"
	"github.com/abaturovskyi/tongate/config"
	"github.com/uptrace/bunrouter"
	"github.com/xssnick/tonutils-go/liteclient"
)

var wait time.Duration = 15 * time.Second

func main() {
	sigChannel := make(chan os.Signal, 1)
	errs := make(chan error, 1)
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()

	lClient := liteclient.NewConnectionPool()
	err := lClient.AddConnectionsFromConfigUrl(ctx, config.Config.LiteServerConfigURL)
	if err != nil {
		log.Fatalf("connection err: %v", err.Error())
	}

	// tonApi := ton.NewAPIClient(lClient)

	router := bunrouter.New(
		bunrouter.Use(api.HeadersMiddleware()),
		bunrouter.Use(api.AuthMiddleware()),
		bunrouter.WithNotFoundHandler(notFoundHandler),
	)
	h := api.NewHandler()

	router.WithGroup("/v1", func(group *bunrouter.Group) {
		h.Register(group)
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", config.Config.APIPort),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      router,
	}

	go func() {
		errs <- srv.ListenAndServe()
	}()

	select {
	case <-sigChannel:
		shutdown(srv, wait)
	case <-errs:
		shutdown(srv, wait)
	}
}

func shutdown(srv *http.Server, wait time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	os.Exit(0)
}

func notFoundHandler(w http.ResponseWriter, req bunrouter.Request) error {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(
		w,
		"<html>can't find a route that matches <strong>%s</strong></html>",
		req.URL.Path,
	)
	return nil
}
