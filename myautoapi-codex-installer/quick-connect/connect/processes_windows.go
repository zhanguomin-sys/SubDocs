//go:build windows

package connect

import (
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// StopClients closes the codex and chatgpt processes of this login session,
// waiting up to five seconds in total, like install/codex.ps1.
func StopClients() (result StopResult) {
	processes, err := listProcesses()
	if err != nil {
		return StopResult{Left: true}
	}
	targets, result := PlanStop(processes, windows.GetCurrentProcessId())
	var waiting []windows.Handle
	for _, target := range targets {
		handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, target.PID)
		if err != nil {
			if err != windows.ERROR_INVALID_PARAMETER { // ERROR_INVALID_PARAMETER: already gone
				result.Left = true
			}
			continue
		}
		// The PID could have been reused since the snapshot.
		if clientName(imageName(handle)) == "" {
			windows.CloseHandle(handle)
			continue
		}
		if err := windows.TerminateProcess(handle, 1); err != nil && !exited(handle, 0) {
			result.Left = true
			windows.CloseHandle(handle)
			continue
		}
		waiting = append(waiting, handle)
	}
	deadline := time.Now().Add(5 * time.Second)
	for _, handle := range waiting {
		remaining := time.Until(deadline).Milliseconds()
		if remaining < 0 {
			remaining = 0
		}
		if !exited(handle, uint32(remaining)) {
			result.Left = true
		}
		windows.CloseHandle(handle)
	}
	return result
}

func exited(handle windows.Handle, milliseconds uint32) bool {
	event, err := windows.WaitForSingleObject(handle, milliseconds)
	return err == nil && event == windows.WAIT_OBJECT_0
}

func imageName(handle windows.Handle) string {
	path := imagePath(handle)
	if path == "" {
		return ""
	}
	return filepath.Base(path)
}

func imagePath(handle windows.Handle) string {
	buffer := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(len(buffer))
	if windows.QueryFullProcessImageName(handle, 0, &buffer[0], &size) != nil {
		return ""
	}
	return windows.UTF16ToString(buffer[:size])
}

func listProcesses() ([]Process, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snapshot)
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	var processes []Process
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		var session uint32
		if windows.ProcessIdToSessionId(entry.ProcessID, &session) != nil {
			session = ^uint32(0)
		}
		processes = append(processes, Process{
			PID:     entry.ProcessID,
			PPID:    entry.ParentProcessID,
			Name:    strings.TrimSpace(windows.UTF16ToString(entry.ExeFile[:])),
			Session: session,
		})
	}
	if err != windows.ERROR_NO_MORE_FILES {
		return nil, err
	}
	return processes, nil
}

// StopClaudeDesktop closes Claude Desktop (every claude.exe of this login
// session that runs from its Store package), waiting up to five seconds.
// The Claude Code CLI, also claude.exe, is left alone.
func StopClaudeDesktop() (result ClaudeStopResult) {
	processes, err := listProcesses()
	if err != nil {
		return ClaudeStopResult{Left: true}
	}
	var session uint32
	if windows.ProcessIdToSessionId(windows.GetCurrentProcessId(), &session) != nil {
		return ClaudeStopResult{Left: true}
	}
	var waiting []windows.Handle
	for _, process := range processes {
		if process.Session != session || !strings.EqualFold(process.Name, "claude.exe") {
			continue
		}
		handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, process.PID)
		if err != nil {
			// Gone already is fine. Any other refusal (an elevated Claude,
			// say) may be Claude Desktop still running with the old
			// settings, so the window asks the user to quit it.
			if err != windows.ERROR_INVALID_PARAMETER {
				result.Found, result.Left = true, true
			}
			continue
		}
		if !IsClaudeDesktopProcess(imagePath(handle)) {
			windows.CloseHandle(handle)
			continue
		}
		result.Found = true
		if err := windows.TerminateProcess(handle, 1); err != nil && !exited(handle, 0) {
			result.Left = true
			windows.CloseHandle(handle)
			continue
		}
		waiting = append(waiting, handle)
	}
	deadline := time.Now().Add(5 * time.Second)
	for _, handle := range waiting {
		remaining := time.Until(deadline).Milliseconds()
		if remaining < 0 {
			remaining = 0
		}
		if !exited(handle, uint32(remaining)) {
			result.Left = true
		}
		windows.CloseHandle(handle)
	}
	return result
}
