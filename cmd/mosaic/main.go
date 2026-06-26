package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
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
	var (
		inputPath    = flag.String("in", "", "Path to input image (PNG, JPEG, WebP, etc.) [required]")
		outputPath   = flag.String("out", "", "Output path. Can be file or directory. Empty = input_<effect>.png")
		effect       = flag.String("effect", effectTextMosaic, `Effect to generate: "textmosaic" or "wordcloud"`)
		width        = flag.Int("width", 0, "Target width in pixels. Common: 1080, 1920, 3840. 0 = original size")
		fontPath     = flag.String("font", "", "Path to monospace TTF/OTF font. Empty = bundled font when available")
		text         = flag.String("text", "", "Text to use for the selected effect")
		textFile     = flag.String("text-file", "", "Path to UTF-8 text file to use as effect text")
		overwrite    = flag.Bool("overwrite", true, "Allow overwriting an existing output file")
		createDirs   = flag.Bool("create-dirs", true, "Create missing output directories")
		verbose      = flag.Bool("v", false, "Enable verbose/debug logging")
		baseFontSize = flag.Float64("font-size", 0, "Text mosaic base font size before scaling. 0 = default")
		bw           = flag.Bool("bw", false, "Text mosaic: convert source image to black and white before sampling")
		contrast     = flag.Float64("contrast", 0, "Text mosaic: contrast adjustment percent. 0 = no change, 20 = increase by 20%")

		packingProfile             = flag.String("packing-profile", "", `Word cloud: packing profile: "", "binary-silhouette", "tonal-detail", or "foreground-background"`)
		qualityPreset              = flag.String("quality", "", `Word cloud: quality preset: "", "fast", "balanced", "dense", or "poster". Empty = auto`)
		maskType                   = flag.String("mask-type", "", `Word cloud: mask type: "", "dark", "light", or "all". Empty = profile/default`)
		maskThreshold              = flag.Float64("mask-threshold", defaultOptionalFloatFlag, "Word cloud: mask luminance threshold from 0..1. Negative = default")
		foregroundThreshold        = flag.Float64("foreground-threshold", defaultOptionalFloatFlag, "Word cloud: foreground mask threshold from 0..1. Negative = default")
		foregroundMaskPath         = flag.String("foreground-mask", "", "Word cloud: optional foreground mask path. White = foreground, black = background")
		saveForegroundMaskPath     = flag.String("save-foreground-mask", "", "Word cloud: optional path to save the OpenCV foreground mask")
		foregroundRect             = flag.String("foreground-rect", "", "Word cloud: OpenCV GrabCut rectangle as x,y,width,height. Empty = default central foreground")
		grabCutIterations          = flag.Int("grabcut-iterations", 5, "Word cloud: OpenCV GrabCut iterations for automatic foreground masks")
		includeProbableForeground  = flag.Bool("include-probable-foreground", true, "Word cloud: include OpenCV probable-foreground pixels in the mask")
		alphaThreshold             = flag.Float64("alpha-threshold", defaultOptionalFloatFlag, "Word cloud: source alpha threshold from 0..1. Negative = default")
		glyphAlphaThreshold        = flag.Float64("glyph-alpha-threshold", defaultOptionalFloatFlag, "Word cloud: rendered glyph alpha threshold from 0..1. Negative = default")
		minFontSize                = flag.Int("min-font-size", 0, "Word cloud: minimum font size. 0 = default")
		maxFontSize                = flag.Int("max-font-size", 0, "Word cloud: maximum font size. 0 = default")
		sizeExponent               = flag.Float64("size-exponent", 0, "Word cloud: frequency-to-font-size exponent. 0 = default")
		fillerWordCount            = flag.Int("filler-word-count", 0, "Word cloud: number of extra filler word candidates. 0 = quality preset/default")
		densityAlias               = flag.Int("density", 0, "Word cloud: alias for -filler-word-count")
		wordPadding                = flag.Int("word-padding", defaultOptionalIntegerFlag, "Word cloud: padding in pixels around glyphs. Negative = profile/default")
		maxPlacementAttempts       = flag.Int("max-placement-attempts", 0, "Word cloud: max placement checks per word. 0 = quality preset/default")
		maxHeroPlacementAttempts   = flag.Int("max-hero-placement-attempts", 0, "Word cloud: max placement checks for hero words. 0 = default")
		maxFillerPlacementAttempts = flag.Int("max-filler-placement-attempts", 0, "Word cloud: max placement checks for filler words. 0 = default")
		heroWordCount              = flag.Int("hero-word-count", 0, "Word cloud: number of largest words to treat as hero words. 0 = default")
		finalFillPasses            = flag.Int("final-fill-passes", defaultOptionalIntegerFlag, "Word cloud: final tiny-word fill passes. Negative = preset/default")
		finalFillFontSize          = flag.Int("final-fill-font-size", 0, "Word cloud: final fill font size. 0 = min font size")
		fillerMaxScale             = flag.Float64("filler-max-scale", 0, "Word cloud: max filler size as fraction of max font size. 0 = default")
		detailPlacementBias        = flag.Float64("detail-placement-bias", 0, "Word cloud: prefer source-detail regions during tonal final fill. 0 = profile/default")
		colorMode                  = flag.String("color-mode", "", `Word cloud: color mode: "", "source", "palette", "luminance-palette", "random-palette", or "sequential-palette". Empty = auto`)
		palette                    = flag.String("palette", "", "Word cloud: comma-separated hex colors, e.g. '#111111,#777777,#eeeeee'")
		background                 = flag.String("background", "#ffffff", "Word cloud: canvas background hex color. Empty = effect default")
		inferBackground            = flag.Bool("infer-background", false, "Word cloud: infer background from non-playable pixels")
		seed                       = flag.Int64("seed", 1, "Word cloud: deterministic random seed")
		minWordLength              = flag.Int("min-word-length", 0, "Word cloud: minimum parsed word length. 0 = default")
		maxWordLength              = flag.Int("max-word-length", 0, "Word cloud: maximum parsed word length. 0 = no limit")
		stopWords                  = flag.String("stop-words", "", "Word cloud: comma-separated words to exclude")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s — image effects toolkit\n\n", appName)
		fmt.Fprint(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s -effect textmosaic -in photo.jpg -font /path/to/monofont.ttf -text-file message.txt\n", appName)
		fmt.Fprintf(os.Stderr, "  %s -effect wordcloud -in silhouette.png -font /path/to/monofont.ttf -text-file words.txt -quality dense\n\n", appName)
		fmt.Fprint(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprint(os.Stderr, "\nCommon widths: 1080 (HD), 1920 (Full HD), 3840 (4K)\n")
	}

	flag.Parse()

	selectedEffect, err := normalizeEffectName(*effect)
	if err != nil {
		flag.Usage()
		return err
	}

	logger := newLogger(*verbose)
	logger.Info("mosaic starting", "version", appVersion, "effect", selectedEffect)

	if err := validateRequiredFlags(*inputPath); err != nil {
		flag.Usage()
		return err
	}

	logger.Debug("loading image", "path", *inputPath)

	img, err := imaging.Open(*inputPath)
	if err != nil {
		return fmt.Errorf("open input image %q: %w", *inputPath, err)
	}
	resolvedFontPath, err := resolveFontPath(*fontPath)
	if err != nil {
		return err
	}
	effectText, err := resolveText(*text, *textFile, defaultTextForEffect(selectedEffect))
	if err != nil {
		return err
	}
	resolvedWidth := resolveTargetWidth(*width, selectedEffect)

	finalOutputPath, err := util.ResolveOutputPath(*inputPath, *outputPath, util.OutputPathOptions{
		DefaultExt:        defaultOutputExt,
		DefaultSuffix:     defaultOutputSuffix(selectedEffect),
		AllowCreateParent: *createDirs,
		AllowOverwrite:    *overwrite,
	})
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}

	logger.Debug("resolved output path", "path", finalOutputPath)

	if *createDirs {
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
		outputImage, err = textmosaic.Generate(textmosaic.Config{
			Logger:          logger,
			Text:            effectText,
			InputImage:      img,
			MonoFontPath:    resolvedFontPath,
			TargetWidth:     resolvedWidth,
			BaseFontSize:    *baseFontSize,
			IsBlackAndWhite: *bw,
			ContrastPercent: *contrast,
		})
		if err != nil {
			return fmt.Errorf("generate text mosaic: %w", err)
		}
	case effectWordCloud:
		result, err := generateWordCloud(logger, img, effectText, resolvedFontPath, resolvedWidth, wordCloudOptions{
			packingProfile:             *packingProfile,
			qualityPreset:              *qualityPreset,
			maskType:                   *maskType,
			maskThreshold:              *maskThreshold,
			foregroundThreshold:        *foregroundThreshold,
			foregroundMaskPath:         *foregroundMaskPath,
			saveForegroundMaskPath:     *saveForegroundMaskPath,
			foregroundRect:             *foregroundRect,
			grabCutIterations:          *grabCutIterations,
			includeProbableForeground:  *includeProbableForeground,
			alphaThreshold:             *alphaThreshold,
			glyphAlphaThreshold:        *glyphAlphaThreshold,
			minFontSize:                *minFontSize,
			maxFontSize:                *maxFontSize,
			sizeExponent:               *sizeExponent,
			fillerWordCount:            *fillerWordCount,
			densityAlias:               *densityAlias,
			wordPadding:                *wordPadding,
			maxPlacementAttempts:       *maxPlacementAttempts,
			maxHeroPlacementAttempts:   *maxHeroPlacementAttempts,
			maxFillerPlacementAttempts: *maxFillerPlacementAttempts,
			heroWordCount:              *heroWordCount,
			finalFillPasses:            *finalFillPasses,
			finalFillFontSize:          *finalFillFontSize,
			fillerMaxScale:             *fillerMaxScale,
			detailPlacementBias:        *detailPlacementBias,
			colorMode:                  *colorMode,
			palette:                    *palette,
			background:                 *background,
			inferBackground:            *inferBackground,
			seed:                       *seed,
			minWordLength:              *minWordLength,
			maxWordLength:              *maxWordLength,
			stopWords:                  *stopWords,
		})
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
