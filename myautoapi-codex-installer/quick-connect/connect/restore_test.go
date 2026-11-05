package connect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type restoreFixture struct {
	name, config, expected string
}

func restoreFixtures(t *testing.T) []restoreFixture {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("testdata", "restore", "*.expected.toml"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no restore fixtures: %v", err)
	}
	var fixtures []restoreFixture
	for _, expected := range paths {
		name := strings.TrimSuffix(filepath.Base(expected), ".expected.toml")
		fixtures = append(fixtures, restoreFixture{
			name:     name,
			config:   string(readFixture(t, filepath.Join("restore", name+".toml"))),
			expected: string(readFixture(t, filepath.Join("restore", name+".expected.toml"))),
		})
	}
	return fixtures
}

func TestCleanConfigFixtures(t *testing.T) {
	for _, fixture := range restoreFixtures(t) {
		if got := CleanConfig(fixture.config); got != fixture.expected {
			t.Errorf("%s:\n got %q\nwant %q", fixture.name, got, fixture.expected)
		}
	}
}

func TestRestoreMatchesPowerShellRestorer(t *testing.T) {
	for _, fixture := range restoreFixtures(t) {
		t.Run(fixture.name, func(t *testing.T) {
			goHome := filepath.Join(t.TempDir(), ".codex")
			writeFile(t, filepath.Join(goHome, "config.toml"), fixture.config)
			writeFile(t, filepath.Join(goHome, "auth.json"), `{"OPENAI_API_KEY":"sk-test"}`+"\n")
			if err := RestoreOfficial(goHome, fixedTime); err != nil {
				t.Fatal(err)
			}
			goConfig := readFile(t, filepath.Join(goHome, "config.toml"))
			if goConfig != fixture.expected || readFile(t, filepath.Join(goHome, "auth.json")) != "" {
				t.Fatalf("Go restore: %q", goConfig)
			}
			if readFile(t, filepath.Join(goHome, "config.toml.bak.20261101090807")) != fixture.config ||
				readFile(t, filepath.Join(goHome, "auth.json.bak.20261101090807")) != `{"OPENAI_API_KEY":"sk-test"}`+"\n" {
				t.Fatal("Go restore backups do not hold the originals")
			}

			home := t.TempDir()
			psHome := filepath.Join(home, ".codex")
			writeFile(t, filepath.Join(psHome, "config.toml"), fixture.config)
			writeFile(t, filepath.Join(psHome, "auth.json"), `{"OPENAI_API_KEY":"sk-test"}`+"\n")
			if output, err := runPowerShell(t, home, "", installScript(t, "codex-restore.ps1")); err != nil {
				t.Fatalf("codex-restore.ps1: %v\n%s", err, output)
			}
			if psConfig := readFile(t, filepath.Join(psHome, "config.toml")); psConfig != goConfig {
				t.Fatalf("PowerShell %q\n       Go %q", psConfig, goConfig)
			}
			if readFile(t, filepath.Join(psHome, "auth.json")) != "" {
				t.Fatal("PowerShell did not empty auth.json")
			}
			if len(listBackups(t, psHome, "config.toml.bak.")) != 1 || len(listBackups(t, psHome, "auth.json.bak.")) != 1 {
				t.Fatal("PowerShell backups differ")
			}
		})
	}
}

func TestRestoreMissingFiles(t *testing.T) {
	// No .codex folder: nothing to do.
	missing := filepath.Join(t.TempDir(), ".codex")
	if err := RestoreOfficial(missing, fixedTime); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("restore created the folder")
	}

	// Only auth.json: it is emptied and config.toml is not created.
	onlyAuth := filepath.Join(t.TempDir(), ".codex")
	writeFile(t, filepath.Join(onlyAuth, "auth.json"), "{}")
	if err := RestoreOfficial(onlyAuth, fixedTime); err != nil {
		t.Fatal(err)
	}
	assertOnlyFiles(t, onlyAuth, "auth.json", "auth.json.bak.20261101090807")
	if readFile(t, filepath.Join(onlyAuth, "auth.json")) != "" {
		t.Fatal("auth.json not emptied")
	}

	// Only config.toml: auth.json is not created.
	onlyConfig := filepath.Join(t.TempDir(), ".codex")
	writeFile(t, filepath.Join(onlyConfig, "config.toml"), "model = \"x\"\n[features]\ngoals = true\n")
	if err := RestoreOfficial(onlyConfig, fixedTime); err != nil {
		t.Fatal(err)
	}
	assertOnlyFiles(t, onlyConfig, "config.toml", "config.toml.bak.20261101090807")
}

func TestRestoreRollsBackConfigWhenAuthFails(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), ".codex")
	writeFile(t, filepath.Join(codexHome, "config.toml"), string(readFixture(t, "restore/01-installer.toml")))
	writeFile(t, filepath.Join(codexHome, "auth.json"), "secret")
	calls := 0
	renameFile = func(from, to string) error {
		calls++
		if calls == 2 {
			return os.ErrPermission
		}
		return os.Rename(from, to)
	}
	err := RestoreOfficial(codexHome, fixedTime)
	renameFile = os.Rename
	if err == nil {
		t.Fatal("expected an error")
	}
	if readFile(t, filepath.Join(codexHome, "config.toml")) != string(readFixture(t, "restore/01-installer.toml")) ||
		readFile(t, filepath.Join(codexHome, "auth.json")) != "secret" {
		t.Fatal("original files were not kept")
	}
}

// An access error while looking at the folder or files must be reported, not
// treated as "nothing to restore".
func TestRestoreReportsInspectionErrors(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores folder permissions")
	}
	parent := t.TempDir()
	codexHome := filepath.Join(parent, ".codex")
	writeFile(t, filepath.Join(codexHome, "config.toml"), "model = \"x\"\n")
	writeFile(t, filepath.Join(codexHome, "auth.json"), "secret")

	for _, locked := range []string{codexHome, parent} {
		if err := os.Chmod(locked, 0); err != nil {
			t.Fatal(err)
		}
		err := RestoreOfficial(codexHome, fixedTime)
		if chmodErr := os.Chmod(locked, 0o700); chmodErr != nil {
			t.Fatal(chmodErr)
		}
		if err == nil {
			t.Fatalf("locked %s: expected an error", locked)
		}
		if readFile(t, filepath.Join(codexHome, "config.toml")) != "model = \"x\"\n" ||
			readFile(t, filepath.Join(codexHome, "auth.json")) != "secret" {
			t.Fatalf("locked %s: files changed", locked)
		}
	}
	assertOnlyFiles(t, codexHome, "auth.json", "config.toml")
}
