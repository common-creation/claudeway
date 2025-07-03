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
	"github.com/mohemohe/claudeway/internal/assets"
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

	// Use embedded Dockerfile content
	dockerfileContent := assets.DockerfileContent

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

	// Use embedded entrypoint script content
	entrypointContent := assets.EntrypointContent

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