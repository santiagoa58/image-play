package textutil

import (
	"container/heap"
	"fmt"
	"strings"
	"unicode"
)

// CountWords counts words in a text file using streaming (low memory usage)
// and returns a heap of word frequencies (most frequent first).
func CountWords(txtPath string) (WordHeap, error) {
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

	return buildWordHeap(counts), nil
}

// countWordsInLine extracts words using Unicode rules and updates the count map.
// It is designed to be allocation-efficient for large files.
func countWordsInLine(line string, counts map[string]int) {
	var word strings.Builder

	for _, r := range line {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			word.WriteRune(unicode.ToLower(r))
			continue
		}
		// Non-word character → flush current word
		processWord(&word, counts)
	}
	// Flush any remaining word at the end of the line
	processWord(&word, counts)
}

func processWord(word *strings.Builder, counts map[string]int) {
	if word.Len() == 0 {
		return
	}

	w := word.String()
	word.Reset()

	if IsStopWord(w) {
		return
	}

	counts[w]++
}

func buildWordHeap(wordMap map[string]int) WordHeap {
	h := make(WordHeap, 0, len(wordMap))

	for word, count := range wordMap {
		h = append(h, WordCount{
			Word:  word,
			Count: count,
		})
	}

	heap.Init(&h)
	return h
}
