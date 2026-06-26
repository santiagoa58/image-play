package main

import (
	"errors"
	"fmt"
	"image"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/santiagoa58/image-play/internal/effects/wordcloud"
	"github.com/santiagoa58/image-play/internal/util"
)

const (
	appName                    = "mosaic"
	appVersion                 = "dev"
	effectTextMosaic           = "textmosaic"
	effectWordCloud            = "wordcloud"
	defaultOutputExt           = ".png"
	textMosaicOutputSuffix     = "_mosaic"
	wordCloudOutputSuffix      = "_wordcloud"
	defaultOptionalFloatFlag   = -1.0
	defaultOptionalIntegerFlag = -1
	defaultFontPath            = "fonts/NotoSansMono-VariableFont_wdth,wght.ttf"
	defaultWordCloudText       = "memory portrait travel celebration family friends together joy love promise heart city sky building detail journey bright calm"
	defaultTextMosaicText      = "Every image effect in this project starts with a source picture and turns it into a new visual texture."
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	opts := registerCLIFlags(flagCommandLine())
	configureUsage(flagCommandLine())
	flagCommandLine().Parse(os.Args[1:])
	if opts.common.advanced {
		printAdvancedUsage(flagCommandLine())
		return nil
	}

	selectedEffect, err := normalizeEffectName(opts.common.effect)
	if err != nil {
		flagCommandLine().Usage()
		return err
	}

	logger := newLogger(opts.common.verbose)
	logger.Info("mosaic starting", "version", appVersion, "effect", selectedEffect)

	if err := validateRequiredFlags(opts.common.inputPath); err != nil {
		flagCommandLine().Usage()
		return err
	}

	logger.Debug("loading image", "path", opts.common.inputPath)

	img, err := imaging.Open(opts.common.inputPath)
	if err != nil {
		return fmt.Errorf("open input image %q: %w", opts.common.inputPath, err)
	}
	resolvedFontPath, err := resolveFontPath(opts.common.fontPath)
	if err != nil {
		return err
	}
	effectText, err := resolveText(opts.common.text, opts.common.textFile, defaultTextForEffect(selectedEffect))
	if err != nil {
		return err
	}
	resolvedWidth := resolveTargetWidth(opts.common.width, selectedEffect)

	finalOutputPath, err := util.ResolveOutputPath(opts.common.inputPath, opts.common.outputPath, util.OutputPathOptions{
		DefaultExt:        defaultOutputExt,
		DefaultSuffix:     defaultOutputSuffix(selectedEffect),
		AllowCreateParent: opts.common.createDirs,
		AllowOverwrite:    opts.common.overwrite,
	})
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}

	logger.Debug("resolved output path", "path", finalOutputPath)

	if opts.common.createDirs {
		if err := os.MkdirAll(filepath.Dir(finalOutputPath), 0755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
	}

	var (
		outputImage image.Image
		stats       *wordcloud.Stats
	)

	switch selectedEffect {
	case effectTextMosaic:
		outputImage, err = generateTextMosaic(logger, img, effectText, resolvedFontPath, resolvedWidth, opts.text)
		if err != nil {
			return fmt.Errorf("generate text mosaic: %w", err)
		}
	case effectWordCloud:
		result, err := generateWordCloud(logger, img, effectText, resolvedFontPath, resolvedWidth, opts.wordCloud)
		if err != nil {
			return fmt.Errorf("generate word cloud: %w", err)
		}
		outputImage = result.Image
		stats = &result.Stats
	default:
		return fmt.Errorf("unsupported effect %q", selectedEffect)
	}

	logger.Debug("saving output", "path", finalOutputPath)

	if err := imaging.Save(outputImage, finalOutputPath); err != nil {
		return fmt.Errorf("save image %q: %w", finalOutputPath, err)
	}

	logger.Info(
		"success",
		"output", finalOutputPath,
		"size", fmt.Sprintf("%dx%d", outputImage.Bounds().Dx(), outputImage.Bounds().Dy()),
	)

	if stats != nil {
		fmt.Printf(
			"✅ Done! Saved to %s (placed %d/%d words, occupied %.1f%% of mask)\n",
			finalOutputPath,
			stats.PlacedWords,
			stats.AttemptedWords,
			stats.OccupiedCoverage*100,
		)
		return nil
	}

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

func resolveFontPath(fontPath string) (string, error) {
	fontPath = strings.TrimSpace(fontPath)
	if fontPath != "" {
		return fontPath, nil
	}
	if _, err := os.Stat(defaultFontPath); err == nil {
		return defaultFontPath, nil
	}
	return "", errors.New("missing required flag: -font")
}

func resolveText(text, textFile string, defaultText string) (string, error) {
	text = strings.TrimSpace(text)
	textFile = strings.TrimSpace(textFile)

	if text != "" && textFile != "" {
		return "", errors.New("use either -text or -text-file, not both")
	}

	if text != "" {
		return text, nil
	}

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

	return defaultText, nil
}

func defaultTextForEffect(effect string) string {
	if effect == effectWordCloud {
		return defaultWordCloudText
	}
	return defaultTextMosaicText
}

func resolveTargetWidth(width int, effect string) int {
	if width != 0 {
		return width
	}
	if effect == effectWordCloud {
		return 512
	}
	return width
}

func normalizeEffectName(effect string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(effect)) {
	case "", "text", "mosaic", "text-mosaic", effectTextMosaic:
		return effectTextMosaic, nil
	case "word-cloud", "cloud", effectWordCloud:
		return effectWordCloud, nil
	default:
		return "", fmt.Errorf("unknown effect %q", effect)
	}
}

func defaultOutputSuffix(effect string) string {
	switch effect {
	case effectWordCloud:
		return wordCloudOutputSuffix
	default:
		return textMosaicOutputSuffix
	}
}
