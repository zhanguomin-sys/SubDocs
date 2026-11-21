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
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	productName       = "MyAutoAPI 接入助手"
	version           = "1.2.2"
	api               = "https://api.myautoapi.com"
	docs              = "https://docs.myautoapi.com"
	keyVariable       = "MYAUTOAPI_CODEX_KEY"
	provider          = "myautoapi"
	defaultCodexModel = "gpt-6.1-sol"
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

func codexConfig(old []byte) ([]byte, error) {
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
		if config["model_provider"] != provider || config["model"] == nil {
			config["model"] = defaultCodexModel
		}
		config["model_provider"] = provider
		return nil
	})
}

func localeConfig(old []byte) ([]byte, error) {
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

func claudeCodeConfig(old []byte, key string) ([]byte, error) {
	config, err := jsonObject(old)
	if err != nil {
		return nil, err
	}
	env, err := jsonObject(config["env"])
	if err != nil {
		return nil, err
	}
	env["ANTHROPIC_BASE_URL"], _ = json.Marshal(api)
	env["ANTHROPIC_AUTH_TOKEN"], _ = json.Marshal(key)
	// Claude Code accepts a gateway token instead of an Anthropic API key.
	// A saved direct Anthropic key would cause two competing credentials.
	delete(env, "ANTHROPIC_API_KEY")
	config["env"], _ = json.Marshal(env)
	data, err := json.MarshalIndent(config, "", "  ")
	return append(data, '\n'), err
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
