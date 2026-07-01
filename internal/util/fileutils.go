package util

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type OutputPathOptions struct {
	// Added when output path has no extension.
	DefaultExt string

	// Added when output path is empty or resolves to a directory.
	DefaultSuffix string
}

func ResolveOutputPath(inputPath, outputPath string, opts OutputPathOptions) (string, error) {
	opts.DefaultExt = normalizeExt(opts.DefaultExt)

	inputPath = strings.TrimSpace(inputPath)
	outputPath = strings.TrimSpace(outputPath)

	if inputPath == "" {
		return "", errors.New("input path is required")
	}

	inputName, err := stem(inputPath)
	if err != nil {
		return "", err
	}

	defaultName := func() (string, error) {
		if opts.DefaultExt == "" {
			return "", errors.New("default extension is required")
		}

		return inputName + opts.DefaultSuffix + opts.DefaultExt, nil
	}

	var out string

	switch {
	case outputPath == "":
		out, err = defaultName()

	case looksLikeDir(outputPath):
		name, nameErr := defaultName()
		if nameErr != nil {
			return "", nameErr
		}
		out = filepath.Join(filepath.Clean(outputPath), name)

	default:
		out = filepath.Clean(outputPath)

		isDir, dirErr := existingDir(out)
		if dirErr != nil {
			return "", dirErr
		}

		if isDir {
			name, nameErr := defaultName()
			if nameErr != nil {
				return "", nameErr
			}
			out = filepath.Join(out, name)
		} else if filepath.Ext(out) == "" {
			if opts.DefaultExt == "" {
				return "", fmt.Errorf("output path %q has no extension and no default extension was provided", outputPath)
			}
			out += opts.DefaultExt
		}
	}

	if err != nil {
		return "", err
	}

	if err := validateOutputPath(out); err != nil {
		return "", err
	}

	return out, nil
}

func validateOutputPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("output path is required")
	}

	path = filepath.Clean(path)

	if filepath.Ext(path) == "" {
		return fmt.Errorf("output path %q must include a file extension", path)
	}

	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("output path %q is a directory, expected file path", path)
		}
		return nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat output path %q: %w", path, err)
	}

	parent := filepath.Dir(path)

	info, err = os.Stat(parent)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("output parent path %q is not a directory", parent)
		}
		return nil
	}

	// Preserve your original behavior: missing parent dirs are allowed.
	// If you want to require existing parent dirs, return an error here instead.
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return fmt.Errorf("stat output directory %q: %w", parent, err)
}

func stem(path string) (string, error) {
	clean := filepath.Clean(strings.TrimSpace(path))
	base := filepath.Base(clean)

	if base == "." || base == string(filepath.Separator) || base == "" {
		return "", fmt.Errorf("invalid input path %q", path)
	}

	name := strings.TrimSuffix(base, filepath.Ext(base))
	if name == "" {
		return "", fmt.Errorf("invalid input filename %q", base)
	}

	return name, nil
}

func normalizeExt(ext string) string {
	ext = strings.TrimSpace(ext)

	if ext != "" && !strings.HasPrefix(ext, ".") {
		return "." + ext
	}

	return ext
}

func looksLikeDir(path string) bool {
	return strings.HasSuffix(path, "/") ||
		strings.HasSuffix(path, string(os.PathSeparator))
}

func existingDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return info.IsDir(), nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, fmt.Errorf("stat path %q: %w", path, err)
}
