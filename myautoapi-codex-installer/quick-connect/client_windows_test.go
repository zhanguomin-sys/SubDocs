package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCodexTargetsIncludeHeadlessProcessesButExcludeAncestors(t *testing.T) {
	entries := map[uint32]clientProcessEntry{
		5: {4, "assistant.exe"}, 4: {3, "powershell.exe"}, 3: {1, "CoDeX.ExE"},
		2: {1, "ChatGPT.exe"}, 7: {2, "chatgpt.exe"}, 8: {1, "codex.exe"},
		9: {1, "codex.exe"}, 10: {8, "codex-command-runner.exe"}, 11: {1, "claude.exe"},
		12: {1, "codex-windows-sandbox-service.exe"}, 13: {1, "codex-code-mode-host.exe"},
		14: {1, "codex-code-mode-host.exe"}, 15: {9, "codex-command-runner.exe"},
	}
	pids, protected := codexProcessTargets(entries, 5, func(pid uint32) bool { return pid != 9 && pid != 15 },
		func(pid uint32) bool { return pid == 13 })
	slices.Sort(pids)
	if !slices.Equal(pids, []uint32{2, 7, 8, 10, 13}) || !protected {
		t.Fatalf("targets=%v protected=%v", pids, protected)
	}
}

func TestShutdownFixture(t *testing.T) {
	if os.Getenv("MYAUTOAPI_SHUTDOWN_FIXTURE") == "1" {
		time.Sleep(time.Minute)
	}
}

func TestNativeTerminationStopsBackgroundClientsOnly(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	var pids []uint32
	entries := make(map[uint32]clientProcessEntry)
	var done []chan error
	for _, name := range []string{"ChatGPT.exe", "codex.exe", "codex-command-runner.exe", "codex-windows-sandbox-service.exe", "unrelated.exe"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, binary, 0700); err != nil {
			t.Fatal(err)
		}
		cmd := exec.CommandContext(t.Context(), path, "-test.run=^TestShutdownFixture$")
		cmd.Env = append(cmd.Environ(), "MYAUTOAPI_SHUTDOWN_FIXTURE=1")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		wait := make(chan error, 1)
		go func() { wait <- cmd.Wait(); close(wait) }()
		t.Cleanup(func() { _ = cmd.Process.Kill(); <-wait })
		pid := uint32(cmd.Process.Pid)
		pids = append(pids, pid)
		entries[pid] = clientProcessEntry{name: name}
		done = append(done, wait)
	}
	// Only the fixture PIDs are allowed; the running user session is untouched.
	terminateCodexTargets(pids, entries, func(pid uint32) bool { return slices.Contains(pids, pid) })
	for i, wait := range done[:3] {
		select {
		case err := <-wait:
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 1 {
				t.Fatalf("fixture %d: %v", i, err)
			}
		case <-time.After(time.Second):
			t.Fatalf("fixture %d did not exit", i)
		}
	}
	for _, wait := range done[3:] {
		select {
		case err := <-wait:
			t.Fatalf("service or unrelated fixture exited: %v", err)
		default:
		}
	}
}

func TestConnectCodexClosesOnceAfterSuccessfulSave(t *testing.T) {
	for _, saveFails := range []bool{false, true} {
		saved, closed := false, 0
		a := &App{closeClient: func() error {
			if !saved {
				t.Fatal("closed before successful save")
			}
			closed++
			return errors.New("fixture client is still running")
		}}
		reply, err := a.connectCodex("test-key", func(key string) (string, error) {
			if key != "test-key" {
				t.Fatal("connect passed the wrong key")
			}
			if saveFails {
				return "", errors.New("save failed")
			}
			saved = true
			return "fixture-backup", nil
		})
		if saveFails {
			if err == nil || closed != 0 {
				t.Fatal("failed save closed the client or returned success")
			}
			continue
		}
		if err != nil || closed != 1 || !strings.Contains(reply, "未能完全关闭") ||
			!strings.Contains(reply, "fixture-backup") || !strings.Contains(reply, "High") {
			t.Fatalf("unexpected result: %q, %v, close calls=%d", reply, err, closed)
		}
	}
}
