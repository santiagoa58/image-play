package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/santiagoa58/image-play/internal/effects/wordcloud"
)

const (
	appName    = "mosaic"
	appVersion = "dev"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var (
		inputPath  = flag.String("in", "", "Path to input image (PNG, JPEG, WebP, etc.) [required]")
		outputPath = flag.String("out", "", "Output PNG path or directory. Empty = input_wordcloud.png")
		textFile   = flag.String("text", "", "Path to text file [required]")
		fontPath   = flag.String("font", "", "Path to a TTF or OTF font file [required]")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s — image-shaped word cloud generator\n\n", appName)
		fmt.Fprintln(os.Stderr, "Usage:")
		flag.PrintDefaults()
	}

	flag.Parse()

	slog.Info("mosaic starting", "version", appVersion)

	if err := validateRequiredFlags(*inputPath, *textFile, *fontPath); err != nil {
		flag.Usage()
		return err
	}

	cfg := wordcloud.NewConfig(
		wordcloud.WithInputPath(*inputPath),
		wordcloud.WithOutputPath(*outputPath),
		wordcloud.WithTextPath(*textFile),
		wordcloud.WithFontPath(*fontPath),
	)

	if err := wordcloud.Generate(cfg); err != nil {
		return fmt.Errorf("generate word cloud: %w", err)
	}

	return nil
}

func validateRequiredFlags(inputPath, textFile, fontPath string) error {
	for _, required := range []struct {
		name  string
		value string
	}{
		{"-in", inputPath},
		{"-text", textFile},
		{"-font", fontPath},
	} {
		if strings.TrimSpace(required.value) == "" {
			return fmt.Errorf("missing required flag: %s", required.name)
		}
	}

	return nil
}
