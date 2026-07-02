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
	"github.com/santiagoa58/image-play/internal/textutils"
)

const (
	appName             = "mosaic"
	appVersion          = "dev"
	defaultOutputExt    = ".png"
	defaultOutputSuffix = "_mosaic"
)

const (
	fontPath        = "./fonts/NotoSansMono-VariableFont_wdth,wght.ttf"
	defaultTextFile = "./testdata/text/sample_text_message.txt"
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
		outputPath = flag.String("out", "", "Output path. Can be file or directory. Empty = input_mosaic.png")
		textFile   = flag.String("text", defaultTextFile, "Path to the text file to use for the mosaic. If empty, uses default text file.")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s — image effects toolkit\n\n", appName)
		fmt.Fprint(os.Stderr, "Usage:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	logger := slog.Default()
	logger.Info("mosaic starting", "version", appVersion)

	if err := validateRequiredFlags(*inputPath); err != nil {
		flag.Usage()
		return err
	}

	// err := mosaicRun(*inputPath, *outputPath, *textFile)
	// if err != nil {
	// 	return fmt.Errorf("mosaic run: %w", err)
	// }
	err := wordcloud.GenWordCloud(*inputPath, *outputPath, *textFile)
	if err != nil {
		return fmt.Errorf("wordcloud run: %w", err)
	}
	return nil
}

func mosaicRun(in, out, textFile string) error {
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

	finalOutputPath, err := textutils.ResolveOutputPath(in, out, "textmosaic")
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

func validateRequiredFlags(inputPath string) error {
	if strings.TrimSpace(inputPath) == "" {
		return errors.New("missing required flag: -in")
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
