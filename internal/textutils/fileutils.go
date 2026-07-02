package textutils

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// initialScanBufferSize is the default buffer size for bufio.Scanner.
	initialScanBufferSize = 64 * 1024 // 64 KiB

	// maxScanLineSize is the maximum allowed line length when using scanner-based reading.
	maxScanLineSize = 4 * 1024 * 1024 // 4 MiB
)

// ResolveOutputPath determines the final output file path according to these rules:
//   - If out is empty → {input basename}_{suffix}{ext} in the same directory as input
//   - If out is an existing directory → {out}/{input basename}_{suffix}{ext}
//   - Otherwise → use out as-is (after cleaning)
func ResolveOutputPath(in, out, suffix string) (string, error) {
	in = strings.TrimSpace(in)
	if in == "" {
		return "", fmt.Errorf("input path is required")
	}

	inClean := filepath.Clean(in)

	defaultPath, err := buildDefaultOutputPath(inClean, suffix)
	if err != nil {
		return "", fmt.Errorf("build default output for %q: %w", inClean, err)
	}

	out = strings.TrimSpace(out)
	if out == "" {
		return defaultPath, nil
	}

	outClean := filepath.Clean(out)

	// If the user provided an existing directory, place the file inside it
	if info, err := os.Stat(outClean); err == nil && info.IsDir() {
		return filepath.Join(outClean, filepath.Base(defaultPath)), nil
	}

	// Otherwise treat out as the exact target file path (new or existing)
	return outClean, nil
}

// buildDefaultOutputPath returns inputName_suffix.ext in the same directory as input.
func buildDefaultOutputPath(inputClean, suffix string) (string, error) {
	ext := filepath.Ext(inputClean)
	if ext == "" {
		return "", fmt.Errorf("input path %q has no file extension", inputClean)
	}

	base := filepath.Base(inputClean)
	name := strings.TrimSuffix(base, ext)
	if name == "" {
		return "", fmt.Errorf("input path %q has no valid filename", inputClean)
	}

	suffix = strings.TrimSpace(suffix)
	if suffix == "" {
		suffix = "processed"
	}

	return filepath.Join(
		filepath.Dir(inputClean),
		name+"_"+suffix+ext,
	), nil
}

// ReadTextFile reads the entire file into memory and validates it is not blank.
// This is the recommended approach for small-to-medium text files (configs, data, etc.).
func ReadTextFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file %q: %w", path, err)
	}

	if len(bytes.TrimSpace(data)) == 0 {
		return "", fmt.Errorf("text file %q is empty or contains only whitespace", path)
	}

	return string(data), nil
}

// ProcessLines streams the file line by line without loading everything into memory.
// Use this for large files or when you want to process data incrementally.
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
			return fmt.Errorf("processing line %q: %w", line, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan file %q: %w", path, err)
	}

	return nil
}
