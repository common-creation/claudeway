package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/mohemohe/claudeway/internal/config"
)

var (
	globalFlag bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize claudeway configuration",
	Long:  `Create a default claudeway.yaml configuration file or initialize global configuration with Docker assets.`,
	RunE:  runInit,
}

func init() {
	initCmd.Flags().BoolVar(&globalFlag, "global", false, "Initialize global configuration and Docker assets")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
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
		if err := writeDockerfile(dockerfilePath); err != nil {
			return fmt.Errorf("failed to write Dockerfile: %w", err)
		}
		fmt.Printf("Created %s\n", dockerfilePath)
	}

	entrypointPath := filepath.Join(libDir, "entrypoint.sh")
	if shouldWriteFile(entrypointPath, "entrypoint.sh") {
		if err := writeEntrypoint(entrypointPath); err != nil {
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

func writeDockerfile(path string) error {
	content := `FROM ubuntu:22.04

# Set environment variables
ENV DEBIAN_FRONTEND=noninteractive
ENV ASDF_DIR=/opt/asdf
ENV PATH="${ASDF_DIR}/bin:${ASDF_DIR}/shims:${PATH}"

# Install basic dependencies
RUN apt-get update && apt-get install -y \
    curl \
    git \
    build-essential \
    libssl-dev \
    zlib1g-dev \
    libbz2-dev \
    libreadline-dev \
    libsqlite3-dev \
    wget \
    llvm \
    libncurses5-dev \
    libncursesw5-dev \
    xz-utils \
    tk-dev \
    libffi-dev \
    liblzma-dev \
    unzip \
    rsync \
    sudo \
    && rm -rf /var/lib/apt/lists/*

# Install asdf
RUN git clone https://github.com/asdf-vm/asdf.git ${ASDF_DIR} --branch v0.14.0 && \
    echo '. /opt/asdf/asdf.sh' >> /etc/bash.bashrc && \
    echo '. /opt/asdf/completions/asdf.bash' >> /etc/bash.bashrc

# Install commonly used asdf plugins
RUN . ${ASDF_DIR}/asdf.sh && \
    asdf plugin add nodejs && \
    asdf plugin add python && \
    asdf plugin add golang && \
    asdf plugin add ruby && \
    asdf plugin add java

# Create host directory for copy operations
RUN mkdir -p /host

# Copy entrypoint script
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Set working directory
WORKDIR /workspace

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

# Default command
CMD ["/bin/bash"]`

	return os.WriteFile(path, []byte(content), 0644)
}

func writeEntrypoint(path string) error {
	content := `#!/bin/bash
set -e

# Function to expand tilde in paths
expand_path() {
    local path="$1"
    if [[ "$path" == "~/"* ]]; then
        echo "${HOME}/${path:2}"
    else
        echo "$path"
    fi
}

# Copy files specified in CLAUDEWAY_COPY
if [ -n "$CLAUDEWAY_COPY" ]; then
    echo "Copying specified files..."
    IFS=';' read -ra COPY_FILES <<< "$CLAUDEWAY_COPY"
    for file in "${COPY_FILES[@]}"; do
        # Expand path
        expanded_file=$(expand_path "$file")
        
        # Get absolute path
        if [[ "$expanded_file" = /* ]]; then
            abs_path="$expanded_file"
        else
            abs_path="$(pwd)/$expanded_file"
        fi
        
        # Source path in /host
        src_path="/host$abs_path"
        
        # Create parent directory if needed
        parent_dir=$(dirname "$abs_path")
        if [ ! -d "$parent_dir" ]; then
            mkdir -p "$parent_dir"
        fi
        
        # Copy the file/directory
        if [ -e "$src_path" ]; then
            if [ -d "$src_path" ]; then
                cp -r "$src_path" "$abs_path"
                echo "  Copied directory: $file -> $abs_path"
            else
                cp "$src_path" "$abs_path"
                echo "  Copied file: $file -> $abs_path"
            fi
        else
            echo "  Warning: Source not found: $src_path"
        fi
    done
fi

# Run initialization commands
if [ -n "$CLAUDEWAY_INIT" ]; then
    echo "Running initialization commands..."
    IFS=';' read -ra INIT_COMMANDS <<< "$CLAUDEWAY_INIT"
    for cmd in "${INIT_COMMANDS[@]}"; do
        echo "  Running: $cmd"
        bash -c "$cmd" || {
            echo "  Warning: Command failed: $cmd"
        }
    done
fi

# Execute the main command
exec "$@"`

	return os.WriteFile(path, []byte(content), 0755)
}