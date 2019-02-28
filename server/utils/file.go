package utils

import (
	"os"
	"path/filepath"
)

// GetPluginRootPath ToDo
func GetPluginRootPath() string {
	ex, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(ex)))
}
