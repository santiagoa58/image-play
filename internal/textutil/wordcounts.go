package textutil

import "sort"

// WordCount pairs a normalized token with its observed frequency.
type WordCount struct {
	Word  string
	Count int
}

// WordCounts is an unordered collection of word frequencies.
type WordCounts []WordCount

// ToSortedSlice returns a copy sorted by descending frequency. Alphabetical
// ordering makes equal-frequency input deterministic before font measurement.
func (counts WordCounts) ToSortedSlice() []WordCount {
	words := append([]WordCount(nil), counts...)

	sort.Slice(words, func(i, j int) bool {
		if words[i].Count == words[j].Count {
			return words[i].Word < words[j].Word
		}
		return words[i].Count > words[j].Count
	})

	return words
}
