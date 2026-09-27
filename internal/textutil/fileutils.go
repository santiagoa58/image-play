package textutil

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	initialScanBufferSize = 64 * 1024
	maxScanLineSize       = 4 * 1024 * 1024
)

// ResolveOutputPath resolves an output file using ext as the required format.
//
// If out is empty, the result is inputName_suffix.ext beside the input file.
// Existing directories and paths ending in a separator are treated as output
// directories. Explicit filenames without an extension receive ext.
func ResolveOutputPath(in, out, suffix, ext string) (string, error) {
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

	defaultPath, err := buildDefaultOutputPath(filepath.Clean(in), suffix, ext)
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

func buildDefaultOutputPath(inputClean, suffix, ext string) (string, error) {
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

// ProcessLines streams path line by line and calls fn for each line.
func ProcessLines(path string, fn func(line string) error) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file %q: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, initialScanBufferSize), maxScanLineSize)

	for scanner.Scan() {
		line := scanner.Text()
		if err := fn(line); err != nil {
			return fmt.Errorf("process line %q: %w", line, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan file %q: %w", path, err)
	}

	return nil
}
