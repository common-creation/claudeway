package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "claudeway",
	Short: "A CLI tool for running AI agents safely in Docker containers",
	Long: `claudeway is a CLI tool that allows you to run AI agents like Claude Code
safely in Docker containers with proper directory bindings and environment isolation.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
}