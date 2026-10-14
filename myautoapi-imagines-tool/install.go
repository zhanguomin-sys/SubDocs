package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const (
	managedBy          = "myautoapi-imagines-tool"
	toolDirectoryName  = "myautoapi-imagines-tool"
	skillName          = "myautoapi-image-generation"
	stateFileName      = "install-manifest.json"
	keyFileName        = "api-key"
	keySourceFileName  = "key-source"
	keySourceDedicated = "dedicated-image-key"
	keySourceAuthJSON  = "auth-json"
	legacyKeyFileName  = keyFileName
	legacyPythonName   = "生成图片.py"
	ruleStart          = "<!-- myautoapi-image-generation:start -->"
	ruleEnd            = "<!-- myautoapi-image-generation:end -->"
)

type installState struct {
	SchemaVersion int               `json:"schema_version"`
	ManagedBy     string            `json:"managed_by"`
	Version       string            `json:"version"`
	Release       string            `json:"release,omitempty"`
	Platform      string            `json:"platform,omitempty"`
	Files         map[string]string `json:"files"`
}

type installLayout struct {
	codexHome   string
	toolDir     string
	skillDir    string
	agentsPath  string
	override    string
	statePath   string
	binaryPath  string
	versionPath string
	releasePath string
	skillPath   string
	agentPath   string
}

type desiredFile struct {
	path    string
	content []byte
	mode    os.FileMode
}

type backupEntry struct {
	path    string
	existed bool
	content []byte
	mode    os.FileMode
}

type fileTransaction struct {
	journal []backupEntry
}

type installOptions struct {
	integrated            bool
	compareProgrammingKey bool
	failureHook           func(string) error
}

func installedBinaryName() string {
	if runtime.GOOS == "windows" {
		return "myautoapi-imagines-tool.exe"
	}
	return "myautoapi-imagines-tool"
}

func buildInstallLayout(root string) installLayout {
	toolDir := filepath.Join(root, "tools", toolDirectoryName)
	skillDir := filepath.Join(root, "skills", skillName)
	return installLayout{
		codexHome:   root,
		toolDir:     toolDir,
		skillDir:    skillDir,
		agentsPath:  filepath.Join(root, "AGENTS.md"),
		override:    filepath.Join(root, "AGENTS.override.md"),
		statePath:   filepath.Join(toolDir, stateFileName),
		binaryPath:  filepath.Join(toolDir, installedBinaryName()),
		versionPath: filepath.Join(toolDir, "VERSION"),
		releasePath: filepath.Join(toolDir, "RELEASE"),
		skillPath:   filepath.Join(skillDir, "SKILL.md"),
		agentPath:   filepath.Join(skillDir, "agents", "openai.yaml"),
	}
}

func install(stdout io.Writer) error {
	return installWithOptions(stdout, os.Stdin, installOptions{})
}

func installWithInput(stdout io.Writer, input io.Reader) error {
	return installWithOptions(stdout, input, installOptions{})
}

func installWithOptions(stdout io.Writer, input io.Reader, options installOptions) error {
	fmt.Fprintln(stdout, "将安装图片工具、‘中转站生图’ Skill 和全局默认生图规则。")
	fmt.Fprintln(stdout, "本次需要填写‘专门画图分组’的 Key；工具运行时只读取工具目录中的 Key。")
	showInstallStep(stdout, "正在检查安装位置……")
	root, err := codexHome()
	if err != nil {
		return err
	}
	layout := buildInstallLayout(root)
	if err := checkManagedPaths(layout); err != nil {
		return err
	}
	previousState, err := readInstallState(layout)
	if err != nil {
		return err
	}
	agentsContent, err := readAgents(layout, true)
	if err != nil {
		return err
	}
	desiredAgents, err := installRule(agentsContent)
	if err != nil {
		return err
	}
	desired, state, err := desiredInstallFiles(layout)
	if err != nil {
		return err
	}
	legacyManaged := previousState != nil && previousState.Files["tool/"+legacyPythonName] != ""
	legacyPath := filepath.Join(layout.toolDir, legacyPythonName)
	keyPath := filepath.Join(layout.toolDir, keyFileName)

	changed := make([]desiredFile, 0, len(desired))
	for _, file := range desired {
		same, err := sameDesiredFile(file)
		if err != nil {
			return err
		}
		if !same {
			changed = append(changed, file)
		}
	}
	agentsChanged := desiredAgents != agentsContent
	legacyExists := legacyManaged && regularFileExists(legacyPath)

	reader := bufio.NewReader(input)
	if options.integrated {
		confirmed, err := confirmDefaultYes(stdout, reader, "是否同时安装 MyAutoAPI 生图工具？\n直接回车安装；输入 N 跳过并退出生图安装。\n[Y/n]")
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(stdout, "已跳过生图工具安装。")
			fmt.Fprintln(stdout, "当前 Codex 基础功能可以正常使用；以后需要生图时，再运行生图工具安装程序即可。")
			return nil
		}
	}

	pendingKey, err := promptDedicatedKey(stdout, reader)
	if err != nil {
		return err
	}
	if options.compareProgrammingKey {
		programmingKey, err := loadAuthJSONKey(root)
		if err != nil {
			return fmt.Errorf("无法检查前面填写的编程分组 Key：%w", err)
		}
		if pendingKey == programmingKey {
			confirmed, err := confirmDefaultYes(stdout, reader, "检测到‘专门画图分组 Key’和前面填写的‘编程分组 Key’完全相同。\n这可能是重复复制，请检查两个 Key 是否复制反了。\n安装器无法判断 Key 是否正确，必须由你确认。\n\n直接回车继续使用当前 Key；输入 N 退出生图安装。\n[Y/n]")
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Fprintln(stdout, "已退出生图工具安装。")
				fmt.Fprintln(stdout, "当前 Codex 基础功能可以正常使用；以后需要生图时，再重新运行生图工具安装程序即可。")
				return nil
			}
		}
	}

	transaction := &fileTransaction{}
	rollback := func(cause error) error {
		if rollbackError := transaction.rollback(); rollbackError != nil {
			return fmt.Errorf("%v；回滚不完整：%w", cause, rollbackError)
		}
		cleanupEmptyDirectories(layout)
		return cause
	}
	for _, file := range changed {
		if file.path == filepath.Join(layout.toolDir, keySourceFileName) {
			continue
		}
		if file.path == layout.agentsPath {
			continue
		}
		showInstallStep(stdout, "正在安装图片工具和 Skill……")
		break
	}
	for _, file := range changed {
		if err := transaction.write(file.path, file.content, file.mode); err != nil {
			return rollback(fmt.Errorf("本地文件操作失败：%w", err))
		}
	}
	if legacyExists {
		if err := transaction.remove(legacyPath); err != nil {
			return rollback(fmt.Errorf("无法清理旧版 Python 工具：%w", err))
		}
	}
	if agentsChanged {
		showInstallStep(stdout, "正在更新 Codex 生图规则……")
		if regularFileExists(layout.agentsPath) {
			agentsMode, err := regularFileMode(layout.agentsPath)
			if err != nil {
				return rollback(fmt.Errorf("无法读取 AGENTS.md 权限：%w", err))
			}
			if err := transaction.write(agentsBackupPath(layout.agentsPath), []byte(agentsContent), agentsMode); err != nil {
				return rollback(fmt.Errorf("无法备份 AGENTS.md：%w", err))
			}
		}
		if err := transaction.writePreservingMode(layout.agentsPath, []byte(desiredAgents), 0o666); err != nil {
			return rollback(fmt.Errorf("无法写入 AGENTS.md：%w", err))
		}
	}
	showInstallStep(stdout, "正在保存专门画图分组 Key……")
	if err := transaction.writePreservingMode(keyPath, []byte(pendingKey), 0o666); err != nil {
		return rollback(fmt.Errorf("无法保存专门画图分组 Key：%w", err))
	}
	if options.failureHook != nil {
		if err := options.failureHook("key-written"); err != nil {
			return rollback(fmt.Errorf("本地文件操作失败：%w", err))
		}
	}
	showInstallStep(stdout, "正在确认安装结果……")
	for _, file := range desired {
		same, err := sameDesiredFile(file)
		if err != nil || !same {
			if err == nil {
				err = errors.New("内容不一致")
			}
			return rollback(fmt.Errorf("安装后校验失败：%s：%w", file.path, err))
		}
	}
	installedAgents, err := os.ReadFile(layout.agentsPath)
	if err != nil || string(installedAgents) != desiredAgents {
		if err == nil {
			err = errors.New("内容不一致")
		}
		return rollback(fmt.Errorf("全局规则写入后校验失败：%w", err))
	}

	status := "安装"
	if previousState != nil {
		status = "修复"
		if previousState.Version != version() {
			status = "升级"
		} else if previousState.Release != releaseRevision {
			status = "更新"
		} else if len(changed) == 0 && !agentsChanged && !legacyExists {
			status = "配置"
		}
	}
	fmt.Fprintln(stdout, "专门画图分组 Key 已保存。")
	if status == "更新" {
		fmt.Fprintf(stdout, "%s完成：%s\n", status, releaseRevision)
	} else {
		fmt.Fprintf(stdout, "%s完成：V%s\n", status, state.Version)
	}
	fmt.Fprintf(stdout, "工具：%s\n", layout.toolDir)
	fmt.Fprintf(stdout, "Skill：%s\n", layout.skillDir)
	fmt.Fprintf(stdout, "全局规则：%s\n", layout.agentsPath)
	fmt.Fprintln(stdout, "请重新打开 Codex 或开始一个新任务，然后直接说：画一张白底红苹果。")
	return nil
}

func confirmDefaultYes(stdout io.Writer, reader *bufio.Reader, prompt string) (bool, error) {
	for {
		fmt.Fprintln(stdout, prompt)
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return false, fmt.Errorf("无法读取确认选项：%w", err)
		}
		choice := strings.ToLower(strings.TrimSpace(line))
		if choice == "" || choice == "y" {
			return true, nil
		}
		if choice == "n" {
			return false, nil
		}
		if err == io.EOF {
			return false, errors.New("未收到有效确认选项，请直接回车同意或输入 N 退出。")
		}
		fmt.Fprintln(stdout, "请输入 Y 或 N，直接回车表示同意。")
	}
}

func promptDedicatedKey(stdout io.Writer, reader *bufio.Reader) (string, error) {
	fmt.Fprintln(stdout, "现在需要填写‘专门画图分组’的 Key。")
	fmt.Fprintln(stdout, "【请仔细核对】")
	fmt.Fprintln(stdout, "请粘贴‘专门画图分组’的 Key。")
	fmt.Fprintln(stdout, "不要把‘编程分组 Key’和‘专门画图分组 Key’复制反。")
	fmt.Fprintln(stdout, "安装器无法分辨 Key 是否复制错，必须确保自己复制正确。")
	for {
		fmt.Fprint(stdout, "请输入专门画图分组 Key：")
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("无法读取专门画图分组 Key：%w", err)
		}
		key := strings.TrimSpace(line)
		if key != "" {
			return key, nil
		}
		if err == io.EOF {
			return "", errors.New("专门画图分组 Key 不能为空。")
		}
		fmt.Fprintln(stdout, "Key 不能为空，请重新输入。")
	}
}

func uninstall(stdout io.Writer) error {
	fmt.Fprintln(stdout, "将卸载图片工具、‘中转站生图’ Skill 和全局默认生图规则。")
	fmt.Fprintln(stdout, "卸载不会读取、修改或删除 Codex auth.json。")
	showInstallStep(stdout, "正在检查安装状态……")
	root, err := codexHome()
	if err != nil {
		return err
	}
	layout := buildInstallLayout(root)
	if err := checkManagedPaths(layout); err != nil {
		return err
	}
	state, err := readInstallState(layout)
	if err != nil {
		return err
	}
	if state == nil {
		content := ""
		if regularFileExists(layout.agentsPath) {
			content, err = readAgents(layout, false)
			if err != nil {
				return err
			}
		}
		span, err := markerSpan(content)
		if err != nil {
			return err
		}
		if span != nil {
			return errors.New("发现规则块但缺少安装状态，无法确认卸载范围。")
		}
		fmt.Fprintln(stdout, "未发现可确认的图片工具安装，无需卸载。")
		return nil
	}
	managedPaths, err := assertKnownUninstallFiles(layout, state)
	if err != nil {
		return err
	}
	agentsContent, err := readAgents(layout, false)
	if err != nil {
		return err
	}
	desiredAgents, err := removeRule(agentsContent)
	if err != nil {
		return err
	}
	transaction := &fileTransaction{}
	rollback := func(cause error) error {
		if rollbackError := transaction.rollback(); rollbackError != nil {
			return fmt.Errorf("%v；回滚不完整：%w", cause, rollbackError)
		}
		return cause
	}
	if regularFileExists(layout.agentsPath) {
		showInstallStep(stdout, "正在移除 Codex 生图规则……")
		agentsMode, err := regularFileMode(layout.agentsPath)
		if err != nil {
			return rollback(fmt.Errorf("无法读取 AGENTS.md 权限：%w", err))
		}
		if err := transaction.write(agentsBackupPath(layout.agentsPath), []byte(agentsContent), agentsMode); err != nil {
			return rollback(fmt.Errorf("无法备份 AGENTS.md：%w", err))
		}
		if err := transaction.writePreservingMode(layout.agentsPath, []byte(desiredAgents), 0o666); err != nil {
			return rollback(fmt.Errorf("无法移除全局规则：%w", err))
		}
	}
	if len(managedPaths) != 0 {
		showInstallStep(stdout, "正在删除图片工具和 Skill 文件……")
	}
	for _, path := range managedPaths {
		if err := transaction.remove(path); err != nil {
			return rollback(fmt.Errorf("无法删除托管文件 %s：%w", path, err))
		}
	}
	cleanupEmptyDirectories(layout)
	fmt.Fprintln(stdout, "卸载完成。")
	return nil
}

func showInstallStep(stdout io.Writer, message string) {
	fmt.Fprintln(stdout, message)
	time.Sleep(100 * time.Millisecond)
}

func desiredInstallFiles(layout installLayout) ([]desiredFile, *installState, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, nil, fmt.Errorf("无法定位安装程序：%w", err)
	}
	binary, err := os.ReadFile(executable)
	if err != nil {
		return nil, nil, fmt.Errorf("无法读取安装程序：%w", err)
	}
	state := &installState{
		SchemaVersion: 1,
		ManagedBy:     managedBy,
		Version:       version(),
		Release:       releaseRevision,
		Platform:      runtime.GOOS + "/" + runtime.GOARCH,
		Files: map[string]string{
			"tool/" + installedBinaryName(): sha256Hex(binary),
			"VERSION":                       sha256Hex([]byte(embeddedVersion)),
			"RELEASE":                       sha256Hex([]byte(releaseRevision)),
			"tool/" + keySourceFileName:     sha256Hex([]byte(keySourceDedicated)),
			"skill-template/" + skillName + "/SKILL.md":           sha256Hex([]byte(embeddedSkill)),
			"skill-template/" + skillName + "/agents/openai.yaml": sha256Hex([]byte(embeddedAgentConfig)),
		},
	}
	stateBytes, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("无法生成安装状态：%w", err)
	}
	stateBytes = append(stateBytes, '\n')
	desired := []desiredFile{
		{layout.binaryPath, binary, 0o777},
		{layout.versionPath, []byte(embeddedVersion), 0o666},
		{layout.releasePath, []byte(releaseRevision), 0o666},
		{layout.statePath, stateBytes, 0o666},
		{layout.skillPath, []byte(embeddedSkill), 0o666},
		{layout.agentPath, []byte(embeddedAgentConfig), 0o666},
		{filepath.Join(layout.toolDir, keySourceFileName), []byte(keySourceDedicated), 0o666},
	}
	return desired, state, nil
}

func readInstallState(layout installLayout) (*installState, error) {
	if err := ensureDirectoryTarget(layout.toolDir, "工具目录"); err != nil {
		return nil, err
	}
	if err := ensureDirectoryTarget(layout.skillDir, "Skill 目录"); err != nil {
		return nil, err
	}
	if regularFileExists(layout.statePath) {
		content, err := os.ReadFile(layout.statePath)
		if err != nil {
			return nil, fmt.Errorf("无法读取安装状态：%s", layout.statePath)
		}
		state := &installState{}
		if json.Unmarshal(content, state) != nil || state.SchemaVersion != 1 || state.ManagedBy != managedBy {
			return nil, fmt.Errorf("目标目录无法确认由本工具管理：%s", layout.toolDir)
		}
		if state.Version != version() && state.Version != "0.2.0" && state.Version != "0.3.0" && state.Version != "0.4.0" && state.Version != "0.4.1" && state.Version != "0.5.0" && state.Version != "0.6.0" && state.Version != "0.7.0" && state.Version != "0.7.1" && state.Version != "0.7.2" && state.Version != "0.7.3" && state.Version != "0.7.4" && state.Version != "0.7.5" && state.Version != "0.7.6" && state.Version != "0.8.0" {
			return nil, fmt.Errorf("已安装版本为 %s，当前安装包为 %s；不支持直接升级。", state.Version, version())
		}
		return state, nil
	}
	if pathExists(layout.statePath) {
		return nil, fmt.Errorf("安装状态文件不是普通文件：%s", layout.statePath)
	}
	entries, err := os.ReadDir(layout.toolDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("无法检查工具目录：%w", err)
	}
	for _, entry := range entries {
		if entry.Name() != legacyKeyFileName {
			return nil, fmt.Errorf("目标目录无法确认由本工具管理：%s", layout.toolDir)
		}
	}
	entries, err = os.ReadDir(layout.skillDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("无法检查 Skill 目录：%w", err)
	}
	if len(entries) != 0 {
		return nil, fmt.Errorf("目标目录无法确认由本工具管理：%s", layout.skillDir)
	}
	return nil, nil
}

func checkManagedPaths(layout installLayout) error {
	if info, err := os.Lstat(layout.codexHome); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("Codex Home 是符号链接，停止操作：%s", layout.codexHome)
	}
	for _, item := range []struct {
		path  string
		label string
	}{
		{layout.toolDir, "工具目录"},
		{layout.skillDir, "Skill 目录"},
		{filepath.Join(layout.skillDir, "agents"), "Skill 子目录"},
		{layout.agentsPath, "AGENTS.md"},
		{layout.override, "AGENTS.override.md"},
	} {
		if err := rejectSymlinkComponents(layout.codexHome, item.path, item.label); err != nil {
			return err
		}
	}
	return nil
}

func rejectSymlinkComponents(base, target, label string) error {
	relative, err := filepath.Rel(base, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s 不在 Codex Home 内：%s", label, target)
	}
	current := base
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		if part == "." || part == "" {
			continue
		}
		current = filepath.Join(current, part)
		if info, err := os.Lstat(current); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s 路径包含符号链接，停止操作：%s", label, current)
		}
	}
	return nil
}

func ensureDirectoryTarget(path, label string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("无法检查%s：%w", label, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("%s不是可管理的普通目录：%s", label, path)
	}
	return nil
}

func readAgents(layout installLayout, checkOverride bool) (string, error) {
	if checkOverride && pathExists(layout.override) {
		info, err := os.Lstat(layout.override)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return "", fmt.Errorf("AGENTS.override.md 不是普通文件：%s", layout.override)
		}
		if info.Size() > 0 {
			return "", fmt.Errorf("发现非空 AGENTS.override.md，会覆盖同级 AGENTS.md：%s", layout.override)
		}
	}
	if !pathExists(layout.agentsPath) {
		return "", nil
	}
	info, err := os.Lstat(layout.agentsPath)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("AGENTS.md 不是普通文件：%s", layout.agentsPath)
	}
	content, err := os.ReadFile(layout.agentsPath)
	if err != nil {
		return "", fmt.Errorf("无法读取 AGENTS.md：%s", layout.agentsPath)
	}
	if _, err := markerSpan(string(content)); err != nil {
		return "", err
	}
	return string(content), nil
}

type textSpan struct{ start, end int }

func markerSpan(content string) (*textSpan, error) {
	starts := strings.Count(content, ruleStart)
	ends := strings.Count(content, ruleEnd)
	if starts == 0 && ends == 0 {
		return nil, nil
	}
	if starts != 1 || ends != 1 {
		return nil, errors.New("AGENTS.md 中的中转站生图标记缺失、重复或不完整。")
	}
	start := strings.Index(content, ruleStart)
	endStart := strings.Index(content, ruleEnd)
	if endStart < start {
		return nil, errors.New("AGENTS.md 中的中转站生图标记顺序错误。")
	}
	return &textSpan{start, endStart + len(ruleEnd)}, nil
}

func installRule(content string) (string, error) {
	span, err := markerSpan(content)
	if err != nil {
		return "", err
	}
	rule := strings.TrimSpace(embeddedAgentsRule)
	if span != nil {
		return content[:span.start] + rule + content[span.end:], nil
	}
	if content == "" {
		return rule + "\n", nil
	}
	return strings.TrimRight(content, "\n") + "\n\n" + rule + "\n", nil
}

func removeRule(content string) (string, error) {
	span, err := markerSpan(content)
	if err != nil {
		return "", err
	}
	if span == nil {
		return "", errors.New("AGENTS.md 中缺少本工具规则块，停止卸载。")
	}
	return content[:span.start] + content[span.end:], nil
}

func assertKnownUninstallFiles(layout installLayout, state *installState) ([]string, error) {
	managed, expectedHashes, err := managedPathsFromState(layout, state)
	if err != nil {
		return nil, err
	}
	allowedTool := map[string]bool{legacyKeyFileName: true, stateFileName: true}
	allowedSkill := map[string]bool{}
	allowAgentsDirectory := false
	for logical, path := range managed {
		if strings.HasPrefix(logical, "skill-template/") {
			relative, _ := filepath.Rel(layout.skillDir, path)
			allowedSkill[relative] = true
			if strings.HasPrefix(relative, "agents"+string(filepath.Separator)) {
				allowAgentsDirectory = true
			}
		} else {
			allowedTool[filepath.Base(path)] = true
		}
	}
	entries, err := os.ReadDir(layout.toolDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, entry := range entries {
		info, err := os.Lstat(filepath.Join(layout.toolDir, entry.Name()))
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || !allowedTool[entry.Name()] {
			return nil, fmt.Errorf("工具目录包含未知文件，停止卸载：%s", filepath.Join(layout.toolDir, entry.Name()))
		}
	}
	err = filepath.WalkDir(layout.skillDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == layout.skillDir {
			return nil
		}
		relative, _ := filepath.Rel(layout.skillDir, path)
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("Skill 目录包含未知文件，停止卸载：%s", path)
		}
		if entry.IsDir() {
			if relative != "agents" || !allowAgentsDirectory {
				return fmt.Errorf("Skill 目录包含未知目录，停止卸载：%s", path)
			}
			return nil
		}
		if !allowedSkill[relative] {
			return fmt.Errorf("Skill 目录包含未知文件，停止卸载：%s", path)
		}
		return nil
	})
	if os.IsNotExist(err) {
		err = nil
	}
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(managed)+1)
	for logical, path := range managed {
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("托管文件缺失或无法读取，停止卸载：%s", path)
		}
		if sha256Hex(content) != expectedHashes[logical] {
			return nil, fmt.Errorf("托管文件内容已变化，停止卸载：%s", path)
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	paths = append(paths, layout.statePath)
	return paths, nil
}

func managedPathsFromState(layout installLayout, state *installState) (map[string]string, map[string]string, error) {
	known := map[string]string{
		"VERSION":                                             layout.versionPath,
		"RELEASE":                                             layout.releasePath,
		"tool/myautoapi-imagines-tool":                        filepath.Join(layout.toolDir, "myautoapi-imagines-tool"),
		"tool/myautoapi-imagines-tool.exe":                    filepath.Join(layout.toolDir, "myautoapi-imagines-tool.exe"),
		"tool/" + legacyPythonName:                            layout.toolDir + string(filepath.Separator) + legacyPythonName,
		"tool/" + keySourceFileName:                           filepath.Join(layout.toolDir, keySourceFileName),
		"skill-template/" + skillName + "/SKILL.md":           layout.skillPath,
		"skill-template/" + skillName + "/agents/openai.yaml": layout.agentPath,
	}
	managed := make(map[string]string, len(state.Files))
	hashes := make(map[string]string, len(state.Files))
	for logical, expectedHash := range state.Files {
		path, ok := known[logical]
		if !ok || len(expectedHash) != 64 {
			return nil, nil, fmt.Errorf("安装状态包含未知托管文件，停止卸载：%s", logical)
		}
		if _, err := hex.DecodeString(expectedHash); err != nil {
			return nil, nil, fmt.Errorf("安装状态中的文件哈希无效，停止卸载：%s", logical)
		}
		managed[logical] = path
		hashes[logical] = strings.ToLower(expectedHash)
	}
	if len(managed) == 0 {
		return nil, nil, errors.New("安装状态没有可确认的托管文件，停止卸载。")
	}
	return managed, hashes, nil
}

func sameFile(path string, expected []byte) (bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return false, fmt.Errorf("托管文件路径发生冲突：%s", path)
	}
	content, err := os.ReadFile(path)
	return string(content) == string(expected), err
}

func sameDesiredFile(file desiredFile) (bool, error) {
	same, err := sameFile(file.path, file.content)
	if err != nil || !same || runtime.GOOS == "windows" || file.mode&0o111 == 0 {
		return same, err
	}
	info, err := os.Stat(file.path)
	if err != nil {
		return false, err
	}
	return info.Mode().Perm()&0o111 != 0, nil
}

func (transaction *fileTransaction) write(path string, content []byte, mode os.FileMode) error {
	entry := backupEntry{path: path}
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("文件路径发生冲突：%s", path)
		}
		entry.existed = true
		entry.mode = info.Mode().Perm()
		entry.content, err = os.ReadFile(path)
		if err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	transaction.journal = append(transaction.journal, entry)
	return atomicWrite(path, content, mode)
}

func (transaction *fileTransaction) writePreservingMode(path string, content []byte, defaultMode os.FileMode) error {
	mode := defaultMode
	if regularFileExists(path) {
		currentMode, err := regularFileMode(path)
		if err != nil {
			return err
		}
		mode = currentMode
	}
	return transaction.write(path, content, mode)
}

func (transaction *fileTransaction) remove(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("文件路径发生冲突：%s", path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	transaction.journal = append(transaction.journal, backupEntry{path: path, existed: true, content: content, mode: info.Mode().Perm()})
	return os.Remove(path)
}

func (transaction *fileTransaction) rollback() error {
	var failures []string
	for index := len(transaction.journal) - 1; index >= 0; index-- {
		entry := transaction.journal[index]
		var err error
		if entry.existed {
			err = atomicWrite(entry.path, entry.content, entry.mode)
		} else if removeErr := os.Remove(entry.path); removeErr != nil && !os.IsNotExist(removeErr) {
			err = removeErr
		}
		if err != nil {
			failures = append(failures, entry.path)
		}
	}
	if len(failures) != 0 {
		return fmt.Errorf("无法恢复：%s", strings.Join(failures, "、"))
	}
	return nil
}

func atomicWrite(path string, content []byte, mode os.FileMode) error {
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o777); err != nil {
		return err
	}
	random := make([]byte, 12)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	temporary := filepath.Join(parent, ".myautoapi-tmp-"+hex.EncodeToString(random))
	file, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporary)
		}
	}()
	if _, err := file.Write(content); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		return err
	}
	removeTemporary = false
	return nil
}

func cleanupEmptyDirectories(layout installLayout) {
	for _, path := range []string{
		filepath.Join(layout.skillDir, "agents"),
		layout.skillDir,
		filepath.Join(layout.codexHome, "skills"),
		layout.toolDir,
		filepath.Join(layout.codexHome, "tools"),
	} {
		_ = os.Remove(path)
	}
}

func agentsBackupPath(path string) string {
	stamp := time.Now().Format("20060102-150405-000000000")
	return filepath.Join(filepath.Dir(path), "AGENTS.md.myautoapi-backup-"+stamp)
}

func sha256Hex(content []byte) string {
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:])
}

func regularFileExists(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}

func nonEmptyRegularFile(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 && info.Size() > 0
}

func regularFileMode(path string) (os.FileMode, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return 0, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return 0, fmt.Errorf("不是普通文件：%s", path)
	}
	return info.Mode().Perm(), nil
}

func pathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func sortedDesiredFiles(files []desiredFile) []desiredFile {
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	return files
}
