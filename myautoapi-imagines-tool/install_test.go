package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNativeInstallRepeatedAndUninstall(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	authPath := filepath.Join(root, "auth.json")
	authContent := []byte(`{"OPENAI_API_KEY":"fake-key-not-real"}`)
	if err := os.WriteFile(authPath, authContent, 0o666); err != nil {
		t.Fatal(err)
	}
	authHash := sha256.Sum256(authContent)

	var output bytes.Buffer
	if err := installWithTestKey(&output); err != nil {
		t.Fatalf("install failed: %v", err)
	}
	layout := buildInstallLayout(root)
	for _, path := range []string{
		layout.binaryPath,
		layout.versionPath,
		layout.releasePath,
		layout.statePath,
		filepath.Join(layout.toolDir, keyFileName),
		filepath.Join(layout.toolDir, keySourceFileName),
		layout.skillPath,
		layout.agentPath,
		layout.agentsPath,
	} {
		if !regularFileExists(path) {
			t.Errorf("managed file missing: %s", path)
		}
	}
	keyInfo, err := os.Stat(filepath.Join(layout.toolDir, keyFileName))
	if err != nil || keyInfo.Size() == 0 {
		t.Fatal("dedicated image key was not saved")
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(layout.binaryPath)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0o111 == 0 {
			t.Fatal("installed native tool is not executable")
		}
	}
	if strings.Contains(output.String(), "fake-key-not-real") {
		t.Fatal("installer output leaked auth key")
	}

	output.Reset()
	if err := installWithTestKey(&output); err != nil {
		t.Fatalf("second install failed: %v", err)
	}
	if !strings.Contains(output.String(), "请输入专门画图分组 Key") {
		t.Fatalf("second install was not idempotent: %s", output.String())
	}

	output.Reset()
	if err := uninstall(&output); err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}
	for _, path := range []string{layout.binaryPath, layout.statePath, layout.skillPath, layout.agentPath} {
		if pathExists(path) {
			t.Errorf("managed file remained after uninstall: %s", path)
		}
	}
	currentAuth, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(currentAuth) != authHash {
		t.Fatal("auth.json changed during install or uninstall")
	}
}

func TestNativeInstallUpgradesV073ReplacesExistingKey(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	if err := installWithTestKey(&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	layout := buildInstallLayout(root)
	keyPath := filepath.Join(layout.toolDir, keyFileName)
	keyBefore := mustReadFile(t, keyPath)
	stateBytes, err := os.ReadFile(layout.statePath)
	if err != nil {
		t.Fatal(err)
	}
	state := installState{}
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		t.Fatal(err)
	}
	state.Version = "0.7.3"
	state.Release = "V0.7.3"
	state.Files["VERSION"] = sha256Hex([]byte("0.7.3\n"))
	state.Files["RELEASE"] = sha256Hex([]byte("V0.7.3"))
	if err := os.WriteFile(layout.versionPath, []byte("0.7.3\n"), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.releasePath, []byte("V0.7.3"), 0o666); err != nil {
		t.Fatal(err)
	}
	updated, _ := json.MarshalIndent(state, "", "  ")
	if err := os.WriteFile(layout.statePath, append(updated, '\n'), 0o666); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := installWithInput(&output, strings.NewReader("upgrade-replacement-key\n")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "升级完成：V0.8.1") {
		t.Fatalf("version upgrade output missing: %s", output.String())
	}
	if !strings.Contains(output.String(), "请输入专门画图分组 Key") {
		t.Fatalf("version upgrade unexpectedly requested the dedicated key: %s", output.String())
	}
	if got := mustReadFile(t, keyPath); bytes.Equal(got, keyBefore) {
		t.Fatal("release update did not replace the dedicated key")
	}
	if got := mustReadFile(t, layout.releasePath); string(got) != releaseRevision {
		t.Fatalf("release file was not updated: %q", got)
	}
}

func TestNativeInstallRepairsLegacyPythonTool(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	if err := installWithTestKey(&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	layout := buildInstallLayout(root)
	stateContent, err := os.ReadFile(layout.statePath)
	if err != nil {
		t.Fatal(err)
	}
	state := installState{}
	if err := json.Unmarshal(stateContent, &state); err != nil {
		t.Fatal(err)
	}
	legacyPath := filepath.Join(layout.toolDir, legacyPythonName)
	legacyContent := []byte("print('legacy')\n")
	if err := os.WriteFile(legacyPath, legacyContent, 0o666); err != nil {
		t.Fatal(err)
	}
	state.Files["tool/"+legacyPythonName] = sha256Hex(legacyContent)
	updatedState, _ := json.MarshalIndent(state, "", "  ")
	if err := os.WriteFile(layout.statePath, append(updatedState, '\n'), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(layout.binaryPath); err != nil {
		t.Fatal(err)
	}

	if err := installWithTestKey(&bytes.Buffer{}); err != nil {
		t.Fatalf("repair failed: %v", err)
	}
	if pathExists(legacyPath) {
		t.Fatal("legacy Python tool was not removed")
	}
	if !regularFileExists(layout.binaryPath) {
		t.Fatal("native tool was not restored")
	}
}

func TestNativeStandaloneInstallPromptsForDedicatedKey(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	var output bytes.Buffer
	if err := installWithInput(&output, strings.NewReader("\nnew-dedicated-key\n")); err != nil {
		t.Fatal(err)
	}
	layout := buildInstallLayout(root)
	if !nonEmptyRegularFile(filepath.Join(layout.toolDir, keyFileName)) {
		t.Fatal("dedicated image key was not saved")
	}
	if !strings.Contains(output.String(), "专门画图分组") || !strings.Contains(output.String(), "Key 不能为空") {
		t.Fatalf("prompt did not explain the key check: %s", output.String())
	}
	if strings.Contains(output.String(), "new-dedicated-key") {
		t.Fatal("installer output echoed the entered key")
	}
}

func TestNativeIntegratedInstallDefaultsToYesAndWarnsForDuplicateKey(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	if err := os.MkdirAll(root, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "auth.json"), []byte(`{"OPENAI_API_KEY":"same-key"}`), 0o666); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	input := strings.NewReader("\nsame-key\n\n")
	err := installWithOptions(&output, input, installOptions{integrated: true, compareProgrammingKey: true})
	if err != nil {
		t.Fatal(err)
	}
	layout := buildInstallLayout(root)
	if !nonEmptyRegularFile(filepath.Join(layout.toolDir, keyFileName)) {
		t.Fatal("integrated install did not save the dedicated key")
	}
	if !strings.Contains(output.String(), "完全相同") || !strings.Contains(output.String(), "直接回车继续") {
		t.Fatalf("duplicate-key warning missing: %s", output.String())
	}
}

func TestNativeIntegratedInstallExplicitNoExitsWithoutWrites(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	var output bytes.Buffer
	if err := installWithOptions(&output, strings.NewReader("n\n"), installOptions{integrated: true, compareProgrammingKey: true}); err != nil {
		t.Fatal(err)
	}
	if pathExists(filepath.Join(root, "tools")) || pathExists(filepath.Join(root, "skills")) {
		t.Fatal("explicit no left installation directories behind")
	}
	if !strings.Contains(output.String(), "已跳过生图工具安装") {
		t.Fatalf("skip message missing: %s", output.String())
	}
}

func TestNativeInstallUpgradesV02StateAndRequiresDedicatedKey(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	if err := installWithTestKey(&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	layout := buildInstallLayout(root)
	stateBytes, err := os.ReadFile(layout.statePath)
	if err != nil {
		t.Fatal(err)
	}
	state := installState{}
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		t.Fatal(err)
	}
	state.Version = "0.2.0"
	delete(state.Files, "tool/"+keySourceFileName)
	updated, _ := json.MarshalIndent(state, "", "  ")
	if err := os.WriteFile(layout.statePath, append(updated, '\n'), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(layout.toolDir, keyFileName)); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := installWithInput(&output, strings.NewReader("new-dedicated-key\n")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "升级完成：V0.8.1") {
		t.Fatalf("upgrade output missing: %s", output.String())
	}
	if !nonEmptyRegularFile(filepath.Join(layout.toolDir, keyFileName)) {
		t.Fatal("upgrade did not save a dedicated key")
	}
	if !strings.Contains(string(mustReadFile(t, layout.versionPath)), "0.8.1") {
		t.Fatal("upgrade did not update VERSION")
	}
}

func TestNativeInstallUpgradesV075ReplacesExistingKey(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	if err := installWithTestKey(&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	layout := buildInstallLayout(root)
	stateBytes, err := os.ReadFile(layout.statePath)
	if err != nil {
		t.Fatal(err)
	}
	state := installState{}
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		t.Fatal(err)
	}
	state.Version = "0.7.5"
	state.Release = "V0.7.5"
	state.Files["VERSION"] = sha256Hex([]byte("0.7.5\n"))
	state.Files["RELEASE"] = sha256Hex([]byte("V0.7.5"))
	if err := os.WriteFile(layout.versionPath, []byte("0.7.5\n"), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.releasePath, []byte("V0.7.5"), 0o666); err != nil {
		t.Fatal(err)
	}
	updated, _ := json.MarshalIndent(state, "", "  ")
	if err := os.WriteFile(layout.statePath, append(updated, '\n'), 0o666); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := installWithInput(&output, strings.NewReader("upgrade-replacement-key\n")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "升级完成：V0.8.1") {
		t.Fatalf("version upgrade output missing: %s", output.String())
	}
	if !strings.Contains(string(mustReadFile(t, layout.versionPath)), "0.8.1") {
		t.Fatal("upgrade did not update VERSION")
	}
}

func TestNativeInstallUpgradesV076ReplacesExistingKey(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	if err := installWithTestKey(&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	layout := buildInstallLayout(root)
	keyPath := filepath.Join(layout.toolDir, keyFileName)
	keyBefore := mustReadFile(t, keyPath)
	stateBytes, err := os.ReadFile(layout.statePath)
	if err != nil {
		t.Fatal(err)
	}
	state := installState{}
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		t.Fatal(err)
	}
	state.Version = "0.7.6"
	state.Release = "V0.7.6"
	state.Files["VERSION"] = sha256Hex([]byte("0.7.6\n"))
	state.Files["RELEASE"] = sha256Hex([]byte("V0.7.6"))
	if err := os.WriteFile(layout.versionPath, []byte("0.7.6\n"), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.releasePath, []byte("V0.7.6"), 0o666); err != nil {
		t.Fatal(err)
	}
	updated, _ := json.MarshalIndent(state, "", "  ")
	if err := os.WriteFile(layout.statePath, append(updated, '\n'), 0o666); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := installWithInput(&output, strings.NewReader("upgrade-replacement-key\n")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "升级完成：V0.8.1") {
		t.Fatalf("version upgrade output missing: %s", output.String())
	}
	if !strings.Contains(output.String(), "请输入专门画图分组 Key") {
		t.Fatalf("version upgrade unexpectedly requested the dedicated key: %s", output.String())
	}
	if got := mustReadFile(t, keyPath); bytes.Equal(got, keyBefore) {
		t.Fatal("version upgrade did not replace the dedicated key")
	}
}

func TestNativeInstallRollsBackDedicatedKeyWithManagedFiles(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	var output bytes.Buffer
	err := installWithOptions(&output, strings.NewReader("rollback-key\n"), installOptions{
		failureHook: func(stage string) error {
			if stage == "key-written" {
				return errors.New("simulated interruption")
			}
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "simulated interruption") {
		t.Fatalf("expected rollback failure, got %v", err)
	}
	layout := buildInstallLayout(root)
	if pathExists(layout.toolDir) || pathExists(layout.skillDir) || pathExists(layout.agentsPath) {
		t.Fatal("managed files remained after failed first install")
	}
	if pathExists(filepath.Join(layout.toolDir, keyFileName)) {
		t.Fatal("dedicated key remained after failed first install")
	}
}

func TestNativeInstallStopsForOverrideAndSymlink(t *testing.T) {
	t.Run("override", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("CODEX_HOME", root)
		if err := os.WriteFile(filepath.Join(root, "AGENTS.override.md"), []byte("# override\n"), 0o666); err != nil {
			t.Fatal(err)
		}
		if err := installWithTestKey(&bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "AGENTS.override.md") {
			t.Fatalf("expected override error, got %v", err)
		}
	})

	t.Run("tools symlink", func(t *testing.T) {
		root := t.TempDir()
		outside := t.TempDir()
		t.Setenv("CODEX_HOME", root)
		if err := os.Symlink(outside, filepath.Join(root, "tools")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if err := installWithTestKey(&bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "符号链接") {
			t.Fatalf("expected symlink error, got %v", err)
		}
		entries, err := os.ReadDir(outside)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Fatal("installer wrote through tools symlink")
		}
	})
}

func TestNativeUninstallPreservesUnownedLegacyFilename(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	if err := installWithTestKey(&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	layout := buildInstallLayout(root)
	legacyPath := filepath.Join(layout.toolDir, legacyPythonName)
	if err := os.WriteFile(legacyPath, []byte("user-owned\n"), 0o666); err != nil {
		t.Fatal(err)
	}

	err := uninstall(&bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "未知文件") {
		t.Fatalf("expected unknown-file stop, got %v", err)
	}
	content, readErr := os.ReadFile(legacyPath)
	if readErr != nil || string(content) != "user-owned\n" {
		t.Fatal("uninstaller changed an unowned legacy filename")
	}
	if !regularFileExists(layout.binaryPath) || !regularFileExists(layout.statePath) {
		t.Fatal("uninstaller partially removed managed files")
	}
}

func TestNativeUninstallPreservesModifiedManagedFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	if err := installWithTestKey(&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	layout := buildInstallLayout(root)
	if err := os.WriteFile(layout.skillPath, []byte("user modified\n"), 0o666); err != nil {
		t.Fatal(err)
	}

	err := uninstall(&bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "内容已变化") {
		t.Fatalf("expected modified-file stop, got %v", err)
	}
	content, readErr := os.ReadFile(layout.skillPath)
	if readErr != nil || string(content) != "user modified\n" {
		t.Fatal("uninstaller changed a modified managed file")
	}
	if !regularFileExists(layout.binaryPath) || !regularFileExists(layout.statePath) {
		t.Fatal("uninstaller partially removed managed files")
	}
}

func TestNativeInstallPreservesExistingAgentsMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes are not available on Windows")
	}
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	agentsPath := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("# private rules\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := installWithTestKey(&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("AGENTS.md mode changed to %04o", info.Mode().Perm())
	}
	backups, err := filepath.Glob(filepath.Join(root, "AGENTS.md.myautoapi-backup-*"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected one AGENTS backup, got %v (%v)", backups, err)
	}
	backupInfo, err := os.Stat(backups[0])
	if err != nil {
		t.Fatal(err)
	}
	if backupInfo.Mode().Perm() != 0o600 {
		t.Fatalf("AGENTS backup mode changed to %04o", backupInfo.Mode().Perm())
	}
}

func TestNativeInstallRepairsMissingExecutablePermission(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX executable bits are not available on Windows")
	}
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	if err := installWithTestKey(&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	layout := buildInstallLayout(root)
	if err := os.Chmod(layout.binaryPath, 0o644); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := installWithTestKey(&output); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "已经完整安装") {
		t.Fatal("installer ignored a missing executable permission")
	}
	info, err := os.Stat(layout.binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatal("installer did not restore an executable permission")
	}
}

func installWithTestKey(output *bytes.Buffer) error {
	return installWithInput(output, strings.NewReader("dedicated-image-key-test\n"))
}

func TestNativeRepeatedInstallAlwaysReplacesDedicatedKey(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	if err := installWithTestKey(&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	layout := buildInstallLayout(root)
	skillBefore := mustReadFile(t, layout.skillPath)
	rulesBefore := mustReadFile(t, layout.agentsPath)
	var output bytes.Buffer
	if err := installWithInput(&output, strings.NewReader("replacement-test-key\n")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "请输入专门画图分组 Key") {
		t.Fatal("repeated install did not ask for a new key")
	}
	if string(mustReadFile(t, filepath.Join(layout.toolDir, keyFileName))) != "replacement-test-key" {
		t.Fatal("repeated install did not replace the previous key")
	}
	if !bytes.Equal(mustReadFile(t, layout.skillPath), skillBefore) || !bytes.Equal(mustReadFile(t, layout.agentsPath), rulesBefore) {
		t.Fatal("key replacement changed unrelated managed content")
	}
	if strings.Contains(output.String(), "replacement-test-key") || strings.Contains(output.String(), "dedicated-image-key-test") {
		t.Fatal("key replacement printed a key")
	}
}

func TestNativeRepeatedInstallEmptyInputPreservesExistingKey(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	if err := installWithTestKey(&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	layout := buildInstallLayout(root)
	keyPath := filepath.Join(layout.toolDir, keyFileName)
	before := mustReadFile(t, keyPath)
	if err := installWithInput(&bytes.Buffer{}, strings.NewReader("\n")); err == nil {
		t.Fatal("empty input unexpectedly completed key replacement")
	}
	if !bytes.Equal(mustReadFile(t, keyPath), before) {
		t.Fatal("empty input erased the previous key")
	}
}

func TestNativeRepeatedInstallRollsBackExistingKey(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	if err := installWithTestKey(&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	layout := buildInstallLayout(root)
	keyPath := filepath.Join(layout.toolDir, keyFileName)
	before := mustReadFile(t, keyPath)
	err := installWithOptions(&bytes.Buffer{}, strings.NewReader("failed-replacement-test-key\n"), installOptions{
		failureHook: func(stage string) error { return errors.New("simulated replacement failure") },
	})
	if err == nil {
		t.Fatal("replacement unexpectedly succeeded")
	}
	if !bytes.Equal(mustReadFile(t, keyPath), before) {
		t.Fatal("failed replacement did not restore the previous key")
	}
}

func TestNativeInstallUpgradesV080AndReplacesKey(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	if err := installWithTestKey(&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	layout := buildInstallLayout(root)
	var state installState
	if err := json.Unmarshal(mustReadFile(t, layout.statePath), &state); err != nil {
		t.Fatal(err)
	}
	state.Version = "0.8.0"
	state.Release = "V0.8.0"
	content, _ := json.Marshal(state)
	if err := os.WriteFile(layout.statePath, content, 0o666); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := installWithInput(&output, strings.NewReader("v080-upgrade-test-key\n")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "升级完成：V0.8.1") {
		t.Fatal("V0.8.0 was not upgraded")
	}
	if string(mustReadFile(t, filepath.Join(layout.toolDir, keyFileName))) != "v080-upgrade-test-key" {
		t.Fatal("upgrade preserved the previous key")
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
}
