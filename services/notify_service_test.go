package services

import (
	"encoding/json"
	"testing"

	"github.com/AxeForging/pipekit/domain"
)

func TestBuildSlackPayload_DeterministicFieldOrder(t *testing.T) {
	msg := domain.NotifyMessage{
		Status: "success",
		Title:  "Test",
		Fields: map[string]string{
			"zebra":  "z",
			"alpha":  "a",
			"middle": "m",
			"beta":   "b",
		},
	}

	first, _ := json.Marshal(BuildSlackPayload(msg))
	for i := 0; i < 20; i++ {
		next, _ := json.Marshal(BuildSlackPayload(msg))
		if string(first) != string(next) {
			t.Fatalf("payload differed across calls (run %d):\n%s\nvs\n%s",
				i, first, next)
		}
	}

	// Field text must include the keys in lexicographic order.
	out := string(first)
	if idxA := indexOf(out, "alpha"); idxA == -1 {
		t.Fatalf("alpha not in payload: %s", out)
	}
	if indexOf(out, "alpha") > indexOf(out, "beta") ||
		indexOf(out, "beta") > indexOf(out, "middle") ||
		indexOf(out, "middle") > indexOf(out, "zebra") {
		t.Errorf("fields not in sorted order: %s", out)
	}
}

func TestSortedFieldKeys(t *testing.T) {
	got := sortedFieldKeys(map[string]string{"c": "1", "a": "2", "b": "3"})
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("length mismatch")
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
