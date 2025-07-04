package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

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
	SilenceUsage: true,
	SilenceErrors: true,
}

func init() {
	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
	// Wrap the main logic to handle errors
	if err := runUpInternal(cmd, args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return nil
}

func runUpInternal(cmd *cobra.Command, args []string) error {
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

		// Wait for initialization to complete
		fmt.Println("Waiting for container initialization...")
		if err := manager.WaitForInitialization(ctx); err != nil {
			// If initialization failed, stop and remove the container
			fmt.Fprintf(os.Stderr, "Initialization failed: %v\n", err)
			fmt.Println("Cleaning up failed container...")
			if cleanupErr := manager.StopAndRemoveContainer(ctx); cleanupErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to clean up container: %v\n", cleanupErr)
			}
			return err
		}
	}

	// Try to exec into the container
	fmt.Printf("Entering container %s...\n", manager.GetContainerName())
	if err := manager.ExecInteractive(ctx, []string{"/bin/bash", "-l"}); err != nil {
		// If interactive exec fails (e.g., in non-TTY environments), provide alternative instructions
		if strings.Contains(err.Error(), "raw terminal") || strings.Contains(err.Error(), "operation not supported") {
			fmt.Println("\nInteractive shell is not available in this environment.")
			fmt.Printf("Container '%s' is running successfully.\n", manager.GetContainerName())
			fmt.Println("\nYou can connect to the container using:")
			fmt.Printf("  docker exec -it %s /bin/bash -l\n", manager.GetContainerName())
			fmt.Println("\nOr use claudeway exec command:")
			fmt.Println("  ./claudeway exec /bin/bash")
			return nil
		}
		return fmt.Errorf("failed to exec into container: %w", err)
	}

	return nil
}