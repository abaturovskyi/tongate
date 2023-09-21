package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abaturovskyi/tongate/internal/config"
	"github.com/uptrace/bunrouter"

	"github.com/abaturovskyi/tongate/api"
	"github.com/abaturovskyi/tongate/data"
	"github.com/abaturovskyi/tongate/models"
	"github.com/abaturovskyi/tongate/service"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/ton"
	"go.uber.org/zap"
)

var logger *zap.SugaredLogger
var wait time.Duration = 15 * time.Second

func main() {
	// Setup logging.
	var logConfig = zap.NewProductionConfig()
	logConfig.OutputPaths = append(logConfig.OutputPaths, config.Config.LogOutputPath)
	logConfig.DisableStacktrace = true
	l, _ := logConfig.Build()
	logger = l.Sugar()

	// Setup db connection.
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(config.Config.DatabaseURI)))
	bunDB := bun.NewDB(sqldb, pgdialect.New())

	// Setup repos.
	br := data.NewBlockRepository(bunDB)

	// Setup ton network connection pool.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()

	lClient := liteclient.NewConnectionPool()
	err := lClient.AddConnectionsFromConfigUrl(ctx, config.Config.LiteServerConfigURL)
	if err != nil {
		log.Fatalf("connection err: %v", err.Error())
	}

	// Setup block scanner.
	tonApi := ton.NewAPIClient(lClient)

	lastBlock, err := br.GetLastBlock(ctx)
	if err != nil && !errors.Is(err, data.ErrNotFound) {
		panic(err)
	}

	// Get last block from chain if there is nothing in the database.
	masterBlock := lastBlock.BlockIDExt
	if lastBlock == nil {
		masterBlock, err = tonApi.GetMasterchainInfo(ctx)
		if err != nil {
			log.Fatalf("Failed to GetMasterchainInfo error: %v", err)
		}
	}

	scanner := service.NewShardScanner(logger, models.Workchain, tonApi)
	go scanner.Start(context.Background(), masterBlock)

	// Setup router.
	router := bunrouter.New(
		bunrouter.Use(api.HeadersMiddleware()),
		bunrouter.Use(api.AuthMiddleware(config.Config.AdminToken)),
		bunrouter.WithNotFoundHandler(notFoundHandler),
	)
	h := api.NewHandler()

	router.WithGroup("/v1", func(group *bunrouter.Group) {
		h.Register(group)
	})

	// Setup server.
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", config.Config.APIPort),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      router,
	}

	// Setup shutdown signals.
	sigChannel := make(chan os.Signal, 1)
	errs := make(chan error, 1)
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)

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

	_ = srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	os.Exit(0)
}

func notFoundHandler(w http.ResponseWriter, req bunrouter.Request) error {
	w.WriteHeader(http.StatusNotFound)
	_, _ = fmt.Fprintf(
		w,
		"<html>can't find a route that matches <strong>%s</strong></html>",
		req.URL.Path,
	)
	return nil
}
