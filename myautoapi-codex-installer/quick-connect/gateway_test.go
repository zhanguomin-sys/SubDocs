package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestConnectRequiresGatewayApprovalBeforeEditingConfig(t *testing.T) {
	const fixtureKey = "sk-local-test-fixture-key"
	for _, test := range []struct {
		name, payload, errorText string
		status                   int
	}{
		{"active key", `{"mode":"unrestricted","isValid":true}`, "", http.StatusOK},
		{"wrong site key", `{"error":"private server message"}`, "没有识别", http.StatusUnauthorized},
		{"key blocked", "", "拒绝使用", http.StatusForbidden},
		{"invalid key", `{"isValid":false}`, "当前不可用", http.StatusOK},
		{"expired key", `{"isValid":true,"status":"expired"}`, "已过期", http.StatusOK},
		{"quota exceeded", `{"isValid":true,"status":"quota_exhausted"}`, "已用完", http.StatusOK},
		{"not a usage result", "<html>error page</html>", "无法识别", http.StatusOK},
		{"api error", "", "HTTP 502", http.StatusBadGateway},
	} {
		t.Run(test.name, func(t *testing.T) {
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.RequestURI() != "/v1/usage" || r.Header.Get("Authorization") != "Bearer "+fixtureKey {
					t.Error("expected a read-only request with the test bearer token")
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.status)
				fmt.Fprint(w, test.payload)
			}))
			defer srv.Close()
			client := trustedClient(time.Second * 5)
			client.Transport = srv.Client().Transport
			file := filepath.Join(t.TempDir(), "settings.json")
			old := []byte(`{"env":{"ANTHROPIC_BASE_URL":"https://example.invalid"},"keep":true}`)
			if err := os.WriteFile(file, old, 0600); err != nil {
				t.Fatal(err)
			}
			_, err := configureWithKey(context.Background(), client, srv.URL, "  "+fixtureKey, func(key string) (string, error) {
				return applyConfig(file, func(data []byte) ([]byte, error) { return claudeCodeConfig(data, key, []string{"claude-sonnet-4-6"}) })
			})
			data, readErr := os.ReadFile(file)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if test.errorText != "" {
				if err == nil || !strings.Contains(err.Error(), test.errorText) || strings.Contains(err.Error(), "private server message") || strings.Contains(err.Error(), fixtureKey) {
					t.Fatalf("unexpected validation result: %v", err)
				}
				if !bytes.Equal(old, data) {
					t.Fatal("rejected key changed the config")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if configured, err := claudeCodeUsesMyAutoAPI(data); err != nil || !configured {
					t.Fatal("approved key did not configure the fixture for MyAutoAPI")
				}
			}
		})
	}
}

func TestGatewayValidationDoesNotFollowRedirects(t *testing.T) {
	var called atomic.Bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/usage" {
			http.Redirect(w, r, "/other", http.StatusTemporaryRedirect)
			return
		}
		called.Store(true)
		w.Write([]byte(`{"isValid":true}`))
	}))
	defer server.Close()
	client := trustedClient(5 * time.Second)
	client.Transport = server.Client().Transport
	if err := verifyGatewayKey(context.Background(), client, server.URL, "sk-local-test-fixture-key"); err == nil || called.Load() {
		t.Fatal("validation must reject and not follow the redirect")
	}
}

func TestClaudeConnectChecksModelsAndCompletesInitialSetup(t *testing.T) {
	const key = "sk-local-claude-fixture-key"
	const oldSettings = `{"model":"claude-old-unavailable","env":{"ANTHROPIC_API_KEY":"stale","CLAUDE_CODE_USE_VERTEX":"1","KEEP":"yes"},"permissions":{"deny":["Read(private)"]}}`
	const oldState = `{"hasCompletedOnboarding":false,"projects":{"other":{"hasTrustDialogAccepted":false}}}`
	for _, tc := range []struct {
		name, models, state, wantError string
	}{
		{"usable Claude group", `{"data":[{"id":"gpt-6.2-sol"},{"id":"claude-sonnet-4-6"}]}`, oldState, ""},
		{"wrong model group", `{"data":[{"id":"gpt-6.2-sol"}]}`, oldState, "没有可用的 Claude 模型"},
		{"malformed startup state", `{"data":[{"id":"claude-sonnet-4-6"}]}`, "{broken", "无法更新 Claude Code 首次启动状态"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			profile := t.TempDir()
			t.Setenv("CLAUDE_CONFIG_DIR", profile)
			settings := filepath.Join(profile, "settings.json")
			state := filepath.Join(profile, ".claude.json")
			if err := os.WriteFile(settings, []byte(oldSettings), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(state, []byte(tc.state), 0600); err != nil {
				t.Fatal(err)
			}
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer "+key {
					t.Error("expected read-only gateway request with fixture key")
				}
				switch r.URL.Path {
				case "/v1/usage":
					fmt.Fprint(w, `{"isValid":true}`)
				case "/v1/models":
					fmt.Fprint(w, tc.models)
				default:
					t.Errorf("unexpected validation request: %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			result, err := connectClaudeCode(t.Context(), server.Client(), server.URL, key)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("unexpected connection result: %v", err)
				}
				nextSettings, settingsErr := os.ReadFile(settings)
				nextState, stateErr := os.ReadFile(state)
				if settingsErr != nil || stateErr != nil || string(nextSettings) != oldSettings || string(nextState) != tc.state {
					t.Fatal("unsuccessful setup modified the original files")
				}
				return
			}
			if err != nil || !strings.Contains(result, "原配置备份") || !strings.Contains(result, "原启动状态备份") {
				t.Fatalf("did not complete or report both backups: %v", err)
			}
			var saved struct {
				Model       string
				Env         map[string]string
				Permissions json.RawMessage
			}
			written, err := os.ReadFile(settings)
			if err != nil || json.Unmarshal(written, &saved) != nil ||
				saved.Model != "claude-sonnet-4-6" || saved.Env["ANTHROPIC_BASE_URL"] != api ||
				saved.Env["ANTHROPIC_AUTH_TOKEN"] != key || saved.Env["ANTHROPIC_API_KEY"] != "" ||
				saved.Env["CLAUDE_CODE_USE_VERTEX"] != "" || saved.Env["KEEP"] != "yes" ||
				!bytes.Contains(saved.Permissions, []byte("Read(private)")) {
				t.Fatal("saved Claude settings still conflict or lost prior settings")
			}
			var savedState struct {
				HasCompletedOnboarding bool
				Projects               map[string]struct{ HasTrustDialogAccepted bool }
			}
			written, err = os.ReadFile(state)
			if err != nil || json.Unmarshal(written, &savedState) != nil || !savedState.HasCompletedOnboarding ||
				len(savedState.Projects) != 1 || savedState.Projects["other"].HasTrustDialogAccepted {
				t.Fatal("setup did not finish onboarding or changed a project trust decision")
			}
		})
	}
}
