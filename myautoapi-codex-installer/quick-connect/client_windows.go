package main

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type clientProcessEntry struct {
	parent uint32
	name   string
}

func clientProcessSnapshot() (map[uint32]clientProcessEntry, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snapshot)
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	entries := make(map[uint32]clientProcessEntry)
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		entries[entry.ProcessID] = clientProcessEntry{entry.ParentProcessID, windows.UTF16ToString(entry.ExeFile[:])}
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return nil, err
	}
	return entries, nil
}

func isCodexProcess(name string) bool {
	return strings.EqualFold(name, "ChatGPT.exe") || strings.EqualFold(name, "codex.exe")
}

func isCodexHelper(name string) bool {
	switch strings.ToLower(name) {
	case "codex-command-runner.exe", "codex-code-mode-host.exe", "codex-computer-use-swift.exe":
		return true
	default:
		return false
	}
}

func processImage(handle windows.Handle) string {
	buf := make([]uint16, 32768)
	size := uint32(len(buf))
	if windows.QueryFullProcessImageName(handle, 0, &buf[0], &size) != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:size])
}

func isInstalledCodexHelper(path string) bool {
	if !isCodexHelper(filepath.Base(path)) {
		return false
	}
	for _, root := range []string{
		filepath.Join(os.Getenv("LOCALAPPDATA"), "OpenAI", "Codex", "bin"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "OpenAI", "Codex", "runtimes"),
	} {
		rel, err := filepath.Rel(root, path)
		if err == nil && filepath.IsLocal(rel) {
			return true
		}
	}
	// Bundled helpers can survive the desktop process that launched them.
	packageDir := filepath.Dir(filepath.Dir(filepath.Dir(path)))
	appName := strings.ToLower(filepath.Base(packageDir))
	return strings.HasPrefix(appName, "openai.codex_") &&
		strings.HasSuffix(appName, "_2p2nqsd0c76g0") &&
		strings.EqualFold(filepath.Base(filepath.Dir(packageDir)), "WindowsApps") &&
		strings.EqualFold(filepath.Dir(path), filepath.Join(packageDir, "app", "resources"))
}

func installedCodexHelper(pid uint32) bool {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)
	return isInstalledCodexHelper(processImage(handle))
}

func codexProcessTargets(entries map[uint32]clientProcessEntry, self uint32, inSession, installedHelper func(uint32) bool) (targets []uint32, protected bool) {
	ancestors := make(map[uint32]bool)
	for pid := self; pid != 0 && !ancestors[pid]; pid = entries[pid].parent {
		ancestors[pid] = true
	}

	depths := make(map[uint32]int)
	for pid, entry := range entries {
		if (!isCodexProcess(entry.name) && !isCodexHelper(entry.name)) || !inSession(pid) {
			continue
		}
		owned := isCodexProcess(entry.name)
		seen := map[uint32]bool{pid: true}
		for parent := entry.parent; parent != 0 && !seen[parent]; parent = entries[parent].parent {
			seen[parent] = true
			depths[pid]++
			if isCodexProcess(entries[parent].name) && inSession(parent) {
				owned = true
			}
		}
		if !owned && !installedHelper(pid) {
			continue
		}
		if ancestors[pid] {
			protected = true
		} else {
			targets = append(targets, pid)
		}
	}
	// Stop helpers first so their parent links are still present on retry.
	sort.Slice(targets, func(i, j int) bool { return depths[targets[i]] > depths[targets[j]] })
	return targets, protected
}

func terminateCodexTargets(pids []uint32, entries map[uint32]clientProcessEntry, inSession func(uint32) bool) {
	var handles []windows.Handle
	defer func() {
		for _, h := range handles {
			windows.CloseHandle(h)
		}
	}()
	for _, pid := range pids {
		expected := entries[pid].name
		if !isCodexProcess(expected) && !isCodexHelper(expected) {
			continue
		}
		h, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, pid)
		if err != nil {
			continue
		}
		// Recheck the name and session after opening the process so a recycled
		// PID for an unrelated application cannot be stopped.
		if !inSession(pid) || !strings.EqualFold(filepath.Base(processImage(h)), expected) {
			windows.CloseHandle(h)
			continue
		}
		if err := windows.TerminateProcess(h, 1); err != nil {
			windows.CloseHandle(h)
			continue
		}
		handles = append(handles, h)
	}
	deadline := time.Now().Add(5 * time.Second)
	for _, h := range handles {
		if remaining := time.Until(deadline); remaining > 0 {
			_, _ = windows.WaitForSingleObject(h, uint32(remaining.Milliseconds()))
		}
	}
}

var errStartedFromCodex = errors.New("the assistant was started from Codex")
var errCodexStillRunning = errors.New("Codex background processes are still running")

// Wait for the current interactive session to clear. The Windows sandbox
// service runs outside this session and manages its own lifecycle.
func closeOpenAIClients() error {
	self := windows.GetCurrentProcessId()
	var session uint32
	if err := windows.ProcessIdToSessionId(self, &session); err != nil {
		return err
	}
	inSession := func(pid uint32) bool {
		var id uint32
		return windows.ProcessIdToSessionId(pid, &id) == nil && id == session
	}
	deadline := time.Now().Add(8 * time.Second)
	for {
		entries, err := clientProcessSnapshot()
		if err != nil {
			return err
		}
		targets, protected := codexProcessTargets(entries, self, inSession, installedCodexHelper)
		if protected {
			return errStartedFromCodex
		}
		if len(targets) == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return errCodexStillRunning
		}
		terminateCodexTargets(targets, entries, inSession)
		time.Sleep(200 * time.Millisecond)
	}
}
