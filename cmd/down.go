package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/common-creation/claudeway/internal/docker"
)

var downCmd = &cobra.Command{
	Use:           "down",
	Short:         "Stop and remove the claudeway container",
	Long:          `Stop the running claudeway container for the current directory and remove it.`,
	RunE:          runDown,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.AddCommand(downCmd)
}

func runDown(cmd *cobra.Command, args []string) error {
	if err := runDownInternal(cmd, args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return nil
}

func runDownInternal(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create Docker manager
	manager, err := docker.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create docker manager: %w", err)
	}

	// Check if container exists
	exists, err := manager.ContainerExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check container status: %w", err)
	}

	if !exists {
		fmt.Println("No container found for the current directory")
		return nil
	}

	// Stop and remove container
	fmt.Printf("Stopping container %s...\n", manager.GetContainerName())
	if err := manager.StopAndRemoveContainer(ctx); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	fmt.Println("Container stopped and removed")
	return nil
}