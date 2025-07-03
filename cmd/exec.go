package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/mohemohe/claudeway/internal/docker"
)

var execCmd = &cobra.Command{
	Use:   "exec [command...]",
	Short: "Execute a command in the running claudeway container",
	Long: `Execute a command in the running claudeway container for the current directory.
If no command is specified, it will open an interactive bash shell.`,
	RunE: runExec,
}

func init() {
	rootCmd.AddCommand(execCmd)
}

func runExec(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create Docker manager
	manager, err := docker.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create docker manager: %w", err)
	}

	// Check if container is running
	running, err := manager.IsContainerRunning(ctx)
	if err != nil {
		return fmt.Errorf("failed to check container status: %w", err)
	}

	if !running {
		return fmt.Errorf("no running container found for the current directory. Use 'claudeway up' to start one")
	}

	// Exec into the container
	if err := manager.ExecInteractive(ctx, args); err != nil {
		return fmt.Errorf("failed to exec into container: %w", err)
	}

	return nil
}