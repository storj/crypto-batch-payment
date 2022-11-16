package main

import (
	"fmt"
	"os"
)

func main() {
	rootCmd := newRootCommand()
	if err := rootCmd.Execute(); err != nil {
		// usage errors exit with 1
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}
