package connect

import (
	"os"
	"testing"
)

func mustRead(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return data
}

func TestPlanStop(t *testing.T) {
	processes := []Process{
		{PID: 4, PPID: 0, Name: "System", Session: 0},
		{PID: 100, PPID: 4, Name: "explorer.exe", Session: 1},
		{PID: 200, PPID: 100, Name: "MyAutoAPI中转快速接入助手.exe", Session: 1},
		{PID: 300, PPID: 100, Name: "ChatGPT.exe", Session: 1},
		{PID: 301, PPID: 300, Name: "ChatGPT.exe", Session: 1},
		{PID: 400, PPID: 100, Name: "codex.exe", Session: 1},
		{PID: 500, PPID: 4, Name: "codex.exe", Session: 2},      // another user
		{PID: 600, PPID: 100, Name: "codexbar.exe", Session: 1}, // similar name
		{PID: 700, PPID: 100, Name: "chatgpt-helper.exe", Session: 1},
	}
	targets, result := PlanStop(processes, 200)
	if result != (StopResult{ChatGPT: true, Codex: true}) || !result.Closed() {
		t.Fatalf("result %+v", result)
	}
	var pids []uint32
	for _, target := range targets {
		pids = append(pids, target.PID)
	}
	if len(pids) != 3 || pids[0] != 300 || pids[1] != 301 || pids[2] != 400 {
		t.Fatalf("targets %v", pids)
	}

	// Started from inside Codex: Codex is our ancestor and is left running.
	inside := append([]Process{}, processes...)
	inside[2].PPID = 400
	targets, result = PlanStop(inside, 200)
	if !result.Left || result.Closed() || len(targets) != 2 {
		t.Fatalf("ancestor case: %+v %v", result, targets)
	}

	// Nothing running.
	targets, result = PlanStop(processes[:3], 200)
	if len(targets) != 0 || result != (StopResult{}) || result.Closed() {
		t.Fatalf("idle case: %+v %v", result, targets)
	}

	// Our own process missing from the list: touch nothing.
	if targets, _ = PlanStop(processes, 999); len(targets) != 0 {
		t.Fatal("unknown self must not close anything")
	}
}

func TestProtocolRegistration(t *testing.T) {
	cases := map[string]string{
		`C:\Users\张三\Downloads\MyAutoAPI中转快速接入助手.exe`:        `"C:\Users\张三\Downloads\MyAutoAPI中转快速接入助手.exe" "%1"`,
		`D:\My Tools\1.MyAutoAPI中转快速接入助手.exe`:                `"D:\My Tools\1.MyAutoAPI中转快速接入助手.exe" "%1"`,
		`C:\Users\a\myautoapi-quick-connect-windows-x64.exe`: `"C:\Users\a\myautoapi-quick-connect-windows-x64.exe" "%1"`,
	}
	for exe, want := range cases {
		if got := ProtocolCommand(exe); got != want {
			t.Errorf("ProtocolCommand(%q) = %q", exe, got)
		}
	}
	if ProtocolKey != `Software\Classes\myautoapi` || ProtocolCommandKey != `Software\Classes\myautoapi\shell\open\command` {
		t.Fatal("registry keys changed")
	}
	if ProtocolDescription != "URL:MyAutoAPI中转快速接入助手" {
		t.Fatal("protocol description changed")
	}
}
