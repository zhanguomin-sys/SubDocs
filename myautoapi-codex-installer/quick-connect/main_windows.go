package main

import (
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"myautoapi-quick-connect/connect"
)

// singleInstanceID keeps one window per user; later launches hand their
// link to it.
const singleInstanceID = "myautoapi-quick-connect-5c0f2a7e"

func main() {
	// A link opened while the window exists starts a second process that only
	// forwards the link. Let the running window take the foreground.
	allowAnyForegroundWindow()

	codexHome, _ := connect.CodexHome()
	imageHome, _ := connect.ImageCodexHome()
	service := &connect.Service{
		CodexHome:   codexHome,
		BaseURL:     connect.BaseURL,
		Client:      connect.NewHTTPClient(),
		Now:         time.Now,
		StopClients: connect.StopClients,
		NamesFile:   connect.NamesFile(),
		ImageHome:   imageHome,
	}
	// A nil *BundledImageTool must not become a non-nil interface.
	if tool := bundledImageTool(filepath.Join(webviewDataPath(), "image-tool")); tool != nil {
		service.Image = tool
	}
	service.ModelsHome = imageHome // the script also follows CODEX_HOME
	if script := bundledModelsScript(filepath.Join(webviewDataPath(), "models-tool")); script != nil {
		service.Models = script
	}
	localAppData, _ := os.UserCacheDir()    // %LOCALAPPDATA%
	roamingAppData, _ := os.UserConfigDir() // %APPDATA%
	userHome, _ := os.UserHomeDir()
	service.Claude = connect.ClaudeDesktop{LocalAppData: localAppData}
	service.ClaudeCLI = connect.ClaudeCode{Home: userHome, AppData: roamingAppData}
	service.StopClaude = connect.StopClaudeDesktop
	app := newApp(service, connect.FindImportLink(os.Args[1:]))

	err := wails.Run(&options.App{
		Title:              connect.ProductName,
		Width:              420,
		Height:             690,
		DisableResize:      true,
		BackgroundColour:   &options.RGBA{R: 244, G: 247, B: 250, A: 255},
		AssetServer:        &assetserver.Options{Assets: frontendFiles()},
		OnStartup:          app.startup,
		Bind:               []interface{}{app},
		Logger:             logger.NewDefaultLogger(),
		LogLevel:           logger.ERROR,
		LogLevelProduction: logger.ERROR,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               singleInstanceID,
			OnSecondInstanceLaunch: app.secondInstance,
		},
		Windows: &windows.Options{
			Theme:               windows.Light,
			WebviewUserDataPath: webviewDataPath(),
			Messages:            chineseMessages(),
		},
	})
	if err != nil {
		showError(err.Error())
		os.Exit(1)
	}
}

// showError makes a failed start visible instead of a silent exit.
func showError(message string) {
	text, _ := syscall.UTF16PtrFromString(message)
	title, _ := syscall.UTF16PtrFromString(connect.ProductName)
	const mbIconWarning = 0x30
	_, _, _ = syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Call(0,
		uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), mbIconWarning)
}

func webviewDataPath() string {
	base, err := os.UserCacheDir() // %LOCALAPPDATA%
	if err != nil {
		return ""
	}
	return filepath.Join(base, "MyAutoAPI", "QuickConnect")
}

func allowAnyForegroundWindow() {
	const asfwAny = ^uintptr(0)
	_, _, _ = syscall.NewLazyDLL("user32.dll").NewProc("AllowSetForegroundWindow").Call(asfwAny)
}

// chineseMessages are shown by Wails before the window exists, for example
// when the WebView2 runtime is missing. The build uses the "browser"
// strategy: a message box, then the official download page.
func chineseMessages() *windows.Messages {
	const download = "https://developer.microsoft.com/zh-cn/microsoft-edge/webview2/"
	return &windows.Messages{
		InstallationRequired: "需要先安装 Microsoft Edge WebView2 运行库。点“确定”打开下载页面：" + download,
		UpdateRequired:       "需要更新 Microsoft Edge WebView2 运行库。点“确定”打开下载页面：" + download,
		MissingRequirements:  connect.ProductName,
		Webview2NotInstalled: "没有安装 Microsoft Edge WebView2 运行库。",
		Error:                connect.ProductName,
		FailedToInstall:      "请安装 WebView2 运行库后，再打开" + connect.ProductName + "。",
		DownloadPage:         "需要先安装 Microsoft Edge WebView2 运行库。点“确定”打开下载页面（" + download + "），安装后再打开助手。最低版本：",
		PressOKToInstall:     "点“确定”安装。",
		ContactAdmin:         "需要 Microsoft Edge WebView2 运行库，请联系电脑管理员安装。",
		InvalidFixedWebview2: "WebView2 运行库不可用，请重新安装：" + download,
		WebView2ProcessCrash: "WebView2 出错，请关闭后重新打开助手。",
	}
}
