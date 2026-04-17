package calculator

import (
	"errors"
	"reflect"
	"testing"
)

func allocSum(r Result) int {
	total := 0
	for _, a := range r.Packs {
		total += a.Size * a.Quantity
	}
	return total
}

func allocCount(r Result) int {
	total := 0
	for _, a := range r.Packs {
		total += a.Quantity
	}
	return total
}

func asMap(r Result) map[int]int {
	out := make(map[int]int, len(r.Packs))
	for _, a := range r.Packs {
		out[a.Size] = a.Quantity
	}
	return out
}

func TestCalculate_TableExamples(t *testing.T) {
	sizes := []int{250, 500, 1000, 2000, 5000}

	cases := []struct {
		name       string
		items      int
		wantItems  int
		wantPacks  int
		wantAlloc  map[int]int
	}{
		{"one item -> single 250 pack", 1, 250, 1, map[int]int{250: 1}},
		{"exact 250", 250, 250, 1, map[int]int{250: 1}},
		{"251 -> one 500 pack", 251, 500, 1, map[int]int{500: 1}},
		{"501 -> 500 + 250", 501, 750, 2, map[int]int{500: 1, 250: 1}},
		{"12001 -> 2x5000 + 2000 + 250", 12001, 12250, 4, map[int]int{5000: 2, 2000: 1, 250: 1}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := Calculate(tc.items, sizes)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.TotalItems != tc.wantItems {
				t.Errorf("total items = %d, want %d", got.TotalItems, tc.wantItems)
			}
			if got.TotalPacks != tc.wantPacks {
				t.Errorf("total packs = %d, want %d", got.TotalPacks, tc.wantPacks)
			}
			gotAlloc := asMap(got)
			for size, qty := range tc.wantAlloc {
				if gotAlloc[size] != qty {
					t.Errorf("allocation[%d] = %d, want %d (full=%v)", size, gotAlloc[size], qty, gotAlloc)
				}
			}
			if s := allocSum(got); s != got.TotalItems {
				t.Errorf("items sum mismatch: %d vs %d", s, got.TotalItems)
			}
			if c := allocCount(got); c != got.TotalPacks {
				t.Errorf("packs sum mismatch: %d vs %d", c, got.TotalPacks)
			}
		})
	}
}

func TestCalculate_LargeOrder_Invariants(t *testing.T) {
	sizes := []int{101, 103, 107}
	items := 99_999
	got, err := Calculate(items, sizes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalItems < items {
		t.Fatalf("shipped %d < ordered %d", got.TotalItems, items)
	}
	if s := allocSum(got); s != got.TotalItems {
		t.Fatalf("allocation sum %d vs total_items %d", s, got.TotalItems)
	}
	if c := allocCount(got); c != got.TotalPacks {
		t.Fatalf("allocation count %d vs total_packs %d", c, got.TotalPacks)
	}
	for i := items; i < got.TotalItems; i++ {
		if isReachable(i, sizes) {
			t.Fatalf("smallest shipped total should be first reachable ≥ order: %d reachable but got %d", i, got.TotalItems)
		}
	}
}

func TestCalculate_ZeroItems(t *testing.T) {
	got, err := Calculate(0, []int{250, 500})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalItems != 0 || got.TotalPacks != 0 || len(got.Packs) != 0 {
		t.Fatalf("expected empty allocation, got %+v", got)
	}
}

func TestCalculate_Validation(t *testing.T) {
	cases := []struct {
		name    string
		items   int
		sizes   []int
		wantErr error
	}{
		{"negative items", -1, []int{250}, ErrNegativeItems},
		{"empty sizes", 100, []int{}, ErrNoPackSizes},
		{"zero size", 100, []int{0, 250}, ErrInvalidSize},
		{"negative size", 100, []int{-5, 250}, ErrInvalidSize},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := Calculate(tc.items, tc.sizes)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestCalculate_DuplicateSizes(t *testing.T) {
	got, err := Calculate(1000, []int{250, 250, 500, 500, 1000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalItems != 1000 || got.TotalPacks != 1 {
		t.Fatalf("expected 1x1000, got %+v", got)
	}
}

func TestCalculate_UnsortedSizesSameAsSorted(t *testing.T) {
	sorted := []int{250, 500, 1000, 2000, 5000}
	shuffled := []int{2000, 250, 5000, 1000, 500}
	for _, n := range []int{1, 249, 251, 4999, 12001, 50_000} {
		a, errA := Calculate(n, sorted)
		b, errB := Calculate(n, shuffled)
		if errA != nil || errB != nil {
			t.Fatalf("n=%d: errA=%v errB=%v", n, errA, errB)
		}
		if a.TotalItems != b.TotalItems || a.TotalPacks != b.TotalPacks {
			t.Fatalf("n=%d: totals differ: %+v vs %+v", n, a, b)
		}
		if !reflect.DeepEqual(asMap(a), asMap(b)) {
			t.Fatalf("n=%d: allocations differ: %+v vs %+v", n, asMap(a), asMap(b))
		}
	}
}

func TestCalculate_ExtraTableExamples(t *testing.T) {
	sizes := []int{250, 500, 1000, 2000, 5000}
	cases := []struct {
		name      string
		items     int
		wantItems int
		wantPacks int
		wantAlloc map[int]int
	}{
		{
			name: "249 rounds up to one 250 pack", items: 249,
			wantItems: 250, wantPacks: 1, wantAlloc: map[int]int{250: 1},
		},
		{
			name: "499 rounds up to one 500 pack", items: 499,
			wantItems: 500, wantPacks: 1, wantAlloc: map[int]int{500: 1},
		},
		{
			name: "10000 is two 5000 packs with no overshoot", items: 10_000,
			wantItems: 10_000, wantPacks: 2, wantAlloc: map[int]int{5000: 2},
		},
		{
			name: "7500 decomposes to 5000+2000+500", items: 7500,
			wantItems: 7500, wantPacks: 3, wantAlloc: map[int]int{5000: 1, 2000: 1, 500: 1},
		},
		{
			name: "9999 needs 10000 shipped in two packs", items: 9999,
			wantItems: 10_000, wantPacks: 2, wantAlloc: map[int]int{5000: 2},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := Calculate(tc.items, sizes)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.TotalItems != tc.wantItems {
				t.Errorf("total items = %d, want %d", got.TotalItems, tc.wantItems)
			}
			if got.TotalPacks != tc.wantPacks {
				t.Errorf("total packs = %d, want %d", got.TotalPacks, tc.wantPacks)
			}
			gotAlloc := asMap(got)
			for size, qty := range tc.wantAlloc {
				if gotAlloc[size] != qty {
					t.Errorf("allocation[%d] = %d, want %d (full=%v)", size, gotAlloc[size], qty, gotAlloc)
				}
			}
		})
	}
}

func TestCalculate_CoPrimeSizesOvershoot(t *testing.T) {
	got, err := Calculate(7, []int{3, 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalItems != 8 || got.TotalPacks != 2 {
		t.Fatalf("got %+v", got)
	}
	if m := asMap(got); m[3] != 1 || m[5] != 1 {
		t.Fatalf("want one 3-pack and one 5-pack, got %v", m)
	}
}

func TestCalculate_ExactSumNoOvershootCoPrime(t *testing.T) {
	got, err := Calculate(11, []int{3, 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalItems != 11 || got.TotalPacks != 3 {
		t.Fatalf("got %+v", got)
	}
	if m := asMap(got); m[3] != 2 || m[5] != 1 {
		t.Fatalf("want 3+3+5, got %v", m)
	}
}

func TestCalculate_MinPacksAmongMinimalItems(t *testing.T) {
	got, err := Calculate(5, []int{1, 2, 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalItems != 5 || got.TotalPacks != 3 {
		t.Fatalf("got %+v", got)
	}
	if m := asMap(got); m[1] != 1 || m[2] != 2 {
		t.Fatalf("want one 1-pack and two 2-packs, got %v", m)
	}
}

func TestCalculate_EvenOnlySizes(t *testing.T) {
	got, err := Calculate(3, []int{4, 6})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalItems != 4 || got.TotalPacks != 1 {
		t.Fatalf("got %+v", got)
	}
	if m := asMap(got); m[4] != 1 {
		t.Fatalf("want one 4-pack, got %v", m)
	}
}

func TestCalculate_SingleSize(t *testing.T) {
	got, err := Calculate(1, []int{7})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalItems != 7 || got.TotalPacks != 1 {
		t.Fatalf("want 1x7, got %+v", got)
	}
}

func TestCalculate_Invariants(t *testing.T) {
	sizes := []int{250, 500, 1000, 2000, 5000}
	orders := []int{1, 249, 250, 251, 499, 500, 501, 999, 1000, 1001, 2499, 2500, 2501, 9999, 10000, 10001, 12001}

	for _, n := range orders {
		n := n
		t.Run("order", func(t *testing.T) {
			got, err := Calculate(n, sizes)
			if err != nil {
				t.Fatalf("order=%d: %v", n, err)
			}

			if got.TotalItems < n {
				t.Fatalf("order=%d shipped %d < ordered", n, got.TotalItems)
			}

			for i := n; i < got.TotalItems; i++ {
				if isReachable(i, sizes) {
					t.Fatalf("order=%d shipped %d but %d was reachable", n, got.TotalItems, i)
				}
			}
		})
	}
}

func isReachable(target int, sizes []int) bool {
	dp := make([]bool, target+1)
	dp[0] = true
	for i := 1; i <= target; i++ {
		for _, s := range sizes {
			if i-s >= 0 && dp[i-s] {
				dp[i] = true
				break
			}
		}
	}
	return dp[target]
}

func BenchmarkCalculate_LargeOrder(b *testing.B) {
	sizes := []int{101, 103, 107}
	for i := 0; i < b.N; i++ {
		_, err := Calculate(500_000, sizes)
		if err != nil {
			b.Fatal(err)
		}
	}
}
