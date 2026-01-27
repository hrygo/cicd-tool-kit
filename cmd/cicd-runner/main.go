// Package main is the entry point for cicd-runner
package main

import (
	"fmt"
	"os"
	"runtime/debug"
)

func main() {
	// Panic recovery to prevent crashes and provide diagnostic info
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "PANIC: %v\n", r)
			fmt.Fprintf(os.Stderr, "\nStack trace:\n%s\n", debug.Stack())
			os.Exit(2)
		}
	}()

	if err := Execute(); err != nil {
		os.Exit(1)
	}
}
