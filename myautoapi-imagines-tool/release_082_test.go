package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPromptFilePreservesChineseQuotesAndNewlines(t *testing.T) {
	prompt := "画一张带有 \"MyAutoAPI\" 的海报\n保留 '中文' 和 $ 符号。"
	path := filepath.Join(t.TempDir(), "图片要求.txt")
	if err := os.WriteFile(path, append([]byte{0xef, 0xbb, 0xbf}, []byte(prompt)...), 0o600); err != nil {
		t.Fatal(err)
	}
	options, err := parseGenerationOptions([]string{"--prompt-file", path})
	if err != nil {
		t.Fatal(err)
	}
	if options.prompt != prompt {
		t.Fatalf("prompt changed: %q", options.prompt)
	}
	if _, err := parseGenerationOptions([]string{"--prompt", "另一段", "--prompt-file", path}); err == nil {
		t.Fatal("both prompt inputs were accepted")
	}
	if err := os.WriteFile(path, []byte{0xff}, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := parseGenerationOptions([]string{"--prompt-file", path}); err == nil {
		t.Fatal("invalid UTF-8 was accepted")
	}
}

func TestSavingContinuesAfterMiddleImageCannotBeSaved(t *testing.T) {
	root := t.TempDir()
	for index := 1; index <= 999; index++ {
		name := "画面-2.png"
		if index > 1 {
			name = fmt.Sprintf("画面-2-%d.png", index)
		}
		if err := os.WriteFile(filepath.Join(root, name), []byte("existing"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	paths, err := saveOutputImages(root, "画面", [][]byte{tinyPNG, tinyPNG, tinyPNG})
	if err == nil {
		t.Fatal("save failure was lost")
	}
	if len(paths) != 2 || filepath.Base(paths[1]) != "画面-3.png" {
		t.Fatalf("later image was not saved: %v, %v", paths, err)
	}
}

func TestGenerationReportsSaveFailureAndKeepsLaterImages(t *testing.T) {
	root := prepareNativeAuth(t)
	outputRoot := filepath.Join(root, "images")
	if err := os.MkdirAll(outputRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	for index := 1; index <= 999; index++ {
		name := "画面-2.png"
		if index > 1 {
			name = fmt.Sprintf("画面-2-%d.png", index)
		}
		if err := os.WriteFile(filepath.Join(outputRoot, name), []byte("existing"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	payload, _ := json.Marshal(map[string]any{"data": []any{map[string]any{"b64_json": base64.StdEncoding.EncodeToString(tinyPNG)}}})
	client := release082Doer(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(payload))}, nil
	})
	options := testGenerationOptions(outputRoot)
	options.n = 3
	options.outputName = "画面"
	var output bytes.Buffer
	if err := runGeneration(options, &output, client, "https://synthetic.invalid"); err != nil {
		t.Fatal(err)
	}
	var summary map[string]any
	if err := json.Unmarshal(output.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if summary["ok"] != true || summary["output_count"] != float64(2) || summary["requested_count"] != float64(3) {
		t.Fatalf("unexpected partial result: %v", summary)
	}
	if diagnostic, ok := summary["diagnostic"].(string); !ok || !strings.Contains(diagnostic, "第 2 张") {
		t.Fatalf("save reason missing: %v", summary)
	}
}

type release082Doer func(*http.Request) (*http.Response, error)

func (f release082Doer) Do(r *http.Request) (*http.Response, error) { return f(r) }

func TestImageDownloadErrorDoesNotExposeSignedURL(t *testing.T) {
	signed := "https://images.invalid/private.png?token=synthetic-signature"
	client := release082Doer(func(*http.Request) (*http.Response, error) {
		return nil, &url.Error{Op: "Get", URL: signed, Err: errors.New("connection refused")}
	})
	_, err := downloadImage(signed, client)
	if err == nil || strings.Contains(err.Error(), "synthetic-signature") || strings.Contains(err.Error(), "private.png") {
		t.Fatalf("unsafe download error: %v", err)
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("network reason lost: %v", err)
	}
}

func TestV081UpgradeAndDedicatedKeyPermissions(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	if err := installWithTestKey(&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	layout := buildInstallLayout(root)
	state := mustReadFile(t, layout.statePath)
	state = bytes.ReplaceAll(state, []byte(`"version": "`+version()+`"`), []byte(`"version": "0.8.1"`))
	if err := os.WriteFile(layout.statePath, state, 0o666); err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(layout.toolDir, keyFileName)
	if err := os.Chmod(keyPath, 0o666); err != nil {
		t.Fatal(err)
	}
	if err := installWithInput(&bytes.Buffer{}, strings.NewReader("synthetic-upgrade-key\n")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("key mode is %o", info.Mode().Perm())
	}
}
