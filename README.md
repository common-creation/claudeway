# claudeway

A CLI tool for running AI agents like Claude Code safely in Docker containers with proper directory bindings and environment isolation.

## Features

- Bind mount current directory with the same path inside the container
- Support for additional bind mounts and file copies
- Persistent containers per directory
- Environment variable inheritance
- Ubuntu-based container with asdf for easy runtime management
- XDG Base Directory compliant configuration
- Customizable Docker assets (Dockerfile and entrypoint script)

## Installation

```bash
go install github.com/mohemohe/claudeway@latest
```

## Quick Start

1. Initialize global configuration and Docker assets:

```bash
claudeway init --global
```

This creates customizable Docker assets in `$XDG_CONFIG_HOME/claudeway/lib/`:
- `Dockerfile`: Base image configuration
- `entrypoint.sh`: Container startup script

2. Create a project configuration (optional):

```bash
claudeway init
```

3. Start a container for your project:

```bash
claudeway up
```

## Usage

### Initialize configuration

Create Docker assets and global configuration:

```bash
claudeway init --global
```

Create a project-specific `claudeway.yaml`:

```bash
claudeway init
```

### Start container

Start a container for the current directory and enter it:

```bash
claudeway up
```

### Stop container

Stop and remove the container:

```bash
claudeway down
```

### Execute commands

Execute a command in the running container:

```bash
claudeway exec npm test
```

## Configuration

Configuration files are loaded from:
1. `$XDG_CONFIG_HOME/claudeway/claudeway.yaml` (global)
2. `./claudeway.yaml` (project-specific)

Example `claudeway.yaml`:

```yaml
# Commands to run on container startup
init:
  - npm ci
  - go mod download

# Additional directories to bind mount
bind:
  - /opt/bin

# Files to copy into the container
copy:
  - ~/.zshrc
  - ~/.gitconfig
```

### Configuration Priority

- Global configuration takes precedence for `init` commands
- Both global and local configurations are merged for `bind` and `copy` entries

## How it Works

1. **Container Naming**: Each directory gets a unique container name based on its path hash (`claudeway-XXXX`)
2. **Path Preservation**: The current directory is mounted at the same path inside the container
3. **File Copying**: Files specified in `copy` are mounted read-only under `/host` and copied on startup
4. **Environment**: All environment variables are passed through to the container
5. **Docker Assets**: 
   - First checks `$XDG_CONFIG_HOME/claudeway/lib/` for custom Dockerfile and entrypoint.sh
   - Falls back to embedded assets if not found
   - Run `claudeway init --global` to create customizable assets

## Requirements

- Docker
- Go 1.22+ (for building from source)

## License

[WTFPL](http://www.wtfpl.net/) - Do What The Fuck You Want To Public License