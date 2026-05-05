package services

import (
	"strings"
	"testing"
	"time"
)

func TestFormatTime_NamedLayouts(t *testing.T) {
	ref := time.Date(2026, 5, 5, 12, 30, 45, 0, time.UTC)
	tests := []struct {
		format string
		want   string
	}{
		{"rfc3339", "2026-05-05T12:30:45Z"},
		{"date", "2026-05-05"},
		{"datetime", "2026-05-05 12:30:45"},
		{"compact", "20260505-123045"},
		{"tag", "20260505-1230"},
		{"unix", "1777984245"},
	}
	for _, tc := range tests {
		got := FormatTime(ref, tc.format)
		if got != tc.want {
			t.Errorf("FormatTime(%q) = %q, want %q", tc.format, got, tc.want)
		}
	}
}

func TestParseTime_RoundTrip(t *testing.T) {
	got, err := ParseTime("2026-05-05T12:30:45Z", "rfc3339")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	out := FormatTime(got, "rfc3339")
	if !strings.Contains(out, "2026-05-05") {
		t.Errorf("round-trip lost: %s", out)
	}
}

func TestAddDuration(t *testing.T) {
	base := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	d, _ := time.ParseDuration("90m")
	got := AddDuration(base, d)
	if FormatTime(got, "datetime") != "2026-05-05 13:30:00" {
		t.Errorf("got %s", FormatTime(got, "datetime"))
	}
}
