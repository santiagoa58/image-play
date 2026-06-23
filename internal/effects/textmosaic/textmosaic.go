package textmosaic

import (
	"errors"
	"image"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
)

type Config struct {
	Text            string
	InputImage      image.Image
	TargetWidth     int
	FontPath        string
	BaseFontSize    float64
	IsBlackAndWhite bool
	ContrastFactor  float64
}

type ImageFont struct {
	Context    *gg.Context
	FontSize   float64
	CharWidth  int
	CharHeight int
}

func Generate(conf Config) (image.Image, error) {
	if conf.InputImage == nil {
		return nil, errors.New("input image is required")
	}
	if conf.Text == "" {
		return nil, errors.New("text cannot be empty")
	}
	if conf.FontPath == "" {
		return nil, errors.New("font path is required")
	}
	applyDefaults(&conf)
	img := processImg(conf)
	_, err := loadImageFont(conf, img)
	if err != nil {
		return nil, err
	}

	return nil, errors.New("Not implemented")
}

func applyDefaults(conf *Config) {
	var (
		DEFAULT_FONT_SIZE       = 14.0
		DEFAULT_CONTRAST_FACTOR = 1.5
		DEFAULT_TARGET_WIDTH    = 1920
	)
	if conf.BaseFontSize <= 0 {
		conf.BaseFontSize = DEFAULT_FONT_SIZE
	}
	if conf.ContrastFactor <= 0 {
		conf.ContrastFactor = DEFAULT_CONTRAST_FACTOR
	}
	if conf.TargetWidth <= 0 {
		conf.TargetWidth = DEFAULT_TARGET_WIDTH
	}
}

func processImg(conf Config) image.Image {
	img := conf.InputImage
	if conf.TargetWidth > 0 {
		img = imaging.Resize(img, conf.TargetWidth, 0, imaging.Lanczos)
	}
	if conf.ContrastFactor != 1.0 {
		img = imaging.AdjustContrast(img, conf.ContrastFactor)
	}
	if conf.IsBlackAndWhite {
		img = imaging.Grayscale(img)
	}
	return img
}

func loadImageFont(c Config, img image.Image) (*ImageFont, error) {
	imgWidth := img.Bounds().Dx()
	fontSize := c.BaseFontSize
	ctx := gg.NewContextForImage(img)
	switch {
	case imgWidth >= 7200:
		fontSize *= 4.5
	case imgWidth >= 4800:
		fontSize *= 4.0
	case imgWidth >= 2160:
		fontSize *= 2.0
	case imgWidth >= 1080:
		fontSize *= 1.5
	case imgWidth <= 720:
		fontSize *= 0.75
	}
	if err := ctx.LoadFontFace(c.FontPath, fontSize); err != nil {
		return nil, err
	}
	// measures capital M: one of the widest letters
	_charW, _charH := ctx.MeasureString("M")

	charW := int(_charW)
	// makes spacing a bit more generous
	charH := int(_charH * 1.4)

	imgFnt := ImageFont{
		Context:    ctx,
		FontSize:   fontSize,
		CharWidth:  charW,
		CharHeight: charH,
	}

	return &imgFnt, nil
}
