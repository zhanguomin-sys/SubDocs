package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestCodexKeepsExistingSettingsAndUsesDedicatedCredential(t *testing.T) {
	old := []byte("model = \"gpt-existing\"\nmodel_provider = \"custom\"\nmodel_catalog_json = \"mine.json\"\nsandbox_mode = \"workspace-write\"\n" +
		"[desktop]\ntheme = \"dark\"\n[model_providers.custom]\nname = \"Old provider\"\n[projects.\"C:/My Project\"]\ntrust_level = \"trusted\"\n")
	next, err := codexConfig(old, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := localeConfig(next)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if _, err := toml.Decode(string(res), &got); err != nil {
		t.Fatal(err)
	}
	if got["sandbox_mode"] != "workspace-write" || got["model_catalog_json"] != "mine.json" ||
		got["model"] != gpt62Model || got["model_reasoning_effort"] != gpt62Effort || got["model_provider"] != provider {
		t.Fatal("unexpected change to Codex preferences")
	}
	providers := got["model_providers"].(map[string]any)
	mine := providers[provider].(map[string]any)
	if mine["base_url"] != api+"/v1" || mine["env_key"] != keyVariable || mine["requires_openai_auth"] != false ||
		providers["custom"].(map[string]any)["name"] != "Old provider" {
		t.Fatal("provider configuration lost existing settings or used the wrong credentials")
	}
	project := got["projects"].(map[string]any)["C:/My Project"].(map[string]any)
	desktop := got["desktop"].(map[string]any)
	if desktop["localeOverride"] != "zh-CN" || desktop["theme"] != "dark" || project["trust_level"] != "trusted" {
		t.Fatal("project settings or language preference were lost")
	}
	for _, old := range []string{"[bad TOML", "profile = \"current\""} {
		if _, err := codexConfig([]byte(old), nil); err == nil {
			t.Fatal("a malformed file or active profile must be left intact")
		}
	}
}

func TestCodexDefaultsReplacePreviousModelAndEnableHigh(t *testing.T) {
	old := []byte("model_provider = \"myautoapi\"\nmodel = \"gpt-existing\"\nmodel_reasoning_effort = \"medium\"\nmodel_catalog_json = \"mine.json\"\n[desktop]\ntheme = \"dark\"\nenabled-reasoning-efforts = [\"low\", \"medium\"]\n")
	next, err := codexConfig(old, nil)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Model   string `toml:"model"`
		Effort  string `toml:"model_reasoning_effort"`
		Catalog string `toml:"model_catalog_json"`
		Desktop struct {
			Theme   string   `toml:"theme"`
			Efforts []string `toml:"enabled-reasoning-efforts"`
		} `toml:"desktop"`
	}
	if _, err := toml.Decode(string(next), &got); err != nil {
		t.Fatal(err)
	}
	if ok, err := codexUsesMyAutoAPI(next); err != nil || !ok || got.Model != gpt62Model || got.Effort != gpt62Effort ||
		got.Catalog != "mine.json" || got.Desktop.Theme != "dark" || strings.Join(got.Desktop.Efforts, ",") != "low,medium,high" {
		t.Fatalf("default model, effort, or kept preferences are wrong: %+v", got)
	}
	again, err := codexConfig(next, nil)
	if err != nil || !bytes.Equal(again, next) {
		t.Fatal("reconnecting changed an already configured default")
	}
}

func TestReusableCodexKeyRequiresMyAutoAPIAndSavedKey(t *testing.T) {
	connected, err := codexConfig(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	key, err := reusableCodexKey(connected, "sk-saved-fixture-key")
	if err != nil || key != "sk-saved-fixture-key" {
		t.Fatalf("could not reuse configured GPT key: %q, %v", key, err)
	}
	for _, tc := range []struct {
		name, config, stored string
	}{
		{"not configured", "model_provider = \"openai\"\n", "sk-saved-fixture-key"},
		{"missing key", string(connected), ""},
		{"invalid key", string(connected), "not-a-key"},
		{"invalid TOML", "[broken", "sk-saved-fixture-key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := reusableCodexKey([]byte(tc.config), tc.stored); err == nil {
				t.Fatal("unexpectedly reused an unconfigured or invalid key")
			}
		})
	}
}

func TestClaudePreservesPermissionsAndUnrelatedEnvironment(t *testing.T) {
	old := []byte("{\"env\":{\"KEEP\":\"yes\",\"ANTHROPIC_API_KEY\":\"stale-demo\"},\"permissions\":{\"deny\":[\"Read(fake)\"]}}")
	key := "sk-synthetic-key"
	res, err := claudeCodeConfig(old, key, []string{"claude-sonnet-4-6"})
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Env         map[string]string
		Permissions struct{ Deny []string }
	}
	if err := json.Unmarshal(res, &got); err != nil {
		t.Fatal(err)
	}
	if got.Env["KEEP"] != "yes" || got.Env["ANTHROPIC_BASE_URL"] != api || got.Env["ANTHROPIC_AUTH_TOKEN"] != key ||
		got.Env["ANTHROPIC_API_KEY"] != "" || len(got.Permissions.Deny) != 1 {
		t.Fatal("did not preserve existing settings or wrote a conflicting token")
	}
	if _, err := validateKey("sk-demo\nAuthorization: invalid"); err == nil {
		t.Fatal("accepted multiline credentials")
	}
}

func TestSaveBacksUpExistingBytesAndRejectsConcurrentEdit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test-config.toml")
	config, err := codexConfig([]byte("[desktop]\ntheme = \"dark\"\n"), nil)
	if err != nil {
		t.Fatal(err)
	}
	old := append([]byte("# user comment\n"), config...)
	if err := os.WriteFile(path, old, 0600); err != nil {
		t.Fatal(err)
	}
	backup, err := applyConfig(path, localeConfig)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(backup)
	if err != nil || !bytes.Equal(saved, old) {
		t.Fatal("backup is not the exact previous config")
	}
	next, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(next), "localeOverride = \"zh-CN\"") {
		t.Fatal("language preference was not saved")
	}
	if _, err := saveConfig(path, old, true, []byte("stale write")); err == nil {
		t.Fatal("overwrote a file modified by another writer")
	}
}
