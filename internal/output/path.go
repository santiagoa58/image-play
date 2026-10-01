// Package output resolves effect output paths and prepares their directories.
package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PreparePNG resolves the output PNG path and creates its parent directory.
func PreparePNG(in, out, suffix string) (string, error) {
	path, err := ResolvePath(in, out, suffix, ".png")
	if err != nil {
		return "", fmt.Errorf("resolve output path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}
	return path, nil
}

// ResolvePath resolves an output file using ext as the required format.
//
// If out is empty, the result is inputName_suffix.ext beside the input file.
// Existing directories and paths ending in a separator are treated as output
// directories. Explicit filenames without an extension receive ext.
func ResolvePath(in, out, suffix, ext string) (string, error) {
	in = strings.TrimSpace(in)
	if in == "" {
		return "", fmt.Errorf("input path is required")
	}

	ext = strings.TrimSpace(ext)
	if ext == "" {
		return "", fmt.Errorf("output extension is required")
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	defaultPath, err := buildDefaultPath(filepath.Clean(in), suffix, ext)
	if err != nil {
		return "", err
	}

	out = strings.TrimSpace(out)
	if out == "" {
		return defaultPath, nil
	}

	asDirectory := strings.HasSuffix(out, "/") || strings.HasSuffix(out, "\\")
	outClean := filepath.Clean(out)

	info, statErr := os.Stat(outClean)
	switch {
	case statErr == nil && info.IsDir():
		asDirectory = true
	case statErr != nil && !os.IsNotExist(statErr):
		return "", fmt.Errorf("inspect output path %q: %w", outClean, statErr)
	}

	if asDirectory {
		return filepath.Join(outClean, filepath.Base(defaultPath)), nil
	}

	outputExt := filepath.Ext(outClean)
	if outputExt == "" {
		return outClean + ext, nil
	}
	if !strings.EqualFold(outputExt, ext) {
		return "", fmt.Errorf("output file must use %s extension", ext)
	}

	return outClean, nil
}

func buildDefaultPath(inputClean, suffix, ext string) (string, error) {
	base := filepath.Base(inputClean)
	inputExt := filepath.Ext(base)
	name := strings.TrimSuffix(base, inputExt)
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("input path %q has no valid filename", inputClean)
	}

	suffix = strings.TrimSpace(suffix)
	if suffix == "" {
		suffix = "processed"
	}

	return filepath.Join(filepath.Dir(inputClean), name+"_"+suffix+ext), nil
}
