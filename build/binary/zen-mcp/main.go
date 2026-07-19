package main

import (
	"fmt"
	"os"

	zenmcp "github.com/xhanio/zen/pkg/components/cmd/zen-mcp"
)

func main() {
	rootCmd := zenmcp.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
