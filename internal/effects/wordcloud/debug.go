package wordcloud

import (
	"fmt"
	"image"
	"path/filepath"
	"strings"

	"github.com/santiagoa58/image-play/internal/imageutil"
	"gocv.io/x/gocv"
)

func debugPath(outpath, label string) string {
	ext := filepath.Ext(outpath)
	base := strings.TrimSuffix(outpath, ext)
	return base + "_" + label + ".png"
}

func writeDebugMat(enabled bool, outpath, label string, mat gocv.Mat) error {
	if !enabled {
		return nil
	}
	path := debugPath(outpath, label)
	if ok := gocv.IMWrite(path, mat); !ok {
		return fmt.Errorf("write debug image %q", path)
	}
	return nil
}

func writeMaskDebug(
	enabled bool,
	outputPath string,
	mask *imageutil.Mask,
) error {
	if !enabled {
		return nil
	}
	if mask == nil || mask.BinaryMat == nil {
		return fmt.Errorf("write mask debug image: binary mask is unavailable")
	}

	return writeDebugMat(true, outputPath, "01-mask", *mask.BinaryMat)
}

func writeDistanceDebug(
	enabled bool,
	outputPath string,
	mask *imageutil.Mask,
) error {
	if !enabled {
		return nil
	}
	if mask == nil || mask.DistMat == nil {
		return fmt.Errorf("write distance debug image: distance mask is unavailable")
	}

	normalized := gocv.NewMat()
	defer normalized.Close()

	if err := gocv.Normalize(
		*mask.DistMat,
		&normalized,
		0,
		255,
		gocv.NormMinMax,
	); err != nil {
		return fmt.Errorf("normalize distance debug image: %w", err)
	}

	display := gocv.NewMat()
	defer display.Close()

	if err := normalized.ConvertTo(&display, gocv.MatTypeCV8U); err != nil {
		return fmt.Errorf("convert distance debug image: %w", err)
	}

	return writeDebugMat(true, outputPath, "02-distance", display)
}

func writeRegionsDebug(enabled bool, outputPath string, regions *regionPolicy) error {
	if !enabled {
		return nil
	}
	if regions == nil {
		return fmt.Errorf("write regions debug image: regions are unavailable")
	}
	display := gocv.NewMatWithSize(regions.height, regions.width, gocv.MatTypeCV8UC1)
	defer display.Close()
	pixels, err := display.DataPtrUint8()
	if err != nil {
		return err
	}
	for i, id := range regions.labels {
		if id >= 0 {
			pixels[i] = uint8(40 + (id*73)%200)
		}
	}
	return writeDebugMat(true, outputPath, "03-regions", display)
}

func writePlacementDebug(
	enabled bool,
	outputPath string,
	safeZone,
	occupancy gocv.Mat,
) error {
	if !enabled {
		return nil
	}

	if err := writeDebugMat(
		true,
		outputPath,
		"04-safe-zone",
		safeZone,
	); err != nil {
		return err
	}

	return writeDebugMat(
		true,
		outputPath,
		"05-occupancy",
		occupancy,
	)
}

func writeWordcloudDebug(
	enabled bool,
	outputPath string,
	img image.Image,
) error {
	if !enabled {
		return nil
	}
	if img == nil {
		return fmt.Errorf("write word-cloud debug image: rendered image is unavailable")
	}

	display, err := gocv.ImageToMatRGB(img)
	if err != nil {
		return fmt.Errorf("convert rendered word cloud to matrix: %w", err)
	}
	defer display.Close()

	return writeDebugMat(true, outputPath, "06-wordcloud", display)
}
