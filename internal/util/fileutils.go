package util

import (
	"os"
	"path/filepath"
	"strings"
)

var DEFAULT_EXT string = ".png"

// ResolveOutputPath makes the CLI user-friendly by handling common cases:
//
//	-out result.png           → use as-is
//	-out result               → result.png
//	-out ./results/           → ./results/inputname_mosaic.png
//	-out ./results            → ./results/inputname_mosaic.png
func ResolveOutputPath(inputPath, outputPath string) string {
	cleanPath := filepath.Clean(outputPath)

	if isDirectoryPath(cleanPath) {
		base := filepath.Base(inputPath)
		name := strings.TrimSuffix(base, filepath.Ext(base))
		return filepath.Join(cleanPath, name+"_mosaic"+DEFAULT_EXT)
	}

	// Not a directory - treat as filename
	if filepath.Ext(cleanPath) == "" {
		return cleanPath + DEFAULT_EXT
	}

	return cleanPath
}

// isDirectoryPath returns true if the path appears to be a directory
// (either it exists and is a dir, or it looks like one)
func isDirectoryPath(path string) bool {
	if path == "" || path == "." || path == "/" {
		return true
	}

	// Check if it has a trailing separator
	if strings.HasSuffix(path, string(filepath.Separator)) {
		return true
	}

	// If it has no extension, it's likely intended as a directory
	if filepath.Ext(path) == "" {
		return true
	}

	// Final check: if it exists on disk, trust the filesystem
	if info, err := os.Stat(path); err == nil {
		return info.IsDir()
	}

	return false
}
