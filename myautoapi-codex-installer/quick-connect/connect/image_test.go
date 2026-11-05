package connect

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type fakeImageTool struct {
	version string
	err     error
	calls   []string
}

func (f *fakeImageTool) Version() string { return f.version }

func (f *fakeImageTool) Install(_ context.Context, home, key string) error {
	f.calls = append(f.calls, key)
	if f.err != nil {
		return f.err
	}
	dir := filepath.Join(home, "tools", imageToolDirectory)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	manifest := `{"schema_version":1,"managed_by":"myautoapi-imagines-tool","version":"` + f.version + `","files":{}}`
	if err := os.WriteFile(filepath.Join(dir, imageManifestName), []byte(manifest), 0o600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, imageKeyName), []byte(key), 0o600)
}

// fakeGateway answers /v1/usage and /v1/models like the site does.
func fakeGateway(t *testing.T, usageStatus int, usage string, modelsStatus int, models string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+fixtureKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/usage":
			w.WriteHeader(usageStatus)
			_, _ = w.Write([]byte(usage))
		case "/v1/models":
			w.WriteHeader(modelsStatus)
			_, _ = w.Write([]byte(models))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

const (
	imageModels  = `{"object":"list","data":[{"id":"gpt-image-2.6-flare"},{"id":"gpt-image-2.6-sunburst"}]}`
	codingModels = `{"object":"list","data":[{"id":"gpt-6.1-sol"},{"id":"gpt-image-2.5-flare"}]}`
	textModels   = `{"object":"list","data":[{"id":"gpt-6.1-sol"},{"id":"gpt-6.1-terra"}]}`
)

func TestInstallImageMessages(t *testing.T) {
	cases := []struct {
		name         string
		usageStatus  int
		usage        string
		modelsStatus int
		models       string
		key          string
		tone         string
		message      string
		installed    bool
	}{
		{"image group", 200, `{"isValid":true,"remaining":5}`, 200, imageModels, fixtureKey, ToneSuccess,
			"生图工具 V0.8.2 已装好 ✓ 重新打开 Codex 或新建任务，说“画一张白底红苹果”试试。", true},
		{"coding group that also lists image models counts", 200, `{"isValid":true}`, 200, codingModels, fixtureKey, ToneSuccess,
			"生图工具 V0.8.2 已装好 ✓ 重新打开 Codex 或新建任务，说“画一张白底红苹果”试试。", true},
		{"no gpt-image model", 200, `{"isValid":true}`, 200, textModels, fixtureKey, ToneError, MessageImageNoModel, false},
		{"empty model list", 200, `{"isValid":true}`, 200, `{"data":[]}`, fixtureKey, ToneError, MessageImageNoModel, false},
		{"zero balance still installs", 200, `{"isValid":true,"remaining":0}`, 200, imageModels, fixtureKey, ToneWarning, MessageImageNoBalance, true},
		{"zero balance: models refused", 200, `{"isValid":true,"remaining":0}`, 403,
			`{"code":"INSUFFICIENT_BALANCE","message":"Insufficient account balance"}`, fixtureKey, ToneError, MessageImageNeedBalance, false},
		{"zero balance: models 403 without code", 200, `{"isValid":true,"remaining":0}`, 403, `{}`, fixtureKey, ToneError, MessageImageNeedBalance, false},
		{"rejected", 200, `{}`, 200, imageModels, "sk-someone_else_key_0123456789", ToneError, MessageImageRejected, false},
		{"disabled", 200, `{"isValid":false}`, 200, imageModels, fixtureKey, ToneError, MessageImageDisabled, false},
		{"models server error", 200, `{"isValid":true}`, 502, `{}`, fixtureKey, ToneError,
			"MyAutoAPI 服务器暂时出错（HTTP 502），无法确认密钥。生图工具没有改动。稍后再试一次，仍不行请进 QQ 群联系客服。", false},
		{"models not JSON", 200, `{"isValid":true}`, 200, `<html>`, fixtureKey, ToneError,
			"MyAutoAPI 服务器暂时出错（返回内容异常），无法确认密钥。生图工具没有改动。稍后再试一次，仍不行请进 QQ 群联系客服。", false},
		{"bad format", 200, `{"isValid":true}`, 200, imageModels, "pk-abc", ToneError, MessageBadFormat, false},
		{"empty", 200, `{"isValid":true}`, 200, imageModels, " ", ToneError, MessageImageEmpty, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			server := fakeGateway(t, c.usageStatus, c.usage, c.modelsStatus, c.models)
			tool := &fakeImageTool{version: "0.8.2"}
			stops := 0
			home := t.TempDir()
			service := &Service{
				CodexHome:   filepath.Join(home, ".codex"),
				ImageHome:   filepath.Join(home, ".codex"),
				BaseURL:     server.URL,
				Client:      NewHTTPClient(),
				Now:         func() time.Time { return fixedTime },
				StopClients: func() StopResult { stops++; return StopResult{} },
				NamesFile:   filepath.Join(home, "names.json"),
				Image:       tool,
			}
			result := service.InstallImage(context.Background(), c.key, "画图")
			if result.Tone != c.tone || result.Message != c.message {
				t.Fatalf("got %s %q\nwant %s %q", result.Tone, result.Message, c.tone, c.message)
			}
			if stops != 0 {
				t.Fatal("installing the image tool must not close Codex")
			}
			if _, err := os.Stat(filepath.Join(service.CodexHome, "config.toml")); err == nil {
				t.Fatal("installing the image tool must not write Codex files")
			}
			if c.installed {
				if len(tool.calls) != 1 || tool.calls[0] != fixtureKey {
					t.Fatalf("installer calls %v", tool.calls)
				}
				if result.Image.State != ImageReady || result.Image.Key != MaskKey(fixtureKey) || result.Image.Name != "画图" {
					t.Fatalf("image card %+v", result.Image)
				}
			} else if len(tool.calls) != 0 {
				t.Fatalf("installer ran for a refused key: %v", tool.calls)
			}
			if (result.Action == ActionOpenSite) != (c.message == MessageImageEmpty) {
				t.Fatalf("action %q", result.Action)
			}
		})
	}
}

func TestInstallImageFailures(t *testing.T) {
	server := fakeGateway(t, 200, `{"isValid":true}`, 200, imageModels)
	home := t.TempDir()
	service := &Service{ImageHome: home, BaseURL: server.URL, Client: NewHTTPClient(), Now: time.Now}

	if got := service.InstallImage(context.Background(), fixtureKey, ""); got.Message != MessageImageNotBundled {
		t.Fatalf("no bundled tool: %q", got.Message)
	}
	service.Image = &fakeImageTool{version: "0.8.2", err: errors.New("目标目录无法确认归属")}
	got := service.InstallImage(context.Background(), fixtureKey, "")
	if got.Tone != ToneError || got.Message != "安装生图工具失败：目标目录无法确认归属。原有文件没有改动，可以重试。" {
		t.Fatalf("installer failure: %s %q", got.Tone, got.Message)
	}
}

func TestReadImageStatus(t *testing.T) {
	writeTool := func(t *testing.T, home, manifest, key, source string) {
		dir := filepath.Join(home, "tools", imageToolDirectory)
		if manifest != "" {
			writeFile(t, filepath.Join(dir, imageManifestName), manifest)
		}
		if key != "" {
			writeFile(t, filepath.Join(dir, imageKeyName), key+"\n")
		}
		if source != "" {
			writeFile(t, filepath.Join(dir, imageKeySourceName), source)
		}
	}
	managed := func(version string) string {
		return `{"managed_by":"myautoapi-imagines-tool","version":"` + version + `"}`
	}
	cases := []struct {
		name, manifest, key, source, auth string
		want                              ImageStatus
	}{
		{"not installed", "", "", "", "", ImageStatus{State: ImageNone, Bundled: "0.8.2"}},
		{"ready", managed("0.8.2"), fixtureKey, "dedicated-image-key", "",
			ImageStatus{State: ImageReady, Version: "0.8.2", Bundled: "0.8.2", Key: MaskKey(fixtureKey)}},
		{"older version", managed("0.8.1"), fixtureKey, "", "",
			ImageStatus{State: ImageOutdated, Version: "0.8.1", Bundled: "0.8.2", Key: MaskKey(fixtureKey)}},
		{"newer than bundled", managed("0.10.0"), fixtureKey, "", "",
			ImageStatus{State: ImageReady, Version: "0.10.0", Bundled: "0.8.2", Key: MaskKey(fixtureKey)}},
		{"key missing", managed("0.8.2"), "", "", "",
			ImageStatus{State: ImageNoKey, Version: "0.8.2", Bundled: "0.8.2"}},
		{"key shared with Codex", managed("0.8.2"), "", "auth-json", `{"OPENAI_API_KEY":"` + fixtureKey + `"}`,
			ImageStatus{State: ImageReady, Version: "0.8.2", Bundled: "0.8.2", Key: MaskKey(fixtureKey), Shared: true}},
		{"foreign manifest", `{"managed_by":"someone"}`, fixtureKey, "", "", ImageStatus{State: ImageUnmanaged, Bundled: "0.8.2"}},
		{"broken manifest", `{`, fixtureKey, "", "", ImageStatus{State: ImageUnmanaged, Bundled: "0.8.2"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			home := t.TempDir()
			writeTool(t, home, c.manifest, c.key, c.source)
			if c.auth != "" {
				writeFile(t, filepath.Join(home, "auth.json"), c.auth)
			}
			got, _ := ReadImageStatus(home, "0.8.2")
			if got != c.want {
				t.Fatalf("got %+v\nwant %+v", got, c.want)
			}
		})
	}
}

func TestImageCodexHomeFollowsCodexHome(t *testing.T) {
	t.Setenv("CODEX_HOME", " /tmp/somewhere/.codex-x ")
	if got, _ := ImageCodexHome(); got != filepath.Clean("/tmp/somewhere/.codex-x") {
		t.Fatalf("got %q", got)
	}
}

func TestInstallFailureHidesKey(t *testing.T) {
	output := "将安装图片工具\n安装失败：无法保存 " + fixtureKey + "：拒绝访问\n本次操作未完成；写入前已停止或已恢复本次修改。\n"
	got := installFailure(output, fixtureKey)
	if strings.Contains(got, fixtureKey) || !strings.Contains(got, MaskKey(fixtureKey)) {
		t.Fatalf("got %q", got)
	}
}

// TestBundledImageToolInstallsRelease runs the released Linux image tool from
// the parent repository's publish folder into a temporary CODEX_HOME, the
// same way the Windows window runs the Windows program.
func TestBundledImageToolInstallsRelease(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("发布目录里用于本测试的是 linux-amd64 程序")
	}
	releases := filepath.Join(installerRoot(t), "..", "docs", "public", "install", "myautoapi-imagines-tool", "releases")
	matches, _ := filepath.Glob(filepath.Join(releases, "v*", "myautoapi-imagines-tool-v*-linux-amd64"))
	if len(matches) == 0 {
		t.Skip("没有找到已发布的生图程序")
	}
	path := matches[len(matches)-1]
	program, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	release := strings.TrimPrefix(filepath.Base(filepath.Dir(path)), "v")
	sum := sha256.Sum256(program)
	tool := &BundledImageTool{Program: program, Release: release, SHA256: hex.EncodeToString(sum[:]), CacheDir: t.TempDir()}

	home := filepath.Join(t.TempDir(), ".codex")
	t.Setenv("HOME", t.TempDir()) // the tool must use CODEX_HOME, never the real home
	if err := tool.Install(context.Background(), home, fixtureKey); err != nil {
		t.Fatal(err)
	}
	status, key := ReadImageStatus(home, release)
	if status.State != ImageReady || status.Version != release || key != fixtureKey {
		t.Fatalf("after install: %+v", status)
	}
	if _, err := os.Stat(filepath.Join(home, "skills", "myautoapi-image-generation", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	// Running again replaces the key, as the tool does on every install.
	other := "sk-second_fixture_key_0123456789abcdef"
	if err := tool.Install(context.Background(), home, other); err != nil {
		t.Fatal(err)
	}
	if _, key := ReadImageStatus(home, release); key != other {
		t.Fatalf("key not replaced: %q", key)
	}

	tool.SHA256 = strings.Repeat("0", 64)
	if err := tool.Install(context.Background(), home, fixtureKey); err == nil || !strings.Contains(err.Error(), "校验不通过") {
		t.Fatalf("tampered program accepted: %v", err)
	}
}

func TestUpdateImage(t *testing.T) {
	server := fakeGateway(t, 200, `{"isValid":true,"remaining":5}`, 200, imageModels)
	home := t.TempDir()
	codexHome := filepath.Join(home, ".codex")
	old := &fakeImageTool{version: "0.8.1"}
	if err := old.Install(context.Background(), codexHome, fixtureKey); err != nil {
		t.Fatal(err)
	}
	tool := &fakeImageTool{version: "0.8.2"}
	service := &Service{
		CodexHome: codexHome,
		ImageHome: codexHome,
		BaseURL:   server.URL,
		Client:    NewHTTPClient(),
		Now:       func() time.Time { return fixedTime },
		NamesFile: filepath.Join(home, "names.json"),
		Image:     tool,
	}
	if status := service.ImageStatus(); status.State != ImageOutdated {
		t.Fatalf("before: %+v", status)
	}
	result := service.UpdateImage(context.Background())
	if result.Tone != ToneSuccess || len(tool.calls) != 1 || tool.calls[0] != fixtureKey || result.Image.State != ImageReady {
		t.Fatalf("update: %+v calls %v", result, tool.calls)
	}

	empty := &Service{ImageHome: t.TempDir(), Image: tool, Now: service.Now}
	if result := empty.UpdateImage(context.Background()); result.Message != MessageImageNoSavedKey || result.Action != ActionOpenSite {
		t.Fatalf("no key: %+v", result)
	}
}
