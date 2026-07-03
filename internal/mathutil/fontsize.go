package mathutil

import "math"

// CalculateFontSize maps word frequency to a font size using a log scale.
// This prevents one extremely frequent word from dominating the entire cloud.
func CalculateFontSize(count int, maxCount, maxFontSize, minFontSize float64) float64 {
	if maxCount == 0 {
		return minFontSize
	}
	normalized := math.Log(1+float64(count)) / math.Log(1+maxCount)
	return minFontSize + (maxFontSize-minFontSize)*normalized
}
