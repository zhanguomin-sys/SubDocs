//go:build darwin || linux

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"myautoapi-quick-connect/connect"
)

// Synthetic; no test uses a real key or the network.
const fixtureKey = "sk-macapp_fixture-0123456789abcdefXYZ"

// rig is one .app's Resources with the real run.command and a fake
// install.sh that records what it was given, a fake pbpaste, and a
// Terminal stand-in that runs the command line in the background.
type rig struct {
	t         *testing.T
	home      string
	resources string
	record    string
	assistant *Assistant

	mu       sync.Mutex
	commands []string
	alerts   []string
	shown    int
	windows  []*window
}

type window struct {
	done   chan struct{}
	output bytes.Buffer
}

const fakeInstaller = `#!/bin/bash
printf '%s' "${MYAUTOAPI_KEY-<unset>}" > "$RECORD/key"
printf '%s\n' "$@" > "$RECORD/args"
ps -o command= -p $$ > "$RECORD/ps"
if [ -n "${HOLD:-}" ]; then
  while [ ! -e "$HOLD" ]; do sleep 0.1; done
fi
exit "${EXIT_CODE:-0}"
`

func newRig(t *testing.T) *rig {
	t.Helper()
	root := t.TempDir()
	r := &rig{t: t, home: filepath.Join(root, "家 o'brien"), resources: filepath.Join(root, "中文 app's $(x)", "Resources"), record: filepath.Join(root, "record")}
	for _, directory := range []string{r.home, r.resources, r.record, filepath.Join(root, "bin")} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	runner, err := os.ReadFile("run.command")
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(r.resources, "run.command"), string(runner))
	write(t, filepath.Join(r.resources, "install.sh"), fakeInstaller)
	write(t, filepath.Join(root, "bin", "pbpaste"), "#!/bin/sh\ntouch \"$RECORD/pbpaste\"\nprintf 'sk-clipboard-0123456789'\n")
	t.Setenv("PATH", filepath.Join(root, "bin")+":"+os.Getenv("PATH"))
	t.Setenv("HOME", r.home)
	t.Setenv("RECORD", r.record)
	r.assistant = &Assistant{
		Runner:         filepath.Join(r.resources, "run.command"),
		StateDir:       filepath.Join(r.home, stateDirName),
		OpenTerminal:   r.openTerminal,
		Alert:          func(message string) { r.mu.Lock(); r.alerts = append(r.alerts, message); r.mu.Unlock() },
		ShowTerminal:   func() { r.mu.Lock(); r.shown++; r.mu.Unlock() },
		HandoffTimeout: 10 * time.Second,
	}
	t.Cleanup(r.waitWindows)
	return r
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o755); err != nil {
		t.Fatal(err)
	}
}

// openTerminal behaves like Terminal's "do script": the line goes to a shell
// and keeps running after the call returns.
func (r *rig) openTerminal(command string) error {
	w := &window{done: make(chan struct{})}
	shell := exec.Command("/bin/bash", "-c", command)
	shell.Stdin = strings.NewReader("")
	shell.Stdout, shell.Stderr = &w.output, &w.output
	if err := shell.Start(); err != nil {
		return err
	}
	go func() { _ = shell.Wait(); close(w.done) }()
	r.mu.Lock()
	r.commands = append(r.commands, command)
	r.windows = append(r.windows, w)
	r.mu.Unlock()
	return nil
}

func (r *rig) waitWindows() {
	r.mu.Lock()
	windows := append([]*window(nil), r.windows...)
	r.mu.Unlock()
	for _, w := range windows {
		select {
		case <-w.done:
		case <-time.After(40 * time.Second):
			r.t.Fatal("terminal window did not finish")
		}
	}
}

func (r *rig) recorded(name string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(r.record, name))
	return string(data), err == nil
}

func (r *rig) output() string {
	r.waitWindows()
	r.mu.Lock()
	defer r.mu.Unlock()
	var all strings.Builder
	for _, w := range r.windows {
		all.Write(w.output.Bytes())
	}
	return all.String()
}

func (r *rig) assertClean() {
	r.t.Helper()
	entries, _ := os.ReadDir(r.assistant.StateDir)
	for _, entry := range entries {
		r.t.Errorf("left in state directory: %s", entry.Name())
	}
	if !r.assistant.Idle() {
		r.t.Error("assistant not idle")
	}
}

// mask keeps fixture keys out of failure messages.
func mask(text string) string {
	return regexp.MustCompile(`sk-[A-Za-z0-9_-]+`).ReplaceAllStringFunc(text, connect.MaskKey)
}

// Every link on test-link.html, with and without a name, gives what the
// page's title promises: an import with the key passed as MYAUTOAPI_KEY,
// or the same message the Windows window shows.
func TestEveryLinkOnTheTestPage(t *testing.T) {
	page, err := os.ReadFile(filepath.Join("..", "test-link.html"))
	if err != nil {
		t.Fatal(err)
	}
	cases := regexp.MustCompile(`\['(?:main)?', '([^']+)', \(k, n\) => `+"`([^`]+)`"+`\]`).FindAllStringSubmatch(string(page), -1)
	if len(cases) != 7 {
		t.Fatalf("test-link.html has %d links, this test knows 7; update both together", len(cases))
	}
	literal := regexp.MustCompile(`\$\{encodeURIComponent\('([^']*)'\)\}`)
	for _, c := range cases {
		for _, name := range []string{"", "我的 Key & 测试"} {
			title := c[1]
			href := literal.ReplaceAllStringFunc(c[2], func(m string) string {
				return encodeURIComponent(literal.FindStringSubmatch(m)[1])
			})
			nameParameter := ""
			if name != "" {
				nameParameter = "&name=" + encodeURIComponent(name)
			}
			href = strings.NewReplacer("${k}", encodeURIComponent(fixtureKey), "${n}", nameParameter).Replace(href)
			t.Run(fmt.Sprintf("%s/name=%q", title, name), func(t *testing.T) {
				r := newRig(t)
				r.assistant.OpenLink(href)
				output := r.output()
				switch {
				case strings.Contains(title, "只有 Windows 支持"):
					r.expectRejected(MessageNoClaude)
				case strings.Contains(title, "导入链接不完整"):
					r.expectRejected(connect.MessageLinkIncomplete)
				case strings.Contains(title, "请下载新版助手"):
					r.expectRejected(connect.MessageLinkUnsupported)
				default:
					r.expectImported(output, name)
				}
				r.assertClean()
			})
		}
	}
}

func (r *rig) expectRejected(message string) {
	r.t.Helper()
	if len(r.alerts) != 1 || r.alerts[0] != message || len(r.commands) != 0 {
		r.t.Fatalf("alerts %q, terminal commands %d; want only %q", r.alerts, len(r.commands), message)
	}
	if _, ran := r.recorded("args"); ran {
		r.t.Fatal("install.sh ran for a rejected link")
	}
}

func (r *rig) expectImported(output, name string) {
	r.t.Helper()
	if len(r.alerts) != 0 || len(r.commands) != 1 {
		r.t.Fatalf("alerts %q, terminal commands %d", r.alerts, len(r.commands))
	}
	if strings.Contains(r.commands[0], fixtureKey) || !strings.Contains(r.commands[0], " --link ") {
		r.t.Fatalf("terminal command line: %s", mask(r.commands[0]))
	}
	key, _ := r.recorded("key")
	args, _ := r.recorded("args")
	processes, _ := r.recorded("ps")
	if key != fixtureKey || strings.TrimSpace(args) != "" || strings.Contains(processes, fixtureKey) {
		r.t.Fatalf("install.sh got MYAUTOAPI_KEY %q, args %q, command line %q", mask(key), args, mask(processes))
	}
	if _, read := r.recorded("pbpaste"); read {
		r.t.Fatal("an import link read the clipboard")
	}
	want := "已收到网页导入的密钥：" + connect.MaskKey(fixtureKey)
	if name != "" {
		want = "已收到网页导入的密钥：" + name + "（" + connect.MaskKey(fixtureKey) + "）"
	}
	if !strings.Contains(output, want+"\n") || strings.Contains(output, fixtureKey) {
		r.t.Fatalf("terminal output: %s", mask(output))
	}
}

func TestDoubleClickStillReadsTheClipboard(t *testing.T) {
	t.Setenv("MYAUTOAPI_KEY", "")
	r := newRig(t)
	r.assistant.Open()
	r.output()
	args, _ := r.recorded("args")
	key, _ := r.recorded("key")
	if strings.TrimSpace(args) != "--from-app" || key != "" || len(r.alerts) != 0 {
		t.Fatalf("args %q, MYAUTOAPI_KEY %q, alerts %q", args, mask(key), r.alerts)
	}
	if strings.Contains(r.commands[0], "--link") {
		t.Fatalf("double click command: %s", r.commands[0])
	}
	r.assertClean()
}

func TestInstallerExitCodeIsShown(t *testing.T) {
	t.Setenv("EXIT_CODE", "3")
	r := newRig(t)
	r.assistant.OpenLink("myautoapi://import?v=1&app=codex&key=" + fixtureKey)
	if output := r.output(); !strings.Contains(output, "退出码：3") || !strings.Contains(output, "按回车结束") {
		t.Fatalf("output: %s", mask(output))
	}
	r.assertClean()
}

// While a window is still configuring Codex, neither a link nor a double
// click opens a second one; the first window is brought to the front.
func TestSecondWindowIsRefusedWhileOneIsRunning(t *testing.T) {
	r := newRig(t)
	release := filepath.Join(r.record, "release")
	t.Setenv("HOLD", release)
	r.assistant.OpenLink("myautoapi://import?v=1&app=codex&key=" + fixtureKey)
	waitFor(t, func() bool { _, ok := r.recorded("args"); return ok })

	r.assistant.OpenLink("myautoapi://import?v=1&app=codex&key=sk-second-0123456789")
	r.assistant.Open()
	r.mu.Lock()
	alerts, commands, shown := append([]string(nil), r.alerts...), len(r.commands), r.shown
	r.mu.Unlock()
	if commands != 1 || shown != 2 || len(alerts) != 2 || alerts[0] != MessageBusy || alerts[1] != MessageBusy {
		t.Fatalf("commands %d, shown %d, alerts %q", commands, shown, alerts)
	}

	write(t, release, "")
	r.output()
	if key, _ := r.recorded("key"); key != fixtureKey {
		t.Fatalf("the first import was replaced: %s", mask(key))
	}
	// Once the first window is done the next link imports again.
	_ = os.Remove(filepath.Join(r.record, "args"))
	t.Setenv("HOLD", "")
	r.assistant.OpenLink("myautoapi://import?v=1&app=codex&key=sk-second-0123456789")
	r.output()
	if key, _ := r.recorded("key"); key != "sk-second-0123456789" {
		t.Fatalf("second import: %s", mask(key))
	}
	r.assertClean()
}

// Two links in quick succession, before the first window has started: the
// key waiting in the FIFO also counts as a running import.
func TestQuickSecondLinkIsRefused(t *testing.T) {
	r := newRig(t)
	opened := make(chan struct{})
	start := r.assistant.OpenTerminal
	r.assistant.OpenTerminal = func(command string) error {
		go func() { <-opened; _ = start(command) }()
		return nil
	}
	done := make(chan struct{})
	go func() { r.assistant.OpenLink("myautoapi://import?v=1&app=codex&key=" + fixtureKey); close(done) }()
	waitFor(t, func() bool { r.assistant.mu.Lock(); defer r.assistant.mu.Unlock(); return r.assistant.handoffs == 1 })
	r.assistant.OpenLink("myautoapi://import?v=1&app=codex&key=sk-second-0123456789")
	r.mu.Lock()
	alerts := append([]string(nil), r.alerts...)
	r.mu.Unlock()
	if len(alerts) != 1 || alerts[0] != MessageBusy {
		t.Fatalf("alerts %q", alerts)
	}
	close(opened)
	<-done
	r.output()
	if key, _ := r.recorded("key"); key != fixtureKey {
		t.Fatalf("key %s", mask(key))
	}
	r.assertClean()
}

// run.command keeps its own lock, so two windows started some other way
// still never run install.sh at the same time.
func TestRunnerLockWithoutTheLauncher(t *testing.T) {
	r := newRig(t)
	release := filepath.Join(r.record, "release")
	t.Setenv("HOLD", release)
	if err := r.openTerminal("/bin/bash " + shellQuote(r.assistant.Runner)); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { _, ok := r.recorded("args"); return ok })
	if !lockHeld(r.assistant.StateDir) {
		t.Fatal("lock not visible to the launcher")
	}
	_ = os.Remove(filepath.Join(r.record, "args"))

	fifo, err := r.assistant.makeHandoff()
	if err != nil {
		t.Fatal(err)
	}
	second := exec.Command("/bin/bash", r.assistant.Runner, "--link", fifo)
	second.Stdin = strings.NewReader("")
	output, _ := second.CombinedOutput()
	if !strings.Contains(string(output), "另一个快速接入助手窗口正在配置 Codex") {
		t.Fatalf("second window: %s", output)
	}
	if _, err := os.Stat(fifo); !os.IsNotExist(err) {
		t.Fatal("refused window left the FIFO, the launcher would wait for nothing")
	}
	if _, ran := r.recorded("args"); ran {
		t.Fatal("second install.sh ran")
	}
	_ = os.RemoveAll(filepath.Dir(fifo))
	write(t, release, "")
	r.output()
	r.assertClean()
}

func TestLockLeftByAClosedWindowIsIgnored(t *testing.T) {
	r := newRig(t)
	finished := exec.Command("/bin/sh", "-c", "exit 0")
	if err := finished.Run(); err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(r.assistant.StateDir, lockName)
	if err := os.MkdirAll(lock, 0o700); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(lock, "pid"), fmt.Sprintf("%d\n", finished.Process.Pid))
	if lockHeld(r.assistant.StateDir) {
		t.Fatal("dead pid counted as running")
	}
	// A live process that is not run.command (pid reused) does not count either.
	write(t, filepath.Join(lock, "pid"), fmt.Sprintf("%d\n", os.Getpid()))
	if lockHeld(r.assistant.StateDir) {
		t.Fatal("unrelated process counted as running")
	}
	r.assistant.OpenLink("myautoapi://import?v=1&app=codex&key=" + fixtureKey)
	r.output()
	if key, _ := r.recorded("key"); key != fixtureKey || len(r.alerts) != 0 {
		t.Fatalf("key %s, alerts %q", mask(key), r.alerts)
	}
	r.assertClean()
}

// If the window never collects the key, the launcher stops waiting and
// removes the FIFO; a window that opens late reports the missing key.
func TestKeyNotCollected(t *testing.T) {
	r := newRig(t)
	r.assistant.HandoffTimeout = 300 * time.Millisecond
	var command string
	r.assistant.OpenTerminal = func(line string) error { command = line; return nil }
	r.assistant.OpenLink("myautoapi://import?v=1&app=codex&key=" + fixtureKey)
	r.assertClean()
	if err := r.openTerminal(command); err != nil {
		t.Fatal(err)
	}
	if output := r.output(); !strings.Contains(output, "没有收到网页传来的密钥") {
		t.Fatalf("output: %s", mask(output))
	}
	if _, ran := r.recorded("args"); ran {
		t.Fatal("install.sh ran without a key")
	}
}

func TestTerminalCannotBeOpened(t *testing.T) {
	r := newRig(t)
	r.assistant.OpenTerminal = func(string) error { return fmt.Errorf("不允许控制终端") }
	r.assistant.OpenLink("myautoapi://import?v=1&app=codex&key=" + fixtureKey)
	r.assistant.Open()
	if len(r.alerts) != 2 || r.alerts[0] != MessageOpenFailed+"不允许控制终端" || r.alerts[1] != r.alerts[0] {
		t.Fatalf("alerts %q", r.alerts)
	}
	r.assertClean()
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if condition() {
			return
		}
	}
	t.Fatal("timed out")
}

// encodeURIComponent matches the JavaScript function used by the website.
func encodeURIComponent(text string) string {
	var out strings.Builder
	for _, b := range []byte(text) {
		if 'a' <= b && b <= 'z' || 'A' <= b && b <= 'Z' || '0' <= b && b <= '9' || strings.IndexByte("-_.!~*'()", b) >= 0 {
			out.WriteByte(b)
		} else {
			fmt.Fprintf(&out, "%%%02X", b)
		}
	}
	return out.String()
}
