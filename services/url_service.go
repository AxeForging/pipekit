package services

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/AxeForging/pipekit/domain"
)

// ParseURL splits a URL into structured components and returns them as
// KeyValue pairs. Keys are uppercased and prefixed with the optional prefix
// (e.g. "DB_") so they can flow directly into env-var writers.
func ParseURL(raw, prefix string) ([]domain.KeyValue, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}
	if u.Scheme == "" {
		return nil, fmt.Errorf("URL missing scheme: %q", raw)
	}

	var pwd string
	user := ""
	if u.User != nil {
		user = u.User.Username()
		pwd, _ = u.User.Password()
	}

	pairs := []domain.KeyValue{
		{Key: "SCHEME", Value: u.Scheme},
		{Key: "HOST", Value: u.Hostname()},
		{Key: "PORT", Value: u.Port()},
		{Key: "USER", Value: user},
		{Key: "PASSWORD", Value: pwd},
		{Key: "PATH", Value: u.Path},
		{Key: "QUERY", Value: u.RawQuery},
		{Key: "FRAGMENT", Value: u.Fragment},
	}

	// Drop empties so we don't pollute the environment with blank vars.
	out := make([]domain.KeyValue, 0, len(pairs))
	for _, kv := range pairs {
		if kv.Value == "" {
			continue
		}
		if prefix != "" {
			kv.Key = prefix + kv.Key
		}
		out = append(out, kv)
	}
	return out, nil
}
