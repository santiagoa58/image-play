package util

import "testing"

func TestResolveOutputPath(t *testing.T) {
	testFileName := "couple_tour"
	testExt := ".jpg"
	imgPath := "../testdata/images/" + testFileName + testExt // relative to the test file
	testCases := []struct {
		name     string
		input    string
		output   string
		expected string
	}{
		{
			name:     "explicit filename with extension",
			input:    imgPath,
			output:   "result.png",
			expected: "result.png",
		},
		{
			name:     "no extension",
			input:    imgPath,
			output:   "result",
			expected: "result/" + testFileName + "_mosaic.png",
		},
		{
			name:     "directory with trailing slash",
			input:    imgPath,
			output:   "results/",
			expected: "results/" + testFileName + "_mosaic.png",
		},
		{
			name:     "directory without trailing slash",
			input:    imgPath,
			output:   "results",
			expected: "results/" + testFileName + "_mosaic.png",
		},
		{
			name:     "deep nested directory",
			input:    imgPath,
			output:   "/output/2026/summer/",
			expected: "/output/2026/summer/" + testFileName + "_mosaic.png",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ResolveOutputPath(tc.input, tc.output)
			if result != tc.expected {
				t.Errorf("ResolveOutputPath(%q, %q) = %q; want %q", tc.input, tc.output, result, tc.expected)
			}
		})
	}
}
