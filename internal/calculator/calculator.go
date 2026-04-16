// Package calculator implements pack allocation for an order size and a set of pack sizes.
//
// Priority: (1) whole packs only, (2) minimise total items shipped, (3) minimise pack count.
// The implementation uses bottom-up dynamic programming, then backtracks along parent pointers.
package calculator

import (
	"errors"
	"fmt"
	"sort"
)

var (
	ErrNoPackSizes    = errors.New("pack sizes must not be empty")
	ErrInvalidSize    = errors.New("pack sizes must be positive integers")
	ErrNegativeItems  = errors.New("items must be non-negative")
	ErrUnfulfillable  = errors.New("no combination of pack sizes can fulfil the order")
)

type PackAllocation struct {
	Size     int `json:"size"`
	Quantity int `json:"quantity"`
}

type Result struct {
	Packs      []PackAllocation `json:"packs"`
	TotalItems int              `json:"total_items"`
	TotalPacks int              `json:"total_packs"`
}

// Calculate finds a feasible shipment total and pack counts. Unreachable states use -1 in packs[].
func Calculate(items int, sizes []int) (Result, error) {
	if items < 0 {
		return Result{}, ErrNegativeItems
	}
	if len(sizes) == 0 {
		return Result{}, ErrNoPackSizes
	}
	for _, s := range sizes {
		if s <= 0 {
			return Result{}, fmt.Errorf("%w: got %d", ErrInvalidSize, s)
		}
	}
	if items == 0 {
		return Result{Packs: []PackAllocation{}, TotalItems: 0, TotalPacks: 0}, nil
	}

	uniq := uniqueSortedDesc(sizes)
	maxPack := uniq[0]
	// Any optimal total is at most items + maxPack (see README algorithm note).
	upper := items + maxPack

	// packs[i] = min packs to reach exactly i items (-1 = unreachable). parent[i] = one pack used at i.
	packs := make([]int, upper+1)
	parent := make([]int, upper+1)
	for i := range packs {
		packs[i] = -1
	}
	packs[0] = 0

	for i := 1; i <= upper; i++ {
		best := -1
		bestSize := 0
		for _, s := range uniq {
			prev := i - s
			if prev < 0 {
				continue
			}
			if packs[prev] < 0 {
				continue
			}
			cand := packs[prev] + 1
			if best == -1 || cand < best {
				best = cand
				bestSize = s
			}
		}
		packs[i] = best
		parent[i] = bestSize
	}

	// First reachable total at or above the order minimises overshoot; packs[target] is then minimal.
	target := -1
	for i := items; i <= upper; i++ {
		if packs[i] >= 0 {
			target = i
			break
		}
	}
	if target == -1 {
		return Result{}, fmt.Errorf("%w: items=%d sizes=%v", ErrUnfulfillable, items, sizes)
	}

	counts := make(map[int]int, len(uniq))
	for i := target; i > 0; { // unwind parent chain
		size := parent[i]
		counts[size]++
		i -= size
	}

	out := make([]PackAllocation, 0, len(counts))
	for _, s := range uniq {
		if c, ok := counts[s]; ok {
			out = append(out, PackAllocation{Size: s, Quantity: c})
		}
	}

	return Result{
		Packs:      out,
		TotalItems: target,
		TotalPacks: packs[target],
	}, nil
}

func uniqueSortedDesc(in []int) []int {
	seen := make(map[int]struct{}, len(in))
	out := make([]int, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	return out
}
