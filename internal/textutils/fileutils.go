package textutils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ResolveOutputPath(inputPath, outputPath, suffix string) (string, error) {
	inputPath = strings.TrimSpace(inputPath)
	outputPath = strings.TrimSpace(outputPath)

	if inputPath == "" {
		return "", fmt.Errorf("input path is required")
	}

	defaultPath, err := defaultOutputPath(inputPath, suffix)
	if err != nil {
		return "", err
	}

	if outputPath == "" {
		return defaultPath, nil
	}

	out := filepath.Clean(outputPath)

	if info, err := os.Stat(out); err == nil && info.IsDir() {
		return filepath.Join(out, filepath.Base(defaultPath)), nil
	}

	return out, nil
}

func defaultOutputPath(inputPath, suffix string) (string, error) {
	clean := filepath.Clean(inputPath)
	suffix = "_" + suffix
	ext := filepath.Ext(clean)
	if ext == "" {
		return "", fmt.Errorf("input path %q has no file extension", inputPath)
	}

	base := filepath.Base(clean)
	name := strings.TrimSuffix(base, ext)

	if name == "" {
		return "", fmt.Errorf("input path %q has no valid file name", inputPath)
	}

	return filepath.Join(filepath.Dir(clean), name+suffix+ext), nil
}
