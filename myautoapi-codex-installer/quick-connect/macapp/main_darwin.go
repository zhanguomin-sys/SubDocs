//go:build darwin

package main

/*
#cgo CFLAGS: -mmacosx-version-min=12.0
#cgo LDFLAGS: -framework Cocoa -mmacosx-version-min=12.0
void runApplication(void);
void stopApplication(void);
*/
import "C"

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

const terminalScript = `on run argv
    tell application "Terminal"
        activate
        do script (item 1 of argv)
    end tell
end run`

const alertScript = `on run argv
    display alert "MyAutoAPI中转快速接入助手" message (item 1 of argv) as critical
end run`

// Cocoa needs the main thread.
func init() { runtime.LockOSThread() }

var (
	assistant *Assistant
	events    sync.WaitGroup
	clock     sync.Mutex
	lastEvent time.Time
	lastLink  time.Time
	handled   bool
)

func main() {
	executable, err := os.Executable()
	home, homeErr := os.UserHomeDir()
	if err != nil || homeErr != nil {
		showAlert(MessageOpenFailed + "找不到程序位置。")
		os.Exit(1)
	}
	assistant = &Assistant{
		Runner:         filepath.Join(filepath.Dir(executable), "..", "Resources", "run.command"),
		StateDir:       filepath.Join(home, stateDirName),
		OpenTerminal:   openTerminal,
		Alert:          showAlert,
		ShowTerminal:   func() { _ = exec.Command("/usr/bin/open", "-a", "Terminal").Run() },
		HandoffTimeout: 5 * time.Minute,
	}
	clock.Lock()
	lastEvent = time.Now()
	clock.Unlock()
	go quitWhenIdle()
	C.runApplication()
}

// quitWhenIdle ends the launcher a moment after the last event once nothing
// is in progress. A launcher started without any event quits after 10 s.
func quitWhenIdle() {
	for range time.Tick(250 * time.Millisecond) {
		clock.Lock()
		quiet := time.Since(lastEvent)
		seen := handled
		clock.Unlock()
		if assistant.Idle() && ((seen && quiet > 2*time.Second) || quiet > 10*time.Second) {
			C.stopApplication()
			return
		}
	}
}

func noteEvent(link bool) time.Time {
	clock.Lock()
	defer clock.Unlock()
	now := time.Now()
	lastEvent, handled = now, true
	if link {
		lastLink = now
	}
	return now
}

//export goOpenURL
func goOpenURL(url *C.char) {
	raw := C.GoString(url)
	noteEvent(true)
	assistant.begin()
	go func() {
		defer assistant.end()
		assistant.OpenLink(raw)
	}()
}

// goOpenApplication is a double click, or a click on the running app. If a
// link arrives right after, the link wins and no clipboard window opens.
//
//export goOpenApplication
func goOpenApplication() {
	at := noteEvent(false)
	assistant.begin()
	go func() {
		defer assistant.end()
		time.Sleep(500 * time.Millisecond)
		clock.Lock()
		linked := !lastLink.Before(at.Add(-time.Second))
		clock.Unlock()
		if !linked {
			assistant.Open()
		}
	}()
}

func openTerminal(command string) error {
	output, err := exec.Command("/usr/bin/osascript", "-e", terminalScript, command).CombinedOutput()
	if err != nil {
		return &openError{string(output)}
	}
	return nil
}

type openError struct{ output string }

func (e *openError) Error() string { return e.output }

func showAlert(message string) {
	_ = exec.Command("/usr/bin/osascript", "-e", alertScript, message).Start()
}
