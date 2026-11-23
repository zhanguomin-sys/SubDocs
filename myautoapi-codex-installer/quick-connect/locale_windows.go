package main

import (
	"context"
	_ "embed"
	"errors"
	"strings"
	"time"
)

//go:embed locale/Enable-CodexI18nCache.ps1
var desktopLocaleEntry []byte

//go:embed locale/Enable-CodexI18nCurrentCache.ps1
var desktopLocaleCore []byte

// Include the upstream MIT notice in the distributed executable.
//
//go:embed locale/LICENSE
var desktopLocaleLicense []byte

func installDesktopLocale(ctx context.Context, root string, check bool) error {
	return stageInstallFiles(map[string][]byte{
		"Enable-CodexI18nCache.ps1":        desktopLocaleEntry,
		"Enable-CodexI18nCurrentCache.ps1": desktopLocaleCore,
		"LICENSE":                          desktopLocaleLicense,
	}, func(dir string) error {
		const entry = `& '.\Enable-CodexI18nCache.ps1'`
		command, marker, timeout := entry+" -BootstrapMissing", "MYAUTOAPI_LOCALE_READY", 5*time.Minute
		if check {
			command, marker, timeout = entry+" -CheckOnly", "MYAUTOAPI_LOCALE_FOUND", 20*time.Second
		}
		output, err := runEmbeddedPowerShell(ctx, timeout, root, dir, command)
		switch {
		case err == nil && hasLocaleMarker(output, marker):
			return nil
		case errors.Is(err, errScriptBlocked):
			return errors.New("Windows 的 PowerShell 策略阻止了中文设置脚本。")
		case check:
			return errors.New("未能识别当前 Windows 用户安装的 Codex 桌面版。")
		case hasLocaleMarker(output, "MYAUTOAPI_LOCALE_BUSY"):
			return errors.New("Codex 在设置期间重新打开了。请完全退出 Codex 后重试。")
		case hasLocaleMarker(output, "MYAUTOAPI_LOCALE_START_FAILED"):
			return errors.New("未能临时启动 Codex 来设置中文。")
		case hasLocaleMarker(output, "MYAUTOAPI_LOCALE_RENDERER_FAILED"):
			return errors.New("Codex 未加载设置中文所需的页面。请重试。")
		case hasLocaleMarker(output, "MYAUTOAPI_LOCALE_CACHE_FAILED"):
			return errors.New("未能写入或验证 Codex 的中文缓存。请重试。")
		case hasLocaleMarker(output, "MYAUTOAPI_LOCALE_RESTART_FAILED"):
			return errors.New("Codex 未能自动重新打开。请启动 Codex 检查界面语言。")
		case errors.Is(err, context.DeadlineExceeded):
			return errors.New("设置 Codex 中文超时。请重试。")
		default:
			return errors.New("中文设置脚本未返回成功结果。请重试。")
		}
	})
}

func hasLocaleMarker(output, marker string) bool {
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == marker {
			return true
		}
	}
	return false
}
