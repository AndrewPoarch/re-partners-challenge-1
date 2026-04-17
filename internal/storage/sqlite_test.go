package storage

import (
	"context"
	"errors"
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"

	db, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var n int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='pack_sizes'").Scan(&n); err != nil {
		t.Fatalf("query: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected pack_sizes table, got count=%d", n)
	}
}

func TestMigrateIdempotent(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"

	db, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate 1: %v", err)
	}
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate 2: %v", err)
	}
}

func TestInsertPackSize(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"

	db, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	_, err = db.ExecContext(ctx, "INSERT INTO pack_sizes(size) VALUES (?)", 42)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	var got int
	if err := db.QueryRowContext(ctx, "SELECT size FROM pack_sizes WHERE size = ?", 42).Scan(&got); err != nil {
		t.Fatalf("select: %v", err)
	}
	if got != 42 {
		t.Fatalf("got %d want 42", got)
	}
}

func TestOpen_ContextCancelledBeforePing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Open(ctx, "file:mem_ctx?mode=memory&cache=shared")
	if err == nil {
		t.Fatal("expected error when context is cancelled before Ping")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
