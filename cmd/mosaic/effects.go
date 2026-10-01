package main

import (
	"fmt"
	"strings"

	"github.com/santiagoa58/image-play/internal/effects/textmosaic"
	"github.com/santiagoa58/image-play/internal/effects/wordcloud"
)

type effect string

const (
	effectWordCloud  effect = "wordcloud"
	effectTextMosaic effect = "textmosaic"
)

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
