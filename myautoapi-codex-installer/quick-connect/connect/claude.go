package connect

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Claude Desktop (the Claude app) has a third-party mode in which it sends
// every request to a gateway instead of Anthropic. The mode is switched by
// files the app reads at start (the same ones CC Switch writes):
//
//   - claude_desktop_config.json: "deploymentMode": "3p". The app keeps one
//     in %LOCALAPPDATA%\Claude-3p; the Store (MSIX) build also has its normal
//     one inside the package's virtualised AppData; CC Switch adds one in
//     %LOCALAPPDATA%\Claude. All of them get the same value.
//   - Claude-3p\configLibrary\<id>.json: the connection (gateway address,
//     key, models). MyAutoAPI uses its own fixed id, so a CC Switch profile
//     next to it stays untouched.
//   - Claude-3p\configLibrary\_meta.json: lists the profiles; appliedId is
//     the one in use.
//
// Only the current Store build is supported: it is what claude.com/download
// installs. Older installs under %LOCALAPPDATA%\AnthropicClaude are not.
const (
	ClaudeProfileID   = "00000000-0000-4000-8000-00000000600d"
	ClaudeProfileName = "MyAutoAPI"

	claudePackagePrefix  = "Claude_"
	claudePackageFamily  = "Claude_pzs8sxrjxfjjc"
	claudeConfigName     = "claude_desktop_config.json"
	claudeLibraryDirName = "configLibrary"
	claudeMetaName       = "_meta.json"
)

// Claude card states.
const (
	ClaudeNotInstalled = "notinstalled"
	ClaudeNone         = "none"
	ClaudeMyAutoAPI    = "myautoapi"
	ClaudeOther        = "other"
)

// ClaudeStatus is what the Claude card shows. Key is masked.
type ClaudeStatus struct {
	State  string `json:"state"`
	Key    string `json:"key,omitempty"`
	Name   string `json:"name,omitempty"`
	Models int    `json:"models,omitempty"`
}

// ClaudeDesktop finds Claude Desktop's files for one Windows user.
// LocalAppData is %LOCALAPPDATA%.
type ClaudeDesktop struct {
	LocalAppData string
}

// packageDir is the Store package's data folder, "" when Claude Desktop is
// not installed.
func (d ClaudeDesktop) packageDir() string {
	if d.LocalAppData == "" {
		return ""
	}
	packages := filepath.Join(d.LocalAppData, "Packages")
	if isDir(filepath.Join(packages, claudePackageFamily)) {
		return filepath.Join(packages, claudePackageFamily)
	}
	matches, _ := filepath.Glob(filepath.Join(packages, claudePackagePrefix+"*"))
	sort.Strings(matches)
	for _, match := range matches {
		if isDir(match) {
			return match
		}
	}
	return ""
}

// Installed reports whether the Store build of Claude Desktop is present.
func (d ClaudeDesktop) Installed() bool { return d.packageDir() != "" }

func (d ClaudeDesktop) threePDir() string { return filepath.Join(d.LocalAppData, "Claude-3p") }

func (d ClaudeDesktop) libraryDir() string {
	return filepath.Join(d.threePDir(), claudeLibraryDirName)
}

func (d ClaudeDesktop) profilePath() string {
	return filepath.Join(d.libraryDir(), ClaudeProfileID+".json")
}

func (d ClaudeDesktop) metaPath() string { return filepath.Join(d.libraryDir(), claudeMetaName) }

// configPaths lists every claude_desktop_config.json that carries the
// deployment mode, the third-party one first.
func (d ClaudeDesktop) configPaths() []string {
	paths := []string{
		filepath.Join(d.threePDir(), claudeConfigName),
		filepath.Join(d.LocalAppData, "Claude", claudeConfigName),
	}
	if pkg := d.packageDir(); pkg != "" {
		paths = append(paths, filepath.Join(pkg, "LocalCache", "Roaming", "Claude", claudeConfigName))
	}
	return paths
}

// ReadClaudeStatus reports MyAutoAPI when the third-party mode is on, the
// applied profile is MyAutoAPI's and it points at api.myautoapi.com.
func ReadClaudeStatus(d ClaudeDesktop) ClaudeStatus {
	status, _ := readClaudeStatus(d)
	return status
}

// readClaudeStatus also returns the full key, for looking up its name.
func readClaudeStatus(d ClaudeDesktop) (ClaudeStatus, string) {
	if !d.Installed() {
		return ClaudeStatus{State: ClaudeNotInstalled}, ""
	}
	thirdParty := false
	for _, path := range d.configPaths() {
		if config, _, err := readJSONObject(path); err == nil && config.getString("deploymentMode") == "3p" {
			thirdParty = true
			break
		}
	}
	if !thirdParty {
		return ClaudeStatus{State: ClaudeNone}, ""
	}
	meta, _, err := readJSONObject(d.metaPath())
	if err != nil || meta.getString("appliedId") != ClaudeProfileID {
		return ClaudeStatus{State: ClaudeOther}, ""
	}
	profile, exists, err := readJSONObject(d.profilePath())
	if err != nil || !exists || !isMyAutoAPI(profile.getString("inferenceGatewayBaseUrl")) {
		return ClaudeStatus{State: ClaudeOther}, ""
	}
	key := profile.getString("inferenceGatewayApiKey")
	status := ClaudeStatus{State: ClaudeMyAutoAPI}
	if key != "" {
		status.Key = MaskKey(key)
	}
	if raw, ok := profile.get("inferenceModels"); ok {
		var models []json.RawMessage
		if json.Unmarshal(raw, &models) == nil {
			status.Models = len(models)
		}
	}
	return status, key
}

// WriteClaudeDesktop turns on the third-party mode with MyAutoAPI's profile:
// key and the given models (already filtered by ClaudeDesktopModels). Other
// settings in every file stay as they are; the config files and _meta.json
// get a .bak copy first. All files are written or none.
func WriteClaudeDesktop(d ClaudeDesktop, key string, models []string, now time.Time) error {
	if !d.Installed() {
		return errors.New("没有安装 Claude 桌面版")
	}
	if len(models) == 0 {
		return errors.New("没有可写入的 Claude 模型")
	}
	var changes []fileChange
	for _, path := range d.configPaths() {
		config, _, err := readJSONObject(path)
		if err != nil {
			return err
		}
		if err := config.set("deploymentMode", "3p"); err != nil {
			return err
		}
		data, err := config.encode()
		if err != nil {
			return err
		}
		changes = append(changes, fileChange{path: path, data: data, backup: true})
	}

	profile, _, err := readJSONObject(d.profilePath())
	if err != nil {
		profile = newJSONObject() // our own file: a damaged copy is simply replaced
	}
	if _, ok := profile.get("coworkEgressAllowedHosts"); !ok {
		if err := profile.set("coworkEgressAllowedHosts", []string{"*"}); err != nil {
			return err
		}
	}
	for _, field := range []struct {
		key   string
		value any
	}{
		{"disableDeploymentModeChooser", true},
		{"inferenceProvider", "gateway"},
		{"inferenceGatewayBaseUrl", BaseURL},
		{"inferenceGatewayAuthScheme", "bearer"},
		{"inferenceGatewayApiKey", key},
		{"modelDiscoveryEnabled", false},
		{"inferenceModels", models},
	} {
		if err := profile.set(field.key, field.value); err != nil {
			return err
		}
	}
	profileData, err := profile.encode()
	if err != nil {
		return err
	}
	changes = append(changes, fileChange{path: d.profilePath(), data: profileData})

	meta, _, err := readJSONObject(d.metaPath())
	if err != nil {
		return err
	}
	if err := applyClaudeMeta(meta, true); err != nil {
		return err
	}
	metaData, err := meta.encode()
	if err != nil {
		return err
	}
	changes = append(changes, fileChange{path: d.metaPath(), data: metaData, backup: true})
	return writeFileGroup(changes, now)
}

// RestoreClaudeDesktop switches Claude Desktop back to the official login:
// every config file that names a deployment mode gets "1p", MyAutoAPI's
// entry leaves _meta.json (appliedId passes to the next profile, if any) and
// its profile file, which holds the key, is removed. Files that do not exist
// stay missing. When another profile (CC Switch, say) is the applied one,
// only MyAutoAPI's leftovers are removed and the mode is left alone.
func RestoreClaudeDesktop(d ClaudeDesktop, now time.Time) error {
	if d.LocalAppData == "" {
		return errors.New("找不到当前用户的文件夹")
	}
	meta, metaExists, err := readJSONObject(d.metaPath())
	if err != nil {
		return err
	}
	ours := !metaExists || meta.getString("appliedId") == "" || meta.getString("appliedId") == ClaudeProfileID
	var changes []fileChange
	for _, path := range d.configPaths() {
		if !ours {
			break
		}
		config, exists, err := readJSONObject(path)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		if _, ok := config.get("deploymentMode"); !ok {
			continue
		}
		if err := config.set("deploymentMode", "1p"); err != nil {
			return err
		}
		data, err := config.encode()
		if err != nil {
			return err
		}
		changes = append(changes, fileChange{path: path, data: data, backup: true})
	}
	if metaExists {
		if err := applyClaudeMeta(meta, false); err != nil {
			return err
		}
		data, err := meta.encode()
		if err != nil {
			return err
		}
		changes = append(changes, fileChange{path: d.metaPath(), data: data, backup: true})
	}
	if fileExists(d.profilePath()) {
		changes = append(changes, fileChange{path: d.profilePath()})
	}
	return writeFileGroup(changes, now)
}

// applyClaudeMeta adds (applied) or removes MyAutoAPI's entry in
// _meta.json. Other entries keep their place and every field.
func applyClaudeMeta(meta *jsonObject, applied bool) error {
	var entries []json.RawMessage
	if raw, ok := meta.get("entries"); ok {
		if err := json.Unmarshal(raw, &entries); err != nil {
			return errors.New("_meta.json 的 entries 不是列表")
		}
	}
	entryID := func(raw json.RawMessage) string {
		var entry struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(raw, &entry)
		return entry.ID
	}
	var kept []json.RawMessage
	found := false
	for _, entry := range entries {
		if entryID(entry) != ClaudeProfileID {
			kept = append(kept, entry)
			continue
		}
		if applied && !found {
			object, err := parseJSONObject(entry)
			if err != nil {
				object = newJSONObject()
				_ = object.set("id", ClaudeProfileID)
			}
			if err := object.set("name", ClaudeProfileName); err != nil {
				return err
			}
			data, err := object.encode()
			if err != nil {
				return err
			}
			kept = append(kept, json.RawMessage(data))
			found = true
		}
	}
	if applied {
		if !found {
			data, _ := json.Marshal(map[string]string{"id": ClaudeProfileID, "name": ClaudeProfileName})
			kept = append(kept, data)
		}
		if kept == nil {
			kept = []json.RawMessage{}
		}
		if err := meta.set("entries", kept); err != nil {
			return err
		}
		return meta.set("appliedId", ClaudeProfileID)
	}
	if _, ok := meta.get("entries"); ok {
		if kept == nil {
			kept = []json.RawMessage{}
		}
		if err := meta.set("entries", kept); err != nil {
			return err
		}
	}
	if meta.getString("appliedId") == ClaudeProfileID {
		next := ""
		for _, entry := range kept {
			if id := entryID(entry); id != "" {
				next = id
				break
			}
		}
		if next == "" {
			meta.remove("appliedId")
		} else if err := meta.set("appliedId", next); err != nil {
			return err
		}
	}
	return nil
}

// ClaudeCode finds the Claude Code CLI in its default install places:
// Home is %USERPROFILE%, AppData is %APPDATA%.
type ClaudeCode struct {
	Home    string
	AppData string
}

// claudeCodeModelOverrides would send another service's model names to
// MyAutoAPI, or send the CLI to another cloud (Bedrock, Vertex, Foundry)
// altogether, so connecting removes them.
var claudeCodeModelOverrides = []string{
	"CLAUDE_CODE_USE_BEDROCK",
	"CLAUDE_CODE_USE_VERTEX",
	"CLAUDE_CODE_USE_FOUNDRY",
	"ANTHROPIC_API_KEY",
	"ANTHROPIC_MODEL",
	"ANTHROPIC_DEFAULT_OPUS_MODEL",
	"ANTHROPIC_DEFAULT_SONNET_MODEL",
	"ANTHROPIC_DEFAULT_HAIKU_MODEL",
	"ANTHROPIC_SMALL_FAST_MODEL",
}

// Installed checks the native installer's and npm's default places only.
func (c ClaudeCode) Installed() bool {
	candidates := []string{}
	if c.Home != "" {
		candidates = append(candidates, filepath.Join(c.Home, ".local", "bin", "claude.exe"))
	}
	if c.AppData != "" {
		candidates = append(candidates, filepath.Join(c.AppData, "npm", "claude.cmd"))
	}
	for _, path := range candidates {
		if fileExists(path) {
			return true
		}
	}
	return false
}

func (c ClaudeCode) settingsPath() string {
	return filepath.Join(c.Home, ".claude", "settings.json")
}

// ConnectedWith reports whether the CLI already uses MyAutoAPI with key.
func (c ClaudeCode) ConnectedWith(key string) bool {
	settings, _, err := readJSONObject(c.settingsPath())
	if err != nil {
		return false
	}
	raw, ok := settings.get("env")
	if !ok {
		return false
	}
	env, err := parseJSONObject(raw)
	return err == nil && isMyAutoAPI(env.getString("ANTHROPIC_BASE_URL")) && env.getString("ANTHROPIC_AUTH_TOKEN") == key
}

// WriteClaudeCode points the CLI at MyAutoAPI through the env block of
// ~/.claude/settings.json. Everything else in the file stays.
func WriteClaudeCode(c ClaudeCode, key string, now time.Time) error {
	if c.Home == "" {
		return errors.New("找不到当前用户的文件夹")
	}
	settings, _, err := readJSONObject(c.settingsPath())
	if err != nil {
		return err
	}
	env := newJSONObject()
	if raw, ok := settings.get("env"); ok && strings.TrimSpace(string(raw)) != "null" {
		if env, err = parseJSONObject(raw); err != nil {
			return errors.New("settings.json 的 env 不是对象")
		}
	}
	for _, name := range claudeCodeModelOverrides {
		env.remove(name)
	}
	// A model the CLI was set to elsewhere (glm-5, a relay's look-alike
	// name) would keep being asked for; aliases and official IDs stay.
	if model := settings.getString("model"); model != "" && !claudeCodeModelKept(model) {
		settings.remove("model")
	}
	if err := env.set("ANTHROPIC_BASE_URL", BaseURL); err != nil {
		return err
	}
	if err := env.set("ANTHROPIC_AUTH_TOKEN", key); err != nil {
		return err
	}
	envData, err := env.encode()
	if err != nil {
		return err
	}
	if err := settings.set("env", json.RawMessage(envData)); err != nil {
		return err
	}
	data, err := settings.encode()
	if err != nil {
		return err
	}
	changes := []fileChange{{path: c.settingsPath(), data: data, backup: true}}
	if change, ok := c.skipOnboarding(); ok {
		changes = append(changes, change)
	}
	return writeFileGroup(changes, now)
}

// claudeCodeModelKept tells the models MyAutoAPI can serve: the CLI's
// aliases (opus, sonnet[1m], ...) and official Claude model IDs.
func claudeCodeModelKept(model string) bool {
	name := strings.TrimSuffix(strings.TrimSpace(model), "[1m]")
	switch name {
	case "default", "best", "opus", "sonnet", "haiku", "fable", "opusplan":
		return true
	}
	return claudeModelPattern.MatchString(name)
}

func (c ClaudeCode) statePath() string { return filepath.Join(c.Home, ".claude.json") }

// skipOnboarding marks the CLI's first-run setup as done in ~/.claude.json.
// A CLI that was never opened would otherwise start with the official login
// screen even though settings.json already holds the MyAutoAPI key. Nothing
// changes when the flag is already set or the file cannot be read.
func (c ClaudeCode) skipOnboarding() (fileChange, bool) {
	state, _, err := readJSONObject(c.statePath())
	if err != nil {
		return fileChange{}, false
	}
	if raw, ok := state.get("hasCompletedOnboarding"); ok && strings.TrimSpace(string(raw)) == "true" {
		return fileChange{}, false
	}
	if state.set("hasCompletedOnboarding", true) != nil {
		return fileChange{}, false
	}
	data, err := state.encode()
	if err != nil {
		return fileChange{}, false
	}
	return fileChange{path: c.statePath(), data: data, backup: true}, true
}

// RestoreClaudeCode takes the CLI off MyAutoAPI: when settings.json points
// at MyAutoAPI, ANTHROPIC_BASE_URL and ANTHROPIC_AUTH_TOKEN leave its env
// block (a backup is kept). Settings for any other service stay untouched.
// It reports whether anything changed.
func RestoreClaudeCode(c ClaudeCode, now time.Time) (bool, error) {
	if c.Home == "" {
		return false, nil
	}
	settings, exists, err := readJSONObject(c.settingsPath())
	if err != nil || !exists {
		return false, err
	}
	raw, ok := settings.get("env")
	if !ok {
		return false, nil
	}
	env, err := parseJSONObject(raw)
	if err != nil || !isMyAutoAPI(env.getString("ANTHROPIC_BASE_URL")) {
		return false, nil
	}
	env.remove("ANTHROPIC_BASE_URL")
	env.remove("ANTHROPIC_AUTH_TOKEN")
	envData, err := env.encode()
	if err != nil {
		return false, err
	}
	if err := settings.set("env", json.RawMessage(envData)); err != nil {
		return false, err
	}
	data, err := settings.encode()
	if err != nil {
		return false, err
	}
	return true, writeFileGroup([]fileChange{{path: c.settingsPath(), data: data, backup: true}}, now)
}

// CodexAppDetected guesses whether the ChatGPT/Codex app is on this
// computer: a Store package from OpenAI, or a Codex folder that some Codex
// already created. It only decides whether the window offers the download.
func CodexAppDetected(localAppData, codexHome string) bool {
	if codexHome != "" && isDir(codexHome) {
		return true
	}
	if localAppData == "" {
		return false
	}
	matches, _ := filepath.Glob(filepath.Join(localAppData, "Packages", "OpenAI.*"))
	for _, match := range matches {
		if isDir(match) {
			return true
		}
	}
	return false
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
