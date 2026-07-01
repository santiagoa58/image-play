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
	"github.com/santiagoa58/image-play/internal/util"
)

const (
	appName             = "mosaic"
	appVersion          = "dev"
	defaultOutputExt    = ".png"
	defaultOutputSuffix = "_mosaic"
)

const (
	fontPath = "./fonts/NotoSansMono-VariableFont_wdth,wght.ttf"
	textFile = "./testdata/text/sample_text_message.txt"
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
		verbose    = flag.Bool("v", true, "Enable verbose/debug logging")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s — image effects toolkit\n\n", appName)
		fmt.Fprint(os.Stderr, "Usage:\n")
		fmt.Fprint(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprint(os.Stderr, "\nCommon widths: 1080 (HD), 1920 (Full HD), 3840 (4K)\n")
	}

	flag.Parse()

	logger := newLogger(*verbose)
	logger.Info("mosaic starting", "version", appVersion)

	if err := validateRequiredFlags(*inputPath); err != nil {
		flag.Usage()
		return err
	}

	// err := mosaicRun(*inputPath, *outputPath, logger)
	// if err != nil {
	// 	return fmt.Errorf("mosaic run: %w", err)
	// }
	err := wordcloud.GenWordCloud(*inputPath, *outputPath, textFile)
	if err != nil {
		return fmt.Errorf("wordcloud run: %w", err)
	}
	return nil
}

func mosaicRun(in, out string, logger *slog.Logger) error {
	mosaicText, err := resolveText(textFile)
	if err != nil {
		return err
	}

	logger.Debug("loading image", "path", in)

	img, err := imaging.Open(in)
	if err != nil {
		return fmt.Errorf("open input image %q: %w", in, err)
	}

	finalOutputPath, err := util.ResolveOutputPath(in, out, util.OutputPathOptions{
		DefaultExt:    defaultOutputExt,
		DefaultSuffix: defaultOutputSuffix,
	})
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
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

func newLogger(verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
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
