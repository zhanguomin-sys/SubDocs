package main

import (
	"io/fs"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestFrontendContent(t *testing.T) {
	files := frontendFiles()
	for _, name := range []string{"index.html", "style.css", "app.js", "logo.png", "penguin.png"} {
		if _, err := fs.Stat(files, name); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	page := readEmbedded(t, files, "index.html")
	for _, text := range []string{
		"<title>MyAutoAPI中转快速接入助手</title>",
		`My<span class="good">Auto</span>API`, "中转快速接入助手",
		"粘贴 API Key", "一键连接 Codex",
		"QQ 交流群", "685861462", "复制群号", "加入群聊", "新人进群领 2 元",
		"打开官网", "恢复官方登录", `id="version"`,
		"安装生图工具", "导入国产模型", "返回 Codex 接入", "移除国产模型",
	} {
		if !strings.Contains(page, text) {
			t.Errorf("index.html lacks %q", text)
		}
	}
	for _, banned := range []string{"注意事项", "二维码", "<iframe", "confirm(", "alert("} {
		for _, name := range []string{"index.html", "app.js"} {
			if strings.Contains(readEmbedded(t, files, name), banned) {
				t.Errorf("%s contains %q", name, banned)
			}
		}
	}
	// Everything ships inside the exe; the page loads nothing from the network.
	remote := regexp.MustCompile(`(?:src|href)\s*=\s*["']https?:|url\(\s*["']?https?:|import\s+.*https?:`)
	for _, name := range []string{"index.html", "style.css", "app.js"} {
		if remote.MatchString(readEmbedded(t, files, name)) {
			t.Errorf("%s loads a remote resource", name)
		}
	}
	css := readEmbedded(t, files, "style.css")
	for _, color := range []string{"#0d9488", "#0891b2", "#2563eb"} {
		if !strings.Contains(css, color) {
			t.Errorf("style.css lacks brand colour %s", color)
		}
	}
}

// Every bridge method the page calls exists on App, and the page never sends
// an address to open.
func TestFrontendCallsExistingMethods(t *testing.T) {
	script := readEmbedded(t, frontendFiles(), "app.js")
	bridge, err := os.ReadFile("app_windows.go")
	if err != nil {
		t.Fatal(err)
	}
	calls := regexp.MustCompile(`(?:bridge\(\)|app)\.([A-Z][A-Za-z]+)\(`).FindAllStringSubmatch(script, -1)
	if len(calls) < 8 {
		t.Fatalf("found only %d bridge calls", len(calls))
	}
	for _, call := range calls {
		if !strings.Contains(string(bridge), "func (a *App) "+call[1]+"(") {
			t.Errorf("app.js calls missing method %s", call[1])
		}
	}
	for _, open := range []string{"JoinGroup()", "OpenSite()"} {
		if !strings.Contains(script, open) {
			t.Errorf("app.js should call %s without arguments", open)
		}
	}
}

func readEmbedded(t *testing.T, files fs.FS, name string) string {
	t.Helper()
	data, err := fs.ReadFile(files, name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
