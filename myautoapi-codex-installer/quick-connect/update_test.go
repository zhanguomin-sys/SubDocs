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
	"sync"
	"testing"
)

func TestTrustedUpdateChecksPathVersionSizeAndHash(t *testing.T) {
	valid := append([]byte("MZ"), bytes.Repeat([]byte{9}, 2046)...)
	invalid := append([]byte("MZ"), bytes.Repeat([]byte{1}, 2046)...)
	var major, minor, patch int
	if _, err := fmt.Sscanf(version, "%d.%d.%d", &major, &minor, &patch); err != nil {
		t.Fatal(err)
	}
	release := Release{Version: fmt.Sprintf("%d.%d.%d", major, minor, patch+1), SHA256: fmt.Sprintf("%x", sha256.Sum256(valid)), Size: int64(len(valid))}
	release.Path = "/download/quick-connect/" + release.filename()
	payload, manifestRelease := valid, release
	var state sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state.Lock()
		defer state.Unlock()
		switch r.URL.Path {
		case manifestPath:
			_ = json.NewEncoder(w).Encode(manifestRelease)
		case release.Path:
			_, _ = w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	u := updater{http: server.Client(), origin: server.URL}
	latest, err := u.readLatest(t.Context())
	if err != nil || !latest.newer() {
		t.Fatalf("valid update was rejected: %v", err)
	}
	dir := t.TempDir()
	destination := filepath.Join(dir, "downloaded.exe")
	if err := os.WriteFile(destination, []byte("previous"), 0600); err != nil {
		t.Fatal(err)
	}
	state.Lock()
	payload = invalid
	state.Unlock()
	if err := u.download(t.Context(), release, destination); err == nil {
		t.Fatal("did not reject a corrupt binary")
	}
	if data, _ := os.ReadFile(destination); string(data) != "previous" {
		t.Fatal("failed download changed the destination")
	}
	state.Lock()
	payload = valid
	state.Unlock()
	if err := u.download(t.Context(), release, destination); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(destination); !bytes.Equal(data, valid) {
		t.Fatal("verified binary was not published")
	}
	state.Lock()
	manifestRelease.Path = "https://elsewhere.invalid/" + release.filename()
	state.Unlock()
	if _, err := u.readLatest(t.Context()); err == nil {
		t.Fatal("accepted an untrusted download URL")
	}
	same := release
	same.Version = version
	same.Path = "/download/quick-connect/" + same.filename()
	if same.newer() {
		t.Fatal("offered the same version again")
	}
}

func TestUpdaterRejectsRedirects(t *testing.T) {
	redirected := make(chan struct{}, 1)
	testEndpoint := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == manifestPath {
			http.Redirect(w, r, "/redirected", http.StatusFound)
			return
		}
		redirected <- struct{}{}
	}))
	defer testEndpoint.Close()
	client := trustedClient(0)
	client.Transport = testEndpoint.Client().Transport
	u := updater{http: client, origin: testEndpoint.URL}
	if _, err := u.readLatest(t.Context()); err == nil {
		t.Fatal("accepted a redirect")
	}
	select {
	case <-redirected:
		t.Fatal("followed a redirect")
	default:
	}
}
