package textutil

import (
	"bufio"
	"fmt"
	"os"
)

const (
	initialScanBufferSize = 64 * 1024
	maxScanLineSize       = 4 * 1024 * 1024
)

// ProcessLines streams path line by line and calls fn for each line.
func ProcessLines(path string, fn func(line string) error) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file %q: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, initialScanBufferSize), maxScanLineSize)

	for scanner.Scan() {
		line := scanner.Text()
		if err := fn(line); err != nil {
			return fmt.Errorf("process line %q: %w", line, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan file %q: %w", path, err)
	}

	return nil
}
