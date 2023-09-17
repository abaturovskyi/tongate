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

	"github.com/abaturovskyi/tongate/api"
	"github.com/abaturovskyi/tongate/config"
	"github.com/abaturovskyi/tongate/data"
	"github.com/abaturovskyi/tongate/models"
	"github.com/abaturovskyi/tongate/service"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bunrouter"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/ton"
	"go.uber.org/zap"
)

var logger *zap.SugaredLogger
var wait time.Duration = 15 * time.Second

func main() {
	var logConfig = zap.NewProductionConfig()

	logConfig.OutputPaths = append(logConfig.OutputPaths, config.Config.LogOutputPath)

	logConfig.DisableStacktrace = true
	l, _ := logConfig.Build()
	logger = l.Sugar()

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

	tonApi := ton.NewAPIClient(lClient)

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(config.Config.DatabaseURI)))
	bunDB := bun.NewDB(sqldb, pgdialect.New())

	br := data.NewBlockRepository(bunDB)

	lastBlock, err := br.GetLastBlock(ctx)

	if err != nil && !errors.Is(err, data.ErrNotFound) {
		panic(err)
	}

	var masterBlock *ton.BlockIDExt

	if lastBlock != nil {
		masterBlock = lastBlock.BlockIDExt
	} else {
		masterBlock, err = tonApi.GetMasterchainInfo(ctx)
		if err != nil {
			log.Fatalf("Failed to GetMasterchainInfo error: %v", err)
		}
	}

	scanner := service.NewShardScanner(logger, models.Workchain, tonApi)
	go scanner.Start(context.Background(), masterBlock)

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
