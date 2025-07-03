package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/mohemohe/claudeway/internal/config"
)

type BuildOptions struct {
	NoCache bool
}

func BuildImage(ctx context.Context) error {
	return BuildImageWithOptions(ctx, BuildOptions{})
}

func BuildImageWithOptions(ctx context.Context, options BuildOptions) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	defer cli.Close()

	// Check if image already exists (skip if no-cache is enabled)
	if !options.NoCache {
		images, err := cli.ImageList(ctx, types.ImageListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list images: %w", err)
		}

		for _, img := range images {
			for _, tag := range img.RepoTags {
				if tag == ImageName {
					// Image already exists
					return nil
				}
			}
		}
	}

	fmt.Println("Building Docker image...")

	// Create build context
	buildContext, err := createBuildContext()
	if err != nil {
		return fmt.Errorf("failed to create build context: %w", err)
	}

	// Build options
	buildOptions := types.ImageBuildOptions{
		Context:    buildContext,
		Dockerfile: "Dockerfile",
		Tags:       []string{ImageName},
		Remove:     true,
		NoCache:    options.NoCache,
	}

	// Build the image
	resp, err := cli.ImageBuild(ctx, buildContext, buildOptions)
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}
	defer resp.Body.Close()

	// Read build output
	decoder := json.NewDecoder(resp.Body)
	for {
		var message struct {
			Stream string `json:"stream"`
			Error  string `json:"error"`
		}

		if err := decoder.Decode(&message); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode build output: %w", err)
		}

		if message.Error != "" {
			return fmt.Errorf("build error: %s", message.Error)
		}

		if message.Stream != "" {
			fmt.Print(message.Stream)
		}
	}

	fmt.Println("Docker image built successfully")
	return nil
}

func createBuildContext() (io.Reader, error) {
	// Check for external Docker assets first
	configDir := config.GetConfigDir()
	libDir := filepath.Join(configDir, "claudeway", "lib")
	dockerfilePath := filepath.Join(libDir, "Dockerfile")
	entrypointPath := filepath.Join(libDir, "entrypoint.sh")

	// Check if external files exist
	useExternalFiles := false
	if _, err := os.Stat(dockerfilePath); err == nil {
		if _, err := os.Stat(entrypointPath); err == nil {
			useExternalFiles = true
			fmt.Println("Using Docker assets from:", libDir)
		}
	}

	if useExternalFiles {
		return createBuildContextFromFiles(dockerfilePath, entrypointPath)
	}

	// Fall back to embedded content
	fmt.Println("Using embedded Docker assets (run 'claudeway init --global' to create customizable assets)")
	return createBuildContextEmbedded()
}

func createBuildContextFromFiles(dockerfilePath, entrypointPath string) (io.Reader, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	defer tw.Close()

	// Add Dockerfile from file
	if err := addFileToTar(tw, dockerfilePath, "Dockerfile"); err != nil {
		return nil, fmt.Errorf("failed to add Dockerfile: %w", err)
	}

	// Add entrypoint script from file
	if err := addFileToTar(tw, entrypointPath, "entrypoint.sh"); err != nil {
		return nil, fmt.Errorf("failed to add entrypoint.sh: %w", err)
	}

	return &buf, nil
}

func createBuildContextEmbedded() (io.Reader, error) {
	// Create tar archive
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	defer tw.Close()

	// Embed Dockerfile content
	dockerfileContent := `FROM ubuntu:22.04

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

	// Add Dockerfile
	header := &tar.Header{
		Name: "Dockerfile",
		Mode: 0644,
		Size: int64(len(dockerfileContent)),
	}
	if err := tw.WriteHeader(header); err != nil {
		return nil, fmt.Errorf("failed to write Dockerfile header: %w", err)
	}
	if _, err := tw.Write([]byte(dockerfileContent)); err != nil {
		return nil, fmt.Errorf("failed to write Dockerfile content: %w", err)
	}

	// Embed entrypoint script content
	entrypointContent := `#!/bin/bash
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

	// Add entrypoint script
	scriptHeader := &tar.Header{
		Name: "entrypoint.sh",
		Mode: 0755,
		Size: int64(len(entrypointContent)),
	}
	if err := tw.WriteHeader(scriptHeader); err != nil {
		return nil, fmt.Errorf("failed to write entrypoint header: %w", err)
	}
	if _, err := tw.Write([]byte(entrypointContent)); err != nil {
		return nil, fmt.Errorf("failed to write entrypoint content: %w", err)
	}

	return &buf, nil
}

func addFileToTar(tw *tar.Writer, sourcePath, destPath string) error {
	file, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	header := &tar.Header{
		Name:    destPath,
		Mode:    int64(stat.Mode()),
		Size:    stat.Size(),
		ModTime: stat.ModTime(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tw, file)
	return err
}

func init() {
	// Override the BuildDockerImage function in utils.go
	BuildDockerImage = func() error {
		return BuildImage(context.Background())
	}
}