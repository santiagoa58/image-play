package textutil

import (
	"fmt"
	"sort"
	"strings"
)

type WordCount struct {
	Word  string
	Count int
}

type WordHeap []WordCount

func (h WordHeap) Len() int { return len(h) }

func (h WordHeap) Less(i, j int) bool {
	if h[i].Count == h[j].Count {
		return h[i].Word < h[j].Word
	}
	return h[i].Count > h[j].Count
}

func (h WordHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *WordHeap) Push(x any) {
	*h = append(*h, x.(WordCount))
}

func (h *WordHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

// ToSortedSlice returns a new slice sorted by frequency (highest first).
func (h WordHeap) ToSortedSlice() []WordCount {
	words := make([]WordCount, len(h))
	copy(words, h)

	sort.Slice(words, func(i, j int) bool {
		return words[i].Count > words[j].Count
	})
	return words
}

// String returns a debug-friendly string of the top words.
func (h WordHeap) String() string {
	s := h.ToSortedSlice()
	var b strings.Builder

	for _, wc := range s {
		fmt.Fprintf(&b, "%s: %d\n", wc.Word, wc.Count)
	}

	return b.String()
}
