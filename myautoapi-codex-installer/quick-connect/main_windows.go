package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	winoptions "github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

//go:embed all:frontend
var assets embed.FS

type App struct {
	ctx         context.Context
	mutex       sync.Mutex
	update      updater
	latest      Release
	toolRepo    toolRepository
	gateway     keyGateway
	toolUpdates map[string]toolRelease
	smoke       bool
	ready       bool
	closeClient func() error // Tests supply an isolated client lifecycle.
	setupLocale func(context.Context, string, bool) error
}

type AppInfo struct {
	Version string `json:"version"`
	Domain  string `json:"domain"`
	QQGroup string `json:"qqGroup"`
}

func (a *App) Started() AppInfo {
	return AppInfo{Version: version, Domain: strings.TrimPrefix(site, "https://"), QQGroup: qqGroup}
}

func (a *App) FitWindow(contentHeight int) {
	if contentHeight <= 0 || contentHeight > 16384 {
		return
	}
	fitted := fitAssistantWindow(contentHeight, false)
	if a.smoke {
		a.ready = fitted
		go runtime.Quit(a.ctx)
	}
}

func (a *App) OpenGuide(section string) {
	// The embedded interface opens only these published pages.
	links := map[string]string{
		"site":   site,
		"keys":   site + "/keys",
		"help":   docs + "/quick-connect",
		"locale": docs + "/codex-chinese",
		"models": docs + "/codex-models",
		"images": docs + "/image-video-group-image",
		"qq":     qqInvite,
	}
	if link, ok := links[section]; ok {
		runtime.BrowserOpenURL(a.ctx, link)
	}
}

func (a *App) CopyGroup() error {
	return runtime.ClipboardSetText(a.ctx, qqGroup)
}

func (a *App) ClipboardAPIKey() string {
	if a.smoke {
		return ""
	}
	text, err := runtime.ClipboardGetText(a.ctx)
	if err != nil {
		return ""
	}
	key, err := validateKey(text)
	if err != nil {
		return ""
	}
	return key
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

func (a *App) ConnectionStatus(target string) (bool, error) {
	var configPath func() (string, error)
	var check func([]byte) (bool, error)
	switch target {
	case "codex":
		configPath, check = codexPath, codexUsesMyAutoAPI
	case "claude-code":
		if files, installed := findClaudeDesktop(); installed {
			return claudeDesktopConfigured(files)
		}
		configPath = func() (string, error) {
			settings, _, err := claudeCodePaths()
			return settings, err
		}
		check = claudeCodeUsesMyAutoAPI
	default:
		return false, errors.New("请选择受支持的客户端")
	}
	path, err := configPath()
	if err != nil {
		return false, errors.New("无法找到客户端配置")
	}
	data, exists, err := readConfig(path)
	if err != nil {
		return false, errors.New("无法读取客户端配置")
	}
	if !exists || len(data) == 0 {
		return false, nil
	}
	configured, err := check(data)
	if err != nil {
		return false, errors.New("无法识别客户端配置")
	}
	return configured, nil
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
	official, err := bundledOfficialModels()
	if err != nil {
		return "", err
	}
	next, catalog, err := prepareCodexConnection(path, old, official)
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
	var backup string
	var catalogSaved bool
	if catalog.path != "" {
		_, err = saveConfig(catalog.path, catalog.old, catalog.exists, catalog.next)
		catalogSaved = err == nil
		if err != nil {
			err = fmt.Errorf("无法写入 GPT-6.2 模型目录：%w", err)
		}
	}
	if err == nil {
		backup, err = saveConfig(path, old, exists, next)
	}
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
		if catalogSaved && catalog.active && catalog.exists {
			if _, restoreErr := saveConfig(catalog.path, catalog.next, true, catalog.old); restoreErr != nil {
				err = errors.Join(err, errors.New("无法自动恢复原模型目录，可使用目录旁的备份恢复"))
			}
		}
		if undoErr != nil {
			return "", errors.Join(err, errors.New("无法恢复旧的 MYAUTOAPI_CODEX_KEY；请查看 Windows 用户环境变量"))
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

func savedCodexKey() (string, error) {
	path, err := codexPath()
	if err != nil {
		return "", errors.New("无法找到 Codex 配置")
	}
	config, exists, err := readConfig(path)
	if err != nil || !exists {
		return "", errors.New("Codex 尚未配置 MyAutoAPI，请输入 GPT 分组 API Key")
	}
	env, err := registry.OpenKey(registry.CURRENT_USER, "Environment", registry.QUERY_VALUE)
	if err != nil {
		return "", errors.New("没有找到已保存的 GPT 分组 API Key，请重新输入")
	}
	defer env.Close()
	stored, _, err := env.GetStringValue(keyVariable)
	if err != nil {
		return "", errors.New("没有找到已保存的 GPT 分组 API Key，请重新输入")
	}
	return reusableCodexKey(config, stored)
}

func (a *App) finishCodexOperation(message string) string {
	closeClient := a.closeClient
	if closeClient == nil {
		closeClient = closeOpenAIClients
	}
	if err := closeClient(); err != nil {
		return message + "\n未能完全关闭 ChatGPT/Codex。请完全退出客户端后再打开。"
	}
	return message + "\nChatGPT/Codex 已完全退出。请重新打开客户端并新建任务。"
}

func (a *App) connectCodex(key string, save func(string) (string, error)) (string, error) {
	backup, err := save(key)
	if err != nil {
		return "", err
	}
	message := "已连接 MyAutoAPI。新任务默认使用 GPT-6.2 Sol，推理力度为 High。模型列表也包含 Astra。"
	return report(a.finishCodexOperation(message), backup), nil
}

func (a *App) Configure(target, input string) (string, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	gateway := a.gateway
	switch target {
	case "claude-code":
		return a.configureClaude(input)
	case "codex", "codex-gpt62":
	default:
		return "", errors.New("请选择受支持的客户端")
	}
	if strings.TrimSpace(input) == "" {
		key, err := savedCodexKey()
		if err != nil {
			return "", err
		}
		input = key
	}
	return configureWithKey(a.ctx, gateway.client, gateway.origin, input, func(key string) (string, error) {
		if err := gateway.requireModels(a.ctx, key, "GPT 分组", gpt62Model, gpt62Astra); err != nil {
			return "", fmt.Errorf("%w\nCodex 配置没有改动。", err)
		}
		return a.connectCodex(key, configureCodex)
	})
}

func (a *App) SetChinese() (string, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	path, err := codexPath()
	if err != nil {
		return "", err
	}
	old, exists, err := readConfig(path)
	if err != nil {
		return "", err
	}
	updated, err := localeConfig(old)
	if err != nil {
		return "", err
	}
	root, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return "", errors.New("无法找到 Codex 配置目录")
	}
	setupLocale := a.setupLocale
	if setupLocale == nil {
		setupLocale = installDesktopLocale
	}
	if err := setupLocale(a.ctx, root, true); err != nil {
		return "", err
	}
	if _, err := saveConfig(path, old, exists, updated); err != nil {
		return "", err
	}
	closeClient := a.closeClient
	if closeClient == nil {
		closeClient = closeOpenAIClients
	}
	if err := closeClient(); err != nil {
		if errors.Is(err, errStartedFromCodex) {
			return "", errors.New("接入助手由 Codex 启动，旧会话仍在运行。请退出 Codex，从资源管理器双击接入助手，再重试。")
		}
		return "", errors.New("Codex 的桌面或后台进程未能完全退出。请关闭 Codex 后重试。")
	}
	if err := setupLocale(a.ctx, root, false); err != nil {
		return "", err
	}
	return "已写入简体中文缓存并重新打开 Codex。", nil
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
		return UpdateStatus{Message: "当前没有可用的新版本。"}, nil
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
	a := &App{
		update:      updater{http: trustedClient(45 * time.Second), origin: docs},
		toolRepo:    toolRepository{client: trustedClient(35 * time.Second), origin: docs},
		gateway:     keyGateway{client: trustedClient(12 * time.Second), origin: api},
		toolUpdates: make(map[string]toolRelease),
		smoke:       len(os.Args) == 2 && os.Args[1] == "--smoke-ui",
	}
	err := wails.Run(&options.App{
		Title: productName,
		Width: assistantWindowWidth, Height: startupWindowHeight,
		DisableResize: true,
		StartHidden:   a.smoke,
		AssetServer:   &assetserver.Options{Assets: assets},
		Windows:       &winoptions.Options{WindowClassName: assistantWindowClass},
		OnStartup: func(ctx context.Context) {
			a.ctx = ctx
			fitAssistantWindow(0, true)
		},
		Bind: []interface{}{a},
	})
	if err != nil || (a.smoke && !a.ready) {
		os.Exit(1)
	}
}
