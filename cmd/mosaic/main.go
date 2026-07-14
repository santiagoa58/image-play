package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/santiagoa58/image-play/internal/effects/textmosaic"
	"github.com/santiagoa58/image-play/internal/effects/wordcloud"
	"github.com/santiagoa58/image-play/internal/textutil"
)

const (
	appName             = "mosaic"
	appVersion          = "dev"
	defaultOutputExt    = ".png"
	defaultOutputSuffix = "_mosaic"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		inputPath  = flag.String("in", "", "Path to input image (PNG, JPEG, WebP, etc.) [required]")
		outputPath = flag.String("out", "", "Output path. Can be file or directory. Empty = input_wordcloud.png")
		textFile   = flag.String("text", "", "Path to text file [required]")
		fontPath   = flag.String("font", "", "Path to a TTF or OTF font file [required]")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s — image effects toolkit\n\n", appName)
		fmt.Fprint(os.Stderr, "Usage:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	logger := slog.Default()
	logger.Info("mosaic starting", "version", appVersion)

	if err := validateRequiredFlags(*inputPath, *textFile, *fontPath); err != nil {
		flag.Usage()
		return err
	}

	// err := mosaicRun(*inputPath, *outputPath, *textFile, *fontPath)
	// if err != nil {
	// 	return fmt.Errorf("mosaic run: %w", err)
	// }
	wcConfig := wordcloud.NewConfig(
		wordcloud.WithInputPath(*inputPath),
		wordcloud.WithOutputPath(*outputPath),
		wordcloud.WithTextPath(*textFile),
		wordcloud.WithFontPath(*fontPath),
	)

	if err := wordcloud.Generate(wcConfig); err != nil {
		return fmt.Errorf("wordcloud run: %w", err)
	}
	return nil
}

func mosaicRun(in, out, textFile, fontPath string) error {
	logger := slog.Default()
	mosaicText, err := resolveText(textFile)
	if err != nil {
		return err
	}

	logger.Debug("loading image", "path", in)

	img, err := imaging.Open(in)
	if err != nil {
		return fmt.Errorf("open input image %q: %w", in, err)
	}

	finalOutputPath, err := textutil.ResolveOutputPath(in, out, "textmosaic")
	if err != nil {
		return err
	}
	logger.Debug("resolved output path", "path", finalOutputPath)

	if err := os.MkdirAll(filepath.Dir(finalOutputPath), 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	txtMosaicImg, err := textmosaic.Generate(textmosaic.Config{
		Logger:          logger,
		Text:            mosaicText,
		InputImage:      img,
		MonoFontPath:    fontPath,
		TargetWidth:     0,
		BaseFontSize:    0,
		ContrastPercent: 0,
	})
	if err != nil {
		return fmt.Errorf("generate mosaic: %w", err)
	}

	logger.Debug("saving output", "path", finalOutputPath)

	if err := imaging.Save(txtMosaicImg, finalOutputPath); err != nil {
		return fmt.Errorf("save image %q: %w", finalOutputPath, err)
	}

	logger.Info(
		"success",
		"output", finalOutputPath,
		"size", fmt.Sprintf("%dx%d", txtMosaicImg.Bounds().Dx(), txtMosaicImg.Bounds().Dy()),
	)

	fmt.Printf("✅ Done! Saved to %s\n", finalOutputPath)
	return nil
}

func validateRequiredFlags(inputPath, textFile, fontPath string) error {
	for _, flag := range []struct {
		name  string
		value string
	}{
		{"-in", inputPath},
		{"-text", textFile},
		{"-font", fontPath},
	} {
		if strings.TrimSpace(flag.value) == "" {
			return fmt.Errorf("missing required flag: %s", flag.name)
		}
	}
	return nil
}

func resolveText(textFile string) (string, error) {
	textFile = strings.TrimSpace(textFile)

	if textFile != "" {
		b, err := os.ReadFile(textFile)
		if err != nil {
			return "", fmt.Errorf("read text file %q: %w", textFile, err)
		}

		content := strings.TrimSpace(string(b))
		if content == "" {
			return "", fmt.Errorf("text file %q is empty", textFile)
		}

		return content, nil
	}

	return "", errors.New("missing mosaic text: provide -text or -text-file")
}
