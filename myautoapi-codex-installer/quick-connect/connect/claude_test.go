package connect

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestClaudeDesktopModels(t *testing.T) {
	listed := []string{
		"gpt-6.1-sol",
		"claude-sonnet-5-5",
		"claude-opus-4-6",
		"claude-opus-5.5",            // dot instead of hyphen: a relay typo
		"Claude-Opus-5-5",            // wrong case
		"claude-opus-5-5[1m]",        // context marker
		"claude-opus-5-5-thinking",   // relay variant
		"anthropic/claude-opus-5-5",  // prefixed
		"claude-3-5-sonnet-20241022", // old naming
		"claude-opus-5",
		"claude-fable-5-1",
		"claude-haiku-4-5-20251001",
		"claude-haiku-5-5",
		"claude-opus-5-5",
		"claude-opus-5-5-20260301", // dated copy of a listed version
		"claude-sonnet-",
		"deepseek-v4",
		"claude-opus-5-5", // duplicate
		" claude-sonnet-4-6 ",
	}
	want := []string{
		"claude-fable-5-1",
		"claude-opus-5-5", "claude-opus-5", "claude-opus-4-6",
		"claude-sonnet-5-5", "claude-sonnet-4-6",
		"claude-haiku-5-5", "claude-haiku-4-5-20251001",
	}
	if got := ClaudeDesktopModels(listed); !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
	if got := ClaudeDesktopModels([]string{"claude-opus-5.5", "gpt-6.1-sol"}); len(got) != 0 {
		t.Fatalf("only look-alikes: got %q", got)
	}
}

// claudeFixture lays out a user's files as found on a real Windows machine
// with the Store build and a CC Switch profile already applied.
func claudeFixture(t *testing.T) ClaudeDesktop {
	t.Helper()
	local := filepath.Join(t.TempDir(), "Local")
	writeFile(t, filepath.Join(local, "Claude", "claude_desktop_config.json"), "{\n  \"deploymentMode\": \"3p\"\n}\n")
	writeFile(t, filepath.Join(local, "Packages", "Claude_pzs8sxrjxfjjc", "LocalCache", "Roaming", "Claude", "claude_desktop_config.json"),
		`{"preferences":{"remoteToolsDeviceName":"desktop-1","coworkWebSearchEnabled":true}}`)
	writeFile(t, filepath.Join(local, "Claude-3p", "claude_desktop_config.json"),
		"\ufeff"+`{"deploymentMode":"3p","coworkUserFilesPath":"C:\\Users\\a\\Claude","preferences":{"sidebarMode":"epitaxy"}}`)
	writeFile(t, filepath.Join(local, "Claude-3p", "configLibrary", "_meta.json"),
		`{"appliedId":"00000000-0000-4000-8000-000000157210","entries":[{"id":"00000000-0000-4000-8000-000000157210","name":"CC Switch"}]}`)
	writeFile(t, filepath.Join(local, "Claude-3p", "configLibrary", "00000000-0000-4000-8000-000000157210.json"),
		`{"inferenceProvider":"gateway","inferenceGatewayBaseUrl":"https://other.example","inferenceGatewayApiKey":"sk-other"}`)
	return ClaudeDesktop{LocalAppData: local}
}

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal([]byte(strings.TrimPrefix(readFile(t, path), "\ufeff")), &value); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return value
}

var testClaudeModels = []string{"claude-fable-5-1", "claude-opus-5-5", "claude-opus-4-6"}

func TestWriteClaudeDesktop(t *testing.T) {
	d := claudeFixture(t)
	if got := ReadClaudeStatus(d); got.State != ClaudeOther {
		t.Fatalf("before: %+v", got)
	}
	if err := WriteClaudeDesktop(d, fixtureKey, testClaudeModels, fixedTime); err != nil {
		t.Fatal(err)
	}

	threeP := readJSON(t, filepath.Join(d.LocalAppData, "Claude-3p", "claude_desktop_config.json"))
	if threeP["deploymentMode"] != "3p" || threeP["coworkUserFilesPath"] != `C:\Users\a\Claude` ||
		threeP["preferences"].(map[string]any)["sidebarMode"] != "epitaxy" {
		t.Fatalf("Claude-3p config: %v", threeP)
	}
	roaming := readJSON(t, filepath.Join(d.LocalAppData, "Packages", "Claude_pzs8sxrjxfjjc", "LocalCache", "Roaming", "Claude", "claude_desktop_config.json"))
	if roaming["deploymentMode"] != "3p" || roaming["preferences"].(map[string]any)["remoteToolsDeviceName"] != "desktop-1" {
		t.Fatalf("package config: %v", roaming)
	}
	// Existing keys keep their order; the new one goes last.
	raw := readFile(t, filepath.Join(d.LocalAppData, "Packages", "Claude_pzs8sxrjxfjjc", "LocalCache", "Roaming", "Claude", "claude_desktop_config.json"))
	if strings.Index(raw, "preferences") > strings.Index(raw, "deploymentMode") {
		t.Fatalf("key order changed:\n%s", raw)
	}
	if readJSON(t, filepath.Join(d.LocalAppData, "Claude", "claude_desktop_config.json"))["deploymentMode"] != "3p" {
		t.Fatal("Local\\Claude config")
	}

	meta := readJSON(t, d.metaPath())
	entries := meta["entries"].([]any)
	if meta["appliedId"] != ClaudeProfileID || len(entries) != 2 ||
		entries[0].(map[string]any)["name"] != "CC Switch" || entries[1].(map[string]any)["id"] != ClaudeProfileID {
		t.Fatalf("meta: %v", meta)
	}
	profile := readJSON(t, d.profilePath())
	want := map[string]any{
		"coworkEgressAllowedHosts":     []any{"*"},
		"disableDeploymentModeChooser": true,
		"inferenceProvider":            "gateway",
		"inferenceGatewayBaseUrl":      BaseURL,
		"inferenceGatewayAuthScheme":   "bearer",
		"inferenceGatewayApiKey":       fixtureKey,
		"modelDiscoveryEnabled":        false,
		"inferenceModels":              []any{"claude-fable-5-1", "claude-opus-5-5", "claude-opus-4-6"},
	}
	if !reflect.DeepEqual(profile, want) {
		t.Fatalf("profile:\n%v\nwant\n%v", profile, want)
	}
	// CC Switch's own profile is untouched.
	if readJSON(t, filepath.Join(d.libraryDir(), "00000000-0000-4000-8000-000000157210.json"))["inferenceGatewayApiKey"] != "sk-other" {
		t.Fatal("CC Switch profile changed")
	}
	// Config files and _meta.json are backed up; MyAutoAPI's profile is not
	// (it is new, and later copies would only multiply the key on disk).
	if got := listBackups(t, filepath.Join(d.LocalAppData, "Claude-3p"), "claude_desktop_config.json.bak."); len(got) != 1 {
		t.Fatalf("Claude-3p backups: %v", got)
	}
	if got := listBackups(t, d.libraryDir(), "_meta.json.bak."); len(got) != 1 {
		t.Fatalf("meta backups: %v", got)
	}
	if got := listBackups(t, d.libraryDir(), ClaudeProfileID+".json.bak."); len(got) != 0 {
		t.Fatalf("profile backups: %v", got)
	}

	status := ReadClaudeStatus(d)
	if status.State != ClaudeMyAutoAPI || status.Key != MaskKey(fixtureKey) || status.Models != 3 {
		t.Fatalf("after: %+v", status)
	}

	// Writing again keeps one entry and settings the app added to the profile.
	profileObject, _, _ := readJSONObject(d.profilePath())
	_ = profileObject.set("coworkEgressAllowedHosts", []string{"example.com"})
	_ = profileObject.set("someSettingFromTheApp", 1)
	data, _ := profileObject.encode()
	writeFile(t, d.profilePath(), string(data))
	if err := WriteClaudeDesktop(d, fixtureKey, []string{"claude-sonnet-5-5"}, fixedTime); err != nil {
		t.Fatal(err)
	}
	profile = readJSON(t, d.profilePath())
	if profile["someSettingFromTheApp"] != float64(1) || !reflect.DeepEqual(profile["coworkEgressAllowedHosts"], []any{"example.com"}) ||
		!reflect.DeepEqual(profile["inferenceModels"], []any{"claude-sonnet-5-5"}) {
		t.Fatalf("second write: %v", profile)
	}
	if entries := readJSON(t, d.metaPath())["entries"].([]any); len(entries) != 2 {
		t.Fatalf("second write entries: %v", entries)
	}
}

func TestWriteClaudeDesktopFreshInstall(t *testing.T) {
	local := t.TempDir()
	if err := os.MkdirAll(filepath.Join(local, "Packages", "Claude_pzs8sxrjxfjjc"), 0o700); err != nil {
		t.Fatal(err)
	}
	d := ClaudeDesktop{LocalAppData: local}
	if got := ReadClaudeStatus(d); got.State != ClaudeNone {
		t.Fatalf("fresh: %+v", got)
	}
	if err := WriteClaudeDesktop(d, fixtureKey, testClaudeModels, fixedTime); err != nil {
		t.Fatal(err)
	}
	meta := readJSON(t, d.metaPath())
	if meta["appliedId"] != ClaudeProfileID || len(meta["entries"].([]any)) != 1 {
		t.Fatalf("meta: %v", meta)
	}
	for _, path := range d.configPaths() {
		if readJSON(t, path)["deploymentMode"] != "3p" {
			t.Fatalf("%s not 3p", path)
		}
	}
	if got := ReadClaudeStatus(d); got.State != ClaudeMyAutoAPI {
		t.Fatalf("after: %+v", got)
	}
}

func TestWriteClaudeDesktopRefuses(t *testing.T) {
	notInstalled := ClaudeDesktop{LocalAppData: t.TempDir()}
	if ReadClaudeStatus(notInstalled).State != ClaudeNotInstalled {
		t.Fatal("status of a computer without Claude")
	}
	if err := WriteClaudeDesktop(notInstalled, fixtureKey, testClaudeModels, fixedTime); err == nil {
		t.Fatal("wrote without Claude installed")
	}
	if entries, _ := os.ReadDir(notInstalled.LocalAppData); len(entries) != 0 {
		t.Fatalf("created %v", entries)
	}

	d := claudeFixture(t)
	if err := WriteClaudeDesktop(d, fixtureKey, nil, fixedTime); err == nil {
		t.Fatal("wrote an empty model list")
	}
	// A damaged file is reported, never overwritten.
	broken := filepath.Join(d.LocalAppData, "Claude-3p", "claude_desktop_config.json")
	writeFile(t, broken, `{"deploymentMode": `)
	before := snapshot(t, d.LocalAppData)
	if err := WriteClaudeDesktop(d, fixtureKey, testClaudeModels, fixedTime); err == nil {
		t.Fatal("wrote over a damaged config")
	}
	if after := snapshot(t, d.LocalAppData); !reflect.DeepEqual(before, after) {
		t.Fatal("files changed after a refused write")
	}
}

func TestWriteClaudeDesktopRollsBack(t *testing.T) {
	d := claudeFixture(t)
	before := snapshot(t, d.LocalAppData)
	calls := 0
	renameFile = func(from, to string) error {
		calls++
		if calls == 3 {
			return errors.New("拒绝访问。")
		}
		return defaultRename(from, to)
	}
	defer func() { renameFile = defaultRename }()
	if err := WriteClaudeDesktop(d, fixtureKey, testClaudeModels, fixedTime); err == nil {
		t.Fatal("no error")
	}
	renameFile = defaultRename
	after := snapshot(t, d.LocalAppData)
	for path, content := range before {
		if after[path] != content {
			t.Fatalf("%s not rolled back:\n%s", path, after[path])
		}
	}
	for path := range after {
		if _, ok := before[path]; !ok && !strings.Contains(path, ".bak.") {
			t.Fatalf("left behind %s", path)
		}
	}
}

// snapshot reads every file under root.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		files[path] = string(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestRestoreClaudeDesktop(t *testing.T) {
	d := claudeFixture(t)
	if err := WriteClaudeDesktop(d, fixtureKey, testClaudeModels, fixedTime); err != nil {
		t.Fatal(err)
	}
	if err := RestoreClaudeDesktop(d, fixedTime); err != nil {
		t.Fatal(err)
	}
	for _, path := range d.configPaths() {
		if readJSON(t, path)["deploymentMode"] != "1p" {
			t.Fatalf("%s not 1p", path)
		}
	}
	if readJSON(t, filepath.Join(d.LocalAppData, "Claude-3p", "claude_desktop_config.json"))["coworkUserFilesPath"] == nil {
		t.Fatal("other settings lost")
	}
	meta := readJSON(t, d.metaPath())
	if meta["appliedId"] != "00000000-0000-4000-8000-000000157210" || len(meta["entries"].([]any)) != 1 {
		t.Fatalf("meta: %v", meta)
	}
	if fileExists(d.profilePath()) {
		t.Fatal("MyAutoAPI profile with the key is still there")
	}
	if got := ReadClaudeStatus(d); got.State != ClaudeNone {
		t.Fatalf("after restore: %+v", got)
	}

	// Another service's profile applied: only MyAutoAPI's leftovers go, the
	// mode stays third-party.
	other := claudeFixture(t)
	if err := WriteClaudeDesktop(other, fixtureKey, testClaudeModels, fixedTime); err != nil {
		t.Fatal(err)
	}
	metaObject, _, _ := readJSONObject(other.metaPath())
	_ = metaObject.set("appliedId", "00000000-0000-4000-8000-000000157210")
	data, _ := metaObject.encode()
	writeFile(t, other.metaPath(), string(data))
	if err := RestoreClaudeDesktop(other, fixedTime); err != nil {
		t.Fatal(err)
	}
	for _, path := range other.configPaths() {
		if readJSON(t, path)["deploymentMode"] != "3p" {
			t.Fatalf("%s switched off another service", path)
		}
	}
	if meta := readJSON(t, other.metaPath()); meta["appliedId"] != "00000000-0000-4000-8000-000000157210" || len(meta["entries"].([]any)) != 1 || fileExists(other.profilePath()) {
		t.Fatalf("leftovers: %v", meta)
	}

	// Without another profile, appliedId goes away.
	fresh := t.TempDir()
	_ = os.MkdirAll(filepath.Join(fresh, "Packages", "Claude_pzs8sxrjxfjjc"), 0o700)
	only := ClaudeDesktop{LocalAppData: fresh}
	if err := WriteClaudeDesktop(only, fixtureKey, testClaudeModels, fixedTime); err != nil {
		t.Fatal(err)
	}
	if err := RestoreClaudeDesktop(only, fixedTime); err != nil {
		t.Fatal(err)
	}
	if meta := readJSON(t, only.metaPath()); meta["appliedId"] != nil || len(meta["entries"].([]any)) != 0 {
		t.Fatalf("meta: %v", meta)
	}

	// Nothing there: nothing created.
	empty := ClaudeDesktop{LocalAppData: t.TempDir()}
	if err := RestoreClaudeDesktop(empty, fixedTime); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(empty.LocalAppData); len(entries) != 0 {
		t.Fatalf("created %v", entries)
	}
}

func TestClaudeCode(t *testing.T) {
	home, appData := t.TempDir(), t.TempDir()
	c := ClaudeCode{Home: home, AppData: appData}
	if c.Installed() {
		t.Fatal("nothing installed")
	}
	writeFile(t, filepath.Join(appData, "npm", "claude.cmd"), "@echo off")
	if !c.Installed() {
		t.Fatal("npm install not found")
	}
	native := ClaudeCode{Home: home}
	writeFile(t, filepath.Join(home, ".local", "bin", "claude.exe"), "MZ")
	if !native.Installed() {
		t.Fatal("native install not found")
	}

	settings := filepath.Join(home, ".claude", "settings.json")
	writeFile(t, settings, `{"model":"opus","env":{"DISABLE_TELEMETRY":"1","ANTHROPIC_API_KEY":"sk-ant-old","ANTHROPIC_DEFAULT_OPUS_MODEL":"glm-5","CLAUDE_CODE_USE_BEDROCK":"1"},"permissions":{"allow":["Bash(ls)"]}}`)
	if c.ConnectedWith(fixtureKey) {
		t.Fatal("connected before writing")
	}
	if err := WriteClaudeCode(c, fixtureKey, fixedTime); err != nil {
		t.Fatal(err)
	}
	got := readJSON(t, settings)
	want := map[string]any{
		"model": "opus",
		"env": map[string]any{
			"DISABLE_TELEMETRY":    "1",
			"ANTHROPIC_BASE_URL":   BaseURL,
			"ANTHROPIC_AUTH_TOKEN": fixtureKey,
		},
		"permissions": map[string]any{"allow": []any{"Bash(ls)"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("settings:\n%v\nwant\n%v", got, want)
	}
	if !c.ConnectedWith(fixtureKey) || c.ConnectedWith("sk-other") {
		t.Fatal("ConnectedWith")
	}
	if backups := listBackups(t, filepath.Join(home, ".claude"), "settings.json.bak."); len(backups) != 1 {
		t.Fatalf("backups: %v", backups)
	}

	// A model MyAutoAPI cannot serve is dropped; aliases and official IDs stay.
	for model, kept := range map[string]bool{"glm-5": false, "claude-opus-5.5": false, "sonnet[1m]": true, "claude-opus-5-5": true} {
		writeFile(t, settings, `{"model":"`+model+`"}`)
		if err := WriteClaudeCode(c, fixtureKey, fixedTime); err != nil {
			t.Fatal(err)
		}
		if _, has := readJSON(t, settings)["model"]; has != kept {
			t.Errorf("model %q kept=%v", model, has)
		}
	}

	// A CLI already opened keeps its state; the first-run setup is marked done.
	state := filepath.Join(home, ".claude.json")
	writeFile(t, state, `{"numStartups":3,"projects":{"C:/a":{"allowedTools":[]}}}`)
	if err := WriteClaudeCode(c, fixtureKey, fixedTime); err != nil {
		t.Fatal(err)
	}
	wantState := map[string]any{"numStartups": 3.0, "projects": map[string]any{"C:/a": map[string]any{"allowedTools": []any{}}}, "hasCompletedOnboarding": true}
	if got := readJSON(t, state); !reflect.DeepEqual(got, wantState) {
		t.Fatalf("state: %v", got)
	}
	// Already marked: the file is not rewritten.
	if err := WriteClaudeCode(c, fixtureKey, fixedTime.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if backups := listBackups(t, home, ".claude.json.bak."); len(backups) != 1 {
		t.Fatalf("state backups: %v", backups)
	}

	// No settings file yet.
	empty := ClaudeCode{Home: t.TempDir()}
	if err := WriteClaudeCode(empty, fixtureKey, fixedTime); err != nil {
		t.Fatal(err)
	}
	if !empty.ConnectedWith(fixtureKey) {
		t.Fatal("fresh settings")
	}
	if got := readJSON(t, filepath.Join(empty.Home, ".claude.json")); got["hasCompletedOnboarding"] != true {
		t.Fatalf("fresh state: %v", got)
	}

	// A broken ~/.claude.json is left alone and does not fail the connection.
	broken := ClaudeCode{Home: t.TempDir()}
	writeFile(t, filepath.Join(broken.Home, ".claude.json"), "{oops")
	if err := WriteClaudeCode(broken, fixtureKey, fixedTime); err != nil || !broken.ConnectedWith(fixtureKey) {
		t.Fatalf("broken state: %v", err)
	}
}

func TestIsClaudeDesktopProcess(t *testing.T) {
	cases := map[string]bool{
		`C:\Program Files\WindowsApps\Claude_1.24012.9.0_x64__pzs8sxrjxfjjc\app\claude.exe`: true,
		`C:\PROGRAM FILES\WINDOWSAPPS\CLAUDE_1.0_X64__PZS8SXRJXFJJC\APP\CLAUDE.EXE`:         true,
		`C:\Users\a\.local\bin\claude.exe`:                                                  false,
		`C:\Users\a\AppData\Roaming\npm\node_modules\@anthropic-ai\claude-code\claude.exe`:  false,
		`C:\Program Files\WindowsApps\Claude_1.0_x64__pzs8sxrjxfjjc\app\resources\node.exe`: false,
		``: false,
	}
	for path, want := range cases {
		if got := IsClaudeDesktopProcess(path); got != want {
			t.Errorf("%q: got %v", path, got)
		}
	}
}

func TestCodexAppDetected(t *testing.T) {
	local := t.TempDir()
	home := filepath.Join(t.TempDir(), ".codex")
	if CodexAppDetected(local, home) {
		t.Fatal("nothing there")
	}
	_ = os.MkdirAll(filepath.Join(local, "Packages", "OpenAI.ChatGPT-Desktop_2p2nqsd0c76g0"), 0o700)
	if !CodexAppDetected(local, home) {
		t.Fatal("OpenAI package")
	}
	_ = os.MkdirAll(home, 0o700)
	if !CodexAppDetected(t.TempDir(), home) {
		t.Fatal("Codex folder")
	}
}

const claudeModelsAnswer = `{"object":"list","data":[{"id":"claude-opus-5.5"},{"id":"claude-opus-5-5"},{"id":"claude-fable-5-1"},{"id":"gpt-6.1-sol"}]}`

func newClaudeService(t *testing.T, usage string, models string) (*Service, *int) {
	t.Helper()
	server := fakeGateway(t, 200, usage, 200, models)
	stops := 0
	local := t.TempDir()
	if err := os.MkdirAll(filepath.Join(local, "Packages", "Claude_pzs8sxrjxfjjc"), 0o700); err != nil {
		t.Fatal(err)
	}
	service := &Service{
		CodexHome: filepath.Join(t.TempDir(), ".codex"),
		BaseURL:   server.URL,
		Client:    NewHTTPClient(),
		Now:       func() time.Time { return fixedTime },
		Claude:    ClaudeDesktop{LocalAppData: local},
		ClaudeCLI: ClaudeCode{Home: t.TempDir(), AppData: t.TempDir()},
	}
	service.StopClaude = func() ClaudeStopResult {
		stops++
		// Claude must be closed before anything is written: a running
		// app saves its config on exit.
		if fileExists(service.Claude.profilePath()) && service.Claude.LocalAppData != "" {
			if status := ReadClaudeStatus(service.Claude); status.State == ClaudeMyAutoAPI && stops == 1 {
				t.Error("Claude stopped after the profile was written")
			}
		}
		return ClaudeStopResult{Found: true}
	}
	return service, &stops
}

func TestConnectClaude(t *testing.T) {
	service, stops := newClaudeService(t, `{"isValid":true,"remaining":3}`, claudeModelsAnswer)
	result := service.ConnectClaude(context.Background(), fixtureKey, "工作")
	want := "已连接 MyAutoAPI ✓ 余额 $3.00，写入了 2 个 Claude 模型。" + MessageClaudeClosed
	if result.Tone != ToneSuccess || result.Message != want {
		t.Fatalf("got %s %q", result.Tone, result.Message)
	}
	if *stops != 1 || result.Claude.State != ClaudeMyAutoAPI || result.Claude.Models != 2 || result.OfferCLI {
		t.Fatalf("result: %+v stops %d", result, *stops)
	}
	if got := readJSON(t, service.Claude.profilePath())["inferenceModels"]; !reflect.DeepEqual(got, []any{"claude-fable-5-1", "claude-opus-5-5"}) {
		t.Fatalf("models: %v", got)
	}

	// The CLI is offered when installed and not on this key yet.
	writeFile(t, filepath.Join(service.ClaudeCLI.Home, ".local", "bin", "claude.exe"), "MZ")
	if result := service.ConnectClaude(context.Background(), fixtureKey, ""); !result.OfferCLI {
		t.Fatal("CLI not offered")
	}
	if result := service.ConnectClaudeCode(fixtureKey); result.Tone != ToneSuccess || result.Message != MessageClaudeCodeConnected {
		t.Fatalf("CLI: %+v", result)
	}
	if result := service.ConnectClaude(context.Background(), fixtureKey, ""); result.OfferCLI {
		t.Fatal("CLI offered again for the same key")
	}

	result = service.RestoreClaude()
	if result.Tone != ToneSuccess || result.Claude.State != ClaudeNone || !strings.Contains(result.Message, MessageClaudeCodeRestored) {
		t.Fatalf("restore: %+v", result)
	}
	settings := readJSON(t, filepath.Join(service.ClaudeCLI.Home, ".claude", "settings.json"))
	if env := settings["env"].(map[string]any); len(env) != 0 {
		t.Fatalf("CLI env after restore: %v", env)
	}
	// A CLI on another service is left alone.
	writeFile(t, filepath.Join(service.ClaudeCLI.Home, ".claude", "settings.json"), `{"env":{"ANTHROPIC_BASE_URL":"https://other.example","ANTHROPIC_AUTH_TOKEN":"x"}}`)
	if changed, err := RestoreClaudeCode(service.ClaudeCLI, fixedTime); changed || err != nil {
		t.Fatalf("other service: %v %v", changed, err)
	}
}

func TestConnectClaudeRefusals(t *testing.T) {
	cases := []struct {
		name    string
		usage   string
		models  string
		key     string
		message string
	}{
		{"no Claude model", `{"isValid":true}`, textModels, fixtureKey, MessageClaudeNoModel},
		{"only look-alike models", `{"isValid":true}`, `{"data":[{"id":"claude-opus-5.5"}]}`, fixtureKey, MessageClaudeNoModel},
		{"disabled", `{"isValid":false}`, claudeModelsAnswer, fixtureKey, MessageClaudeDisabled},
		{"rejected", `{"isValid":true}`, claudeModelsAnswer, "sk-wrong", MessageClaudeRejected},
		{"empty", `{"isValid":true}`, claudeModelsAnswer, " ", MessageClaudeEmpty},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			service, stops := newClaudeService(t, c.usage, c.models)
			result := service.ConnectClaude(context.Background(), c.key, "")
			if result.Tone != ToneError || result.Message != c.message || *stops != 0 {
				t.Fatalf("got %s %q, stops %d", result.Tone, result.Message, *stops)
			}
			if fileExists(service.Claude.metaPath()) || fileExists(service.Claude.profilePath()) {
				t.Fatal("wrote files")
			}
		})
	}

	// sub2api refuses /v1/models while the balance is $0.
	server := fakeGateway(t, 200, `{"isValid":true,"remaining":0}`, 403, `{"code":"INSUFFICIENT_BALANCE"}`)
	broke, stops := newClaudeService(t, `{}`, `{}`)
	broke.BaseURL = server.URL
	if result := broke.ConnectClaude(context.Background(), fixtureKey, ""); result.Tone != ToneError || result.Message != MessageClaudeNeedBalance || *stops != 0 || fileExists(broke.Claude.profilePath()) {
		t.Fatalf("zero balance: %+v stops %d", result, *stops)
	}

	service, _ := newClaudeService(t, `{"isValid":true}`, claudeModelsAnswer)
	service.Claude = ClaudeDesktop{LocalAppData: t.TempDir()}
	if result := service.ConnectClaude(context.Background(), fixtureKey, ""); result.Message != MessageClaudeNotInstalled || result.Claude.State != ClaudeNotInstalled {
		t.Fatalf("not installed: %+v", result)
	}
}
