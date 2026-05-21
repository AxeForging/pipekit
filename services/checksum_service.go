package services

import (
	"bufio"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileChecksum holds one checksum result.
type FileChecksum struct {
	Path      string `json:"path"`
	Checksum  string `json:"checksum"`
	Algorithm string `json:"algorithm"`
}

// ChecksumFiles hashes each file independently.
func ChecksumFiles(files []string, algorithm string) ([]FileChecksum, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("at least one file required")
	}
	algorithm = normalizeChecksumAlgorithm(algorithm)
	var sums []FileChecksum
	for _, path := range files {
		sum, err := ChecksumFile(path, algorithm)
		if err != nil {
			return nil, err
		}
		sums = append(sums, FileChecksum{Path: filepath.ToSlash(path), Checksum: sum, Algorithm: algorithm})
	}
	return sums, nil
}

// ChecksumFile hashes a single file with the requested algorithm.
func ChecksumFile(path, algorithm string) (string, error) {
	h, algorithm, err := newChecksumHash(algorithm)
	if err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hashing %s: %w", path, err)
	}
	_ = algorithm
	return hex.EncodeToString(h.Sum(nil)), nil
}

// FormatChecksums formats checksum output in common checksum-file form:
// "<hex>  <path>".
func FormatChecksums(sums []FileChecksum) string {
	var b strings.Builder
	for _, sum := range sums {
		fmt.Fprintf(&b, "%s  %s\n", sum.Checksum, sum.Path)
	}
	return b.String()
}

// VerifyChecksums verifies a checksum file in "<hex>  <path>" form.
func VerifyChecksums(manifestPath, algorithm string) error {
	f, err := os.Open(manifestPath)
	if err != nil {
		return fmt.Errorf("opening %s: %w", manifestPath, err)
	}
	defer f.Close()

	base := filepath.Dir(manifestPath)
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return fmt.Errorf("%s:%d: expected '<checksum> <path>'", manifestPath, lineNo)
		}
		want := fields[0]
		path := strings.Join(fields[1:], " ")
		if !filepath.IsAbs(path) {
			path = filepath.Join(base, path)
		}
		got, err := ChecksumFile(path, algorithm)
		if err != nil {
			return err
		}
		if !strings.EqualFold(got, want) {
			return fmt.Errorf("%s: checksum mismatch: got %s, want %s", path, got, want)
		}
	}
	return scanner.Err()
}

func newChecksumHash(algorithm string) (hash.Hash, string, error) {
	algorithm = normalizeChecksumAlgorithm(algorithm)
	switch algorithm {
	case "sha256":
		return sha256.New(), algorithm, nil
	case "sha1":
		return sha1.New(), algorithm, nil
	case "md5":
		return md5.New(), algorithm, nil
	default:
		return nil, "", fmt.Errorf("unknown checksum algorithm %q (valid: sha256, sha1, md5)", algorithm)
	}
}

func normalizeChecksumAlgorithm(algorithm string) string {
	if algorithm == "" {
		return "sha256"
	}
	return strings.ToLower(strings.TrimSpace(algorithm))
}
