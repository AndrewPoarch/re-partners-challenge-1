package packsize

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/andreigliga/pack-calculator/internal/storage"
)

func newTestDB(t *testing.T) *storage.DB {
	t.Helper()
	ctx := context.Background()
	db, err := storage.Open(ctx, "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestService_GetEmpty(t *testing.T) {
	svc := New(newTestDB(t))
	got, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestService_SeedIfEmpty(t *testing.T) {
	ctx := context.Background()
	svc := New(newTestDB(t))

	defaults := []int{250, 500, 1000, 2000, 5000}
	if err := svc.SeedIfEmpty(ctx, defaults); err != nil {
		t.Fatalf("seed: %v", err)
	}

	got, err := svc.Get(ctx)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !reflect.DeepEqual(got, defaults) {
		t.Fatalf("expected %v, got %v", defaults, got)
	}

	if err := svc.SeedIfEmpty(ctx, []int{7}); err != nil {
		t.Fatalf("second seed: %v", err)
	}
	got2, _ := svc.Get(ctx)
	if !reflect.DeepEqual(got2, defaults) {
		t.Fatalf("seed overrode existing data: %v", got2)
	}
}

func TestService_ReplaceAndGet(t *testing.T) {
	ctx := context.Background()
	svc := New(newTestDB(t))

	returned, err := svc.Replace(ctx, []int{500, 250, 500, 1000})
	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	want := []int{250, 500, 1000}
	if !reflect.DeepEqual(returned, want) {
		t.Fatalf("replace returned %v, want %v (dedup+sort)", returned, want)
	}

	got, err := svc.Get(ctx)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("get = %v, want %v", got, want)
	}
}

func TestService_ReplaceValidation(t *testing.T) {
	ctx := context.Background()
	svc := New(newTestDB(t))

	cases := []struct {
		name  string
		sizes []int
	}{
		{"empty", []int{}},
		{"zero", []int{0, 250}},
		{"negative", []int{-1, 250}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Replace(ctx, tc.sizes)
			if !errors.Is(err, ErrInvalidSizes) {
				t.Fatalf("want ErrInvalidSizes, got %v", err)
			}
		})
	}
}
