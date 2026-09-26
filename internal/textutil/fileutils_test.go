package textutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveOutputPath(t *testing.T) {
	t.Run("defaults to PNG beside input", func(t *testing.T) {
		got, err := ResolveOutputPath("images/photo.jpg", "", "wordcloud", ".png")
		if err != nil {
			t.Fatalf("ResolveOutputPath() error = %v", err)
		}
		want := filepath.Join("images", "photo_wordcloud.png")
		if got != want {
			t.Fatalf("ResolveOutputPath() = %q, want %q", got, want)
		}
	})

	t.Run("uses an existing directory", func(t *testing.T) {
		dir := t.TempDir()
		got, err := ResolveOutputPath("photo.jpg", dir, "wordcloud", ".png")
		if err != nil {
			t.Fatalf("ResolveOutputPath() error = %v", err)
		}
		want := filepath.Join(dir, "photo_wordcloud.png")
		if got != want {
			t.Fatalf("ResolveOutputPath() = %q, want %q", got, want)
		}
	})

	t.Run("treats a trailing separator as a directory", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "new-output")
		got, err := ResolveOutputPath("photo.jpg", dir+string(os.PathSeparator), "wordcloud", ".png")
		if err != nil {
			t.Fatalf("ResolveOutputPath() error = %v", err)
		}
		want := filepath.Join(dir, "photo_wordcloud.png")
		if got != want {
			t.Fatalf("ResolveOutputPath() = %q, want %q", got, want)
		}
	})

	t.Run("appends missing extension", func(t *testing.T) {
		got, err := ResolveOutputPath("photo.jpg", "result", "wordcloud", ".png")
		if err != nil {
			t.Fatalf("ResolveOutputPath() error = %v", err)
		}
		if got != "result.png" {
			t.Fatalf("ResolveOutputPath() = %q, want result.png", got)
		}
	})

	t.Run("rejects unsupported output extension", func(t *testing.T) {
		if _, err := ResolveOutputPath("photo.jpg", "result.jpg", "wordcloud", ".png"); err == nil {
			t.Fatal("ResolveOutputPath() error = nil, want extension error")
		}
	})
}
