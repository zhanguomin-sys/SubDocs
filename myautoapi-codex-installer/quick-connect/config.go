package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	productName = "MyAutoAPI 中转快速接入助手"
	version     = "1.2.25"
	site        = "https://myautoapi.com"
	api         = "https://api.myautoapi.com"
	docs        = "https://docs.myautoapi.com"
	qqGroup     = "685861462"
	qqInvite    = "https://qm.qq.com/q/C2UlbcUTmK"
	keyVariable = "MYAUTOAPI_CODEX_KEY"
	provider    = "myautoapi"
	gpt62Model  = "gpt-6.2-sol"
	gpt62Effort = "high"
)

var keyPattern = regexp.MustCompile("^sk-[A-Za-z0-9._-]{8,500}$")

func validateKey(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !keyPattern.MatchString(value) {
		return "", errors.New("请粘贴 MyAutoAPI 对应分组的完整 API Key")
	}
	return value, nil
}

func setTOML(old []byte, change func(map[string]any) error) ([]byte, error) {
	config := make(map[string]any)
	if _, err := toml.Decode(strings.TrimPrefix(string(old), "\ufeff"), &config); err != nil {
		return nil, errors.New("Codex 配置不是有效的 TOML；请先在 Codex 中修复配置")
	}
	if err := change(config); err != nil {
		return nil, err
	}
	var b bytes.Buffer
	if err := toml.NewEncoder(&b).Encode(config); err != nil {
		return nil, errors.New("无法重新保存 Codex 配置")
	}
	return b.Bytes(), nil
}

func tomlTable(config map[string]any, name string) (map[string]any, error) {
	if existing, ok := config[name]; ok {
		if table, ok := existing.(map[string]any); ok {
			return table, nil
		}
		return nil, fmt.Errorf("Codex 配置的 %s 不是表格；已保留原文件", name)
	}
	table := make(map[string]any)
	config[name] = table
	return table, nil
}

func codexConfig(old []byte, updateCatalog func(map[string]any) error) ([]byte, error) {
	return setTOML(old, func(config map[string]any) error {
		// An active named profile can override both the provider and the model.
		// Do not silently reset an existing profile.
		if profile, _ := config["profile"].(string); profile != "" {
			return errors.New("当前启用了 Codex 命名配置。请切回默认配置，或按本站手动接入指南配置该 Profile")
		}
		providers, err := tomlTable(config, "model_providers")
		if err != nil {
			return err
		}
		providers[provider] = map[string]any{
			"name":                 "MyAutoAPI",
			"base_url":             api + "/v1",
			"wire_api":             "responses",
			"env_key":              keyVariable,
			"requires_openai_auth": false,
		}
		config["model"] = gpt62Model
		config["model_reasoning_effort"] = gpt62Effort
		config["model_provider"] = provider
		// Older installs may restrict which effort levels the desktop picker shows.
		if desktop, ok := config["desktop"].(map[string]any); ok {
			if efforts, ok := desktop["enabled-reasoning-efforts"].([]any); ok && !slices.Contains(efforts, gpt62Effort) {
				desktop["enabled-reasoning-efforts"] = append(efforts, gpt62Effort)
			}
		}
		if updateCatalog != nil {
			return updateCatalog(config)
		}
		return nil
	})
}

func localeConfig(old []byte) ([]byte, error) {
	// Check the same bytes that will be edited, using the status card's rule.
	configured, err := codexUsesMyAutoAPI(old)
	if err != nil {
		return nil, errors.New("无法识别 Codex 配置，未更改语言设置。")
	}
	if !configured {
		return nil, errors.New("Codex 尚未配置 MyAutoAPI。请先点击「一键连接 Codex」，再设置简体中文。")
	}
	return setTOML(old, func(config map[string]any) error {
		desktop, err := tomlTable(config, "desktop")
		if err != nil {
			return err
		}
		desktop["localeOverride"] = "zh-CN"
		return nil
	})
}

func jsonObject(old []byte) (map[string]json.RawMessage, error) {
	object := make(map[string]json.RawMessage)
	if len(old) != 0 {
		if err := json.Unmarshal(bytes.TrimPrefix(old, []byte{0xef, 0xbb, 0xbf}), &object); err != nil || object == nil {
			return nil, errors.New("现有设置不是有效的 JSON 对象；已保留原文件")
		}
	}
	return object, nil
}

func codexUsesMyAutoAPI(data []byte) (bool, error) {
	var config struct {
		Profile   string `toml:"profile"`
		Provider  string `toml:"model_provider"`
		Providers map[string]struct {
			BaseURL string `toml:"base_url"`
		} `toml:"model_providers"`
	}
	if _, err := toml.Decode(strings.TrimPrefix(string(data), "\ufeff"), &config); err != nil {
		return false, err
	}
	return config.Profile == "" && strings.TrimRight(config.Providers[config.Provider].BaseURL, "/") == api+"/v1", nil
}

func reusableCodexKey(config []byte, stored string) (string, error) {
	configured, err := codexUsesMyAutoAPI(config)
	if err != nil || !configured {
		return "", errors.New("Codex 尚未配置 MyAutoAPI，请输入 GPT 分组 API Key")
	}
	key, err := validateKey(stored)
	if err != nil {
		return "", errors.New("没有找到已保存的 GPT 分组 API Key，请重新输入")
	}
	return key, nil
}

func readConfig(path string) ([]byte, bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if !info.Mode().IsRegular() || info.Size() > 8<<20 {
		return nil, false, errors.New("已有配置不是普通文件，或文件过大；已保留原文件")
	}
	data, err := os.ReadFile(path)
	return data, true, err
}

// saveConfig uses an adjacent temporary file so a reader sees either the
// old or the complete new configuration, and returns the backup name.
func saveConfig(path string, old []byte, exists bool, updated []byte) (string, error) {
	if exists && bytes.Equal(old, updated) {
		return "", nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".myautoapi-write-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	_, err = f.Write(updated)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", err
	}
	current, currentlyExists, err := readConfig(path)
	if err != nil {
		return "", err
	}
	if currentlyExists != exists || !bytes.Equal(current, old) {
		return "", errors.New("另一程序刚更新了配置；请关闭该应用后重试")
	}
	var backup string
	if exists {
		b, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".myautoapi-*.bak")
		if err != nil {
			return "", err
		}
		backup = b.Name()
		_, err = b.Write(old)
		if closeErr := b.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(backup)
			return "", err
		}
	}
	return backup, os.Rename(f.Name(), path)
}

func applyConfig(path string, change func([]byte) ([]byte, error)) (string, error) {
	old, exists, err := readConfig(path)
	if err != nil {
		return "", err
	}
	updated, err := change(old)
	if err != nil {
		return "", err
	}
	return saveConfig(path, old, exists, updated)
}

func report(message, backup string) string {
	if backup != "" {
		return message + "\n原配置备份：" + backup
	}
	return message
}
