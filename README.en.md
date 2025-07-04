<p align="center">
  <img width="320" src="./assets/claudeway_logo.png" />
</p>
<h2 align="center">
  claudeway - The Ultimate Defense: Securing Claude Code Execution
</h2>

----

A CLI tool for safely running AI coding agents like Claude Code inside a Docker container.

[日本語版はこちら](README.md)

## How it works

It starts a Docker container by bind-mounting the current directory.  
At this time, it inherits the host's environment variables (dotenv friendly) with a few exceptions, initializes the container according to the configuration file, and executes with the same UID and GID as the host.  
This prevents catastrophic damage to the host, even if the AI coding agent were to execute `rm -rf --no-preserve-root`.

## Installation

```bash
go install github.com/mohemohe/claudeway@latest
```

## Usage

### Initial Setup

```bash
# Initialize global settings and Docker assets
claudeway init --global

# Initialize project-specific settings
claudeway init
```

### Basic Usage

```bash
# Start the container and enter an interactive shell
claudeway up

# Enter an interactive shell of an already running container
claudeway exec

# Stop and remove the container
claudeway down
```

### Other Commands

```bash
# Build the Docker image
claudeway image build

# Build the image without cache
claudeway image build --no-cache
```

## Configuration File

Format of `claudeway.yaml`:

```yaml
# Volume mount settings
bind:
  - /var/run/docker.sock:/var/run/docker.sock  # Docker socket
  - ~/.claude.json:~/.claude.json               # Claude settings
  - ~/.claude:~/.claude                         # Claude directory

# Files to copy into the container
copy:
  - ~/.gitconfig                                 # Git settings
  - ~/.ssh                                       # SSH keys

# Initialization commands (executed on container start)
init:
  - curl -Ls get.docker.com | sh                # Install Docker
  - asdf plugin add nodejs                       # Add Node.js plugin
  - asdf install nodejs 22.17.0                  # Install Node.js
  - asdf global nodejs 22.17.0                   # Set default version
  - npm i -g @anthropic-ai/claude-code          # Install Claude Code
```

For a real-world example, please check [`claudeway.yaml`](./claudeway.yaml).
