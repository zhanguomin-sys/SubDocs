package main

import (
	"bytes"
	"crypto/sha256"
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

func TestWindowsAssistantInstallsRepositoryPackagesWithMockGateway(t *testing.T) {
	bundle := testBundle(t)
	root := filepath.Join(t.TempDir(), "Codex 测试目录")
	t.Setenv("CODEX_HOME", root)
	// The installers only consume the validated key and bundled payloads.
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("NO_PROXY", "")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	originalConfig := "model = \"gpt-default-test\"\nmodel_provider = \"my-provider\"\n\n[features]\ngoals = true\n"
	if err := os.WriteFile(filepath.Join(root, "config.toml"), []byte(originalConfig), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# Existing test notes\n"), 0600); err != nil {
		t.Fatal(err)
	}
	const key = "sk-assistant-image-test-fixture"
	const wrongGroup = "sk-assistant-gpt-test-fixture"
	var imageAccess atomic.Bool
	imageAccess.Store(true)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bearer := r.Header.Get("Authorization")
		if r.Method != http.MethodGet || (bearer != "Bearer "+key && bearer != "Bearer "+wrongGroup) {
			t.Error("expected read-only validation with a fixture key")
		}
		switch r.URL.Path {
		case "/v1/usage":
			fmt.Fprint(w, `{"isValid":true}`)
		case "/v1/models":
			model := gpt62Model
			if imageAccess.Load() && bearer == "Bearer "+key {
				model = defaultImageModel
			}
			fmt.Fprintf(w, `{"data":[{"id":%q}]}`, model)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	closeCalls := 0
	app := &App{ctx: t.Context(), gateway: keyGateway{client: server.Client(), origin: server.URL}, toolUpdates: make(map[string]toolRelease), closeClient: func() error {
		closeCalls++
		if closeCalls < 3 {
			if !regularNonempty(filepath.Join(root, "skills", "myautoapi-image-generation", "SKILL.md")) {
				t.Fatal("closed before installing the drawing tool")
			}
		} else if !regularNonempty(filepath.Join(root, modelReceipt)) {
			t.Fatal("closed before recording the model import")
		}
		return nil
	}}
	if _, err := app.InstallTool(imageTool, "bundled", ""); err == nil || closeCalls != 0 {
		t.Fatal("first image install skipped the dedicated key prompt")
	}
	if _, err := app.InstallTool(imageTool, "bundled", wrongGroup); err == nil || !strings.Contains(err.Error(), defaultImageModel) || closeCalls != 0 {
		t.Fatal("an active key from the wrong group should be rejected before install")
	}
	if _, err := os.Stat(filepath.Join(root, "tools")); !os.IsNotExist(err) {
		t.Fatal("rejected image key created tool files")
	}
	image, err := app.InstallTool(imageTool, "bundled", key)
	if err != nil {
		t.Fatal(err)
	}
	if got := image.Status.InstalledVersion; got != bundle.Image.Version {
		t.Fatalf("image version after install = %q, want %s", got, bundle.Image.Version)
	}
	for _, path := range []string{
		filepath.Join("skills", "myautoapi-image-generation", "SKILL.md"),
		filepath.Join("rules", "myautoapi-imagines-tool.rules"),
	} {
		if !regularNonempty(filepath.Join(root, path)) {
			t.Fatalf("installer omitted %s", path)
		}
	}
	rules, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil || !bytes.Contains(rules, []byte("# Existing test notes")) ||
		!bytes.Contains(rules, []byte("<!-- myautoapi-image-generation:start -->")) {
		t.Fatal("installer did not preserve and extend fixture instructions")
	}
	imageAccess.Store(false)
	if _, err := app.InstallTool(imageTool, "bundled", ""); err == nil || !strings.Contains(err.Error(), defaultImageModel) || closeCalls != 1 {
		t.Fatal("reinstall accepted a saved key that lost access to the default image model")
	}
	imageAccess.Store(true)
	if _, err := app.InstallTool(imageTool, "bundled", ""); err != nil {
		t.Fatal(err)
	}
	// Only read a synthetic credential in this test directory.
	keyPath := filepath.Join(root, "tools", "myautoapi-imagines-tool", "api-key")
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil || strings.TrimSpace(string(keyBytes)) != key {
		t.Fatal("reinstall did not preserve the fixture's dedicated key")
	}

	models, err := app.InstallTool(modelsTool, "bundled", "")
	if err != nil {
		t.Fatal(err)
	}
	if !models.Status.Installed || models.Status.InstalledVersion != bundle.Models.Version {
		t.Fatal("model import did not record the bundled version")
	}
	cfg, err := os.ReadFile(filepath.Join(root, "config.toml"))
	if err != nil || !bytes.Contains(cfg, []byte("model = \"gpt-default-test\"")) ||
		!bytes.Contains(cfg, []byte("model_provider = \"my-provider\"")) || !bytes.Contains(cfg, []byte("goals = true")) {
		t.Fatal("model import changed existing fixture settings")
	}
	checkImportedModels(t, root, bundle.ModelCount)
	if closeCalls != 3 {
		t.Fatalf("closed %d times for three successful operations", closeCalls)
	}
}

func TestWindowsAssistantUpdatesModelsUsingDataOnly(t *testing.T) {
	bundle := testBundle(t)
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	script, err := bundle.modelScript()
	if err != nil {
		t.Fatal(err)
	}
	catalog, ok := readLegacyCatalogs(script)
	if !ok {
		t.Fatal("cannot load the repository's model fixture")
	}
	const testVersion = "99.1.0"
	catalog.Version = testVersion
	const testTitle = "Fixture model label from update"
	var mods struct {
		Models []map[string]json.RawMessage `json:"models"`
	}
	if err := json.Unmarshal(catalog.Domestic, &mods); err != nil {
		t.Fatal(err)
	}
	mods.Models[0]["display_name"], _ = json.Marshal(testTitle)
	catalog.Domestic, _ = json.Marshal(mods)
	payload, _ := json.Marshal(catalog)
	release := toolRelease{testVersion, "releases/v" + testVersion + "/catalog.json",
		fmt.Sprintf("%x", sha256.Sum256(payload)), int64(len(payload))}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case modelsBase + "latest.json":
			json.NewEncoder(w).Encode(release)
		case modelsBase + release.Path:
			w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := trustedClient(10 * time.Second)
	client.Transport = server.Client().Transport
	closeCalls := 0
	app := &App{ctx: t.Context(), toolRepo: toolRepository{client: client, origin: server.URL}, toolUpdates: make(map[string]toolRelease), closeClient: func() error { closeCalls++; return nil }}
	if _, err := app.InstallTool(modelsTool, "online", ""); err == nil {
		t.Fatal("online installation started before checking updates")
	}
	status, err := app.CheckToolUpdate(modelsTool)
	if err != nil || !status.OnlineAvailable || status.OnlineVersion != testVersion {
		t.Fatalf("did not find the online model package: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "config.toml")); !os.IsNotExist(err) {
		t.Fatal("checking for updates modified the configuration")
	}
	if closeCalls != 0 {
		t.Fatal("checking for updates closed the client")
	}
	result, err := app.InstallTool(modelsTool, "online", "")
	if err != nil || result.Status.InstalledVersion != testVersion || result.Status.CanUseBundled || closeCalls != 1 {
		t.Fatalf("online update failed or still offered downgrade: %v", err)
	}
	data := checkImportedModels(t, root, bundle.ModelCount)
	if !bytes.Contains(data, []byte(testTitle)) {
		t.Fatal("downloaded catalog was not used")
	}
}

func checkImportedModels(t *testing.T, root string, count int) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "myautoapi-models.json"))
	if err != nil {
		t.Fatal(err)
	}
	var models struct{ Models []struct{ Slug string } }
	if err := json.Unmarshal(data, &models); err != nil {
		t.Fatal(err)
	}
	if len(models.Models) <= count || !bytes.Contains(data, []byte("gpt-")) {
		t.Fatal("GPT models were not retained")
	}
	var managed []string
	ids, err := os.ReadFile(filepath.Join(root, "myautoapi-models.managed.json"))
	if err != nil || json.Unmarshal(ids, &managed) != nil || len(managed) != count {
		t.Fatal("did not import exactly the bundled domestic models")
	}
	return data
}

func TestCodexButtonsRejectMissingPresetModelsBeforeSaving(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	config := filepath.Join(root, "config.toml")
	original := []byte("model = \"existing-provider-model\"\n")
	if err := os.WriteFile(config, original, 0600); err != nil {
		t.Fatal(err)
	}
	const key = "sk-codex-preset-test-fixture"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer "+key {
			t.Error("expected read-only validation with the fixture key")
		}
		switch r.URL.Path {
		case "/v1/usage":
			fmt.Fprint(w, `{"isValid":true}`)
		case "/v1/models":
			fmt.Fprintf(w, `{"data":[{"id":%q},{"id":%q}]}`, gpt62Model, defaultImageModel)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	app := &App{ctx: t.Context(), gateway: keyGateway{client: server.Client(), origin: server.URL}, closeClient: func() error {
		t.Fatal("rejected GPT key should not close Codex")
		return nil
	}}
	for _, target := range []string{"codex", "codex-gpt62"} {
		_, err := app.Configure(target, key)
		if err == nil || !strings.Contains(err.Error(), gpt62Astra) {
			t.Fatalf("%s did not reject a key missing Astra: %v", target, err)
		}
		actual, err := os.ReadFile(config)
		if err != nil || !bytes.Equal(actual, original) {
			t.Fatalf("%s modified configuration after validation failed", target)
		}
	}
}
