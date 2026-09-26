package textutil

import (
	"fmt"
	"strings"
	"unicode"
)

// CountWords counts normalized words in a text file and returns their
// frequencies. Stop words are excluded.
func CountWords(txtPath string) (WordCounts, error) {
	counts := make(map[string]int)

	err := ProcessLines(txtPath, func(line string) error {
		countWordsInLine(line, counts)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("count words in %q: %w", txtPath, err)
	}

	if len(counts) == 0 {
		return nil, fmt.Errorf("text file %q contains no words", txtPath)
	}

	result := make(WordCounts, 0, len(counts))
	for word, count := range counts {
		result = append(result, WordCount{
			Word:  word,
			Count: count,
		})
	}

	return result, nil
}

func countWordsInLine(line string, counts map[string]int) {
	var word strings.Builder

	for _, r := range line {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			word.WriteRune(unicode.ToLower(r))
			continue
		}
		processWord(&word, counts)
	}

	processWord(&word, counts)
}

func processWord(word *strings.Builder, counts map[string]int) {
	if word.Len() == 0 {
		return
	}

	value := word.String()
	word.Reset()

	if IsStopWord(value) {
		return
	}

	counts[value]++
}
