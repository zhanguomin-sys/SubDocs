package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

//go:embed all:frontend
var assets embed.FS

type App struct {
	ctx    context.Context
	mutex  sync.Mutex
	update updater
	latest Release
	smoke  bool
	ready  bool
}

func (a *App) Started() string {
	if a.smoke {
		a.ready = true
		go runtime.Quit(a.ctx)
	}
	return version
}

func (a *App) OpenGuide(section string) {
	// The embedded interface opens only these published pages.
	slugs := map[string]string{
		"help":   "/quick-connect",
		"locale": "/codex-chinese",
		"models": "/codex-models",
		"images": "/images/codex-image.html",
		"claude": "/quick-connect#claude-桌面版",
		"keys":   "/quick-start",
	}
	if slug, ok := slugs[section]; ok {
		runtime.BrowserOpenURL(a.ctx, docs+slug)
	}
}

func codexPath() (string, error) {
	home := os.Getenv("CODEX_HOME")
	if home == "" {
		profile, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		home = filepath.Join(profile, ".codex")
	}
	return filepath.Join(home, "config.toml"), nil
}

func verifyKey(ctx context.Context, key string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api+"/v1/usage", nil)
	if err != nil {
		return errors.New("无法准备 Key 检查")
	}
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := trustedClient(12 * time.Second).Do(req)
	if err != nil {
		return errors.New("暂时无法连接 MyAutoAPI，请检查网络后重试")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return errors.New("Key 未通过验证，请查看 Key 是否启用并确认分组")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("MyAutoAPI 暂时无法验证 Key（HTTP %d）", resp.StatusCode)
	}
	return nil
}

func configureCodex(key string) (string, error) {
	path, err := codexPath()
	if err != nil {
		return "", err
	}
	old, exists, err := readConfig(path)
	if err != nil {
		return "", err
	}
	next, err := codexConfig(old)
	if err != nil {
		return "", err
	}

	env, _, err := registry.CreateKey(registry.CURRENT_USER, "Environment", registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return "", errors.New("无法保存当前 Windows 用户的接入 Key")
	}
	defer env.Close()
	previous, kind, oldErr := env.GetStringValue(keyVariable)
	if oldErr != nil && !errors.Is(oldErr, registry.ErrNotExist) {
		return "", errors.New("已有同名环境变量类型不兼容")
	}
	if err := env.SetStringValue(keyVariable, key); err != nil {
		return "", errors.New("无法保存当前 Windows 用户的接入 Key")
	}
	backup, err := saveConfig(path, old, exists, next)
	if err != nil {
		var undoErr error
		switch {
		case errors.Is(oldErr, registry.ErrNotExist):
			undoErr = env.DeleteValue(keyVariable)
		case kind == registry.EXPAND_SZ:
			undoErr = env.SetExpandStringValue(keyVariable, previous)
		default:
			undoErr = env.SetStringValue(keyVariable, previous)
		}
		if undoErr != nil {
			return "", errors.New("配置未写入，且无法恢复旧的 MYAUTOAPI_CODEX_KEY；请查看 Windows 用户环境变量")
		}
		return "", err
	}
	if name, e := windows.UTF16PtrFromString("Environment"); e == nil {
		var result uintptr
		fire := windows.NewLazySystemDLL("user32.dll").NewProc("SendMessageTimeoutW")
		// Notify the Windows shell so subsequently opened apps inherit the key.
		fire.Call(0xffff, 0x001a, 0, uintptr(unsafe.Pointer(name)), 2, 1000, uintptr(unsafe.Pointer(&result)))
	}
	return backup, nil
}

func report(message, backup string) string {
	if backup != "" {
		return message + "\n原配置备份：" + backup
	}
	return message
}

func (a *App) Configure(target, input string) (string, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if target != "codex" && target != "claude-code" {
		return "", errors.New("请选择受支持的客户端")
	}
	key, err := validateKey(input)
	if err != nil {
		return "", err
	}
	if err = verifyKey(a.ctx, key); err != nil {
		return "", err
	}
	if target == "codex" {
		backup, err := configureCodex(key)
		return report("配置已保存。请完全退出并重新打开 Codex。", backup), err
	}
	home := os.Getenv("CLAUDE_CONFIG_DIR")
	if home == "" {
		profile, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		home = filepath.Join(profile, ".claude")
	}
	backup, err := applyConfig(filepath.Join(home, "settings.json"), func(old []byte) ([]byte, error) {
		return claudeCodeConfig(old, key)
	})
	return report("配置已保存。请重新启动 Claude Code。", backup), err
}

func (a *App) SetChinese() (string, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	path, err := codexPath()
	if err != nil {
		return "", err
	}
	backup, err := applyConfig(path, localeConfig)
	return report("已保存简体中文偏好。请完全退出并重新打开 Codex；若仍显示英文，请查看中文设置指南。", backup), err
}

func (a *App) SaveClaudeTemplate() (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "保存 Claude 桌面版接入模板",
		DefaultFilename: "myautoapi-claude-desktop.json",
		Filters:         []runtime.FileFilter{{DisplayName: "Claude 配置 (*.json)", Pattern: "*.json"}},
	})
	if err != nil || path == "" {
		return "", err
	}
	if !strings.EqualFold(filepath.Ext(path), ".json") {
		return "", errors.New("请选择 .json 文件")
	}
	// This is the documented import format, not an installed profile.
	// The user enters their Key inside Claude Desktop's own settings.
	data := "{\n  \"inferenceProvider\": \"gateway\",\n  \"inferenceGatewayBaseUrl\": \"" + api + "\",\n  \"inferenceGatewayAuthScheme\": \"bearer\"\n}\n"
	_, err = applyConfig(path, func([]byte) ([]byte, error) { return []byte(data), nil })
	if err != nil {
		return "", err
	}
	return "模板已保存。请在 Claude 桌面版导入后填写对应分组的 Key。", nil
}

type UpdateStatus struct {
	Message   string `json:"message"`
	Available bool   `json:"available"`
}

func (a *App) CheckUpdate() (UpdateStatus, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.latest = Release{}
	latest, err := a.update.readLatest(a.ctx)
	if err != nil {
		return UpdateStatus{}, errors.New("无法获取更新，请稍后重试或打开下载页")
	}
	a.latest = latest
	if !latest.newer() {
		return UpdateStatus{Message: "当前已是最新版本"}, nil
	}
	return UpdateStatus{Message: "发现 " + latest.Version + "，可下载校验后的新版。", Available: true}, nil
}

func (a *App) DownloadUpdate() (string, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if !a.latest.newer() {
		return "", errors.New("请先检查更新")
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "保存新版接入助手",
		DefaultFilename: a.latest.filename(),
		Filters:         []runtime.FileFilter{{DisplayName: "Windows 应用 (*.exe)", Pattern: "*.exe"}},
	})
	if err != nil || path == "" {
		return "", err
	}
	if !strings.EqualFold(filepath.Ext(path), ".exe") {
		return "", errors.New("请选择 .exe 文件")
	}
	if err := a.update.download(a.ctx, a.latest, path); err != nil {
		return "", fmt.Errorf("未能保存新版：%w", err)
	}
	return "新版已下载并通过 SHA-256 校验。双击打开即可使用。\n" + path, nil
}

func main() {
	a := &App{update: updater{http: trustedClient(45 * time.Second), origin: docs},
		smoke: len(os.Args) == 2 && os.Args[1] == "--smoke-ui"}
	err := wails.Run(&options.App{
		Title: productName,
		Width: 780, Height: 730, MinWidth: 530, MinHeight: 600,
		StartHidden: a.smoke,
		AssetServer: &assetserver.Options{Assets: assets},
		OnStartup:   func(ctx context.Context) { a.ctx = ctx },
		Bind:        []interface{}{a},
	})
	if err != nil || (a.smoke && !a.ready) {
		os.Exit(1)
	}
}
