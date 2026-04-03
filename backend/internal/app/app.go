package app

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/raywong-bitscube/stepup/backend/internal/config"
	"github.com/raywong-bitscube/stepup/backend/internal/database"
	"github.com/raywong-bitscube/stepup/backend/internal/router"
)

func Run() error {
	cfg := config.Load()

	var db *sql.DB
	if cfg.DBDSN != "" {
		var err error
		db, err = database.OpenMySQL(cfg.DBDSN)
		if err != nil {
			log.Printf("database: open failed (%v); continuing without DB-backed stores", err)
			db = nil
		}
	}
	if db != nil {
		defer func() { _ = db.Close() }()
	}

	handler := router.New(cfg, db)

	srv := &http.Server{
		Addr:              cfg.HTTPAddress(),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	case <-quit:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}
}
