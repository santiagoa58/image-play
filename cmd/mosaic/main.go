package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
	"github.com/santiagoa58/image-play/internal/util"
)

func main() {
	// === CLI Flags (grouped for clarity) ===
	var (
		inputPath  = flag.String("in", "", "Path to input image (PNG, JPEG, WebP, etc.) [required]")
		outputPath = flag.String("out", "output.png", "Output path. Can be file or directory (auto filename if dir)")
		width      = flag.Int("width", 0, "Target width in pixels. Common: 1080, 1920, 3840. 0 = original size")
		verbose    = flag.Bool("v", false, "Enable verbose/debug logging")
	)

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, "mosaic — Image effects toolkit\n\n")
		fmt.Fprint(os.Stderr, "Usage:\n  mosaic -in photo.jpg -out result.png -width 1920\n\n")
		fmt.Fprint(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprint(os.Stderr, "\nCommon widths: 1080 (HD), 1920 (Full HD), 3840 (4K)\n")
	}

	flag.Parse()

	// === Logging Setup ===
	level := slog.LevelInfo
	if *verbose {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})))

	slog.Info("mosaic starting", "version", "dev")

	// === Validate Input Path ===
	if *inputPath == "" {
		slog.Error("missing required flag", "flag", "-in")
		flag.Usage()
		os.Exit(1)
	}

	// === Load Image ===
	slog.Debug("loading image", "path", *inputPath)
	img, err := imaging.Open(*inputPath)
	if err != nil {
		slog.Error("failed to open image", "error", err, "path", *inputPath)
		os.Exit(1)
	}

	// === Resize (if requested) ===
	if *width > 0 {
		slog.Debug("resizing image", "target_width", *width)
		img = imaging.Resize(img, *width, 0, imaging.Lanczos)
		slog.Debug("resize completed", "new_size", fmt.Sprintf("%dx%d", img.Bounds().Dx(), img.Bounds().Dy()))
	}

	// === Smart Output Path Handling ===
	finalOutputPath := util.ResolveOutputPath(*inputPath, *outputPath)
	slog.Debug("resolved output path", "path", finalOutputPath)

	// Create directory if needed
	if dir := filepath.Dir(finalOutputPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			slog.Error("failed to create output directory", "dir", dir, "error", err)
			os.Exit(1)
		}
	}

	// === Save ===
	slog.Debug("saving output", "path", finalOutputPath)
	if err := imaging.Save(img, finalOutputPath); err != nil {
		slog.Error("failed to save image", "error", err, "path", finalOutputPath)
		os.Exit(1)
	}

	slog.Info("success", "output", finalOutputPath, "size", fmt.Sprintf("%dx%d", img.Bounds().Dx(), img.Bounds().Dy()))
	fmt.Printf("✅ Done! Saved to %s\n", finalOutputPath)
}
