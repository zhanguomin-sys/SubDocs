package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

const (
	modelReceipt      = "myautoapi-models.release"
	defaultImageModel = "gpt-image-2.6-flare"
)

type ToolStatus struct {
	Installed        bool   `json:"installed"`
	InstalledVersion string `json:"installedVersion"`
	BundledVersion   string `json:"bundledVersion"`
	ModelCount       int    `json:"modelCount"`
	CanUseBundled    bool   `json:"canUseBundled"`
	OnlineVersion    string `json:"onlineVersion"`
	OnlineAvailable  bool   `json:"onlineAvailable"`
	Message          string `json:"message"`
}

type ToolResult struct {
	Status  ToolStatus `json:"status"`
	Message string     `json:"message"`
}

func regularNonempty(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
}

func readLocalToolFile(path string, limit int64) []byte {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > limit {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(b)) > limit {
		return nil
	}
	return b
}

func readToolVersion(path string) string {
	raw := strings.TrimSpace(string(readLocalToolFile(path, 96)))
	raw = strings.TrimPrefix(strings.TrimPrefix(raw, "v"), "V")
	if versionPattern.MatchString(raw) {
		return raw
	}
	return ""
}

func localToolStatus(root, kind string, bundle toolBundle) (ToolStatus, error) {
	var status ToolStatus
	switch kind {
	case imageTool:
		toolDir := filepath.Join(root, "tools", "myautoapi-imagines-tool")
		if regularNonempty(filepath.Join(toolDir, "myautoapi-imagines-tool.exe")) {
			status.Installed = true
			status.InstalledVersion = readToolVersion(filepath.Join(toolDir, "VERSION"))
			if status.InstalledVersion == "" {
				status.InstalledVersion = readToolVersion(filepath.Join(toolDir, "RELEASE"))
			}
		}
		status.BundledVersion = bundle.Image.Version
	case modelsTool:
		var ids []string
		idsJSON := readLocalToolFile(filepath.Join(root, "myautoapi-models.managed.json"), 64<<10)
		if json.Unmarshal(idsJSON, &ids) == nil && len(ids) > 0 && regularNonempty(filepath.Join(root, "myautoapi-models.json")) {
			status.Installed = true
			status.InstalledVersion = readToolVersion(filepath.Join(root, modelReceipt))
		}
		status.BundledVersion = bundle.Models.Version
		status.ModelCount = bundle.ModelCount
	default:
		return ToolStatus{}, errors.New("请选择受支持的工具")
	}
	status.CanUseBundled = !versionAfter(status.InstalledVersion, status.BundledVersion)
	return status, nil
}

func (s ToolStatus) baseline() string {
	if versionAfter(s.InstalledVersion, s.BundledVersion) {
		return s.InstalledVersion
	}
	return s.BundledVersion
}

func (a *App) toolEnvironment(kind string) (toolBundle, string, ToolStatus, error) {
	bundle, err := readToolBundle(embeddedPackages)
	if err != nil {
		return bundle, "", ToolStatus{}, err
	}
	config, err := codexPath()
	if err != nil {
		return bundle, "", ToolStatus{}, errors.New("无法找到 Codex 配置目录")
	}
	root, err := filepath.Abs(filepath.Dir(config))
	if err != nil {
		return bundle, "", ToolStatus{}, errors.New("无法找到 Codex 配置目录")
	}
	status, err := localToolStatus(root, kind, bundle)
	return bundle, root, status, err
}

func (a *App) GetToolStatus(kind string) (ToolStatus, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	_, _, status, err := a.toolEnvironment(kind)
	return status, err
}

func withOnlineStatus(kind string, status ToolStatus, release toolRelease) ToolStatus {
	if !release.valid(kind) {
		return status
	}
	status.OnlineVersion = release.Version
	switch {
	case versionAfter(release.Version, status.baseline()):
		status.OnlineAvailable = true
		status.Message = "发现新版 V" + release.Version + "。"
	case !status.Installed:
		status.Message = "可以直接使用内置安装包。"
	case status.InstalledVersion == "":
		status.Message = "本机旧安装未记录版本，可使用内置包重新导入。"
	case versionAfter(status.BundledVersion, status.InstalledVersion):
		status.Message = "内置包有更新，点下方按钮即可导入。"
	default:
		status.Message = "当前没有可用的新版本。"
	}
	return status
}

func (a *App) CheckToolUpdate(kind string) (ToolStatus, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	delete(a.toolUpdates, kind)
	bundle, _, status, err := a.toolEnvironment(kind)
	if err != nil {
		return status, err
	}
	release, err := a.toolRepo.latest(a.ctx, kind, bundle)
	if err != nil {
		return status, errors.New("暂时无法查询官网版本。请稍后重试，或继续使用内置包。")
	}
	a.toolUpdates[kind] = release
	return withOnlineStatus(kind, status, release), nil
}

func (a *App) InstallTool(kind, source, input string) (ToolResult, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	bundle, root, status, err := a.toolEnvironment(kind)
	if err != nil {
		return ToolResult{}, err
	}
	targetVersion := status.BundledVersion
	var release toolRelease
	switch source {
	case "bundled":
		if !status.CanUseBundled {
			return ToolResult{}, errors.New("本机版本比内置包更新，请先联网检查更新。")
		}
	case "online":
		release = a.toolUpdates[kind]
		if !withOnlineStatus(kind, status, release).OnlineAvailable {
			return ToolResult{}, errors.New("请先联网检查可安装的新版。")
		}
		targetVersion = release.Version
	default:
		return ToolResult{}, errors.New("请选择内置包或已检查的在线新版")
	}

	var key string
	if kind == imageTool {
		key, err = a.verifyImageInstallKey(root, input, status.Installed)
		if err != nil {
			return ToolResult{}, fmt.Errorf("%w\n绘图工具没有改动。", err)
		}
	}
	var onlinePayload []byte
	if source == "online" {
		onlinePayload, err = a.toolRepo.download(a.ctx, kind, release)
		if err != nil {
			return ToolResult{}, fmt.Errorf("无法获取新版：%w", err)
		}
	}

	var message string
	switch kind {
	case imageTool:
		payload := onlinePayload
		if payload == nil {
			payload, err = bundle.imageBytes()
			if err != nil {
				return ToolResult{}, err
			}
		}
		if err := installImagePackage(a.ctx, root, payload, key); err != nil {
			return ToolResult{}, err
		}
		message = "绘图工具 V" + targetVersion + " 已安装。"
	case modelsTool:
		script, err := bundle.modelScript()
		if err != nil {
			return ToolResult{}, err
		}
		if err := installModelPackage(a.ctx, root, script, onlinePayload); err != nil {
			return ToolResult{}, err
		}
		message = "国产模型已导入。"
		if _, err := applyConfig(filepath.Join(root, modelReceipt), func([]byte) ([]byte, error) {
			return []byte(targetVersion + "\n"), nil
		}); err != nil {
			message += "\n本次导入成功，但未能保存版本记录。"
		}
	}
	status, _ = localToolStatus(root, kind, bundle)
	status = withOnlineStatus(kind, status, a.toolUpdates[kind])
	return ToolResult{Status: status, Message: a.finishCodexOperation(message)}, nil
}

func (a *App) verifyImageInstallKey(root, input string, installed bool) (string, error) {
	if strings.TrimSpace(input) == "" && installed {
		input = string(readLocalToolFile(filepath.Join(root, "tools", "myautoapi-imagines-tool", "api-key"), 4<<10))
	}
	key, err := validateKey(input)
	if err != nil {
		return "", errors.New("请填写专门画图分组的完整 API 密钥。")
	}
	if err := verifyGatewayKey(a.ctx, a.gateway.client, a.gateway.origin, key); err != nil {
		return "", err
	}
	if err := a.gateway.requireModels(a.ctx, key, "专门画图分组", defaultImageModel); err != nil {
		return "", err
	}
	return key, nil
}

func stageInstallFiles(files map[string][]byte, run func(string) error) error {
	dir, err := os.MkdirTemp("", "myautoapi-tools-")
	if err != nil {
		return errors.New("无法准备本机临时安装目录")
	}
	defer os.Remove(dir)
	for name, data := range files {
		path := filepath.Join(dir, name)
		defer os.Remove(path)
		if err := os.WriteFile(path, data, 0600); err != nil {
			return errors.New("无法准备本机安装文件")
		}
	}
	return run(dir)
}

func installImagePackage(ctx context.Context, root string, payload []byte, key string) error {
	return stageInstallFiles(map[string][]byte{"image-tool.exe": payload}, func(dir string) error {
		args := []string{"install", "--key-stdin"}
		if err := runToolProcess(ctx, root, dir, filepath.Join(dir, "image-tool.exe"), args, strings.NewReader(key+"\n")); err != nil {
			return errors.New("生图技能未能安装。请检查 Codex 配置目录是否可写、旧工具是否在运行，以及画图密钥是否完整。")
		}
		return nil
	})
}

func installModelPackage(ctx context.Context, root string, script, catalog []byte) error {
	files := map[string][]byte{"codex-models.ps1": script}
	command := `& '.\codex-models.ps1' -CodexHomePath $env:CODEX_HOME -KeepClientsOpen`
	if catalog != nil {
		files["catalog.json"] = catalog
		command += ` -CatalogFile ([IO.Path]::GetFullPath('.\catalog.json'))`
	}
	return stageInstallFiles(files, func(dir string) error {
		output, err := runEmbeddedPowerShell(ctx, 2*time.Minute, root, dir, command)
		if err == nil {
			return nil
		}
		if errors.Is(err, errScriptBlocked) {
			return errors.New("Windows 的 PowerShell 策略阻止了内置模型导入脚本。")
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return errors.New("国产模型导入超时。请关闭正在编辑 Codex 配置的程序后重试。")
		}
		// The repository script's diagnostic output contains local paths. Only
		// surface known step names, never echo installer output into the webview.
		for _, step := range []string{"创建 Codex 配置目录", "读取 Codex 配置", "解析 Codex 配置", "解析原模型目录路径", "读取原模型目录", "读取原输出模型目录", "读取已管理模型记录", "校验并合并模型元数据", "合并模型目录指向和桌面档位", "读取新版目录", "写入并校验导入结果", "写入暂存文件", "校验暂存文件", "备份原文件", "提交导入文件"} {
			if strings.Contains(output, "失败步骤："+step+"\n") {
				return fmt.Errorf("国产模型导入失败。失败步骤为「%s」，请检查 Codex 配置和文件权限。", step)
			}
		}
		return errors.New("国产模型未能导入。请检查 Codex 配置是否有效、目录是否可写，再重试。")
	})
}

var errScriptBlocked = errors.New("script blocked by Windows policy")

func toolCommand(ctx context.Context, root, dir, program string, args []string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(), "CODEX_HOME="+root)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	return cmd
}

func runToolProcess(ctx context.Context, root, dir, program string, args []string, stdin io.Reader) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := toolCommand(ctx, root, dir, program, args)
	cmd.Stdin = stdin
	return cmd.Run()
}

func runEmbeddedPowerShell(ctx context.Context, timeout time.Duration, root, dir, command string) (string, error) {
	system, err := windows.GetSystemDirectory()
	if err != nil {
		return "", errors.New("无法找到 Windows PowerShell")
	}
	shell := filepath.Join(system, "WindowsPowerShell", "v1.0", "powershell.exe")
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	// Only staged scripts from the executable are invoked. A process-local
	// RemoteSigned policy leaves machine and user Group Policy in effect.
	const setup = `[Console]::OutputEncoding = [Text.UTF8Encoding]::new($false); $ErrorActionPreference = 'Stop'; `
	cmd := toolCommand(ctx, root, dir, shell, []string{
		"-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "RemoteSigned", "-Command", setup + command,
	})
	env := cmd.Env[:0]
	for _, entry := range cmd.Env {
		key, _, _ := strings.Cut(entry, "=")
		if !strings.EqualFold(key, "PSModulePath") {
			env = append(env, entry)
		}
	}
	cmd.Env = env
	var output, errOutput cappedOutput
	cmd.Stdout, cmd.Stderr = &output, &errOutput
	err = cmd.Run()
	if ctx.Err() != nil {
		err = ctx.Err()
	} else if err != nil && bytes.Contains(errOutput.data, []byte("PSSecurityException")) {
		err = errScriptBlocked
	}
	return strings.ReplaceAll(string(output.data), "\r\n", "\n"), err
}

type cappedOutput struct{ data []byte }

func (c *cappedOutput) Write(data []byte) (int, error) {
	n := len(data)
	const limit = 4 << 10
	if room := limit - len(c.data); room > 0 {
		c.data = append(c.data, data[:min(room, n)]...)
	}
	return n, nil
}
