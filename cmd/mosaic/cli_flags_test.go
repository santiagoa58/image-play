package main

import (
	"bytes"
	"flag"
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/santiagoa58/image-play/internal/effects/wordcloud"
)

func TestRegisterCLIFlagsParsesSimpleWordCloudCommand(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	opts := registerCLIFlags(fs)

	if err := fs.Parse([]string{
		"-effect", "wordcloud",
		"-in", "input.jpg",
		"-out", "output.png",
	}); err != nil {
		t.Fatalf("parse simple flags: %v", err)
	}

	if opts.common.effect != "wordcloud" {
		t.Fatalf("expected wordcloud effect, got %q", opts.common.effect)
	}
	if opts.common.inputPath != "input.jpg" || opts.common.outputPath != "output.png" {
		t.Fatalf("unexpected paths: %#v", opts.common)
	}
	if opts.common.fontPath != "" || opts.wordCloud.minFontSize != 0 {
		t.Fatalf("expected optional defaults to remain unset before effect defaults, got %#v %#v", opts.common, opts.wordCloud)
	}
}

func TestUsageKeepsDefaultHelpConcise(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	registerCLIFlags(fs)
	var out bytes.Buffer
	fs.SetOutput(&out)
	configureUsage(fs)

	fs.Usage()
	usage := out.String()
	if !strings.Contains(usage, "Simple usage:") {
		t.Fatalf("expected simple usage, got:\n%s", usage)
	}
	if !strings.Contains(usage, "-help-advanced") {
		t.Fatalf("expected advanced-help hint, got:\n%s", usage)
	}
	if strings.Contains(usage, "-glyph-alpha-threshold") {
		t.Fatalf("default help should not dump advanced flags, got:\n%s", usage)
	}
}

func TestRegisterCLIFlagsUsesReadableTextMosaicDefaults(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	opts := registerCLIFlags(fs)

	if err := fs.Parse([]string{
		"-effect", "textmosaic",
		"-in", "input.jpg",
		"-out", "output.png",
	}); err != nil {
		t.Fatalf("parse simple text mosaic flags: %v", err)
	}

	if opts.text.contrast <= 0 {
		t.Fatalf("expected default text mosaic contrast boost, got %.2f", opts.text.contrast)
	}
	if opts.text.textWeight < 2 {
		t.Fatalf("expected default text mosaic weight to be bold, got %d", opts.text.textWeight)
	}
}

func TestResolveWordCloudAutoOptionsSelectsHighContrastPoster(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 255})
		}
	}
	for y := 20; y < 80; y++ {
		for x := 20; x < 80; x++ {
			img.Set(x, y, color.RGBA{180, 40, 20, 255})
		}
	}

	opts := resolveWordCloudAutoOptions(img, wordCloudOptions{})

	if opts.packingProfile != wordcloud.PackingProfileBinarySilhouette {
		t.Fatalf("expected binary silhouette profile, got %q", opts.packingProfile)
	}
	if opts.maskType != wordcloud.MaskTypeContrast {
		t.Fatalf("expected contrast mask type, got %q", opts.maskType)
	}
	if opts.background != "" {
		t.Fatalf("expected image-derived background, got %q", opts.background)
	}
	if opts.colorMode != wordcloud.ColorModeRandomPalette {
		t.Fatalf("expected random source-palette mode, got %q", opts.colorMode)
	}
	if opts.palette != "" {
		t.Fatalf("expected no hard-coded palette, got %q", opts.palette)
	}
	if !opts.derivePaletteFromSource {
		t.Fatal("expected palette to be derived from the input image")
	}
}

func TestResolveWordCloudWorkWidthAutoCapsLargeOutput(t *testing.T) {
	got, err := resolveWordCloudWorkWidth(3840, 0)
	if err != nil {
		t.Fatalf("resolveWordCloudWorkWidth returned error: %v", err)
	}
	if got != 1920 {
		t.Fatalf("expected large output to auto-pack at 1920px, got %d", got)
	}
}

func TestResolveWordCloudWorkWidthAllowsDirectRenderOverride(t *testing.T) {
	got, err := resolveWordCloudWorkWidth(3840, 3840)
	if err != nil {
		t.Fatalf("resolveWordCloudWorkWidth returned error: %v", err)
	}
	if got != 3840 {
		t.Fatalf("expected explicit work width to be honored, got %d", got)
	}
}
