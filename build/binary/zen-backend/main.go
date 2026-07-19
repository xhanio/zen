package main

import (
	"fmt"
	"os"

	zenbackend "github.com/xhanio/zen/pkg/components/cmd/zen-backend"
)

func main() {
	rootCmd := zenbackend.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
