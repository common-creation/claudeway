#!/bin/bash
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
exec "$@"