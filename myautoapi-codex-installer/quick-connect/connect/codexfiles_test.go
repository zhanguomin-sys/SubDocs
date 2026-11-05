package connect

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

var fixedTime = time.Date(2026, 11, 1, 9, 8, 7, 0, time.Local)

func normalizeLineEndings(value string) string {
	return strings.ReplaceAll(value, "\r\n", "\n")
}

// The Go writer and install/codex.ps1 are compared against the same
// expected files, so neither can drift from the other.
func TestWriterMatchesExpectedFiles(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), ".codex")
	if err := WriteCodexFiles(codexHome, fixtureKey, fixedTime); err != nil {
		t.Fatal(err)
	}
	if got := normalizeLineEndings(readFile(t, filepath.Join(codexHome, "config.toml"))); got != normalizeLineEndings(string(readFixture(t, "expected-config.toml"))) {
		t.Fatalf("config.toml differs:\n%s", got)
	}
	if got := readFile(t, filepath.Join(codexHome, "auth.json")); got != string(readFixture(t, "expected-auth.json")) {
		t.Fatalf("auth.json differs:\n%s", got)
	}
	assertOnlyFiles(t, codexHome, "auth.json", "config.toml")
}

func TestPowerShellInstallerWritesTheSameFiles(t *testing.T) {
	usage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/usage" || r.Header.Get("Authorization") != "Bearer "+fixtureKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"isValid":true,"remaining":3.5,"unit":"USD"}`))
	}))
	defer usage.Close()
	home := t.TempDir()
	codexHome := filepath.Join(home, ".codex")
	writeFile(t, filepath.Join(codexHome, "config.toml"), "old config\n")
	writeFile(t, filepath.Join(codexHome, "auth.json"), "old auth\n")

	// Only the address of the key check is redirected to the local server.
	prelude := `function Invoke-WebRequest {
  [CmdletBinding()] param([string]$Uri, [hashtable]$Headers, [int]$TimeoutSec, [switch]$UseBasicParsing)
  $Uri = $Uri.Replace('https://api.myautoapi.com', $env:MYAUTOAPI_TEST_SERVER)
  Microsoft.PowerShell.Utility\Invoke-WebRequest -Uri $Uri -Headers $Headers -TimeoutSec $TimeoutSec -UseBasicParsing
}`
	output, err := runPowerShell(t, home, prelude, installScript(t, "codex.ps1"),
		"MYAUTOAPI_KEY="+fixtureKey, "MYAUTOAPI_TEST_SERVER="+usage.URL)
	if err != nil {
		t.Fatalf("codex.ps1 failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "密钥检查通过，余额 $3.50。") {
		t.Fatalf("key check result missing:\n%s", output)
	}
	if got := normalizeLineEndings(readFile(t, filepath.Join(codexHome, "config.toml"))); got != normalizeLineEndings(string(readFixture(t, "expected-config.toml"))) {
		t.Fatalf("codex.ps1 config.toml differs from the Go writer:\n%q", got)
	}
	if got := readFile(t, filepath.Join(codexHome, "auth.json")); got != string(readFixture(t, "expected-auth.json")) {
		t.Fatalf("codex.ps1 auth.json differs from the Go writer:\n%q", got)
	}
	for _, prefix := range []string{"config.toml.bak.", "auth.json.bak."} {
		backups := listBackups(t, codexHome, prefix)
		if len(backups) != 1 || !backupName.MatchString(backups[0]) {
			t.Fatalf("PowerShell backups %v", backups)
		}
	}
}

func TestWriterBacksUpWithScriptNames(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), ".codex")
	writeFile(t, filepath.Join(codexHome, "config.toml"), "first config")
	writeFile(t, filepath.Join(codexHome, "auth.json"), "first auth")
	for round := 0; round < 3; round++ {
		if err := WriteCodexFiles(codexHome, fixtureKey, fixedTime); err != nil {
			t.Fatal(err)
		}
	}
	config := listBackups(t, codexHome, "config.toml.bak.")
	sort.Strings(config)
	want := []string{"config.toml.bak.20261101090807", "config.toml.bak.20261101090807.1", "config.toml.bak.20261101090807.2"}
	if strings.Join(config, ",") != strings.Join(want, ",") {
		t.Fatalf("backups %v, want %v", config, want)
	}
	if readFile(t, filepath.Join(codexHome, want[0])) != "first config" {
		t.Fatal("first backup does not hold the original config")
	}
	if readFile(t, filepath.Join(codexHome, "auth.json.bak.20261101090807")) != "first auth" {
		t.Fatal("auth backup does not hold the original auth")
	}
	if readFile(t, filepath.Join(codexHome, want[1])) != string(readFixture(t, "expected-config.toml")) {
		t.Fatal("second backup should hold the first written config")
	}
}

func TestWriterRollsBackWhenAuthCannotBeReplaced(t *testing.T) {
	for _, existing := range []bool{true, false} {
		codexHome := filepath.Join(t.TempDir(), ".codex")
		if existing {
			writeFile(t, filepath.Join(codexHome, "config.toml"), "original config")
			writeFile(t, filepath.Join(codexHome, "auth.json"), "original auth")
		}
		calls := 0
		renameFile = func(from, to string) error {
			calls++
			if calls == 2 {
				return &os.LinkError{Op: "rename", Old: from, New: to, Err: errors.New("disk is full")}
			}
			return os.Rename(from, to)
		}
		err := WriteCodexFiles(codexHome, fixtureKey, fixedTime)
		renameFile = os.Rename
		if err == nil || reason(err) != "disk is full" {
			t.Fatalf("error = %v", err)
		}
		if existing {
			if readFile(t, filepath.Join(codexHome, "config.toml")) != "original config" || readFile(t, filepath.Join(codexHome, "auth.json")) != "original auth" {
				t.Fatal("the original files were not kept")
			}
			assertOnlyFiles(t, codexHome, "auth.json", "auth.json.bak.20261101090807", "config.toml", "config.toml.bak.20261101090807")
		} else {
			assertOnlyFiles(t, codexHome)
		}
	}
}

func TestWriterLeavesOriginalsWhenFirstReplacementFails(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), ".codex")
	writeFile(t, filepath.Join(codexHome, "config.toml"), "original config")
	renameFile = func(string, string) error { return errors.New("locked") }
	defer func() { renameFile = os.Rename }()
	if err := WriteCodexFiles(codexHome, fixtureKey, fixedTime); err == nil {
		t.Fatal("expected an error")
	}
	if readFile(t, filepath.Join(codexHome, "config.toml")) != "original config" {
		t.Fatal("config changed")
	}
	assertOnlyFiles(t, codexHome, "config.toml", "config.toml.bak.20261101090807")
}

func TestWriterReportsUnwritableFolder(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, ".codex")
	writeFile(t, blocker, "a file where the folder should be")
	if err := WriteCodexFiles(blocker, fixtureKey, fixedTime); err == nil {
		t.Fatal("expected an error")
	}
	if readFile(t, blocker) != "a file where the folder should be" {
		t.Fatal("blocker changed")
	}
}

func TestAuthContentEscapesJSON(t *testing.T) {
	if got := AuthContent(`sk-a"b\c`); got != "{\n  \"OPENAI_API_KEY\": \"sk-a\\\"b\\\\c\"\n}\n" {
		t.Fatalf("got %q", got)
	}
}

func assertOnlyFiles(t *testing.T, dir string, names ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && len(names) == 0 {
			return
		}
		t.Fatal(err)
	}
	var got []string
	for _, entry := range entries {
		got = append(got, entry.Name())
	}
	sort.Strings(got)
	sort.Strings(names)
	if strings.Join(got, ",") != strings.Join(names, ",") {
		t.Fatalf("files in %s: %v, want %v", dir, got, names)
	}
}

// The Codex App on Windows only offers Full Access once its sandbox is set
// up; that choice lives in [windows] and must survive a reconnect.
func TestWriterKeepsWindowsSandbox(t *testing.T) {
	base := string(readFixture(t, "expected-config.toml"))
	base = strings.TrimSuffix(base, "\n\n[windows]\nsandbox = \"unelevated\"")
	cases := []struct {
		name, old string
		sandbox   bool
		want      string
	}{
		{"kept as it was", "model = \"x\"\r\n\r\n[windows]\r\nsandbox = \"elevated\"\r\n\r\n[mcp_servers.a]\r\ncommand = \"a\"\r\n",
			false, "[windows]\nsandbox = \"elevated\""},
		{"value kept, table rewritten", "[windows]\nsandbox = 'unelevated' # chosen\n[windows.extra]\nx = 1\n\n", true, "[windows]\nsandbox = \"unelevated\""},
		// A [windows] line inside a multi-line string is not a table; nothing
		// of the old file is copied, so the result stays valid.
		{"inside a multi-line string", "developer_instructions = \"\"\"\n[windows]\nnote\n\"\"\"\n", false, "[windows]\nsandbox = \"unelevated\""},
		{"sandbox outside [windows] ignored", "[other]\nsandbox = \"elevated\"\n", false, "[windows]\nsandbox = \"unelevated\""},
		{"set up before", "model = \"x\"\n", true, "[windows]\nsandbox = \"elevated\""},
		{"new user", "", false, "[windows]\nsandbox = \"unelevated\""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			codexHome := filepath.Join(t.TempDir(), ".codex")
			if c.old != "" {
				writeFile(t, filepath.Join(codexHome, "config.toml"), c.old)
			}
			if c.sandbox {
				if err := os.MkdirAll(filepath.Join(codexHome, ".sandbox"), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if err := WriteCodexFiles(codexHome, fixtureKey, fixedTime); err != nil {
				t.Fatal(err)
			}
			if got := readFile(t, filepath.Join(codexHome, "config.toml")); got != base+"\n\n"+c.want {
				t.Fatalf("config.toml:\n%s", got)
			}
		})
	}
}

func TestPowerShellInstallerKeepsWindowsSandbox(t *testing.T) {
	usage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"isValid":true,"remaining":3.5,"unit":"USD"}`))
	}))
	defer usage.Close()
	home := t.TempDir()
	codexHome := filepath.Join(home, ".codex")
	old := "model = \"x\"\r\n\r\n[windows]\r\nsandbox = 'elevated' # chosen\r\n\r\n[mcp_servers.a]\r\ncommand = \"a\"\r\n"
	writeFile(t, filepath.Join(codexHome, "config.toml"), old)
	prelude := `function Invoke-WebRequest {
  [CmdletBinding()] param([string]$Uri, [hashtable]$Headers, [int]$TimeoutSec, [switch]$UseBasicParsing)
  $Uri = $Uri.Replace('https://api.myautoapi.com', $env:MYAUTOAPI_TEST_SERVER)
  Microsoft.PowerShell.Utility\Invoke-WebRequest -Uri $Uri -Headers $Headers -TimeoutSec $TimeoutSec -UseBasicParsing
}`
	output, err := runPowerShell(t, home, prelude, installScript(t, "codex.ps1"),
		"MYAUTOAPI_KEY="+fixtureKey, "MYAUTOAPI_TEST_SERVER="+usage.URL)
	if err != nil {
		t.Fatalf("codex.ps1 failed: %v\n%s", err, output)
	}
	goHome := filepath.Join(t.TempDir(), ".codex")
	writeFile(t, filepath.Join(goHome, "config.toml"), old)
	if err := WriteCodexFiles(goHome, fixtureKey, fixedTime); err != nil {
		t.Fatal(err)
	}
	if got, want := normalizeLineEndings(readFile(t, filepath.Join(codexHome, "config.toml"))), normalizeLineEndings(readFile(t, filepath.Join(goHome, "config.toml"))); got != want {
		t.Fatalf("codex.ps1:\n%q\nGo:\n%q", got, want)
	}
}
