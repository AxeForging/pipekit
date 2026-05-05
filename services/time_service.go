package services

import (
	"fmt"
	"strings"
	"time"
)

// TimeNamedLayouts maps friendly names to Go time layouts.
var TimeNamedLayouts = map[string]string{
	"rfc3339":  time.RFC3339,
	"rfc1123":  time.RFC1123,
	"rfc822":   time.RFC822,
	"unix":     "unix",  // sentinel handled in FormatTime
	"unix-ms":  "unixms",
	"compact":  "20060102-150405",
	"date":     "2006-01-02",
	"datetime": "2006-01-02 15:04:05",
	"tag":      "20060102-1504",
	"iso":      "2006-01-02T15:04:05Z07:00",
}

// FormatTime renders t using the named layout or a raw Go time layout.
func FormatTime(t time.Time, format string) string {
	layout := resolveLayout(format)
	switch layout {
	case "unix":
		return fmt.Sprintf("%d", t.Unix())
	case "unixms":
		return fmt.Sprintf("%d", t.UnixMilli())
	}
	return t.Format(layout)
}

// ParseTime parses s using the named layout or a raw Go layout.
func ParseTime(s, format string) (time.Time, error) {
	layout := resolveLayout(format)
	if layout == "unix" || layout == "unixms" {
		return time.Time{}, fmt.Errorf("cannot parse from %s yet", layout)
	}
	return time.Parse(layout, s)
}

// AddDuration returns t + duration.
func AddDuration(t time.Time, d time.Duration) time.Time {
	return t.Add(d)
}

func resolveLayout(name string) string {
	if l, ok := TimeNamedLayouts[strings.ToLower(name)]; ok {
		return l
	}
	if name == "" {
		return time.RFC3339
	}
	return name
}
