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
	"github.com/santiagoa58/image-play/internal/util"
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
		inputPath    = flag.String("in", "", "Path to input image (PNG, JPEG, WebP, etc.) [required]")
		outputPath   = flag.String("out", "", "Output path. Can be file or directory. Empty = input_mosaic.png")
		width        = flag.Int("width", 0, "Target width in pixels. Common: 1080, 1920, 3840. 0 = original size")
		fontPath     = flag.String("font", "", "Path to monospace TTF/OTF font [required]")
		text         = flag.String("text", "", "Text to repeat across the mosaic")
		textFile     = flag.String("text-file", "", "Path to UTF-8 text file to use as mosaic text")
		bw           = flag.Bool("bw", false, "Convert source image to black and white before sampling")
		contrast     = flag.Float64("contrast", 0, "Contrast adjustment percent. 0 = no change, 20 = increase by 20%")
		overwrite    = flag.Bool("overwrite", true, "Allow overwriting an existing output file")
		createDirs   = flag.Bool("create-dirs", true, "Create missing output directories")
		verbose      = flag.Bool("v", false, "Enable verbose/debug logging")
		baseFontSize = flag.Float64("font-size", 0, "Base font size before scaling. 0 = default")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s — image effects toolkit\n\n", appName)
		fmt.Fprint(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s -in photo.jpg -font /path/to/monofont.ttf -text \"hello world\"\n", appName)
		fmt.Fprintf(os.Stderr, "  %s -in photo.jpg -out result.png -width 1920 -font /path/to/monofont.ttf -text-file message.txt\n\n", appName)
		fmt.Fprint(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprint(os.Stderr, "\nCommon widths: 1080 (HD), 1920 (Full HD), 3840 (4K)\n")
	}

	flag.Parse()

	logger := newLogger(*verbose)
	logger.Info("mosaic starting", "version", appVersion)

	if err := validateRequiredFlags(*inputPath, *fontPath); err != nil {
		flag.Usage()
		return err
	}

	mosaicText, err := resolveText(*text, *textFile)
	if err != nil {
		return err
	}

	logger.Debug("loading image", "path", *inputPath)

	img, err := imaging.Open(*inputPath)
	if err != nil {
		return fmt.Errorf("open input image %q: %w", *inputPath, err)
	}

	finalOutputPath, err := util.ResolveOutputPath(*inputPath, *outputPath, util.OutputPathOptions{
		DefaultExt:        defaultOutputExt,
		DefaultSuffix:     defaultOutputSuffix,
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

	txtMosaicImg, err := textmosaic.Generate(textmosaic.Config{
		Logger:          logger,
		Text:            mosaicText,
		InputImage:      img,
		MonoFontPath:    *fontPath,
		TargetWidth:     *width,
		BaseFontSize:    *baseFontSize,
		IsBlackAndWhite: *bw,
		ContrastPercent: *contrast,
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

func validateRequiredFlags(inputPath, fontPath string) error {
	if strings.TrimSpace(inputPath) == "" {
		return errors.New("missing required flag: -in")
	}

	if strings.TrimSpace(fontPath) == "" {
		return errors.New("missing required flag: -font")
	}

	return nil
}

func resolveText(text, textFile string) (string, error) {
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

	return "", errors.New("missing mosaic text: provide -text or -text-file")
}
