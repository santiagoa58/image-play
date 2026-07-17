package mathutil

import (
	"math"
	"testing"
)

func TestScaleLinear(t *testing.T) {
	tests := []struct {
		name   string
		value  float64
		input  Range
		output Range
		want   float64
	}{
		{
			name:   "maps input minimum to output minimum",
			value:  0,
			input:  Range{Min: 0, Max: 100},
			output: Range{Min: 10, Max: 20},
			want:   10,
		},
		{
			name:   "maps midpoint linearly",
			value:  50,
			input:  Range{Min: 0, Max: 100},
			output: Range{Min: 10, Max: 20},
			want:   15,
		},
		{
			name:   "maps input maximum to output maximum",
			value:  100,
			input:  Range{Min: 0, Max: 100},
			output: Range{Min: 10, Max: 20},
			want:   20,
		},
		{
			name:   "clamps below input minimum",
			value:  -1,
			input:  Range{Min: 0, Max: 100},
			output: Range{Min: 10, Max: 20},
			want:   10,
		},
		{
			name:   "clamps above input maximum",
			value:  101,
			input:  Range{Min: 0, Max: 100},
			output: Range{Min: 10, Max: 20},
			want:   20,
		},
		{
			name:   "supports descending output range",
			value:  25,
			input:  Range{Min: 0, Max: 100},
			output: Range{Min: 20, Max: 0},
			want:   15,
		},
		{
			name:   "zero-width input returns output minimum",
			value:  10,
			input:  Range{Min: 5, Max: 5},
			output: Range{Min: 10, Max: 20},
			want:   10,
		},
		{
			name:   "reversed input returns output minimum",
			value:  10,
			input:  Range{Min: 20, Max: 10},
			output: Range{Min: 10, Max: 20},
			want:   10,
		},
		{
			name:   "NaN value returns output minimum",
			value:  math.NaN(),
			input:  Range{Min: 0, Max: 100},
			output: Range{Min: 10, Max: 20},
			want:   10,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ScaleLinear(test.value, test.input, test.output)
			assertClose(t, got, test.want)
		})
	}
}

func TestScaleLog(t *testing.T) {
	input := Range{Min: 1, Max: 100}
	output := Range{Min: 12, Max: 72}

	tests := []struct {
		name  string
		value float64
		want  float64
	}{
		{
			name:  "maps input minimum to output minimum",
			value: 1,
			want:  12,
		},
		{
			name:  "maps input maximum to output maximum",
			value: 100,
			want:  72,
		},
		{
			name:  "clamps below input minimum",
			value: 0,
			want:  12,
		},
		{
			name:  "clamps above input maximum",
			value: 200,
			want:  72,
		},
		{
			name:  "maps intermediate value logarithmically",
			value: 9,
			want:  12 + 60*(math.Log1p(9)-math.Log1p(1))/(math.Log1p(100)-math.Log1p(1)),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ScaleLog(test.value, input, output)
			assertClose(t, got, test.want)
		})
	}
}

func TestScaleLogInvalidInputReturnsOutputMinimum(t *testing.T) {
	output := Range{Min: 12, Max: 72}

	for _, input := range []Range{
		{Min: 1, Max: 1},
		{Min: 2, Max: 1},
		{Min: -1, Max: 100},
		{Min: math.NaN(), Max: 100},
		{Min: 0, Max: math.Inf(1)},
	} {
		if got := ScaleLog(10, input, output); got != output.Min {
			t.Errorf(
				"ScaleLog(10, %#v, %#v) = %v, want %v",
				input,
				output,
				got,
				output.Min,
			)
		}
	}
}

func TestScaleLogIsStrictlyIncreasing(t *testing.T) {
	input := Range{Min: 1, Max: 250}
	output := Range{Min: 8, Max: 64}
	previous := ScaleLog(input.Min, input, output)

	for value := input.Min + 1; value <= input.Max; value++ {
		current := ScaleLog(value, input, output)
		if current <= previous {
			t.Fatalf(
				"ScaleLog(%v) = %v, want greater than previous value %v",
				value,
				current,
				previous,
			)
		}
		if current < output.Min || current > output.Max {
			t.Fatalf(
				"ScaleLog(%v) = %v, want within [%v,%v]",
				value,
				current,
				output.Min,
				output.Max,
			)
		}
		previous = current
	}
}

func assertClose(t *testing.T, got, want float64) {
	t.Helper()

	const tolerance = 1e-9
	if math.Abs(got-want) > tolerance {
		t.Errorf("got %v, want %v", got, want)
	}
}
