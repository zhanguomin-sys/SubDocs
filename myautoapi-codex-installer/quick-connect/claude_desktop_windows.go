package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const claudeWindowsPackage = "Claude_pzs8sxrjxfjjc"

func hasClaudeDesktopPackage() bool {
	family, _ := windows.UTF16PtrFromString(claudeWindowsPackage)
	getPackages := windows.NewLazySystemDLL("kernel32.dll").NewProc("GetPackagesByPackageFamily")
	if getPackages.Find() != nil {
		return false
	}
	var count, chars uint32
	result, _, _ := getPackages.Call(uintptr(unsafe.Pointer(family)),
		uintptr(unsafe.Pointer(&count)), 0, uintptr(unsafe.Pointer(&chars)), 0, 0)
	return result == uintptr(windows.ERROR_INSUFFICIENT_BUFFER) && count > 0
}

func findClaudeDesktop() (claudeDesktopFiles, bool) {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		return claudeDesktopFiles{}, false
	}
	installed := hasClaudeDesktopPackage()
	for _, folder := range []string{"AnthropicClaude", "Claude"} {
		matches, _ := filepath.Glob(filepath.Join(local, folder, "app-*", "Claude.exe"))
		for _, exe := range matches {
			if info, err := os.Stat(exe); err == nil && info.Mode().IsRegular() {
				installed = true
			}
		}
	}
	if !installed {
		return claudeDesktopFiles{}, false
	}
	root := filepath.Join(local, "Claude-3p")
	// Current releases migrate the older roaming directory to LocalAppData.
	// If it still exists by itself, update it in place without masking migration.
	if roam := os.Getenv("APPDATA"); roam != "" {
		legacy := filepath.Join(roam, "Claude-3p")
		if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
			if info, err := os.Stat(legacy); err == nil && info.IsDir() {
				root = legacy
			}
		}
	}
	return claudeDesktopPaths(root), true
}

func claudeDesktopManaged() (bool, error) {
	for _, hive := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
		key, err := registry.OpenKey(hive, "SOFTWARE\\Policies\\Claude", registry.QUERY_VALUE)
		if errors.Is(err, registry.ErrNotExist) {
			continue
		}
		if err != nil {
			return false, err
		}
		names, err := key.ReadValueNames(-1)
		key.Close()
		if err != nil {
			return false, err
		}
		for _, name := range names {
			name = strings.ToLower(name)
			if strings.HasPrefix(name, "inference") || name == "bootstrapurl" ||
				name == "deploymentmode" || name == "disabledeploymentmodechooser" {
				return true, nil
			}
		}
	}
	return false, nil
}

func (a *App) configureClaude(input string) (string, error) {
	gateway := a.gateway
	return connectClaudeModels(a.ctx, gateway.client, gateway.origin, input, func(key string, models []string) (string, error) {
		files, installed := findClaudeDesktop()
		if !installed {
			return configureClaudeCode(key, models)
		}
		managed, err := claudeDesktopManaged()
		if err != nil {
			return "", errors.New("无法读取 Claude 桌面版的设备策略，未改动配置。")
		}
		if managed {
			return "", errors.New("Claude 桌面版的接入由设备策略管理，未改动配置。请联系设备管理员。")
		}
		message, err := writeClaudeDesktop(files, api, key, models)
		if err != nil {
			return "", err
		}
		cliMessage, err := configureClaudeCode(key, models)
		if err != nil {
			return message + "\nClaude Code 命令行配置未完成：" + err.Error(), nil
		}
		return message + "\n\n" + cliMessage, nil
	})
}
