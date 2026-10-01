package output

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePath(t *testing.T) {
	t.Run("defaults to PNG beside input", func(t *testing.T) {
		got, err := ResolvePath("images/photo.jpg", "", "wordcloud", ".png")
		if err != nil {
			t.Fatalf("ResolvePath() error = %v", err)
		}
		want := filepath.Join("images", "photo_wordcloud.png")
		if got != want {
			t.Fatalf("ResolvePath() = %q, want %q", got, want)
		}
	})

	t.Run("uses an existing directory", func(t *testing.T) {
		dir := t.TempDir()
		got, err := ResolvePath("photo.jpg", dir, "wordcloud", ".png")
		if err != nil {
			t.Fatalf("ResolvePath() error = %v", err)
		}
		want := filepath.Join(dir, "photo_wordcloud.png")
		if got != want {
			t.Fatalf("ResolvePath() = %q, want %q", got, want)
		}
	})

	t.Run("treats a trailing separator as a directory", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "new-output")
		got, err := ResolvePath("photo.jpg", dir+string(os.PathSeparator), "wordcloud", ".png")
		if err != nil {
			t.Fatalf("ResolvePath() error = %v", err)
		}
		want := filepath.Join(dir, "photo_wordcloud.png")
		if got != want {
			t.Fatalf("ResolvePath() = %q, want %q", got, want)
		}
	})

	t.Run("appends missing extension", func(t *testing.T) {
		got, err := ResolvePath("photo.jpg", "result", "wordcloud", ".png")
		if err != nil {
			t.Fatalf("ResolvePath() error = %v", err)
		}
		if got != "result.png" {
			t.Fatalf("ResolvePath() = %q, want result.png", got)
		}
	})

	t.Run("rejects unsupported output extension", func(t *testing.T) {
		if _, err := ResolvePath("photo.jpg", "result.jpg", "wordcloud", ".png"); err == nil {
			t.Fatal("ResolvePath() error = nil, want extension error")
		}
	})
}

func TestPreparePNGCreatesOutputDirectory(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "nested")
	got, err := PreparePNG("photo.jpg", outputDir+string(os.PathSeparator), "textmosaic")
	if err != nil {
		t.Fatalf("PreparePNG() error = %v", err)
	}
	want := filepath.Join(outputDir, "photo_textmosaic.png")
	if got != want {
		t.Fatalf("PreparePNG() = %q, want %q", got, want)
	}
	if info, err := os.Stat(outputDir); err != nil || !info.IsDir() {
		t.Fatalf("output directory not created: info = %v, error = %v", info, err)
	}
}
