package main

import (
	"os"

	"github.com/taiga533/gh-pr-md/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
