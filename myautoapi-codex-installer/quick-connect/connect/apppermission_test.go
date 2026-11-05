package connect

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readAppState(t *testing.T, codexHome string) map[string]any {
	t.Helper()
	var state map[string]any
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(codexHome, appStateFile))), &state); err != nil {
		t.Fatal(err)
	}
	return state
}

func assertFullAccess(t *testing.T, atoms map[string]any) {
	t.Helper()
	hosts, _ := atoms[appModeKey].(map[string]any)
	selection, _ := atoms[appSelectionKey].(map[string]any)
	if hosts["local"] != "full-access" || selection["kind"] != "agent-mode" || selection["agentMode"] != "full-access" {
		t.Fatalf("not full access: %v", atoms)
	}
	if atoms[appWarningKey] != float64(fixedTime.UnixMilli()) {
		t.Fatalf("warning dismissed at %v", atoms[appWarningKey])
	}
}

func TestGrantAppFullAccessCreatesMissingFile(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), ".codex")
	if err := GrantAppFullAccess(codexHome, fixedTime); err != nil {
		t.Fatal(err)
	}
	state := readAppState(t, codexHome)
	if len(state) != 1 {
		t.Fatalf("only the App entries may be written: %v", state)
	}
	atoms := state[appStateRoot].(map[string]any)
	if len(atoms) != 3 {
		t.Fatalf("atoms %v", atoms)
	}
	assertFullAccess(t, atoms)
	assertOnlyFiles(t, codexHome, appStateFile)
}

func TestGrantAppFullAccessKeepsOtherEntries(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), ".codex")
	original := `{"other":{"big":12345678901234567890,"text":"<中文>"},` +
		`"electron-persisted-atom-state":{"agent-mode-by-host-id":{"local":"auto","remote":"auto"},` +
		`"permission-selection-by-host-id:local":{"kind":"agent-mode","agentMode":"auto"},"keep":[1,2.5,null]}}`
	writeFile(t, filepath.Join(codexHome, appStateFile), original)
	if err := GrantAppFullAccess(codexHome, fixedTime); err != nil {
		t.Fatal(err)
	}
	raw := readFile(t, filepath.Join(codexHome, appStateFile))
	if !strings.Contains(raw, `"big":12345678901234567890`) || !strings.Contains(raw, `"text":"<中文>"`) {
		t.Fatalf("other entries changed: %s", raw)
	}
	atoms := readAppState(t, codexHome)[appStateRoot].(map[string]any)
	assertFullAccess(t, atoms)
	if hosts := atoms[appModeKey].(map[string]any); hosts["remote"] != "auto" {
		t.Fatalf("other hosts changed: %v", hosts)
	}
	if keep, _ := json.Marshal(atoms["keep"]); string(keep) != `[1,2.5,null]` {
		t.Fatalf("keep = %s", keep)
	}
	assertOnlyFiles(t, codexHome, appStateFile)
}

func TestGrantAppFullAccessLeavesUnknownFilesAlone(t *testing.T) {
	for _, content := range []string{"", "{broken", "[]", "null", `{"a":1} {"b":2}`, `{"electron-persisted-atom-state":[]}`} {
		codexHome := filepath.Join(t.TempDir(), ".codex")
		writeFile(t, filepath.Join(codexHome, appStateFile), content)
		if err := GrantAppFullAccess(codexHome, fixedTime); err != ErrAppStateUnreadable {
			t.Fatalf("%q: err = %v", content, err)
		}
		if got := readFile(t, filepath.Join(codexHome, appStateFile)); got != content {
			t.Fatalf("%q was changed to %q", content, got)
		}
		assertOnlyFiles(t, codexHome, appStateFile)
	}
}

// Codex has to be closed before anything is written, or the App would write
// its old permission back when it exits.
func TestConnectClosesCodexBeforeWriting(t *testing.T) {
	service, _ := newTestService(t, 200, `{"isValid":true,"remaining":1}`, StopResult{Codex: true})
	service.StopClients = func() StopResult {
		for _, name := range []string{"config.toml", "auth.json", appStateFile} {
			if _, err := os.Stat(filepath.Join(service.CodexHome, name)); err == nil {
				t.Fatalf("%s was written before Codex was closed", name)
			}
		}
		return StopResult{Codex: true}
	}
	if result := service.Connect(context.Background(), fixtureKey, ""); result.Tone != ToneSuccess {
		t.Fatalf("%+v", result)
	}
	assertFullAccess(t, readAppState(t, service.CodexHome)[appStateRoot].(map[string]any))
}

func TestConnectTellsWhenFullAccessFails(t *testing.T) {
	service, _ := newTestService(t, 200, `{"isValid":true,"remaining":5}`, StopResult{Codex: true})
	writeFile(t, filepath.Join(service.CodexHome, appStateFile), "not json")
	result := service.Connect(context.Background(), fixtureKey, "")
	if !strings.HasSuffix(result.Message, MessageFullAccessManual) {
		t.Fatalf("message %q", result.Message)
	}
}
