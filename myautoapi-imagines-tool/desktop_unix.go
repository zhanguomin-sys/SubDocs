//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func desktopDirectory() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "linux" {
		config := os.Getenv("XDG_CONFIG_HOME")
		if config == "" {
			config = filepath.Join(home, ".config")
		}
		if content, readErr := os.ReadFile(filepath.Join(config, "user-dirs.dirs")); readErr == nil {
			for _, line := range strings.Split(string(content), "\n") {
				if value, ok := strings.CutPrefix(strings.TrimSpace(line), "XDG_DESKTOP_DIR="); ok {
					if decoded, unquoteErr := strconv.Unquote(value); unquoteErr == nil {
						decoded = strings.ReplaceAll(decoded, "${HOME}", home)
						decoded = strings.ReplaceAll(decoded, "$HOME", home)
						if filepath.IsAbs(decoded) {
							return filepath.Clean(decoded), nil
						}
					}
				}
			}
		}
	}
	return filepath.Join(home, "Desktop"), nil
}
