package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/santiagoa58/image-play/internal/effects/textmosaic"
	"github.com/santiagoa58/image-play/internal/effects/wordcloud"
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
	inputPath     string
	outputPath    string
	textPath      string
	fontPath      string
	uppercase     bool
	letterSpacing float64
	wordSpacing   float64
	outputScale   float64
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	textDefaults := textmosaic.NewConfig()
	var (
		uppercase  = flag.Bool("uppercase", false, "Display text in uppercase")
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
		letterSpacing = flag.Float64("letter-spacing", textDefaults.LetterSpacing, "Textmosaic character spacing in em; negative values tighten")
		wordSpacing   = flag.Float64("word-spacing", textDefaults.WordSpacing, "Textmosaic extra space after word gaps in em, beyond letter spacing; negative values tighten")
		outputScale   = flag.Float64("output-scale", textDefaults.OutputScale, "Textmosaic export scale (1 = source resolution); upscaling caps at 4096 pixels on the longest side")
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
		inputPath:     *inputPath,
		outputPath:    *outputPath,
		textPath:      *textPath,
		fontPath:      *fontPath,
		uppercase:     *uppercase,
		letterSpacing: *letterSpacing,
		wordSpacing:   *wordSpacing,
		outputScale:   *outputScale,
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
		cfg := wordcloud.NewConfig(
			wordcloud.WithInputPath(opts.inputPath),
			wordcloud.WithOutputPath(opts.outputPath),
			wordcloud.WithTextPath(opts.textPath),
			wordcloud.WithFontPath(opts.fontPath),
			wordcloud.WithUppercase(opts.uppercase),
		)
		if err := wordcloud.Generate(cfg); err != nil {
			return fmt.Errorf("generate word cloud: %w", err)
		}
		return nil

	case effectTextMosaic:
		cfg := textmosaic.NewConfig(
			textmosaic.WithInputPath(opts.inputPath),
			textmosaic.WithOutputPath(opts.outputPath),
			textmosaic.WithTextPath(opts.textPath),
			textmosaic.WithFontPath(opts.fontPath),
			textmosaic.WithUppercase(opts.uppercase),
			textmosaic.WithLetterSpacing(opts.letterSpacing),
			textmosaic.WithWordSpacing(opts.wordSpacing),
			textmosaic.WithOutputScale(opts.outputScale),
		)
		if err := textmosaic.Generate(cfg); err != nil {
			return fmt.Errorf("generate text mosaic: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unsupported effect %q", selected)
	}
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
