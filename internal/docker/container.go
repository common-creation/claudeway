package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/mohemohe/claudeway/internal/config"
	"github.com/mohemohe/claudeway/internal/utils"
)

const ImageName = "claudeway:latest"

type Manager struct {
	client       *client.Client
	containerName string
	workDir      string
}

func NewManager() (*Manager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	containerName := fmt.Sprintf("claudeway-%s", utils.HashPath(workDir))

	return &Manager{
		client:        cli,
		containerName: containerName,
		workDir:       workDir,
	}, nil
}

func (m *Manager) ContainerExists(ctx context.Context) (bool, error) {
	containers, err := m.client.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("name", m.containerName)),
	})
	if err != nil {
		return false, err
	}

	return len(containers) > 0, nil
}

func (m *Manager) IsContainerRunning(ctx context.Context) (bool, error) {
	containers, err := m.client.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(filters.Arg("name", m.containerName)),
	})
	if err != nil {
		return false, err
	}

	return len(containers) > 0, nil
}

func (m *Manager) CreateAndStartContainer(ctx context.Context, cfg *config.Config) error {
	// Prepare mounts
	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: m.workDir,
			Target: m.workDir,
		},
	}

	// Add additional bind mounts
	for _, bind := range cfg.Bind {
		if strings.HasPrefix(bind, "#") {
			continue
		}
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: bind,
			Target: bind,
		})
	}

	// Add copy mounts as read-only under /host
	for _, copy := range cfg.Copy {
		if strings.HasPrefix(copy, "#") {
			continue
		}
		// Expand ~ to home directory
		expandedPath := copy
		if strings.HasPrefix(copy, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			expandedPath = filepath.Join(home, copy[2:])
		}

		// Get absolute path
		absPath, err := filepath.Abs(expandedPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for %s: %w", copy, err)
		}

		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   absPath,
			Target:   filepath.Join("/host", absPath),
			ReadOnly: true,
		})
	}

	// Get environment variables
	env := os.Environ()

	// Prepare init commands
	initCommands := []string{}
	for _, cmd := range cfg.Init {
		if !strings.HasPrefix(cmd, "#") {
			initCommands = append(initCommands, cmd)
		}
	}

	// Create container config
	containerConfig := &container.Config{
		Image:        ImageName,
		Env:          env,
		WorkingDir:   m.workDir,
		Tty:          true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		OpenStdin:    true,
		StdinOnce:    false,
	}

	// Add init commands and copy files as environment variables
	if len(initCommands) > 0 {
		containerConfig.Env = append(containerConfig.Env, fmt.Sprintf("CLAUDEWAY_INIT=%s", strings.Join(initCommands, ";")))
	}

	copyFiles := []string{}
	for _, copy := range cfg.Copy {
		if !strings.HasPrefix(copy, "#") {
			copyFiles = append(copyFiles, copy)
		}
	}
	if len(copyFiles) > 0 {
		containerConfig.Env = append(containerConfig.Env, fmt.Sprintf("CLAUDEWAY_COPY=%s", strings.Join(copyFiles, ";")))
	}

	hostConfig := &container.HostConfig{
		Mounts: mounts,
	}

	// Create container
	resp, err := m.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, m.containerName)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := m.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return nil
}

func (m *Manager) StopAndRemoveContainer(ctx context.Context) error {
	// Stop container
	if err := m.client.ContainerStop(ctx, m.containerName, nil); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Remove container
	if err := m.client.ContainerRemove(ctx, m.containerName, types.ContainerRemoveOptions{}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	return nil
}

func (m *Manager) ExecInteractive(ctx context.Context, cmd []string) error {
	if len(cmd) == 0 {
		cmd = []string{"/bin/bash"}
	}

	execConfig := types.ExecConfig{
		Cmd:          cmd,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
	}

	execResp, err := m.client.ContainerExecCreate(ctx, m.containerName, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec: %w", err)
	}

	resp, err := m.client.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{
		Tty: true,
	})
	if err != nil {
		return fmt.Errorf("failed to attach to exec: %w", err)
	}
	defer resp.Close()

	// Set terminal to raw mode
	oldState, err := SetRawTerminal(os.Stdin.Fd())
	if err != nil {
		return fmt.Errorf("failed to set raw terminal: %w", err)
	}
	defer RestoreTerminal(os.Stdin.Fd(), oldState)

	// Handle resize
	resizeTty := func() {
		size, err := GetTerminalSize()
		if err == nil {
			m.client.ContainerExecResize(ctx, execResp.ID, types.ResizeOptions{
				Height: uint(size.Height),
				Width:  uint(size.Width),
			})
		}
	}
	resizeTty()

	// Start goroutines for input/output
	errCh := make(chan error, 2)

	go func() {
		_, err := io.Copy(resp.Conn, os.Stdin)
		errCh <- err
	}()

	go func() {
		_, err := io.Copy(os.Stdout, resp.Reader)
		errCh <- err
	}()

	// Wait for either goroutine to finish
	err = <-errCh
	return err
}

func (m *Manager) GetContainerName() string {
	return m.containerName
}