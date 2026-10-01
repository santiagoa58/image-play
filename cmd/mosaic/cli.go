package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/santiagoa58/image-play/internal/effects/textmosaic"
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
