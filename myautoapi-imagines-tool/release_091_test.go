package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestV091ModelSetAndGIFBoundaries(t *testing.T) {
	if len(supportedImageModels) != 10 || defaultModelName != "gpt-image-2.6-flare" {
		t.Fatalf("unexpected model set: %v", supportedImageModels)
	}
	for _, model := range []string{flareModelName, sunburstModelName, legacyFlareModelName, legacySunburstModelName} {
		static, err := parseGenerationOptions([]string{"--prompt", "synthetic", "--model", model, "--quality", "max"})
		if err != nil || static.size != "2048x2048" || static.outputFormat != "png" {
			t.Fatalf("static compatibility failed for %s: %+v %v", model, static, err)
		}
	}
	for _, model := range []string{flareModelName, sunburstModelName} {
		options, err := parseGenerationOptions([]string{"--prompt", "synthetic", "--model", model, "--output-format", "gif"})
		if err != nil || options.size != "auto" || options.quality != "low" || options.n != 1 || options.frameCount != 9 || options.frameDurationMS != 200 {
			t.Fatalf("GIF defaults: %+v %v", options, err)
		}
		payload := generationPayload(options)
		if payload["frame_count"] != 9 || payload["frame_duration_ms"] != 200 || payload["output_format"] != "gif" {
			t.Fatalf("GIF payload: %v", payload)
		}
	}
	invalid := [][]string{
		{"--model", legacyFlareModelName, "--output-format", "gif"},
		{"--model", legacySunburstModelName, "--output-format", "gif"},
		{"--model", modelName, "--output-format", "gif"},
		{"--model", banana2ModelName, "--output-format", "gif"},
		{"--output-format", "gif", "--frame-count", "1"},
		{"--output-format", "gif", "--frame-count", "17"},
		{"--output-format", "gif", "--frame-duration-ms", "19"},
		{"--output-format", "gif", "--frame-duration-ms", "2001"},
		{"--output-format", "gif", "--image", "synthetic.png"},
		{"--frame-count", "9"}, {"--frame-duration-ms", "200"},
	}
	for _, arguments := range invalid {
		if _, err := parseGenerationOptions(append([]string{"--prompt", "synthetic"}, arguments...)); err == nil {
			t.Fatalf("invalid GIF options accepted: %v", arguments)
		}
	}
	if _, err := parseEditOptions([]string{"--prompt", "synthetic", "--image", "synthetic.png", "--output-format", "gif"}); err == nil {
		t.Fatal("GIF edit accepted")
	}
	static, _ := parseGenerationOptions([]string{"--prompt", "synthetic"})
	if _, ok := generationPayload(static)["frame_count"]; ok {
		t.Fatal("static request sent animation parameters")
	}
}

func syntheticGIF(t *testing.T) []byte {
	t.Helper()
	palette := color.Palette{color.Black, color.White}
	frame := image.NewPaletted(image.Rect(0, 0, 2, 2), palette)
	frame.SetColorIndex(0, 0, 1)
	var buffer bytes.Buffer
	if err := gif.EncodeAll(&buffer, &gif.GIF{Image: []*image.Paletted{frame, frame}, Delay: []int{12, 15}, LoopCount: 0}); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestV091GIFReportsActualMetadataAndKeepsBytes(t *testing.T) {
	root := prepareNativeAuth(t)
	content := syntheticGIF(t)
	options, err := parseGenerationOptions([]string{"--prompt", "synthetic", "--output-format", "gif", "--output-dir", filepath.Join(root, "images"), "--output-name", "循环"})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{"data": []any{map[string]any{"b64_json": base64.StdEncoding.EncodeToString(content)}}})
	calls := 0
	client := release082Doer(func(request *http.Request) (*http.Response, error) {
		calls++
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["frame_count"] != float64(9) || payload["frame_duration_ms"] != float64(200) || payload["size"] != "auto" || payload["quality"] != "low" {
			t.Fatalf("unexpected GIF request: %v", payload)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body))}, nil
	})
	var output bytes.Buffer
	if err := runGeneration(options, &output, client, "https://synthetic.invalid"); err != nil {
		t.Fatal(err)
	}
	var result struct {
		OutputFile string          `json:"output_file"`
		Animations []gifOutputInfo `json:"animations"`
		Warnings   []string        `json:"warnings"`
	}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || filepath.Ext(result.OutputFile) != ".gif" || !bytes.Equal(content, mustReadFile(t, result.OutputFile)) {
		t.Fatalf("GIF was retried/modified: %s", output.String())
	}
	if len(result.Animations) != 1 {
		t.Fatalf("actual animation info missing: %s", output.String())
	}
	animation := result.Animations[0]
	if animation.Width != 2 || animation.Height != 2 || animation.FrameCount != 2 || animation.DurationMS != 270 || animation.LoopCount != 0 || len(animation.FrameDurationsMS) != 2 || animation.FrameDurationsMS[0] != 120 || animation.FrameDurationsMS[1] != 150 || animation.OutputFile != result.OutputFile || len(result.Warnings) < 2 {
		t.Fatalf("incorrect actual GIF metadata: %s", output.String())
	}
	unsafeGIF := append([]byte(nil), content...)
	for i := 13; i+9 < len(unsafeGIF); i++ {
		if unsafeGIF[i] == 0x2c {
			unsafeGIF[i+5], unsafeGIF[i+6], unsafeGIF[i+7], unsafeGIF[i+8] = 255, 255, 255, 255
			break
		}
	}
	if _, err := inspectGIF(unsafeGIF); err == nil {
		t.Fatal("unbounded GIF decode accepted")
	}
}

func TestV091RulesInstallUpgradeAndUninstallPreserveOtherRules(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	layout := buildInstallLayout(root)
	if err := os.MkdirAll(layout.rulesDir, 0o700); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(layout.rulesDir, "user.rules")
	if err := os.WriteFile(other, []byte("# personal rule\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := installWithTestKey(&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	policy := string(mustReadFile(t, layout.rulesPath))
	for _, command := range []string{"generate", "edit", "models", "paths", "doctor", "recover", "version"} {
		if !strings.Contains(policy, "pattern = ["+strconv.Quote(installedToolPath(layout))+", "+strconv.Quote(command)+"]") {
			t.Fatalf("missing precise prefix for %s: %s", command, policy)
		}
	}
	for _, forbidden := range []string{`"install"`, `"uninstall"`, `"bash"`, `"powershell"`, `"pwsh"`, `"sh"`, `"python"`, "pattern = [" + strconv.Quote(installedToolPath(layout)) + "]"} {
		if strings.Contains(policy, forbidden) {
			t.Fatalf("overbroad automatic allow list: %s", policy)
		}
	}
	var state installState
	if err := json.Unmarshal(mustReadFile(t, layout.statePath), &state); err != nil {
		t.Fatal(err)
	}
	state.Version = "0.8.2"
	delete(state.Files, execPolicyLogicalPath)
	stateContent, _ := json.Marshal(state)
	if err := os.WriteFile(layout.statePath, stateContent, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(layout.rulesPath); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := installWithTestKey(&output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "升级完成：V0.9.2") || !strings.Contains(output.String(), "完全退出") {
		t.Fatalf("upgrade/restart message missing: %s", output.String())
	}
	if !regularFileExists(layout.rulesPath) {
		t.Fatal("upgrade did not install exec policy")
	}
	if err := uninstall(&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if pathExists(layout.rulesPath) || string(mustReadFile(t, other)) != "# personal rule\n" {
		t.Fatal("uninstall damaged unrelated rules or kept own rule")
	}
}

func TestV091RulesRejectUnknownTargetsAndRollback(t *testing.T) {
	t.Run("unknown", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("CODEX_HOME", root)
		layout := buildInstallLayout(root)
		if err := os.MkdirAll(layout.rulesDir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(layout.rulesPath, []byte("personal"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := installWithTestKey(&bytes.Buffer{}); err == nil {
			t.Fatal("unknown rule overwritten")
		}
		if string(mustReadFile(t, layout.rulesPath)) != "personal" || pathExists(layout.binaryPath) {
			t.Fatal("failed install mutated existing content")
		}
	})
	t.Run("rollback", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("CODEX_HOME", root)
		layout := buildInstallLayout(root)
		err := installWithOptions(&bytes.Buffer{}, strings.NewReader("synthetic-key\n"), installOptions{failureHook: func(string) error { return errors.New("synthetic failure") }})
		if err == nil || pathExists(layout.rulesPath) || pathExists(layout.binaryPath) || pathExists(layout.rulesDir) {
			t.Fatal("new rules were not rolled back")
		}
	})
	if runtime.GOOS != "windows" {
		t.Run("symlink", func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("CODEX_HOME", root)
			layout := buildInstallLayout(root)
			if err := os.Symlink(t.TempDir(), layout.rulesDir); err != nil {
				t.Fatal(err)
			}
			if err := installWithTestKey(&bytes.Buffer{}); err == nil {
				t.Fatal("symlink rule directory accepted")
			}
		})
	}
}

func TestV091RenderedSkillQuotesSpecialPaths(t *testing.T) {
	layout := buildInstallLayout(filepath.Join(t.TempDir(), "空 格'\"$`\\name"))
	path := installedToolPath(layout)
	got := renderSkillTemplate("{{MYAUTOAPI_IMAGE_TOOL_BASH}} generate\n$ToolPath = {{MYAUTOAPI_IMAGE_TOOL_POWERSHELL}}", layout)
	want := bashPathExpression(path) + " generate\n$ToolPath = " + powerShellPathExpression(path)
	if got != want || strings.Contains(got, "{{") {
		t.Fatalf("unsafe tool-path rendering: %q", got)
	}
	if !strings.Contains(bashPathExpression(path), "'\"'\"'") || !strings.Contains(powerShellPathExpression(path), "''") {
		t.Fatal("single quotes were not escaped")
	}
}

func TestV091ModelsPathsAndDoctorUseNativeBoundaries(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_HOME", root)
	if err := installWithTestKey(&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	client := release082Doer(func(request *http.Request) (*http.Response, error) {
		if request.Method == http.MethodHead {
			if request.URL.String() != "https://api.myautoapi.com" || request.Header.Get("Authorization") != "" {
				t.Fatal("doctor sent credentials or unexpected endpoint")
			}
			return &http.Response{StatusCode: 405, Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		if request.Method != http.MethodGet || request.URL.String() != apiModelsURL || request.Header.Get("Authorization") != "Bearer synthetic-model-key" {
			t.Fatal("model request did not use fixed native boundary")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"gpt-image-2.6-flare","display_name":"Flare"},{"id":"other-model"}]}`))}, nil
	})
	if err := requestModels(&output, client, apiModelsURL, "synthetic-model-key"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "synthetic-model-key") || !strings.Contains(output.String(), `"http_status":200`) || !strings.Contains(output.String(), `"tool_supported":true`) || !strings.Contains(output.String(), `"tool_supported":false`) {
		t.Fatalf("invalid native models output: %s", output.String())
	}
	output.Reset()
	if err := paths(nil, &output); err != nil {
		t.Fatal(err)
	}
	var pathResult map[string]any
	if err := json.Unmarshal(output.Bytes(), &pathResult); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"desktop", "desktop_images", "project_images", "tool_path"} {
		path, ok := pathResult[name].(string)
		if !ok || !filepath.IsAbs(path) {
			t.Fatalf("paths missing absolute %s: %v", name, pathResult)
		}
	}
	output.Reset()
	if err := runDoctor(filepath.Join(root, "images"), &output, client); err != nil {
		t.Fatal(err)
	}
	var report struct {
		OK     bool              `json:"ok"`
		Checks []diagnosticCheck `json:"checks"`
	}
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if !report.OK || len(report.Checks) < 8 {
		t.Fatalf("native diagnosis failed: %s", output.String())
	}
	if strings.Contains(output.String(), "synthetic-model-key") || strings.Contains(output.String(), "native-test-key") {
		t.Fatal("diagnosis exposed key content")
	}
}

type failingModelBody struct{ secret string }

func (body failingModelBody) Read([]byte) (int, error) {
	return 0, errors.New("synthetic read failure: " + body.secret)
}
func (body failingModelBody) Close() error { return nil }

func TestV091ModelReadErrorsRedactDedicatedKey(t *testing.T) {
	key := "synthetic-model-read-secret"
	client := release082Doer(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: failingModelBody{key}}, nil
	})
	err := requestModels(&bytes.Buffer{}, client, apiModelsURL, key)
	if err == nil || strings.Contains(err.Error(), key) {
		t.Fatalf("unsafe model body read error: %v", err)
	}
}

func TestV091RecoveryFiltersOnlyCurrentRunAndReportsGIF(t *testing.T) {
	root := t.TempDir()
	since := time.Now().Add(-time.Second)
	for _, name := range []string{"本次-1.gif", "本次-2-2.png", "其它.png", "本次-3.png"} {
		content := tinyPNG
		if strings.HasSuffix(name, ".gif") {
			content = syntheticGIF(t)
		}
		if err := os.WriteFile(filepath.Join(root, name), content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	old := since.Add(-time.Minute)
	if err := os.Chtimes(filepath.Join(root, "本次-3.png"), old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "本次-4.png"), 0o700); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	args := []string{"--output-dir", root, "--output-name", "本次", "--since", strconv.FormatInt(since.UnixMilli(), 10), "--n", "2"}
	if err := recoverOutputs(args, &output); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Count      int             `json:"output_count"`
		Files      []string        `json:"output_files"`
		Animations []gifOutputInfo `json:"animations"`
		Warnings   []string        `json:"warnings"`
	}
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Count != 2 || len(result.Files) != 2 || len(result.Animations) != 1 || result.Animations[0].FrameCount != 2 || len(result.Warnings) != 0 {
		t.Fatalf("wrong recovered run: %s", output.String())
	}
	if err := os.WriteFile(filepath.Join(root, "本次-5.png"), tinyPNG, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := recoverOutputs(args, &bytes.Buffer{}); err == nil {
		t.Fatal("ambiguous recovery arbitrarily chose recent files")
	}
}
