// Package main is the entry point for the cicd-runner CLI.
package main

import (
	"os"
)

func main() {
	if err := Execute(); err != nil {
		os.Exit(1)
	}
}
