// Package packsize loads and replaces pack sizes in SQLite with validation (positive, deduped, sorted).
package packsize

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"github.com/andreigliga/pack-calculator/internal/storage"
)

var ErrInvalidSizes = errors.New("invalid pack sizes")

type Service struct {
	db *storage.DB
}

func New(db *storage.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Get(ctx context.Context) ([]int, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT size FROM pack_sizes ORDER BY size ASC")
	if err != nil {
		return nil, fmt.Errorf("query pack sizes: %w", err)
	}
	defer rows.Close()

	var out []int
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// Replace replaces all rows in one transaction: delete all, then insert the validated list.
func (s *Service) Replace(ctx context.Context, sizes []int) ([]int, error) {
	cleaned, err := validate(sizes)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, "DELETE FROM pack_sizes"); err != nil {
		return nil, fmt.Errorf("clear pack_sizes: %w", err)
	}
	stmt, err := tx.PrepareContext(ctx, "INSERT INTO pack_sizes(size) VALUES (?)")
	if err != nil {
		return nil, fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, v := range cleaned {
		if _, err := stmt.ExecContext(ctx, v); err != nil {
			return nil, fmt.Errorf("insert size %d: %w", v, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return cleaned, nil
}

// SeedIfEmpty inserts defaults only when the table has no rows (first run).
func (s *Service) SeedIfEmpty(ctx context.Context, defaults []int) error {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pack_sizes").Scan(&count)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("count pack_sizes: %w", err)
	}
	if count > 0 {
		return nil
	}
	_, err = s.Replace(ctx, defaults)
	return err
}

func validate(sizes []int) ([]int, error) {
	if len(sizes) == 0 {
		return nil, fmt.Errorf("%w: must contain at least one size", ErrInvalidSizes)
	}
	seen := make(map[int]struct{}, len(sizes))
	out := make([]int, 0, len(sizes))
	for _, v := range sizes {
		if v <= 0 {
			return nil, fmt.Errorf("%w: size %d must be positive", ErrInvalidSizes, v)
		}
		if _, dup := seen[v]; dup {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Ints(out)
	return out, nil
}
