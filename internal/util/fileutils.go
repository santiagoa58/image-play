package util

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type OutputPathOptions struct {
	// DefaultExt is added when the output path has no extension.
	// Example: "result" + ".png" -> "result.png".
	DefaultExt string

	// DefaultSuffix is added when outputPath resolves to a directory or is empty.
	// Example: input "photo.jpg" + "_mosaic" + ".png" -> "photo_mosaic.png".
	DefaultSuffix string

	// AllowCreateParent allows the resolved output path's parent directory to be missing.
	AllowCreateParent bool

	// AllowOverwrite allows the resolved output path to already exist as a file.
	AllowOverwrite bool
}

// ResolveOutputPath turns an input path and user-provided output path into a concrete file path.
//
// Examples with DefaultExt ".png" and DefaultSuffix "_mosaic":
//
//	input: "photo.jpg", output: ""           -> "photo_mosaic.png"
//	input: "photo.jpg", output: "result"     -> "result.png"
//	input: "photo.jpg", output: "result.png" -> "result.png"
//	input: "photo.jpg", output: "out/"       -> "out/photo_mosaic.png"
//	input: "photo.jpg", output: "out"        -> "out/photo_mosaic.png" if out exists as a directory,
//	                                            otherwise "out.png"
func ResolveOutputPath(inputPath, outputPath string, opts OutputPathOptions) (string, error) {
	opts = normalizeOutputPathOptions(opts)

	inputPath = strings.TrimSpace(inputPath)
	outputPath = strings.TrimSpace(outputPath)

	if inputPath == "" {
		return "", errors.New("input path is required")
	}

	cleanInputPath := filepath.Clean(inputPath)

	inputName, err := baseNameWithoutExt(cleanInputPath)
	if err != nil {
		return "", err
	}

	resolved, err := resolveOutputPath(inputName, outputPath, opts)
	if err != nil {
		return "", err
	}

	if err := ValidateOutputPath(resolved, opts); err != nil {
		return "", err
	}

	return resolved, nil
}

func resolveOutputPath(inputName, outputPath string, opts OutputPathOptions) (string, error) {
	defaultName, err := defaultOutputName(inputName, opts)
	if err != nil {
		return "", err
	}

	if outputPath == "" {
		return defaultName, nil
	}

	// Check this before filepath.Clean because Clean removes trailing separators.
	if looksLikeExplicitDirectory(outputPath) {
		return filepath.Join(filepath.Clean(outputPath), defaultName), nil
	}

	cleanOutputPath := filepath.Clean(outputPath)

	isDir, err := pathIsExistingDir(cleanOutputPath)
	if err != nil {
		return "", err
	}

	if isDir {
		return filepath.Join(cleanOutputPath, defaultName), nil
	}

	if filepath.Ext(cleanOutputPath) == "" {
		if opts.DefaultExt == "" {
			return "", fmt.Errorf("output path %q has no extension and no default extension was provided", outputPath)
		}
		return cleanOutputPath + opts.DefaultExt, nil
	}

	return cleanOutputPath, nil
}

// ValidateOutputPath checks whether path is a usable output file path.
// It validates the path shape, parent directory, and overwrite rules.
func ValidateOutputPath(path string, opts OutputPathOptions) error {
	opts = normalizeOutputPathOptions(opts)

	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("output path is required")
	}

	cleanPath := filepath.Clean(path)

	if err := validateOutputPathShape(cleanPath); err != nil {
		return err
	}

	if err := validateParentDir(filepath.Dir(cleanPath), opts); err != nil {
		return err
	}

	if err := validateOutputFile(cleanPath, opts); err != nil {
		return err
	}

	return nil
}

func validateOutputPathShape(path string) error {
	if filepath.Ext(path) == "" {
		return fmt.Errorf("output path %q must include a file extension", path)
	}

	return nil
}

func validateParentDir(parent string, opts OutputPathOptions) error {
	info, err := os.Stat(parent)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("output parent path %q is not a directory", parent)
		}
		return nil
	}

	if errors.Is(err, os.ErrNotExist) {
		if opts.AllowCreateParent {
			return nil
		}
		return fmt.Errorf("output directory %q does not exist", parent)
	}

	return fmt.Errorf("stat output directory %q: %w", parent, err)
}

func validateOutputFile(path string, opts OutputPathOptions) error {
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("output path %q is a directory, expected file path", path)
		}
		if !opts.AllowOverwrite {
			return fmt.Errorf("output file %q already exists", path)
		}
		return nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return fmt.Errorf("stat output path %q: %w", path, err)
}

func defaultOutputName(inputName string, opts OutputPathOptions) (string, error) {
	if opts.DefaultExt == "" {
		return "", errors.New("default extension is required when output path is empty or resolves to a directory")
	}

	return inputName + opts.DefaultSuffix + opts.DefaultExt, nil
}

func normalizeOutputPathOptions(opts OutputPathOptions) OutputPathOptions {
	opts.DefaultExt = strings.TrimSpace(opts.DefaultExt)

	if opts.DefaultExt != "" && !strings.HasPrefix(opts.DefaultExt, ".") {
		opts.DefaultExt = "." + opts.DefaultExt
	}

	return opts
}

func baseNameWithoutExt(cleanPath string) (string, error) {
	base := filepath.Base(cleanPath)

	if base == "." || base == string(filepath.Separator) || base == "" {
		return "", fmt.Errorf("invalid input path %q", cleanPath)
	}

	name := strings.TrimSuffix(base, filepath.Ext(base))
	if name == "" {
		return "", fmt.Errorf("invalid input filename %q", base)
	}

	return name, nil
}

func looksLikeExplicitDirectory(path string) bool {
	return strings.HasSuffix(path, string(os.PathSeparator)) ||
		strings.HasSuffix(path, "/")
}

func pathIsExistingDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return info.IsDir(), nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, fmt.Errorf("stat path %q: %w", path, err)
}
