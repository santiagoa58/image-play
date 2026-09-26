package textmosaic

import (
	"fmt"
	"os"
	"strings"
)

func loadText(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read text file %q: %w", path, err)
	}

	text := strings.TrimSpace(string(data))
	if text == "" {
		return "", fmt.Errorf("text file %q is empty", path)
	}

	return text, nil
}

// normalizeText collapses whitespace so characters flow consistently across
// the rendering grid.
func normalizeText(text string) ([]rune, error) {
	normalized := strings.Join(strings.Fields(text), " ")
	if normalized == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	return []rune(normalized + " "), nil
}
