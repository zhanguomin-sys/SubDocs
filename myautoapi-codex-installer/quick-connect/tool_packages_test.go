package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func testBundle(t *testing.T) toolBundle {
	t.Helper()
	if embeddedPackages == nil {
		t.Skip("prepare the repository bundle first")
	}
	bundle, err := readToolBundle(embeddedPackages)
	if err != nil {
		t.Fatal(err)
	}
	return bundle
}

func TestBundleAndLegacyModelQuery(t *testing.T) {
	bundle := testBundle(t)
	if _, err := bundle.imageBytes(); err != nil {
		t.Fatal(err)
	}
	script, err := bundle.modelScript()
	if err != nil {
		t.Fatal(err)
	}
	legacy := bytes.ReplaceAll(append(bytes.Clone(script), []byte("\n# Older installer wrapper\n")...), []byte("\n"), []byte("\r\n"))
	catalogs, ok := readLegacyCatalogs(script)
	catalogs.Version = bundle.Models.Version
	data, _ := json.Marshal(catalogs)
	if !ok || !validModelCatalog(data, bundle.Models.Version) {
		t.Fatal("repository installer did not provide valid catalogs")
	}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/install/codex-models.ps1" {
			// This line must be ignored as PowerShell and only the data compared.
			w.Write(legacy)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	client := trustedClient(time.Second * 5)
	client.Transport = srv.Client().Transport
	repo := toolRepository{client: client, origin: srv.URL}
	release, err := repo.latest(t.Context(), modelsTool, bundle)
	if err != nil || release != bundle.Models {
		t.Fatalf("public script fallback did not match the packaged catalogs: %v", err)
	}
}

func TestOfficialToolUpdateRejectsCorruptionAndRedirect(t *testing.T) {
	payload := append([]byte("MZ"), bytes.Repeat([]byte{12}, 1800)...)
	release := toolRelease{"0.9.10", "releases/v0.9.10-upstream-abcdef1/myautoapi-imagines-tool-v0.9.10-windows-amd64.exe",
		fmt.Sprintf("%x", sha256.Sum256(payload)), int64(len(payload))}
	var mode atomic.Int32
	var redirected atomic.Bool
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("version checks and downloads must not send credentials")
		}
		switch r.URL.Path {
		case imageBase + "latest.json":
			_ = json.NewEncoder(w).Encode(map[string]any{"version": release.Version, "files": map[string]toolRelease{"windows/amd64": release}})
		case imageBase + release.Path:
			switch mode.Load() {
			case 1:
				w.Write(bytes.Repeat([]byte{7}, len(payload)))
			case 2:
				http.Redirect(w, r, "/redirect", http.StatusTemporaryRedirect)
			default:
				w.Write(payload)
			}
		case "/redirect":
			redirected.Store(true)
			w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	client := trustedClient(time.Second * 5)
	client.Transport = srv.Client().Transport
	repo := toolRepository{client: client, origin: srv.URL}
	got, err := repo.latest(t.Context(), imageTool, toolBundle{})
	if err != nil || got != release {
		t.Fatalf("did not select the Windows x64 release: %v", err)
	}
	if got, err := repo.download(t.Context(), imageTool, release); err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("failed to download a matching release: %v", err)
	}
	mode.Store(1)
	if _, err := repo.download(t.Context(), imageTool, release); err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatal("accepted a corrupt EXE")
	}
	mode.Store(2)
	if _, err := repo.download(t.Context(), imageTool, release); err == nil || redirected.Load() {
		t.Fatal("followed an unapproved download redirect")
	}
	bad := release
	bad.Path = "releases/v0.9.10/../../other.exe"
	if _, err := repo.download(t.Context(), imageTool, bad); err == nil {
		t.Fatal("accepted a path outside the expected Windows release format")
	}
}
