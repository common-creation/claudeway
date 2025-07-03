#!/bin/bash
# Test script to verify claudeway environment

echo "Testing claudeway development environment..."
echo ""

# Check if we're in a claudeway container
if [ -f /.dockerenv ]; then
    echo "✓ Running inside Docker container"
else
    echo "✗ Not running inside Docker container"
fi

# Check asdf
if [ -f /opt/asdf/asdf.sh ]; then
    echo "✓ asdf is installed"
    . /opt/asdf/asdf.sh
else
    echo "✗ asdf not found"
fi

# Check Go
if command -v go &> /dev/null; then
    echo "✓ Go is installed: $(go version)"
else
    echo "✗ Go is not installed"
fi

# Check Docker socket
if [ -S /var/run/docker.sock ]; then
    echo "✓ Docker socket is mounted"
else
    echo "✗ Docker socket not found"
fi

# Check Claude config
if [ -f ~/.claude.json ]; then
    echo "✓ Claude config is mounted"
else
    echo "✗ Claude config not found"
fi

echo ""
echo "Environment setup complete!"