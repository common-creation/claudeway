package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/common-creation/claudeway/internal/docker"
	"github.com/spf13/cobra"
)

var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Manage Docker images",
	Long:  `Manage Docker images for claudeway containers`,
}

var buildCmd = &cobra.Command{
	Use:           "build",
	Short:         "Build Docker image",
	Long:          `Build or rebuild the Docker image for claudeway containers`,
	RunE:          runBuild,
	SilenceUsage:  true,
	SilenceErrors: true,
}

var noCache bool

func init() {
	rootCmd.AddCommand(imageCmd)
	imageCmd.AddCommand(buildCmd)
	buildCmd.Flags().BoolVar(&noCache, "no-cache", false, "Do not use cache when building the image")
}

func runBuild(cmd *cobra.Command, args []string) error {
	if err := runBuildInternal(cmd, args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return nil
}

func runBuildInternal(cmd *cobra.Command, args []string) error {
	fmt.Println("Building Docker image...")
	
	ctx := context.Background()
	err := docker.BuildImageWithOptions(ctx, docker.BuildOptions{
		NoCache: noCache,
	})
	
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}
	
	fmt.Println("Image built successfully")
	return nil
}