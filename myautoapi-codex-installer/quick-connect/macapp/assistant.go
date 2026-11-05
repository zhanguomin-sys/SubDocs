//go:build darwin || linux

// Command macapp is the launcher inside the Mac .app of the quick-connect
// assistant. It has no window: a double click opens Terminal on run.command,
// which reads a key from the clipboard; a myautoapi://import link is checked
// with the same rules as the Windows window (connect.ParseImportLink) and its
// key is handed to run.command through a private FIFO, so the key never
// appears in a command line, the shell history or a log.
//
// This file holds everything except the Cocoa event loop, so it is tested on
// Linux and on the macOS runner alike.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"myautoapi-quick-connect/connect"
)

// Shown when another Terminal window of the assistant is still configuring
// Codex: only one window may write the configuration at a time.
const MessageBusy = "另一个快速接入助手窗口正在配置 Codex。请先在那个终端窗口完成或关闭它，再点一次「一键导入」。"

// MessageOpenFailed prefixes the reason Terminal could not be opened.
const MessageOpenFailed = "无法打开终端："

// MessageNoClaude answers an app=claude link: Claude Desktop import is in the
// Windows assistant only.
const MessageNoClaude = "Mac 版助手只能接入 Codex，一键导入 Claude 只有 Windows 版支持。Mac 上请按教程「Claude Desktop 接入」手动配置。"

// The state directory is shared with run.command, which takes the lock
// "run.lock" (a directory holding the owner's pid) for the whole time
// install.sh runs.
const (
	stateDirName = "Library/Caches/com.myautoapi.codex-config"
	lockName     = "run.lock"
	runnerMarker = "run.command"
)

// Assistant reacts to a double click (Open) and to import links (OpenLink).
type Assistant struct {
	Runner   string // Contents/Resources/run.command
	StateDir string // see stateDirName

	// OpenTerminal runs one shell command line in a new Terminal window and
	// returns once the window is open; the command keeps running there.
	OpenTerminal func(command string) error
	// Alert shows a message without waiting for it to be dismissed.
	Alert func(message string)
	// ShowTerminal brings Terminal to the front.
	ShowTerminal func()

	// HandoffTimeout bounds how long the key waits for run.command, which
	// includes the first-run "control Terminal" permission prompt.
	HandoffTimeout time.Duration

	mu       sync.Mutex
	handoffs int // keys not yet collected by run.command
	working  int // calls in progress, for Idle
}

// Idle reports whether nothing is in progress, so the launcher may quit.
func (a *Assistant) Idle() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.handoffs == 0 && a.working == 0
}

func (a *Assistant) begin() { a.mu.Lock(); a.working++; a.mu.Unlock() }
func (a *Assistant) end()   { a.mu.Lock(); a.working--; a.mu.Unlock() }

// Open is the double click: unchanged behaviour, run.command reads the key
// from the clipboard.
func (a *Assistant) Open() {
	a.begin()
	defer a.end()
	if a.busy() {
		a.reportBusy()
		return
	}
	if err := a.OpenTerminal("/bin/bash " + shellQuote(a.Runner)); err != nil {
		a.Alert(MessageOpenFailed + err.Error())
	}
}

// OpenLink handles one myautoapi:// link from the browser.
func (a *Assistant) OpenLink(raw string) {
	a.begin()
	defer a.end()
	link, problem := connect.ParseImportLink(raw)
	if problem == connect.LinkOK && link.App == connect.LinkAppClaude {
		a.Alert(MessageNoClaude) // the Mac assistant connects Codex only
		return
	}
	switch problem {
	case connect.LinkOK:
	case connect.LinkUnsupported:
		a.Alert(connect.MessageLinkUnsupported)
		return
	default:
		a.Alert(connect.MessageLinkIncomplete)
		return
	}

	// Checking and reserving under one lock keeps two quick clicks from
	// both opening a window.
	a.mu.Lock()
	if a.handoffs > 0 || lockHeld(a.StateDir) {
		a.mu.Unlock()
		a.reportBusy()
		return
	}
	a.handoffs++
	a.mu.Unlock()
	defer func() { a.mu.Lock(); a.handoffs--; a.mu.Unlock() }()

	fifo, err := a.makeHandoff()
	if err != nil {
		a.Alert(MessageOpenFailed + err.Error())
		return
	}
	defer os.RemoveAll(filepath.Dir(fifo))
	if err := a.OpenTerminal("/bin/bash " + shellQuote(a.Runner) + " --link " + shellQuote(fifo)); err != nil {
		a.Alert(MessageOpenFailed + err.Error())
		return
	}
	shown := connect.MaskKey(link.Key)
	if link.Name != "" {
		shown = link.Name + "（" + shown + "）"
	}
	// Errors here mean run.command gave up or never started; it reports
	// that in its own window, so nothing more is shown.
	_ = deliver(fifo, []byte(link.Key+"\n"+shown+"\n"), a.HandoffTimeout)
}

func (a *Assistant) busy() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.handoffs > 0 || lockHeld(a.StateDir)
}

func (a *Assistant) reportBusy() {
	if a.ShowTerminal != nil {
		a.ShowTerminal()
	}
	a.Alert(MessageBusy)
}

// makeHandoff creates a FIFO in a fresh directory only the user can enter.
func (a *Assistant) makeHandoff() (string, error) {
	if err := os.MkdirAll(a.StateDir, 0o700); err != nil {
		return "", err
	}
	directory, err := os.MkdirTemp(a.StateDir, "import-")
	if err != nil {
		return "", err
	}
	fifo := filepath.Join(directory, "key")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		_ = os.RemoveAll(directory)
		return "", err
	}
	return fifo, nil
}

// deliver writes payload once run.command has opened the FIFO. Until then a
// non-blocking open fails with ENXIO; run.command removes the FIFO when it
// will not read it, which ends the wait early.
func deliver(fifo string, payload []byte, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		file, err := os.OpenFile(fifo, os.O_WRONLY|syscall.O_NONBLOCK, 0)
		if err == nil {
			_, err = file.Write(payload)
			if closeErr := file.Close(); err == nil {
				err = closeErr
			}
			return err
		}
		if !errors.Is(err, syscall.ENXIO) {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("终端没有接收密钥：%w", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// lockHeld reports whether a live run.command owns the lock. A lock left by
// a closed window, or a pid now used by another program, does not count.
func lockHeld(stateDir string) bool {
	data, err := os.ReadFile(filepath.Join(stateDir, lockName, "pid"))
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return false
	}
	if err := syscall.Kill(pid, 0); err != nil && !errors.Is(err, syscall.EPERM) {
		return false
	}
	command, err := exec.Command("/bin/ps", "-o", "command=", "-p", strconv.Itoa(pid)).Output()
	return err == nil && strings.Contains(string(command), runnerMarker)
}

func shellQuote(text string) string {
	return "'" + strings.ReplaceAll(text, "'", `'\''`) + "'"
}
