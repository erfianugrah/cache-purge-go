package main

import (
	"fmt"
	"os"

	"cfpurge/cmd"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, buildTime)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
