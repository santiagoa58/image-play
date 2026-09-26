package main

import (
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
	appName    = "mosaic"
	appVersion = "dev"
)

type effect string

const (
	effectWordCloud  effect = "wordcloud"
	effectTextMosaic effect = "textmosaic"
)

type options struct {
	inputPath  string
	outputPath string
	textPath   string
	fontPath   string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var (
		effectName = flag.String(
			"effect",
			"",
			"Effect to apply: wordcloud or textmosaic [required]",
		)
		inputPath = flag.String(
			"in",
			"",
			"Path to input image (PNG, JPEG, WebP, etc.) [required]",
		)
		outputPath = flag.String(
			"out",
			"",
			"Output PNG path or directory. Empty = input_<effect>.png",
		)
		textPath = flag.String(
			"text",
			"",
			"Path to text file [required]",
		)
		fontPath = flag.String(
			"font",
			"",
			"Path to a TTF or OTF font file [required]",
		)
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s — image effects toolkit\n\n", appName)
		fmt.Fprintln(os.Stderr, "Usage:")
		flag.PrintDefaults()
	}

	flag.Parse()

	selectedEffect, err := parseEffect(*effectName)
	if err != nil {
		flag.Usage()
		return err
	}

	opts := options{
		inputPath:  *inputPath,
		outputPath: *outputPath,
		textPath:   *textPath,
		fontPath:   *fontPath,
	}
	if err := validateRequiredFlags(opts); err != nil {
		flag.Usage()
		return err
	}

	slog.Info(
		"mosaic starting",
		"version", appVersion,
		"effect", selectedEffect,
	)

	return runEffect(selectedEffect, opts)
}

func parseEffect(value string) (effect, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(effectWordCloud):
		return effectWordCloud, nil
	case string(effectTextMosaic):
		return effectTextMosaic, nil
	case "":
		return "", fmt.Errorf("missing required flag: -effect")
	default:
		return "", fmt.Errorf(
			"unsupported effect %q: choose %q or %q",
			value,
			effectWordCloud,
			effectTextMosaic,
		)
	}
}

func runEffect(selected effect, opts options) error {
	switch selected {
	case effectWordCloud:
		return runWordCloud(opts)
	case effectTextMosaic:
		return runTextMosaic(opts)
	default:
		return fmt.Errorf("unsupported effect %q", selected)
	}
}

func runWordCloud(opts options) error {
	cfg := wordcloud.NewConfig(
		wordcloud.WithInputPath(opts.inputPath),
		wordcloud.WithOutputPath(opts.outputPath),
		wordcloud.WithTextPath(opts.textPath),
		wordcloud.WithFontPath(opts.fontPath),
	)

	if err := wordcloud.Generate(cfg); err != nil {
		return fmt.Errorf("generate word cloud: %w", err)
	}

	return nil
}

func runTextMosaic(opts options) error {
	textBytes, err := os.ReadFile(opts.textPath)
	if err != nil {
		return fmt.Errorf("read text file %q: %w", opts.textPath, err)
	}

	mosaicText := strings.TrimSpace(string(textBytes))
	if mosaicText == "" {
		return fmt.Errorf("text file %q is empty", opts.textPath)
	}

	inputImage, err := imaging.Open(opts.inputPath)
	if err != nil {
		return fmt.Errorf("open input image %q: %w", opts.inputPath, err)
	}

	outputPath, err := textutil.ResolveOutputPath(
		opts.inputPath,
		opts.outputPath,
		string(effectTextMosaic),
		".png",
	)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	result, err := textmosaic.Generate(textmosaic.Config{
		Logger:       slog.Default(),
		Text:         mosaicText,
		InputImage:   inputImage,
		MonoFontPath: opts.fontPath,
	})
	if err != nil {
		return fmt.Errorf("generate text mosaic: %w", err)
	}

	if err := imaging.Save(result, outputPath); err != nil {
		return fmt.Errorf("save text mosaic %q: %w", outputPath, err)
	}

	slog.Info(
		"text mosaic generated",
		"output", outputPath,
		"width", result.Bounds().Dx(),
		"height", result.Bounds().Dy(),
	)

	return nil
}

func validateRequiredFlags(opts options) error {
	for _, required := range []struct {
		name  string
		value string
	}{
		{"-in", opts.inputPath},
		{"-text", opts.textPath},
		{"-font", opts.fontPath},
	} {
		if strings.TrimSpace(required.value) == "" {
			return fmt.Errorf("missing required flag: %s", required.name)
		}
	}

	return nil
}
