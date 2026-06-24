package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveOutputPath(t *testing.T) {
	const (
		testFileName = "couple_tour"
		testExt      = ".jpg"
	)

	opts := OutputPathOptions{
		DefaultExt:        ".png",
		DefaultSuffix:     "_mosaic",
		AllowCreateParent: true,
		AllowOverwrite:    true,
	}

	inputPath := filepath.Join("..", "testdata", "images", testFileName+testExt)
	tempDir := t.TempDir()
	existingDir := filepath.Join(tempDir, "results")

	if err := os.MkdirAll(existingDir, 0755); err != nil {
		t.Fatalf("create test directory: %v", err)
	}

	testCases := []struct {
		name     string
		input    string
		output   string
		expected string
	}{
		{
			name:     "empty output uses input name with suffix and default extension",
			input:    inputPath,
			output:   "",
			expected: testFileName + "_mosaic.png",
		},
		{
			name:     "explicit filename with extension",
			input:    inputPath,
			output:   filepath.Join(tempDir, "result.png"),
			expected: filepath.Join(tempDir, "result.png"),
		},
		{
			name:     "filename without extension",
			input:    inputPath,
			output:   filepath.Join(tempDir, "result"),
			expected: filepath.Join(tempDir, "result.png"),
		},
		{
			name:     "directory with trailing slash",
			input:    inputPath,
			output:   existingDir + string(filepath.Separator),
			expected: filepath.Join(existingDir, testFileName+"_mosaic.png"),
		},
		{
			name:     "existing directory without trailing slash",
			input:    inputPath,
			output:   existingDir,
			expected: filepath.Join(existingDir, testFileName+"_mosaic.png"),
		},
		{
			name:     "non-existing path without extension is treated as filename",
			input:    inputPath,
			output:   filepath.Join(tempDir, "results_missing"),
			expected: filepath.Join(tempDir, "results_missing.png"),
		},
		{
			name:     "deep nested directory with trailing slash",
			input:    inputPath,
			output:   filepath.Join(tempDir, "output", "2026", "summer") + string(filepath.Separator),
			expected: filepath.Join(tempDir, "output", "2026", "summer", testFileName+"_mosaic.png"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ResolveOutputPath(tc.input, tc.output, opts)
			if err != nil {
				t.Fatalf("ResolveOutputPath(%q, %q) returned error: %v", tc.input, tc.output, err)
			}

			if result != tc.expected {
				t.Errorf("ResolveOutputPath(%q, %q) = %q; want %q", tc.input, tc.output, result, tc.expected)
			}
		})
	}
}

func TestResolveOutputPathErrors(t *testing.T) {
	t.Run("missing input path", func(t *testing.T) {
		_, err := ResolveOutputPath("", "result.png", OutputPathOptions{
			DefaultExt:        ".png",
			DefaultSuffix:     "_mosaic",
			AllowCreateParent: true,
			AllowOverwrite:    true,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("missing parent directory when create parent is disabled", func(t *testing.T) {
		_, err := ResolveOutputPath("photo.jpg", filepath.Join(t.TempDir(), "missing", "result.png"), OutputPathOptions{
			DefaultExt:        ".png",
			DefaultSuffix:     "_mosaic",
			AllowCreateParent: false,
			AllowOverwrite:    true,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("empty output without default extension", func(t *testing.T) {
		_, err := ResolveOutputPath("photo.jpg", "", OutputPathOptions{
			DefaultExt:        "",
			DefaultSuffix:     "_mosaic",
			AllowCreateParent: true,
			AllowOverwrite:    true,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestValidateOutputPathOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "result.png")

	if err := os.WriteFile(outputPath, []byte("existing"), 0644); err != nil {
		t.Fatalf("create existing output file: %v", err)
	}

	err := ValidateOutputPath(outputPath, OutputPathOptions{
		AllowOverwrite: false,
	})
	if err == nil {
		t.Fatal("expected overwrite error, got nil")
	}

	err = ValidateOutputPath(outputPath, OutputPathOptions{
		AllowOverwrite: true,
	})
	if err != nil {
		t.Fatalf("expected overwrite to be allowed, got error: %v", err)
	}
}
