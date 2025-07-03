package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/mohemohe/claudeway/internal/config"
	"github.com/mohemohe/claudeway/internal/docker"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start the claudeway container and enter it",
	Long: `Start a Docker container with the current directory mounted and enter it interactively.
If the container is already running, it will exec into it instead.`,
	RunE: runUp,
}

func init() {
	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create Docker manager
	manager, err := docker.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create docker manager: %w", err)
	}

	// Check if container is already running
	running, err := manager.IsContainerRunning(ctx)
	if err != nil {
		return fmt.Errorf("failed to check container status: %w", err)
	}

	if !running {
		// Check if container exists but stopped
		exists, err := manager.ContainerExists(ctx)
		if err != nil {
			return fmt.Errorf("failed to check if container exists: %w", err)
		}

		if exists {
			// Remove the stopped container
			if err := manager.StopAndRemoveContainer(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to remove existing container: %v\n", err)
			}
		}

		// Build Docker image if needed
		fmt.Println("Checking Docker image...")
		if err := docker.BuildDockerImage(); err != nil {
			return fmt.Errorf("failed to build Docker image: %w", err)
		}

		// Create and start container
		fmt.Println("Starting container...")
		if err := manager.CreateAndStartContainer(ctx, cfg); err != nil {
			return fmt.Errorf("failed to start container: %w", err)
		}
	}

	// Exec into the container
	fmt.Printf("Entering container %s...\n", manager.GetContainerName())
	if err := manager.ExecInteractive(ctx, []string{"/bin/bash"}); err != nil {
		return fmt.Errorf("failed to exec into container: %w", err)
	}

	return nil
}