package main

import (
	"fmt"
	"os"
)

const (
	appName    = "mosaic"
	appVersion = "dev"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
