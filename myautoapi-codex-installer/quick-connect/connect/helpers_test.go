package connect

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

// fixtureKey is synthetic; no test uses a real key or the real server.
const fixtureKey = "sk-quickconnect_fixture-0123456789abcdefXYZ"

var backupName = regexp.MustCompile(`^(config\.toml|auth\.json)\.bak\.\d{14}(\.\d+)?$`)

func installerRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func installScript(t *testing.T, name string) string {
	return filepath.Join(installerRoot(t), "install", name)
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// powerShell finds the development PowerShell 7 kept next to the domestic
// model tools, or one on PATH. Tests that need it are skipped without it.
func powerShell(t *testing.T) string {
	t.Helper()
	if path := os.Getenv("PWSH"); path != "" {
		return path
	}
	local := filepath.Join(installerRoot(t), "codex-domestic-models", ".dev-tools", "pwsh", "pwsh")
	if info, err := os.Stat(local); err == nil && !info.IsDir() {
		return local
	}
	if path, err := exec.LookPath("pwsh"); err == nil {
		return path
	}
	t.Skip("本机没有 PowerShell，跳过与 PowerShell 脚本的逐字节比对")
	return ""
}

// runPowerShell runs a script with HOME pointing at home. Sleeps are skipped
// and Get-Process returns nothing, so no real process is ever touched.
func runPowerShell(t *testing.T, home, prelude, script string, env ...string) (string, error) {
	t.Helper()
	command := exec.Command(powerShell(t), "-NoLogo", "-NoProfile", "-NonInteractive", "-Command",
		"function Start-Sleep { }\nfunction Get-Process { }\nfunction Stop-Process { }\n"+prelude+"\n. $env:MYAUTOAPI_TEST_SCRIPT")
	command.Env = append(os.Environ(),
		"HOME="+home, "USERPROFILE="+home, "MYAUTOAPI_TEST_SCRIPT="+script,
		"POWERSHELL_TELEMETRY_OPTOUT=1", "POWERSHELL_UPDATECHECK=Off", "DOTNET_CLI_TELEMETRY_OPTOUT=1",
		"XDG_CACHE_HOME="+filepath.Join(home, ".cache"), "XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"XDG_DATA_HOME="+filepath.Join(home, ".local", "share"))
	command.Env = append(command.Env, env...)
	var output bytes.Buffer
	command.Stdout, command.Stderr = &output, &output
	err := command.Run()
	return output.String(), err
}

func listBackups(t *testing.T, dir, prefix string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		if len(entry.Name()) > len(prefix) && entry.Name()[:len(prefix)] == prefix {
			names = append(names, entry.Name())
		}
	}
	return names
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

var defaultRename = os.Rename

// osPathError looks like a Windows access error with its localised text.
var osPathError = fs.PathError{Op: "rename", Path: `C:\Users\a\.codex\config.toml`, Err: errors.New("拒绝访问。")}
