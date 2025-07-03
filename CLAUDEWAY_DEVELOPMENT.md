# Developing claudeway with claudeway

This project supports developing claudeway within its own container environment - the ultimate dogfooding experience!

## Getting Started

1. Ensure Docker is running on your system
2. Build and install claudeway:
   ```bash
   go build
   go install
   ```

3. Initialize and start the development container:
   ```bash
   claudeway init --global  # If not already done
   claudeway up
   ```

4. Inside the container, the environment will set up automatically:
   - Go 1.23.5 will be installed via asdf
   - Development tools (gopls, delve) will be installed
   - Dependencies will be downloaded

5. After setup completes, you can develop as usual:
   ```bash
   # Run tests
   go test ./...
   
   # Build claudeway
   go build
   
   # Run claudeway commands
   ./claudeway --help
   ```

## Features

- **Docker-in-Docker**: The Docker socket is mounted, allowing you to manage containers from within
- **Claude Integration**: Your Claude configuration is automatically mounted
- **Persistent Development**: All changes are reflected in your host filesystem
- **Automated Setup**: Go and development tools are installed automatically

## Configuration

The `claudeway.yaml` file in the project root defines:
- Volume mounts for Docker socket
- Bind mounts for Claude configuration
- Initialization commands for setting up Go
- Environment variables for Go development

## Troubleshooting

If Go is not available after startup:
1. Exit the container (`exit`)
2. Stop it (`claudeway down`)
3. Start fresh (`claudeway up`)
4. The initialization scripts will run again

The environment setup may take a few minutes on first run as it downloads and installs Go.