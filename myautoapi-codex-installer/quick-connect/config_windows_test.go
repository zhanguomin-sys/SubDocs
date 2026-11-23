package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetChineseRequiresCurrentMyAutoAPIConnection(t *testing.T) {
	connected, err := codexConfig([]byte("[desktop]\ntheme = \"dark\"\n"), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		data    []byte
		message string
	}{
		{"missing config", nil, "尚未配置 MyAutoAPI"},
		{"official login", []byte("model_provider = \"openai\"\n"), "尚未配置 MyAutoAPI"},
		{"other provider", bytes.ReplaceAll(connected, []byte(api), []byte("https://other.example")), "尚未配置 MyAutoAPI"},
		{"active profile", append([]byte("profile = \"work\"\n"), connected...), "尚未配置 MyAutoAPI"},
		{"invalid config", []byte("[missing bracket"), "无法识别 Codex 配置"},
		{"connected", connected, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("CODEX_HOME", root)
			path := filepath.Join(root, "config.toml")
			closeCalls := 0
			var localeCalls []bool
			app := &App{ctx: t.Context(), setupLocale: func(_ context.Context, gotRoot string, check bool) error {
				if gotRoot != root || (!check && closeCalls != 1) {
					t.Fatal("language cache updated before shutdown or for the wrong configuration")
				}
				localeCalls = append(localeCalls, check)
				return nil
			}, closeClient: func() error {
				closeCalls++
				current, err := os.ReadFile(path)
				if err != nil || !bytes.Contains(current, []byte("localeOverride")) {
					t.Fatal("client closed before saving language")
				}
				return nil
			}}
			if tc.data != nil {
				// Simulate a change after the status card last read a good config.
				if err := os.WriteFile(path, connected, 0600); err != nil {
					t.Fatal(err)
				}
				if displayed, err := app.ConnectionStatus("codex"); err != nil || !displayed {
					t.Fatal("expected initial UI status to be configured")
				}
				if err := os.WriteFile(path, tc.data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			reply, err := app.SetChinese()
			if tc.message == "" {
				current, readErr := os.ReadFile(path)
				if err != nil || readErr != nil || closeCalls != 1 || !bytes.Contains(current, []byte(`localeOverride = "zh-CN"`)) ||
					!bytes.Contains(current, []byte(`theme = "dark"`)) || !strings.Contains(reply, "重新打开 Codex") || len(localeCalls) != 2 || !localeCalls[0] || localeCalls[1] {
					t.Fatalf("failed to save the language setting: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.message) || reply != "" || closeCalls != 0 || len(localeCalls) != 0 {
				t.Fatalf("expected refusal, got %q, %v", reply, err)
			}
			entries, err := os.ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			if tc.data == nil {
				if len(entries) != 0 {
					t.Fatal("rejected request created files")
				}
				return
			}
			current, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(current, tc.data) || len(entries) != 1 {
				t.Fatal("rejected request modified the configuration or created a backup")
			}
		})
	}
}
