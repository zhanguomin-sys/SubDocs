package main

import (
	"context"
	"os"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"myautoapi-quick-connect/connect"
)

// App is the bridge the page calls through window.go.main.App. The page
// never passes an address: links it may open are fixed here.
type App struct {
	ctx     context.Context
	service *connect.Service

	mu          sync.Mutex
	pendingLink string
	pageReady   bool
}

func newApp(service *connect.Service, launchLink string) *App {
	return &App{service: service, pendingLink: launchLink}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if exe, err := os.Executable(); err == nil {
		_ = connect.RegisterProtocol(exe)
	}
}

// secondInstance receives the arguments of a later launch, normally a link
// opened from the website, and brings this window to the front.
func (a *App) secondInstance(data options.SecondInstanceData) {
	if a.ctx == nil {
		return
	}
	runtime.WindowUnminimise(a.ctx)
	runtime.WindowShow(a.ctx)
	runtime.WindowSetAlwaysOnTop(a.ctx, true)
	runtime.WindowSetAlwaysOnTop(a.ctx, false)
	link := connect.FindImportLink(data.Args)
	if link == "" {
		return
	}
	a.mu.Lock()
	ready := a.pageReady
	if !ready {
		a.pendingLink = link
	}
	a.mu.Unlock()
	if ready {
		runtime.EventsEmit(a.ctx, "import-link", link)
	}
}

// Info is static product information for the page.
type Info struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	QQ      string `json:"qq"`
	QQName  string `json:"qqName"`
}

func (a *App) Info() Info {
	return Info{Name: connect.ProductName, Version: connect.Version, QQ: connect.QQGroupNumber, QQName: connect.QQGroupName}
}

// TakeLaunchLink returns the link this window was started with, once, and
// marks the page ready for links from later launches.
func (a *App) TakeLaunchLink() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pageReady = true
	link := a.pendingLink
	a.pendingLink = ""
	return link
}

func (a *App) Status() connect.Status { return a.service.Status() }

func (a *App) ParseLink(raw string) connect.LinkResult { return a.service.ParseLink(raw) }

// Connect is the button: a pasted key has no name.
func (a *App) Connect(key string) connect.Result { return a.service.Connect(a.ctx, key, "") }

// Import connects a key from an import link and remembers its name.
func (a *App) Import(key, name string) connect.Result { return a.service.Connect(a.ctx, key, name) }

func (a *App) Restore() connect.Result { return a.service.Restore() }

func (a *App) ImageStatus() connect.ImageStatus { return a.service.ImageStatus() }

func (a *App) ModelsStatus() connect.ModelsStatus { return a.service.ModelsStatus() }

// ImportModels runs the domestic model import after the window asked for
// confirmation; RemoveModels takes the imported models out again.
func (a *App) ImportModels() connect.Result { return a.service.ImportModels(a.ctx, false) }

func (a *App) RemoveModels() connect.Result { return a.service.ImportModels(a.ctx, true) }

// InstallImage is the button in image mode: a pasted key has no name.
func (a *App) InstallImage(key string) connect.Result {
	return a.service.InstallImage(a.ctx, key, "")
}

// UpdateImage reinstalls the image tool with the key it already holds.
func (a *App) UpdateImage() connect.Result { return a.service.UpdateImage(a.ctx) }

// ImportImage installs the image tool with a key from an import link that
// arrived while the window was in image mode.
func (a *App) ImportImage(key, name string) connect.Result {
	return a.service.InstallImage(a.ctx, key, name)
}

// ClipboardKey returns the clipboard text when it is a MyAutoAPI key.
func (a *App) ClipboardKey() string {
	text, err := runtime.ClipboardGetText(a.ctx)
	if err != nil {
		return ""
	}
	if key, ok := connect.NormalizeKey(text); ok {
		return key
	}
	return ""
}

func (a *App) ClaudeStatus() connect.ClaudeStatus { return a.service.ClaudeStatus() }

// ConnectClaude is the button on the Claude page: a pasted key has no name.
func (a *App) ConnectClaude(key string) connect.Result { return a.connectClaude(key, "") }

// ImportClaude connects Claude Desktop with a key from an app=claude link.
func (a *App) ImportClaude(key, name string) connect.Result { return a.connectClaude(key, name) }

// connectClaude connects Claude Desktop, then, when the Claude Code CLI is
// installed and not yet on this key, asks whether to connect it too.
func (a *App) connectClaude(key, name string) connect.Result {
	result := a.service.ConnectClaude(a.ctx, key, name)
	if !result.OfferCLI {
		return result
	}
	result.OfferCLI = false
	answer, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.QuestionDialog,
		Title:   "检测到 Claude Code CLI",
		Message: "检测到 Claude Code CLI，是否同时接入 MyAutoAPI？\n\n会把同一个密钥写入 Claude Code CLI 的设置（用户文件夹下的 .claude\\settings.json），其它设置保留。",
	})
	if err != nil || (answer != "Yes" && answer != "是") {
		return result
	}
	cli := a.service.ConnectClaudeCode(key)
	cli.Message = result.Message + "\n" + cli.Message
	if result.Tone != connect.ToneSuccess && cli.Tone == connect.ToneSuccess {
		cli.Tone = result.Tone
	}
	return cli
}

func (a *App) RestoreClaude() connect.Result { return a.service.RestoreClaude() }

// OpenDownload opens the official download page of "codex" (ChatGPT, which
// carries Codex) or "claude". Addresses are fixed here.
func (a *App) OpenDownload(app string) {
	switch app {
	case "codex":
		runtime.BrowserOpenURL(a.ctx, connect.ChatGPTDownloadURL)
	case "claude":
		runtime.BrowserOpenURL(a.ctx, connect.ClaudeDownloadURL)
	}
}

func (a *App) CopyGroupNumber() bool {
	return runtime.ClipboardSetText(a.ctx, connect.QQGroupNumber) == nil
}

func (a *App) JoinGroup() { runtime.BrowserOpenURL(a.ctx, connect.QQGroupURL) }

func (a *App) OpenSite() { runtime.BrowserOpenURL(a.ctx, connect.OfficialSite) }
