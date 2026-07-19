package main

import (
	"fmt"
	"os"

	zenchannel "github.com/xhanio/zen/pkg/components/cmd/zen-channel"
)

func main() {
	rootCmd := zenchannel.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
