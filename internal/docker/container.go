package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/common-creation/claudeway/internal/config"
	"github.com/common-creation/claudeway/internal/utils"
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
		
		// Parse source and target from bind string (format: "source:target" or just "path")
		var sourcePath, targetPath string
		parts := strings.SplitN(bind, ":", 2)
		if len(parts) == 2 {
			sourcePath = parts[0]
			targetPath = parts[1]
		} else {
			sourcePath = bind
			targetPath = bind
		}
		
		// Expand ~ to home directory for source path
		if strings.HasPrefix(sourcePath, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			sourcePath = filepath.Join(home, sourcePath[2:])
		}
		
		// Get absolute path for source
		absSourcePath, err := filepath.Abs(sourcePath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for %s: %w", sourcePath, err)
		}
		
		// Resolve symlinks for source path
		resolvedSourcePath, err := filepath.EvalSymlinks(absSourcePath)
		if err != nil {
			// If symlink evaluation fails, use the absolute path
			resolvedSourcePath = absSourcePath
		}
		
		// For target path, expand ~ to container's home directory
		if strings.HasPrefix(targetPath, "~/") {
			// Check if we have host user info
			hostUser := os.Getenv("USER")
			if hostUser != "" && os.Getuid() >= 0 {
				// Use host user's home directory
				targetPath = filepath.Join("/home", hostUser, targetPath[2:])
			} else {
				// Fallback to root
				targetPath = filepath.Join("/root", targetPath[2:])
			}
		}
		
		// Get absolute path for target
		absTargetPath, err := filepath.Abs(targetPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for %s: %w", targetPath, err)
		}
		
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: resolvedSourcePath,
			Target: absTargetPath,
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

	// Get environment variables - only copy essential ones
	env := []string{
		"PATH=/opt/asdf/shims:/opt/asdf/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"ASDF_DIR=/opt/asdf",
		"ASDF_DATA_DIR=/opt/asdf",
		"TMPDIR=/tmp",
		"TERM=xterm-256color",
		"LANG=C.UTF-8",
	}

	// Add host user information
	if uid := os.Getuid(); uid >= 0 {
		env = append(env, fmt.Sprintf("HOST_UID=%d", uid))
		
		if gid := os.Getgid(); gid >= 0 {
			env = append(env, fmt.Sprintf("HOST_GID=%d", gid))
		}
		if user := os.Getenv("USER"); user != "" {
			env = append(env, fmt.Sprintf("HOST_USER=%s", user))
			// Set HOME for the container user
			env = append(env, fmt.Sprintf("HOME=/home/%s", user))
		} else {
			// Fallback to root if no user info
			env = append(env, "HOME=/root")
		}
	} else {
		// No UID info, use root
		env = append(env, "HOME=/root")
	}

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
		cmd = []string{"/bin/bash", "-l"}
	}

	// First, ensure user is set up by running the setup_user function
	setupScript := `
if [ -n "$HOST_UID" ] && [ -n "$HOST_GID" ] && [ -n "$HOST_USER" ]; then
    if ! id -u "$HOST_USER" >/dev/null 2>&1; then
        echo "Creating user $HOST_USER with UID=$HOST_UID GID=$HOST_GID..."
        if ! getent group "$HOST_GID" >/dev/null 2>&1; then
            groupadd -g "$HOST_GID" "$HOST_USER" 2>/dev/null || true
        fi
        useradd -u "$HOST_UID" -g "$HOST_GID" -m -d "/home/$HOST_USER" -s /bin/bash "$HOST_USER" 2>/dev/null || true
        echo '. /opt/asdf/asdf.sh' >> "/home/$HOST_USER/.bashrc"
        echo '. /opt/asdf/completions/asdf.bash' >> "/home/$HOST_USER/.bashrc"
        if [ -f "/root/.tool-versions" ]; then
            cp "/root/.tool-versions" "/home/$HOST_USER/.tool-versions"
            chown "$HOST_USER:$HOST_GID" "/home/$HOST_USER/.tool-versions"
        fi
        echo "$HOST_USER ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/$HOST_USER
        chmod 0440 /etc/sudoers.d/$HOST_USER
    fi
fi
`
	setupConfig := types.ExecConfig{
		Cmd:          []string{"/bin/bash", "-c", setupScript},
		AttachStdout: true,
		AttachStderr: true,
		Env: []string{
			fmt.Sprintf("HOST_UID=%d", os.Getuid()),
			fmt.Sprintf("HOST_GID=%d", os.Getgid()),
			fmt.Sprintf("HOST_USER=%s", os.Getenv("USER")),
		},
	}
	
	setupResp, err := m.client.ContainerExecCreate(ctx, m.containerName, setupConfig)
	if err == nil {
		m.client.ContainerExecStart(ctx, setupResp.ID, types.ExecStartCheck{})
		// Wait a moment for user setup to complete
		time.Sleep(500 * time.Millisecond)
	}

	// Check if we have host user info
	hostUser := os.Getenv("USER")
	hostUID := os.Getuid()
	
	// If we have host user info, use sudo to switch user
	if hostUser != "" && hostUID >= 0 {
		// Prepend sudo command to run as the host user
		sudoCmd := []string{"sudo", "-u", hostUser, "-E", "-H"}
		cmd = append(sudoCmd, cmd...)
	}

	execConfig := types.ExecConfig{
		Cmd:          cmd,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Env: []string{
			fmt.Sprintf("HOST_UID=%d", os.Getuid()),
			fmt.Sprintf("HOST_GID=%d", os.Getgid()),
			fmt.Sprintf("HOST_USER=%s", os.Getenv("USER")),
		},
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

// WaitForInitialization waits for container initialization to complete
func (m *Manager) WaitForInitialization(ctx context.Context) error {
	fmt.Println("Container logs:")
	
	// Start following logs immediately
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: false,
	}

	reader, err := m.client.ContainerLogs(ctx, m.containerName, options)
	if err != nil {
		return fmt.Errorf("failed to get container logs: %w", err)
	}
	defer reader.Close()
	
	// Channel to signal completion
	done := make(chan error, 1)
	
	// Start log processing goroutine
	go func() {
		// Simple byte-by-byte reading with line accumulation
		buffer := make([]byte, 4096)
		var accumulated strings.Builder
		
		for {
			n, err := reader.Read(buffer)
			if n > 0 {
				// Print to stdout
				os.Stdout.Write(buffer[:n])
				
				// Accumulate for pattern matching
				accumulated.WriteString(string(buffer[:n]))
				content := accumulated.String()
				
				// Check for completion
				if strings.Contains(content, "Claudeway initialization complete.") {
					done <- nil
					return
				}
				
				// Check for failure
				if strings.Contains(content, "Claudeway initialization failed.") {
					done <- fmt.Errorf("initialization failed: one or more init commands failed")
					return
				}
			}
			
			if err != nil {
				// Any error (including EOF) means log stream ended
				done <- fmt.Errorf("log stream ended unexpectedly")
				return
			}
		}
	}()
	
	// Monitor container status
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case err := <-done:
			return err
			
		case <-ticker.C:
			// Check if container is still running
			inspect, err := m.client.ContainerInspect(ctx, m.containerName)
			if err != nil {
				return fmt.Errorf("failed to inspect container: %w", err)
			}
			
			if !inspect.State.Running {
				// Container stopped - this means initialization failed
				return fmt.Errorf("container stopped unexpectedly (initialization failed)")
			}
			
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// waitForInitializationFile uses file-based approach to check for initialization completion
func (m *Manager) waitForInitializationFile(ctx context.Context, timeout time.Duration) error {
	start := time.Now()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Check if initialization is complete by looking for the marker file
			execConfig := types.ExecConfig{
				Cmd:          []string{"test", "-f", "/tmp/.claudeway_init_complete"},
				AttachStdout: false,
				AttachStderr: false,
			}

			execResp, err := m.client.ContainerExecCreate(ctx, m.containerName, execConfig)
			if err != nil {
				return fmt.Errorf("failed to create exec for init check: %w", err)
			}

			inspectResp, err := m.client.ContainerExecInspect(ctx, execResp.ID)
			if err != nil {
				return fmt.Errorf("failed to inspect exec: %w", err)
			}

			// Start the exec
			if err := m.client.ContainerExecStart(ctx, execResp.ID, types.ExecStartCheck{}); err != nil {
				return fmt.Errorf("failed to start exec: %w", err)
			}

			// Wait for the exec to complete
			for {
				inspectResp, err = m.client.ContainerExecInspect(ctx, execResp.ID)
				if err != nil {
					return fmt.Errorf("failed to inspect exec: %w", err)
				}
				if !inspectResp.Running {
					break
				}
				time.Sleep(50 * time.Millisecond)
			}

			// Exit code 0 means the file exists
			if inspectResp.ExitCode == 0 {
				return nil
			}

			// Check timeout
			if time.Since(start) > timeout {
				return fmt.Errorf("initialization timeout after %v", timeout)
			}
		}
	}
}
