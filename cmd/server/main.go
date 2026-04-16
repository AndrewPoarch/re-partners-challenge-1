// Binary pack-calculator: SQLite-backed pack sizes, JSON API, embedded UI, graceful shutdown.
// Environment: PORT (default 8080), DB_PATH (default ./app.db, /data/app.db in the container image).
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/andreigliga/pack-calculator/internal/api"
	"github.com/andreigliga/pack-calculator/internal/packsize"
	"github.com/andreigliga/pack-calculator/internal/storage"
	"github.com/andreigliga/pack-calculator/internal/webui"
)

var defaultPackSizes = []int{250, 500, 1000, 2000, 5000}

func main() {
	port := getenv("PORT", "8080")
	dbPath := getenv("DB_PATH", "./app.db")

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Fatalf("mkdir db dir: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	db, err := storage.Open(ctx, "file:"+dbPath+"?_pragma=busy_timeout(5000)")
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	packSvc := packsize.New(db)
	if err := packSvc.SeedIfEmpty(ctx, defaultPackSizes); err != nil {
		log.Fatalf("seed defaults: %v", err)
	}

	handlers := &api.Handlers{Packs: packSvc}
	router := api.NewRouter(handlers, webui.FS())

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	go func() {
		log.Printf("pack-calculator listening on :%s (db=%s)", port, dbPath)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown signal received")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
