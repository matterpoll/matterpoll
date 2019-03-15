package utils

import (
	"os"
	"path/filepath"
)

// GetPluginRootPath returns the bundle path
func GetPluginRootPath() string {
	ex, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(ex)))
}
