// Package main is the entry point for the Chronicle CLI.
package main

import (
	"os"

	"github.com/joshua-temple/chronicle/pkg/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
