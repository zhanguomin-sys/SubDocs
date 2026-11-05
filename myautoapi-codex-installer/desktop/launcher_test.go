package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func payloadFixture(t *testing.T, names []string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "中文 launcher's tool.exe")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	// Exercise ZIPs with a native executable prepended, rather than ordinary ZIPs.
	_, _ = file.Write([]byte("MZ-prefixed-executable"))
	archive := zip.NewWriter(file)
	for _, name := range names {
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = entry.Write([]byte("fixture"))
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPrefixedPayloadRejectsInvalidEntriesAndCleansTemporaryFiles(t *testing.T) {
	for _, names := range [][]string{
		{"start.ps1", "../escape.ps1"},
		{"start.ps1", "start.ps1"},
		{"install.ps1"},
	} {
		executable := payloadFixture(t, names)
		root := t.TempDir()
		t.Setenv("TMPDIR", root)
		t.Setenv("TMP", root)
		t.Setenv("TEMP", root)
		_, err := extractPayload(executable)
		if err == nil {
			t.Fatalf("invalid payload accepted: %v", names)
		}
		entries, err := os.ReadDir(root)
		if err != nil || len(entries) != 0 {
			t.Fatalf("failed extraction left temporary files: %v, %v", entries, err)
		}
	}
}

func TestChildGetsInteractiveInputAndExitCodeAndTemporaryFilesAreCleaned(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("此测试使用 POSIX 假 PowerShell；Windows 成品另需实机双击验证。")
	}
	root := t.TempDir()
	command := filepath.Join(root, "fake powershell")
	log := filepath.Join(root, "args")
	t.Setenv("LAUNCHER_TEST_LOG", log)
	script := `#!/bin/sh
printf '%s\n' "$@" > "$LAUNCHER_TEST_LOG"
test -f "$6" || exit 91
IFS= read -r value
printf 'input:%s\n' "$value"
printf 'fixture-error\n' >&2
exit 7
`
	if err := os.WriteFile(command, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code, err := runWindows(payloadFixture(t, []string{"start.ps1", "install.ps1"}), command, strings.NewReader("synthetic-input\n"), &stdout, &stderr)
	if err != nil || code != 7 || stdout.String() != "input:synthetic-input\n" || stderr.String() != "fixture-error\n" {
		t.Fatalf("child input, output or exit code lost: %d, %v, %q, %q", code, err, stdout.String(), stderr.String())
	}
	arguments, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(arguments)), "\n")
	if len(lines) != 6 || strings.Join(lines[:5], " ") != "-NoLogo -NoProfile -ExecutionPolicy Bypass -File" {
		t.Fatalf("unexpected arguments: %q", arguments)
	}
	if _, err := os.Stat(filepath.Dir(lines[len(lines)-1])); !os.IsNotExist(err) {
		t.Fatal("installer temporary directory was not removed")
	}
}
