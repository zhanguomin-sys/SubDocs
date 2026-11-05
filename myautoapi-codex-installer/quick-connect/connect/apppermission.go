package connect

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// The Codex App keeps the permission picked under the input box in
// .codex-global-state.json, not in config.toml, so sandbox_mode alone does
// not change what the App shows. These are the three entries the App itself
// writes when Full Access is chosen; the last one marks the Full Access
// warning as already dismissed.
const (
	appStateFile    = ".codex-global-state.json"
	appStateRoot    = "electron-persisted-atom-state"
	appModeKey      = "agent-mode-by-host-id"
	appSelectionKey = "permission-selection-by-host-id:local"
	appWarningKey   = "full-access-warning-dismissed-at-v2"
	appFullAccess   = "full-access"
)

// ErrAppStateUnreadable means .codex-global-state.json exists but is not a
// JSON object this code understands; it is left exactly as it was.
var ErrAppStateUnreadable = errors.New("Codex App 状态文件无法识别，未修改")

// GrantAppFullAccess sets the Codex App permission to Full Access. Codex must
// already be closed: the App writes this file back when it exits. A missing
// file is created with only these entries; every other entry of an existing
// file is kept.
func GrantAppFullAccess(codexHome string, now time.Time) error {
	if err := os.MkdirAll(codexHome, 0o700); err != nil {
		return err
	}
	path := filepath.Join(codexHome, appStateFile)
	state := map[string]any{}
	mode := fs.FileMode(0o644)
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return err
	default:
		if info, err := os.Stat(path); err == nil {
			mode = info.Mode().Perm()
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber() // large numbers stay exactly as written
		if decoder.Decode(&state) != nil || state == nil || decoder.More() {
			return ErrAppStateUnreadable
		}
	}

	atoms := map[string]any{}
	if existing, found := state[appStateRoot]; found {
		object, ok := existing.(map[string]any)
		if !ok {
			return ErrAppStateUnreadable
		}
		atoms = object
	}
	hosts := map[string]any{}
	if existing, ok := atoms[appModeKey].(map[string]any); ok {
		hosts = existing
	}
	hosts["local"] = appFullAccess
	atoms[appModeKey] = hosts
	atoms[appSelectionKey] = map[string]any{"kind": "agent-mode", "agentMode": appFullAccess}
	atoms[appWarningKey] = now.UnixMilli()
	state[appStateRoot] = atoms

	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(state); err != nil {
		return err
	}
	temporary, err := writeTemporary(codexHome, "."+appStateFile+".tmp.", bytes.TrimRight(out.Bytes(), "\n"))
	if err != nil {
		return err
	}
	defer os.Remove(temporary)
	_ = os.Chmod(temporary, mode)
	return renameFile(temporary, path)
}
