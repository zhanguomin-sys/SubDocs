package connect

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// The domestic model import is install/codex-models.ps1 (source in
// codex-domestic-models/). The assistant carries the published script and
// runs it with Windows PowerShell in the background, without a console
// window. These file names mirror what the script writes; the assistant only
// reads them.
const (
	modelsCatalogName = "myautoapi-models.json"
	modelsManagedName = "myautoapi-models.managed.json"
	modelsTimeout     = 3 * time.Minute
)

// Domestic model card states.
const (
	ModelsNone     = "none"     // never imported
	ModelsActive   = "active"   // imported and in use
	ModelsInactive = "inactive" // imported, but config.toml no longer points at the catalog
)

// ModelsStatus is what the domestic model card shows.
type ModelsStatus struct {
	State string `json:"state"`
	Count int    `json:"count,omitempty"`
}

// ReadModelsStatus inspects codexHome the way codex-models.ps1 does.
func ReadModelsStatus(codexHome string) ModelsStatus {
	if codexHome == "" {
		return ModelsStatus{State: ModelsNone}
	}
	data, err := os.ReadFile(filepath.Join(codexHome, modelsManagedName))
	if err != nil {
		return ModelsStatus{State: ModelsNone}
	}
	var ids []any
	_ = json.Unmarshal(bytes.TrimPrefix(data, []byte("\ufeff")), &ids)
	count := 0
	for _, id := range ids {
		if text, ok := id.(string); ok && text != "" && !strings.HasPrefix(text, "gpt-") {
			count++
		}
	}
	if count == 0 {
		return ModelsStatus{State: ModelsNone}
	}
	if catalogInUse(codexHome) {
		return ModelsStatus{State: ModelsActive, Count: count}
	}
	return ModelsStatus{State: ModelsInactive, Count: count}
}

// catalogInUse reports whether config.toml's model_catalog_json resolves to
// the catalog the script writes (Get-ConfiguredCatalogPath).
func catalogInUse(codexHome string) bool {
	data, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil {
		return false
	}
	var config struct {
		Catalog string `toml:"model_catalog_json"`
	}
	if _, err := toml.Decode(strings.TrimPrefix(string(data), "\ufeff"), &config); err != nil || config.Catalog == "" {
		return false
	}
	value := config.Catalog
	if strings.HasPrefix(value, "~/") || strings.HasPrefix(value, `~\`) {
		if home, err := os.UserHomeDir(); err == nil {
			value = filepath.Join(home, value[2:])
		}
	}
	if !filepath.IsAbs(value) {
		value = filepath.Join(codexHome, value)
	}
	want := filepath.Join(codexHome, modelsCatalogName)
	return strings.EqualFold(filepath.Clean(value), filepath.Clean(want))
}

// ModelsImporter runs the domestic model script.
type ModelsImporter interface {
	Available() bool
	// Run imports (uninstall false) or removes the models and returns the
	// lines the script printed.
	Run(ctx context.Context, codexHome string, uninstall bool) ([]string, error)
}

// BundledModelsScript is codex-models.ps1 carried inside the assistant.
type BundledModelsScript struct {
	Script     []byte
	CacheDir   string
	PowerShell string // powershell.exe; tests use a stand-in
}

func (b *BundledModelsScript) Available() bool { return b != nil && len(b.Script) > 0 }

// psWrapper asks Windows PowerShell 5.1 for UTF-8 output and runs the script
// from paths passed in the environment, so a Chinese user name or a quote in
// a path never has to be escaped on the command line.
const psWrapper = `$OutputEncoding = New-Object Text.UTF8Encoding($false)
try { [Console]::OutputEncoding = $OutputEncoding } catch { }
if ($env:MYAUTOAPI_MODELS_UNINSTALL -eq '1') {
  & $env:MYAUTOAPI_MODELS_SCRIPT -CodexHomePath $env:MYAUTOAPI_MODELS_HOME -Uninstall
} else {
  & $env:MYAUTOAPI_MODELS_SCRIPT -CodexHomePath $env:MYAUTOAPI_MODELS_HOME
}`

func (b *BundledModelsScript) unpack() (string, error) {
	path := filepath.Join(b.CacheDir, "codex-models.ps1")
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, b.Script) {
		return path, nil
	}
	if err := os.MkdirAll(b.CacheDir, 0o700); err != nil {
		return "", err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, b.Script, 0o600); err != nil {
		return "", err
	}
	return path, os.Rename(temporary, path)
}

func (b *BundledModelsScript) Run(ctx context.Context, codexHome string, uninstall bool) ([]string, error) {
	if !b.Available() {
		return nil, errors.New("助手没有内置国产模型导入")
	}
	script, err := b.unpack()
	if err != nil {
		return nil, errors.New("无法准备导入脚本：" + reason(err))
	}
	ctx, cancel := context.WithTimeout(ctx, modelsTimeout)
	defer cancel()
	shell := b.PowerShell
	if shell == "" {
		shell = "powershell.exe"
	}
	command := exec.CommandContext(ctx, shell, "-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", psWrapper)
	flag := "0"
	if uninstall {
		flag = "1"
	}
	command.Env = append(os.Environ(),
		"MYAUTOAPI_MODELS_SCRIPT="+script,
		"MYAUTOAPI_MODELS_HOME="+codexHome,
		"MYAUTOAPI_MODELS_UNINSTALL="+flag)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	hideConsole(command)
	runErr := command.Run()
	lines := outputLines(stdout.Bytes())
	if runErr != nil {
		if ctx.Err() != nil {
			return lines, errors.New("导入脚本没有在 3 分钟内完成")
		}
		if len(lines) == 0 {
			lines = outputLines(stderr.Bytes())
		}
		return lines, errors.New("脚本出错：" + reason(runErr))
	}
	return lines, nil
}

// outputLines decodes the script's output: UTF-8 as asked, or the Chinese
// ANSI code page when the console refused to switch.
func outputLines(raw []byte) []string {
	raw = bytes.TrimPrefix(raw, []byte("\ufeff"))
	text := string(raw)
	if !utf8.Valid(raw) {
		if decoded, err := simplifiedchinese.GB18030.NewDecoder().Bytes(raw); err == nil {
			text = string(decoded)
		}
	}
	var lines []string
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// Messages for the domestic model import.
const (
	MessageModelsNotBundled = "这个版本的助手没有内置国产模型导入，请下载新版助手。"
	MessageModelsFailed     = "国产模型导入未完成，Codex 没有关闭。"
	MessageRemoveFailed     = "国产模型移除未完成，Codex 没有关闭。"
)

// ModelsStatus reads the domestic model card.
func (s *Service) ModelsStatus() ModelsStatus { return ReadModelsStatus(s.ModelsHome) }

// ImportModels runs the script; uninstall removes the models it imported.
// The script itself closes ChatGPT and Codex after a change.
func (s *Service) ImportModels(ctx context.Context, uninstall bool) Result {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Models == nil || !s.Models.Available() {
		return s.result(ToneError, MessageModelsNotBundled)
	}
	if s.ModelsHome == "" {
		return s.result(ToneError, "找不到当前用户的文件夹。")
	}
	lines, err := s.Models.Run(ctx, s.ModelsHome, uninstall)
	lines = dropNotices(lines)
	if err != nil {
		failed := MessageModelsFailed
		if uninstall {
			failed = MessageRemoveFailed
		}
		if len(lines) > 0 {
			return s.result(ToneError, strings.Join(lines, "\n")+"\n"+failed)
		}
		return s.result(ToneError, err.Error()+"。"+failed)
	}
	if len(lines) == 0 {
		lines = []string{"已完成。"}
	}
	return s.result(ToneSuccess, strings.Join(lines, "\n"))
}

// dropNotices removes the warning the script prints before it starts; the
// window asked for confirmation already.
func dropNotices(lines []string) []string {
	kept := lines[:0:0]
	for _, line := range lines {
		if strings.Contains(line, "后会自动关闭 ChatGPT/Codex") {
			continue
		}
		kept = append(kept, line)
	}
	return kept
}
