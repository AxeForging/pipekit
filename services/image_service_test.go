package services

import "testing"

func TestParseImage(t *testing.T) {
	tests := []struct {
		raw      string
		registry string
		repo     string
		tag      string
		digest   string
	}{
		{
			raw:      "ghcr.io/org/repo:v1.2.3",
			registry: "ghcr.io",
			repo:     "org/repo",
			tag:      "v1.2.3",
		},
		{
			raw:      "ghcr.io/org/repo:v1.2.3@sha256:abc123",
			registry: "ghcr.io",
			repo:     "org/repo",
			tag:      "v1.2.3",
			digest:   "sha256:abc123",
		},
		{
			raw:      "redis:7",
			registry: "docker.io",
			repo:     "library/redis",
			tag:      "7",
		},
		{
			raw:      "redis",
			registry: "docker.io",
			repo:     "library/redis",
			tag:      "latest",
		},
		{
			raw:      "myorg/myapp:v1",
			registry: "docker.io",
			repo:     "myorg/myapp",
			tag:      "v1",
		},
		{
			raw:      "registry.local:5000/team/svc:dev",
			registry: "registry.local:5000",
			repo:     "team/svc",
			tag:      "dev",
		},
		{
			raw:      "localhost:5000/foo:bar",
			registry: "localhost:5000",
			repo:     "foo",
			tag:      "bar",
		},
		{
			raw:      "ghcr.io/org/repo@sha256:deadbeef",
			registry: "ghcr.io",
			repo:     "org/repo",
			digest:   "sha256:deadbeef",
		},
	}
	for _, tc := range tests {
		got, err := ParseImage(tc.raw)
		if err != nil {
			t.Errorf("ParseImage(%q) error: %v", tc.raw, err)
			continue
		}
		if got.Registry != tc.registry || got.Repository != tc.repo ||
			got.Tag != tc.tag || got.Digest != tc.digest {
			t.Errorf("ParseImage(%q) = %+v, want registry=%q repo=%q tag=%q digest=%q",
				tc.raw, got, tc.registry, tc.repo, tc.tag, tc.digest)
		}
	}
}

func TestImageRef_StringRoundTrip(t *testing.T) {
	originals := []string{
		"ghcr.io/org/repo:v1.2.3",
		"ghcr.io/org/repo:v1.2.3@sha256:abc123",
		"docker.io/library/redis:7",
	}
	for _, o := range originals {
		ref, _ := ParseImage(o)
		got := ref.String()
		if got != o {
			t.Errorf("round-trip lost: %q -> %q", o, got)
		}
	}
}
