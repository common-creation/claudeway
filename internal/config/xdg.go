package config

import (
	"os"
	"path/filepath"
	"runtime"
)

func GetConfigDir() string {
	// Check XDG_CONFIG_HOME first
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return xdgConfigHome
	}

	// Fallback based on OS
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Last resort fallback
		return "."
	}

	switch runtime.GOOS {
	case "windows":
		// On Windows, use %APPDATA%
		if appData := os.Getenv("APPDATA"); appData != "" {
			return appData
		}
		return filepath.Join(homeDir, "AppData", "Roaming")
	case "darwin":
		// On macOS, use ~/Library/Application Support
		return filepath.Join(homeDir, "Library", "Application Support")
	default:
		// On Linux and other Unix-like systems, use ~/.config
		return filepath.Join(homeDir, ".config")
	}
}