//go:build windows

package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"unsafe"
)

func desktopDirectory() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	keyName, _ := syscall.UTF16PtrFromString(`Software\Microsoft\Windows\CurrentVersion\Explorer\User Shell Folders`)
	var key syscall.Handle
	if err := syscall.RegOpenKeyEx(syscall.HKEY_CURRENT_USER, keyName, 0, syscall.KEY_QUERY_VALUE, &key); err == nil {
		defer syscall.RegCloseKey(key)
		name, _ := syscall.UTF16PtrFromString("Desktop")
		var valueType, size uint32
		if syscall.RegQueryValueEx(key, name, nil, &valueType, nil, &size) == nil && size > 2 && size <= 32768 {
			buffer := make([]uint16, (size+1)/2)
			if syscall.RegQueryValueEx(key, name, nil, &valueType, (*byte)(unsafe.Pointer(&buffer[0])), &size) == nil {
				path := syscall.UTF16ToString(buffer)
				path = regexp.MustCompile(`%[^%]+%`).ReplaceAllStringFunc(path, func(variable string) string {
					return os.Getenv(strings.Trim(variable, "%"))
				})
				if filepath.IsAbs(path) {
					return filepath.Clean(path), nil
				}
			}
		}
	}
	return filepath.Join(home, "Desktop"), nil
}
