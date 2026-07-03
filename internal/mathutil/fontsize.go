package mathutil

import "math"

// CalculateFontSize maps word frequency to a logarithmic font size range.
// Counts at or below zero, or a non-positive maxCount, return minFontSize.
func CalculateFontSize(count int, maxCount, maxFontSize, minFontSize float64) float64 {
	if maxCount <= 0 || count <= 0 {
		return minFontSize
	}

	clampedCount := math.Min(float64(count), maxCount)
	// Log scaling compresses high-frequency words so one word does not dominate.
	normalized := math.Log1p(clampedCount) / math.Log1p(maxCount)
	return minFontSize + (maxFontSize-minFontSize)*normalized
}
