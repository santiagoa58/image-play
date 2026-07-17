package mathutil

import "math"

// Range describes an inclusive numeric interval.
//
// Input ranges must be ordered from Min to Max. Output ranges may be reversed
// to map larger inputs to smaller outputs.
type Range struct {
	Min float64
	Max float64
}

// ScaleLinear maps value from input to output using linear interpolation.
//
// Values outside input are clamped to its endpoints. If input is invalid or
// has no width, ScaleLinear returns output.Min.
func ScaleLinear(value float64, input, output Range) float64 {
	ratio, ok := linearRatio(value, input)
	if !ok {
		return output.Min
	}
	return interpolate(output, ratio)
}

// ScaleLog maps value from input to output using logarithmic interpolation.
//
// Logarithmic scaling gives more of the output range to smaller input values,
// which is useful for skewed data such as word frequencies. Values outside
// input are clamped. Input.Min must be greater than -1 because ScaleLog uses
// math.Log1p. If input is invalid or has no width, ScaleLog returns output.Min.
func ScaleLog(value float64, input, output Range) float64 {
	if input.Min <= -1 || !validInputRange(input) || math.IsNaN(value) {
		return output.Min
	}

	value = min(max(value, input.Min), input.Max)

	logMin := math.Log1p(input.Min)
	logMax := math.Log1p(input.Max)
	if logMax <= logMin {
		return output.Min
	}

	ratio := (math.Log1p(value) - logMin) / (logMax - logMin)
	return interpolate(output, ratio)
}

func linearRatio(value float64, input Range) (float64, bool) {
	if !validInputRange(input) || math.IsNaN(value) {
		return 0, false
	}

	value = min(max(value, input.Min), input.Max)
	return (value - input.Min) / (input.Max - input.Min), true
}

func validInputRange(input Range) bool {
	return !math.IsNaN(input.Min) &&
		!math.IsNaN(input.Max) &&
		!math.IsInf(input.Min, 0) &&
		!math.IsInf(input.Max, 0) &&
		input.Max > input.Min
}

func interpolate(output Range, ratio float64) float64 {
	return output.Min + ratio*(output.Max-output.Min)
}
