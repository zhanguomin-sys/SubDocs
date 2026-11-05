package connect

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Result tones: green, amber and red in the window.
const (
	ToneSuccess = "success"
	ToneWarning = "warning"
	ToneError   = "error"
)

// Messages shown in the result bar.
const (
	MessageEmptyKey          = "还没有输入密钥。到 MyAutoAPI 网站的「API 密钥」页复制密钥，粘贴到上面，再点「一键连接 Codex」。"
	MessageEmptyKeyConnected = "当前已连接。要换密钥：到 MyAutoAPI 网站的「API 密钥」页复制密钥，粘贴到上面，再点「一键连接 Codex」。"
	MessageBadFormat         = "这不是 MyAutoAPI 的密钥：应以 sk- 开头，只含字母、数字、- 和 _。请回网站重新复制完整的密钥。"
	MessageRejected          = "密钥无效，请回网站重新复制完整的密钥。Codex 配置没有改动。"
	MessageDisabled          = "这个密钥已停用、过期或额度用完，请在网站换一个可用的密钥。Codex 配置没有改动。"
	MessageTimeout           = "网络超时：MyAutoAPI 服务器 10 秒内没有回应。Codex 配置没有改动。网络慢或开了代理时，稍后再点一次。"
	MessageOffline           = "连不上 MyAutoAPI 服务器：网络不通，或被代理、防火墙拦截。Codex 配置没有改动。检查网络后再点一次。"
	MessageServerError       = "MyAutoAPI 服务器暂时出错（%s），无法确认密钥。Codex 配置没有改动。稍后再点一次，仍不行请进 QQ 群联系客服。"
	MessageLinkIncomplete    = "导入链接不完整，请回网站重新点「一键导入」。"
	MessageLinkUnsupported   = "当前助手不支持这个导入，请下载新版助手。"
	MessageNoBalance         = "已连接，但余额为 $0。充值后即可使用；新人进群领 2 元。"
	MessageCloseFailed       = "配置已写好，但没能自动关闭 Codex，请手动退出后重新打开。"
	MessageFullAccessManual  = "\n没能把 Codex 的权限自动改成「完全访问」：打开 Codex 后，在输入框下方的权限按钮手动选一次「完全访问」。"
	MessageNotClosed         = "没能自动关闭 Codex，请手动退出后重新打开。"
	MessageClosed            = "Codex 已关闭，重新打开即可。"
	MessageRestored          = "已恢复官方登录，重新打开 Codex 后用官方账号登录。"
	MessageRestoredOpen      = "已恢复官方登录，但没能自动关闭 Codex，请手动退出后重新打开，用官方账号登录。"
)

// ActionOpenSite asks the window to offer an "打开网站" button with the result.
const ActionOpenSite = "open-site"

// Result is what one action shows: the bar and the refreshed Codex card.
type Result struct {
	Tone    string       `json:"tone"`
	Message string       `json:"message"`
	Status  Status       `json:"status"`
	Image   ImageStatus  `json:"image"`
	Models  ModelsStatus `json:"models"`
	Claude  ClaudeStatus `json:"claude"`
	Action  string       `json:"action,omitempty"`
	// OfferCLI asks the window whether to connect the Claude Code CLI too.
	OfferCLI bool `json:"offerCli,omitempty"`
}

// Service runs one action at a time; a link that arrives during a
// connection waits for it.
type Service struct {
	CodexHome   string
	BaseURL     string
	Client      *http.Client
	Now         func() time.Time
	StopClients func() StopResult
	// NamesFile remembers the names of imported keys (see names.go). Empty
	// means names are not kept.
	NamesFile string
	// ImageHome is where the image tool lives (see ImageCodexHome); Image
	// installs it. A nil Image means the assistant carries no image tool.
	ImageHome string
	Image     ImageInstaller
	// ModelsHome is where the domestic model script works (CODEX_HOME or
	// ~/.codex, as the script decides); Models runs it.
	ModelsHome string
	Models     ModelsImporter
	// Claude is Claude Desktop, ClaudeCLI the Claude Code CLI; StopClaude
	// closes Claude Desktop (nil: never closed).
	Claude     ClaudeDesktop
	ClaudeCLI  ClaudeCode
	StopClaude func() ClaudeStopResult

	mu sync.Mutex
}

// Status reads the current Codex configuration.
func (s *Service) Status() Status {
	status, key := readStatus(s.CodexHome)
	if key != "" {
		status.Name = lookupName(s.NamesFile, key)
	}
	status.AppInstalled = CodexAppDetected(s.Claude.LocalAppData, s.CodexHome)
	return status
}

// Connect checks the key, closes Codex and writes the Codex files. Nothing is
// written unless the server confirms the key. name comes from an import link
// and is remembered for the Codex card; a pasted key passes "".
func (s *Service) Connect(ctx context.Context, raw, name string) Result {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, ok := NormalizeKey(raw)
	if !ok {
		if key == "" {
			result := s.result(ToneError, MessageEmptyKey)
			if result.Status.State == StateMyAutoAPI {
				result.Message = MessageEmptyKeyConnected
			}
			result.Action = ActionOpenSite
			return result
		}
		return s.result(ToneError, MessageBadFormat)
	}
	check := CheckKey(ctx, s.Client, s.BaseURL, key)
	switch check.State {
	case KeyRejected:
		return s.result(ToneError, MessageRejected)
	case KeyDisabled:
		return s.result(ToneError, MessageDisabled)
	case KeyTimeout:
		return s.result(ToneError, MessageTimeout)
	case KeyOffline:
		return s.result(ToneError, MessageOffline)
	case KeyServerError:
		return s.result(ToneError, fmt.Sprintf(MessageServerError, serverProblem(check.StatusCode)))
	}
	if s.CodexHome == "" {
		return s.result(ToneError, writeFailed(errors.New("找不到当前用户的文件夹")))
	}
	// Codex is closed first: the App writes its permission file back when it
	// exits, which would undo the Full Access set below.
	stopped := s.StopClients()
	if err := WriteCodexFiles(s.CodexHome, key, s.Now()); err != nil {
		return s.result(ToneError, writeFailed(err))
	}
	// config.toml already allows full access, so a file the App cannot be
	// switched through does not fail the import; the message says so.
	fullAccessNote := ""
	if err := GrantAppFullAccess(s.CodexHome, s.Now()); err != nil {
		fullAccessNote = MessageFullAccessManual
	}
	if name != "" {
		// Only shown on the card, so a failure here does not fail the import.
		_ = rememberName(s.NamesFile, key, name)
	}

	if check.State == KeyNoBalance {
		message := MessageNoBalance
		if stopped.Left {
			message += MessageNotClosed
		} else if stopped.Closed() {
			message += MessageClosed
		}
		return s.result(ToneWarning, message+fullAccessNote)
	}
	if stopped.Left {
		return s.result(ToneWarning, MessageCloseFailed+fullAccessNote)
	}
	message := "已连接 MyAutoAPI ✓"
	if check.HasRemaining {
		message += " 余额 " + FormatUSD(check.Remaining)
	}
	if stopped.Closed() {
		message += "。Codex 已关闭，重新打开即可，新建任务，发送“测试”。"
	} else {
		message += "。打开 Codex，新建任务，发送“测试”。"
	}
	return s.result(ToneSuccess, message+fullAccessNote)
}

// LinkResult is a parsed link, or the message to show when it is rejected.
type LinkResult struct {
	OK     bool   `json:"ok"`
	App    string `json:"app"`
	Key    string `json:"key"`
	Name   string `json:"name"`
	Result Result `json:"result"`
}

// ParseLink checks an import link before the window starts the connection.
func (s *Service) ParseLink(raw string) LinkResult {
	link, problem := ParseImportLink(raw)
	switch problem {
	case LinkOK:
		return LinkResult{OK: true, App: link.App, Key: link.Key, Name: link.Name}
	case LinkUnsupported:
		return LinkResult{Result: s.result(ToneError, MessageLinkUnsupported)}
	default:
		return LinkResult{Result: s.result(ToneError, MessageLinkIncomplete)}
	}
}

// Restore switches Codex back to the official login and closes it.
func (s *Service) Restore() Result {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.CodexHome != "" {
		if err := RestoreOfficial(s.CodexHome, s.Now()); err != nil {
			return s.result(ToneError, "恢复官方登录失败："+reason(err)+"。原配置没有改动，可以重试。")
		}
	}
	if s.StopClients().Left {
		return s.result(ToneWarning, MessageRestoredOpen)
	}
	return s.result(ToneSuccess, MessageRestored)
}

func (s *Service) result(tone, message string) Result {
	return Result{Tone: tone, Message: message, Status: s.Status(), Image: s.ImageStatus(), Models: s.ModelsStatus(), Claude: s.ClaudeStatus()}
}

// serverProblem names what went wrong for MessageServerError.
func serverProblem(code int) string {
	if code == 0 || code == http.StatusOK {
		return "返回内容异常"
	}
	return fmt.Sprintf("HTTP %d", code)
}

// FormatUSD shows dollars with two decimals, as the website does.
func FormatUSD(amount float64) string {
	return fmt.Sprintf("$%.2f", amount)
}

func writeFailed(err error) string {
	return "写入 Codex 配置失败：" + reason(err) + "。原配置没有改动，可以重试。"
}

// reason keeps the system's own wording (localised on Windows) without the
// Go operation prefix or the full path.
func reason(err error) string {
	var pathError *fs.PathError
	var linkError *os.LinkError
	if errors.As(err, &pathError) {
		err = pathError.Err
	} else if errors.As(err, &linkError) {
		err = linkError.Err
	}
	return strings.TrimRight(strings.TrimSpace(err.Error()), "。.")
}
