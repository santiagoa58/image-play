package imageutil

import (
	"image"
	"image/color"
	"math"
)

type LightBackgroundMaskConfig struct {
	DistanceThreshold float64
}

func LightBackgroundForegroundMask(img image.Image, conf LightBackgroundMaskConfig) image.Image {
	bounds := img.Bounds()
	threshold := conf.DistanceThreshold
	if threshold <= 0 {
		threshold = 0.075
	}
	bgR, bgG, bgB := averageBorderColor(img)
	out := image.NewGray(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			r, g, b, _ := rgbaUnit(img.At(bounds.Min.X+x, bounds.Min.Y+y))
			dist := math.Sqrt((r-bgR)*(r-bgR) + (g-bgG)*(g-bgG) + (b-bgB)*(b-bgB))
			if dist > threshold {
				out.SetGray(x, y, color.Gray{Y: 255})
			}
		}
	}
	return out
}

func averageBorderColor(img image.Image) (float64, float64, float64) {
	bounds := img.Bounds()
	step := max(1, min(bounds.Dx(), bounds.Dy())/128)
	var rSum, gSum, bSum float64
	var samples float64
	var allR, allG, allB float64
	var allSamples float64
	add := func(x, y int) {
		r, g, b, _ := rgbaUnit(img.At(x, y))
		allR += r
		allG += g
		allB += b
		allSamples++
		if luminanceUnit(r, g, b) > 0.85 {
			rSum += r
			gSum += g
			bSum += b
			samples++
		}
	}
	for x := bounds.Min.X; x < bounds.Max.X; x += step {
		add(x, bounds.Min.Y)
		add(x, bounds.Max.Y-1)
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		add(bounds.Min.X, y)
		add(bounds.Max.X-1, y)
	}
	if samples == 0 && allSamples > 0 {
		return allR / allSamples, allG / allSamples, allB / allSamples
	}
	if samples == 0 {
		return 1, 1, 1
	}
	return rSum / samples, gSum / samples, bSum / samples
}

func luminanceUnit(r, g, b float64) float64 {
	return 0.299*r + 0.587*g + 0.114*b
}

func rgbaUnit(c color.Color) (float64, float64, float64, float64) {
	r, g, b, a := c.RGBA()
	return float64(r) / 65535, float64(g) / 65535, float64(b) / 65535, float64(a) / 65535
}
