package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestConnectClaudeDesktopWithValidatedKey(t *testing.T) {
	const key = "sk-local-claude-fixture-key"
	dir := t.TempDir()
	files := claudeDesktopPaths(dir)
	if err := os.MkdirAll(filepath.Dir(files.library), 0700); err != nil {
		t.Fatal(err)
	}
	mode := "{\"deploymentMode\":\"1p\",\"otherSetting\":true}"
	library := "{\"appliedId\":\"old\",\"entries\":[{\"id\":\"old\",\"name\":\"Existing gateway\",\"keep\":true}]}"
	for name, value := range map[string]string{files.mode: mode, files.library: library} {
		if err := os.WriteFile(name, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var resultModels atomic.Value
	resultModels.Store("{\"data\":[{\"id\":\"gpt-6.2-sol\"}]}")
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+key {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/usage":
			_, _ = w.Write([]byte("{\"isValid\":true}"))
		case "/v1/models":
			_, _ = w.Write([]byte(resultModels.Load().(string)))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	connect := func() (string, error) {
		return connectClaudeModels(t.Context(), srv.Client(), srv.URL, key, func(key string, models []string) (string, error) {
			return writeClaudeDesktop(files, api, key, models)
		})
	}
	if _, err := connect(); err == nil || !strings.Contains(err.Error(), "没有可用的 Claude 模型") {
		t.Fatalf("wrong group: %v", err)
	}
	if data, _ := os.ReadFile(files.mode); string(data) != mode {
		t.Fatal("wrong group changed the startup mode")
	}
	if _, err := os.Stat(files.profile); !os.IsNotExist(err) {
		t.Fatal("wrong group created a gateway profile")
	}
	resultModels.Store("{\"data\":[{\"id\":\"claude-sonnet-4-6\"},{\"id\":\"gpt-6.2-sol\"}]}")
	if _, err := connect(); err != nil {
		t.Fatal(err)
	}
	if ok, err := claudeDesktopConfigured(files); err != nil || !ok {
		t.Fatalf("Claude Desktop does not report the new configuration: %v", err)
	}
	if data, _ := os.ReadFile(files.mode); !strings.Contains(string(data), "\"otherSetting\": true") {
		t.Fatal("lost an existing desktop preference")
	}
	if data, _ := os.ReadFile(files.library); !strings.Contains(string(data), "\"Existing gateway\"") ||
		!strings.Contains(string(data), "\"keep\": true") {
		t.Fatal("lost the existing gateway profile or its fields")
	}
	var profile struct {
		Key    string               `json:"inferenceGatewayApiKey"`
		Auth   string               `json:"inferenceGatewayAuthScheme"`
		Models []claudeDesktopModel `json:"inferenceModels"`
	}
	data, err := os.ReadFile(files.profile)
	if err != nil {
		t.Fatal(err)
	}
	if json.Unmarshal(data, &profile) != nil || profile.Key != key || profile.Auth != "bearer" ||
		len(profile.Models) != 1 || profile.Models[0].Name != "claude-sonnet-4-6" {
		t.Fatal("incorrect gateway credential settings or model list")
	}
}

func TestClaudeDesktopDoesNotReplaceServerManagedProfile(t *testing.T) {
	files := claudeDesktopPaths(t.TempDir())
	if err := os.MkdirAll(filepath.Dir(files.library), 0700); err != nil {
		t.Fatal(err)
	}
	library := "{\"hybridPointer\":{\"bootstrapUrl\":\"https://example.invalid\"}}"
	if err := os.WriteFile(files.library, []byte(library), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := writeClaudeDesktop(files, api, "sk-fixture", []string{"claude-sonnet-4-6"}); err == nil {
		t.Fatal("overwrote a server-managed profile")
	}
	if _, err := os.Stat(files.profile); !os.IsNotExist(err) {
		t.Fatal("created a new profile despite the server configuration")
	}
}
