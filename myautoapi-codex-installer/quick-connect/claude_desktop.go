package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const claudeDesktopMyAutoAPIID = "9a866cf2-32dd-4d0a-aee8-7a5fcb730f2f"

type claudeDesktopFiles struct {
	mode, library, profile string
}

func claudeDesktopPaths(dir string) claudeDesktopFiles {
	library := filepath.Join(dir, "configLibrary")
	return claudeDesktopFiles{
		mode:    filepath.Join(dir, "claude_desktop_config.json"),
		library: filepath.Join(library, "_meta.json"),
		profile: filepath.Join(library, claudeDesktopMyAutoAPIID+".json"),
	}
}

func claudeDesktopJSON(v any) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	return append(data, '\n'), err
}

func claudeDesktopMode(old []byte) ([]byte, error) {
	cfg, err := jsonObject(old)
	if err != nil {
		return nil, err
	}
	cfg["deploymentMode"] = json.RawMessage("\"3p\"")
	return claudeDesktopJSON(cfg)
}

func claudeDesktopLibrary(old []byte) ([]byte, error) {
	cfg, err := jsonObject(old)
	if err != nil {
		return nil, err
	}
	if pointer := cfg["hybridPointer"]; len(pointer) > 0 && !bytes.Equal(bytes.TrimSpace(pointer), []byte("null")) {
		return nil, errors.New("Claude 桌面版已由配置服务器管理，请在 Claude 中切换到本地配置后重试")
	}
	var entries []map[string]json.RawMessage
	if raw, ok := cfg["entries"]; ok {
		if err := json.Unmarshal(raw, &entries); err != nil {
			return nil, errors.New("Claude 桌面版配置列表无效，已保留原文件")
		}
	}
	id, _ := json.Marshal(claudeDesktopMyAutoAPIID)
	found := false
	for _, entry := range entries {
		var savedID string
		if json.Unmarshal(entry["id"], &savedID) == nil && savedID == claudeDesktopMyAutoAPIID {
			found = true
			break
		}
	}
	if !found {
		entries = append(entries, map[string]json.RawMessage{
			"id": id, "name": json.RawMessage("\"MyAutoAPI\""),
		})
	}
	cfg["entries"], _ = json.Marshal(entries)
	cfg["appliedId"] = id
	return claudeDesktopJSON(cfg)
}

type claudeDesktopModel struct {
	Name string `json:"name"`
}

func claudeDesktopProfile(origin, key string, models []string) ([]byte, error) {
	if len(models) == 0 {
		return nil, errors.New("该密钥没有可用的 Claude 模型")
	}
	allowed := make([]claudeDesktopModel, 0, len(models))
	for _, model := range models {
		allowed = append(allowed, claudeDesktopModel{Name: model})
	}
	return claudeDesktopJSON(map[string]any{
		"inferenceProvider":          "gateway",
		"inferenceCredentialKind":    "static",
		"inferenceGatewayBaseUrl":    origin,
		"inferenceGatewayAuthScheme": "bearer",
		"inferenceGatewayApiKey":     key,
		"inferenceModels":            allowed,
	})
}

func claudeDesktopConfigured(files claudeDesktopFiles) (bool, error) {
	check := func(path, name, expected string) (bool, error) {
		data, exists, err := readConfig(path)
		if err != nil || !exists {
			return false, err
		}
		cfg, err := jsonObject(data)
		if err != nil {
			return false, err
		}
		var value string
		return json.Unmarshal(cfg[name], &value) == nil && strings.TrimRight(value, "/") == expected, nil
	}
	if ok, err := check(files.mode, "deploymentMode", "3p"); !ok || err != nil {
		return false, err
	}
	if ok, err := check(files.library, "appliedId", claudeDesktopMyAutoAPIID); !ok || err != nil {
		return false, err
	}
	if ok, err := check(files.profile, "inferenceGatewayBaseUrl", api); !ok || err != nil {
		return false, err
	}
	if ok, err := check(files.profile, "inferenceProvider", "gateway"); !ok || err != nil {
		return false, err
	}
	data, exists, err := readConfig(files.profile)
	if err != nil || !exists {
		return false, err
	}
	var profile struct {
		Key string `json:"inferenceGatewayApiKey"`
	}
	if err := json.Unmarshal(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf}), &profile); err != nil {
		return false, err
	}
	// Claude may replace the plaintext key with a sealed string after its first launch.
	return strings.TrimSpace(profile.Key) != "", nil
}

type claudeDesktopChange struct {
	path, backup  string
	before, after []byte
	existed       bool
}

func writeClaudeDesktop(files claudeDesktopFiles, origin, key string, models []string) (string, error) {
	profile, err := claudeDesktopProfile(origin, key, models)
	if err != nil {
		return "", err
	}
	// Prepare every change before touching a live configuration. Set startup mode last.
	updates := []struct {
		path    string
		prepare func([]byte) ([]byte, error)
	}{
		{files.profile, func([]byte) ([]byte, error) { return profile, nil }},
		{files.library, claudeDesktopLibrary},
		{files.mode, claudeDesktopMode},
	}
	var prepared []claudeDesktopChange
	for _, update := range updates {
		before, existed, err := readConfig(update.path)
		if err != nil {
			return "", fmt.Errorf("无法读取 Claude 桌面版配置：%w", err)
		}
		after, err := update.prepare(before)
		if err != nil {
			return "", fmt.Errorf("无法更新 Claude 桌面版配置：%w", err)
		}
		if !existed || !bytes.Equal(before, after) {
			prepared = append(prepared, claudeDesktopChange{path: update.path, before: before, after: after, existed: existed})
		}
	}

	message := "已配置 Claude 桌面版接入 MyAutoAPI。请从托盘完全退出 Claude，再重新打开。"
	for i := range prepared {
		change := &prepared[i]
		backup, err := saveConfig(change.path, change.before, change.existed, change.after)
		if err != nil {
			err = fmt.Errorf("无法保存 Claude 桌面版配置：%w", err)
			for j := i - 1; j >= 0; j-- {
				prior := prepared[j]
				current, exists, readErr := readConfig(prior.path)
				if readErr != nil || !exists || !bytes.Equal(current, prior.after) {
					err = errors.Join(err, errors.New(report("配置已被其他进程修改，无法自动恢复。", prior.backup)))
					continue
				}
				var undoErr error
				if prior.existed {
					undoErr = os.Rename(prior.backup, prior.path)
				} else {
					undoErr = os.Remove(prior.path)
				}
				if undoErr != nil {
					err = errors.Join(err, errors.New(report("无法自动恢复原配置。", prior.backup)))
				}
			}
			return "", err
		}
		change.backup = backup
		message = report(message, backup)
	}
	return message, nil
}
