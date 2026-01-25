// Package main is the entry point for the Chronicle CLI.
package main

import (
	"fmt"
	"os"

	"github.com/joshua-temple/chronicle/pkg/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
