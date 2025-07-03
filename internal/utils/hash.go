package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
)

func HashPath(path string) string {
	// Get absolute path to ensure consistency
	absPath, err := filepath.Abs(path)
	if err != nil {
		// Fallback to using the provided path
		absPath = path
	}

	// Create SHA256 hash
	hash := sha256.Sum256([]byte(absPath))
	
	// Return first 8 characters of hex representation
	return hex.EncodeToString(hash[:])[:8]
}