package services

import (
	"reflect"
	"testing"
)

func TestMatrixShard(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e", "f", "g"}
	tests := []struct {
		total, index int
		want         []string
	}{
		{2, 0, []string{"a", "c", "e", "g"}},
		{2, 1, []string{"b", "d", "f"}},
		{3, 0, []string{"a", "d", "g"}},
		{3, 1, []string{"b", "e"}},
		{3, 2, []string{"c", "f"}},
	}
	for _, tc := range tests {
		got, err := MatrixShard(items, tc.total, tc.index)
		if err != nil {
			t.Errorf("MatrixShard(%d,%d) error: %v", tc.total, tc.index, err)
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("MatrixShard(%d,%d) = %v, want %v", tc.total, tc.index, got, tc.want)
		}
	}

	// Union of all shards covers every item exactly once.
	total := 4
	covered := map[string]int{}
	for i := 0; i < total; i++ {
		got, _ := MatrixShard(items, total, i)
		for _, item := range got {
			covered[item]++
		}
	}
	for _, item := range items {
		if covered[item] != 1 {
			t.Errorf("item %s covered %d times, want 1", item, covered[item])
		}
	}
}

func TestMatrixShard_BadInput(t *testing.T) {
	if _, err := MatrixShard(nil, 0, 0); err == nil {
		t.Error("expected error for total=0")
	}
	if _, err := MatrixShard(nil, 3, 5); err == nil {
		t.Error("expected error for index>=total")
	}
}
