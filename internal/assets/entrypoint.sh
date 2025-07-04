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
            
            # Ensure home directory has correct ownership
            chown "$HOST_UID:$HOST_GID" "/home/$HOST_USER"
            chmod 755 "/home/$HOST_USER"
            
            # Setup asdf for the user
            echo '. /opt/asdf/asdf.sh' >> "/home/$HOST_USER/.bashrc"
            echo '. /opt/asdf/completions/asdf.bash' >> "/home/$HOST_USER/.bashrc"
            chown "$HOST_UID:$HOST_GID" "/home/$HOST_USER/.bashrc"
            
            # Create .tool-versions in user's home if it exists in root
            if [ -f "/root/.tool-versions" ]; then
                cp "/root/.tool-versions" "/home/$HOST_USER/.tool-versions"
                chown "$HOST_UID:$HOST_GID" "/home/$HOST_USER/.tool-versions"
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
        # For tilde expansion, we need to use the original host user's home
        if [ -n "$HOST_USER" ]; then
            # Check common home directory patterns
            if [ -d "/host/Users/$HOST_USER" ]; then
                # macOS pattern
                echo "/Users/$HOST_USER/${path:2}"
            elif [ -d "/host/home/$HOST_USER" ]; then
                # Linux pattern
                echo "/home/$HOST_USER/${path:2}"
            else
                # Fallback to current HOME
                echo "${HOME}/${path:2}"
            fi
        else
            echo "${HOME}/${path:2}"
        fi
    else
        echo "$path"
    fi
}

# Function to expand destination path inside container
expand_dest_path() {
    local path="$1"
    if [[ "$path" == "~/"* ]]; then
        # In container, always use /home/$HOST_USER for tilde expansion
        if [ -n "$HOST_USER" ]; then
            echo "/home/$HOST_USER/${path:2}"
        else
            echo "${HOME}/${path:2}"
        fi
    else
        echo "$path"
    fi
}

# Copy files specified in CLAUDEWAY_COPY
if [ -n "$CLAUDEWAY_COPY" ]; then
    echo "Copying specified files..."
    IFS=';' read -ra COPY_FILES <<< "$CLAUDEWAY_COPY"
    for file in "${COPY_FILES[@]}"; do
        # Expand source path
        src_expanded=$(expand_path "$file")
        
        # Get absolute source path
        if [[ "$src_expanded" = /* ]]; then
            src_abs_path="$src_expanded"
        else
            src_abs_path="$(pwd)/$src_expanded"
        fi
        
        # Source path in /host
        src_path="/host$src_abs_path"
        
        # Expand destination path
        dest_expanded=$(expand_dest_path "$file")
        
        # Get absolute destination path
        if [[ "$dest_expanded" = /* ]]; then
            dest_abs_path="$dest_expanded"
        else
            dest_abs_path="$(pwd)/$dest_expanded"
        fi
        
        # Create parent directory if needed
        parent_dir=$(dirname "$dest_abs_path")
        if [ ! -d "$parent_dir" ]; then
            mkdir -p "$parent_dir"
        fi
        
        # Copy the file/directory
        if [ -e "$src_path" ]; then
            if [ -d "$src_path" ]; then
                cp -r "$src_path" "$dest_abs_path"
                echo "  Copied directory: $file -> $dest_abs_path"
            else
                cp "$src_path" "$dest_abs_path"
                echo "  Copied file: $file -> $dest_abs_path"
            fi
            
            # Change ownership to the user if HOST_USER is set
            if [ -n "$HOST_USER" ] && [ -n "$HOST_UID" ] && [ -n "$HOST_GID" ]; then
                chown -R "$HOST_UID:$HOST_GID" "$dest_abs_path"
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

# Keep the container running
tail -f /dev/null