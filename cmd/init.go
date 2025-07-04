package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/common-creation/claudeway/internal/assets"
	"github.com/common-creation/claudeway/internal/config"
)

var (
	globalFlag bool
)

var initCmd = &cobra.Command{
	Use:           "init",
	Short:         "Initialize claudeway configuration",
	Long:          `Create a default claudeway.yaml configuration file or initialize global configuration with Docker assets.`,
	RunE:          runInit,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	initCmd.Flags().BoolVar(&globalFlag, "global", false, "Initialize global configuration and Docker assets")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	if err := runInitInternal(cmd, args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return nil
}

func runInitInternal(cmd *cobra.Command, args []string) error {
	if globalFlag {
		return initGlobalConfig()
	}
	
	// Check if claudeway.yaml already exists
	if _, err := os.Stat("claudeway.yaml"); err == nil {
		return fmt.Errorf("claudeway.yaml already exists in the current directory")
	}

	// Create default config
	if err := config.CreateDefaultConfig("claudeway.yaml"); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	fmt.Println("Created claudeway.yaml in the current directory")
	return nil
}

func initGlobalConfig() error {
	configDir := config.GetConfigDir()
	claudewayDir := filepath.Join(configDir, "claudeway")
	libDir := filepath.Join(claudewayDir, "lib")

	// Create directories
	if err := os.MkdirAll(libDir, 0755); err != nil {
		return fmt.Errorf("failed to create lib directory: %w", err)
	}

	// Create or confirm overwrite for Docker assets
	dockerfilePath := filepath.Join(libDir, "Dockerfile")
	if shouldWriteFile(dockerfilePath, "Dockerfile") {
		if err := os.WriteFile(dockerfilePath, []byte(assets.DockerfileContent), 0644); err != nil {
			return fmt.Errorf("failed to write Dockerfile: %w", err)
		}
		fmt.Printf("Created %s\n", dockerfilePath)
	}

	entrypointPath := filepath.Join(libDir, "entrypoint.sh")
	if shouldWriteFile(entrypointPath, "entrypoint.sh") {
		if err := os.WriteFile(entrypointPath, []byte(assets.EntrypointContent), 0755); err != nil {
			return fmt.Errorf("failed to write entrypoint.sh: %w", err)
		}
		fmt.Printf("Created %s\n", entrypointPath)
	}


	// Create global config if it doesn't exist
	globalConfigPath := filepath.Join(claudewayDir, "claudeway.yaml")
	if _, err := os.Stat(globalConfigPath); os.IsNotExist(err) {
		if err := config.CreateDefaultConfig(globalConfigPath); err != nil {
			return fmt.Errorf("failed to create global config: %w", err)
		}
		fmt.Printf("Created %s\n", globalConfigPath)
	}

	fmt.Println("\nGlobal configuration initialized successfully!")
	fmt.Printf("Configuration directory: %s\n", claudewayDir)
	fmt.Println("\nYou can now customize the Docker assets in the lib/ directory.")
	return nil
}

func shouldWriteFile(path, name string) bool {
	if _, err := os.Stat(path); err == nil {
		// File exists, ask for confirmation
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("%s already exists. Overwrite? (y/N): ", name)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		return response == "y" || response == "yes"
	}
	return true
}

