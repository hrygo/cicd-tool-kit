// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	versionShort bool
	versionJSON  bool
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		if versionJSON {
			fmt.Printf(`{"version":"%s","commit":"%s","buildDate":"%s"}`,
				getVersion(), getCommit(), getBuildDate())
			return nil
		}

		if versionShort {
			fmt.Println(getVersion())
			return nil
		}

		fmt.Printf("CICD AI Toolkit Runner\n")
		fmt.Printf("Version: %s\n", getVersion())
		fmt.Printf("Commit: %s\n", getCommit())
		fmt.Printf("Build Date: %s\n", getBuildDate())

		return nil
	},
}

func init() {
	versionCmd.Flags().BoolVar(&versionShort, "short", false,
		"Show only version number")
	versionCmd.Flags().BoolVar(&versionJSON, "json", false,
		"Output version as JSON")
}

func getCommit() string {
	// Will be set by ldflags during build
	return "unknown"
}

func getBuildDate() string {
	// Will be set by ldflags during build
	return "unknown"
}
