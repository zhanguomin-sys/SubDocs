package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

func connectionConfig(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var config map[string]any
	if _, err := toml.Decode(string(data), &config); err != nil {
		t.Fatal(err)
	}
	return config
}

func TestConnectAddsGPT62FromBundle(t *testing.T) {
	if embeddedPackages == nil {
		t.Skip("requires the release bundle")
	}
	official, err := bundledOfficialModels()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	next, catalog, err := prepareCodexConnection(path, nil, official)
	if err != nil {
		t.Fatal(err)
	}
	config := connectionConfig(t, next)
	if config["model_catalog_json"] != catalog.path || config["model"] != gpt62Model || config["model_reasoning_effort"] != gpt62Effort {
		t.Fatal("missing connection defaults")
	}
	got, err := readCodexModelCatalog(catalog.next)
	if err != nil {
		t.Fatal(err)
	}
	for _, slug := range []string{gpt62Model, gpt62Astra, "gpt-6.1-sol", "codex-auto-review"} {
		if _, ok := got.index[slug]; !ok {
			t.Errorf("missing model %s", slug)
		}
	}
	for _, slug := range []string{"qwen3.8-max", "deepseek-v4.1-flash-1017", "gpt-5.5"} {
		if _, ok := got.index[slug]; ok {
			t.Errorf("unexpected model %s", slug)
		}
	}
}

func TestConnectionPreservesExistingCatalogEntries(t *testing.T) {
	official := []byte(`{"models":[
		{"slug":"gpt-6.2-sol","visibility":"list","supported_in_api":true,"supported_reasoning_levels":[{"effort":"high"}]},
		{"slug":"gpt-6.2-astra","visibility":"list","supported_in_api":true,"supported_reasoning_levels":[{"effort":"high"}]},
		{"slug":"gpt-6.1-sol","visibility":"list","supported_in_api":true}
	]}`)
	existing := []byte(`{"models":[
		{"slug":"qwen-private","custom":{"keep":true}},
		{"slug":"gpt-6.2-sol","visibility":"list","supported_in_api":true,"supported_reasoning_levels":[{"effort":"high"}],"custom":"My Sol"},
		{"slug":"gpt-6.2-astra","visibility":"list","supported_in_api":true,"supported_reasoning_levels":[{"effort":"low"}]}
	],"owner":"my catalog"}`)
	for _, filename := range []string{"custom.json", domesticCatalogName} {
		t.Run(filename, func(t *testing.T) {
			root := t.TempDir()
			source := filepath.Join(root, filename)
			if err := os.WriteFile(source, existing, 0600); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "config.toml")
			old := []byte("model_provider = \"myautoapi\"\nmodel = \"qwen-private\"\nmodel_reasoning_effort = \"medium\"\nmodel_catalog_json = \"" + filename + "\"\n")
			next, catalog, err := prepareCodexConnection(path, old, official)
			if err != nil {
				t.Fatal(err)
			}
			if (catalog.path == source) != (filename == domesticCatalogName) {
				t.Fatal("should continue using the domestic installer's directory and preserve custom files")
			}
			config := connectionConfig(t, next)
			if config["model"] != gpt62Model || config["model_reasoning_effort"] != gpt62Effort {
				t.Fatal("connecting should update the previous default")
			}
			if config["model_catalog_json"] != catalog.path && config["model_catalog_json"] != filename {
				t.Fatal("config does not point to the merged catalog")
			}
			if _, err := saveConfig(catalog.path, catalog.old, catalog.exists, catalog.next); err != nil {
				t.Fatal(err)
			}
			if filename != domesticCatalogName {
				data, err := os.ReadFile(source)
				if err != nil || !bytes.Equal(data, existing) {
					t.Fatal("the custom source file was overwritten")
				}
			}
			got, err := readCodexModelCatalog(catalog.next)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.models) != 3 || got.index["qwen-private"] != 0 || got.index[gpt62Model] != 1 ||
				got.index[gpt62Astra] != 2 || !usableGPT62Model(got.models[2]) ||
				!bytes.Contains(got.models[1], []byte("My Sol")) || !bytes.Contains(got.fields["owner"], []byte("my catalog")) {
				t.Fatal("lost existing metadata or wrote duplicate models")
			}
			again, extra, err := prepareCodexConnection(path, next, official)
			if err != nil || extra.path != "" || !bytes.Equal(next, again) {
				t.Fatal("reconnecting should leave a complete catalog alone")
			}
		})
	}
}
