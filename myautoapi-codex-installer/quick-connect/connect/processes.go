package connect

import "strings"

// Process is one entry of the system process list.
type Process struct {
	PID     uint32
	PPID    uint32
	Name    string
	Session uint32
}

// StopResult mirrors Stop-RunningClients in install/codex.ps1.
type StopResult struct {
	ChatGPT bool
	Codex   bool
	// Left means a client was found but at least one could not be closed.
	Left bool
}

// Closed reports that clients were running and all of them were closed.
func (r StopResult) Closed() bool { return (r.ChatGPT || r.Codex) && !r.Left }

// PlanStop picks the codex and chatgpt processes of the current login
// session. Processes the assistant itself runs under are left alone and
// reported as not closed, as the script does.
func PlanStop(processes []Process, self uint32) (targets []Process, result StopResult) {
	byPID := make(map[uint32]Process, len(processes))
	for _, process := range processes {
		byPID[process.PID] = process
	}
	own, found := byPID[self]
	if !found {
		return nil, result
	}
	ancestors := map[uint32]bool{}
	for pid := self; pid != 0 && !ancestors[pid]; {
		ancestors[pid] = true
		process, ok := byPID[pid]
		if !ok {
			break
		}
		pid = process.PPID
	}
	for _, process := range processes {
		if process.Session != own.Session {
			continue
		}
		switch clientName(process.Name) {
		case "chatgpt":
			result.ChatGPT = true
		case "codex":
			result.Codex = true
		default:
			continue
		}
		if ancestors[process.PID] {
			result.Left = true
			continue
		}
		targets = append(targets, process)
	}
	return targets, result
}

// clientName matches Get-Process -Name "codex", "chatgpt": the file name
// without .exe, ignoring case.
func clientName(name string) string {
	lower := strings.ToLower(name)
	lower = strings.TrimSuffix(lower, ".exe")
	if lower == "codex" || lower == "chatgpt" {
		return lower
	}
	return ""
}

// IsClaudeDesktopProcess tells Claude Desktop from the Claude Code CLI: both
// run as claude.exe, but the desktop app runs from its Store package under
// WindowsApps (C:\Program Files\WindowsApps\Claude_<version>_x64__<id>\...).
// Only the desktop app is ever closed.
func IsClaudeDesktopProcess(imagePath string) bool {
	lower := strings.ToLower(strings.ReplaceAll(imagePath, "/", `\`))
	return strings.HasSuffix(lower, `\claude.exe`) && strings.Contains(lower, `\windowsapps\claude_`)
}
