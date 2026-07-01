package textutils

import (
	"bufio"
	"container/heap"
	"fmt"
	"os"
	"strings"
	"unicode"
)

const (
	initialScanBufferSize = 64 * 1024       // 64 KiB allocated up front
	maxScanLineSize       = 4 * 1024 * 1024 // 4 MiB max allowed line
)

func CountWords(txtPath string) (WordHeap, error) {
	txtFile, err := os.Open(txtPath)
	if err != nil {
		return nil, fmt.Errorf("open text file %q: %w", txtPath, err)
	}
	defer txtFile.Close()

	counts := make(map[string]int)
	scanner := bufio.NewScanner(txtFile)
	scanner.Buffer(make([]byte, initialScanBufferSize), maxScanLineSize)

	for scanner.Scan() {
		// Count words in the current line of text
		countWordsUTF8(scanner.Text(), counts)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan text file %q: %w", txtPath, err)
	}

	if len(counts) == 0 {
		return nil, fmt.Errorf("text file %q has no words", txtPath)
	}

	return wordMapToHeap(counts), nil
}

func countWordsUTF8(s string, counts map[string]int) {
	var word strings.Builder

	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			word.WriteRune(unicode.ToLower(r))
			continue
		}
		flushWord(&word, counts)
	}
	flushWord(&word, counts)
}

func flushWord(word *strings.Builder, counts map[string]int) {
	if word.Len() == 0 {
		return
	}
	counts[word.String()]++
	word.Reset()
}

func wordMapToHeap(wordMap map[string]int) WordHeap {
	wHeap := make(WordHeap, 0, len(wordMap))

	for word, count := range wordMap {
		wHeap = append(wHeap, WordCount{
			Word:  word,
			Count: count,
		})
	}

	heap.Init(&wHeap)
	return wHeap
}
