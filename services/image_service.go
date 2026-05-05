package services

import (
	"fmt"
	"strings"
)

// ImageRef holds the components of a parsed container image reference.
type ImageRef struct {
	Registry   string // e.g. ghcr.io, docker.io
	Repository string // e.g. org/repo or library/redis
	Tag        string // e.g. v1.2.3 (empty if digest-only)
	Digest     string // e.g. sha256:abc... (empty if untagged)
}

// String reassembles the ref. Inverse of ParseImage.
func (r ImageRef) String() string {
	out := ""
	if r.Registry != "" {
		out = r.Registry + "/"
	}
	out += r.Repository
	if r.Tag != "" {
		out += ":" + r.Tag
	}
	if r.Digest != "" {
		out += "@" + r.Digest
	}
	return out
}

// ParseImage parses a container image reference. The grammar is roughly:
//
//	[registry/]repository[:tag][@digest]
//
// A registry is detected when the first slash-separated segment contains a
// "." or a ":" (e.g. ghcr.io, registry.local:5000), or is "localhost".
// Refs without an explicit registry default to docker.io. Single-segment
// repositories (e.g. "redis") are normalized to "library/redis" — matching
// Docker Hub's convention.
func ParseImage(raw string) (ImageRef, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ImageRef{}, fmt.Errorf("empty image reference")
	}

	r := ImageRef{}

	// Pull off the digest (always after @).
	if i := strings.LastIndex(s, "@"); i >= 0 {
		r.Digest = s[i+1:]
		s = s[:i]
	}

	// Pull off the tag — but only if the colon comes AFTER the last slash
	// (otherwise we'd parse a registry port as a tag).
	if i := strings.LastIndex(s, ":"); i >= 0 {
		if j := strings.LastIndex(s, "/"); j < i {
			r.Tag = s[i+1:]
			s = s[:i]
		}
	}

	// Decide whether the leading segment is a registry.
	parts := strings.SplitN(s, "/", 2)
	if len(parts) == 2 && looksLikeRegistry(parts[0]) {
		r.Registry = parts[0]
		r.Repository = parts[1]
	} else {
		r.Registry = "docker.io"
		r.Repository = s
		if !strings.Contains(r.Repository, "/") {
			r.Repository = "library/" + r.Repository
		}
	}

	if r.Tag == "" && r.Digest == "" {
		r.Tag = "latest"
	}
	return r, nil
}

func looksLikeRegistry(s string) bool {
	return s == "localhost" || strings.Contains(s, ".") || strings.Contains(s, ":")
}
