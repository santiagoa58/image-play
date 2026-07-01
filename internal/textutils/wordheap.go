package textutils

import (
	"container/heap"
	"fmt"
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

// String is useful for debugging the heap's internal slice order.
// It does not print words in fully sorted frequency order.
func (h WordHeap) String() string {
	var b strings.Builder

	for _, wc := range h {
		fmt.Fprintf(&b, "%s: %d\n", wc.Word, wc.Count)
	}

	return b.String()
}

// RankedString returns a string representation of the top N words in the heap, sorted by frequency.
// If limit is 0 or greater than the number of words in the heap, all words are included.
func (h WordHeap) RankedString(limit int) string {
	cpy := append(WordHeap(nil), h...)

	n := cpy.Len()
	if limit > 0 && limit < n {
		n = limit
	}

	var b strings.Builder

	for i := 0; i < n; i++ {
		wc := heap.Pop(&cpy).(WordCount)
		fmt.Fprintf(&b, "%s: %d\n", wc.Word, wc.Count)
	}

	return b.String()
}
