package main

import (
	"archive/zip"
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const terminalScript = `on run argv
    set runnerPath to item 1 of argv
    tell application "Terminal"
        activate
        do script ("/bin/bash " & quoted form of runnerPath)
    end tell
end run`

func main() {
	configureConsole()
	executable, err := os.Executable()
	code := 1
	if err == nil {
		switch runtime.GOOS {
		case "windows":
			powershell := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
			code, err = runWindows(executable, powershell, os.Stdin, os.Stdout, os.Stderr)
		case "darwin":
			resources := filepath.Join(filepath.Dir(executable), "..", "Resources")
			output, runErr := exec.Command("/usr/bin/osascript", "-e", terminalScript, filepath.Join(resources, "run.command")).CombinedOutput()
			if runErr != nil {
				err = fmt.Errorf("无法打开终端：%s", output)
			} else {
				code = 0
			}
		default:
			err = errors.New("此启动器仅用于 Windows 和 Apple 芯片 Mac。")
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "启动失败：", err)
		if runtime.GOOS == "darwin" {
			_ = exec.Command("/usr/bin/osascript", "-e", `on run argv
display alert "无法启动安装工具" message (item 1 of argv) as critical
end run`, err.Error()).Run()
		}
	}
	if runtime.GOOS == "windows" {
		fmt.Print("\n按回车退出……")
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	}
	os.Exit(code)
}

// The distribution appends its payload ZIP to the native launcher. The original
// project scripts remain the only maintained source of installation behavior.
func extractPayload(executable string) (directory string, err error) {
	archive, err := zip.OpenReader(executable)
	if err != nil {
		return "", fmt.Errorf("无法读取内置安装文件：%w", err)
	}
	defer archive.Close()
	directory, err = os.MkdirTemp("", "myautoapi-desktop-")
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(directory)
		}
	}()
	allowed := map[string]bool{"start.ps1": true, "install.ps1": true, "tool.exe": true}
	seen := make(map[string]bool)
	for _, file := range archive.File {
		if !allowed[file.Name] || seen[file.Name] || !file.Mode().IsRegular() || file.UncompressedSize64 > 64<<20 {
			return directory, errors.New("内置安装文件无效，请重新获取完整安装工具。")
		}
		seen[file.Name] = true
		if err = extractFile(file, filepath.Join(directory, file.Name)); err != nil {
			return directory, err
		}
	}
	if !seen["start.ps1"] {
		return directory, errors.New("内置启动脚本缺失。")
	}
	return directory, nil
}

func extractFile(file *zip.File, target string) error {
	source, err := file.Open()
	if err != nil {
		return err
	}
	defer source.Close()
	destination, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(destination, source)
	closeErr := destination.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func runWindows(executable, powershell string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	directory, err := extractPayload(executable)
	if err != nil {
		return 1, err
	}
	defer os.RemoveAll(directory)
	command := exec.Command(powershell, "-NoLogo", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", filepath.Join(directory, "start.ps1"))
	command.Stdin, command.Stdout, command.Stderr = stdin, stdout, stderr
	if err := command.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return exitError.ExitCode(), nil
		}
		return 1, fmt.Errorf("无法启动 Windows PowerShell：%w", err)
	}
	return 0, nil
}
