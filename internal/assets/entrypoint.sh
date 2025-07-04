#!/bin/bash -l
set -e

# Setup asdf for root user
echo '. /opt/asdf/asdf.sh' >> /root/.bashrc
echo '. /opt/asdf/completions/asdf.bash' >> /root/.bashrc

# Also setup asdf in current shell
. /opt/asdf/asdf.sh

# Setup asdf in /etc/profile.d for all users (including sudo)
echo '. /opt/asdf/asdf.sh' > /etc/profile.d/asdf.sh
chmod +x /etc/profile.d/asdf.sh

# Add asdf to sudo secure_path
echo 'Defaults    secure_path="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/snap/bin:/opt/asdf/bin:/opt/asdf/shims"' > /etc/sudoers.d/asdf

# Function to setup user
setup_user() {
    if [ -n "$HOST_UID" ] && [ -n "$HOST_GID" ] && [ -n "$HOST_USER" ]; then
        # Check if user already exists
        if ! id -u "$HOST_USER" >/dev/null 2>&1; then
            echo "Creating user $HOST_USER with UID=$HOST_UID GID=$HOST_GID..."
            
            # Create group if it doesn't exist
            if ! getent group "$HOST_GID" >/dev/null 2>&1; then
                groupadd -g "$HOST_GID" "$HOST_USER" 2>/dev/null || true
            fi
            
            # Create user
            useradd -u "$HOST_UID" -g "$HOST_GID" -m -d "/home/$HOST_USER" -s /bin/bash "$HOST_USER" 2>/dev/null || true
            
            # Setup asdf for the user
            echo '. /opt/asdf/asdf.sh' >> "/home/$HOST_USER/.bashrc"
            echo '. /opt/asdf/completions/asdf.bash' >> "/home/$HOST_USER/.bashrc"
            
            # Create .tool-versions in user's home if it exists in root
            if [ -f "/root/.tool-versions" ]; then
                cp "/root/.tool-versions" "/home/$HOST_USER/.tool-versions"
                chown "$HOST_USER:$HOST_GID" "/home/$HOST_USER/.tool-versions"
            fi
            
            # Add user to sudoers
            echo "$HOST_USER ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/$HOST_USER
            chmod 0440 /etc/sudoers.d/$HOST_USER
        fi
    fi
}

# Setup user
setup_user

# Update HOME for this script if user was created
if [ -n "$HOST_UID" ] && [ -n "$HOST_GID" ] && [ -n "$HOST_USER" ]; then
    export HOME="/home/$HOST_USER"
    export USER="$HOST_USER"
fi

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
        if ! bash -lc "$cmd"; then
            echo "  ERROR: Command failed: $cmd"
            echo "Claudeway initialization failed."
            # Exit with error code so container stops
            exit 1
        fi
    done
fi

# Mark initialization as complete
touch /tmp/.claudeway_init_complete
echo "Claudeway initialization complete."

# Execute the main command
if [ -n "$HOST_UID" ] && [ -n "$HOST_GID" ] && [ -n "$HOST_USER" ]; then
    echo "Switching to user $HOST_USER..."
    # Need to ensure the user can access the working directory
    chown -R "$HOST_USER:$HOST_GID" "$PWD" 2>/dev/null || true
    # Switch to the host user
    exec sudo -u "$HOST_USER" -E -H "$@"
else
    echo "Running as root (no host user info)..."
    # Run as root if no host user info
    exec "$@"
fi