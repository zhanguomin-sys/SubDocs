package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

func claudeCodePaths() (settings, state string, err error) {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "settings.json"), filepath.Join(dir, ".claude.json"), nil
	}
	profile, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	return filepath.Join(profile, ".claude", "settings.json"), filepath.Join(profile, ".claude.json"), nil
}

func claudeCodeConfig(old []byte, key string, models []string) ([]byte, error) {
	config, err := jsonObject(old)
	if err != nil {
		return nil, err
	}
	env, err := jsonObject(config["env"])
	if err != nil {
		return nil, err
	}
	if len(models) == 0 {
		return nil, errors.New("该密钥没有可用的 Claude 模型")
	}

	var model string
	_ = json.Unmarshal(env["ANTHROPIC_MODEL"], &model)
	if !slices.Contains(models, model) {
		model = ""
		_ = json.Unmarshal(config["model"], &model)
	}
	if !slices.Contains(models, model) {
		model = models[0]
		for _, id := range models {
			if strings.HasPrefix(id, "claude-sonnet-") {
				model = id
				break
			}
		}
	}

	env["ANTHROPIC_BASE_URL"], _ = json.Marshal(api)
	env["ANTHROPIC_AUTH_TOKEN"], _ = json.Marshal(key)
	// Empty values also mask credentials and providers inherited from the shell.
	// Leave account credentials and other settings untouched.
	for _, name := range []string{
		"ANTHROPIC_API_KEY", "ANTHROPIC_MODEL", "CLAUDE_CODE_USE_BEDROCK",
		"CLAUDE_CODE_USE_VERTEX", "CLAUDE_CODE_USE_FOUNDRY",
	} {
		env[name] = json.RawMessage(`""`)
	}
	for _, name := range []string{"ANTHROPIC_DEFAULT_OPUS_MODEL", "ANTHROPIC_DEFAULT_SONNET_MODEL", "ANTHROPIC_DEFAULT_HAIKU_MODEL"} {
		var existing string
		if raw, ok := env[name]; ok && (json.Unmarshal(raw, &existing) != nil || !slices.Contains(models, existing)) {
			env[name] = json.RawMessage(`""`)
		}
	}
	config["env"], _ = json.Marshal(env)
	config["model"], _ = json.Marshal(model)

	data, err := json.MarshalIndent(config, "", "  ")
	return append(data, '\n'), err
}

func claudeCodeUsesMyAutoAPI(data []byte) (bool, error) {
	var config struct {
		Env struct {
			BaseURL string `json:"ANTHROPIC_BASE_URL"`
			Token   string `json:"ANTHROPIC_AUTH_TOKEN"`
		} `json:"env"`
	}
	if err := json.Unmarshal(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf}), &config); err != nil {
		return false, err
	}
	_, err := validateKey(config.Env.Token)
	return strings.TrimRight(config.Env.BaseURL, "/") == api && err == nil, nil
}

func claudeCodeOnboarding(old []byte) ([]byte, error) {
	state, err := jsonObject(old)
	if err != nil {
		return nil, err
	}
	var completed bool
	if json.Unmarshal(state["hasCompletedOnboarding"], &completed) == nil && completed {
		return old, nil
	}
	// Complete initial setup for a verified gateway key. Leave project trust,
	// tool permissions and stored Anthropic credentials alone.
	state["hasCompletedOnboarding"] = json.RawMessage("true")
	data, err := json.MarshalIndent(state, "", "  ")
	return append(data, '\n'), err
}

var claudeModelID = regexp.MustCompile(`^claude-[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

func connectClaudeCode(ctx context.Context, client *http.Client, origin, input string) (string, error) {
	return connectClaudeModels(ctx, client, origin, input, configureClaudeCode)
}

func connectClaudeModels(ctx context.Context, client *http.Client, origin, input string, configure func(string, []string) (string, error)) (string, error) {
	return configureWithKey(ctx, client, origin, input, func(key string) (string, error) {
		models, err := listGatewayModels(ctx, client, origin, key)
		if err != nil {
			return "", fmt.Errorf("%w\nClaude 配置没有改动。", err)
		}
		models = slices.DeleteFunc(models, func(id string) bool { return !claudeModelID.MatchString(id) })
		if len(models) == 0 {
			return "", errors.New("这个密钥所在的分组没有可用的 Claude 模型。请复制 Claude 分组的 API Key 后重试。\nClaude 配置没有改动。")
		}
		return configure(key, models)
	})
}

func configureClaudeCode(key string, models []string) (string, error) {
	settings, state, err := claudeCodePaths()
	if err != nil {
		return "", fmt.Errorf("无法找到 Claude Code 配置目录：%w", err)
	}
	oldSettings, settingsExist, err := readConfig(settings)
	if err != nil {
		return "", fmt.Errorf("无法读取 Claude Code 用户设置：%w", err)
	}
	nextSettings, err := claudeCodeConfig(oldSettings, key, models)
	if err != nil {
		return "", fmt.Errorf("无法更新 Claude Code 用户设置：%w", err)
	}
	oldState, stateExists, err := readConfig(state)
	if err != nil {
		return "", fmt.Errorf("无法读取 Claude Code 首次启动状态：%w", err)
	}
	nextState, err := claudeCodeOnboarding(oldState)
	if err != nil {
		return "", fmt.Errorf("无法更新 Claude Code 首次启动状态：%w", err)
	}

	settingsBackup, err := saveConfig(settings, oldSettings, settingsExist, nextSettings)
	if err != nil {
		return "", fmt.Errorf("无法保存 Claude Code 用户设置：%w", err)
	}
	stateBackup, err := saveConfig(state, oldState, stateExists, nextState)
	if err != nil {
		message := fmt.Errorf("保存 Claude Code 首次启动状态失败：%w", err)
		if bytes.Equal(oldSettings, nextSettings) {
			return "", message
		}
		current, exists, readErr := readConfig(settings)
		if readErr != nil || !exists || !bytes.Equal(current, nextSettings) {
			return "", errors.New(report(message.Error()+"\n用户设置已被其他程序更新，请关闭 Claude Code 后重试。", settingsBackup))
		}
		// Only undo our write if another process hasn't updated the file.
		var undoErr error
		if settingsExist {
			undoErr = os.Rename(settingsBackup, settings)
		} else {
			undoErr = os.Remove(settings)
		}
		if undoErr != nil {
			return "", errors.New(report(message.Error()+"\n无法自动恢复用户设置。", settingsBackup))
		}
		return "", errors.New(message.Error() + "\n已恢复原用户设置，请关闭 Claude Code 后重试。")
	}

	message := report("已配置 Claude Code 接入 MyAutoAPI。请重新打开 Claude Code 即可使用。", settingsBackup)
	if stateBackup != "" {
		message += "\n原启动状态备份：" + stateBackup
	}
	return message, nil
}
