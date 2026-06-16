package services

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

// ArchiveEntry describes one file or directory inside an archive.
type ArchiveEntry struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Mode string `json:"mode"`
}

// PackArchive writes an archive containing the given files/directories.
func PackArchive(output string, inputs []string, format string) error {
	if output == "" {
		return fmt.Errorf("output archive required")
	}
	if len(inputs) == 0 {
		return fmt.Errorf("at least one input required")
	}
	format = DetectArchiveFormat(output, format)
	if format == "" {
		return fmt.Errorf("archive format required")
	}
	switch format {
	case "zip":
		return packZip(output, inputs)
	case "tar", "tar.gz", "tar.xz", "tar.zst":
		return packTar(output, inputs, format)
	default:
		return fmt.Errorf("unsupported archive format %q", format)
	}
}

// UnpackArchive extracts an archive to dest.
func UnpackArchive(input string, dest string, format string, stripComponents int) error {
	if dest == "" {
		dest = "."
	}
	if stripComponents < 0 {
		return fmt.Errorf("--strip-components cannot be negative")
	}
	format = DetectArchiveFormat(input, format)
	switch format {
	case "zip":
		return unpackZip(input, dest, stripComponents)
	case "tar", "tar.gz", "tar.xz", "tar.zst":
		return unpackTar(input, dest, format, stripComponents)
	default:
		return fmt.Errorf("unsupported archive format %q", format)
	}
}

// ListArchive returns entries in an archive.
func ListArchive(input string, format string) ([]ArchiveEntry, error) {
	format = DetectArchiveFormat(input, format)
	switch format {
	case "zip":
		return listZip(input)
	case "tar", "tar.gz", "tar.xz", "tar.zst":
		return listTar(input, format)
	default:
		return nil, fmt.Errorf("unsupported archive format %q", format)
	}
}

// DetectArchiveFormat returns a normalized archive format from a flag or filename.
func DetectArchiveFormat(filename string, override string) string {
	if override != "" {
		return normalizeArchiveFormat(override)
	}
	name := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(name, ".tar.gz"), strings.HasSuffix(name, ".tgz"):
		return "tar.gz"
	case strings.HasSuffix(name, ".tar.xz"), strings.HasSuffix(name, ".txz"):
		return "tar.xz"
	case strings.HasSuffix(name, ".tar.zst"), strings.HasSuffix(name, ".tzst"):
		return "tar.zst"
	case strings.HasSuffix(name, ".tar"):
		return "tar"
	case strings.HasSuffix(name, ".zip"):
		return "zip"
	default:
		return ""
	}
}

func normalizeArchiveFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "tgz", "gz", "gzip":
		return "tar.gz"
	case "txz", "xz":
		return "tar.xz"
	case "tzst", "zst", "zstd":
		return "tar.zst"
	default:
		return strings.ToLower(strings.TrimSpace(format))
	}
}

func packTar(output string, inputs []string, format string) error {
	out, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("creating %s: %w", output, err)
	}
	defer out.Close()

	w, closeFn, err := tarWriter(out, format)
	if err != nil {
		return err
	}

	err = walkArchiveInputs(inputs, func(path, name string, info os.FileInfo) error {
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(name)
		if info.IsDir() && !strings.HasSuffix(hdr.Name, "/") {
			hdr.Name += "/"
		}
		if err := w.WriteHeader(hdr); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(w, f)
		return err
	})
	closeErr := closeFn()
	if err != nil {
		return err
	}
	return closeErr
}

func tarWriter(out io.Writer, format string) (*tar.Writer, func() error, error) {
	switch format {
	case "tar":
		tw := tar.NewWriter(out)
		return tw, tw.Close, nil
	case "tar.gz":
		gw := gzip.NewWriter(out)
		tw := tar.NewWriter(gw)
		return tw, func() error {
			if err := tw.Close(); err != nil {
				return err
			}
			return gw.Close()
		}, nil
	case "tar.xz":
		xw, err := xz.NewWriter(out)
		if err != nil {
			return nil, nil, err
		}
		tw := tar.NewWriter(xw)
		return tw, func() error {
			if err := tw.Close(); err != nil {
				return err
			}
			return xw.Close()
		}, nil
	case "tar.zst":
		zw, err := zstd.NewWriter(out)
		if err != nil {
			return nil, nil, err
		}
		tw := tar.NewWriter(zw)
		return tw, func() error {
			if err := tw.Close(); err != nil {
				return err
			}
			zw.Close()
			return nil
		}, nil
	default:
		return nil, nil, fmt.Errorf("unsupported tar format %q", format)
	}
}

func tarReader(in io.Reader, format string) (*tar.Reader, func(), error) {
	switch format {
	case "tar":
		return tar.NewReader(in), func() {}, nil
	case "tar.gz":
		gr, err := gzip.NewReader(in)
		if err != nil {
			return nil, nil, err
		}
		return tar.NewReader(gr), func() { gr.Close() }, nil
	case "tar.xz":
		xr, err := xz.NewReader(in)
		if err != nil {
			return nil, nil, err
		}
		return tar.NewReader(xr), func() {}, nil
	case "tar.zst":
		zr, err := zstd.NewReader(in)
		if err != nil {
			return nil, nil, err
		}
		return tar.NewReader(zr), func() { zr.Close() }, nil
	default:
		return nil, nil, fmt.Errorf("unsupported tar format %q", format)
	}
}

func packZip(output string, inputs []string) error {
	out, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("creating %s: %w", output, err)
	}
	defer out.Close()

	zw := zip.NewWriter(out)

	err = walkArchiveInputs(inputs, func(path, name string, info os.FileInfo) error {
		hdr, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(name)
		if info.IsDir() && !strings.HasSuffix(hdr.Name, "/") {
			hdr.Name += "/"
		}
		if !info.IsDir() {
			hdr.Method = zip.Deflate
		}
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(w, f)
		return err
	})
	closeErr := zw.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func walkArchiveInputs(inputs []string, fn func(path, name string, info os.FileInfo) error) error {
	for _, input := range inputs {
		info, err := os.Stat(input)
		if err != nil {
			return fmt.Errorf("stat %s: %w", input, err)
		}
		base := filepath.Dir(input)
		if !info.IsDir() {
			base = filepath.Dir(input)
		}
		if err := filepath.Walk(input, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			name, err := filepath.Rel(base, path)
			if err != nil {
				return err
			}
			name = cleanArchiveName(name)
			if name == "" {
				return nil
			}
			return fn(path, name, info)
		}); err != nil {
			return err
		}
	}
	return nil
}

func cleanArchiveName(name string) string {
	name = filepath.ToSlash(filepath.Clean(name))
	name = strings.TrimPrefix(name, "./")
	if name == "." || name == "/" || strings.HasPrefix(name, "../") || name == ".." {
		return ""
	}
	return strings.TrimPrefix(name, "/")
}

func unpackTar(input string, dest string, format string, stripComponents int) error {
	f, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("opening %s: %w", input, err)
	}
	defer f.Close()

	tr, closeFn, err := tarReader(f, format)
	if err != nil {
		return err
	}
	defer closeFn()

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name, ok, err := stripArchiveName(hdr.Name, stripComponents)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		target, err := safeArchiveTarget(dest, name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		}
	}
}

func unpackZip(input string, dest string, stripComponents int) error {
	zr, err := zip.OpenReader(input)
	if err != nil {
		return fmt.Errorf("opening %s: %w", input, err)
	}
	defer zr.Close()

	for _, file := range zr.File {
		name, ok, err := stripArchiveName(file.Name, stripComponents)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		target, err := safeArchiveTarget(dest, name)
		if err != nil {
			return err
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, file.Mode()); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		rc, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		closeErr := out.Close()
		rcErr := rc.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if rcErr != nil {
			return rcErr
		}
	}
	return nil
}

func listTar(input string, format string) ([]ArchiveEntry, error) {
	f, err := os.Open(input)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", input, err)
	}
	defer f.Close()
	tr, closeFn, err := tarReader(f, format)
	if err != nil {
		return nil, err
	}
	defer closeFn()

	var entries []ArchiveEntry
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		entries = append(entries, ArchiveEntry{Name: hdr.Name, Size: hdr.Size, Mode: os.FileMode(hdr.Mode).String()})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries, nil
}

func listZip(input string) ([]ArchiveEntry, error) {
	zr, err := zip.OpenReader(input)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", input, err)
	}
	defer zr.Close()
	entries := make([]ArchiveEntry, 0, len(zr.File))
	for _, file := range zr.File {
		entries = append(entries, ArchiveEntry{Name: file.Name, Size: int64(file.UncompressedSize64), Mode: file.Mode().String()})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries, nil
}

func stripArchiveName(name string, components int) (string, bool, error) {
	name, err := validateArchiveEntryName(name)
	if err != nil {
		return "", false, err
	}
	if name == "" {
		return "", false, nil
	}
	parts := strings.Split(name, "/")
	if len(parts) <= components {
		return "", false, nil
	}
	return strings.Join(parts[components:], "/"), true, nil
}

func validateArchiveEntryName(name string) (string, error) {
	name = filepath.ToSlash(strings.TrimSpace(name))
	name = strings.TrimPrefix(name, "./")
	if name == "" || name == "." {
		return "", nil
	}
	if strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("archive entry %q escapes destination", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			return "", fmt.Errorf("archive entry %q escapes destination", name)
		}
	}
	return filepath.ToSlash(filepath.Clean(name)), nil
}

func safeArchiveTarget(dest, name string) (string, error) {
	cleanDest, err := filepath.Abs(dest)
	if err != nil {
		return "", err
	}
	target, err := filepath.Abs(filepath.Join(cleanDest, filepath.FromSlash(name)))
	if err != nil {
		return "", err
	}
	if target != cleanDest && !strings.HasPrefix(target, cleanDest+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive entry %q escapes destination", name)
	}
	return target, nil
}
