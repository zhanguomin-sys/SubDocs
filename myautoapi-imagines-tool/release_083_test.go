package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestV082UpgradeInstallsGPTImage26Defaults(t *testing.T) {
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
	state.Version = "0.8.2"
	content, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.statePath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := installWithInput(&output, strings.NewReader("synthetic-v083-upgrade-key\n")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "升级完成：V0.8.3") {
		t.Fatalf("upgrade result missing: %s", output.String())
	}
	for _, path := range []string{layout.agentsPath, layout.skillPath} {
		installed := string(mustReadFile(t, path))
		for _, model := range []string{flareModelName, sunburstModelName, legacyFlareModelName, legacySunburstModelName} {
			if !strings.Contains(installed, model) {
				t.Fatalf("installed guidance is missing %s", model)
			}
		}
	}
}

func TestGPTImage26KeepsGIFOutsideTheRelease(t *testing.T) {
	for _, model := range []string{flareModelName, sunburstModelName} {
		if _, err := parseGenerationOptions([]string{"--prompt", "synthetic", "--model", model, "--output-format", "gif"}); err == nil {
			t.Fatalf("%s accepted unconfirmed GIF output", model)
		}
	}
}
