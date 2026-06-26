package main

import (
	"flag"
	"image"
	"log/slog"

	"github.com/santiagoa58/image-play/internal/effects/textmosaic"
)

type textMosaicOptions struct {
	baseFontSize float64
	bw           bool
	contrast     float64
	textWeight   int
}

func registerTextMosaicFlags(fs *flag.FlagSet, opts *textMosaicOptions) {
	fs.Float64Var(&opts.baseFontSize, "font-size", 0, "Text mosaic: base font size before scaling. 0 = default")
	fs.BoolVar(&opts.bw, "bw", false, "Text mosaic: convert source image to black and white before sampling")
	fs.Float64Var(&opts.contrast, "contrast", 18, "Text mosaic: contrast adjustment percent. 0 = no change")
	fs.IntVar(&opts.textWeight, "text-weight", 2, "Text mosaic: synthetic text weight from 1..4")
}

func generateTextMosaic(logger *slog.Logger, img image.Image, text string, fontPath string, targetWidth int, opts textMosaicOptions) (image.Image, error) {
	return textmosaic.Generate(textmosaic.Config{
		Logger:          logger,
		Text:            text,
		InputImage:      img,
		MonoFontPath:    fontPath,
		TargetWidth:     targetWidth,
		BaseFontSize:    opts.baseFontSize,
		IsBlackAndWhite: opts.bw,
		ContrastPercent: opts.contrast,
		TextWeight:      opts.textWeight,
	})
}
